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

// Package querysuggestions builds simple database-aware query suggestions from
// source storage-unit metadata so GraphQL and the CLI can share the same
// onboarding prompts.
package querysuggestions

import (
	"fmt"

	"github.com/clidey/whodb/core/src/source"
)

const maxSuggestions = 3

// Suggestion is a single database-aware query suggestion.
type Suggestion struct {
	Description string
	Category    string
}

// FromStorageUnits converts storage units into deterministic suggestions that
// mention only tables that actually exist.
func FromStorageUnits(units []source.StorageUnit) []Suggestion {
	if len(units) == 0 {
		return []Suggestion{}
	}

	if len(units) > maxSuggestions {
		units = units[:maxSuggestions]
	}

	suggestions := make([]Suggestion, 0, len(units))
	for i, unit := range units {
		tableName := unit.Name
		suggestion := Suggestion{}
		switch i % 3 {
		case 0:
			suggestion = Suggestion{
				Description: fmt.Sprintf("What are the most recent records in %s?", tableName),
				Category:    "SELECT",
			}
		case 1:
			suggestion = Suggestion{
				Description: fmt.Sprintf("How many total entries are in %s?", tableName),
				Category:    "AGGREGATE",
			}
		default:
			suggestion = Suggestion{
				Description: "Show me all the data in " + tableName,
				Category:    "SELECT",
			}
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}
