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
	scpStore          *SCPStore
	scpMap            map[uint]*model.SCP // use getSCPMap() exclusively
	scpIDs            []uint              // holds the keys of the scpMap to simplify random lookups
	scpListRanked     []model.SCP
	lastSynchronised  time.Time
	synchronisePeriod time.Duration // default 10 seconds
	lock              sync.Mutex
}

func NewSCPCache(scpStore *SCPStore) *SCPCache {
	return NewSCPCacheWithDuration(scpStore, 10*time.Second)
}

func NewSCPCacheWithDuration(scpStore *SCPStore, synchronisePeriod time.Duration) *SCPCache {
	return &SCPCache{
		scpStore:          scpStore,
		synchronisePeriod: synchronisePeriod,
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
	// Invalidates cached SCP collections
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if cache.scpMap != nil {
		// TODO: batch updates / only update changes / better error handling
		for _, scpRef := range cache.scpMap {
			if err = cache.Update(scpRef); err != nil {
				break
			}
		}
	}
	cache.scpMap = nil
	cache.scpListRanked = nil
	cache.scpIDs = nil
	return err
}

func (cache *SCPCache) Update(scpRef *model.SCP) error {
	// Should be no need for locking here, bypasses the cache
	// TODO: only update changes if possible
	// TODO: store changes in the cache and only update once
	//       every 10 seconds or something...
	return cache.scpStore.Update(scpRef)
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
	if cache.scpListRanked == nil || cache.lastSynchronised.IsZero() || time.Now().After(cache.lastSynchronised.Add(cache.synchronisePeriod)) {
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
		cache.lastSynchronised = time.Now()
	}
	// Can re-use cached result otherwise.
	return cache.scpListRanked, nil
}
