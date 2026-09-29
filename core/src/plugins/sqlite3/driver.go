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

package sqlite3

import (
	"database/sql"
	"strings"

	sqlitedriver "github.com/mattn/go-sqlite3"
	sqlitegorm "gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/env"
)

const (
	confinedSQLiteDriverName         = "whodb_sqlite3_confined"
	confinedReadOnlySQLiteDriverName = "whodb_sqlite3_confined_read_only"
)

func init() {
	sql.Register(confinedSQLiteDriverName, confinedSQLiteDriver(false))
	sql.Register(confinedReadOnlySQLiteDriverName, confinedSQLiteDriver(true))
}

func confinedSQLiteDriver(readOnly bool) *sqlitedriver.SQLiteDriver {
	return &sqlitedriver.SQLiteDriver{
		ConnectHook: func(conn *sqlitedriver.SQLiteConn) error {
			conn.RegisterAuthorizer(func(operation int, argument1, _ string, _ string) int {
				if operation == sqlitedriver.SQLITE_ATTACH || operation == sqlitedriver.SQLITE_DETACH {
					return sqlitedriver.SQLITE_DENY
				}
				if readOnly && operation == sqlitedriver.SQLITE_PRAGMA && strings.EqualFold(argument1, "query_only") {
					return sqlitedriver.SQLITE_DENY
				}
				return sqlitedriver.SQLITE_OK
			})
			return nil
		},
	}
}

func sourceSQLiteDialector(dsn string, readOnly bool) gorm.Dialector {
	if env.GetIsLocalMode() {
		return sqlitegorm.Open(dsn)
	}
	driverName := confinedSQLiteDriverName
	if readOnly {
		driverName = confinedReadOnlySQLiteDriverName
	}
	return sqlitegorm.New(sqlitegorm.Config{DriverName: driverName, DSN: dsn})
}
