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

package engine

import (
	"testing"

	"github.com/clidey/whodb/core/src/types"
)

func TestRegistryPluginAndChoose(t *testing.T) {
	engine := &Engine{}
	postgres := &Plugin{Type: DatabaseType_Postgres}
	engine.RegistryPlugin(postgres)

	if got := engine.Choose(DatabaseType("postgres")); got != postgres {
		t.Fatalf("Choose should be case-insensitive and return the registered plugin")
	}

	if got := engine.Choose(DatabaseType("unknown")); got != nil {
		t.Fatalf("Choose should return nil for an unregistered database type")
	}
}

func TestAddLoginProfileAndRetriever(t *testing.T) {
	engine := &Engine{}

	engine.AddLoginProfile(typesDatabaseCredentials("first"))
	engine.RegisterProfileRetriever(func() ([]types.DatabaseCredentials, error) {
		return []types.DatabaseCredentials{typesDatabaseCredentials("retrieved")}, nil
	})

	if len(engine.LoginProfiles) != 1 {
		t.Fatalf("expected 1 stored profile, got %d", len(engine.LoginProfiles))
	}

	if len(engine.ProfileRetrievers) != 1 {
		t.Fatalf("expected 1 profile retriever, got %d", len(engine.ProfileRetrievers))
	}
}

func TestGetStorageUnitModel(t *testing.T) {
	unit := StorageUnit{
		Name: "users",
		Attributes: []Record{
			{Key: "owner", Value: "admin"},
			{Key: "ttl", Value: "3600"},
		},
	}

	model := GetStorageUnitModel(unit)
	if model.Name != "users" {
		t.Fatalf("expected name to be carried over")
	}
	if len(model.Attributes) != 2 {
		t.Fatalf("expected attributes to be copied, got %d", len(model.Attributes))
	}
	if model.IsMockDataGenerationAllowed {
		t.Fatalf("IsMockDataGenerationAllowed is set by resolver and should default to false")
	}
}

func typesDatabaseCredentials(username string) types.DatabaseCredentials {
	return types.DatabaseCredentials{Username: username}
}
