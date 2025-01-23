package sqlite3

import (
	"errors"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"golang.org/x/text/unicode/norm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func getDefaultDirectory() string {
	directory := "/db"
	if env.IsDevelopment {
		directory = "./tmp"
	}
	return directory
}

// VARIANT A
func cleanDatabaseName(name string) string {
	// this won't handle symlinks. could be something for the future
	// attempts to normalize for the different unicode formats
	forms := []norm.Form{
		norm.NFC,
		norm.NFD,
		norm.NFKC,
		norm.NFKD,
	}
	cleanName := name
	for _, form := range forms {
		if !form.IsNormal([]byte(cleanName)) {
			cleanName = form.String(cleanName)
		}
	}
	// attempts to clean bad strings that can allow for path traversal
	cleanName = filepath.Clean(name)
	cleanName = strings.TrimPrefix(cleanName, "../")
	cleanName = strings.TrimPrefix(cleanName, "/")
	cleanName = strings.ReplaceAll(cleanName, "..", "")
	return cleanName
}

// VARIANT B
func doesDatabaseExist(root string, name string) bool {
	found := false
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if path == name {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return false
	}
	return found
}

var errDoesNotExist = errors.New("unauthorized or the database doesn't exist")

// VARIANT A
func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	database := cleanDatabaseName(config.Credentials.Database)
	fileNameDatabase := filepath.Join(getDefaultDirectory(), database)
	fileNameDatabase = filepath.Clean(fileNameDatabase)
	if !strings.HasPrefix(fileNameDatabase, getDefaultDirectory()) {
		return nil, errDoesNotExist
	}
	if _, err := os.Stat(fileNameDatabase); errors.Is(err, os.ErrNotExist) {
		return nil, errDoesNotExist
	}
	db, err := gorm.Open(sqlite.Open(fileNameDatabase), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// VARIANT B
func DB_B(config *engine.PluginConfig) (*gorm.DB, error) {
	database := config.Credentials.Database
	fileNameDatabase := filepath.Join(getDefaultDirectory(), database)
	dbExists := doesDatabaseExist(getDefaultDirectory(), fileNameDatabase)
	if !dbExists {
		return nil, errDoesNotExist
	}
	if _, err := os.Stat(fileNameDatabase); errors.Is(err, os.ErrNotExist) {
		return nil, errDoesNotExist
	}
	db, err := gorm.Open(sqlite.Open(fileNameDatabase), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
