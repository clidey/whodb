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

package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/spf13/cobra"
)

func TestPlatformHostsWithLogin(t *testing.T) {
	cfg := &config.Config{
		CLISection: config.CLISection{
			Platform: config.PlatformConfig{
				Hosts: []config.PlatformHost{
					{URL: "https://app.whodb.com", AccountID: "user-1", Email: "a@example.com"},
					{URL: "https://stale.whodb.com"},
					{AccountID: "user-2"},
				},
			},
		},
	}

	hosts := platformHostsWithLogin(cfg)
	if len(hosts) != 1 {
		t.Fatalf("len(hosts) = %d, want 1", len(hosts))
	}
	if hosts[0].URL != "https://app.whodb.com" {
		t.Fatalf("host URL = %q, want app host", hosts[0].URL)
	}
}

func TestConfirmPlatformLoginReplacementSkipsPromptWhenApprovedByFlag(t *testing.T) {
	approved, err := confirmPlatformLoginReplacement(io.Discard, []config.PlatformHost{
		{URL: "https://app.whodb.com", AccountID: "user-1"},
	}, true)
	if err != nil {
		t.Fatalf("confirmPlatformLoginReplacement() error = %v", err)
	}
	if !approved {
		t.Fatal("confirmPlatformLoginReplacement() approved = false, want true")
	}
}

func TestLocalLogoutHintIncludesHost(t *testing.T) {
	hint := localLogoutHint("http://localhost:8080")
	if !bytes.Contains([]byte(hint), []byte("whodb-cli logout --host http://localhost:8080 --local")) {
		t.Fatalf("localLogoutHint() = %q, want local logout command", hint)
	}
}

func TestPlatformManifestOutputIncludesCacheMetadata(t *testing.T) {
	host := config.PlatformHost{
		URL: "https://app.whodb.com",
		Manifest: &config.PlatformManifestCache{
			FetchedAt: "2026-06-13T12:00:00Z",
		},
	}
	manifest := &platform.PlatformManifest{
		PlatformVersion:         "1.2.3",
		ManifestProtocolVersion: "1",
		GeneratedAt:             "2026-06-13T11:59:00Z",
		Operations: []platform.PlatformManifestOperation{
			{Name: "Me", Kind: "Query"},
			{Name: "CreateSource", Kind: "Mutation"},
		},
		Types: []platform.PlatformManifestType{{Name: "PlatformUser"}},
	}

	output := platformManifestOutput(host, manifest)

	if output.Host != host.URL || output.PlatformVersion != "1.2.3" || output.FetchedAt != host.Manifest.FetchedAt {
		t.Fatalf("platformManifestOutput() = %#v, want host/version/fetched metadata", output)
	}
	if len(output.Operations) != 2 || len(output.Types) != 1 {
		t.Fatalf("platformManifestOutput() = %#v, want operations and types", output)
	}
}

func TestIsAffirmativeConfirmation(t *testing.T) {
	tests := []struct {
		answer string
		want   bool
	}{
		{"y", true},
		{"Y", true},
		{" yes ", true},
		{"no", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.answer, func(t *testing.T) {
			if got := isAffirmativeConfirmation(tt.answer); got != tt.want {
				t.Fatalf("isAffirmativeConfirmation(%q) = %v, want %v", tt.answer, got, tt.want)
			}
		})
	}
}

func TestParseSourceAdvanced(t *testing.T) {
	advanced, err := parseSourceAdvanced([]string{"sslmode=require", " application_name = whodb "})
	if err != nil {
		t.Fatalf("parseSourceAdvanced() error = %v", err)
	}
	if advanced["sslmode"] != "require" {
		t.Fatalf("sslmode = %q, want require", advanced["sslmode"])
	}
	if advanced["application_name"] != "whodb" {
		t.Fatalf("application_name = %q, want whodb", advanced["application_name"])
	}
}

func TestParseSourceAdvancedRejectsMissingKey(t *testing.T) {
	if _, err := parseSourceAdvanced([]string{"=require"}); err == nil {
		t.Fatal("parseSourceAdvanced() error = nil, want error")
	}
}

func TestSourceTypeFromCreateArgs(t *testing.T) {
	got, err := sourceTypeFromCreateArgs([]string{"Postgres"}, "")
	if err != nil {
		t.Fatalf("sourceTypeFromCreateArgs() error = %v", err)
	}
	if got != "Postgres" {
		t.Fatalf("source type = %q, want Postgres", got)
	}
	if _, err := sourceTypeFromCreateArgs([]string{"Postgres"}, "MySQL"); err == nil {
		t.Fatal("sourceTypeFromCreateArgs() error = nil, want conflict")
	}
}

func TestCollectSourceFieldValuesUsesDefaultsAndConsumesKnownAdvancedFields(t *testing.T) {
	portDefault := "5432"
	fields := []platform.SourceConnectionField{
		{Key: "Hostname", Kind: "Text", Required: true},
		{Key: "Port", Kind: "Text", DefaultValue: &portDefault},
		{Key: "Password", Kind: "Password", Required: true},
		{Key: "SSL Mode", Kind: "Text"},
	}
	values, advanced, err := collectSourceFieldValues(fields, map[string]string{
		"Hostname": "localhost",
		"Password": "secret",
	}, map[string]string{
		"SSL Mode":         "require",
		"application_name": "whodb",
	}, nil)
	if err != nil {
		t.Fatalf("collectSourceFieldValues() error = %v", err)
	}
	if values["Port"] != "5432" {
		t.Fatalf("Port = %q, want default 5432", values["Port"])
	}
	if values["SSL Mode"] != "require" {
		t.Fatalf("SSL Mode = %q, want require", values["SSL Mode"])
	}
	if _, ok := advanced["SSL Mode"]; ok {
		t.Fatalf("advanced still contains consumed SSL Mode: %#v", advanced)
	}
	if advanced["application_name"] != "whodb" {
		t.Fatalf("application_name = %q, want whodb", advanced["application_name"])
	}
}

func TestCollectSourceFieldValuesRejectsUnknownExplicitField(t *testing.T) {
	_, _, err := collectSourceFieldValues([]platform.SourceConnectionField{
		{Key: "Hostname", Kind: "Text"},
	}, map[string]string{"Database": "app"}, nil, nil)
	if err == nil {
		t.Fatal("collectSourceFieldValues() error = nil, want unknown field error")
	}
	if !strings.Contains(err.Error(), "sources fields") {
		t.Fatalf("collectSourceFieldValues() error = %q, want fields hint", err)
	}
}

func TestBuildCreateSourceInputMapsKnownFieldsAndAdvanced(t *testing.T) {
	input := buildCreateSourceInput("proj-1", "Postgres", "Warehouse", map[string]string{
		"Hostname": "localhost",
		"Port":     "5432",
		"Username": "postgres",
		"Password": "secret",
		"Database": "test_db",
		"SSL Mode": "require",
	}, map[string]string{"application_name": "whodb"})

	if input.ProjectID != "proj-1" || input.DatabaseType != "Postgres" || input.Name != "Warehouse" {
		t.Fatalf("basic input fields = %#v", input)
	}
	if input.Hostname != "localhost" || input.Port != "5432" || input.Username != "postgres" || input.Password != "secret" || input.Database != "test_db" {
		t.Fatalf("connection input fields = %#v", input)
	}
	if input.Advanced["SSL Mode"] != "require" || input.Advanced["application_name"] != "whodb" {
		t.Fatalf("advanced = %#v, want SSL Mode and application_name", input.Advanced)
	}
}

func TestMergeSourceConfigPreservesUnchangedFields(t *testing.T) {
	existing := &platform.SourceConfig{
		Hostname: "db.example.com",
		Port:     "5432",
		Username: "postgres",
		Password: "********",
		Database: "app",
		Advanced: map[string]string{
			"sslmode":          "disable",
			"application_name": "whodb",
		},
	}

	merged := mergeSourceConfig(existing, map[string]string{
		"Database": "analytics",
		"SSL Mode": "require",
	}, map[string]string{
		"timezone": "UTC",
	})

	if merged.Hostname != "db.example.com" || merged.Port != "5432" || merged.Username != "postgres" || merged.Password != "********" {
		t.Fatalf("connection fields = %#v, want existing values preserved", merged)
	}
	if merged.Database != "analytics" {
		t.Fatalf("database = %q, want analytics", merged.Database)
	}
	if merged.Advanced["SSL Mode"] != "require" || merged.Advanced["application_name"] != "whodb" || merged.Advanced["timezone"] != "UTC" {
		t.Fatalf("advanced = %#v, want merged advanced fields", merged.Advanced)
	}
}

func TestRedactSourceConfigRedactsSecretFields(t *testing.T) {
	sourceType := &platform.SourceType{
		ConnectionFields: []platform.SourceConnectionField{
			{Key: "Password", Kind: "Password"},
			{Key: "Client Secret", Kind: "Password"},
			{Key: "Region", Kind: "Text"},
		},
	}
	config := &platform.SourceConfig{
		Hostname: "db.example.com",
		Password: "secret",
		Advanced: map[string]string{
			"Client Secret": "client-secret",
			"api_token":     "token",
			"Region":        "us-east-1",
		},
	}

	safe := redactSourceConfig(config, sourceType)

	if safe.Password != redactedValue {
		t.Fatalf("password = %q, want redacted", safe.Password)
	}
	if safe.Advanced["Client Secret"] != redactedValue || safe.Advanced["api_token"] != redactedValue {
		t.Fatalf("advanced = %#v, want secret fields redacted", safe.Advanced)
	}
	if safe.Advanced["Region"] != "us-east-1" {
		t.Fatalf("Region = %q, want visible value", safe.Advanced["Region"])
	}
}

func TestReadSourcePasswordFromStdin(t *testing.T) {
	oldPasswordIn := sourcePasswordIn
	oldPasswordEnv := sourcePasswordEnv
	t.Cleanup(func() {
		sourcePasswordIn = oldPasswordIn
		sourcePasswordEnv = oldPasswordEnv
	})
	sourcePasswordIn = true
	sourcePasswordEnv = ""

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("secret\n"))
	password, err := readSourcePassword(cmd)
	if err != nil {
		t.Fatalf("readSourcePassword() error = %v", err)
	}
	if password != "secret" {
		t.Fatalf("password = %q, want secret", password)
	}
}

func TestConfirmSourceDeleteSkipsPromptWhenApprovedByFlag(t *testing.T) {
	approved, err := confirmSourceDelete(io.Discard, &platform.Source{ID: "src-1", Name: "Warehouse"}, true)
	if err != nil {
		t.Fatalf("confirmSourceDelete() error = %v", err)
	}
	if !approved {
		t.Fatal("confirmSourceDelete() approved = false, want true")
	}
}

func TestParseRequiredSourceObjectRefDatabasePath(t *testing.T) {
	ref, err := parseRequiredSourceObjectRef("table:public.users")
	if err != nil {
		t.Fatalf("parseRequiredSourceObjectRef() error = %v", err)
	}
	if ref.Kind != "Table" {
		t.Fatalf("kind = %q, want Table", ref.Kind)
	}
	if len(ref.Path) != 2 || ref.Path[0] != "public" || ref.Path[1] != "users" {
		t.Fatalf("path = %#v, want public/users", ref.Path)
	}
}

func TestParseRequiredSourceObjectRefObjectPath(t *testing.T) {
	ref, err := parseRequiredSourceObjectRef("item:bucket/reports/users.csv")
	if err != nil {
		t.Fatalf("parseRequiredSourceObjectRef() error = %v", err)
	}
	if ref.Kind != "Item" {
		t.Fatalf("kind = %q, want Item", ref.Kind)
	}
	if len(ref.Path) != 3 || ref.Path[0] != "bucket" || ref.Path[1] != "reports" || ref.Path[2] != "users.csv" {
		t.Fatalf("path = %#v, want bucket/reports/users.csv", ref.Path)
	}
}

func TestParseRequiredSourceObjectRefRejectsUnknownKind(t *testing.T) {
	if _, err := parseRequiredSourceObjectRef("unknown:thing"); err == nil {
		t.Fatal("parseRequiredSourceObjectRef() error = nil, want error")
	}
}

func TestValidatePlatformPageRejectsLargeLimit(t *testing.T) {
	if err := validatePlatformPage(1001, 0); err == nil {
		t.Fatal("validatePlatformPage() error = nil, want error")
	}
}
