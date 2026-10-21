//go:build e2e_platform

package e2e_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/appcapture"
	"github.com/clidey/whodb/cli/internal/config"
)

// TestPlatformCLI_PublishedAppCapture exercises real published app rendering,
// interaction and screenshots against a frontend backed by the local platform.
func TestPlatformCLI_PublishedAppCapture(t *testing.T) {
	host := os.Getenv("WHODB_PLATFORM_E2E_CAPTURE_HOST")
	if host == "" {
		t.Skip("set WHODB_PLATFORM_E2E_CAPTURE_HOST to the local frontend with API proxy")
	}
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(os.Getenv("HOME"), ".whodb-cli-platform-e2e-tokens"))
	config.ResetPathsForTesting()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	refresh := mintDevRefreshToken(t, ctx, envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultPlatformE2EKeycloakURL), envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_HOST_HEADER", "127.0.0.1:4001"), defaultPlatformE2EUser, defaultPlatformE2EPassword)
	restoreLogin := seedPlatformLogin(t, ctx, host, refresh)
	defer restoreLogin()
	org := envOrDefault("WHODB_PLATFORM_E2E_ORG", "acme")
	name := "capture-" + time.Now().UTC().Format("20060102150405")
	project := runMutationID(t, "projects", "create", "--host", host, "--org", org, "--name", name, "--format", "json", "--quiet")
	defer bestEffortCLIProjectDelete(t, host, org, name)
	args := []string{"--host", host, "--org", org, "--project", project, "--format", "json", "--quiet"}
	folder := t.TempDir()
	files := map[string]string{
		"app.json":                     `{"name":"Capture acceptance","defaultPage":"main","theme":"light"}`,
		"pages/main/state.json":        `{"status":"Ready"}`,
		"pages/main/tree.json":         `{"version":1,"root":{"id":"root","type":"Col","props":{"gap":4,"p":6},"children":[{"id":"title","type":"Text","props":{"content":"Published capture acceptance"}},{"id":"status","type":"Text","props":{},"bind":{"content":"state.status"}},{"id":"verify","type":"Button","props":{},"on":{"click":"verify"},"children":[{"id":"label","type":"Text","props":{"content":"Check capture"}}]}]}}`,
		"pages/main/scripts/verify.js": `(event, ctx) => { ctx.set('status', 'Capture verified'); }`,
	}
	for path, content := range files {
		target := filepath.Join(folder, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	published := runJSONCommand[struct {
		AppID   string `json:"app_id"`
		Version int    `json:"version"`
	}](t, append([]string{"apps", "publish", folder}, args...)...)
	if published.AppID == "" || published.Version < 1 {
		t.Fatalf("app was not published: %+v", published)
	}
	views := runJSONCommand[struct {
		Views []appcapture.View `json:"views"`
	}](t, append([]string{"apps", "views", published.AppID}, args...)...)
	if len(views.Views) != 1 || views.Views[0].Page != "main" {
		t.Fatalf("published views: %+v", views)
	}
	dir := os.Getenv("WHODB_PLATFORM_E2E_CAPTURE_OUTPUT_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, mobile := range []bool{false, true} {
		label, width, height := "desktop", 1440, 900
		if mobile {
			label, width, height = "mobile", 390, 844
		}
		path := filepath.Join(dir, name+"-"+label+".webp")
		command := []string{"apps", "screenshot", published.AppID, "--page", "main", "--output", path, "--actions", `[{"kind":"expect_text","value":"Published capture acceptance"},{"kind":"click","selector":"[data-node-id=verify]"},{"kind":"expect_text","value":"Capture verified"}]`}
		if mobile {
			command = append(command, "--mobile")
		}
		result := runJSONCommand[struct {
			Width       int                    `json:"width"`
			Height      int                    `json:"height"`
			Text        string                 `json:"visible_text"`
			Passed      bool                   `json:"checks_passed"`
			Diagnostics appcapture.Diagnostics `json:"diagnostics"`
		}](t, append(command, args...)...)
		if !result.Passed || result.Width != width || result.Height != height || !strings.Contains(result.Text, "Capture verified") {
			t.Fatalf("%s capture failed: %+v", label, result)
		}
		d := result.Diagnostics
		if len(d.ConsoleErrors)+len(d.PageErrors)+len(d.FailedRequests)+len(d.HTTPResponses)+len(d.ActionErrors)+len(d.ScriptErrors) != 0 {
			raw, _ := json.Marshal(d)
			t.Fatalf("%s browser errors: %s", label, raw)
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) < 1000 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
			t.Fatalf("%s invalid WebP screenshot: %v", label, err)
		}
		t.Logf("%s: published view, button/state interaction, %dx%d WebP, no browser errors; %s", label, width, height, path)
	}
}
