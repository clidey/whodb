package whodb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ipcStub is a fake functions-runtime IPC server recording request bodies.
type ipcStub struct {
	mu     sync.Mutex
	calls  []string // "path body" per request
	server *httptest.Server
}

func newIpcStub(t *testing.T) *ipcStub {
	t.Helper()
	s := &ipcStub{}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.mu.Lock()
		s.calls = append(s.calls, r.URL.Path+" "+string(body))
		s.mu.Unlock()
		if r.Header.Get("Authorization") != "ipc-token" || r.Header.Get("X-Job-ID") != "job-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/entities":
			_, _ = w.Write([]byte(`[{"id":"ent-1","apiName":"User","primaryKey":"id"}]`))
		case "/query":
			_, _ = w.Write([]byte(`{"columns":["id"],"rows":[["u1"]],"total":1}`))
		case "/create", "/update", "/delete":
			_, _ = w.Write([]byte(`{}`))
		case "/create_many":
			_, _ = w.Write([]byte(`["id-1","id-2"]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *ipcStub) transport() *IpcTransport {
	address := strings.TrimPrefix(s.server.URL, "http://")
	return NewIpcTransport(IpcConfig{Address: address, JobID: "job-1", Token: "ipc-token"})
}

func (s *ipcStub) call(index int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[index]
}

func TestIpcQueryConvertsWhereJSON(t *testing.T) {
	stub := newIpcStub(t)
	data, err := stub.transport().Execute(context.Background(), "OntologyQuery", "", map[string]any{
		"input": map[string]any{"entity": "User", "whereJson": `{"id":{"eq":"u1"}}`, "pageSize": 1, "sort": nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := data["OntologyQuery"].(map[string]any); !ok {
		t.Fatalf("result shape: %v", data)
	}
	call := stub.call(0)
	if !strings.HasPrefix(call, "/query ") {
		t.Fatalf("endpoint: %s", call)
	}
	if strings.Contains(call, "whereJson") || !strings.Contains(call, `"where":{"id":{"eq":"u1"}}`) {
		t.Errorf("whereJson must convert to a where object: %s", call)
	}
	if strings.Contains(call, `"sort"`) {
		t.Errorf("nil inputs must be omitted: %s", call)
	}
}

func TestIpcAddRowResolvesEntityAndConvertsRecordInputs(t *testing.T) {
	stub := newIpcStub(t)
	data, err := stub.transport().Execute(context.Background(), "OntologyAddRow", "", map[string]any{
		"entityId": "ent-1",
		"values":   []map[string]string{{"Key": "name", "Value": "Ada"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, _ := data["OntologyAddRow"].(map[string]any)
	if result["Status"] != true {
		t.Errorf("AddRow result: %v", data)
	}
	create := stub.call(1) // call 0 is /entities
	if !strings.HasPrefix(create, "/create ") || !strings.Contains(create, `"entity":"User"`) || !strings.Contains(create, `"name":"Ada"`) {
		t.Errorf("create call: %s", create)
	}
}

func TestIpcUpdateRowSplitsPrimaryKey(t *testing.T) {
	stub := newIpcStub(t)
	_, err := stub.transport().Execute(context.Background(), "OntologyUpdateRow", "", map[string]any{
		"entityId": "ent-1",
		"values":   []map[string]string{{"Key": "id", "Value": "u1"}, {"Key": "plan", "Value": "pro"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	update := stub.call(1)
	if !strings.Contains(update, `"pk":"u1"`) || strings.Contains(update, `"id":"u1"`) {
		t.Errorf("primary key must move from data to pk: %s", update)
	}
	if !strings.Contains(update, `"plan":"pro"`) {
		t.Errorf("non-pk fields must stay in data: %s", update)
	}
}

func TestIpcCreateManyReturnsInsertedCount(t *testing.T) {
	stub := newIpcStub(t)
	data, err := stub.transport().Execute(context.Background(), "OntologyAddRows", "", map[string]any{
		"entityId":       "ent-1",
		"rows":           []map[string]any{{"values": []map[string]string{{"Key": "name", "Value": "Ada"}}}},
		"idempotencyKey": "batch-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, _ := data["OntologyAddRows"].(map[string]any)
	if result["inserted"] != 2 {
		t.Errorf("inserted count: %v", result)
	}
	if !strings.Contains(stub.call(1), `"idempotencyKey":"batch-1"`) {
		t.Errorf("idempotency key must forward: %s", stub.call(1))
	}
}

func TestIpcUnknownEntityIsNotFound(t *testing.T) {
	stub := newIpcStub(t)
	_, err := stub.transport().Execute(context.Background(), "OntologyAddRow", "", map[string]any{
		"entityId": "no-such-entity",
		"values":   []map[string]string{},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown entity must map to ErrNotFound, got %v", err)
	}
}

func TestIpcUnsupportedOperationIsCapabilityError(t *testing.T) {
	stub := newIpcStub(t)
	_, err := stub.transport().Execute(context.Background(), "RunSourceQuery", "", nil)
	if !errors.Is(err, ErrTransportCapability) {
		t.Errorf("unmapped ops must map to ErrTransportCapability, got %v", err)
	}
}
