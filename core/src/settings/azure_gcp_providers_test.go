package settings

import (
	"errors"
	"testing"

	azureinfra "github.com/clidey/whodb/core/src/azure"
	"github.com/clidey/whodb/core/src/env"
	gcpinfra "github.com/clidey/whodb/core/src/gcp"
	"github.com/clidey/whodb/core/src/providers"
)

func resetAzureProviders() {
	azureProvidersMu.Lock()
	defer azureProvidersMu.Unlock()

	for id := range azureProviders {
		delete(azureProviders, id)
	}

	providers.GetDefaultRegistry().Close(nil)
}

func resetGCPProviders() {
	gcpProvidersMu.Lock()
	defer gcpProvidersMu.Unlock()

	for id := range gcpProviders {
		delete(gcpProviders, id)
	}

	providers.GetDefaultRegistry().Close(nil)
}

func resetAllCloudProviders() {
	resetProviders()
	resetAzureProviders()
	resetGCPProviders()
}

func TestAzureProviderCRUDAndHelpers(t *testing.T) {
	resetAllCloudProviders()
	skipAzurePersist = true
	defer func() { skipAzurePersist = false }()

	cfg := &AzureProviderConfig{
		ID:                 "azure-test",
		Name:               "Azure Test",
		SubscriptionID:     "sub-123",
		TenantID:           "tenant-123",
		ClientID:           "client-123",
		AuthMethod:         "default",
		ResourceGroup:      "rg-app",
		DiscoverPostgreSQL: true,
		DiscoverMySQL:      true,
	}

	state, err := AddAzureProvider(cfg, "secret-123")
	if err != nil {
		t.Fatalf("expected Azure provider add to succeed, got %v", err)
	}
	if state.Config.Name != "Azure Test" || state.Status != "Connected" {
		t.Fatalf("unexpected Azure provider state after add: %#v", state)
	}

	got, err := GetAzureProvider("azure-test")
	if err != nil {
		t.Fatalf("expected Azure provider lookup to succeed, got %v", err)
	}
	if got.Config.ResourceGroup != "rg-app" {
		t.Fatalf("expected Azure provider resource group to be preserved, got %#v", got.Config)
	}
	if len(GetAzureProviders()) != 1 {
		t.Fatalf("expected one Azure provider to be listed, got %d", len(GetAzureProviders()))
	}

	updated, err := UpdateAzureProvider("azure-test", &AzureProviderConfig{
		Name:               "Azure Updated",
		SubscriptionID:     "sub-123",
		AuthMethod:         "default",
		DiscoverPostgreSQL: true,
		DiscoverMySQL:      true,
		DiscoverRedis:      true,
		DiscoverCosmosDB:   true,
	}, "secret-456")
	if err != nil {
		t.Fatalf("expected Azure provider update to succeed, got %v", err)
	}
	if updated.Config.Name != "Azure Updated" || !updated.Config.DiscoverRedis || !updated.Config.DiscoverCosmosDB {
		t.Fatalf("unexpected Azure provider state after update: %#v", updated)
	}

	if err := RemoveAzureProvider("azure-test"); err != nil {
		t.Fatalf("expected Azure provider removal to succeed, got %v", err)
	}
	if _, err := GetAzureProvider("azure-test"); !errors.Is(err, ErrAzureProviderNotFound) {
		t.Fatalf("expected removed Azure provider to be missing, got %v", err)
	}

	providerCfg := configToAzureProviderConfig(cfg, "secret-123")
	if providerCfg.AuthMethod != azureinfra.AuthMethodDefault || providerCfg.ClientSecret != "secret-123" {
		t.Fatalf("expected Azure provider config helper to preserve auth settings, got %#v", providerCfg)
	}
	if got := GenerateAzureProviderID("Azure Test", "sub-123"); got != "azure-AzureTest-sub-123" {
		t.Fatalf("unexpected Azure provider ID: %q", got)
	}
}

func TestAzureProviderErrors(t *testing.T) {
	resetAllCloudProviders()
	skipAzurePersist = true
	defer func() { skipAzurePersist = false }()

	cfg := &AzureProviderConfig{
		ID:             "azure-dup",
		Name:           "Azure Dup",
		SubscriptionID: "sub-dup",
		AuthMethod:     "default",
	}
	if _, err := AddAzureProvider(cfg, "secret"); err != nil {
		t.Fatalf("expected first Azure provider add to succeed, got %v", err)
	}
	if _, err := AddAzureProvider(cfg, "secret"); !errors.Is(err, ErrAzureProviderAlreadyExists) {
		t.Fatalf("expected duplicate Azure provider error, got %v", err)
	}
	if _, err := UpdateAzureProvider("missing", cfg, "secret"); !errors.Is(err, ErrAzureProviderNotFound) {
		t.Fatalf("expected missing Azure provider update error, got %v", err)
	}
	if err := RemoveAzureProvider("missing"); !errors.Is(err, ErrAzureProviderNotFound) {
		t.Fatalf("expected missing Azure provider remove error, got %v", err)
	}
}

func TestGCPProviderCRUDAndHelpers(t *testing.T) {
	resetAllCloudProviders()
	gcpSkipPersist = true
	defer func() { gcpSkipPersist = false }()

	cfg := &GCPProviderConfig{
		ID:                    "gcp-test",
		Name:                  "GCP Test",
		ProjectID:             "project-123",
		Region:                "us-central1",
		AuthMethod:            "service-account-key",
		ServiceAccountKeyPath: "/tmp/key.json",
		DiscoverCloudSQL:      true,
	}

	state, err := AddGCPProvider(cfg)
	if err != nil {
		t.Fatalf("expected GCP provider add to succeed, got %v", err)
	}
	if state.Config.Name != "GCP Test" || state.Status != "Connected" {
		t.Fatalf("unexpected GCP provider state after add: %#v", state)
	}

	got, err := GetGCPProvider("gcp-test")
	if err != nil {
		t.Fatalf("expected GCP provider lookup to succeed, got %v", err)
	}
	if got.Config.ServiceAccountKeyPath != "/tmp/key.json" {
		t.Fatalf("expected GCP provider key path to be preserved, got %#v", got.Config)
	}
	if len(GetGCPProviders()) != 1 {
		t.Fatalf("expected one GCP provider to be listed, got %d", len(GetGCPProviders()))
	}

	updated, err := UpdateGCPProvider("gcp-test", &GCPProviderConfig{
		Name:                "GCP Updated",
		ProjectID:           "project-123",
		Region:              "us-central1",
		AuthMethod:          "default",
		DiscoverCloudSQL:    true,
		DiscoverAlloyDB:     true,
		DiscoverMemorystore: true,
	})
	if err != nil {
		t.Fatalf("expected GCP provider update to succeed, got %v", err)
	}
	if updated.Config.Name != "GCP Updated" || !updated.Config.DiscoverAlloyDB || !updated.Config.DiscoverMemorystore {
		t.Fatalf("unexpected GCP provider state after update: %#v", updated)
	}

	if err := RemoveGCPProvider("gcp-test"); err != nil {
		t.Fatalf("expected GCP provider removal to succeed, got %v", err)
	}
	if _, err := GetGCPProvider("gcp-test"); !errors.Is(err, ErrGCPProviderNotFound) {
		t.Fatalf("expected removed GCP provider to be missing, got %v", err)
	}

	providerCfg := configToGCPProviderConfig(cfg)
	if providerCfg.AuthMethod != gcpinfra.AuthMethodServiceAccountKey || providerCfg.ServiceAccountKeyPath != "/tmp/key.json" {
		t.Fatalf("expected GCP provider config helper to preserve auth settings, got %#v", providerCfg)
	}
	if got := GenerateGCPProviderID("GCP Test", "us-central1"); got != "gcp-GCPTest-us-central1" {
		t.Fatalf("unexpected GCP provider ID: %q", got)
	}
}

func TestGCPProviderErrorsAndTypeLookup(t *testing.T) {
	resetAllCloudProviders()
	skipPersist = true
	gcpSkipPersist = true
	defer func() {
		skipPersist = false
		gcpSkipPersist = false
	}()

	awsCfg := &AWSProviderConfig{
		ID:         "aws-test",
		Name:       "AWS Test",
		Region:     "us-west-2",
		AuthMethod: "default",
	}
	if _, err := AddAWSProvider(awsCfg); err != nil {
		t.Fatalf("expected AWS provider add to succeed, got %v", err)
	}

	gcpCfg := &GCPProviderConfig{
		ID:         "gcp-dup",
		Name:       "GCP Dup",
		ProjectID:  "project-dup",
		Region:     "europe-west1",
		AuthMethod: "default",
	}
	if _, err := AddGCPProvider(gcpCfg); err != nil {
		t.Fatalf("expected first GCP provider add to succeed, got %v", err)
	}
	if _, err := AddGCPProvider(gcpCfg); !errors.Is(err, ErrGCPProviderAlreadyExists) {
		t.Fatalf("expected duplicate GCP provider error, got %v", err)
	}
	if _, err := UpdateGCPProvider("missing", gcpCfg); !errors.Is(err, ErrGCPProviderNotFound) {
		t.Fatalf("expected missing GCP provider update error, got %v", err)
	}
	if err := RemoveGCPProvider("missing"); !errors.Is(err, ErrGCPProviderNotFound) {
		t.Fatalf("expected missing GCP provider remove error, got %v", err)
	}

	if got := GetProviderType("aws-test"); got != providers.ProviderTypeAWS {
		t.Fatalf("expected AWS provider type lookup, got %q", got)
	}
	if got := GetProviderType("gcp-dup"); got != providers.ProviderTypeGCP {
		t.Fatalf("expected GCP provider type lookup, got %q", got)
	}
	if got := GetProviderType("missing"); got != "" {
		t.Fatalf("expected missing provider type lookup to be empty, got %q", got)
	}
}

func TestInitProvidersFromEnv(t *testing.T) {
	resetAllCloudProviders()
	skipPersist = true
	skipAzurePersist = true
	gcpSkipPersist = true
	originalAWS := env.IsAWSProviderEnabled
	originalAzure := env.IsAzureProviderEnabled
	originalGCP := env.IsGCPProviderEnabled
	defer func() {
		skipPersist = false
		skipAzurePersist = false
		gcpSkipPersist = false
		env.IsAWSProviderEnabled = originalAWS
		env.IsAzureProviderEnabled = originalAzure
		env.IsGCPProviderEnabled = originalGCP
	}()

	env.IsAWSProviderEnabled = true
	env.IsAzureProviderEnabled = true
	env.IsGCPProviderEnabled = true

	t.Setenv("WHODB_AWS_PROVIDER", `[{"name":"Env AWS","region":"us-west-2","authMethod":"default","discoverRDS":false}]`)
	t.Setenv("WHODB_AZURE_PROVIDER", `[{"name":"Env Azure","subscriptionId":"sub-env","authMethod":"default","discoverRedis":false}]`)
	t.Setenv("WHODB_GCP_PROVIDER", `[{"projectId":"project-env","region":"us-central1","serviceAccountKeyPath":"/tmp/key.json","discoverMemorystore":false}]`)

	if err := InitAWSProvidersFromEnv(); err != nil {
		t.Fatalf("expected AWS env init to succeed, got %v", err)
	}
	if err := InitAzureProvidersFromEnv(); err != nil {
		t.Fatalf("expected Azure env init to succeed, got %v", err)
	}
	if err := InitGCPProvidersFromEnv(); err != nil {
		t.Fatalf("expected GCP env init to succeed, got %v", err)
	}

	awsProviders := GetAWSProviders()
	if len(awsProviders) != 1 || awsProviders[0].Config.ID != "aws-EnvAWS-us-west-2" || awsProviders[0].Config.DiscoverRDS || !awsProviders[0].Config.DiscoverElastiCache || !awsProviders[0].Config.DiscoverS3 {
		t.Fatalf("unexpected AWS providers initialized from env: %#v", awsProviders)
	}

	azureProviders := GetAzureProviders()
	if len(azureProviders) != 1 || azureProviders[0].Config.ID != "azure-EnvAzure-sub-env" || azureProviders[0].Config.DiscoverRedis || !azureProviders[0].Config.DiscoverPostgreSQL {
		t.Fatalf("unexpected Azure providers initialized from env: %#v", azureProviders)
	}

	gcpProviders := GetGCPProviders()
	if len(gcpProviders) != 1 || gcpProviders[0].Config.Name != "GCP-1" || gcpProviders[0].Config.AuthMethod != "service-account-key" || gcpProviders[0].Config.DiscoverMemorystore || !gcpProviders[0].Config.DiscoverCloudSQL {
		t.Fatalf("unexpected GCP providers initialized from env: %#v", gcpProviders)
	}
}
