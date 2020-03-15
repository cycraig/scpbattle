package store_test

import (
	"os"
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/cycraig/scpbattle/db"
	"github.com/cycraig/scpbattle/model"
	"github.com/cycraig/scpbattle/store"
)

func TestSCPStore(t *testing.T) {

	// Initialise database.
	fdb := "TestSCPStore.db"
	os.Remove(fdb)
	d := db.NewDB("sqlite3", fdb, false)
	scpStore := store.NewSCPStore(d)
	defer func() {
		if err := d.Close(); err != nil {
			t.Log(err)
		}
		if err := os.Remove(fdb); err != nil {
			t.Log(err)
		}
	}()

	// Populate example data.
	s1 := model.NewSCP("SCP-049", "The Plague Doctor", "scp_049.jpg", "http://www.scp-wiki.net/scp-049")
	s2 := model.NewSCP("SCP-096", "The Shy Guy", "scp_096.jpg", "http://www.scp-wiki.net/scp-096")
	s3 := model.NewSCP("SCP-106", "The Old Man", "scp_106.jpg", "http://www.scp-wiki.net/scp-106")
	s4 := model.NewSCP("SCP-173", "The Sculpture", "scp_173.jpg", "http://www.scp-wiki.net/scp-173")
	s5 := model.NewSCP("SCP-682", "The Hard-To-Destroy Reptile", "scp_682.jpg", "http://www.scp-wiki.net/scp-682")
	s6 := model.NewSCP("SCP-939", "With Many Voices", "scp_939.jpg", "http://www.scp-wiki.net/scp-939")
	s7 := model.NewSCP("●●|●●●●●|●●|●", "", "scp_2521.jpg", "http://www.scp-wiki.net/scp-2521")

	// Store in database (this will populate default fields, e.g. ID, in s1...s7 as well).
	AssertNoError(t, scpStore.Create(s1))
	AssertNoError(t, scpStore.Create(s2))
	AssertNoError(t, scpStore.Create(s3))
	AssertNoError(t, scpStore.Create(s4))
	AssertNoError(t, scpStore.Create(s5))
	AssertNoError(t, scpStore.Create(s6))
	AssertNoError(t, scpStore.Create(s7))

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

	// Test retrieving SCPs from the store matches the database.
	storeSCPs, err := scpStore.GetAllSCPs()
	AssertNoError(t, err)
	AssertSCPEqual(t, s1, storeSCPs[0])
	AssertSCPEqual(t, s2, storeSCPs[1])
	AssertSCPEqual(t, s3, storeSCPs[2])
	AssertSCPEqual(t, s4, storeSCPs[3])
	AssertSCPEqual(t, s5, storeSCPs[4])
	AssertSCPEqual(t, s6, storeSCPs[5])
	AssertSCPEqual(t, s7, storeSCPs[6])

	// Test retrieving SCPs by ID from the store.
	for _, scp := range allSCPs {
		retSCP, err := scpStore.GetByID(scp.ID)
		AssertNoError(t, err)
		AssertSCPEqual(t, scp, retSCP)
	}
	// Non-existent IDs should return nil, no error
	ret, err := scpStore.GetByID(123456789)
	AssertNoError(t, err)
	if ret != nil {
		t.Errorf("Expected nil, got %v", ret)
	}

	// Test updates.
	allSCPs[0].Rating = 1.0
	allSCPs[1].Rating = 2.0
	allSCPs[2].Rating = 3.0
	allSCPs[3].Rating = 4.0
	allSCPs[4].Rating = 5.0
	allSCPs[5].Rating = 6.0
	allSCPs[6].Rating = 7.0
	for _, scp := range allSCPs {
		AssertNoError(t, scpStore.Update(scp))
	}

	// Ensure the values were updated in the database.
	var updatedSCPs []*model.SCP
	AssertNoError(t, d.Order("ID asc").Find(&updatedSCPs).Error)
	AssertEqual(t, updatedSCPs[0].Rating, 1.0)
	AssertEqual(t, updatedSCPs[1].Rating, 2.0)
	AssertEqual(t, updatedSCPs[2].Rating, 3.0)
	AssertEqual(t, updatedSCPs[3].Rating, 4.0)
	AssertEqual(t, updatedSCPs[4].Rating, 5.0)
	AssertEqual(t, updatedSCPs[5].Rating, 6.0)
	AssertEqual(t, updatedSCPs[6].Rating, 7.0)
}

func AssertEqual(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		return
	}
	t.Errorf("Received %v (type %v), expected %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

func AssertNotEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		return
	}
	t.Errorf("Expected %v (type %v) to be different from %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

func AssertSCPEqual(t *testing.T, a *model.SCP, b *model.SCP) {
	AssertEqual(t, a.ID, b.ID)
	AssertEqual(t, a.Name, b.Name)
	AssertEqual(t, a.Description, b.Description)
	AssertEqual(t, a.Image, b.Image)
	AssertEqual(t, a.Link, b.Link)
	AssertEqual(t, a.Rating, b.Rating)
	AssertEqual(t, a.Wins, b.Wins)
	AssertEqual(t, a.Losses, b.Losses)
}

func AssertTrue(t *testing.T, cond bool, msg string) {
	if cond != true {
		t.Errorf(msg)
	}
}

func AssertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(string(debug.Stack()), err)
	}
}

func AssertError(t *testing.T, err error) {
	if err == nil {
		t.Fatal(string(debug.Stack()), err)
	}
}
