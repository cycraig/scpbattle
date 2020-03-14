package store

import (
	"github.com/jinzhu/gorm"

	"github.com/cycraig/scpbattle/model"
)

// SCPStore is a simple wrapper for persisting SCP instances.
type SCPStore struct {
	db *gorm.DB
}

// NewSCPStore returns a new SCPStore backed by the given database instace.
func NewSCPStore(db *gorm.DB) *SCPStore {
	return &SCPStore{
		db: db,
	}
}

// GetByID returns the SCP instance with the given ID if it exists in the database, otherwise nil.
func (store *SCPStore) GetByID(id uint) (*model.SCP, error) {
	var m model.SCP
	if err := store.db.First(&m, id).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// Create persists the given SCP instance in the database.
func (store *SCPStore) Create(scp *model.SCP) error {
	return store.db.Create(scp).Error
}

// Update writes the entire SCP instance back to its corresponding database entry.
// This does not create an entry in the database.
func (store *SCPStore) Update(scp *model.SCP) error {
	return store.db.Model(scp).Update(scp).Error
}

// GetAllSCPs returns a slice containing all SCP instances from the database.
func (store *SCPStore) GetAllSCPs() ([]*model.SCP, error) {
	var allSCPs []*model.SCP
	err := store.db.Find(&allSCPs).Error
	if err != nil {
		return nil, err
	}
	return allSCPs, nil
}
