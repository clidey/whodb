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
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestParseConnectionConfigUsesExplicitPort(t *testing.T) {
	plugin := newTestPlugin()

	input, err := plugin.ParseConnectionConfig(&engine.PluginConfig{
		Credentials: &engine.Credentials{
			Advanced: []engine.Record{{Key: "Port", Value: "5432"}},
		},
	})
	if err != nil {
		t.Fatalf("expected ParseConnectionConfig to succeed, got %v", err)
	}
	if input.Port != 5432 {
		t.Fatalf("expected explicit port 5432, got %d", input.Port)
	}
}

func TestParseConnectionConfigAllowsMissingPort(t *testing.T) {
	plugin := newTestPlugin()

	input, err := plugin.ParseConnectionConfig(&engine.PluginConfig{
		Credentials: &engine.Credentials{},
	})
	if err != nil {
		t.Fatalf("expected ParseConnectionConfig to allow missing port, got %v", err)
	}
	if input.Port != 0 {
		t.Fatalf("expected missing port to remain unset, got %d", input.Port)
	}
}

func TestParseConnectionConfigRejectsOutOfRangePort(t *testing.T) {
	plugin := newTestPlugin()

	for _, port := range []string{"0", "-1", "65536"} {
		t.Run(port, func(t *testing.T) {
			_, err := plugin.ParseConnectionConfig(&engine.PluginConfig{
				Credentials: &engine.Credentials{
					Advanced: []engine.Record{{Key: "Port", Value: port}},
				},
			})
			if err == nil {
				t.Fatalf("expected ParseConnectionConfig to reject port %s", port)
			}
		})
	}
}

func TestParseConnectionConfigNormalizesClickHouseToggleFields(t *testing.T) {
	plugin := newTestPlugin()
	plugin.Type = engine.DatabaseType_ClickHouse

	input, err := plugin.ParseConnectionConfig(&engine.PluginConfig{
		Credentials: &engine.Credentials{
			Advanced: []engine.Record{
				{Key: "Readonly", Value: "1"},
				{Key: "Debug", Value: "true"},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected ParseConnectionConfig to succeed, got %v", err)
	}
	if input.ReadOnly != "enable" {
		t.Fatalf("expected Readonly to normalize to enable, got %q", input.ReadOnly)
	}
	if input.Debug != "enable" {
		t.Fatalf("expected Debug to normalize to enable, got %q", input.Debug)
	}
}

func TestParseConnectionConfigRejectsInvalidClickHouseToggleValue(t *testing.T) {
	plugin := newTestPlugin()
	plugin.Type = engine.DatabaseType_ClickHouse

	_, err := plugin.ParseConnectionConfig(&engine.PluginConfig{
		Credentials: &engine.Credentials{
			Advanced: []engine.Record{{Key: "Readonly", Value: "maybe"}},
		},
	})
	if err == nil {
		t.Fatal("expected ParseConnectionConfig to reject invalid toggle value")
	}
}
