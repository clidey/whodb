/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sqlite3

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func getDefaultDirectory() string {
	directory := "/db/"
	if env.IsDevelopment {
		directory = "tmp/"
	}
	return directory
}

var errDoesNotExist = errors.New("unauthorized or the database doesn't exist")

func (p *Sqlite3Plugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}
	database := connectionInput.Database

	var fileNameDatabase string

	// Desktop mode: treat database field as full path
	if env.GetIsDesktopMode() {
		// In desktop mode, the database field contains the full path
		fileNameDatabase = database

		// Verify file exists
		if _, err := os.Stat(fileNameDatabase); errors.Is(err, os.ErrNotExist) {
			log.Logger.WithError(err).WithFields(map[string]any{
				"database": database,
				"path":     fileNameDatabase,
			}).Error("SQLite database file does not exist")
			return nil, errDoesNotExist
		}
	} else {
		// Server mode: use default directory restriction
		fileNameDatabase = filepath.Join(getDefaultDirectory(), database)
		fileNameDatabase, err = filepath.EvalSymlinks(fileNameDatabase)
		if err != nil {
			log.Logger.WithError(err).WithFields(map[string]any{
				"database": database,
				"path":     fileNameDatabase,
			}).Error("Failed to evaluate SQLite database symlinks")
			return nil, err
		}
		if !strings.HasPrefix(fileNameDatabase, getDefaultDirectory()) {
			log.Logger.WithFields(map[string]any{
				"database":         database,
				"path":             fileNameDatabase,
				"defaultDirectory": getDefaultDirectory(),
			}).Error("SQLite database path is outside allowed directory")
			return nil, errDoesNotExist
		}
		if _, err := os.Stat(fileNameDatabase); errors.Is(err, os.ErrNotExist) {
			log.Logger.WithError(err).WithFields(map[string]any{
				"database": database,
				"path":     fileNameDatabase,
			}).Error("SQLite database file does not exist")
			return nil, errDoesNotExist
		}
	}

	db, err := gorm.Open(sqlite.Open(fileNameDatabase), &gorm.Config{Logger: logger.Default.LogMode(plugins.GetGormLogConfig())})
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"database": database,
			"path":     fileNameDatabase,
		}).Error("Failed to connect to SQLite database")
		return nil, err
	}
	return db, nil
}
