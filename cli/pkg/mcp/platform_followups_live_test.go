//go:build e2e_platform

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestPlatformFollowups_RealStack(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(t.TempDir(), "tokens"))
	config.ResetPathsForTesting()
	clearPendingPlatformActions(t)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	host := liveEnvOrDefault("WHODB_PLATFORM_E2E_HOST", defaultLivePlatformE2EHost)
	parsed, err := url.Parse(host)
	if err != nil || (parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1") {
		t.Fatal("followup fixtures require a loopback local stack")
	}
	refresh := liveMintDevRefreshToken(t, ctx, liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultLivePlatformE2EKeycloakURL), "127.0.0.1:4001", defaultLivePlatformE2EUser, defaultLivePlatformE2EPassword)
	liveSeedPlatformLogin(t, ctx, host, refresh)
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client := scopeTestClient(t)
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		out := scopeCall(t, client, "whodb_platform_"+name, args)
		if out["error"] != nil && out["error"] != "" {
			t.Fatalf("%s: %v", name, out["error"])
		}
		return out
	}
	call("subscription", nil)
	call("plan_features", map[string]any{"plan": "free"})
	call("subscription_preview", map[string]any{"plan": "free", "interval": "monthly"})
	unavailable := scopeCall(t, client, "whodb_platform_plans", nil)
	if message, _ := unavailable["error"].(string); !strings.Contains(message, "billing is not configured") {
		t.Fatalf("unconfigured billing was not explicit: %#v", unavailable)
	}
	catalog := call("script_packages", map[string]any{"language": "python", "limit": 1})
	if catalog["count"] != float64(1) || catalog["truncated"] != true {
		t.Fatalf("script catalog bounds lost: %#v", catalog)
	}
	call("script_package_search", map[string]any{"language": "python", "query": "pandas", "limit": 2})
	pkg := call("script_package", map[string]any{"language": "python", "name": "pandas"})
	if pkg["data"].(map[string]any)["name"] != "pandas" {
		t.Fatalf("wrong package: %#v", pkg)
	}
	sample := `{"columns":["value"],"rows":[["custom-sample"],["second-row"]]}`
	script := func(source string) map[string]any {
		t.Helper()
		result := liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "script", Action: "execute", Payload: map[string]any{"language": "python", "script": source, "dependencies": []any{}, "sampleData": sample}})
		var data map[string]any
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	success := script("return input")
	if success["success"] != true || success["inputRowCount"] != float64(2) || success["outputRowCount"] != float64(2) || !strings.Contains(fmt.Sprint(success["output"]), "custom-sample") || strings.Contains(fmt.Sprint(success["runtime"]), "demo") {
		t.Fatalf("real script did not use supplied rows: %#v", success)
	}
	failed := script("raise RuntimeError('expected-preview-failure')")
	if failed["success"] != false || !strings.Contains(fmt.Sprint(failed["error"]), "expected-preview-failure") || failed["outputRowCount"] != float64(0) {
		t.Fatalf("script failure was hidden: %#v", failed)
	}
	transform := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{Resource: "transform", Payload: map[string]any{"name": fmt.Sprint("history-", time.Now().UnixNano()), "description": "lookup regression", "graphJson": `{"nodes":[],"edges":[]}`, "scheduleCron": "", "triggerMode": "manual"}})
	defer liveBestEffortGenericDelete(ctx, "transform", transform)
	cfg, err := pgx.ParseConfig(liveEnvOrDefault("WHODB_CAPABILITY_TEST_DSN", "postgres://postgres:whodb@localhost:15431/whodb_platform?sslmode=disable"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "localhost" && cfg.Host != "127.0.0.1" && cfg.Host != "::1" {
		t.Fatal("SQL fixtures must be local")
	}
	cfg.Database = "org_" + session.Host.DefaultOrgID
	db, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(ctx)
	table := pgx.Identifier{"project_" + session.Host.DefaultProjectID, "transform_runs"}.Sanitize()
	oldRun := uuid.NewString()
	if _, err := db.Exec(ctx, `INSERT INTO `+table+`(id,transform_id,status,created_at) VALUES($1,$2,'success',NOW()-INTERVAL '1 day')`, oldRun, transform); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO `+table+`(id,transform_id,status,created_at) SELECT gen_random_uuid(),$1,'success',NOW() FROM generate_series(1,105)`, transform); err != nil {
		t.Fatal(err)
	}
	old := call("transform_run", map[string]any{"transform_id": transform, "id": oldRun})
	if old["data"].(map[string]any)["id"] != oldRun {
		t.Fatalf("old run lost: %#v", old)
	}
	waited := call("transform_wait", map[string]any{"transform_id": transform, "run_id": oldRun, "timeout_seconds": 2, "poll_seconds": 1})
	if waited["data"].(map[string]any)["id"] != oldRun {
		t.Fatalf("wait did not find old run: %#v", waited)
	}
	absent := scopeCall(t, client, "whodb_platform_transform_wait", map[string]any{"transform_id": transform, "run_id": uuid.NewString(), "timeout_seconds": 2})
	if absent["error"] == nil {
		t.Fatal("missing run was silently waited on")
	}
	// The CLI subprocess exercises discovery, argument forwarding and JSON output.
	binary := liveEnvOrDefault("WHODB_PLATFORM_E2E_CLI_BINARY", filepath.Join("..", "..", "whodb"))
	file := filepath.Join(t.TempDir(), "preview.py")
	if err := os.WriteFile(file, []byte("return input\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"transforms", "get-run", transform, oldRun}, {"scripts", "packages", "--language", "python"}, {"scripts", "search", "--language", "python", "--query", "pandas"}, {"scripts", "package", "python", "pandas"}, {"scripts", "run", "--file", file, "--sample-data", sample, "--yes"}, {"usage", "subscription"}, {"usage", "plan-features", "free"}, {"usage", "preview-plan", "free", "monthly"}} {
		command := exec.CommandContext(ctx, binary, append(args, "--host", host, "--org", "acme", "--project", "default", "--format", "json")...)
		command.Env = os.Environ()
		var stderr bytes.Buffer
		command.Stderr = &stderr
		out, err := command.Output()
		if err != nil || !json.Valid(out) {
			t.Fatalf("CLI %v: %v %s %s", args, err, stderr.String(), out)
		}
		if len(args) > 1 && args[0] == "scripts" && args[1] == "run" && !bytes.Contains(out, []byte("custom-sample")) {
			t.Fatalf("CLI sample data lost: %s", out)
		}
	}
	// A viewer cannot execute script previews or read private transform runs.
	viewerRefresh := liveMintDevRefreshToken(t, ctx, liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultLivePlatformE2EKeycloakURL), "127.0.0.1:4001", "viewer@acme.test", defaultLivePlatformE2EPassword)
	tokens, err := platformapi.RefreshToken(ctx, host, viewerRefresh)
	if err != nil {
		t.Fatal(err)
	}
	viewer, err := platformapi.NewClient(host, tokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	viewer.SetWorkspaceContext(session.Host.DefaultOrgID, session.Host.DefaultProjectID)
	if _, err := viewer.TransformRun(ctx, session.Host.DefaultProjectID, transform, oldRun); err == nil {
		t.Fatal("private transform run exposed to viewer")
	}
	if _, err := viewer.PlatformMutation(ctx, "ExecuteScript", map[string]any{"input": map[string]any{"projectId": session.Host.DefaultProjectID, "language": "python", "script": "return input", "dependencies": []any{}}}); err == nil {
		t.Fatal("viewer executed script")
	}
}
