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
	_ "embed"
	"sync"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed whodb-sample.sql
var sampleSQL string

const SampleDatabaseName = "whodb-sample"

const sampleDatabaseURI = "file:" + SampleDatabaseName + "?mode=memory&cache=shared"

var (
	sampleDBOnce sync.Once
	sampleDBErr  error
)

func IsSampleDatabase(name string) bool {
	return name == SampleDatabaseName
}

func GetSampleProfile() types.DatabaseCredentials {
	return types.DatabaseCredentials{
		Alias:     "Sample SQLite Database",
		Database:  SampleDatabaseName,
		Type:      engine.DatabaseType_Sqlite3,
		IsProfile: true,
		Source:    "builtin",
	}
}

func GetSampleDatabase() (*gorm.DB, error) {
	sampleDBOnce.Do(func() {
		db, err := gorm.Open(sqlite.Open(sampleDatabaseURI), &gorm.Config{
			Logger: logger.Default.LogMode(plugins.GetGormLogConfig()),
		})
		if err != nil {
			log.Logger.WithError(err).Error("Failed to create sample in-memory database")
			sampleDBErr = err
			return
		}

		if err := db.Exec(sampleSQL).Error; err != nil {
			log.Logger.WithError(err).Error("Failed to initialize sample database schema")
			sampleDBErr = err
			return
		}
	})

	if sampleDBErr != nil {
		return nil, sampleDBErr
	}

	return gorm.Open(sqlite.Open(sampleDatabaseURI), &gorm.Config{
		Logger: logger.Default.LogMode(plugins.GetGormLogConfig()),
	})
}
