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

package postgres

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	portKey = "Port"
)

func escape(x string) string {
	return strings.ReplaceAll(x, "'", "\\'")
}

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "5432"))
	if err != nil {
		return nil, err
	}
	host := escape(config.Credentials.Hostname)
	username := escape(config.Credentials.Username)
	password := escape(config.Credentials.Password)
	database := escape(config.Credentials.Database)

	params := strings.Builder{}
	if config.Credentials.IsProfile {
		for _, record := range config.Credentials.Advanced {
			switch record.Key {
			case portKey:
				continue
			default:
				params.WriteString(fmt.Sprintf("%v='%v' ", record.Key, escape(record.Value)))
			}
		}
	}

	dsn := fmt.Sprintf("host='%v' user='%v' password='%v' dbname='%v' port='%v'",
		host, username, password, database, port)

	if params.Len() > 0 {
		dsn = fmt.Sprintf("%v %v", dsn, params.String())
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
