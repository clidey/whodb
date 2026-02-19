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

package gorm_plugin

import (
	"database/sql"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

// newTestPlugin creates a GormPlugin suitable for unit testing.
// It uses the default GormPlugin implementations (no DB needed).
func newTestPlugin() *GormPlugin {
	p := &GormPlugin{}
	p.Type = engine.DatabaseType_Postgres // type doesn't matter for conversion tests
	p.GormPluginFunctions = p             // self-referencing so interface calls resolve to defaults
	return p
}

func TestConvertStringValue_NullableCaseInsensitive(t *testing.T) {
	p := newTestPlugin()

	tests := []struct {
		name       string
		value      string
		columnType string
		wantErr    bool
		checkVal   func(t *testing.T, val any)
	}{
		{
			name:       "Nullable(Int32) mixed case",
			value:      "100",
			columnType: "Nullable(Int32)",
			checkVal: func(t *testing.T, val any) {
				nv, ok := val.(sql.NullInt64)
				if !ok {
					t.Fatalf("expected sql.NullInt64, got %T", val)
				}
				if !nv.Valid || nv.Int64 != 100 {
					t.Fatalf("expected NullInt64{100, true}, got %+v", nv)
				}
			},
		},
		{
			name:       "NULLABLE(INT32) all uppercase",
			value:      "100",
			columnType: "NULLABLE(INT32)",
			checkVal: func(t *testing.T, val any) {
				nv, ok := val.(sql.NullInt64)
				if !ok {
					t.Fatalf("expected sql.NullInt64, got %T", val)
				}
				if !nv.Valid || nv.Int64 != 100 {
					t.Fatalf("expected NullInt64{100, true}, got %+v", nv)
				}
			},
		},
		{
			name:       "NULLABLE(INT32) null value",
			value:      "",
			columnType: "NULLABLE(INT32)",
			checkVal: func(t *testing.T, val any) {
				nv, ok := val.(sql.NullInt64)
				if !ok {
					t.Fatalf("expected sql.NullInt64, got %T", val)
				}
				if nv.Valid {
					t.Fatalf("expected NullInt64 with Valid=false, got %+v", nv)
				}
			},
		},
		{
			name:       "plain INT32 without nullable",
			value:      "42",
			columnType: "INT32",
			checkVal: func(t *testing.T, val any) {
				iv, ok := val.(int64)
				if !ok {
					t.Fatalf("expected int64, got %T", val)
				}
				if iv != 42 {
					t.Fatalf("expected 42, got %d", iv)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := p.ConvertStringValue(tt.value, tt.columnType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.checkVal != nil {
				tt.checkVal(t, val)
			}
		})
	}
}

func TestConvertArrayValue_NoInfiniteRecursion(t *testing.T) {
	p := newTestPlugin()

	// This previously caused infinite recursion (stack overflow) because
	// convertArrayValue used case-sensitive "Array(" prefix but received
	// uppercased "ARRAY(INT32)" from ConvertStringValue.
	tests := []struct {
		name       string
		value      string
		columnType string
		wantLen    int
		wantErr    bool
	}{
		{
			name:       "ARRAY(INT32) uppercase",
			value:      "[100, 200]",
			columnType: "ARRAY(INT32)",
			wantLen:    2,
		},
		{
			name:       "Array(Int32) mixed case",
			value:      "[1, 2, 3]",
			columnType: "Array(Int32)",
			wantLen:    3,
		},
		{
			name:       "ARRAY(INT32) empty",
			value:      "[]",
			columnType: "ARRAY(INT32)",
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := p.ConvertStringValue(tt.value, tt.columnType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			arr, ok := val.([]any)
			if !ok {
				t.Fatalf("expected []any, got %T", val)
			}
			if len(arr) != tt.wantLen {
				t.Fatalf("expected %d elements, got %d: %v", tt.wantLen, len(arr), arr)
			}
		})
	}
}
