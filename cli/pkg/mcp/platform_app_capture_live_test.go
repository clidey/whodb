//go:build e2e_platform && e2e_browser

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestPlatformAppCapture_RealPublishedApp requires the current frontend, sandbox
// kernel, backend and Playwright Chromium. Unlike the catalog's error-path smoke
// check it must render and interact with a promoted app successfully.
func TestPlatformAppCapture_RealPublishedApp(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(t.TempDir(), "tokens"))
	config.ResetPathsForTesting()
	clearPendingPlatformActions(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	host := liveEnvOrDefault("WHODB_PLATFORM_E2E_HOST", defaultLivePlatformE2EHost)
	parsed, err := url.Parse(host)
	if err != nil || (parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1") {
		t.Fatal("published capture fixtures require a loopback local stack")
	}
	refresh := liveMintDevRefreshToken(t, ctx, liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultLivePlatformE2EKeycloakURL), "127.0.0.1:4001", defaultLivePlatformE2EUser, defaultLivePlatformE2EPassword)
	liveSeedPlatformLogin(t, ctx, host, refresh)
	name := fmt.Sprint("capture-", time.Now().UnixNano())
	app := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{Resource: "app", Payload: map[string]any{"name": name, "description": "Published capture regression", "ontologyIds": []any{}}})
	defer liveBestEffortGenericDelete(ctx, "app", app)
	files := map[string]string{
		"app.json":                      `{"defaultPage":"main","offline":{"entities":[],"writes":[]}}`,
		"pages/main/state.json":         `{"status":"Ready"}`,
		"pages/main/tree.json":          `{"version":1,"root":{"id":"root","type":"Col","props":{"gap":4,"p":6},"children":[{"id":"title","type":"Text","props":{"content":"Published capture main"}},{"id":"status","type":"Text","props":{},"bind":{"content":"state.status"}},{"id":"name","type":"Input","props":{"placeholder":"Capture name"}},{"id":"confirm","type":"Button","props":{},"on":{"click":"confirm"},"children":[{"id":"confirm-text","type":"Text","props":{"content":"Confirm capture"}}]},{"id":"next","type":"Button","props":{},"on":{"click":"next"},"children":[{"id":"next-text","type":"Text","props":{"content":"Open details"}}]}]}}`,
		"pages/main/scripts/confirm.js": `(event,ctx)=>ctx.set('status','Capture clicked')`,
		"pages/main/scripts/next.js":    `(event,ctx)=>ctx.navigate('details')`,
		"pages/details/state.json":      `{}`,
		"pages/details/tree.json":       `{"version":1,"root":{"id":"details","type":"Col","props":{"p":6},"children":[{"id":"details-title","type":"Text","props":{"content":"Published details page"}}]}}`,
	}
	for path, content := range files {
		liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "app", Action: "upsert_file", Payload: map[string]any{"appId": app, "path": path, "content": content}})
	}
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "object", Action: "promote", Payload: map[string]any{"objectId": app, "objectType": "app", "message": "Publish capture regression"}})
	// A later draft must never leak into the default published capture.
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "app", Action: "upsert_file", Payload: map[string]any{"appId": app, "path": "pages/details/tree.json", "content": `{"version":1,"root":{"id":"draft","type":"Text","props":{"content":"UNPUBLISHED DRAFT"}}}`}})
	client := scopeTestClient(t)
	views := scopeCall(t, client, "whodb_platform_app_views", map[string]any{"app": app})
	if views["error"] != nil {
		t.Fatal(views["error"])
	}
	if len(views["views"].([]any)) != 2 {
		t.Fatalf("missing published pages: %#v", views)
	}
	// Exercise real tools/call, including image content and structured diagnostics.
	capture := func(args map[string]any) PlatformAppScreenshotOutput {
		t.Helper()
		args["app"] = app
		result, err := client.CallTool(ctx, &mcp.CallToolParams{Name: "whodb_platform_app_screenshot", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		var out PlatformAppScreenshotOutput
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatal(err)
		}
		if out.Error != "" || result.IsError {
			failure, _ := json.Marshal(result)
			t.Fatalf("capture failed: %s", failure)
		}
		imageFound := false
		for _, content := range result.Content {
			if image, ok := content.(*mcp.ImageContent); ok {
				if image.MIMEType != "image/webp" || len(image.Data) < 1000 {
					t.Fatalf("invalid image: %s %d bytes", image.MIMEType, len(image.Data))
				}
				imageFound = true
			}
		}
		if !imageFound {
			t.Fatal("MCP image content missing")
		}
		return out
	}
	desktop := capture(map[string]any{"page": "main", "trace": true, "actions": []map[string]any{
		{"kind": "fill", "selector": "[data-node-id='name'] input", "value": "Capture value"},
		{"kind": "click", "selector": "[data-node-id='confirm']"},
		{"kind": "expect_text", "value": "Capture clicked"},
	}, "script": "document.querySelector('[data-node-id=name] input').value"})
	if !desktop.ChecksPassed || !strings.Contains(desktop.VisibleText, "Capture clicked") || desktop.ScriptResultJSON != `"Capture value"` || len(desktop.Trace) == 0 {
		t.Fatalf("desktop interaction failed: %#v", desktop)
	}
	mobile := capture(map[string]any{"page": "details", "mobile": true, "orientation": "landscape"})
	if !mobile.ChecksPassed || mobile.Width != 844 || mobile.Height != 390 || !strings.Contains(mobile.VisibleText, "Published details page") || strings.Contains(mobile.VisibleText, "UNPUBLISHED") {
		t.Fatalf("published mobile page failed: %#v", mobile)
	}
	navigated := capture(map[string]any{"page": "main", "actions": []map[string]any{{"kind": "click", "selector": "[data-node-id='next']"}, {"kind": "expect_text", "value": "Published details page"}}})
	if !navigated.ChecksPassed || !strings.Contains(navigated.VisibleText, "Published details page") {
		t.Fatalf("navigation failed: %#v", navigated)
	}
	diagnostics := capture(map[string]any{"page": "main", "actions": []map[string]any{{"kind": "expect_text", "value": "definitely missing text"}}})
	if diagnostics.ChecksPassed || len(diagnostics.Diagnostics.ActionErrors) == 0 {
		t.Fatalf("failed assertion was hidden: %#v", diagnostics)
	}
	// Rich metadata and page selection must also work through the public read tool.
	detail := scopeCall(t, client, "whodb_platform_app", map[string]any{"id": app})
	data := detail["data"].(map[string]any)
	if data["offlineCapable"] != true {
		t.Fatalf("offline capability omitted: %#v", detail)
	}
	page := scopeCall(t, client, "whodb_platform_app_view", map[string]any{"id": app, "page": "details"})
	encoded, _ := json.Marshal(page)
	if !strings.Contains(string(encoded), "Published details page") || strings.Contains(string(encoded), "UNPUBLISHED") {
		t.Fatalf("published page targeting failed: %s", encoded)
	}
}
