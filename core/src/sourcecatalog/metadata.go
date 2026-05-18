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

package sourcecatalog

import (
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog/specs"
)

var (
	sessionMetadataMu    sync.RWMutex
	sessionMetadataSpecs = map[string]source.TypeSessionMetadata{}

	objectCreationMetadataMu    sync.RWMutex
	objectCreationMetadataSpecs = map[string]source.ObjectCreationMetadata{}

	discoveryPrefillMu    sync.RWMutex
	discoveryPrefillSpecs = map[string]source.DiscoveryPrefill{}
)

func init() {
	registerSessionMetadata()

	RegisterDiscoveryPrefill(connectorPostgres, source.DiscoveryPrefill{
		AdvancedDefaults: []source.DiscoveryAdvancedDefault{
			{Key: "SSL Mode", Value: "require"},
		},
	})
	RegisterDiscoveryPrefill(connectorMySQL, source.DiscoveryPrefill{
		AdvancedDefaults: []source.DiscoveryAdvancedDefault{
			{Key: "SSL Mode", Value: "require"},
		},
	})
	RegisterDiscoveryPrefill(connectorMariaDB, source.DiscoveryPrefill{
		AdvancedDefaults: []source.DiscoveryAdvancedDefault{
			{Key: "SSL Mode", Value: "require"},
		},
	})
	RegisterDiscoveryPrefill("ElastiCache", source.DiscoveryPrefill{
		AdvancedDefaults: []source.DiscoveryAdvancedDefault{
			{
				Key:        "SSL Mode",
				Value:      "enabled",
				Conditions: []source.DiscoveryMetadataCondition{{Key: "transitEncryption", Value: "true"}},
			},
		},
	})
	RegisterDiscoveryPrefill("Valkey", source.DiscoveryPrefill{
		AdvancedDefaults: []source.DiscoveryAdvancedDefault{
			{
				Key:        "SSL Mode",
				Value:      "enabled",
				Conditions: []source.DiscoveryMetadataCondition{{Key: "transitEncryption", Value: "true"}},
			},
		},
	})
	RegisterDiscoveryPrefill("DocumentDB", source.DiscoveryPrefill{
		AdvancedDefaults: []source.DiscoveryAdvancedDefault{
			{
				Key:   "URL Params",
				Value: "?tls=true&tlsInsecure=true&replicaSet=rs0&retryWrites=false&readPreference=secondaryPreferred",
			},
		},
	})
	RegisterDiscoveryPrefill(connectorRedis, source.DiscoveryPrefill{
		AdvancedDefaults: []source.DiscoveryAdvancedDefault{
			{Key: "SSL Mode", Value: "enabled", ProviderTypes: []string{"Azure"}},
		},
	})
	RegisterDiscoveryPrefill(connectorMongoDB, source.DiscoveryPrefill{
		AdvancedDefaults: []source.DiscoveryAdvancedDefault{
			{
				Key:           "URL Params",
				Value:         "?tls=true&tlsInsecure=true&retryWrites=false",
				ProviderTypes: []string{"Azure"},
			},
		},
	})
}

func registerSessionMetadata() {
	RegisterSessionMetadataAliases(
		SessionMetadataFromOperatorMap(specs.PostgresTypeDefinitions, specs.PostgreSQLSupportedOperators, specs.PostgresAliasMap),
		string(engine.DatabaseType_Postgres),
		string(engine.DatabaseType_YugabyteDB),
	)
	RegisterSessionMetadata(
		string(engine.DatabaseType_QuestDB),
		SessionMetadataFromOperatorMap(specs.QuestDBTypeDefinitions, specs.PostgreSQLSupportedOperators, specs.QuestDBAliasMap),
	)
	RegisterSessionMetadata(
		string(engine.DatabaseType_CockroachDB),
		SessionMetadataFromOperatorMap(specs.CockroachDBTypeDefinitions, specs.PostgreSQLSupportedOperators, specs.PostgresAliasMap),
	)
	RegisterSessionMetadataAliases(
		SessionMetadataFromOperatorMap(specs.MySQLTypeDefinitions, specs.MySQLSupportedOperators, specs.MySQLAliasMap),
		string(engine.DatabaseType_MySQL),
		string(engine.DatabaseType_MariaDB),
		string(engine.DatabaseType_TiDB),
	)
	RegisterSessionMetadata(
		string(engine.DatabaseType_ClickHouse),
		SessionMetadataFromOperatorMap(specs.ClickHouseTypeDefinitions, specs.ClickHouseSupportedOperators, specs.ClickHouseAliasMap),
	)
	RegisterSessionMetadata(
		string(engine.DatabaseType_Sqlite3),
		SessionMetadataFromOperatorMap(specs.SQLiteTypeDefinitions, specs.SQLiteSupportedOperators, specs.SQLiteAliasMap),
	)
	RegisterSessionMetadata(
		string(engine.DatabaseType_DuckDB),
		SessionMetadataFromOperatorMap(specs.DuckDBTypeDefinitions, specs.DuckDBSupportedOperators, specs.DuckDBAliasMap),
	)
	RegisterSessionMetadata(
		string(engine.DatabaseType_ElasticSearch),
		SessionMetadataFromOperatorMap(specs.ElasticSearchTypeDefinitions, specs.ElasticSearchSupportedOperators, nil),
	)
	RegisterSessionMetadata(
		string(engine.DatabaseType_MongoDB),
		SessionMetadataFromOperatorMap(specs.MongoDBTypeDefinitions, specs.MongoDBSupportedOperators, nil),
	)
	RegisterSessionMetadata(
		string(engine.DatabaseType_Redis),
		SessionMetadataFromOperatorMap(nil, specs.RedisOperators, nil),
	)
	RegisterSessionMetadata(
		string(engine.DatabaseType_Memcached),
		SessionMetadataFromOperatorMap(nil, specs.MemcachedOperators, nil),
	)

	RegisterObjectCreationMetadataAliases(
		relationalObjectCreationMetadata(specs.PostgresTypeDefinitions, "GENERATED ALWAYS AS IDENTITY"),
		string(engine.DatabaseType_Postgres),
		string(engine.DatabaseType_YugabyteDB),
	)
	RegisterObjectCreationMetadata(
		string(engine.DatabaseType_CockroachDB),
		relationalObjectCreationMetadata(specs.CockroachDBTypeDefinitions, "GENERATED ALWAYS AS IDENTITY"),
	)
	RegisterObjectCreationMetadataAliases(
		relationalObjectCreationMetadata(specs.MySQLTypeDefinitions, "AUTO_INCREMENT"),
		string(engine.DatabaseType_MySQL),
		string(engine.DatabaseType_MariaDB),
		string(engine.DatabaseType_TiDB),
	)
	RegisterObjectCreationMetadata(
		string(engine.DatabaseType_ClickHouse),
		source.ObjectCreationMetadata{
			Supported:       true,
			ObjectKind:      source.ObjectKindTable,
			RequiresColumns: true,
			TypeDefinitions: slices.Clone(specs.ClickHouseTypeDefinitions),
			ColumnCapabilities: source.ColumnCreationCapabilities{
				Types:        true,
				Nullable:     true,
				PrimaryKey:   true,
				DefaultValue: true,
				CheckValues:  true,
				CheckMinMax:  true,
			},
			ColumnLabels:      sqlColumnCreationLabels(""),
			TableCapabilities: source.TableCreationCapabilities{OrderKey: true},
		},
	)
	RegisterObjectCreationMetadata(
		string(engine.DatabaseType_Sqlite3),
		sqliteObjectCreationMetadata(specs.SQLiteTypeDefinitions),
	)
	RegisterObjectCreationMetadata(
		string(engine.DatabaseType_DuckDB),
		relationalObjectCreationMetadata(specs.DuckDBTypeDefinitions, "DEFAULT nextval()"),
	)
	RegisterObjectCreationMetadata(
		string(engine.DatabaseType_QuestDB),
		relationalObjectCreationMetadata(specs.QuestDBTypeDefinitions, ""),
	)
	RegisterObjectCreationMetadata(
		string(engine.DatabaseType_ElasticSearch),
		source.ObjectCreationMetadata{
			Supported:       true,
			ObjectKind:      source.ObjectKindIndex,
			RequiresColumns: false,
			TypeDefinitions: slices.Clone(specs.ElasticSearchTypeDefinitions),
			ColumnCapabilities: source.ColumnCreationCapabilities{
				Types: true,
			},
		},
	)
	RegisterObjectCreationMetadata(
		string(engine.DatabaseType_MongoDB),
		source.ObjectCreationMetadata{
			Supported:       true,
			ObjectKind:      source.ObjectKindCollection,
			RequiresColumns: false,
			TypeDefinitions: slices.Clone(specs.MongoDBTypeDefinitions),
			ColumnCapabilities: source.ColumnCreationCapabilities{
				Types:    true,
				Nullable: true,
			},
		},
	)
	RegisterObjectCreationMetadata(
		string(engine.DatabaseType_Redis),
		source.ObjectCreationMetadata{
			Supported:       true,
			ObjectKind:      source.ObjectKindKey,
			RequiresColumns: false,
			TableCapabilities: source.TableCreationCapabilities{
				KeyValueType: true,
			},
			TableOptions: []source.CreationOptionDefinition{{
				Key:      "type",
				Label:    "Type",
				Required: true,
				Values:   []string{"string", "hash", "list", "set", "zset"},
			}},
		},
	)
}

// RegisterSessionMetadata registers source-owned editor/query metadata for one
// source type or connector id.
func RegisterSessionMetadata(id string, metadata source.TypeSessionMetadata) {
	if strings.TrimSpace(id) == "" {
		return
	}

	sessionMetadataMu.Lock()
	defer sessionMetadataMu.Unlock()
	sessionMetadataSpecs[strings.ToLower(id)] = cloneSessionMetadata(metadata)
}

// RegisterSessionMetadataAliases registers the same source-owned editor/query
// metadata for multiple source type or connector ids.
func RegisterSessionMetadataAliases(metadata source.TypeSessionMetadata, ids ...string) {
	for _, id := range ids {
		RegisterSessionMetadata(id, metadata)
	}
}

// ResolveSessionMetadata resolves source-owned editor/query metadata for the
// first matching source type or connector id.
func ResolveSessionMetadata(ids ...string) (*source.TypeSessionMetadata, bool) {
	sessionMetadataMu.RLock()
	defer sessionMetadataMu.RUnlock()

	for _, id := range ids {
		metadata, ok := sessionMetadataSpecs[strings.ToLower(strings.TrimSpace(id))]
		if !ok {
			continue
		}
		cloned := cloneSessionMetadata(metadata)
		return &cloned, true
	}

	return nil, false
}

// RegisterObjectCreationMetadata registers source-owned create-object metadata
// for one source type or connector id.
func RegisterObjectCreationMetadata(id string, metadata source.ObjectCreationMetadata) {
	if strings.TrimSpace(id) == "" {
		return
	}

	objectCreationMetadataMu.Lock()
	defer objectCreationMetadataMu.Unlock()
	metadata.ColumnLabels = source.ColumnCreationLabelsWithDefaults(metadata.ColumnLabels)
	objectCreationMetadataSpecs[strings.ToLower(id)] = cloneObjectCreationMetadata(metadata)
}

// RegisterObjectCreationMetadataAliases registers the same create-object
// metadata for multiple source type or connector ids.
func RegisterObjectCreationMetadataAliases(metadata source.ObjectCreationMetadata, ids ...string) {
	for _, id := range ids {
		RegisterObjectCreationMetadata(id, metadata)
	}
}

// ResolveObjectCreationMetadata resolves create-object metadata for the first
// matching source type or connector id.
func ResolveObjectCreationMetadata(ids ...string) (source.ObjectCreationMetadata, bool) {
	objectCreationMetadataMu.RLock()
	defer objectCreationMetadataMu.RUnlock()

	for _, id := range ids {
		metadata, ok := objectCreationMetadataSpecs[strings.ToLower(strings.TrimSpace(id))]
		if !ok {
			continue
		}
		return cloneObjectCreationMetadata(metadata), true
	}

	return source.ObjectCreationMetadata{}, false
}

// RegisterDiscoveryPrefill registers source-owned discovered-resource prefill
// rules for one source type or connector id.
func RegisterDiscoveryPrefill(id string, prefill source.DiscoveryPrefill) {
	if strings.TrimSpace(id) == "" {
		return
	}

	discoveryPrefillMu.Lock()
	defer discoveryPrefillMu.Unlock()
	discoveryPrefillSpecs[strings.ToLower(id)] = cloneDiscoveryPrefill(prefill)
}

// RegisterDiscoveryPrefillAliases registers the same discovered-resource
// prefill rules for multiple source type or connector ids.
func RegisterDiscoveryPrefillAliases(prefill source.DiscoveryPrefill, ids ...string) {
	for _, id := range ids {
		RegisterDiscoveryPrefill(id, prefill)
	}
}

// ResolveDiscoveryPrefill resolves source-owned discovered-resource prefill
// rules for the first matching source type or connector id.
func ResolveDiscoveryPrefill(ids ...string) (source.DiscoveryPrefill, bool) {
	discoveryPrefillMu.RLock()
	defer discoveryPrefillMu.RUnlock()

	for _, id := range ids {
		prefill, ok := discoveryPrefillSpecs[strings.ToLower(strings.TrimSpace(id))]
		if !ok {
			continue
		}
		return cloneDiscoveryPrefill(prefill), true
	}

	return source.DiscoveryPrefill{}, false
}

// SessionMetadataFromOperators builds source-owned editor/query metadata from
// canonical type definitions, operator names, and type aliases.
func SessionMetadataFromOperators(
	typeDefinitions []source.TypeDefinition,
	operators []string,
	aliasMap map[string]string,
) source.TypeSessionMetadata {
	clonedOperators := slices.Clone(operators)
	sort.Strings(clonedOperators)
	return source.TypeSessionMetadata{
		TypeDefinitions: slices.Clone(typeDefinitions),
		Operators:       clonedOperators,
		AliasMap:        cloneAliasMap(aliasMap),
	}
}

// SessionMetadataFromOperatorMap builds source-owned editor/query metadata from
// a database operator lookup map.
func SessionMetadataFromOperatorMap(
	typeDefinitions []source.TypeDefinition,
	operators map[string]string,
	aliasMap map[string]string,
) source.TypeSessionMetadata {
	keys := make([]string, 0, len(operators))
	for key := range operators {
		keys = append(keys, key)
	}
	return SessionMetadataFromOperators(typeDefinitions, keys, aliasMap)
}

func cloneSessionMetadata(metadata source.TypeSessionMetadata) source.TypeSessionMetadata {
	return source.TypeSessionMetadata{
		TypeDefinitions: slices.Clone(metadata.TypeDefinitions),
		Operators:       slices.Clone(metadata.Operators),
		AliasMap:        cloneAliasMap(metadata.AliasMap),
	}
}

func cloneAliasMap(aliasMap map[string]string) map[string]string {
	if len(aliasMap) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(aliasMap))
	for key, value := range aliasMap {
		cloned[key] = value
	}
	return cloned
}

func cloneDiscoveryPrefill(prefill source.DiscoveryPrefill) source.DiscoveryPrefill {
	cloned := source.DiscoveryPrefill{
		AdvancedDefaults: make([]source.DiscoveryAdvancedDefault, 0, len(prefill.AdvancedDefaults)),
	}
	for _, item := range prefill.AdvancedDefaults {
		cloned.AdvancedDefaults = append(cloned.AdvancedDefaults, source.DiscoveryAdvancedDefault{
			Key:           item.Key,
			Value:         item.Value,
			MetadataKey:   item.MetadataKey,
			DefaultValue:  item.DefaultValue,
			ProviderTypes: slices.Clone(item.ProviderTypes),
			Conditions:    slices.Clone(item.Conditions),
		})
	}
	return cloned
}

func relationalObjectCreationMetadata(typeDefinitions []source.TypeDefinition, identityLabel string) source.ObjectCreationMetadata {
	return source.ObjectCreationMetadata{
		Supported:       true,
		ObjectKind:      source.ObjectKindTable,
		RequiresColumns: true,
		TypeDefinitions: slices.Clone(typeDefinitions),
		ColumnCapabilities: source.ColumnCreationCapabilities{
			Types:               true,
			Nullable:            true,
			PrimaryKey:          true,
			CompositePrimaryKey: true,
			Unique:              true,
			Identity:            true,
			DefaultValue:        true,
			CheckValues:         true,
			CheckMinMax:         true,
			ForeignKey:          true,
		},
		ColumnLabels: sqlColumnCreationLabels(identityLabel),
	}
}

func sqliteObjectCreationMetadata(typeDefinitions []source.TypeDefinition) source.ObjectCreationMetadata {
	metadata := relationalObjectCreationMetadata(typeDefinitions, "")
	metadata.ColumnCapabilities.Identity = false
	return metadata
}

func sqlColumnCreationLabels(identityLabel string) source.ColumnCreationLabels {
	return source.ColumnCreationLabelsWithDefaults(source.ColumnCreationLabels{
		Nullable:     "NULL",
		PrimaryKey:   "PRIMARY KEY",
		Unique:       "UNIQUE",
		Identity:     identityLabel,
		DefaultValue: "DEFAULT",
		CheckValues:  "CHECK IN",
		CheckMin:     "CHECK >=",
		CheckMax:     "CHECK <=",
		ForeignKey:   "REFERENCES",
	})
}

func cloneObjectCreationMetadata(metadata source.ObjectCreationMetadata) source.ObjectCreationMetadata {
	return source.ObjectCreationMetadata{
		Supported:          metadata.Supported,
		ObjectKind:         metadata.ObjectKind,
		RequiresColumns:    metadata.RequiresColumns,
		TypeDefinitions:    slices.Clone(metadata.TypeDefinitions),
		ColumnCapabilities: metadata.ColumnCapabilities,
		ColumnLabels:       source.ColumnCreationLabelsWithDefaults(metadata.ColumnLabels),
		TableCapabilities:  metadata.TableCapabilities,
		TableOptions:       cloneCreationOptionDefinitions(metadata.TableOptions),
	}
}

func cloneCreationOptionDefinitions(options []source.CreationOptionDefinition) []source.CreationOptionDefinition {
	cloned := make([]source.CreationOptionDefinition, 0, len(options))
	for _, option := range options {
		cloned = append(cloned, source.CreationOptionDefinition{
			Key:      option.Key,
			Label:    option.Label,
			Required: option.Required,
			Values:   slices.Clone(option.Values),
		})
	}
	return cloned
}
