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

package mockdata

import (
	"strings"

	"github.com/clidey/whodb/core/src/env"
)

// IsMockDataGenerationAllowed checks if mock data generation is allowed for the given table.
// Respects the WHODB_DISABLE_MOCK_DATA_GENERATION environment variable:
//   - Empty: all tables allowed
//   - "*": all tables disabled
//   - Comma-separated list: specific tables disabled
func IsMockDataGenerationAllowed(tableName string) bool {
	if env.DisableMockDataGeneration == "" {
		return true
	}

	if env.DisableMockDataGeneration == "*" {
		return false
	}

	disabledTables := strings.Split(env.DisableMockDataGeneration, ",")
	for _, disabled := range disabledTables {
		if strings.TrimSpace(disabled) == tableName {
			return false
		}
	}

	return true
}

// GetMockDataGenerationMaxRowCount returns the maximum number of rows that can be
// generated in a single mock data generation request.
func GetMockDataGenerationMaxRowCount() int {
	return 200
}
