/*
 * Copyright 2026 Clidey, Inc.
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

package duckdb

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
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

func (p *DuckDBPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}
	database := connectionInput.Database

	var fileNameDatabase string

	if env.GetIsLocalMode() {
		fileNameDatabase = database

		if _, err := os.Stat(fileNameDatabase); errors.Is(err, os.ErrNotExist) {
			log.WithError(err).WithFields(map[string]any{
				"database": database,
				"path":     fileNameDatabase,
			}).Error("DuckDB database file does not exist")
			return nil, errDoesNotExist
		}
	} else {
		fileNameDatabase = filepath.Join(getDefaultDirectory(), database)
		fileNameDatabase, err = filepath.EvalSymlinks(fileNameDatabase)
		if err != nil {
			log.WithError(err).WithFields(map[string]any{
				"database": database,
				"path":     fileNameDatabase,
			}).Error("Failed to evaluate DuckDB database symlinks")
			return nil, err
		}
		if !strings.HasPrefix(fileNameDatabase, getDefaultDirectory()) {
			log.WithFields(map[string]any{
				"database":         database,
				"path":             fileNameDatabase,
				"defaultDirectory": getDefaultDirectory(),
			}).Error("DuckDB database path is outside allowed directory")
			return nil, errDoesNotExist
		}
		if _, err := os.Stat(fileNameDatabase); errors.Is(err, os.ErrNotExist) {
			log.WithError(err).WithFields(map[string]any{
				"database": database,
				"path":     fileNameDatabase,
			}).Error("DuckDB database file does not exist")
			return nil, errDoesNotExist
		}
	}

	l := log.WithFields(map[string]any{
		"database": database,
		"path":     fileNameDatabase,
	})

	db, err := gorm.Open(Open(fileNameDatabase), &gorm.Config{Logger: logger.Default.LogMode(plugins.GetGormLogConfig())})
	if err != nil {
		l.WithError(err).Error("Failed to connect to DuckDB database")
		return nil, err
	}

	if err := plugins.ConfigureConnectionPool(db); err != nil {
		l.WithError(err).Warn("Failed to configure connection pool")
	}

	return db, nil
}
