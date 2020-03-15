package store_test

import (
	"os"
	"testing"
	"time"

	"github.com/cycraig/scpbattle/db"
	"github.com/cycraig/scpbattle/model"
	"github.com/cycraig/scpbattle/store"
)

func TestSCPCache(t *testing.T) {

	// Initialise database.
	fdb := "TestSCPCache.db"
	os.Remove(fdb)
	d := db.NewDB("sqlite3", fdb, false)
	scpStore := store.NewSCPStore(d)
	// do not allow the cache to invalidate itself automatically during the test.
	scpCache := store.NewSCPCacheWithDuration(scpStore, 100000*time.Second, 100000*time.Second)
	defer func() {
		if err := d.Close(); err != nil {
			t.Log(err)
		}
		if err := os.Remove(fdb); err != nil {
			t.Log(err)
		}
	}()

	// Initial synchronise to prevent any automatic writes to the database during the test.
	AssertNoError(t, scpCache.SynchroniseThenInvalidate())

	// Populate example data.
	s1 := model.NewSCP("SCP-049", "The Plague Doctor", "scp_049.jpg", "http://www.scp-wiki.net/scp-049")
	s2 := model.NewSCP("SCP-096", "The Shy Guy", "scp_096.jpg", "http://www.scp-wiki.net/scp-096")
	s3 := model.NewSCP("SCP-106", "The Old Man", "scp_106.jpg", "http://www.scp-wiki.net/scp-106")
	s4 := model.NewSCP("SCP-173", "The Sculpture", "scp_173.jpg", "http://www.scp-wiki.net/scp-173")
	s5 := model.NewSCP("SCP-682", "The Hard-To-Destroy Reptile", "scp_682.jpg", "http://www.scp-wiki.net/scp-682")
	s6 := model.NewSCP("SCP-939", "With Many Voices", "scp_939.jpg", "http://www.scp-wiki.net/scp-939")
	s7 := model.NewSCP("●●|●●●●●|●●|●", "", "scp_2521.jpg", "http://www.scp-wiki.net/scp-2521")

	// Store in database from cache (this will populate default fields, e.g. ID, in s1...s7 as well).
	AssertNoError(t, scpCache.Create(s1))
	AssertNoError(t, scpCache.Create(s2))
	AssertNoError(t, scpCache.Create(s3))
	AssertNoError(t, scpCache.Create(s4))
	AssertNoError(t, scpCache.Create(s5))
	AssertNoError(t, scpCache.Create(s6))
	AssertNoError(t, scpCache.Create(s7))

	// Ensure data is in the database.
	var allSCPs []*model.SCP
	AssertNoError(t, d.Order("ID asc").Find(&allSCPs).Error)
	AssertSCPEqual(t, s1, allSCPs[0])
	AssertSCPEqual(t, s2, allSCPs[1])
	AssertSCPEqual(t, s3, allSCPs[2])
	AssertSCPEqual(t, s4, allSCPs[3])
	AssertSCPEqual(t, s5, allSCPs[4])
	AssertSCPEqual(t, s6, allSCPs[5])
	AssertSCPEqual(t, s7, allSCPs[6])

	// Ensure data is in the store.
	storeSCPs, err := scpStore.GetAllSCPs()
	AssertNoError(t, err)
	AssertSCPEqual(t, s1, storeSCPs[0])
	AssertSCPEqual(t, s2, storeSCPs[1])
	AssertSCPEqual(t, s3, storeSCPs[2])
	AssertSCPEqual(t, s4, storeSCPs[3])
	AssertSCPEqual(t, s5, storeSCPs[4])
	AssertSCPEqual(t, s6, storeSCPs[5])
	AssertSCPEqual(t, s7, storeSCPs[6])

	// Test retrieving SCPs by ID from the cache.
	cacheSCPs := make([]*model.SCP, len(allSCPs))
	for i, scp := range allSCPs {
		cacheSCPs[i], err = scpCache.GetByID(scp.ID)
		AssertNoError(t, err)
		AssertSCPEqual(t, scp, cacheSCPs[i])
	}
	// Non-existent IDs should return nil, no error
	ret, err := scpCache.GetByID(123456789)
	AssertNoError(t, err)
	if ret != nil {
		t.Errorf("Expected nil, got %v", ret)
	}

	// Test retrieving random SCPs.
	for i := 0; i < 10; i++ {
		randomSCPs, err := scpCache.GetRandomSCPs(2)
		AssertNoError(t, err)
		AssertNotEqual(t, randomSCPs[0], randomSCPs[1])
	}

	// Test retrieving an invalid number of random SCPs throws an error.
	_, err = scpCache.GetRandomSCPs(1000000)
	AssertError(t, err)
	_, err = scpCache.GetRandomSCPs(len(allSCPs) + 1)
	AssertError(t, err)
	_, err = scpCache.GetRandomSCPs(0)
	AssertError(t, err)
	_, err = scpCache.GetRandomSCPs(-1)
	AssertError(t, err)
	_, err = scpCache.GetRandomSCPs(-100)
	AssertError(t, err)

	// Test retrieving SCPs ordered by rating matches the database.
	rankedSCPs, err := scpCache.GetRankedSCPs()
	AssertNoError(t, err)
	for i := 0; i < len(rankedSCPs)-1; i++ {
		AssertTrue(t, rankedSCPs[i].Rating >= rankedSCPs[i+1].Rating, "Ranked SCPs not in descending order!")
	}

	// Test trying to update non-cache managed SCPs throws errors.
	AssertError(t, scpCache.Update(model.NewSCP("blahblah_name", "Random description", "scp_2521.jpg", "http://www.scp-wiki.net/scp-2521")))
	AssertError(t, scpCache.Update(s1))
	AssertError(t, scpCache.Update(s2))
	AssertError(t, scpCache.Update(s3))
	AssertError(t, scpCache.Update(s4))
	AssertError(t, scpCache.Update(s5))
	AssertError(t, scpCache.Update(s6))
	AssertError(t, scpCache.Update(s7))
	for _, storeSCP := range storeSCPs {
		AssertError(t, scpCache.Update(storeSCP))
	}
	for _, dbSCP := range allSCPs {
		AssertError(t, scpCache.Update(dbSCP))
	}

	// Test updates are cached and do not reflect in the database until synchronised.
	cacheSCPs[0].Rating = 1.0
	cacheSCPs[1].Rating = 2.0
	cacheSCPs[2].Rating = 3.0
	cacheSCPs[3].Rating = 4.0
	cacheSCPs[4].Rating = 5.0
	cacheSCPs[5].Rating = 6.0
	cacheSCPs[6].Rating = 7.0
	for _, scp := range cacheSCPs {
		AssertNoError(t, scpCache.Update(scp))
	}

	// Test retrieving ranked SCPs after cached updates uses the cached result.
	rankedSCPsAfterCacheUpdate, err := scpCache.GetRankedSCPs()
	AssertNoError(t, err)
	for i, rankedSCPAfterUpdate := range rankedSCPsAfterCacheUpdate {
		AssertSCPEqual(t, &rankedSCPAfterUpdate, &rankedSCPs[i])
	}

	// Ensure the values were NOT updated in the database.
	var nonUpdatedSCPs []*model.SCP
	AssertNoError(t, d.Order("ID asc").Find(&nonUpdatedSCPs).Error)
	AssertEqual(t, nonUpdatedSCPs[0].Rating, 1000.0)
	AssertEqual(t, nonUpdatedSCPs[1].Rating, 1000.0)
	AssertEqual(t, nonUpdatedSCPs[2].Rating, 1000.0)
	AssertEqual(t, nonUpdatedSCPs[3].Rating, 1000.0)
	AssertEqual(t, nonUpdatedSCPs[4].Rating, 1000.0)
	AssertEqual(t, nonUpdatedSCPs[5].Rating, 1000.0)
	AssertEqual(t, nonUpdatedSCPs[6].Rating, 1000.0)

	// Synchronise should write changes back to the database.
	AssertNoError(t, scpCache.SynchroniseThenInvalidate())
	var updatedSCPs []*model.SCP
	AssertNoError(t, d.Order("ID asc").Find(&updatedSCPs).Error)
	AssertEqual(t, updatedSCPs[0].Rating, 1.0)
	AssertEqual(t, updatedSCPs[1].Rating, 2.0)
	AssertEqual(t, updatedSCPs[2].Rating, 3.0)
	AssertEqual(t, updatedSCPs[3].Rating, 4.0)
	AssertEqual(t, updatedSCPs[4].Rating, 5.0)
	AssertEqual(t, updatedSCPs[5].Rating, 6.0)
	AssertEqual(t, updatedSCPs[6].Rating, 7.0)

	// Test retrieving ranked SCPs after synchronising with the database using the new values.
	rankedSCPsAfterRealUpdate, err := scpCache.GetRankedSCPs()
	AssertNoError(t, err)
	for i := 0; i < len(rankedSCPsAfterRealUpdate)-1; i++ {
		AssertSCPEqual(t, &rankedSCPsAfterRealUpdate[i], updatedSCPs[len(updatedSCPs)-i-1])
		AssertTrue(t, rankedSCPsAfterRealUpdate[i].Rating >= rankedSCPsAfterRealUpdate[i+1].Rating, "Ranked SCPs not in descending order after update!")
	}
}
