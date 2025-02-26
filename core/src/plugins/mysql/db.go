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

package mysql

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	portKey                    = "Port"
	parseTimeKey               = "Parse Time"
	locKey                     = "Loc"
	allowClearTextPasswordsKey = "Allow clear text passwords"
)

// todo: https://github.com/go-playground/validator
// todo: convert below to their respective types before passing into the configuration. check if it can be done before coming here

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "3306"))
	if err != nil {
		return nil, err
	}
	parseTime := common.GetRecordValueOrDefault(config.Credentials.Advanced, parseTimeKey, "True")
	loc, err := time.LoadLocation(common.GetRecordValueOrDefault(config.Credentials.Advanced, locKey, "UTC"))
	if err != nil {
		return nil, err
	}
	allowClearTextPasswords := common.GetRecordValueOrDefault(config.Credentials.Advanced, allowClearTextPasswordsKey, "0")

	mysqlConfig := mysqldriver.NewConfig()
	mysqlConfig.User = config.Credentials.Username
	mysqlConfig.Passwd = config.Credentials.Password
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = net.JoinHostPort(config.Credentials.Hostname, strconv.Itoa(port))
	mysqlConfig.DBName = config.Credentials.Database
	mysqlConfig.AllowCleartextPasswords = allowClearTextPasswords == "1"
	mysqlConfig.ParseTime = strings.ToLower(parseTime) == "true"
	mysqlConfig.Loc = loc

	// if this config is a pre-configured profile, then allow reading of additional params
	if config.Credentials.IsProfile {
		params := make(map[string]string)
		for _, record := range config.Credentials.Advanced {
			switch record.Key {
			case portKey, parseTimeKey, locKey, allowClearTextPasswordsKey:
				continue
			default:
				params[record.Key] = url.QueryEscape(record.Value)
			}
		}
		mysqlConfig.Params = params
	}

	db, err := gorm.Open(mysql.Open(mysqlConfig.FormatDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
