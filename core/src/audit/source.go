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

package audit

import "strings"

// SourceResource builds a consistent audit resource for one source action.
func SourceResource(sourceType string, sourceID *string) Resource {
	resource := Resource{
		Type: "source",
		ID:   strings.TrimSpace(sourceType),
		Name: strings.TrimSpace(sourceType),
	}
	if sourceID != nil && strings.TrimSpace(*sourceID) != "" {
		resource.ID = strings.TrimSpace(*sourceID)
	}
	return resource
}

// QueryOperation returns the normalized operation type for a source query.
func QueryOperation(query string) string {
	fields := strings.Fields(strings.TrimSpace(query))
	if len(fields) == 0 {
		return "unknown"
	}
	return strings.ToLower(fields[0])
}
