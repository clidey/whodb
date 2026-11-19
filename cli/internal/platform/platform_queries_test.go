package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2"
)

func TestPlatformQuerySpecsAreBoundedAndScoped(t *testing.T) {
	for operation, query := range platformQuerySpecs {
		if !strings.Contains(query, operation) {
			t.Errorf("%s query does not select its operation", operation)
		}
		if strings.Contains(query, "IntrospectionQuery") || strings.Contains(query, "__schema") {
			t.Errorf("%s query exposes GraphQL introspection", operation)
		}
	}
}

func TestPlatformAppViewMetadataAndPage(t *testing.T) {
	schema := loadPlatformSchema(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Query     string
			Variables map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if _, err := gqlparser.LoadQuery(schema, request.Query); err != nil {
			t.Error(err)
			return
		}
		if request.Variables["page"] != "details" || !strings.Contains(request.Query, "page: $page") {
			t.Errorf("page was not forwarded: %#v", request)
		}
		for _, field := range []string{"offlineCapable", "recoveryBindings", "sourceHidden"} {
			if !strings.Contains(request.Query, field) {
				t.Errorf("missing %s", field)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"AppView":{"pages":["details"],"app":{"id":"app-1","offlineCapable":true,"recoveryBindings":{"source":"old-id"}},"files":[]}}}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.PlatformQuery(context.Background(), "AppView", map[string]any{"projectId": "project-1", "id": "app-1", "page": "details"})
	if err != nil {
		t.Fatal(err)
	}
	app := result.(map[string]any)["app"].(map[string]any)
	if app["offlineCapable"] != true || app["recoveryBindings"].(map[string]any)["source"] != "old-id" {
		t.Fatalf("metadata discarded: %#v", result)
	}
}

func TestPlatformQueryRejectsUnknownOperation(t *testing.T) {
	client := &Client{}
	if _, err := client.PlatformQuery(context.Background(), "UnknownOperation", nil); err == nil {
		t.Fatal("PlatformQuery() accepted an unknown operation")
	}
}
