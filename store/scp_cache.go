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

// SCPCache caches SCP instances from the database in memory to avoid slow calls on every request.
type SCPCache struct {
	// lowercase => do not expose/export these variables
	scpStore           *SCPStore
	scpMap             map[uint]*model.SCP // use getSCPMap() exclusively
	scpIDs             []uint              // holds the keys of the scpMap to simplify random lookups
	scpListRanked      []model.SCP
	lastUpdated        time.Time
	updateTTL          time.Duration // default 10 seconds
	rankingLastUpdated time.Time
	rankingTTL         time.Duration // default 5 seconds
	dirty              map[uint]bool // which SCPs need to be written back to the database
	lock               sync.Mutex
	updateLock         sync.Mutex
	rankingsLock       sync.Mutex
}

// NewSCPCache instantiates a new SCPCache with the default cache TTL durations.
func NewSCPCache(scpStore *SCPStore) *SCPCache {
	return NewSCPCacheWithDuration(scpStore, 10*time.Second, 5*time.Second)
}

// NewSCPCacheWithDuration instantiates a new SCPCache with the specified cache TTL durations.
func NewSCPCacheWithDuration(scpStore *SCPStore, updateTTL time.Duration, rankingTTL time.Duration) *SCPCache {
	return &SCPCache{
		scpStore:   scpStore,
		updateTTL:  updateTTL,
		rankingTTL: rankingTTL,
		dirty:      make(map[uint]bool),
	}
}

func (cache *SCPCache) getSCPMap() (*map[uint]*model.SCP, error) {
	if cache.scpMap == nil {
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
	}
	return &cache.scpMap, nil
}

// GetByID returns a reference to the SCP with the given ID if it exists, otherwise nil.
func (cache *SCPCache) GetByID(id uint) (*model.SCP, error) {
	scpMap, err := cache.getSCPMap()
	if err != nil {
		return nil, err
	}
	scpRef := (*scpMap)[id]
	return scpRef, nil
}

// Create adds the SCP reference to the database immediately, unless the database already contains the entry.
func (cache *SCPCache) Create(scp *model.SCP) error {
	// Creating a new SCP requires synchronising the map and database, since a new entry is added
	if err := cache.scpStore.Create(scp); err != nil {
		return err
	}
	cache.SynchroniseThenInvalidate()
	return nil
}

// SynchroniseThenInvalidate writes changes back to the database and invalidates the cache.
func (cache *SCPCache) SynchroniseThenInvalidate() (err error) {
	// Write changes back to the database then invalidate cached SCP collections,
	// causing us to re-fetch the database contents.
	cache.updateLock.Lock()
	defer cache.updateLock.Unlock()
	err = cache.synchroniseDatabase()
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.scpMap = nil
	cache.scpListRanked = nil
	cache.scpIDs = nil
	return err
}

// Update marks SCP references as changed.
// Changes are written to the database whenever this function is called at least updateTTL seconds apart.
func (cache *SCPCache) Update(scpRef ...*model.SCP) error {
	scpMap, err := cache.getSCPMap()
	if err != nil {
		return err
	}
	// The reference to the SCP object already has the changes, just mark it as dirty.
	for _, scp := range scpRef {
		// Ensure the scpRef is in the map, otherwise the changes are in an instance not managed by the cache!
		if mapSCP, ok := (*scpMap)[scp.ID]; !ok || mapSCP != scp {
			return errors.New("cannot update SCP instance not from the cache; use GetByID/GetRandomSCPs")
		}
		cache.dirty[scp.ID] = true
	}
	if cache.lastUpdated.IsZero() || time.Now().After(cache.lastUpdated.Add(cache.updateTTL)) {
		cache.updateLock.Lock()
		defer cache.updateLock.Unlock()
		// Double check in case another goroutine already updated and this one was waiting.
		if cache.lastUpdated.IsZero() || time.Now().After(cache.lastUpdated.Add(cache.updateTTL)) {
			return cache.synchroniseDatabase()
		}
	}
	return nil
}

func (cache *SCPCache) forceUpdate(scpRef *model.SCP) error {
	// Writes the object back to the database immediately.
	cache.dirty[scpRef.ID] = false
	return cache.scpStore.Update(scpRef)
}

func (cache *SCPCache) synchroniseDatabase() (err error) {
	// Check which SCP objects are invalid and write them back to the database.
	scpMap, err := cache.getSCPMap()
	if err != nil {
		return err
	}
	// Batch update (gorm doesn't seem to support real batch SQL updates...)
	for id, needsUpdate := range cache.dirty {
		if needsUpdate {
			if scp, ok := (*scpMap)[id]; ok {
				err = cache.forceUpdate(scp)
				if err != nil {
					break
				}
			} else {
				delete(cache.dirty, id)
			}
		}
	}
	cache.lastUpdated = time.Now()
	return err
}

// GetRandomSCPs returns a slice of n unique, pseudo-uniformly-randomly selected SCP instances.
func (cache *SCPCache) GetRandomSCPs(n int) ([]*model.SCP, error) {
	scpMap, err := cache.getSCPMap()
	if err != nil {
		return nil, err
	}
	numSCPs := len(*scpMap)
	if n < 1 || n > len(*scpMap) {
		return nil, fmt.Errorf("invalid length argument: %d. #SCPs = %d", n, numSCPs)
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
		return nil, errors.New("random number generation exceeded maximimum iterations")
	}
	return randomSCPs, nil
}

// GetRankedSCPs returns a slice containing all SCP instances in descending order of their rating.
func (cache *SCPCache) GetRankedSCPs() ([]model.SCP, error) {
	// Avoid too many expensive calls to get SCPs sorted by rating by caching the last calculated result for a period of time.
	// The returned SCP objects should be treated as read-only, as they are intentionally not in sync with the map.
	if cache.scpListRanked == nil || cache.rankingLastUpdated.IsZero() || time.Now().After(cache.rankingLastUpdated.Add(cache.rankingTTL)) {
		cache.rankingsLock.Lock()
		defer cache.rankingsLock.Unlock()
		// Double check in case another goroutine already updated and this one was waiting.
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
				if rankedSCPs[i].Rating == rankedSCPs[j].Rating {
					// break ties by ID
					return rankedSCPs[i].ID < rankedSCPs[j].ID
				}
				return rankedSCPs[i].Rating > rankedSCPs[j].Rating
			})
			cache.scpListRanked = rankedSCPs
			cache.rankingLastUpdated = time.Now()
		}
	}
	// Can re-use cached result otherwise.
	return cache.scpListRanked, nil
}
