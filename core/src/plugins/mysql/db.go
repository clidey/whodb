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

package mysql

import (
	"net"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func (p *MySQLPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	mysqlConfig := mysqldriver.NewConfig()
	mysqlConfig.User = connectionInput.Username
	mysqlConfig.Passwd = connectionInput.Password
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = net.JoinHostPort(connectionInput.Hostname, strconv.Itoa(connectionInput.Port))
	mysqlConfig.DBName = connectionInput.Database
	mysqlConfig.AllowCleartextPasswords = connectionInput.AllowClearTextPasswords
	mysqlConfig.ParseTime = connectionInput.ParseTime
	mysqlConfig.Loc = connectionInput.Loc
	mysqlConfig.Params = connectionInput.ExtraOptions

	db, err := gorm.Open(mysql.Open(mysqlConfig.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(plugins.GetGormLogConfig())})
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"hostname": connectionInput.Hostname,
			"port":     connectionInput.Port,
			"database": connectionInput.Database,
			"username": connectionInput.Username,
		}).Error("Failed to connect to MySQL database")
		return nil, err
	}
	return db, nil
}
