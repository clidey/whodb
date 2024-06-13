package postgres

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	databaseName := "postgres"
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v", config.Credentials.Hostname, config.Credentials.Username, config.Credentials.Password, databaseName, config.Credentials.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
