//go:build e2e_platform

package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
)

func TestPlatformMCP_RealIdempotencyAndBoundedPages(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(t.TempDir(), "tokens"))
	config.ResetPathsForTesting()
	clearPendingPlatformActions(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	host := liveEnvOrDefault("WHODB_PLATFORM_E2E_HOST", defaultLivePlatformE2EHost)
	refresh := liveMintDevRefreshToken(t, ctx, liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultLivePlatformE2EKeycloakURL), liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_HOST_HEADER", "127.0.0.1:4001"), defaultLivePlatformE2EUser, defaultLivePlatformE2EPassword)
	liveSeedPlatformLogin(t, ctx, host, refresh)
	prefix := "audit-pages-" + time.Now().Format("150405.000000")
	input := PlatformGenericWriteInput{Resource: "function", IdempotencyKey: prefix, PayloadJSON: liveJSON(t, map[string]any{"name": prefix, "language": "python", "dependencies": []any{}, "description": "audit fixture", "entryPoint": "main.run", "files": []any{map[string]any{"path": "main.py", "content": "def run(): return 1"}}})}
	_, preview, err := handlePlatformGenericWrite(ctx, "platform_create", input, "create", true)
	if err != nil || preview.Error != "" {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	_, same, err := handlePlatformGenericWrite(ctx, "platform_create", input, "create", true)
	if err != nil || same.ConfirmationToken != preview.ConfirmationToken {
		t.Fatalf("pending replay: %+v %v", same, err)
	}
	changed := input
	changed.PayloadJSON = strings.Replace(input.PayloadJSON, prefix, prefix+"-different", 1)
	_, conflict, _ := handlePlatformGenericWrite(ctx, "platform_create", changed, "create", true)
	if conflict.ErrorCode != "conflict" {
		t.Fatalf("payload conflict: %+v", conflict)
	}
	liveMustConfirm(t, ctx, preview.ConfirmationToken)
	_, replayed, err := handlePlatformGenericWrite(ctx, "platform_create", input, "create", true)
	if err != nil || !replayed.IdempotencyReplayed || !strings.Contains(replayed.ResultJSON, "id") {
		t.Fatalf("completed replay: %+v %v", replayed, err)
	}
	// Every created object is owned by this isolated test and cleaned up below.
	ids := []string{}
	defer func() {
		for _, id := range ids {
			liveBestEffortGenericDelete(context.Background(), "function", id)
		}
	}()
	_, initial, _ := HandlePlatformFunctions(ctx, nil, PlatformEmptyInput{Search: prefix, Limit: 100})
	for _, item := range initial.Items {
		if id, ok := item["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	for i := range 8 {
		id := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{Resource: "function", PayloadJSON: liveJSON(t, map[string]any{"name": fmt.Sprintf("%s-%02d", prefix, i), "language": "python", "dependencies": []any{}, "description": "audit fixture", "entryPoint": "main.run", "files": []any{map[string]any{"path": "main.py", "content": "def run(): return 1\n" + strings.Repeat("# source\n", 75000)}}})})
		ids = append(ids, id)
	}
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Client.Functions(ctx, session.Host.DefaultProjectID, nil); err == nil {
		t.Fatal("expected legacy full collection to exceed the response cap")
	}
	_, invalidPage, _ := HandlePlatformFunctions(ctx, nil, PlatformEmptyInput{Limit: 101})
	if invalidPage.ErrorCode != "invalid_input" || invalidPage.Retryable {
		t.Fatalf("GraphQL validation category lost: %+v", invalidPage)
	}
	_, page, err := HandlePlatformFunctions(ctx, nil, PlatformEmptyInput{Search: prefix, Limit: 2, Offset: 1, Fields: []string{"id", "name"}})
	if err != nil || page.Error != "" || page.Count != 2 || !page.Truncated {
		t.Fatalf("bounded page: %+v %v", page, err)
	}
	_, detail, err := HandlePlatformFunctions(ctx, nil, PlatformEmptyInput{Search: prefix + "-00", Limit: 1, Fields: []string{"id", "files"}})
	if err != nil || detail.Error != "" || detail.Count != 1 || !detail.Truncated {
		t.Fatalf("bounded explicit detail: %+v %v", detail, err)
	}
	for _, read := range []func(context.Context, *mcp.CallToolRequest, PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error){HandlePlatformSecrets, HandlePlatformAIProviders, HandlePlatformOntologies, HandlePlatformDatasets, HandlePlatformTransforms} {
		_, out, err := read(ctx, nil, PlatformEmptyInput{Limit: 1})
		if err != nil || out.Error != "" || out.Count > 1 {
			t.Fatalf("collection page: %+v %v", out, err)
		}
	}
}
