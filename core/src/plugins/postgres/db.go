package postgres

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	if db != nil {
		return db, nil
	}
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=5432", config.Credentials.Hostname, config.Credentials.Username, config.Credentials.Password, config.Credentials.Database)
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
