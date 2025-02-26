// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package sqlite3

import (
	"errors"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
)

func getDefaultDirectory() string {
	directory := "/db/"
	if env.IsDevelopment {
		directory = "./tmp/"
	}
	return directory
}

var errDoesNotExist = errors.New("unauthorized or the database doesn't exist")

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	database := config.Credentials.Database
	fileNameDatabase := filepath.Join(getDefaultDirectory(), database)
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
