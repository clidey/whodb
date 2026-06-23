//go:build e2e_platform

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

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/platform"
)

const (
	defaultPlatformE2EHost        = "http://localhost:18080"
	defaultPlatformE2EKeycloakURL = "http://localhost:14001"
	defaultPlatformE2EUser        = "owner@acme.test"
	defaultPlatformE2EPassword    = "password"
)

type platformAutomationEnvelope struct {
	Command string          `json:"command"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

func TestPlatformCLI_SourceLifecycle(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(os.Getenv("HOME"), ".whodb-cli-platform-e2e-tokens"))
	config.ResetPathsForTesting()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	host := envOrDefault("WHODB_PLATFORM_E2E_HOST", defaultPlatformE2EHost)
	keycloakURL := envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultPlatformE2EKeycloakURL)
	email := envOrDefault("WHODB_PLATFORM_E2E_USER", defaultPlatformE2EUser)
	password := envOrDefault("WHODB_PLATFORM_E2E_PASSWORD", defaultPlatformE2EPassword)

	keycloakHostHeader := envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_HOST_HEADER", "127.0.0.1:4001")
	refreshToken := mintDevRefreshToken(t, ctx, keycloakURL, keycloakHostHeader, email, password)
	restoreLogin := seedPlatformLogin(t, ctx, host, refreshToken)
	defer restoreLogin()

	orgSlug := envOrDefault("WHODB_PLATFORM_E2E_ORG", "acme")
	projectSlug := envOrDefault("WHODB_PLATFORM_E2E_PROJECT", "default")

	sourceName := "cli-e2e-" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	renamedSource := sourceName + "-renamed"
	sourcePassword := envOrDefault("WHODB_PLATFORM_E2E_SOURCE_PASSWORD", "whodb")
	t.Setenv("WHODB_PLATFORM_E2E_SOURCE_PASSWORD", sourcePassword)

	status := runJSONCommand[map[string]any](t, "status", "--host", host, "--format", "json", "--quiet")
	assertString(t, status, "email", email)

	orgs := runJSONCommand[[]platform.Organization](t, "orgs", "list", "--host", host, "--format", "json", "--quiet")
	requireContainsOrg(t, orgs, orgSlug)

	projects := runJSONCommand[[]platform.Project](t, "projects", "list", "--host", host, "--org", orgSlug, "--format", "json", "--quiet")
	requireContainsProject(t, projects, projectSlug)

	useOutput := runEnvelope(t, "use", "--host", host, "--org", orgSlug, "--project", projectSlug, "--format", "json", "--quiet")
	if useOutput.Command != "use" || !useOutput.Success {
		t.Fatalf("use output = %#v, want successful use envelope", useOutput)
	}

	types := runJSONCommand[[]platform.SourceType](t, "sources", "types", "--host", host, "--format", "json", "--quiet")
	requireContainsSourceType(t, types, "Postgres")

	fields := runJSONCommand[[]platform.SourceConnectionField](t, "sources", "fields", "Postgres", "--host", host, "--format", "json", "--quiet")
	requireContainsField(t, fields, "Hostname")
	requireContainsField(t, fields, "Password")

	sourceHost := envOrDefault("WHODB_PLATFORM_E2E_SOURCE_HOST", "platform-db")
	sourcePort := envOrDefault("WHODB_PLATFORM_E2E_SOURCE_PORT", "5432")
	sourceUser := envOrDefault("WHODB_PLATFORM_E2E_SOURCE_USER", "postgres")
	sourceDatabase := envOrDefault("WHODB_PLATFORM_E2E_SOURCE_DATABASE", "whodb_platform")

	_ = runEnvelope(t,
		"sources", "test",
		"--host", host,
		"--type", "Postgres",
		"--hostname", sourceHost,
		"--port", sourcePort,
		"--username", sourceUser,
		"--database", sourceDatabase,
		"--password-env", "WHODB_PLATFORM_E2E_SOURCE_PASSWORD",
		"--format", "json",
		"--quiet",
	)

	created := runEnvelope(t,
		"sources", "create", "Postgres",
		"--host", host,
		"--org", orgSlug,
		"--project", projectSlug,
		"--name", sourceName,
		"--hostname", sourceHost,
		"--port", sourcePort,
		"--username", sourceUser,
		"--database", sourceDatabase,
		"--password-env", "WHODB_PLATFORM_E2E_SOURCE_PASSWORD",
		"--format", "json",
		"--quiet",
	)
	var createdSource platform.Source
	decodeEnvelopeData(t, created, &createdSource)
	if createdSource.Name != sourceName {
		t.Fatalf("created source name = %q, want %q", createdSource.Name, sourceName)
	}
	sourceForCleanup := sourceName
	defer func() {
		_, _, _ = testharness.RunCLI(t, "sources", "delete", sourceForCleanup, "--host", host, "--yes", "--format", "json", "--quiet")
	}()

	sources := runJSONCommand[[]platform.Source](t, "sources", "list", "--host", host, "--format", "json", "--quiet")
	requireContainsSource(t, sources, sourceName)

	configOutput := runRawCommand(t, "sources", "config", sourceName, "--host", host, "--format", "json", "--quiet")
	testharness.AssertContains(t, configOutput, `"password": "********"`)
	testharness.AssertNotContains(t, configOutput, `"password": "`+sourcePassword+`"`)

	updated := runEnvelope(t, "sources", "update", sourceName, "--host", host, "--name", renamedSource, "--format", "json", "--quiet")
	var updatedSource platform.Source
	decodeEnvelopeData(t, updated, &updatedSource)
	if updatedSource.Name != renamedSource {
		t.Fatalf("updated source name = %q, want %q", updatedSource.Name, renamedSource)
	}
	sourceForCleanup = renamedSource

	_ = runEnvelope(t, "sources", "test", renamedSource, "--host", host, "--format", "json", "--quiet")

	deleted := runEnvelope(t, "sources", "delete", renamedSource, "--host", host, "--yes", "--format", "json", "--quiet")
	if deleted.Command != "sources.delete" || !deleted.Success {
		t.Fatalf("delete output = %#v, want successful delete envelope", deleted)
	}
	sourceForCleanup = ""
}

func mintDevRefreshToken(t *testing.T, ctx context.Context, keycloakURL, hostHeader, username, password string) string {
	t.Helper()
	endpoint := strings.TrimRight(keycloakURL, "/") + "/realms/mothergate/protocol/openid-connect/token"
	form := url.Values{
		"client_id":  {"whodb-cli"},
		"grant_type": {"password"},
		"username":   {username},
		"password":   {password},
		"scope":      {"openid email profile"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("create token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if hostHeader != "" {
		req.Host = hostHeader
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("mint local dev token from %s: %v", endpoint, err)
	}
	defer resp.Body.Close()
	var body bytes.Buffer
	if _, err := body.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read token response: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("mint local dev token failed: %s: %s", resp.Status, strings.TrimSpace(body.String()))
	}
	var tokens platform.TokenResponse
	if err := json.Unmarshal(body.Bytes(), &tokens); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	if strings.TrimSpace(tokens.RefreshToken) == "" {
		t.Fatalf("token response did not include refresh token")
	}
	return tokens.RefreshToken
}

func seedPlatformLogin(t *testing.T, ctx context.Context, host, refreshToken string) func() {
	t.Helper()
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("load CLI config: %v", err)
	}
	tokens, err := platform.RefreshToken(ctx, host, refreshToken)
	if err != nil {
		t.Fatalf("refresh local dev token through WhoDB auth host: %v", err)
	}
	client, err := platform.NewClient(host, tokens.AccessToken)
	if err != nil {
		t.Fatalf("new platform client: %v", err)
	}
	manifest, _ := client.PlatformManifest(ctx)
	client.SetPlatformManifest(manifest)
	user, err := client.Me(ctx)
	if err != nil {
		t.Fatalf("load platform user: %v", err)
	}
	hostEntry := config.PlatformHost{URL: client.Host(), AccountID: user.ID, Email: user.Email}
	if manifest != nil {
		raw, err := json.Marshal(manifest)
		if err != nil {
			t.Fatalf("marshal manifest: %v", err)
		}
		hostEntry.Manifest = &config.PlatformManifestCache{
			PlatformVersion:         manifest.PlatformVersion,
			ManifestProtocolVersion: manifest.ManifestProtocolVersion,
			FetchedAt:               time.Now().UTC().Format(time.RFC3339),
			Raw:                     raw,
		}
	}
	selection, err := client.Organizations(ctx)
	if err != nil {
		t.Fatalf("load organizations: %v", err)
	}
	for _, org := range selection {
		if org.Slug == "acme" {
			hostEntry.DefaultOrgID = org.ID
			hostEntry.DefaultOrgName = org.Name
			break
		}
	}
	if hostEntry.DefaultOrgID != "" {
		projects, err := client.Projects(ctx, hostEntry.DefaultOrgID)
		if err != nil {
			t.Fatalf("load projects: %v", err)
		}
		for _, project := range projects {
			if project.Slug == "default" {
				hostEntry.DefaultProjectID = project.ID
				hostEntry.DefaultProjectName = project.Name
				break
			}
		}
	}
	previousToken, previousErr := cfg.GetPlatformRefreshToken(client.Host(), user.ID)
	cfg.SetOnlyPlatformHost(hostEntry)
	tokenToStore := tokens.RefreshToken
	if tokenToStore == "" {
		tokenToStore = refreshToken
	}
	if err := cfg.SavePlatformRefreshToken(client.Host(), user.ID, tokenToStore); err != nil {
		t.Fatalf("save platform refresh token: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save CLI config: %v", err)
	}
	return func() {
		if previousErr == nil {
			_ = cfg.SavePlatformRefreshToken(client.Host(), user.ID, previousToken)
			return
		}
		_ = cfg.DeletePlatformRefreshToken(client.Host(), user.ID)
	}
}

func runRawCommand(t *testing.T, args ...string) string {
	t.Helper()
	stdout, stderr, exitCode := testharness.RunCLI(t, args...)
	if exitCode != 0 {
		t.Fatalf("CLI command failed: whodb-cli %s\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), stdout, stderr)
	}
	return stdout
}

func runJSONCommand[T any](t *testing.T, args ...string) T {
	t.Helper()
	stdout := runRawCommand(t, args...)
	var value T
	if err := json.Unmarshal([]byte(stdout), &value); err != nil {
		t.Fatalf("decode JSON from whodb-cli %s: %v\noutput:\n%s", strings.Join(args, " "), err, stdout)
	}
	return value
}

func runEnvelope(t *testing.T, args ...string) platformAutomationEnvelope {
	t.Helper()
	return runJSONCommand[platformAutomationEnvelope](t, args...)
}

func decodeEnvelopeData(t *testing.T, envelope platformAutomationEnvelope, target any) {
	t.Helper()
	if !envelope.Success {
		t.Fatalf("envelope %s was not successful: %#v", envelope.Command, envelope)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("decode envelope data for %s: %v\nraw:\n%s", envelope.Command, err, string(envelope.Data))
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func assertString(t *testing.T, data map[string]any, key, want string) {
	t.Helper()
	if got, _ := data[key].(string); got != want {
		t.Fatalf("%s = %#v, want %q", key, data[key], want)
	}
}

func requireContainsOrg(t *testing.T, orgs []platform.Organization, slug string) {
	t.Helper()
	for _, org := range orgs {
		if org.Slug == slug {
			return
		}
	}
	t.Fatalf("organization slug %q not found in %#v", slug, orgs)
}

func requireContainsProject(t *testing.T, projects []platform.Project, slug string) {
	t.Helper()
	for _, project := range projects {
		if project.Slug == slug {
			return
		}
	}
	t.Fatalf("project slug %q not found in %#v", slug, projects)
}

func requireContainsSourceType(t *testing.T, types []platform.SourceType, id string) {
	t.Helper()
	for _, sourceType := range types {
		if sourceType.ID == id || sourceType.Connector == id || sourceType.Label == id {
			return
		}
	}
	t.Fatalf("source type %q not found in %#v", id, types)
}

func requireContainsField(t *testing.T, fields []platform.SourceConnectionField, key string) {
	t.Helper()
	for _, field := range fields {
		if field.Key == key {
			return
		}
	}
	t.Fatalf("source field %q not found in %#v", key, fields)
}

func requireContainsSource(t *testing.T, sources []platform.Source, name string) {
	t.Helper()
	for _, source := range sources {
		if source.Name == name {
			return
		}
	}
	t.Fatalf("source %q not found in %#v", name, sources)
}
