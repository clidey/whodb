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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/platform"
)

const (
	cloneE2EPages        = 30
	cloneE2EFilesPerPage = 100
	cloneE2EFileBytes    = 2000
)

// cloneE2EResult is the apps clone JSON output.
type cloneE2EResult struct {
	TargetAppID   string   `json:"targetAppId"`
	Name          string   `json:"name"`
	Action        string   `json:"action"`
	Files         int      `json:"files"`
	BrandFiles    int      `json:"brandFiles"`
	BackupVersion *int     `json:"backupVersion"`
	DryRun        bool     `json:"dryRun"`
	Warnings      []string `json:"warnings"`
	Undo          string   `json:"undo"`
}

// TestPlatformCLI_AppCloneLargeAppLifecycle clones an app whose files total
// more than the CLI's 4 MiB response cap, then exercises the dry run, a real
// clone, overwrite with brand files, rollback on a failed write, permission
// refusal, and both undo paths against a live platform.
func TestPlatformCLI_AppCloneLargeAppLifecycle(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(os.Getenv("HOME"), ".whodb-cli-platform-e2e-tokens"))
	config.ResetPathsForTesting()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	host := envOrDefault("WHODB_PLATFORM_E2E_HOST", defaultPlatformE2EHost)
	keycloakURL := envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultPlatformE2EKeycloakURL)
	keycloakHostHeader := envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_HOST_HEADER", "127.0.0.1:4001")
	email := envOrDefault("WHODB_PLATFORM_E2E_USER", defaultPlatformE2EUser)
	password := envOrDefault("WHODB_PLATFORM_E2E_PASSWORD", defaultPlatformE2EPassword)
	restoreLogin := seedPlatformLogin(t, ctx, host, mintDevRefreshToken(t, ctx, keycloakURL, keycloakHostHeader, email, password))
	defer restoreLogin()
	api := func(user, orgID, projectID string) *platform.Client {
		return cloneE2EAPIClient(t, ctx, host, keycloakURL, keycloakHostHeader, user, password, orgID, projectID)
	}

	orgSlug := envOrDefault("WHODB_PLATFORM_E2E_ORG", "acme")
	queryArgs := []string{"--host", host, "--format", "json", "--quiet"}
	writeArgs := append([]string{"--yes"}, queryArgs...)
	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	srcName, dstName := "cli-e2e-clone-src-"+suffix, "cli-e2e-clone-dst-"+suffix
	srcID := runMutationID(t, append([]string{"projects", "create", "--org", orgSlug, "--name", srcName}, queryArgs...)...)
	defer bestEffortCLIProjectDelete(t, host, orgSlug, srcName)
	dstID := runMutationID(t, append([]string{"projects", "create", "--org", orgSlug, "--name", dstName}, queryArgs...)...)
	defer bestEffortCLIProjectDelete(t, host, orgSlug, dstName)
	src := []string{"--org", orgSlug, "--project", srcID}
	dst := []string{"--org", orgSlug, "--project", dstID}

	customerAPIName, orphanAPIName := "clone_customer_"+suffix, "clone_orphan_"+suffix
	srcCustomer := cloneE2ECreateOntology(t, customerAPIName, append(src, writeArgs...))
	srcOrphan := cloneE2ECreateOntology(t, orphanAPIName, append(src, writeArgs...))
	dstCustomer := cloneE2ECreateOntology(t, customerAPIName, append(dst, writeArgs...))

	appPayload, _ := json.Marshal(map[string]any{"name": "Ledger", "description": "Large clone fixture", "ontologyIds": []string{srcCustomer, srcOrphan}})
	srcAppID := runMutationID(t, append(append([]string{"resources", "create", "app", "--payload-json", string(appPayload)}, src...), writeArgs...)...)

	orgID := cloneE2EOrgID(t, api(email, "", ""), orgSlug)
	srcAPI := api(email, orgID, srcID)
	start := time.Now()
	sourceFiles := cloneE2ESeedFiles(t, ctx, srcAPI, srcID, srcAppID)
	t.Logf("seeded %d files (%d bytes each) in %s", len(sourceFiles), cloneE2EFileBytes, time.Since(start).Round(time.Millisecond))

	// The original failure: one AppFiles response for this app exceeds the cap.
	if _, err := srcAPI.PlatformQuery(ctx, "AppFiles", map[string]any{"projectId": srcID, "appId": srcAppID}); err == nil || !strings.Contains(err.Error(), "exceeded 4 MiB") {
		t.Fatalf("AppFiles error = %v, want the 4 MiB response cap to reproduce the original failure", err)
	}
	// The paged read the MCP tool now uses returns every file.
	if got := cloneE2EReadFiles(t, ctx, srcAPI, srcID, srcAppID); !reflect.DeepEqual(got, sourceFiles) {
		t.Fatalf("paged source read returned %d files, want the %d seeded", len(got), len(sourceFiles))
	}

	cloneArgs := append(append([]string{"apps", "clone", "Ledger", "--to-project", dstID}, src...), queryArgs...)

	start = time.Now()
	plan := runJSONCommand[cloneE2EResult](t, cloneArgs...)
	t.Logf("dry run took %s", time.Since(start).Round(time.Millisecond))
	if !plan.DryRun || plan.Action != "create" || plan.Files != len(sourceFiles) || len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], orphanAPIName) {
		t.Fatalf("dry run = %#v, want a create plan for %d files warning about %s", plan, len(sourceFiles), orphanAPIName)
	}
	if apps := cloneE2EApps(t, append(dst, queryArgs...)); len(apps) != 0 {
		t.Fatalf("dry run created apps: %#v", apps)
	}

	start = time.Now()
	created := runJSONCommand[cloneE2EResult](t, append(cloneArgs, "--yes")...)
	t.Logf("clone of %d files took %s", len(sourceFiles), time.Since(start).Round(time.Millisecond))
	if created.DryRun || created.Action != "created" || created.TargetAppID == "" || created.BackupVersion != nil || !strings.Contains(created.Undo, "undo-clone "+created.TargetAppID) {
		t.Fatalf("clone = %#v, want a created app with an undo command", created)
	}
	dstAPI := api(email, orgID, dstID)
	if got := cloneE2EReadFiles(t, ctx, dstAPI, dstID, created.TargetAppID); !reflect.DeepEqual(got, sourceFiles) {
		t.Fatalf("cloned app has %d files, want an exact copy of the %d source files", len(got), len(sourceFiles))
	}
	if detail := cloneE2EAppDetail(t, ctx, dstAPI, dstID, created.TargetAppID); !reflect.DeepEqual(detail.OntologyIDs, []string{dstCustomer}) || detail.Description != "Large clone fixture" {
		t.Fatalf("cloned app = %#v, want description copied and ontology remapped to %s", detail, dstCustomer)
	}

	_, stderr, _ := runCommandFailure(t, append(cloneArgs, "--yes")...)
	if !strings.Contains(stderr, "already exists") {
		t.Fatalf("second clone stderr = %q, want an already-exists refusal", stderr)
	}

	// Edit the clone so restoring the overwrite's backup is observable.
	editedPath := sortedCloneE2EPaths(sourceFiles)[10]
	cloneE2EUpsert(t, ctx, dstAPI, dstID, created.TargetAppID, editedPath, "edited in target")
	beforeOverwrite := cloneE2EReadFiles(t, ctx, dstAPI, dstID, created.TargetAppID)

	// A file the database rejects fails the overwrite after the app was
	// updated; nothing may change, including the backup versions.
	badBrand := t.TempDir()
	if err := os.WriteFile(filepath.Join(badBrand, "broken.json"), []byte("bad\x00byte"), 0o600); err != nil {
		t.Fatal(err)
	}
	versionsBefore := cloneE2EVersionCount(t, ctx, dstAPI, dstID, created.TargetAppID)
	_, stderr, _ = runCommandFailure(t, append(cloneArgs, "--overwrite", "--brand-dir", badBrand, "--yes")...)
	if strings.Contains(stderr, "permission") {
		t.Fatalf("failed overwrite stderr = %q, want no permission-cleanup error", stderr)
	}
	if got := cloneE2EReadFiles(t, ctx, dstAPI, dstID, created.TargetAppID); !reflect.DeepEqual(got, beforeOverwrite) {
		t.Fatal("failed overwrite changed the target app's files")
	}
	if got := cloneE2EVersionCount(t, ctx, dstAPI, dstID, created.TargetAppID); got != versionsBefore {
		t.Fatalf("failed overwrite left %d versions, want %d", got, versionsBefore)
	}
	// The same failure on a new app leaves no app behind.
	_, stderr, _ = runCommandFailure(t, append(cloneArgs, "--name-prefix", "Broken ", "--brand-dir", badBrand, "--yes")...)
	if strings.Contains(stderr, "permission") {
		t.Fatalf("failed clone stderr = %q, want no permission-cleanup error", stderr)
	}
	if apps := cloneE2EApps(t, append(dst, queryArgs...)); len(apps) != 1 {
		t.Fatalf("failed clone left apps %#v, want only the first clone", apps)
	}

	brand := t.TempDir()
	if err := os.WriteFile(filepath.Join(brand, "app.json"), []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(brand, "shared"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brand, "shared", "brand.json"), []byte(`{"name":"Acme"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	overwritten := runJSONCommand[cloneE2EResult](t, append(cloneArgs, "--overwrite", "--brand-dir", brand, "--yes")...)
	if overwritten.Action != "updated" || overwritten.TargetAppID != created.TargetAppID || overwritten.BackupVersion == nil || overwritten.BrandFiles != 2 {
		t.Fatalf("overwrite = %#v, want an update of %s with a backup version and two brand files", overwritten, created.TargetAppID)
	}
	afterOverwrite := cloneE2EReadFiles(t, ctx, dstAPI, dstID, created.TargetAppID)
	if afterOverwrite[editedPath] != sourceFiles[editedPath] || afterOverwrite["shared/brand.json"] != `{"name":"Acme"}` {
		t.Fatalf("overwrite left %s = %q and shared/brand.json = %q, want source content and the brand file", editedPath, afterOverwrite[editedPath], afterOverwrite["shared/brand.json"])
	}
	var appJSON map[string]any
	if err := json.Unmarshal([]byte(afterOverwrite["app.json"]), &appJSON); err != nil || appJSON["theme"] != "dark" || appJSON["defaultPage"] != "pg00" {
		t.Fatalf("app.json after overwrite = %s (%v), want the brand theme merged over the source", afterOverwrite["app.json"], err)
	}

	undoArgs := strings.Fields(strings.TrimPrefix(overwritten.Undo, "whodb "))
	restored := runJSONCommand[map[string]any](t, append(undoArgs, writeArgs...)...)
	if restored["action"] != "restored" {
		t.Fatalf("undo overwrite = %#v, want restored", restored)
	}
	// An exact match also proves shared/brand.json was removed again.
	if got := cloneE2EReadFiles(t, ctx, dstAPI, dstID, created.TargetAppID); !reflect.DeepEqual(got, beforeOverwrite) {
		t.Fatal("restoring the backup did not return the app to its pre-overwrite files")
	}

	// A project editor without share rights on the target ontology is refused
	// before anything is written, matching what Mothergate would enforce.
	memberEmail := envOrDefault("WHODB_PLATFORM_E2E_MEMBER_USER", "member@acme.test")
	cloneE2EShare(t, ctx, api(email, orgID, dstID), memberEmail, "editor", "project", dstID)
	cloneE2EShare(t, ctx, api(email, orgID, srcID), memberEmail, "viewer", "app", srcAppID)
	memberAPI := api(memberEmail, orgID, srcID)
	_, err := memberAPI.PlatformMutation(ctx, "CloneApp", map[string]any{"projectId": srcID, "appId": srcAppID, "targetProjectId": dstID, "name": "Member Copy", "dryRun": true})
	if err == nil || !strings.Contains(err.Error(), dstCustomer) {
		t.Fatalf("member clone error = %v, want a refusal naming ontology %s", err, dstCustomer)
	}

	if targetOrg := os.Getenv("WHODB_PLATFORM_E2E_CLONE_TARGET_ORG"); targetOrg != "" {
		otherName := "cli-e2e-clone-other-org-" + suffix
		otherID := runMutationID(t, append([]string{"projects", "create", "--org", targetOrg, "--name", otherName}, queryArgs...)...)
		defer bestEffortCLIProjectDelete(t, host, targetOrg, otherName)
		otherCustomer := cloneE2ECreateOntology(t, customerAPIName, append([]string{"--org", targetOrg, "--project", otherID}, writeArgs...))
		crossOrg := runJSONCommand[cloneE2EResult](t, append(append([]string{"apps", "clone", "Ledger", "--to-org", targetOrg, "--to-project", otherID, "--yes"}, src...), queryArgs...)...)
		otherOrgID := cloneE2EOrgID(t, api(email, "", ""), targetOrg)
		otherAPI := api(email, otherOrgID, otherID)
		if got := cloneE2EReadFiles(t, ctx, otherAPI, otherID, crossOrg.TargetAppID); !reflect.DeepEqual(got, sourceFiles) {
			t.Fatalf("cross-org clone has %d files, want %d", len(got), len(sourceFiles))
		}
		if detail := cloneE2EAppDetail(t, ctx, otherAPI, otherID, crossOrg.TargetAppID); !reflect.DeepEqual(detail.OntologyIDs, []string{otherCustomer}) {
			t.Fatalf("cross-org bindings = %v, want %s", detail.OntologyIDs, otherCustomer)
		}
		// Source visibility does not grant write access to another organization.
		for _, dryRun := range []bool{true, false} {
			_, err := memberAPI.PlatformMutation(ctx, "CloneApp", map[string]any{
				"projectId": srcID, "appId": srcAppID, "targetOrgId": otherOrgID,
				"targetProjectId": otherID, "name": "Denied Cross Org", "dryRun": dryRun,
			})
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "forbidden") {
				t.Fatalf("unauthorized cross-org clone (dryRun=%t): %v", dryRun, err)
			}
		}
		memberAPI.SetWorkspaceContext(otherOrgID, otherID)
		if _, err := memberAPI.PlatformQuery(ctx, "AppFiles", map[string]any{"projectId": otherID, "appId": crossOrg.TargetAppID}); err == nil {
			t.Fatal("forged target workspace headers exposed cross-org app files")
		}
		if apps := cloneE2EApps(t, append([]string{"--org", targetOrg, "--project", otherID}, queryArgs...)); len(apps) != 1 {
			t.Fatalf("denied cross-org clone changed target apps: %#v", apps)
		}
		t.Log("cross-organization clone preserved files and remapped bindings; unauthorized preview, write and file read were denied")
	} else {
		t.Log("WHODB_PLATFORM_E2E_CLONE_TARGET_ORG unset; skipping the cross-organization clone")
	}

	undoArgs = strings.Fields(strings.TrimPrefix(created.Undo, "whodb "))
	deleted := runJSONCommand[map[string]any](t, append(undoArgs, writeArgs...)...)
	if deleted["action"] != "deleted" {
		t.Fatalf("undo clone = %#v, want deleted", deleted)
	}
	if apps := cloneE2EApps(t, append(dst, queryArgs...)); len(apps) != 0 {
		t.Fatalf("undo left apps %#v", apps)
	}
}

func cloneE2EAPIClient(t *testing.T, ctx context.Context, host, keycloakURL, hostHeader, user, password, orgID, projectID string) *platform.Client {
	t.Helper()
	tokens, err := platform.RefreshToken(ctx, host, mintDevRefreshToken(t, ctx, keycloakURL, hostHeader, user, password))
	if err != nil {
		t.Fatalf("refresh token for %s: %v", user, err)
	}
	client, err := platform.NewClient(host, tokens.AccessToken)
	if err != nil {
		t.Fatalf("new platform client: %v", err)
	}
	manifest, err := client.PlatformManifest(ctx)
	if err != nil {
		t.Fatalf("platform manifest: %v", err)
	}
	client.SetPlatformManifest(manifest)
	if orgID != "" {
		client.SetWorkspaceContext(orgID, projectID)
	}
	return client
}

func cloneE2EOrgID(t *testing.T, client *platform.Client, slug string) string {
	t.Helper()
	orgs, err := client.Organizations(context.Background())
	if err != nil {
		t.Fatalf("organizations: %v", err)
	}
	for _, org := range orgs {
		if org.Slug == slug || org.ID == slug {
			return org.ID
		}
	}
	t.Fatalf("organization %s not found", slug)
	return ""
}

func cloneE2ECreateOntology(t *testing.T, apiName string, args []string) string {
	t.Helper()
	return runMutationID(t, append([]string{
		"ontologies", "create",
		"--api-name", apiName,
		"--display-name", apiName,
		"--plural-display-name", apiName + "s",
		"--primary-key", "id",
		"--table-name", apiName,
		"--schema-name", "public",
		"--property-json", `{"apiName":"id","displayName":"ID","description":"ID","columnName":"id","dataType":"String","isRequired":true,"visibility":"normal","isSearchable":true,"isSortable":true,"isEditOnly":false}`,
	}, args...)...)
}

// cloneE2ESeedFiles writes app.json, a shared file, and pages of generated
// files in parallel, returning every draft file the app ends with by path.
func cloneE2ESeedFiles(t *testing.T, ctx context.Context, client *platform.Client, projectID, appID string) map[string]string {
	t.Helper()
	files := map[string]string{
		"app.json":          `{"defaultPage":"pg00","theme":"light"}`,
		"shared/state.json": `{"seeded":true}`,
	}
	for page := 0; page < cloneE2EPages; page++ {
		for file := 0; file < cloneE2EFilesPerPage; file++ {
			path := fmt.Sprintf("pages/pg%02d/f%03d.json", page, file)
			files[path] = strings.Repeat(fmt.Sprintf("%02d%03d", page, file), cloneE2EFileBytes/5)
		}
	}
	paths := make(chan string)
	var wg sync.WaitGroup
	var failed sync.Once
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range paths {
				if _, err := client.PlatformMutation(ctx, "UpsertAppFile", map[string]any{"projectId": projectID, "appId": appID, "path": path, "content": files[path]}); err != nil {
					failed.Do(func() { t.Errorf("seed %s: %v", path, err) })
				}
			}
		}()
	}
	for path := range files {
		paths <- path
	}
	close(paths)
	wg.Wait()
	if t.Failed() {
		t.FailNow()
	}
	// CreateApp seeds pages/main and shared/state.json; the read below is the truth.
	return cloneE2EReadFiles(t, ctx, client, projectID, appID)
}

func cloneE2EReadFiles(t *testing.T, ctx context.Context, client *platform.Client, projectID, appID string) map[string]string {
	t.Helper()
	files, err := platform.ReadAppFilesByPage(ctx, client, projectID, appID)
	if err != nil {
		t.Fatalf("read app files: %v", err)
	}
	byPath := make(map[string]string, len(files))
	for _, file := range files {
		byPath[file.Path] = file.Content
	}
	return byPath
}

func cloneE2EUpsert(t *testing.T, ctx context.Context, client *platform.Client, projectID, appID, path, content string) {
	t.Helper()
	if _, err := client.PlatformMutation(ctx, "UpsertAppFile", map[string]any{"projectId": projectID, "appId": appID, "path": path, "content": content}); err != nil {
		t.Fatalf("upsert %s: %v", path, err)
	}
}

type cloneE2EApp struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	OntologyIDs []string `json:"ontologyIds"`
}

func cloneE2EApps(t *testing.T, args []string) []cloneE2EApp {
	t.Helper()
	return runJSONCommand[[]cloneE2EApp](t, append([]string{"apps", "list"}, args...)...)
}

func cloneE2EAppDetail(t *testing.T, ctx context.Context, client *platform.Client, projectID, appID string) cloneE2EApp {
	t.Helper()
	data, err := client.PlatformQuery(ctx, "AppDetail", map[string]any{"projectId": projectID, "id": appID})
	if err != nil {
		t.Fatalf("app detail: %v", err)
	}
	raw, _ := json.Marshal(data)
	var app cloneE2EApp
	if err := json.Unmarshal(raw, &app); err != nil {
		t.Fatalf("decode app detail: %v", err)
	}
	return app
}

func cloneE2EVersionCount(t *testing.T, ctx context.Context, client *platform.Client, projectID, appID string) int {
	t.Helper()
	versions, err := client.ObjectVersions(ctx, projectID, appID, "app")
	if err != nil {
		t.Fatalf("object versions: %v", err)
	}
	return len(versions)
}

func cloneE2EShare(t *testing.T, ctx context.Context, client *platform.Client, email, role, resourceType, resourceID string) {
	t.Helper()
	if _, err := client.PlatformMutation(ctx, "ShareByEmail", map[string]any{"emails": []string{email}, "role": role, "resourceType": resourceType, "resourceId": resourceID}); err != nil {
		t.Fatalf("share %s %s with %s: %v", resourceType, resourceID, email, err)
	}
}

func sortedCloneE2EPaths(files map[string]string) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
