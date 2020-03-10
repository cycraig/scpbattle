package db

import (
	"github.com/cycraig/scpbattle/model"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // import the sqlite driver statically
)

func NewDB() *gorm.DB {
	db, err := gorm.Open("sqlite3", "./data.db")
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&model.SCP{})
	db.DB().SetMaxIdleConns(3)
	db.LogMode(true)
	return db
}
