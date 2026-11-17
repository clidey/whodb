package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/platform"
)

func TestAppSyncPlanChecksContentAndRejectsUnsafePaths(t *testing.T) {
	folder := t.TempDir()
	path := "pages/main/tree.json"
	if err := os.MkdirAll(filepath.Join(folder, "pages/main"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, path), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(path + "new" + "pages/main/scripts/old.js"))
	plan := appSyncPlan{Upserts: []string{path}, Deletes: []string{"pages/main/scripts/old.js"}, SHA256: hex.EncodeToString(sum[:])}
	if _, err := validateAppSyncPlan(folder, plan); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, path), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := validateAppSyncPlan(folder, plan); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("changed content accepted: %v", err)
	}
	plan.Deletes = []string{"../outside"}
	if _, err := validateAppSyncPlan(folder, plan); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("unsafe path accepted: %v", err)
	}
	plan.Deletes = []string{"app.json"}
	if _, err := validateAppSyncPlan(folder, plan); err == nil || !strings.Contains(err.Error(), "outside page scripts") {
		t.Fatalf("manifest deletion accepted: %v", err)
	}
}

func TestExecuteAppSyncBatchUsesAliasesAndChecksEveryResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if !strings.Contains(request.Query, "a0: DeleteAppFile") || !strings.Contains(request.Query, "a1: DeleteAppFile") {
			t.Errorf("missing aliases: %s", request.Query)
		}
		if request.Variables["path0"] != "pages/main/scripts/a.js" || request.Variables["path1"] != "pages/main/scripts/b.js" {
			t.Errorf("paths: %#v", request.Variables)
		}
		_, _ = w.Write([]byte(`{"data":{"a0":{"Status":true},"a1":{"Status":true}}}`))
	}))
	defer server.Close()
	client, err := platform.NewClient(server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	if err := executeAppSyncBatch(context.Background(), client, "project", "app", []string{"pages/main/scripts/a.js", "pages/main/scripts/b.js"}, nil, false); err != nil {
		t.Fatal(err)
	}
}
