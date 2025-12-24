// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugins

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestGetDefaultPort(t *testing.T) {
	cases := map[engine.DatabaseType]string{
		engine.DatabaseType_Postgres:      "5432",
		engine.DatabaseType_MySQL:         "3306",
		engine.DatabaseType_MariaDB:       "3306",
		engine.DatabaseType_Sqlite3:       "0",
		engine.DatabaseType_ElasticSearch: "9200",
		engine.DatabaseType_Redis:         "6379",
	}

	for dbType, expected := range cases {
		port, ok := GetDefaultPort(dbType)
		if !ok {
			t.Fatalf("expected default port to exist for %s", dbType)
		}
		if port != expected {
			t.Fatalf("expected default port %s for %s, got %s", expected, dbType, port)
		}
	}

	if _, ok := GetDefaultPort("unknown"); ok {
		t.Fatalf("expected unknown database type to return ok=false")
	}
}

func TestRegisterDatabasePort(t *testing.T) {
	original := additionalPorts
	additionalPorts = make(map[engine.DatabaseType]string)
	t.Cleanup(func() {
		additionalPorts = original
	})

	customType := engine.DatabaseType("CustomDB")
	RegisterDatabasePort(customType, "7777")

	port, ok := GetDefaultPort(customType)
	if !ok || port != "7777" {
		t.Fatalf("expected registered port 7777 to be returned, got %s", port)
	}
}
