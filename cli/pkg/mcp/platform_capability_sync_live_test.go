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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

func TestPlatformCapabilities_RealStack(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(t.TempDir(), "tokens"))
	config.ResetPathsForTesting()
	clearPendingPlatformActions(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	host := liveEnvOrDefault("WHODB_PLATFORM_E2E_HOST", defaultLivePlatformE2EHost)
	parsedHost, err := url.Parse(host)
	if err != nil || (parsedHost.Hostname() != "localhost" && parsedHost.Hostname() != "127.0.0.1" && parsedHost.Hostname() != "::1") {
		t.Fatal("capability fixtures require a loopback-only local stack")
	}

	refresh := liveMintDevRefreshToken(t, ctx, liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultLivePlatformE2EKeycloakURL), "127.0.0.1:4001", defaultLivePlatformE2EUser, defaultLivePlatformE2EPassword)
	liveSeedPlatformLogin(t, ctx, host, refresh)
	session, err := loadPlatformToolSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client := scopeTestClient(t)
	call := func(tool string, args map[string]any) map[string]any {
		t.Helper()
		out := scopeCall(t, client, "whodb_platform_"+tool, args)
		if out["error"] != nil && out["error"] != "" {
			t.Fatalf("%s: %v", tool, out["error"])
		}
		return out
	}
	for _, name := range []string{"edge_runners", "usage", "entitlements", "project_transform_runs", "agent_sessions"} {
		call(name, map[string]any{})
	}
	call("feature_access", map[string]any{"feature_key": "functions"})
	// All reads use the real protocol, selected workspace and platform authorization.
	raw := session.Client.(interface {
		ExecuteGraphQL(context.Context, string, any, any) error
	})
	gql := func(document string, vars map[string]any) map[string]any {
		t.Helper()
		var out map[string]any
		if err := raw.ExecuteGraphQL(ctx, document, vars, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	pid := session.Host.DefaultProjectID
	oid := session.Host.DefaultOrgID
	suffix := fmt.Sprint(time.Now().UnixNano())
	fn := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{Resource: "function", Payload: map[string]any{"name": "capability-" + suffix, "description": "local test", "language": "python", "entryPoint": "main", "executionMode": "ASYNC", "liveBehavior": "REQUEST", "files": []any{map[string]any{"path": "main.py", "content": "def main(input):\n return input\n"}}, "dependencies": []any{}}})
	app := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{Resource: "app", Payload: map[string]any{"name": "capability-" + suffix, "description": "local access test", "ontologyIds": []any{}, "functionIds": []any{}}})
	defer liveBestEffortGenericDelete(ctx, "app", app)
	defer liveBestEffortGenericDelete(ctx, "function", fn)
	access := call("app_access", map[string]any{"app_id": app, "limit": 1})
	if access["data"].(map[string]any)["pageSize"] != float64(1) {
		t.Fatal("page size not preserved")
	}
	call("edge_runs", map[string]any{"function_id": fn})
	call("edge_schedules", map[string]any{"function_id": fn})
	registered := gql(`mutation($projectId:ID!,$name:String!){RegisterEdgeDevice(projectId:$projectId,name:$name){runner{id} token}}`, map[string]any{"projectId": pid, "name": "capability-" + suffix})
	device := registered["RegisterEdgeDevice"].(map[string]any)
	runner := device["runner"].(map[string]any)["id"].(string)
	// Device credentials stay in this test process; pairing is never an MCP tool.
	token := device["token"].(string)
	deviceClient, err := platformapi.NewClient(host, token)
	if err != nil {
		t.Fatal(err)
	}
	deviceClient.SetWorkspaceContext(oid, pid)
	var paired any
	if err := deviceClient.ExecuteGraphQL(ctx, `mutation($p:ID!,$r:ID!){PairEdgeDevice(projectId:$p,runnerId:$r)}`, map[string]any{"p": pid, "r": runner}, &paired); err != nil {
		t.Fatal(err)
	}

	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{Resource: "function", ID: fn, Payload: map[string]any{"edgeRunnerId": runner}})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "object", Action: "promote", Payload: map[string]any{"objectId": fn, "objectType": "function", "message": "local edge test"}})
	_, denied, err := handlePlatformGenericWrite(ctx, "platform_action", PlatformGenericWriteInput{Resource: "edge_function", Action: "run", ID: fn, Payload: map[string]any{"runnerId": runner, "input": "{}"}}, "action", false)
	if err != nil || !strings.Contains(denied.Error, "execution access") {
		t.Fatalf("missing device execution consent was not denied: %q %v", denied.Error, err)
	}
	gql(`mutation($p:ID!,$r:ID!){SetMyEdgeExecutionAccess(projectId:$p,runnerId:$r,enabled:true)}`, map[string]any{"p": pid, "r": runner})
	run := liveMustGenericWriteID(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "edge_function", Action: "run", ID: fn, Payload: map[string]any{"runnerId": runner, "input": "{}"}})
	call("edge_run", map[string]any{"id": run})
	// Simulate the runner lease protocol through the real API; no physical-device claim.
	var work map[string]any
	if err := deviceClient.ExecuteGraphQL(ctx, `mutation($p:ID!,$r:ID!){ClaimEdgeWork(projectId:$p,runnerId:$r){runId leaseToken bundle input}}`, map[string]any{"p": pid, "r": runner}, &work); err != nil {
		t.Fatal(err)
	}
	leased := work["ClaimEdgeWork"].(map[string]any)
	if leased["runId"] != run {
		t.Fatalf("wrong leased run: %v", leased["runId"])
	}
	if err := deviceClient.ExecuteGraphQL(ctx, `mutation($p:ID!,$r:ID!,$id:ID!,$lease:String!){CompleteEdgeWork(projectId:$p,runnerId:$r,runId:$id,leaseToken:$lease,output:"{\"ok\":true}",logs:"local runner",error:""){id status}}`, map[string]any{"p": pid, "r": runner, "id": run, "lease": leased["leaseToken"]}, &work); err != nil {
		t.Fatal(err)
	}
	finished := call("edge_run", map[string]any{"id": run})
	if finished["data"].(map[string]any)["logs"] != "local runner" {
		t.Fatal("edge completion/logs lost")
	}
	schedule := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{Resource: "edge_schedule", Payload: map[string]any{"functionId": fn, "runnerId": runner, "input": "{}", "intervalSeconds": 3600}})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "edge_schedule", Action: "stop", ID: schedule})
	call("edge_schedules", map[string]any{"function_id": fn})
	// Seed persisted AI output only in the disposable local database, independent of AI vendors.
	dsn := liveEnvOrDefault("WHODB_CAPABILITY_TEST_DSN", "postgres://postgres:whodb@localhost:15431/whodb_platform?sslmode=disable")
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "localhost" && cfg.Host != "127.0.0.1" && cfg.Host != "::1" {
		t.Fatal("capability SQL fixtures require a loopback-only database")
	}
	cfg.Database = "org_" + oid
	db, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(ctx)
	sid := uuid.NewString()
	artifact := "artifact_" + suffix
	_, err = db.Exec(ctx, `INSERT INTO "__org_meta".agent_sessions(id,user_id,project_id,title,status) VALUES($1,$2,$3,'local automation session','running')`, sid, session.Host.AccountID, pid)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Exec(context.Background(), `DELETE FROM "__org_meta".agent_sessions WHERE id=$1`, sid)
	payload := liveJSON(t, map[string]any{"schema_version": "ontology-chat/v1", "type": "analysis_artifact", "artifact": map[string]any{"kind": "view", "artifact_id": artifact, "status": "partial", "title": "local artifact", "warnings": []string{"test completeness warning"}}})
	provenance := `{"id":"local-provenance","plan_hash":"local","sources":[],"warnings":[]}`
	_, err = db.Exec(ctx, `INSERT INTO "__org_meta".analysis_artifacts(id,org_id,project_id,user_id,session_id,kind,entity_ids,payload,provenance) VALUES($1,$2,$3,$4,$5,'view','[]',$6::jsonb,$7::jsonb)`, artifact, oid, pid, session.Host.AccountID, sid, payload, provenance)
	if err != nil {
		t.Fatal(err)
	}
	call("agent_session", map[string]any{"session_id": sid})
	call("agent_sessions", map[string]any{"limit": 1})
	metadata := call("analysis_artifact", map[string]any{"id": artifact})
	encoded, _ := json.Marshal(metadata)
	if !strings.Contains(string(encoded), "test completeness warning") {
		t.Fatal("artifact warnings lost")
	}

	ontology := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{Resource: "ontology", Payload: map[string]any{
		"apiName": "capability_" + suffix, "displayName": "Capability", "pluralDisplayName": "Capabilities", "description": "local artifact paging", "primaryKey": "id", "tableName": "capability_" + suffix, "schemaName": "public", "storageMode": "operational", "icon": "table", "color": "#3366ff",
		"properties": []any{map[string]any{"apiName": "id", "displayName": "ID", "description": "ID", "columnName": "id", "dataType": "String", "isRequired": true, "visibility": "normal", "isSearchable": true, "isSortable": true, "isEditOnly": false}}, "links": []any{},
	}})
	defer liveBestEffortGenericDelete(ctx, "ontology", ontology)
	for _, value := range []string{uuid.NewString(), uuid.NewString(), uuid.NewString()} {
		liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "ontology", Action: "add_record", ID: ontology, Payload: map[string]any{"values": []any{map[string]any{"Key": "id", "Value": value}}}})
	}
	datasetArtifact := "dataset_" + suffix
	plan := map[string]any{"schema_version": "ontology-query/v1", "request_id": "local-test", "intent": "table", "sources": []any{map[string]any{"alias": "items", "entity_id": ontology}}, "dimensions": []any{map[string]any{"id": "id", "field": "id"}}, "limit": 3}
	datasetPayload := liveJSON(t, map[string]any{"schema_version": "ontology-chat/v1", "type": "analysis_artifact", "artifact": map[string]any{"kind": "dataset", "artifact_id": datasetArtifact, "status": "complete", "title": "local rows", "columns": []any{map[string]any{"id": "id", "label": "ID", "physical_type": "string", "nullable": false}}, "warnings": []string{}}})
	datasetProvenance := liveJSON(t, map[string]any{"id": "local-paging", "plan_hash": "local", "canonical_plan": plan, "sources": []any{}, "warnings": []string{}})
	_, err = db.Exec(ctx, `INSERT INTO "__org_meta".analysis_artifacts(id,org_id,project_id,user_id,session_id,kind,entity_ids,payload,provenance) VALUES($1,$2,$3,$4,$5,'dataset',$6::jsonb,$7::jsonb,$8::jsonb)`, datasetArtifact, oid, pid, session.Host.AccountID, sid, liveJSON(t, []string{ontology}), datasetPayload, datasetProvenance)
	if err != nil {
		t.Fatal(err)
	}
	page := call("analysis_artifact_page", map[string]any{"id": datasetArtifact, "limit": 1})["data"].(map[string]any)
	if len(page["rows"].([]any)) != 1 || page["nextCursor"] == nil {
		t.Fatalf("missing first page/continuation: %#v", page)
	}
	second := call("analysis_artifact_page", map[string]any{"id": datasetArtifact, "limit": 2, "cursor": page["nextCursor"]})["data"].(map[string]any)
	if len(second["rows"].([]any)) != 2 || second["nextCursor"] != nil {
		t.Fatalf("bad final page: %#v", second)
	}

	foreignSession := uuid.NewString()
	_, err = db.Exec(ctx, `INSERT INTO "__org_meta".agent_sessions(id,user_id,project_id,title,status) VALUES($1,$2,$3,'other user','running')`, foreignSession, uuid.NewString(), pid)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Exec(context.Background(), `DELETE FROM "__org_meta".agent_sessions WHERE id=$1`, foreignSession)
	foreign := scopeCall(t, client, "whodb_platform_agent_session", map[string]any{"session_id": foreignSession})
	if foreign["error"] == nil {
		t.Fatal("another user's session was exposed")
	}
	wrongProject := scopeCall(t, client, "whodb_platform_analysis_artifact", map[string]any{"id": artifact, "workspace": map[string]any{"host": host, "org": "acme", "project": "sandbox"}})
	if wrongProject["error"] == nil {
		t.Fatal("cross-project artifact was exposed")
	}
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "agent_session", Action: "cancel", ID: sid})
	rejected := scopeCall(t, client, "whodb_platform_analysis_artifact_page", map[string]any{"id": artifact, "limit": 1})
	if rejected["error"] == nil {
		t.Fatal("view artifact must not have dataset pages")
	}
	binary := liveEnvOrDefault("WHODB_PLATFORM_E2E_CLI_BINARY", filepath.Join("..", "..", "whodb"))
	invokeCLI := func(args []string) []byte {
		t.Helper()

		command := exec.CommandContext(ctx, binary, append(args, "--host", host, "--org", "acme", "--project", "default", "--format", "json")...)
		command.Env = os.Environ()
		var stderr bytes.Buffer
		command.Stderr = &stderr
		out, err := command.Output()
		if err != nil {
			t.Fatalf("CLI %v: %v %s", args, err, stderr.String())
		}
		if !json.Valid(out) {
			t.Fatalf("CLI %v emitted invalid JSON: %s", args, out)
		}
		return out
	}
	for _, args := range [][]string{
		{"scripts", "packages", "--language", "python"}, {"scripts", "search", "--language", "python", "--query", "pandas"}, {"scripts", "package", "python", "pandas"}, {"usage", "subscription"}, {"usage", "plan-features", "free"}, {"usage", "preview-plan", "free", "monthly"},
		{"edge", "runners"}, {"edge", "runs", fn}, {"edge", "run", run}, {"edge", "schedules", fn},
		{"apps", "access", app, "--limit", "1"}, {"transforms", "history"}, {"agent-sessions", "list"}, {"agent-sessions", "get", sid},
		{"analysis", "get", artifact}, {"analysis", "rows", datasetArtifact, "--limit", "1"}, {"usage", "current"}, {"usage", "entitlements"}, {"usage", "feature", "functions"},
		{"functions", "get", fn},
	} {
		out := invokeCLI(args)
		if args[0] == "functions" && !strings.Contains(string(out), `"executionMode": "ASYNC"`) && !strings.Contains(string(out), `"executionMode":"ASYNC"`) {
			t.Fatalf("function execution metadata missing: %s", out)
		}
	}

	invokeCLI([]string{"functions", "update", fn, "--edge-runner-id", runner, "--yes"})
	invokeCLI([]string{"edge", "start", fn, "--runner-id", runner, "--input", "{}", "--yes"})
	scheduleJSON := invokeCLI([]string{"edge", "schedule", "--function-id", fn, "--runner-id", runner, "--interval-seconds", "3600", "--yes"})
	var cliSchedule struct {
		Data struct{ Data struct{ ID string } }
	}
	if err := json.Unmarshal(scheduleJSON, &cliSchedule); err != nil || cliSchedule.Data.Data.ID == "" {
		t.Fatalf("CLI schedule result: %s %v", scheduleJSON, err)
	}
	invokeCLI([]string{"edge", "stop-schedule", cliSchedule.Data.Data.ID, "--yes"})
	invokeCLI([]string{"agent-sessions", "cancel", sid, "--yes"})
	// A missing target must fail at the platform boundary rather than returning empty success.
	missing := scopeCall(t, client, "whodb_platform_agent_session", map[string]any{"session_id": uuid.NewString()})
	if missing["error"] == nil {
		t.Fatal("missing session accepted")
	}
}

// The exhaustive resource lifecycle checks discovery and denial plumbing here;
// TestPlatformCapabilities_RealStack supplies the successful device/session/artifact fixtures.
func liveMustCapabilityCatalog(t *testing.T, ctx context.Context, fx liveGapFixtures) {
	t.Helper()
	for _, c := range platformapi.CapabilityReads {
		in := PlatformCapabilityReadInput{Language: "python", Name: "pandas", Plan: "free", Interval: "monthly", TransformID: fx.transformID, ID: uuid.NewString(), FunctionID: fx.functionID, AppID: fx.appID, SessionID: uuid.NewString(), FeatureKey: "functions", Limit: 2}
		_, out, err := handlePlatformCapabilityRead(ctx, c, in, &SecurityOptions{MaxRows: 2})
		expectDenied := c.Operation == "AvailablePlans" || c.Operation == "AgentSession" || c.Operation == "AnalysisArtifact" || c.Operation == "AnalysisArtifactPage" || c.Operation == "EdgeFunctionRun"
		if err != nil || (out.Error != "") != expectDenied {
			t.Fatalf("%s output error %q; expected denial %v, err %v", c.Operation, out.Error, expectDenied, err)
		}
		liveCoverTool("whodb_platform_" + c.Tool)
	}
	for _, in := range []PlatformGenericWriteInput{
		{Resource: "edge_function", Action: "run", ID: fx.functionID, Payload: map[string]any{"runnerId": uuid.NewString(), "input": "{}"}},
		{Resource: "edge_schedule", Action: "stop", ID: uuid.NewString()},
		{Resource: "agent_session", Action: "cancel", ID: uuid.NewString()},
	} {
		liveMustGenericWriteConfirmError(t, ctx, "platform_action", "action", in)
	}
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "script", Action: "execute", Payload: map[string]any{"language": "python", "script": "return input", "dependencies": []any{}}})
	liveMustGenericWriteConfirmError(t, ctx, "platform_create", "create", PlatformGenericWriteInput{Resource: "edge_schedule", Payload: map[string]any{"functionId": fx.functionID, "runnerId": uuid.NewString(), "input": "{}", "intervalSeconds": 60}})
}
