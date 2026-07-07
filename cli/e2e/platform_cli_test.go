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
	"fmt"
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

func TestPlatformCLI_ResourceLifecycleAndCapabilities(t *testing.T) {
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
	_ = runEnvelope(t, "use", "--host", host, "--org", orgSlug, "--project", projectSlug, "--format", "json", "--quiet")

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	baseArgs := []string{"--host", host, "--yes", "--format", "json", "--quiet"}

	t.Setenv("WHODB_PLATFORM_E2E_TYPED_SECRET", "secret-value-"+suffix)
	secretName := "cli-e2e-secret-" + suffix
	updatedSecretName := "cli-e2e-secret-updated-" + suffix
	secretID := runMutationID(t, append([]string{
		"secrets", "create",
		"--name", secretName,
		"--description", "CLI e2e secret",
		"--value-env", "WHODB_PLATFORM_E2E_TYPED_SECRET",
	}, baseArgs...)...)
	defer bestEffortCLIResourceDelete(t, host, "secret", secretID)
	secrets := runJSONCommand[[]platform.ProjectSecret](t, "secrets", "list", "--host", host, "--format", "json", "--quiet")
	requireContainsSecret(t, secrets, secretID)
	_ = runMutationID(t, append([]string{
		"secrets", "update", secretName,
		"--name", updatedSecretName,
		"--description", "updated",
		"--value-env", "WHODB_PLATFORM_E2E_TYPED_SECRET",
	}, baseArgs...)...)
	secret := runJSONCommand[platform.ProjectSecret](t, "secrets", "get", updatedSecretName, "--host", host, "--format", "json", "--quiet")
	if secret.ID != secretID {
		t.Fatalf("secrets get by name returned id %q, want %q", secret.ID, secretID)
	}

	t.Setenv("WHODB_PLATFORM_E2E_TYPED_PROVIDER_KEY", "test-key-"+suffix)
	providerName := "cli-e2e-provider-" + suffix
	updatedProviderName := "cli-e2e-provider-updated-" + suffix
	providerID := runMutationID(t, append([]string{
		"ai-providers", "create",
		"--name", providerName,
		"--type", "openai",
		"--endpoint", "http://127.0.0.1:1/v1",
		"--api-key-env", "WHODB_PLATFORM_E2E_TYPED_PROVIDER_KEY",
		"--model", "gpt-4.1",
	}, baseArgs...)...)
	defer bestEffortCLIResourceDelete(t, host, "ai_provider", providerID)
	providers := runJSONCommand[[]platform.AIProvider](t, "ai-providers", "list", "--host", host, "--format", "json", "--quiet")
	requireContainsAIProvider(t, providers, providerID)
	_ = runMutationID(t, append([]string{
		"ai-providers", "update", providerName,
		"--name", updatedProviderName,
		"--endpoint", "http://127.0.0.1:1/v1",
		"--model", "gpt-4.1-mini",
	}, baseArgs...)...)
	provider := runJSONCommand[platform.AIProvider](t, "ai-providers", "get", updatedProviderName, "--host", host, "--format", "json", "--quiet")
	if provider.ID != providerID {
		t.Fatalf("ai-providers get by name returned id %q, want %q", provider.ID, providerID)
	}

	datasetName := "cli-e2e-dataset-" + suffix
	datasetID := runMutationID(t, append([]string{
		"datasets", "create",
		"--name", datasetName,
		"--description", "CLI e2e dataset",
		"--schema-mode", "manual",
		"--column", "id:text:primary",
		"--column", "name:text:nullable",
	}, baseArgs...)...)
	defer bestEffortCLIResourceDelete(t, host, "dataset", datasetID)
	datasets := runJSONCommand[[]platform.Dataset](t, "datasets", "list", "--host", host, "--format", "json", "--quiet")
	requireContainsDataset(t, datasets, datasetID)
	dataset := runJSONCommand[platform.Dataset](t, "datasets", "get", datasetName, "--host", host, "--format", "json", "--quiet")
	if dataset.ID != datasetID {
		t.Fatalf("datasets get by name returned id %q, want %q", dataset.ID, datasetID)
	}
	_ = runJSONCommand[platform.DatasetQueryResult](t, "datasets", "rows", datasetName, "--host", host, "--limit", "5", "--format", "json", "--quiet")
	_ = runJSONCommand[platform.DatasetQueryResult](t, "datasets", "query", datasetName, "--host", host, "--limit", "5", "--format", "json", "--quiet")
	_ = runMutationID(t, append([]string{
		"datasets", "update", datasetName,
		"--description", "updated",
		"--schema-mode", "manual",
		"--column", "id:text:primary",
	}, baseArgs...)...)

	transformName := "cli-e2e-transform-" + suffix
	transformID := runMutationID(t, append([]string{
		"transforms", "create",
		"--name", transformName,
		"--description", "CLI e2e transform",
		"--graph-json", `{"nodes":[],"edges":[]}`,
		"--trigger-mode", "manual",
	}, baseArgs...)...)
	defer bestEffortCLIResourceDelete(t, host, "transform", transformID)
	transform := runJSONCommand[platform.Transform](t, "transforms", "get", transformName, "--host", host, "--format", "json", "--quiet")
	if transform.ID != transformID {
		t.Fatalf("transforms get by name returned id %q, want %q", transform.ID, transformID)
	}
	_ = runMutationID(t, append([]string{
		"transforms", "update", transformName,
		"--description", "updated",
	}, baseArgs...)...)
	_ = runMutationID(t, append([]string{"transforms", "run", transformName}, baseArgs...)...)
	_ = runJSONCommand[[]platform.TransformRun](t, "transforms", "runs", transformName, "--host", host, "--format", "json", "--quiet")

	functionName := "cli-e2e-function-" + suffix
	functionPath := filepath.Join(t.TempDir(), "main.py")
	if err := os.WriteFile(functionPath, []byte("def main(input):\n    return input\n"), 0600); err != nil {
		t.Fatalf("write function fixture: %v", err)
	}
	functionID := runMutationID(t, append([]string{
		"functions", "create",
		"--name", functionName,
		"--description", "CLI e2e function",
		"--language", "python",
		"--entry-point", "main",
		"--file", "main.py=" + functionPath,
		"--provider-id", providerID,
		"--provider-config", providerID + "=gpt-4.1-mini",
		"--secret-binding", "CLI_E2E_SECRET=" + secretID,
		"--default-max-tokens", "256",
		"--default-temperature", "0.2",
	}, baseArgs...)...)
	defer bestEffortCLIResourceDelete(t, host, "function", functionID)
	fn := runJSONCommand[platform.Function](t, "functions", "get", functionName, "--host", host, "--format", "json", "--quiet")
	if fn.ID != functionID {
		t.Fatalf("functions get by name returned id %q, want %q", fn.ID, functionID)
	}
	testResult := runJSONCommand[platform.FunctionExecutionResult](t, "functions", "test", functionName, "--host", host, "--input-json", `{"hello":"draft"}`, "--format", "json", "--quiet")
	requireFunctionExecutionResult(t, "functions test", testResult)
	previewResult := runJSONCommand[platform.FunctionExecutionResult](t, "functions", "preview", "--host", host, "--language", "python", "--entry-point", "main", "--file", "main.py="+functionPath, "--input-json", `{"hello":"preview"}`, "--format", "json", "--quiet")
	requireFunctionExecutionResult(t, "functions preview", previewResult)
	versions := runJSONCommand[[]platform.ObjectVersion](t, "functions", "versions", functionName, "--host", host, "--format", "json", "--quiet")
	if len(versions) != 0 {
		t.Fatalf("new function versions = %#v, want empty", versions)
	}
	active := runJSONCommand[map[string]any](t, "functions", "active", functionName, "--host", host, "--format", "json", "--quiet")
	if got, _ := active["active"].(bool); got {
		t.Fatalf("new function active = %#v, want inactive", active)
	}
	_, deployErr, deployExitCode := runCommandFailure(t, append([]string{"functions", "deploy", functionName}, baseArgs...)...)
	if deployExitCode == 0 || !strings.Contains(deployErr, "Promote it first") {
		t.Fatalf("functions deploy without active version stderr = %q exit = %d, want friendly promote guidance", deployErr, deployExitCode)
	}
	promotedV1 := runEnvelope(t, append([]string{
		"functions", "promote", functionName,
		"--message", "initial version",
	}, baseArgs...)...)
	var version1 platform.ObjectVersion
	decodeEnvelopeData(t, promotedV1, &version1)
	if version1.Version != 1 || version1.ObjectID != functionID {
		t.Fatalf("promote v1 = %#v, want version 1 for %q", version1, functionID)
	}
	activeV1 := runJSONCommand[platform.ActiveProdVersion](t, "functions", "active", functionName, "--host", host, "--format", "json", "--quiet")
	if activeV1.Version != 1 || activeV1.ObjectID != functionID {
		t.Fatalf("active v1 = %#v, want version 1 for %q", activeV1, functionID)
	}
	_ = runMutationID(t, append([]string{
		"functions", "update", functionName,
		"--description", "updated",
		"--default-max-tokens", "128",
		"--default-temperature", "0.1",
	}, baseArgs...)...)
	promotedV2 := runEnvelope(t, append([]string{
		"functions", "promote", functionName,
		"--message", "updated version",
	}, baseArgs...)...)
	var version2 platform.ObjectVersion
	decodeEnvelopeData(t, promotedV2, &version2)
	if version2.Version != 2 || version2.ObjectID != functionID {
		t.Fatalf("promote v2 = %#v, want version 2 for %q", version2, functionID)
	}
	versions = runJSONCommand[[]platform.ObjectVersion](t, "functions", "versions", functionName, "--host", host, "--format", "json", "--quiet")
	requireFunctionVersions(t, versions, 1, 2)
	_ = runMutationID(t, append([]string{
		"functions", "update", functionName,
		"--description", "temporary draft",
	}, baseArgs...)...)
	setActiveV1 := runEnvelope(t, append([]string{
		"functions", "set-active", functionName,
		"--version", "1",
	}, baseArgs...)...)
	decodeEnvelopeData(t, setActiveV1, &activeV1)
	if activeV1.Version != 1 || activeV1.ObjectID != functionID {
		t.Fatalf("set-active v1 = %#v, want version 1 for %q", activeV1, functionID)
	}
	restoredDraft := runEnvelope(t, append([]string{
		"functions", "restore-draft", functionName,
		"--version", "2",
	}, baseArgs...)...)
	decodeEnvelopeData(t, restoredDraft, &fn)
	if fn.Description != "updated" {
		t.Fatalf("restored function description = %q, want updated", fn.Description)
	}
	fn = runJSONCommand[platform.Function](t, "functions", "get", functionName, "--host", host, "--format", "json", "--quiet")
	if fn.Description != "updated" {
		t.Fatalf("function draft after restore = %q, want updated", fn.Description)
	}

	folderAName := "cli-e2e-folder-a-" + suffix
	folderBName := "cli-e2e-folder-b-" + suffix
	folderAID := runMutationID(t, append([]string{
		"folders", "create",
		"--name", folderAName,
	}, baseArgs...)...)
	defer bestEffortCLIResourceDelete(t, host, "folder", folderAID)
	folderBID := runMutationID(t, append([]string{
		"folders", "create",
		"--name", folderBName,
	}, baseArgs...)...)
	defer bestEffortCLIResourceDelete(t, host, "folder", folderBID)
	folder := runJSONCommand[platform.ProjectFolder](t, "folders", "get", folderAName, "--host", host, "--format", "json", "--quiet")
	if folder.ID != folderAID {
		t.Fatalf("folders get by name returned id %q, want %q", folder.ID, folderAID)
	}
	tree := runJSONCommand[[]map[string]any](t, "folders", "tree", "--host", host, "--format", "json", "--quiet")
	requireTreeContainsID(t, tree, folderAID)

	csvPath := filepath.Join(t.TempDir(), "cli-e2e-"+suffix+".csv")
	if err := os.WriteFile(csvPath, []byte("id,name\n1,Ada\n"), 0600); err != nil {
		t.Fatalf("write csv fixture: %v", err)
	}
	uploaded := runEnvelope(t, append([]string{
		"files", "upload",
		"--path", csvPath,
		"--folder-id", folderAID,
	}, baseArgs...)...)
	var uploadedFile platform.ProjectFile
	decodeEnvelopeData(t, uploaded, &uploadedFile)
	if uploadedFile.ID == "" {
		t.Fatalf("uploaded file did not include id: %#v", uploadedFile)
	}
	fileID := uploadedFile.ID
	defer bestEffortCLIResourceDelete(t, host, "file", fileID)
	_ = runJSONCommand[platform.FolderContents](t, "files", "list", "--host", host, "--folder-id", folderAID, "--format", "json", "--quiet")
	file := runJSONCommand[platform.ProjectFile](t, "files", "get", uploadedFile.Name, "--host", host, "--format", "json", "--quiet")
	if file.ID != fileID {
		t.Fatalf("files get by name returned id %q, want %q", file.ID, fileID)
	}
	_ = runRawCommand(t, "files", "preview", uploadedFile.Name, "--host", host, "--format", "json", "--quiet")
	downloadPath := filepath.Join(t.TempDir(), "downloaded.csv")
	_ = runRawCommand(t, "files", "download", uploadedFile.Name, "--host", host, "--out", downloadPath, "--quiet")
	testharness.AssertFileContains(t, downloadPath, "Ada")
	fileBeforePromote := runJSONCommand[platform.ProjectFile](t, "files", "get", fileID, "--host", host, "--format", "json", "--quiet")
	if fileBeforePromote.ID != fileID {
		t.Fatalf("files get by id before promote returned id %q, want %q", fileBeforePromote.ID, fileID)
	}

	promotedDatasetName := "cli-e2e-file-dataset-" + suffix
	promotedDatasetID := runMutationID(t, append([]string{
		"files", "promote-to-dataset", fileID,
		"--name", promotedDatasetName,
		"--description", "Promoted from CLI e2e CSV",
		"--column-map", "id:id:text:primary",
		"--column-map", "name:name:text:nullable",
	}, baseArgs...)...)
	defer bestEffortCLIResourceDelete(t, host, "dataset", promotedDatasetID)
	promotedDataset := runJSONCommand[platform.Dataset](t, "datasets", "get", promotedDatasetName, "--host", host, "--format", "json", "--quiet")
	if promotedDataset.ID != promotedDatasetID {
		t.Fatalf("promoted dataset id = %q, want %q", promotedDataset.ID, promotedDatasetID)
	}
	_ = runJSONCommand[platform.LineageGraph](t, "lineage", "project", "--host", host, "--format", "json", "--quiet")

	runMutationOK(t, append([]string{"datasets", "delete", promotedDatasetName}, baseArgs...)...)
	promotedDatasetID = ""
	runMutationOK(t, append([]string{"files", "delete", uploadedFile.Name}, baseArgs...)...)
	fileID = ""

	moveCSVPath := filepath.Join(t.TempDir(), "cli-e2e-move-"+suffix+".csv")
	if err := os.WriteFile(moveCSVPath, []byte("id,name\n2,Grace\n"), 0600); err != nil {
		t.Fatalf("write move csv fixture: %v", err)
	}
	moveUploaded := runEnvelope(t, append([]string{
		"files", "upload",
		"--path", moveCSVPath,
		"--folder-id", folderAID,
	}, baseArgs...)...)
	var moveFile platform.ProjectFile
	decodeEnvelopeData(t, moveUploaded, &moveFile)
	moveFileID := moveFile.ID
	defer bestEffortCLIResourceDelete(t, host, "file", moveFileID)
	renamedFileName := "cli-e2e-renamed-" + suffix + ".csv"
	moveFileID = runMutationID(t, append([]string{"files", "rename", moveFile.Name, "--name", renamedFileName}, baseArgs...)...)
	moveFileID = runMutationID(t, append([]string{"files", "move", renamedFileName, "--folder-id", folderBID}, baseArgs...)...)
	if moveFileID == "" {
		t.Fatalf("moved file did not include id")
	}
	runMutationOK(t, append([]string{"files", "delete", renamedFileName}, baseArgs...)...)
	moveFileID = ""

	runMutationOK(t, append([]string{"folders", "delete", folderAName}, baseArgs...)...)
	folderAID = ""
	runMutationOK(t, append([]string{"folders", "delete", folderBName}, baseArgs...)...)
	folderBID = ""
	runMutationOK(t, append([]string{"functions", "delete", functionName}, baseArgs...)...)
	functionID = ""
	runMutationOK(t, append([]string{"transforms", "delete", transformName}, baseArgs...)...)
	transformID = ""
	runMutationOK(t, append([]string{"datasets", "delete", datasetName}, baseArgs...)...)
	datasetID = ""
	runMutationOK(t, append([]string{"ai-providers", "delete", updatedProviderName}, baseArgs...)...)
	providerID = ""
	runMutationOK(t, append([]string{"secrets", "delete", updatedSecretName}, baseArgs...)...)
	secretID = ""
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

func runCommandFailure(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	stdout, stderr, exitCode := testharness.RunCLI(t, args...)
	if exitCode == 0 {
		t.Fatalf("CLI command succeeded unexpectedly: whodb-cli %s\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), stdout, stderr)
	}
	return stdout, stderr, exitCode
}

func functionExecutionError(result platform.FunctionExecutionResult) string {
	if result.Error != nil {
		return *result.Error
	}
	return fmt.Sprintf("%#v", result)
}

func requireFunctionExecutionResult(t *testing.T, label string, result platform.FunctionExecutionResult) {
	t.Helper()
	if result.Success {
		return
	}
	errText := functionExecutionError(result)
	if strings.Contains(errText, "neither docker nor podman found in PATH") {
		return
	}
	t.Fatalf("%s returned unsuccessful result: %s", label, errText)
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

func runMutationID(t *testing.T, args ...string) string {
	t.Helper()
	data := runMutationOK(t, args...)
	var decoded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode mutation result for whodb-cli %s: %v\nraw:\n%s", strings.Join(args, " "), err, string(data))
	}
	if decoded.ID == "" {
		t.Fatalf("mutation result for whodb-cli %s did not include id: %s", strings.Join(args, " "), string(data))
	}
	return decoded.ID
}

func runMutationOK(t *testing.T, args ...string) json.RawMessage {
	t.Helper()
	envelope := runEnvelope(t, args...)
	var mutation struct {
		Operation string          `json:"Operation"`
		Data      json.RawMessage `json:"Data"`
	}
	if err := json.Unmarshal(envelope.Data, &mutation); err != nil {
		t.Fatalf("decode mutation envelope data for whodb-cli %s: %v\nraw:\n%s", strings.Join(args, " "), err, string(envelope.Data))
	}
	if len(mutation.Data) == 0 {
		t.Fatalf("mutation envelope for whodb-cli %s did not include data: %s", strings.Join(args, " "), string(envelope.Data))
	}
	return mutation.Data
}

func jsonPayload(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON payload: %v", err)
	}
	return string(raw)
}

func bestEffortCLIResourceDelete(t *testing.T, host, resource, id string) {
	t.Helper()
	if strings.TrimSpace(id) == "" {
		return
	}
	_, _, _ = testharness.RunCLI(t, "resources", "delete", resource, id, "--host", host, "--yes", "--format", "json", "--quiet")
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

func requireContainsSecret(t *testing.T, secrets []platform.ProjectSecret, id string) {
	t.Helper()
	for _, secret := range secrets {
		if secret.ID == id {
			return
		}
	}
	t.Fatalf("secret %q not found in %#v", id, secrets)
}

func requireContainsAIProvider(t *testing.T, providers []platform.AIProvider, id string) {
	t.Helper()
	for _, provider := range providers {
		if provider.ID == id {
			return
		}
	}
	t.Fatalf("AI provider %q not found in %#v", id, providers)
}

func requireContainsDataset(t *testing.T, datasets []platform.Dataset, id string) {
	t.Helper()
	for _, dataset := range datasets {
		if dataset.ID == id {
			return
		}
	}
	t.Fatalf("dataset %q not found in %#v", id, datasets)
}

func requireFunctionVersions(t *testing.T, versions []platform.ObjectVersion, expected ...int) {
	t.Helper()
	seen := make(map[int]bool, len(versions))
	for _, version := range versions {
		seen[version.Version] = true
	}
	for _, version := range expected {
		if !seen[version] {
			t.Fatalf("function version %d not found in %#v", version, versions)
		}
	}
}

func requireTreeContainsID(t *testing.T, tree []map[string]any, id string) {
	t.Helper()
	for _, entry := range tree {
		if got, _ := entry["id"].(string); got == id {
			return
		}
	}
	t.Fatalf("tree entry %q not found in %#v", id, tree)
}
