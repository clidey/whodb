//go:build e2e_platform

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/platform"
)

// TestPlatformCLI_RestrictedAutomation proves the CLI can operate without a
// personal login, honors viewer grants and immediately rejects a revoked key.
func TestPlatformCLI_RestrictedAutomation(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	config.ResetPathsForTesting()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	host := envOrDefault("WHODB_PLATFORM_E2E_HOST", defaultPlatformE2EHost)
	refresh := mintDevRefreshToken(t, ctx, envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultPlatformE2EKeycloakURL), envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_HOST_HEADER", "127.0.0.1:4001"), defaultPlatformE2EUser, defaultPlatformE2EPassword)
	tokens, err := platform.RefreshToken(ctx, host, refresh)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := platform.NewClient(host, tokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	org := cloneE2EOrgID(t, owner, "acme")
	projects, err := owner.Projects(ctx, org)
	if err != nil || len(projects) == 0 {
		t.Fatal("no fixture project", err)
	}
	project := projects[0].ID
	owner.SetWorkspaceContext(org, project)
	query := func(doc string, variables map[string]any) (map[string]json.RawMessage, error) {
		return automationGraphQL(ctx, host, tokens.AccessToken, org, project, doc, variables)
	}
	data, err := query(`mutation($org:ID!){CreateServiceAccount(orgId:$org,name:"acceptance-viewer"){id}}`, map[string]any{"org": org})
	if err != nil {
		t.Fatal(err)
	}
	var account struct{ ID string }
	if err := json.Unmarshal(data["CreateServiceAccount"], &account); err != nil || account.ID == "" {
		t.Fatal("missing service account")
	}
	defer func() {
		if _, err := query(`mutation($org:ID!,$id:ID!){DeleteServiceAccount(orgId:$org,id:$id){Status}}`, map[string]any{"org": org, "id": account.ID}); err != nil {
			t.Error("service-account cleanup failed", err)
		}
	}()
	_, err = owner.PlatformMutation(ctx, "GrantAccess", map[string]any{"input": map[string]any{"resourceType": "project", "resourceId": project, "subject": "service_account:" + account.ID, "relation": "viewer"}})
	if err != nil {
		t.Fatal(err)
	}
	source, err := owner.CreateSource(ctx, platform.CreateSourceInput{OrgID: org, ProjectID: project, Name: "acceptance-source-" + account.ID,
		DatabaseType: "Postgres", Hostname: envOrDefault("WHODB_PLATFORM_E2E_SOURCE_HOST", "platform-db"), Port: "5432", Username: "postgres", Password: "whodb", Database: "whodb_platform"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := owner.DeleteSource(ctx, org, project, source.ID); err != nil {
			t.Error("source cleanup failed", err)
		}
	}()
	_, err = owner.PlatformMutation(ctx, "GrantAccess", map[string]any{"input": map[string]any{"resourceType": "source", "resourceId": source.ID, "subject": "service_account:" + account.ID, "relation": "viewer"}})
	if err != nil {
		t.Fatal(err)
	}
	data, err = query(`mutation($org:ID!,$id:ID!){CreateServiceAccountKey(orgId:$org,serviceAccountId:$id){rawKey key{id}}}`, map[string]any{"org": org, "id": account.ID})
	if err != nil {
		t.Fatal(err)
	}
	var key struct {
		RawKey string
		Key    struct{ ID string }
	}
	if err := json.Unmarshal(data["CreateServiceAccountKey"], &key); err != nil || key.RawKey == "" {
		t.Fatal("missing API key")
	}
	t.Setenv(platform.APIKeyEnv, key.RawKey)
	t.Setenv(platform.SessionHostEnv, host)
	t.Setenv(platform.SessionOrgEnv, org)
	t.Setenv(platform.SessionProjectEnv, project)
	// No OIDC tokens or account entries have been saved in this test HOME.
	status := runJSONCommand[map[string]any](t, "status", "--format", "json", "--quiet")
	if status["authentication"] != "api_key" || status["account_id"] != account.ID {
		t.Fatal("automation status lost principal")
	}
	_ = runJSONCommand[map[string]any](t, "whoami", "--format", "json", "--quiet")
	sources := runJSONCommand[[]platform.Source](t, "sources", "list", "--format", "json", "--quiet")
	requireContainsSource(t, sources, source.Name)
	_, _, exitCode := testharness.RunCLI(t, "resources", "create", "app", "--payload-json", `{"name":"must-not-exist"}`, "--yes", "--format", "json", "--quiet")
	if exitCode == 0 {
		t.Fatal("viewer API key created an app")
	}
	_, _, exitCode = testharness.RunCLI(t, "security-audit")
	if exitCode == 0 {
		t.Fatal("viewer API key read security ledger")
	}
	_, _, exitCode = testharness.RunCLI(t, "sources", "update", source.ID, "--name", "must-not-update", "--format", "json", "--quiet")
	if exitCode == 0 {
		t.Fatal("viewer API key updated a source")
	}
	_, err = owner.PlatformMutation(ctx, "GrantAccess", map[string]any{"input": map[string]any{"resourceType": "source", "resourceId": source.ID, "subject": "service_account:" + account.ID, "relation": "editor"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = runEnvelope(t, "sources", "update", source.ID, "--name", "acceptance-updated-"+account.ID, "--format", "json", "--quiet")
	// A public-looking operation name must not authorize a protected field.
	_, err = automationGraphQL(ctx, host, "", org, project, `query Version($input:SecurityAuditInput!){SecurityAudit(input:$input)}`, map[string]any{"input": map[string]any{"orgId": org}})
	if err == nil {
		t.Fatal("anonymous security ledger read succeeded")
	}
	for _, mode := range []string{"events", "exports", "verify"} {
		result, err := owner.SecurityAudit(ctx, platform.SecurityAuditInput{OrgID: org, Mode: mode, Limit: 2})
		if err != nil {
			t.Fatal(mode, err)
		}
		if result["enabled"] != true {
			t.Fatal("acceptance fixture must enable audit")
		}
		if mode == "events" {
			cursor, _ := result["nextCursor"].(string)
			if cursor == "" {
				t.Fatal("expected multiple pages of fixture audit events")
			}
			if _, err := owner.SecurityAudit(ctx, platform.SecurityAuditInput{OrgID: org, Mode: mode, Limit: 2, Cursor: cursor}); err != nil {
				t.Fatal("continue audit page", err)
			}
		}
	}
	// The audit service flushes asynchronously; prove denied ledger access is
	// retained in the canonical ledger, including the service-account actor.
	deadline := time.Now().Add(10 * time.Second)
	for {
		result, err := owner.SecurityAudit(ctx, platform.SecurityAuditInput{OrgID: org, Mode: "events", ActorID: account.ID, Action: "security_audit.read", Outcome: "DENIED"})
		if err != nil {
			t.Fatal(err)
		}
		if events, ok := result["events"].([]any); ok && len(events) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("denied security-ledger read was not recorded")
		}
		time.Sleep(200 * time.Millisecond)
	}
	_, err = query(`mutation($org:ID!,$key:ID!){RevokeServiceAccountKey(orgId:$org,keyId:$key){Status}}`, map[string]any{"org": org, "key": key.Key.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, _, exitCode = testharness.RunCLI(t, "sources", "list", "--format", "json", "--quiet")
	if exitCode == 0 {
		t.Fatal("revoked key remained usable")
	}
	// Neither the credential nor a personal session should have been persisted.
	_ = filepath.WalkDir(os.Getenv("HOME"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr == nil && !entry.IsDir() {
			raw, readErr := os.ReadFile(path)
			if readErr == nil && bytes.Contains(raw, []byte(key.RawKey)) {
				t.Error("API key persisted to disk")
			}
		}
		return nil
	})
}

func automationGraphQL(ctx context.Context, host, token, org, project, query string, variables map[string]any) (map[string]json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/api/query", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Whodb-Org-Id", org)
	req.Header.Set("X-Whodb-Project-Id", project)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var envelope struct {
		Data   map[string]json.RawMessage
		Errors []struct{ Message string }
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&envelope); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK || len(envelope.Errors) > 0 {
		return nil, &automationGraphQLError{}
	}
	return envelope.Data, nil
}

type automationGraphQLError struct{}

func (*automationGraphQLError) Error() string {
	return "acceptance GraphQL request rejected (response omitted to protect credentials)"
}
