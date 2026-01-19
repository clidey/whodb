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

package aws

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func TestMapRDSEngine(t *testing.T) {
	testCases := []struct {
		engine   string
		expected engine.DatabaseType
		ok       bool
	}{
		{"mysql", engine.DatabaseType_MySQL, true},
		{"MySQL", engine.DatabaseType_MySQL, true},
		{"mariadb", engine.DatabaseType_MariaDB, true},
		{"postgres", engine.DatabaseType_Postgres, true},
		{"postgresql", engine.DatabaseType_Postgres, true},
		{"aurora-mysql", engine.DatabaseType_MySQL, true},
		{"aurora-postgresql", engine.DatabaseType_Postgres, true},
		{"sqlserver-se", "", false}, // SQL Server not in CE
		{"oracle-ee", "", false},    // Oracle not in CE
		{"docdb", "", false},        // DocumentDB handled separately
		{"unknown-engine", "", false},
	}

	for _, tc := range testCases {
		dbType, ok := mapRDSEngine(tc.engine)
		if ok != tc.ok {
			t.Errorf("mapRDSEngine(%s): expected ok=%v, got ok=%v", tc.engine, tc.ok, ok)
			continue
		}
		if ok && dbType != tc.expected {
			t.Errorf("mapRDSEngine(%s): expected %s, got %s", tc.engine, tc.expected, dbType)
		}
	}
}

func TestMapRDSStatus(t *testing.T) {
	testCases := []struct {
		status   string
		expected providers.ConnectionStatus
	}{
		{"available", providers.ConnectionStatusAvailable},
		{"Available", providers.ConnectionStatusAvailable},
		{"AVAILABLE", providers.ConnectionStatusAvailable},
		{"starting", providers.ConnectionStatusStarting},
		{"creating", providers.ConnectionStatusStarting},
		{"configuring-enhanced-monitoring", providers.ConnectionStatusStarting},
		{"modifying", providers.ConnectionStatusStarting},
		{"upgrading", providers.ConnectionStatusStarting},
		{"stopped", providers.ConnectionStatusStopped},
		{"stopping", providers.ConnectionStatusStopped},
		{"storage-optimization", providers.ConnectionStatusStopped},
		{"deleting", providers.ConnectionStatusDeleting},
		{"failed", providers.ConnectionStatusFailed},
		{"restore-error", providers.ConnectionStatusFailed},
		{"incompatible-credentials", providers.ConnectionStatusFailed},
		{"incompatible-parameters", providers.ConnectionStatusFailed},
		{"unknown-status", providers.ConnectionStatusUnknown},
		{"", providers.ConnectionStatusUnknown},
	}

	for _, tc := range testCases {
		status := tc.status
		result := mapRDSStatus(&status)
		if result != tc.expected {
			t.Errorf("mapRDSStatus(%s): expected %s, got %s", tc.status, tc.expected, result)
		}
	}

	// Test nil status
	result := mapRDSStatus(nil)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapRDSStatus(nil): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func TestMapRDSEngine_CaseInsensitive(t *testing.T) {
	engines := []string{"mysql", "MySQL", "MYSQL", "MysQL"}
	for _, eng := range engines {
		dbType, ok := mapRDSEngine(eng)
		if !ok {
			t.Errorf("mapRDSEngine(%s): expected to match", eng)
			continue
		}
		if dbType != engine.DatabaseType_MySQL {
			t.Errorf("mapRDSEngine(%s): expected MySQL, got %s", eng, dbType)
		}
	}
}

func TestMapRDSEngine_AuroraVariants(t *testing.T) {
	// Aurora MySQL variants
	mysqlVariants := []string{
		"aurora-mysql",
		"aurora-mysql-5.7",
		"aurora-mysql-8.0",
	}
	for _, eng := range mysqlVariants {
		dbType, ok := mapRDSEngine(eng)
		if !ok {
			t.Errorf("mapRDSEngine(%s): expected to match", eng)
			continue
		}
		if dbType != engine.DatabaseType_MySQL {
			t.Errorf("mapRDSEngine(%s): expected MySQL, got %s", eng, dbType)
		}
	}

	// Aurora PostgreSQL variants
	pgVariants := []string{
		"aurora-postgresql",
		"aurora-postgresql-13",
		"aurora-postgresql-14",
	}
	for _, eng := range pgVariants {
		dbType, ok := mapRDSEngine(eng)
		if !ok {
			t.Errorf("mapRDSEngine(%s): expected to match", eng)
			continue
		}
		if dbType != engine.DatabaseType_Postgres {
			t.Errorf("mapRDSEngine(%s): expected Postgres, got %s", eng, dbType)
		}
	}
}
