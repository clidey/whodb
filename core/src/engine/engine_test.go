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
	"maps"
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
	if len(model.Metadata) != 2 {
		t.Fatalf("expected attributes to be copied, got %d", len(model.Metadata))
	}
	if model.Kind != "Table" {
		t.Fatalf("expected default kind Table, got %s", model.Kind)
	}
	if model.Ref == nil || len(model.Ref.Path) != 1 || model.Ref.Path[0] != "users" {
		t.Fatalf("expected source object ref path to include storage unit name")
	}
}

func TestChooseResolvesDisplayTypesToUnderlyingPlugins(t *testing.T) {
	original := make(map[DatabaseType]DatabaseType)
	maps.Copy(original, pluginTypeAliases)
	defer func() {
		pluginTypeAliases = original
	}()

	pluginTypeAliases = map[DatabaseType]DatabaseType{}

	engine := &Engine{}
	postgres := &Plugin{Type: DatabaseType_Postgres}
	mysql := &Plugin{Type: DatabaseType_MySQL}
	mongo := &Plugin{Type: DatabaseType_MongoDB}
	redis := &Plugin{Type: DatabaseType_Redis}
	elastic := &Plugin{Type: DatabaseType_ElasticSearch}
	engine.RegistryPlugin(postgres)
	engine.RegistryPlugin(mysql)
	engine.RegistryPlugin(mongo)
	engine.RegistryPlugin(redis)
	engine.RegistryPlugin(elastic)

	tests := []struct {
		alias  DatabaseType
		expect *Plugin
	}{
		// Redis aliases
		{DatabaseType_ElastiCache, redis},
		{DatabaseType_Valkey, redis},
		{DatabaseType_Dragonfly, redis},
		// MongoDB aliases
		{DatabaseType_DocumentDB, mongo},
		{DatabaseType_FerretDB, mongo},
		// ElasticSearch aliases
		{DatabaseType_OpenSearch, elastic},
		// MySQL aliases
		{DatabaseType_StarRocks, mysql},
		// Postgres aliases
		{DatabaseType_YugabyteDB, postgres},
	}

	for _, tt := range tests {
		RegisterPluginTypeAlias(tt.alias, tt.expect.Type)
	}

	for _, tt := range tests {
		t.Run(string(tt.alias), func(t *testing.T) {
			if got := engine.Choose(tt.alias); got != tt.expect {
				t.Fatalf("expected %s to resolve to %s plugin", tt.alias, tt.expect.Type)
			}
		})
	}
}

func TestRegisterPluginTypeAlias(t *testing.T) {
	// Clean up after the test
	original := make(map[DatabaseType]DatabaseType)
	maps.Copy(original, pluginTypeAliases)
	defer func() {
		pluginTypeAliases = original
	}()

	engine := &Engine{}
	postgres := &Plugin{Type: DatabaseType_Postgres}
	engine.RegistryPlugin(postgres)

	customDB := DatabaseType("CustomDB")

	// Before registration, CustomDB should not resolve
	if got := engine.Choose(customDB); got != nil {
		t.Fatalf("expected nil before alias registration, got %s", got.Type)
	}

	// Register alias and verify it resolves
	RegisterPluginTypeAlias(customDB, DatabaseType_Postgres)
	if got := engine.Choose(customDB); got != postgres {
		t.Fatalf("expected CustomDB to resolve to Postgres after registration")
	}
}

func typesDatabaseCredentials(username string) types.DatabaseCredentials {
	return types.DatabaseCredentials{Username: username}
}
