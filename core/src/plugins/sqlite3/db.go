package sqlite3

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func getDefaultDirectory() string {
	directory := "/db"
	if env.IsDevelopment {
		directory = "./tmp"
	}
	return directory
}

var errDoesNotExist = errors.New("unauthorized or the database doesn't exist")

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	database := config.Credentials.Database
	if !isValidDatabaseFileName(database) {
		return nil, errDoesNotExist
	}
	fileNameDatabase := filepath.Join(getDefaultDirectory(), database)
	if _, err := os.Stat(fileNameDatabase); errors.Is(err, os.ErrNotExist) {
		return nil, errDoesNotExist
	}
	db, err := gorm.Open(sqlite.Open(fileNameDatabase), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
