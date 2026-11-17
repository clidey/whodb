package platform

import (
	"context"
	"encoding/json"
	"errors"
	coreaudit "github.com/clidey/whodb/core/src/audit"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGraphQLMultipartBoundaryPrecisionAndWorkspace(t *testing.T) {
	const query = "mutation Save($payload:String!,$number:JSON!){ok}"
	for _, tc := range []struct {
		name string
		size int
	}{{"below", 1<<20 - 1}, {"exact", 1 << 20}, {"above", 1<<20 + 1}} {
		t.Run(tc.name, func(t *testing.T) {
			vars := map[string]any{"payload": "", "number": json.Number("9007199254740993")}
			empty, err := json.Marshal(map[string]any{"query": query, "variables": vars})
			if err != nil {
				t.Fatal(err)
			}
			vars["payload"] = strings.Repeat("x", tc.size-len(empty))
			expected, err := json.Marshal(map[string]any{"query": query, "variables": vars})
			if err != nil {
				t.Fatal(err)
			}
			if len(expected) != tc.size {
				t.Fatal("test envelope size", len(expected), tc.size)
			}
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != defaultPath || r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get(workspaceOrgHeader) != "org" || r.Header.Get(workspaceProjectHeader) != "project" || r.Header.Get(coreaudit.ClientHeader) == "" {
					t.Error("request identity, path or workspace changed")
				}
				var encoded []byte
				var readErr error
				if tc.size <= 1<<20 {
					if r.Header.Get("Content-Type") != "application/json" {
						t.Error("JSON boundary used multipart")
					}
					encoded, readErr = io.ReadAll(r.Body)
				} else {
					if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
						t.Error("oversized request did not select multipart")
						http.Error(w, "wrong transport", 400)
						return
					}
					reader, e := r.MultipartReader()
					if e != nil {
						t.Error(e)
						http.Error(w, "invalid multipart", 400)
						return
					}
					part, e := reader.NextPart()
					if e != nil || part.FormName() != "operations" || part.FileName() != "" {
						t.Error("operations must be the first ordinary field")
						http.Error(w, "invalid operations", 400)
						return
					}
					encoded, readErr = io.ReadAll(part)
					part, e = reader.NextPart()
					if e != nil || part.FormName() != "map" || part.FileName() != "" {
						t.Error("map must be the second ordinary field")
						http.Error(w, "invalid map", 400)
						return
					}
					mapping, e := io.ReadAll(part)
					if e != nil || string(mapping) != "{}" {
						t.Error("zero-file map changed")
					}
					if _, e = reader.NextPart(); e != io.EOF {
						t.Error("unexpected file or third field", e)
					}
				}
				if readErr != nil || string(encoded) != string(expected) {
					t.Error("GraphQL envelope changed", readErr)
				}
				decoder := json.NewDecoder(strings.NewReader(string(encoded)))
				decoder.UseNumber()
				var decoded struct {
					Variables map[string]any `json:"variables"`
				}
				if e := decoder.Decode(&decoded); e != nil || decoded.Variables["number"] != json.Number("9007199254740993") {
					t.Error("exact number lost", e)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"data":{"ok":true}}`)
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "test-token")
			if err != nil {
				t.Fatal(err)
			}
			client.workspaceOrgID = "org"
			client.workspaceProjectID = "project"
			var target struct {
				OK bool `json:"ok"`
			}
			if err = client.ExecuteGraphQL(context.Background(), query, vars, &target); err != nil || !target.OK {
				t.Fatal(target, err)
			}
			if calls.Load() != 1 {
				t.Fatal("mutation was sent more than once", calls.Load())
			}
		})
	}
}

func TestGraphQLMultipartDoesNotRetryMutationFailureOrIgnoreCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "injected failure", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string]any{"payload": strings.Repeat("x", 1<<20)}
	if err = client.ExecuteGraphQL(context.Background(), "mutation Save($payload:String!){ok}", vars, new(any)); err == nil {
		t.Fatal("server failure was ignored")
	}
	if calls.Load() != 1 {
		t.Fatal("failed mutation was retried", calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = client.ExecuteGraphQL(ctx, "mutation Save($payload:String!){ok}", vars, new(any)); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
	if calls.Load() != 1 {
		t.Fatal("cancelled request reached the server", calls.Load())
	}
}
