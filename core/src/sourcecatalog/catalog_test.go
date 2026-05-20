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

package sourcecatalog_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/clidey/whodb/core/src/dbcatalog"
	"github.com/clidey/whodb/core/src/source"
	coresourcecatalog "github.com/clidey/whodb/core/src/sourcecatalog"
)

func TestBuildTypeSpecCoversSharedDatabaseCatalog(t *testing.T) {
	t.Parallel()

	for _, entry := range dbcatalog.All() {
		t.Run(string(entry.ID), func(t *testing.T) {
			t.Parallel()

			spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
				ID:             string(entry.ID),
				Label:          entry.Label,
				Connector:      string(entry.PluginType),
				Extra:          maps.Clone(entry.Extra),
				Fields:         coresourcecatalog.FieldVisibility(entry.Fields),
				RequiredFields: coresourcecatalog.FieldRequirements(entry.RequiredFields),
				IsAWSManaged:   entry.IsAWSManaged,
				SSLModes:       sourceSSLModes(entry.SSLModes),
			})
			if !ok {
				t.Fatalf("expected %q to map into the source catalog", entry.ID)
			}
			if spec.ID != string(entry.ID) {
				t.Fatalf("expected source id %q, got %q", entry.ID, spec.ID)
			}
		})
	}
}

func TestBuildTypeSpecContractsAreSelfConsistent(t *testing.T) {
	t.Parallel()

	for _, entry := range dbcatalog.All() {
		t.Run(string(entry.ID), func(t *testing.T) {
			t.Parallel()

			spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
				ID:             string(entry.ID),
				Label:          entry.Label,
				Connector:      string(entry.PluginType),
				Extra:          maps.Clone(entry.Extra),
				Fields:         coresourcecatalog.FieldVisibility(entry.Fields),
				RequiredFields: coresourcecatalog.FieldRequirements(entry.RequiredFields),
				IsAWSManaged:   entry.IsAWSManaged,
				SSLModes:       sourceSSLModes(entry.SSLModes),
			})
			if !ok {
				t.Fatalf("expected %q to map into the source catalog", entry.ID)
			}
			if err := source.ValidateContract(spec); err != nil {
				t.Fatalf("expected source contract to be self-consistent: %v", err)
			}
		})
	}
}

func TestBuildTypeSpecConnectionContractsAreSelfConsistent(t *testing.T) {
	t.Parallel()

	for _, entry := range dbcatalog.All() {
		t.Run(string(entry.ID), func(t *testing.T) {
			t.Parallel()

			spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
				ID:             string(entry.ID),
				Label:          entry.Label,
				Connector:      string(entry.PluginType),
				Extra:          maps.Clone(entry.Extra),
				Fields:         coresourcecatalog.FieldVisibility(entry.Fields),
				RequiredFields: coresourcecatalog.FieldRequirements(entry.RequiredFields),
				IsAWSManaged:   entry.IsAWSManaged,
				SSLModes:       sourceSSLModes(entry.SSLModes),
			})
			if !ok {
				t.Fatalf("expected %q to map into the source catalog", entry.ID)
			}
			if err := source.ValidateConnectionContract(spec); err != nil {
				t.Fatalf("expected source connection contract to be self-consistent: %v", err)
			}
		})
	}
}

func TestBuildTypeSpecMetadataContractsAreSelfConsistent(t *testing.T) {
	t.Parallel()

	for _, entry := range dbcatalog.All() {
		t.Run(string(entry.ID), func(t *testing.T) {
			t.Parallel()

			spec := buildTestTypeSpec(t, entry)
			if err := source.ValidateObjectMetadataContract(spec); err != nil {
				t.Fatalf("expected source object metadata contract to be self-consistent: %v; traits=%v contract=%v", err, spec.Traits.Metadata, spec.Contract)
			}

			sessionMetadata, sessionMetadataOK := coresourcecatalog.ResolveSessionMetadata(spec.ID, spec.Connector)
			if err := source.ValidateSessionMetadataContract(spec, sessionMetadata, sessionMetadataOK); err != nil {
				t.Fatalf("expected source session metadata contract to be self-consistent: %v; traits=%v metadata=%#v", err, spec.Traits.Query, sessionMetadata)
			}

			creationMetadata, creationMetadataOK := coresourcecatalog.ResolveObjectCreationMetadata(spec.ID, spec.Connector)
			if err := source.ValidateObjectCreationMetadataContract(spec, creationMetadata, creationMetadataOK); err != nil {
				t.Fatalf("expected source object creation metadata contract to be self-consistent: %v; contract=%v metadata=%#v", err, spec.Contract, creationMetadata)
			}
		})
	}
}

func TestBuildTypeSpecExposesMutableDataActions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id      string
		kind    source.ObjectKind
		actions []source.Action
	}{
		{
			id:   "Postgres",
			kind: source.ObjectKindTable,
			actions: []source.Action{
				source.ActionUpdateData,
				source.ActionDeleteData,
			},
		},
		{
			id:   "MongoDB",
			kind: source.ObjectKindCollection,
			actions: []source.Action{
				source.ActionUpdateData,
				source.ActionDeleteData,
			},
		},
		{
			id:   "Redis",
			kind: source.ObjectKindKey,
			actions: []source.Action{
				source.ActionUpdateData,
				source.ActionDeleteData,
			},
		},
		{
			id:   "Memcached",
			kind: source.ObjectKindItem,
			actions: []source.Action{
				source.ActionUpdateData,
				source.ActionDeleteData,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()

			entry, ok := dbcatalog.Find(tt.id)
			if !ok {
				t.Fatalf("expected database catalog entry for %q", tt.id)
			}

			spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
				ID:             string(entry.ID),
				Label:          entry.Label,
				Connector:      string(entry.PluginType),
				Extra:          maps.Clone(entry.Extra),
				Fields:         coresourcecatalog.FieldVisibility(entry.Fields),
				RequiredFields: coresourcecatalog.FieldRequirements(entry.RequiredFields),
				IsAWSManaged:   entry.IsAWSManaged,
				SSLModes:       sourceSSLModes(entry.SSLModes),
			})
			if !ok {
				t.Fatalf("expected %q to map into the source catalog", tt.id)
			}

			objectType, ok := spec.Contract.ObjectTypeForKind(tt.kind)
			if !ok {
				t.Fatalf("expected object kind %q for %q", tt.kind, tt.id)
			}

			for _, action := range tt.actions {
				if !slices.Contains(objectType.Actions, action) {
					t.Fatalf("expected %q to expose action %q, got %v", tt.id, action, objectType.Actions)
				}
			}
		})
	}
}

func TestBuildTypeSpecKeepsQuestDBAppendOnlyAndSchemaLess(t *testing.T) {
	t.Parallel()

	entry, ok := dbcatalog.Find("QuestDB")
	if !ok {
		t.Fatal("expected database catalog entry for QuestDB")
	}

	spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
		ID:             string(entry.ID),
		Label:          entry.Label,
		Connector:      string(entry.PluginType),
		Extra:          maps.Clone(entry.Extra),
		Fields:         coresourcecatalog.FieldVisibility(entry.Fields),
		RequiredFields: coresourcecatalog.FieldRequirements(entry.RequiredFields),
		IsAWSManaged:   entry.IsAWSManaged,
		SSLModes:       sourceSSLModes(entry.SSLModes),
	})
	if !ok {
		t.Fatal("expected QuestDB to map into the source catalog")
	}

	if spec.Contract.SupportsSurface(source.SurfaceGraph) {
		t.Fatalf("expected QuestDB graph surface to be disabled, got %v", spec.Contract.Surfaces)
	}
	if slices.Contains(spec.Contract.BrowsePath, source.ObjectKindSchema) {
		t.Fatalf("expected QuestDB browse path to remain schema-less, got %v", spec.Contract.BrowsePath)
	}

	objectType, ok := spec.Contract.ObjectTypeForKind(source.ObjectKindTable)
	if !ok {
		t.Fatal("expected QuestDB table object type")
	}
	for _, action := range []source.Action{source.ActionUpdateData, source.ActionDeleteData, source.ActionGenerateMockData, source.ActionImportData} {
		if slices.Contains(objectType.Actions, action) {
			t.Fatalf("expected QuestDB tables to omit action %q, got %v", action, objectType.Actions)
		}
	}
	if !slices.Contains(objectType.Actions, source.ActionInsertData) {
		t.Fatalf("expected QuestDB tables to keep insert support, got %v", objectType.Actions)
	}
}

func TestQuestDBSessionMetadataUsesQuestDBTypes(t *testing.T) {
	t.Parallel()

	metadata, ok := coresourcecatalog.ResolveSessionMetadata("QuestDB")
	if !ok {
		t.Fatal("expected QuestDB session metadata")
	}

	typeIDs := make(map[string]bool, len(metadata.TypeDefinitions))
	for _, typeDefinition := range metadata.TypeDefinitions {
		typeIDs[typeDefinition.ID] = true
	}

	for _, expected := range []string{"INT", "VARCHAR", "STRING", "TIMESTAMP"} {
		if !typeIDs[expected] {
			t.Fatalf("expected QuestDB type %q, got %#v", expected, typeIDs)
		}
	}
	for _, unsupported := range []string{"CHARACTER", "CHARACTER VARYING"} {
		if typeIDs[unsupported] {
			t.Fatalf("expected QuestDB metadata not to expose %q, got %#v", unsupported, typeIDs)
		}
	}
}

func TestObjectCreationMetadataUsesSourceNativeColumnLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id                     string
		wantIdentity           string
		wantPrimary            string
		wantIdentityCapability bool
	}{
		{id: "MySQL", wantIdentity: "AUTO_INCREMENT", wantPrimary: "PRIMARY KEY", wantIdentityCapability: true},
		{id: "Postgres", wantIdentity: "GENERATED ALWAYS AS IDENTITY", wantPrimary: "PRIMARY KEY", wantIdentityCapability: true},
		{id: "DuckDB", wantIdentity: "DEFAULT nextval()", wantPrimary: "PRIMARY KEY", wantIdentityCapability: true},
		{id: "Sqlite3", wantIdentity: "Identity", wantPrimary: "PRIMARY KEY", wantIdentityCapability: false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()

			metadata, ok := coresourcecatalog.ResolveObjectCreationMetadata(tt.id)
			if !ok {
				t.Fatalf("expected object creation metadata for %q", tt.id)
			}
			if metadata.ColumnLabels.Identity != tt.wantIdentity {
				t.Fatalf("expected identity label %q, got %q", tt.wantIdentity, metadata.ColumnLabels.Identity)
			}
			if metadata.ColumnLabels.PrimaryKey != tt.wantPrimary {
				t.Fatalf("expected primary label %q, got %q", tt.wantPrimary, metadata.ColumnLabels.PrimaryKey)
			}
			if metadata.ColumnCapabilities.Identity != tt.wantIdentityCapability {
				t.Fatalf("expected identity capability %t, got %t", tt.wantIdentityCapability, metadata.ColumnCapabilities.Identity)
			}
		})
	}
}

func TestBuildTypeSpecExposesSourceTraits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   string
		want func(t *testing.T, spec source.TypeSpec)
	}{
		{
			id: "Sqlite3",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Connection.Transport != source.ConnectionTransportFile {
					t.Fatalf("expected Sqlite3 transport %q, got %q", source.ConnectionTransportFile, spec.Traits.Connection.Transport)
				}
				if spec.Traits.Connection.HostInputMode != source.HostInputModeNone {
					t.Fatalf("expected Sqlite3 host input mode %q, got %q", source.HostInputModeNone, spec.Traits.Connection.HostInputMode)
				}
				if !spec.Traits.Connection.SupportsCustomCAContent {
					t.Fatalf("expected Sqlite3 custom CA support to remain enabled")
				}
				if spec.Traits.Presentation.ProfileLabelStrategy != source.ProfileLabelStrategyDatabase {
					t.Fatalf("expected Sqlite3 profile label strategy %q, got %q", source.ProfileLabelStrategyDatabase, spec.Traits.Presentation.ProfileLabelStrategy)
				}
				databaseField, ok := spec.ConnectionFieldByKey("Database")
				if !ok {
					t.Fatalf("expected Sqlite3 database field")
				}
				if databaseField.Kind != source.ConnectionFieldKindFilePath {
					t.Fatalf("expected Sqlite3 database field kind %q, got %q", source.ConnectionFieldKindFilePath, databaseField.Kind)
				}
				if !databaseField.SupportsOptions {
					t.Fatalf("expected Sqlite3 database field options support")
				}
			},
		},
		{
			id: "Postgres",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Connection.HostInputMode != source.HostInputModeHostnameOrURL {
					t.Fatalf("expected Postgres host input mode %q, got %q", source.HostInputModeHostnameOrURL, spec.Traits.Connection.HostInputMode)
				}
				if spec.Traits.Connection.HostInputURLParser != source.HostInputURLParserPostgres {
					t.Fatalf("expected Postgres URL parser %q, got %q", source.HostInputURLParserPostgres, spec.Traits.Connection.HostInputURLParser)
				}
				if !spec.Traits.Connection.SupportsCustomCAContent {
					t.Fatalf("expected Postgres custom CA support to remain enabled")
				}
				if !spec.Traits.Query.SupportsAnalyze {
					t.Fatalf("expected Postgres analyze support")
				}
				if spec.Traits.Query.ExplainMode != source.QueryExplainModeExplainAnalyze {
					t.Fatalf("expected Postgres explain mode %q, got %q", source.QueryExplainModeExplainAnalyze, spec.Traits.Query.ExplainMode)
				}
				if !spec.Traits.Query.SupportsScripts || !spec.Traits.Query.SupportsStreaming || !spec.Traits.Query.SupportsMultiStatement {
					t.Fatalf("expected Postgres execution traits to support scripts, streaming, and multi-statement, got %#v", spec.Traits.Query)
				}
				if !spec.Contract.SupportsRootAction(source.ActionExecute) {
					t.Fatalf("expected Postgres root execute support")
				}
				if !spec.Traits.MockData.SupportsRelationalDependencies {
					t.Fatalf("expected Postgres mock-data relational dependency support")
				}
			},
		},
		{
			id: "YugabyteDB",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Connection.HostInputMode != source.HostInputModeHostnameOrURL {
					t.Fatalf("expected YugabyteDB host input mode %q, got %q", source.HostInputModeHostnameOrURL, spec.Traits.Connection.HostInputMode)
				}
				if spec.Traits.Query.SupportsAnalyze {
					t.Fatalf("expected YugabyteDB analyze support to remain disabled")
				}
			},
		},
		{
			id: "MongoDB",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Connection.HostInputURLParser != source.HostInputURLParserMongoSRV {
					t.Fatalf("expected MongoDB URL parser %q, got %q", source.HostInputURLParserMongoSRV, spec.Traits.Connection.HostInputURLParser)
				}
				if spec.Traits.Presentation.SchemaFidelity != source.SchemaFidelitySampled {
					t.Fatalf("expected MongoDB schema fidelity %q, got %q", source.SchemaFidelitySampled, spec.Traits.Presentation.SchemaFidelity)
				}
				if spec.Traits.Metadata.Columns != source.MetadataFidelitySampled {
					t.Fatalf("expected MongoDB column metadata fidelity %q, got %q", source.MetadataFidelitySampled, spec.Traits.Metadata.Columns)
				}
			},
		},
		{
			id: "Valkey",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Presentation.ProfileLabelStrategy != source.ProfileLabelStrategyHostname {
					t.Fatalf("expected Valkey profile label strategy %q, got %q", source.ProfileLabelStrategyHostname, spec.Traits.Presentation.ProfileLabelStrategy)
				}
			},
		},
		{
			id: "OpenSearch",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Presentation.SchemaFidelity != source.SchemaFidelitySampled {
					t.Fatalf("expected OpenSearch schema fidelity %q, got %q", source.SchemaFidelitySampled, spec.Traits.Presentation.SchemaFidelity)
				}
			},
		},
		{
			id: "ClickHouse",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Query.ExplainMode != source.QueryExplainModeExplainPipeline {
					t.Fatalf("expected ClickHouse explain mode %q, got %q", source.QueryExplainModeExplainPipeline, spec.Traits.Query.ExplainMode)
				}
				if !spec.Traits.Query.SupportsScripts || !spec.Traits.Query.SupportsStreaming {
					t.Fatalf("expected ClickHouse execution traits to support scripts and streaming, got %#v", spec.Traits.Query)
				}
				if spec.Traits.Query.SupportsMultiStatement {
					t.Fatalf("expected ClickHouse multi-statement support to remain disabled")
				}
				if spec.Traits.Metadata.Graph != source.MetadataFidelityInferred {
					t.Fatalf("expected ClickHouse graph metadata fidelity %q, got %q", source.MetadataFidelityInferred, spec.Traits.Metadata.Graph)
				}
				if spec.Traits.MockData.SupportsRelationalDependencies {
					t.Fatalf("expected ClickHouse mock-data relational dependency support to remain disabled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()

			entry, ok := dbcatalog.Find(tt.id)
			if !ok {
				t.Fatalf("expected database catalog entry for %q", tt.id)
			}

			spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
				ID:             string(entry.ID),
				Label:          entry.Label,
				Connector:      string(entry.PluginType),
				Extra:          maps.Clone(entry.Extra),
				Fields:         coresourcecatalog.FieldVisibility(entry.Fields),
				RequiredFields: coresourcecatalog.FieldRequirements(entry.RequiredFields),
				IsAWSManaged:   entry.IsAWSManaged,
				SSLModes:       sourceSSLModes(entry.SSLModes),
			})
			if !ok {
				t.Fatalf("expected %q to map into the source catalog", tt.id)
			}

			tt.want(t, spec)
		})
	}
}

func TestBuildTypeSpecUsesTypedExtraFields(t *testing.T) {
	t.Parallel()

	spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
		ID:        "CustomBridge",
		Label:     "Custom Bridge",
		Connector: "Postgres",
		Extra: map[string]source.ConnectionExtraField{
			"Token": {
				DefaultValue: "secret",
				Kind:         source.ConnectionFieldKindPassword,
				Required:     true,
				LabelKey:     "advancedFields.customToken",
			},
			"SSL": {
				DefaultValue: "false",
				Kind:         source.ConnectionFieldKindBoolean,
				LabelKey:     "advancedFields.customSsl",
			},
		},
		Fields:         coresourcecatalog.FieldVisibility{Hostname: true},
		RequiredFields: coresourcecatalog.FieldRequirements{Hostname: true},
	})
	if !ok {
		t.Fatal("expected custom bridge entry to map into the source catalog")
	}

	tokenField, ok := spec.ConnectionFieldByKey("Token")
	if !ok {
		t.Fatal("expected custom token field")
	}
	if tokenField.Kind != source.ConnectionFieldKindPassword {
		t.Fatalf("expected token field kind %q, got %q", source.ConnectionFieldKindPassword, tokenField.Kind)
	}
	if !tokenField.Required {
		t.Fatal("expected token field to remain required in the built source type")
	}
	if tokenField.LabelKey != "advancedFields.customToken" {
		t.Fatalf("expected token field label key %q, got %q", "advancedFields.customToken", tokenField.LabelKey)
	}

	sslField, ok := spec.ConnectionFieldByKey("SSL")
	if !ok {
		t.Fatal("expected custom ssl field")
	}
	if sslField.Kind != source.ConnectionFieldKindBoolean {
		t.Fatalf("expected ssl field kind %q, got %q", source.ConnectionFieldKindBoolean, sslField.Kind)
	}
	if sslField.LabelKey != "advancedFields.customSsl" {
		t.Fatalf("expected ssl field label key %q, got %q", "advancedFields.customSsl", sslField.LabelKey)
	}
}

func TestBuildTypeSpecDeclaresInternalObjectRules(t *testing.T) {
	t.Parallel()

	entry, ok := dbcatalog.Find("MySQL")
	if !ok {
		t.Fatal("expected MySQL database catalog entry")
	}
	spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
		ID:             string(entry.ID),
		Label:          entry.Label,
		Connector:      string(entry.PluginType),
		Extra:          maps.Clone(entry.Extra),
		Fields:         coresourcecatalog.FieldVisibility(entry.Fields),
		RequiredFields: coresourcecatalog.FieldRequirements(entry.RequiredFields),
		IsAWSManaged:   entry.IsAWSManaged,
		SSLModes:       sourceSSLModes(entry.SSLModes),
	})
	if !ok {
		t.Fatal("expected MySQL to map into the source catalog")
	}
	if !source.ShouldHideObject(spec, source.ObjectKindDatabase, "information_schema") {
		t.Fatal("expected MySQL information_schema to be declared internal")
	}
	if spec.Traits.Metadata.SystemObjectFiltering != source.MetadataFidelityExact {
		t.Fatalf("expected internal object filtering fidelity %q, got %q", source.MetadataFidelityExact, spec.Traits.Metadata.SystemObjectFiltering)
	}
}

func sourceSSLModes(modes []source.SSLModeInfo) []source.SSLModeInfo {
	cloned := make([]source.SSLModeInfo, 0, len(modes))
	for _, mode := range modes {
		cloned = append(cloned, source.SSLModeInfo{
			Value:       mode.Value,
			Label:       mode.Label,
			Description: mode.Description,
			Aliases:     append([]string(nil), mode.Aliases...),
		})
	}
	return cloned
}

func buildTestTypeSpec(t *testing.T, entry dbcatalog.ConnectableDatabase) source.TypeSpec {
	t.Helper()

	spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
		ID:             string(entry.ID),
		Label:          entry.Label,
		Connector:      string(entry.PluginType),
		Extra:          maps.Clone(entry.Extra),
		Fields:         coresourcecatalog.FieldVisibility(entry.Fields),
		RequiredFields: coresourcecatalog.FieldRequirements(entry.RequiredFields),
		IsAWSManaged:   entry.IsAWSManaged,
		SSLModes:       sourceSSLModes(entry.SSLModes),
	})
	if !ok {
		t.Fatalf("expected %q to map into the source catalog", entry.ID)
	}
	return spec
}
