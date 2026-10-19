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

package mysql

import (
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestMySQLFamilyRejectsUnrestrictedLocalInfileInServerMode(t *testing.T) {
	t.Setenv("WHODB_DESKTOP", "false")
	t.Setenv("WHODB_CLI", "false")

	families := []struct {
		name   string
		dbType engine.DatabaseType
		plugin *MySQLPlugin
	}{
		{"MySQL", engine.DatabaseType_MySQL, NewMySQLPlugin().PluginFunctions.(*MySQLPlugin)},
		{"MariaDB", engine.DatabaseType_MariaDB, NewMyMariaDBPlugin().PluginFunctions.(*MySQLPlugin)},
		{"TiDB", engine.DatabaseType_TiDB, NewTiDBPlugin().PluginFunctions.(*MySQLPlugin)},
	}
	options := []struct {
		name string
		key  string
	}{
		{"literal", "allowAllFiles"},
		{"DSN delimiter", "ignored&allowAllFiles"},
	}

	for _, family := range families {
		for _, option := range options {
			t.Run(family.name+"/"+option.name, func(t *testing.T) {
				config := engine.NewPluginConfig(&engine.Credentials{
					Type:      string(family.dbType),
					Hostname:  "127.0.0.1",
					Username:  "user",
					Database:  "test_db",
					IsProfile: true,
					Advanced: []engine.Record{
						{Key: "Port", Value: "1"},
						{Key: option.key, Value: "true"},
					},
				})

				_, err := family.plugin.DB(config)
				if err == nil || !strings.Contains(err.Error(), "allowAllFiles is disabled in server mode") {
					t.Fatalf("DB should reject unrestricted LOCAL INFILE before connecting: %v", err)
				}

				_, err = family.plugin.RawExecute(config, "SELECT 1")
				if err == nil || !strings.Contains(err.Error(), "allowAllFiles is disabled in server mode") {
					t.Fatalf("RawExecute should reject unrestricted LOCAL INFILE before connecting: %v", err)
				}
			})
		}
	}
}
