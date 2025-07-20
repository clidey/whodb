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

package gorm_plugin

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

const (
	portKey                    = "Port"
	parseTimeKey               = "Parse Time"
	locKey                     = "Loc"
	allowClearTextPasswordsKey = "Allow clear text passwords"
	sslModeKey                 = "SSL Mode"
	httpProtocolKey            = "HTTP Protocol"
	readOnlyKey                = "Readonly"
	debugKey                   = "Debug"
	connectionTimeoutKey       = "Connection Timeout"
)


type ConnectionInput struct {
	//common
	Username string `validate:"required"`
	Password string `validate:"required"`
	Database string `validate:"required"`
	Hostname string `validate:"required"`
	Port     int    `validate:"required"`

	//mysql/mariadb
	ParseTime               bool           `validate:"boolean"`
	Loc                     *time.Location `validate:"required"`
	AllowClearTextPasswords bool           `validate:"boolean"`

	//clickhouse
	SSLMode      string
	HTTPProtocol string
	ReadOnly     string
	Debug        string

	ConnectionTimeout int

	ExtraOptions map[string]string `validate:"omitnil"`
}

func (p *GormPlugin) ParseConnectionConfig(config *engine.PluginConfig) (*ConnectionInput, error) {
	//common
	defaultPort, ok := plugins.GetDefaultPort(p.Type)
	if !ok {
		return nil, fmt.Errorf("unsupported database type: %v", p.Type)
	}
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, defaultPort))
	if err != nil {
		return nil, err
	}

	//mysql/mariadb specific
	parseTime, err := strconv.ParseBool(common.GetRecordValueOrDefault(config.Credentials.Advanced, parseTimeKey, "True"))
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(common.GetRecordValueOrDefault(config.Credentials.Advanced, locKey, "Local"))
	if err != nil {
		return nil, err
	}
	allowClearTextPasswords, err := strconv.ParseBool(common.GetRecordValueOrDefault(config.Credentials.Advanced, allowClearTextPasswordsKey, "0"))
	if err != nil {
		return nil, err
	}

	//clickhouse specific
	sslMode := common.GetRecordValueOrDefault(config.Credentials.Advanced, sslModeKey, "disable")
	httpProtocol := common.GetRecordValueOrDefault(config.Credentials.Advanced, httpProtocolKey, "disable")
	readOnly := common.GetRecordValueOrDefault(config.Credentials.Advanced, readOnlyKey, "disable")
	debug := common.GetRecordValueOrDefault(config.Credentials.Advanced, debugKey, "disable")

	connectionTimeout, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, connectionTimeoutKey, "90"))
	if err != nil {
		return nil, err
	}

	database := config.Credentials.Database
	username := config.Credentials.Username
	password := config.Credentials.Password
	hostname := config.Credentials.Hostname

	if p.Type != engine.DatabaseType_Sqlite3 && p.Type != engine.DatabaseType_Postgres {
		database = url.PathEscape(database)
		username = url.PathEscape(username)
		password = url.PathEscape(password)
		hostname = url.PathEscape(hostname)
	}

	input := &ConnectionInput{
		Username:                username,
		Password:                password,
		Database:                database,
		Hostname:                hostname,
		Port:                    port,
		ParseTime:               parseTime,
		Loc:                     loc,
		AllowClearTextPasswords: allowClearTextPasswords,
		SSLMode:                 sslMode,
		HTTPProtocol:            httpProtocol,
		ReadOnly:                readOnly,
		Debug:                   debug,
		ConnectionTimeout:       connectionTimeout,
	}

	// if this config is a pre-configured profile, then allow reading of additional params
	if config.Credentials.IsProfile {
		params := make(map[string]string)
		for _, record := range config.Credentials.Advanced {
			switch record.Key {
			case portKey, parseTimeKey, locKey, allowClearTextPasswordsKey, sslModeKey, httpProtocolKey, readOnlyKey, debugKey, connectionTimeoutKey:
				continue
			default:
				// PostgreSQL uses libpq connection string format, not URL query parameters
				if p.Type == engine.DatabaseType_Postgres {
					params[record.Key] = record.Value // Raw value, PostgreSQL plugin will handle escaping
				} else {
					params[record.Key] = url.QueryEscape(record.Value)
				}
			}
		}
		input.ExtraOptions = params
	}

	return input, nil
}

func (p *GormPlugin) IsAvailable(config *engine.PluginConfig) bool {
	available, err := plugins.WithConnection[bool](config, p.DB, func(db *gorm.DB) (bool, error) {
		sqlDb, err := db.DB()
		if err != nil {
			return false, err
		}
		if err = sqlDb.Ping(); err != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return false
	}

	return available
}
