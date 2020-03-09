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
	// TODO: cache the whole table in memory (it's tiny) / array
	// and select random rows from that?
	// This orderBy generates random values for each table row, extremely slow!
	var allSCPs []*model.SCP
	err := store.db.Find(&allSCPs).Error
	if err != nil {
		return nil, err
	}
	return allSCPs, nil
}

func (store *SCPStore) GetRandomSCPs(numRandom uint) ([]model.SCP, error) {
	// TODO: cache the whole table in memory (it's tiny) / array
	// and select random rows from that?
	// This orderBy generates random values for each table row, extremely slow!
	var randomSCPs []model.SCP
	err := store.db.Order(gorm.Expr("random()")).Limit(numRandom).Find(&randomSCPs).Error
	if err != nil {
		return nil, err
	}
	return randomSCPs, nil
}

func (store *SCPStore) GetRankedSCPs() ([]model.SCP, error) {
	// TODO: cache this result and only recalculate on a timeout?
	// (probably best to make a separate class for that)
	var rankedSCPs []model.SCP
	err := store.db.Order("rating desc").Find(&rankedSCPs).Error
	if err != nil {
		return nil, err
	}
	return rankedSCPs, nil
}
