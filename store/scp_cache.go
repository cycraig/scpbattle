package store

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/cycraig/scpbattle/model"
)

type SCPCache struct {
	// Caches the contents of the database in memory to avoid slow calls on every request
	// lowercase => do not expose/export these variables
	scpStore           *SCPStore
	scpMap             map[uint]*model.SCP // use getSCPMap() exclusively
	scpIDs             []uint              // holds the keys of the scpMap to simplify random lookups
	scpListRanked      []model.SCP
	lastUpdated        time.Time
	updateTTL          time.Duration // default 5 seconds
	rankingLastUpdated time.Time
	rankingTTL         time.Duration // default 10 seconds
	invalidated        map[uint]bool // which SCPs need to be written back to the database
	lock               sync.Mutex
}

func NewSCPCache(scpStore *SCPStore) *SCPCache {
	return NewSCPCacheWithDuration(scpStore, 5*time.Second, 10*time.Second)
}

func NewSCPCacheWithDuration(scpStore *SCPStore, updateTTL time.Duration, rankingTTL time.Duration) *SCPCache {
	return &SCPCache{
		scpStore:    scpStore,
		updateTTL:   updateTTL,
		rankingTTL:  rankingTTL,
		invalidated: make(map[uint]bool),
	}
}

func (cache *SCPCache) getSCPMap() (*map[uint]*model.SCP, error) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if cache.scpMap == nil {
		allSCPs, err := cache.scpStore.GetAllSCPs()
		if err != nil {
			return nil, err
		}
		scpMap := make(map[uint]*model.SCP)
		scpIDs := make([]uint, len(allSCPs))
		for i, scp := range allSCPs {
			scpMap[scp.ID] = scp
			scpIDs[i] = scp.ID
		}
		cache.scpMap = scpMap
		cache.scpIDs = scpIDs
	}
	return &cache.scpMap, nil
}

func (cache *SCPCache) GetByID(id uint) (*model.SCP, error) {
	scpMap, err := cache.getSCPMap()
	if err != nil {
		return nil, err
	}
	scpRef := (*scpMap)[id]
	return scpRef, nil
}

func (cache *SCPCache) Create(scp *model.SCP) error {
	// Creating a new SCP requires synchronising the map and database, since a new entry is added
	if err := cache.scpStore.Create(scp); err != nil {
		return err
	}
	cache.synchroniseThenInvalidate()
	return nil
}

func (cache *SCPCache) synchroniseThenInvalidate() (err error) {
	// Writes changes back to the database then invalidates cached SCP collections,
	// causing us to re-fetch the database contents.
	cache.lock.Lock()
	defer cache.lock.Unlock()
	err = cache.synchroniseDatabase()
	cache.scpMap = nil
	cache.scpListRanked = nil
	cache.scpIDs = nil
	return err
}

func (cache *SCPCache) Update(scpRef ...*model.SCP) error {
	// The reference to the SCP already has the changes, just mark it as needing updating.
	for _, scp := range scpRef {
		cache.invalidated[scp.ID] = true
	}
	if cache.lastUpdated.IsZero() || time.Now().After(cache.lastUpdated.Add(cache.updateTTL)) {
		return cache.synchroniseDatabase()
	}
	return nil
}

func (cache *SCPCache) forceUpdate(scpRef *model.SCP) error {
	// Writes the object back to the database immediately.
	cache.invalidated[scpRef.ID] = false
	return cache.scpStore.Update(scpRef)
}

func (cache *SCPCache) synchroniseDatabase() (err error) {
	// Check which SCP objects are invalid and write them back to the database.
	scpMap, err := cache.getSCPMap()
	if err != nil {
		return err
	}
	for id, needsUpdate := range cache.invalidated {
		if needsUpdate {
			if scp, ok := (*scpMap)[id]; ok {
				// Using a goroutine to avoid blocking, but hides errors...
				err = cache.forceUpdate(scp)
			} else {
				delete(cache.invalidated, id)
			}
		}
	}
	cache.lastUpdated = time.Now()
	return err
}

func (cache *SCPCache) GetRandomSCPs(n int) ([]*model.SCP, error) {
	scpMap, err := cache.getSCPMap()
	if err != nil {
		return nil, err
	}
	numSCPs := len(*scpMap)
	if n < 1 || n > len(*scpMap) {
		return nil, errors.New(fmt.Sprintf("Invalid length argument: %d. #SCPs = %d", n, numSCPs))
	}
	randomSCPs := make([]*model.SCP, n)
	scpIDs := cache.scpIDs
	set := make(map[int]struct{}) // set structure
	maxIterations := 2*n + 50     // prevent infinite loops
	totalIterations := 0
	// Generate n unique random integers by resampling duplicates, should be faster
	// and use less memory than shuffling a list of integers in [0,#SCPs) and taking
	// the first n entries.
	i := 0
	for i = 0; i < n && totalIterations < maxIterations; i++ {
		r := rand.Intn(numSCPs)
		if _, exists := set[r]; exists == true {
			i--
		} else {
			set[r] = struct{}{} // use 0-byte structs instead of bools to save memory
			randomSCPs[i] = (*scpMap)[scpIDs[r]]
		}
		totalIterations++
	}
	if totalIterations >= maxIterations && i < n {
		return nil, errors.New("Random number generation exceeded maximimum iterations.")
	}
	return randomSCPs, nil
}

func (cache *SCPCache) GetRankedSCPs() ([]model.SCP, error) {
	// Avoid too many expensive calls to get SCPs sorted by rating by caching the last calculated result for a period of time.
	// The returned SCP objects should be treated as read-only, as they are intentionally not in sync with the map.
	if cache.scpListRanked == nil || cache.rankingLastUpdated.IsZero() || time.Now().After(cache.rankingLastUpdated.Add(cache.rankingTTL)) {
		scpMap, err := cache.getSCPMap()
		if err != nil {
			return nil, err
		}
		rankedSCPs := make([]model.SCP, len(*scpMap))
		i := 0
		for _, scpRef := range *scpMap {
			rankedSCPs[i] = *scpRef
			i++
		}
		// Sort SCPs by ELO rating in descending order.
		sort.Slice(rankedSCPs, func(i, j int) bool {
			return rankedSCPs[i].Rating > rankedSCPs[j].Rating
		})
		cache.scpListRanked = rankedSCPs
		cache.rankingLastUpdated = time.Now()
	}
	// Can re-use cached result otherwise.
	return cache.scpListRanked, nil
}
