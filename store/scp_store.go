package store

import (
	"github.com/jinzhu/gorm"

	"github.com/cycraig/scpbattle/model"
)

type SCPStore struct {
	db *gorm.DB
}

func NewSCPStore(db *gorm.DB) *SCPStore {
	return &SCPStore{
		db: db,
	}
}

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

func (store *SCPStore) Create(scp *model.SCP) (err error) {
	return store.db.Create(scp).Error
}

func (store *SCPStore) Update(scp *model.SCP) error {
	return store.db.Model(scp).Update(scp).Error
}

func (store *SCPStore) GetAllSCPs() ([]*model.SCP, error) {
	var allSCPs []*model.SCP
	err := store.db.Find(&allSCPs).Error
	if err != nil {
		return nil, err
	}
	return allSCPs, nil
}
