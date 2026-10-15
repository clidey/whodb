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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
)

func getDefaultDirectory() string {
	directory := "/db/"
	if env.IsDevelopment {
		directory = "tmp/"
	}
	return directory
}

var errDoesNotExist = errors.New("unauthorized or the database doesn't exist")

const restrictedDSNOptions = "?enable_external_access=false&autoinstall_known_extensions=false&autoload_known_extensions=false&allow_community_extensions=false&lock_configuration=true"

func (p *DuckDBPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}
	database := connectionInput.Database
	localMode := env.GetIsLocalMode()

	var fileNameDatabase string

	if localMode {
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
	// The driver parses '?' as DSN options and '#' as a URL fragment. Either
	// character in a path can discard the security options appended below.
	if strings.ContainsAny(fileNameDatabase, "?#") {
		return nil, errors.New("DuckDB database path contains unsupported character '?' or '#'")
	}

	l := log.WithFields(map[string]any{
		"database": database,
		"path":     fileNameDatabase,
	})

	dsn := fileNameDatabase
	if !localMode {
		dsn += restrictedDSNOptions
	}
	db, err := gorm.Open(Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(plugins.GetGormLogConfig())})
	if err != nil {
		l.WithError(err).Error("Failed to connect to DuckDB database")
		return nil, err
	}
	if !localMode {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		var externalAccess, autoInstall, autoLoad, communityExtensions, locked bool
		err = sqlDB.QueryRowContext(config.OperationContext(), `SELECT
		current_setting('enable_external_access'),
		current_setting('autoinstall_known_extensions'),
		current_setting('autoload_known_extensions'),
		current_setting('allow_community_extensions'),
		current_setting('lock_configuration')`).Scan(&externalAccess, &autoInstall, &autoLoad, &communityExtensions, &locked)
		if err != nil || externalAccess || autoInstall || autoLoad || communityExtensions || !locked {
			_ = sqlDB.Close()
			if err != nil {
				return nil, fmt.Errorf("verify DuckDB security settings: %w", err)
			}
			return nil, errors.New("DuckDB security settings were not applied")
		}
	}

	if err := plugins.ConfigureConnectionPool(db); err != nil {
		l.WithError(err).Warn("Failed to configure connection pool")
	}

	return db, nil
}
