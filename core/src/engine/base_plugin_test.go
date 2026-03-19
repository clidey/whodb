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

package engine

import (
	"context"
	"errors"
	"testing"
)

func TestBasePluginSatisfiesInterface(t *testing.T) {
	// Compile-time check is in base_plugin.go; this confirms it at runtime too
	var _ PluginFunctions = (*BasePlugin)(nil)
}

func TestBasePluginUserFacingMethodsReturnUnsupported(t *testing.T) {
	bp := &BasePlugin{}
	config := &PluginConfig{}

	if _, err := bp.RawExecute(config, "SELECT 1"); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("RawExecute should return ErrUnsupported, got %v", err)
	}
	if _, err := bp.Chat(config, "", "", ""); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Chat should return ErrUnsupported, got %v", err)
	}
	if _, err := bp.GetGraph(config, ""); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("GetGraph should return ErrUnsupported, got %v", err)
	}
	if _, err := bp.GetAllSchemas(config); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("GetAllSchemas should return ErrUnsupported, got %v", err)
	}
}

func TestBasePluginInternalMethodsReturnEmpty(t *testing.T) {
	bp := &BasePlugin{}
	config := &PluginConfig{}

	constraints, err := bp.GetColumnConstraints(config, "", "")
	if err != nil {
		t.Fatalf("GetColumnConstraints should return nil error, got %v", err)
	}
	if len(constraints) != 0 {
		t.Fatalf("GetColumnConstraints should return empty map, got %v", constraints)
	}

	fks, err := bp.GetForeignKeyRelationships(config, "", "")
	if err != nil {
		t.Fatalf("GetForeignKeyRelationships should return nil error, got %v", err)
	}
	if len(fks) != 0 {
		t.Fatalf("GetForeignKeyRelationships should return empty map, got %v", fks)
	}

	if err := bp.NullifyFKColumn(config, "", "", ""); err != nil {
		t.Fatalf("NullifyFKColumn should return nil, got %v", err)
	}

	if err := bp.MarkGeneratedColumns(config, "", "", nil); err != nil {
		t.Fatalf("MarkGeneratedColumns should return nil, got %v", err)
	}
}

func TestBasePluginWithTransactionCallsOperationDirectly(t *testing.T) {
	bp := &BasePlugin{}
	called := false
	err := bp.WithTransaction(nil, func(tx any) error {
		called = true
		if tx != nil {
			t.Fatalf("WithTransaction should pass nil tx, got %v", tx)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTransaction should return nil, got %v", err)
	}
	if !called {
		t.Fatalf("WithTransaction should call the operation")
	}
}

func TestBasePluginFormatValue(t *testing.T) {
	bp := &BasePlugin{}
	if bp.FormatValue(42) != "42" {
		t.Fatalf("FormatValue(42) should return \"42\", got %q", bp.FormatValue(42))
	}
	if bp.FormatValue("hello") != "hello" {
		t.Fatalf("FormatValue(\"hello\") should return \"hello\", got %q", bp.FormatValue("hello"))
	}
}

func TestBasePluginIsAvailableReturnsFalse(t *testing.T) {
	bp := &BasePlugin{}
	if bp.IsAvailable(context.Background(), nil) {
		t.Fatalf("IsAvailable should return false by default")
	}
}

func TestBasePluginMetadataReturnsNil(t *testing.T) {
	bp := &BasePlugin{}
	if bp.GetDatabaseMetadata() != nil {
		t.Fatalf("GetDatabaseMetadata should return nil by default")
	}
	status, err := bp.GetSSLStatus(nil)
	if err != nil || status != nil {
		t.Fatalf("GetSSLStatus should return nil, nil; got %v, %v", status, err)
	}
}

// TestBasePluginEmbedding verifies that a plugin can embed BasePlugin
// and override only the methods it needs.
func TestBasePluginEmbedding(t *testing.T) {
	type TestPlugin struct {
		BasePlugin
	}

	tp := &TestPlugin{}

	// Should inherit BasePlugin defaults
	if _, err := tp.RawExecute(nil, ""); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("embedded BasePlugin should return ErrUnsupported for RawExecute")
	}

	// Should satisfy the interface
	var _ PluginFunctions = tp
}
