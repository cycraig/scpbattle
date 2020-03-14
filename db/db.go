package db

import (
	"errors"

	"github.com/cycraig/scpbattle/model"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // postgres driver static import
	_ "github.com/jinzhu/gorm/dialects/sqlite"   // sqlite driver static import
)

// NewDB instantiates a new GORM database.
func NewDB(dbType string, dbURL string, doLog bool) *gorm.DB {
	if dbType != "sqlite3" && dbType != "postgres" {
		panic(errors.New("unkown/unsupported database type: " + dbType))
	} else if dbURL == "" {
		panic(errors.New("empty database connection string"))
	}
	db, err := gorm.Open(dbType, dbURL)
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&model.SCP{})
	db.DB().SetMaxIdleConns(3)
	db.LogMode(doLog)
	return db
}
