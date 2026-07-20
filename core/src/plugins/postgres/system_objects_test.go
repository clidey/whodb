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

package postgres

import (
	"errors"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func systemObjectsTestPlugin(t *testing.T) *PostgresPlugin {
	t.Helper()

	plugin, ok := NewPostgresPlugin().PluginFunctions.(*PostgresPlugin)
	if !ok {
		t.Fatalf("unexpected postgres plugin type %T", NewPostgresPlugin().PluginFunctions)
	}
	return plugin
}

func hasSystemObjectAttribute(unit engine.StorageUnit) bool {
	for _, attribute := range unit.Attributes {
		if attribute.Key == SystemObjectAttributeKey {
			return attribute.Value == "true"
		}
	}
	return false
}

func TestMarkSystemObjectsAppendsAttributeToClassifiedUnits(t *testing.T) {
	plugin := systemObjectsTestPlugin(t)
	plugin.classifySystemObjects = func(config *engine.PluginConfig, schema string) (map[string]bool, error) {
		if schema != "public" {
			t.Fatalf("expected classification for schema %q, got %q", "public", schema)
		}
		return map[string]bool{"postgres_log": true}, nil
	}

	units := []engine.StorageUnit{
		{Name: "orders", Attributes: []engine.Record{{Key: "Type", Value: "BASE TABLE"}}},
		{Name: "postgres_log", Attributes: []engine.Record{{Key: "Type", Value: "BASE TABLE"}}},
	}

	plugin.markSystemObjects(engine.NewPluginConfig(&engine.Credentials{}), "public", units)

	if hasSystemObjectAttribute(units[0]) {
		t.Fatal("expected user table to stay unmarked")
	}
	if !hasSystemObjectAttribute(units[1]) {
		t.Fatal("expected classified unit to carry the System Object attribute")
	}
	if units[1].Attributes[0].Key != "Type" {
		t.Fatal("expected existing attributes to be preserved")
	}
}

func TestMarkSystemObjectsFailsOpenOnClassificationError(t *testing.T) {
	plugin := systemObjectsTestPlugin(t)
	plugin.classifySystemObjects = func(config *engine.PluginConfig, schema string) (map[string]bool, error) {
		return nil, errors.New("catalog incompatibility")
	}

	units := []engine.StorageUnit{
		{Name: "orders", Attributes: []engine.Record{{Key: "Type", Value: "BASE TABLE"}}},
		{Name: "postgres_log", Attributes: []engine.Record{{Key: "Type", Value: "BASE TABLE"}}},
	}

	plugin.markSystemObjects(engine.NewPluginConfig(&engine.Credentials{}), "public", units)

	for _, unit := range units {
		if hasSystemObjectAttribute(unit) {
			t.Fatalf("expected no unit to be marked when classification fails, %q was", unit.Name)
		}
	}
}

func TestMarkSystemObjectsLeavesUnitsUntouchedWhenNoneClassified(t *testing.T) {
	plugin := systemObjectsTestPlugin(t)
	plugin.classifySystemObjects = func(config *engine.PluginConfig, schema string) (map[string]bool, error) {
		return map[string]bool{}, nil
	}

	units := []engine.StorageUnit{
		{Name: "orders", Attributes: []engine.Record{{Key: "Type", Value: "BASE TABLE"}}},
	}

	plugin.markSystemObjects(engine.NewPluginConfig(&engine.Credentials{}), "public", units)

	if hasSystemObjectAttribute(units[0]) {
		t.Fatal("expected no marks when classification returns an empty set")
	}
	if len(units[0].Attributes) != 1 {
		t.Fatalf("expected attributes untouched, got %d entries", len(units[0].Attributes))
	}
}
