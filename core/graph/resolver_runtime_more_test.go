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

package graph

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/settings"
)

func TestMutationUpdateSettingsAndQuerySettingsConfig(t *testing.T) {
	originalSettings := settings.Get()
	originalAWS := env.IsAWSProviderEnabled
	originalAzure := env.IsAzureProviderEnabled
	originalGCP := env.IsGCPProviderEnabled
	originalDisableCredentialForm := env.DisableCredentialForm
	originalNewUI := env.IsNewUIEnabled
	originalMaxPageSize := env.MaxPageSize
	t.Cleanup(func() {
		settings.UpdateSettings(settings.MetricsEnabledField(originalSettings.MetricsEnabled))
		env.IsAWSProviderEnabled = originalAWS
		env.IsAzureProviderEnabled = originalAzure
		env.IsGCPProviderEnabled = originalGCP
		env.DisableCredentialForm = originalDisableCredentialForm
		env.IsNewUIEnabled = originalNewUI
		env.MaxPageSize = originalMaxPageSize
	})

	status, err := (&Resolver{}).Mutation().UpdateSettings(context.Background(), model.SettingsConfigInput{
		MetricsEnabled: stringPtr("true"),
	})
	if err != nil {
		t.Fatalf("expected settings update to succeed, got %v", err)
	}
	if status == nil || !status.Status {
		t.Fatalf("expected settings update to report success, got %#v", status)
	}

	env.IsAWSProviderEnabled = false
	env.IsAzureProviderEnabled = true
	env.IsGCPProviderEnabled = false
	env.DisableCredentialForm = true
	env.IsNewUIEnabled = true
	env.MaxPageSize = 321

	cfg, err := (&Resolver{}).Query().SettingsConfig(context.Background())
	if err != nil {
		t.Fatalf("expected settings config query to succeed, got %v", err)
	}
	if cfg.MetricsEnabled == nil || !*cfg.MetricsEnabled {
		t.Fatalf("expected metrics setting to be enabled, got %#v", cfg)
	}
	if !cfg.CloudProvidersEnabled || !cfg.DisableCredentialForm || !cfg.EnableNewUI || cfg.MaxPageSize != 321 {
		t.Fatalf("expected settings config to reflect env flags, got %#v", cfg)
	}
	if cfg.AWSProviderEnabled || !cfg.AzureProviderEnabled || cfg.GCPProviderEnabled {
		t.Fatalf("expected provider-specific settings to reflect env flags, got %#v", cfg)
	}
}

func TestQueryUpdateInfoReturnsDisabledState(t *testing.T) {
	originalVersion := env.ApplicationVersion
	t.Cleanup(func() {
		env.ApplicationVersion = originalVersion
	})

	t.Setenv("WHODB_DISABLE_UPDATE_CHECK", "true")
	env.ApplicationVersion = "2.3.4"

	info, err := (&Resolver{}).Query().UpdateInfo(context.Background())
	if err != nil {
		t.Fatalf("expected update info query to succeed, got %v", err)
	}
	if info.CurrentVersion != "2.3.4" || info.LatestVersion != "2.3.4" || info.UpdateAvailable {
		t.Fatalf("expected disabled update check to echo current version, got %#v", info)
	}
}

func TestMutationAddStorageUnitMapsFieldExtras(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.AddStorageUnitFunc = func(_ *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
		if schema != "public" || storageUnit != "users" {
			t.Fatalf("unexpected add storage unit target: %s.%s", schema, storageUnit)
		}
		if len(fields) != 1 || fields[0].Key != "email" || fields[0].Value != "TEXT" {
			t.Fatalf("unexpected fields payload: %#v", fields)
		}
		if fields[0].Extra["nullable"] != "false" || fields[0].Extra["default"] != "''" {
			t.Fatalf("expected extra metadata to be mapped, got %#v", fields[0].Extra)
		}
		return true, nil
	}
	setEngineMock(t, mock)

	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})
	resp, err := (&Resolver{}).Mutation().CreateSourceObject(ctx, testSourceRefPtr(model.SourceObjectKindSchema, "app", "public"), "users", []*model.RecordInput{{
		Key:   "email",
		Value: "TEXT",
		Extra: []*model.RecordInput{
			{Key: "nullable", Value: "false"},
			{Key: "default", Value: "''"},
		},
	}})
	if err != nil {
		t.Fatalf("expected add storage unit to succeed, got %v", err)
	}
	if resp == nil || !resp.Status {
		t.Fatalf("expected add storage unit success, got %#v", resp)
	}
}

func TestQueryStorageUnitMapsMockDataAllowance(t *testing.T) {
	originalDisabled := env.DisableMockDataGeneration
	t.Cleanup(func() {
		env.DisableMockDataGeneration = originalDisabled
	})
	env.DisableMockDataGeneration = "logs"

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetStorageUnitsFunc = func(*engine.PluginConfig, string) ([]engine.StorageUnit, error) {
		return []engine.StorageUnit{
			{Name: "logs"},
			{Name: "orders"},
		}, nil
	}
	setEngineMock(t, mock)

	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})
	units, err := (&Resolver{}).Query().SourceObjects(ctx, testSourceRefPtr(model.SourceObjectKindSchema, "app", "public"), nil)
	if err != nil {
		t.Fatalf("expected storage unit query to succeed, got %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected two storage units, got %#v", units)
	}

	for _, unit := range units {
		for _, record := range unit.Metadata {
			if record.Key == "IsMockDataGenerationAllowed" {
				t.Fatalf("expected internal mock-data policy metadata to stay out of source object metadata, got %#v", units)
			}
		}
	}

	if units[0].Name != "logs" || units[1].Name != "orders" {
		t.Fatalf("unexpected storage units: %#v", units)
	}
}

func TestQueryDatabaseUsesSessionCredentialsWhenTypeMatches(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetDatabasesFunc = func(config *engine.PluginConfig) ([]string, error) {
		if config.Credentials.Hostname != "db.internal" || config.Credentials.Database != "analytics" {
			t.Fatalf("expected session credentials to be used, got %#v", config.Credentials)
		}
		return []string{"analytics"}, nil
	}
	setEngineMock(t, mock)

	ctx := testSourceContext("Postgres", map[string]string{
		"Hostname": "db.internal",
		"Database": "analytics",
	})
	databases, err := (&Resolver{}).Query().SourceFieldOptions(ctx, "Postgres", "Database", nil)
	if err != nil {
		t.Fatalf("expected database query to succeed, got %v", err)
	}
	if len(databases) != 1 || databases[0] != "analytics" {
		t.Fatalf("unexpected database list: %#v", databases)
	}
}

func TestQueryDatabaseRejectsUnsupportedTypes(t *testing.T) {
	if _, err := (&Resolver{}).Query().SourceFieldOptions(context.Background(), "Unsupported", "Database", nil); err == nil || !strings.Contains(err.Error(), "unsupported source type") {
		t.Fatalf("expected unsupported database error, got %v", err)
	}
}

func TestQueryRawExecuteAndGraphMapPluginResults(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "id", Type: "INTEGER", IsPrimary: true}},
			Rows:    [][]string{{"1"}},
		}, nil
	}
	sourceColumn := "user_id"
	targetColumn := "id"
	mock.GetGraphFunc = func(*engine.PluginConfig, string) ([]engine.GraphUnit, error) {
		return []engine.GraphUnit{{
			Unit: engine.StorageUnit{Name: "orders"},
			Relations: []engine.GraphUnitRelationship{{
				Name:             "orders_user_id_fkey",
				RelationshipType: "ManyToOne",
				SourceColumn:     &sourceColumn,
				TargetColumn:     &targetColumn,
			}},
		}}, nil
	}
	setEngineMock(t, mock)

	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})

	rows, err := (&Resolver{}).Query().RunSourceQuery(ctx, "SELECT 1")
	if err != nil {
		t.Fatalf("expected raw execute to succeed, got %v", err)
	}
	if len(rows.Columns) != 1 || !rows.Columns[0].IsPrimary || len(rows.Rows) != 1 {
		t.Fatalf("expected raw execute result to map column metadata, got %#v", rows)
	}

	graphUnits, err := (&Resolver{}).Query().SourceGraph(ctx, testSourceRefPtr(model.SourceObjectKindSchema, "app", "public"))
	if err != nil {
		t.Fatalf("expected graph query to succeed, got %v", err)
	}
	if len(graphUnits) != 1 || len(graphUnits[0].Relations) != 1 {
		t.Fatalf("expected graph units to be mapped, got %#v", graphUnits)
	}
	if graphUnits[0].Relations[0].Relationship != "ManyToOne" || graphUnits[0].Relations[0].SourceColumn == nil {
		t.Fatalf("expected graph relationship metadata to be preserved, got %#v", graphUnits[0].Relations[0])
	}
}

func TestQuerySSLStatusHandlesMappedNilAndErrorResults(t *testing.T) {
	t.Run("nil status returns nil", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.GetSSLStatusFunc = func(*engine.PluginConfig) (*engine.SSLStatus, error) { return nil, nil }
		setEngineMock(t, mock)

		ctx := testSourceContext("Postgres", nil)
		status, err := (&Resolver{}).Query().SSLStatus(ctx)
		if err != nil {
			t.Fatalf("expected nil SSL status to succeed, got %v", err)
		}
		if status != nil {
			t.Fatalf("expected nil SSL status, got %#v", status)
		}
	})

	t.Run("non-nil status is mapped", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.GetSSLStatusFunc = func(*engine.PluginConfig) (*engine.SSLStatus, error) {
			return &engine.SSLStatus{IsEnabled: true, Mode: "verify-full"}, nil
		}
		setEngineMock(t, mock)

		ctx := testSourceContext("Postgres", nil)
		status, err := (&Resolver{}).Query().SSLStatus(ctx)
		if err != nil {
			t.Fatalf("expected SSL status query to succeed, got %v", err)
		}
		if status == nil || !status.IsEnabled || status.Mode != "verify-full" {
			t.Fatalf("expected SSL status to be mapped, got %#v", status)
		}
	})
}

func TestQueryAIProvidersIncludesConfiguredAndBuiltinProviders(t *testing.T) {
	originalOpenAIAPIKey := env.OpenAIAPIKey
	originalOpenAIName := env.OpenAIName
	originalLMStudioName := env.LMStudioName
	originalLMStudioAPIKey := env.LMStudioAPIKey
	originalOllamaName := env.OllamaName
	originalGenericProviders := append([]env.GenericProviderConfig(nil), env.GenericProviders...)
	t.Cleanup(func() {
		env.OpenAIAPIKey = originalOpenAIAPIKey
		env.OpenAIName = originalOpenAIName
		env.LMStudioName = originalLMStudioName
		env.LMStudioAPIKey = originalLMStudioAPIKey
		env.OllamaName = originalOllamaName
		env.GenericProviders = originalGenericProviders
	})

	env.OpenAIAPIKey = "sk-test"
	env.OpenAIName = "OpenAI Prod"
	env.LMStudioName = "Studio Local"
	env.LMStudioAPIKey = "lm-key"
	env.OllamaName = "Ollama Local"
	env.GenericProviders = []env.GenericProviderConfig{{
		ProviderId: "custom-openai",
		Name:       "Custom OpenAI",
		ClientType: "openai-generic",
		BaseURL:    "http://localhost:9999/v1",
		APIKey:     "custom-key",
	}}

	providers, err := (&Resolver{}).Query().AIProviders(context.Background())
	if err != nil {
		t.Fatalf("expected AI providers query to succeed, got %v", err)
	}

	byID := map[string]*model.AIProvider{}
	for _, provider := range providers {
		byID[provider.ProviderID] = provider
	}
	if byID["openai-1"] == nil || byID["openai-1"].Name != "OpenAI Prod" {
		t.Fatalf("expected OpenAI provider to be returned, got %#v", providers)
	}
	if byID["custom-openai"] == nil || !byID["custom-openai"].IsGeneric {
		t.Fatalf("expected generic AI provider to be returned, got %#v", providers)
	}
	if byID["lmstudio-1"] == nil || byID["lmstudio-1"].Name != "Studio Local" {
		t.Fatalf("expected LM Studio provider to be returned, got %#v", providers)
	}
	if byID["ollama-1"] == nil || byID["ollama-1"].Name != "Ollama Local" {
		t.Fatalf("expected Ollama provider to be returned, got %#v", providers)
	}
}

func TestCloudQueriesReturnEmptyWhenProvidersDisabled(t *testing.T) {
	originalAWS := env.IsAWSProviderEnabled
	originalAzure := env.IsAzureProviderEnabled
	originalGCP := env.IsGCPProviderEnabled
	t.Cleanup(func() {
		env.IsAWSProviderEnabled = originalAWS
		env.IsAzureProviderEnabled = originalAzure
		env.IsGCPProviderEnabled = originalGCP
	})

	env.IsAWSProviderEnabled = false
	env.IsAzureProviderEnabled = false
	env.IsGCPProviderEnabled = false

	query := (&Resolver{}).Query()

	discovered, err := query.DiscoveredConnections(context.Background())
	if err != nil {
		t.Fatalf("expected discovered connections query to succeed, got %v", err)
	}
	if len(discovered) != 0 {
		t.Fatalf("expected no discovered connections when providers are disabled, got %#v", discovered)
	}

	connections, err := query.ProviderConnections(context.Background(), "provider-1")
	if err != nil {
		t.Fatalf("expected provider connections query to succeed, got %v", err)
	}
	if len(connections) != 0 {
		t.Fatalf("expected no provider connections when providers are disabled, got %#v", connections)
	}

	awsProfiles, err := query.LocalAWSProfiles(context.Background())
	if err != nil {
		t.Fatalf("expected local AWS profiles query to succeed, got %v", err)
	}
	if len(awsProfiles) != 0 {
		t.Fatalf("expected no AWS profiles when providers are disabled, got %#v", awsProfiles)
	}

	azureProviders, err := query.AzureProviders(context.Background())
	if err != nil {
		t.Fatalf("expected Azure providers query to succeed, got %v", err)
	}
	if len(azureProviders) != 0 {
		t.Fatalf("expected no Azure providers when disabled, got %#v", azureProviders)
	}

	azureProvider, err := query.AzureProvider(context.Background(), "azure-1")
	if err != nil {
		t.Fatalf("expected Azure provider query to succeed, got %v", err)
	}
	if azureProvider != nil {
		t.Fatalf("expected nil Azure provider when disabled, got %#v", azureProvider)
	}

	gcpProviders, err := query.GCPProviders(context.Background())
	if err != nil {
		t.Fatalf("expected GCP providers query to succeed, got %v", err)
	}
	if len(gcpProviders) != 0 {
		t.Fatalf("expected no GCP providers when disabled, got %#v", gcpProviders)
	}

	gcpProvider, err := query.GCPProvider(context.Background(), "gcp-1")
	if err != nil {
		t.Fatalf("expected GCP provider query to succeed, got %v", err)
	}
	if gcpProvider != nil {
		t.Fatalf("expected nil GCP provider when disabled, got %#v", gcpProvider)
	}

	localProjects, err := query.LocalGCPProjects(context.Background())
	if err != nil {
		t.Fatalf("expected local GCP projects query to succeed, got %v", err)
	}
	if len(localProjects) != 0 {
		t.Fatalf("expected no GCP projects when disabled, got %#v", localProjects)
	}
}

func TestQueryHealthRecoversFromPanicsAndCanceledContexts(t *testing.T) {
	t.Run("plugin panic becomes database error", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.IsAvailableFunc = func(context.Context, *engine.PluginConfig) bool {
			panic("boom")
		}
		setEngineMock(t, mock)

		ctx := testSourceContext("Postgres", nil)
		status, err := (&Resolver{}).Query().Health(ctx)
		if err != nil {
			t.Fatalf("expected health query to recover panic, got %v", err)
		}
		if status.Database != "error" {
			t.Fatalf("expected panic to surface as database error, got %#v", status)
		}
	})

	t.Run("canceled context reports database error", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.IsAvailableFunc = func(ctx context.Context, _ *engine.PluginConfig) bool {
			<-ctx.Done()
			time.Sleep(10 * time.Millisecond)
			return false
		}
		setEngineMock(t, mock)

		baseCtx, cancel := context.WithCancel(testSourceContext("Postgres", nil))
		cancel()
		ctx := baseCtx
		status, err := (&Resolver{}).Query().Health(ctx)
		if err != nil {
			t.Fatalf("expected health query to handle canceled context, got %v", err)
		}
		if status.Database != "error" {
			t.Fatalf("expected canceled context to report database error, got %#v", status)
		}
	})
}
