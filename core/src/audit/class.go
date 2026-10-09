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

import (
	"strings"
	"sync"
)

// Class is the storage-policy label of an audit action. It tells an audit
// backend what guarantees an event needs; it is not part of the event payload.
type Class string

const (
	// ClassCompliance marks security-relevant changes and decisions that belong
	// in a durable, tamper-evident ledger. Unknown actions default to this class.
	ClassCompliance Class = "compliance"
	// ClassAccess marks reads of customer data, including exports and
	// downloads. It needs the same guarantees as ClassCompliance.
	ClassAccess Class = "access"
	// ClassTelemetry marks request and lifecycle noise that a backend may store
	// separately, sample, or drop.
	ClassTelemetry Class = "telemetry"
)

var (
	classMu       sync.RWMutex
	classExact    = map[string]Class{}
	classPrefixes = map[string]Class{}
	defaultExact  = map[string]Class{
		"http.request":                    ClassTelemetry,
		"source.open_session":             ClassTelemetry,
		"source.invalidate_session":       ClassTelemetry,
		"source.shutdown_drivers":         ClassTelemetry,
		"source.check_availability":       ClassTelemetry,
		"source.list_objects":             ClassTelemetry,
		"source.list_objects_page":        ClassTelemetry,
		"source.get_object":               ClassTelemetry,
		"source.columns":                  ClassTelemetry,
		"source.columns_batch":            ClassTelemetry,
		"source.field_constraints":        ClassTelemetry,
		"source.ssl_status":               ClassTelemetry,
		"source.connection_field_options": ClassTelemetry,
		"source.query_suggestions":        ClassTelemetry,
		"source.read_rows":                ClassAccess,
		"source.run_query":                ClassAccess,
		"source.read_content":             ClassAccess,
		"source.read_graph":               ClassAccess,
		"source.download_content":         ClassAccess,
		"source.export_rows":              ClassAccess,
		"source.export_rows_ndjson":       ClassAccess,
	}
	defaultPrefixes = map[string]Class{
		"graphql.query.":        ClassTelemetry,
		"graphql.subscription.": ClassTelemetry,
	}
)

func init() {
	RegisterClassification(defaultExact, defaultPrefixes)
}

// Classify returns the class of an audit action. An exact entry wins over any
// prefix entry, the longest matching prefix wins among prefixes, and actions
// with no entry are ClassCompliance.
func Classify(action string) Class {
	classMu.RLock()
	defer classMu.RUnlock()

	if class, ok := classExact[action]; ok {
		return class
	}
	matched := ""
	class := ClassCompliance
	for prefix, prefixClass := range classPrefixes {
		if len(prefix) > len(matched) && strings.HasPrefix(action, prefix) {
			matched = prefix
			class = prefixClass
		}
	}
	return class
}

// RegisterClassification merges exact-action and action-prefix entries into
// the classification tables. Later registrations override earlier ones for the
// same key. Either map may be nil.
func RegisterClassification(exact map[string]Class, prefixes map[string]Class) {
	classMu.Lock()
	defer classMu.Unlock()

	for action, class := range exact {
		classExact[action] = class
	}
	for prefix, class := range prefixes {
		classPrefixes[prefix] = class
	}
}
