package whodb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

// IpcConfig configures the functions-runtime IPC transport. Zero values read
// the runtime's WHODB_IPC_ADDRESS / WHODB_JOB_ID / WHODB_IPC_TOKEN env vars.
type IpcConfig struct {
	Address string
	JobID   string
	Token   string
}

// IpcTransport runs the same facades inside the WhoDB Functions runtime over
// the in-container IPC server (unix socket in Docker, TCP in K8s). Only the
// ontology operations are available; others return ErrTransportCapability.
// GraphQL entity IDs resolve to apiNames via a cached /entities call.
type IpcTransport struct {
	address string
	jobID   string
	token   string
	client  *http.Client

	mu       sync.Mutex
	entities []map[string]any
}

// NewIpcTransport creates the IPC transport. WhoDB clients construct it
// automatically inside the Functions runtime; explicit use is for tests.
func NewIpcTransport(config IpcConfig) *IpcTransport {
	transport := &IpcTransport{
		address: firstNonEmpty(config.Address, os.Getenv("WHODB_IPC_ADDRESS")),
		jobID:   firstNonEmpty(config.JobID, os.Getenv("WHODB_JOB_ID")),
		token:   firstNonEmpty(config.Token, os.Getenv("WHODB_IPC_TOKEN")),
	}
	if strings.HasPrefix(transport.address, "/") {
		socketPath := transport.address
		transport.client = &http.Client{Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		}}
	} else {
		transport.client = http.DefaultClient
	}
	return transport
}

func (t *IpcTransport) baseURL() string {
	if strings.HasPrefix(t.address, "/") {
		return "http://whodb-ipc"
	}
	return "http://" + t.address
}

func (t *IpcTransport) post(ctx context.Context, path string, body any) (any, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL()+path, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Job-ID", t.jobID)
	request.Header.Set("Authorization", t.token)
	response, err := t.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, &PlatformError{Message: fmt.Sprintf("IPC request %s failed with HTTP %d", path, response.StatusCode), Code: fmt.Sprintf("IPC_%d", response.StatusCode)}
	}
	var result any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, &PlatformError{Message: "IPC request " + path + " returned invalid JSON", Code: "IPC_INVALID_JSON"}
	}
	return result, nil
}

func (t *IpcTransport) entityList(ctx context.Context) ([]map[string]any, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.entities != nil {
		return t.entities, nil
	}
	raw, err := t.post(ctx, "/entities", map[string]any{})
	if err != nil {
		return nil, err
	}
	entries, _ := raw.([]any)
	entities := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		entity, _ := entry.(map[string]any)
		entities = append(entities, entity)
	}
	t.entities = entities
	return entities, nil
}

func (t *IpcTransport) entityField(ctx context.Context, entityID, field string) (string, error) {
	entities, err := t.entityList(ctx)
	if err != nil {
		return "", err
	}
	for _, entity := range entities {
		if entity["id"] == entityID {
			value, _ := entity[field].(string)
			return value, nil
		}
	}
	return "", fmt.Errorf("%w: ontology entity %s not found in this function's scope", ErrNotFound, entityID)
}

func recordInputsToData(values any) map[string]any {
	data := map[string]any{}
	entries, _ := values.([]any)
	for _, entry := range entries {
		record, _ := entry.(map[string]any)
		key, _ := record["Key"].(string)
		data[key] = record["Value"]
	}
	// Also accept the typed shape the facades build directly.
	if typed, ok := values.([]map[string]string); ok {
		for _, record := range typed {
			data[record["Key"]] = record["Value"]
		}
	}
	return data
}

// Execute maps supported operations onto IPC endpoints, reshaping results
// into the GraphQL data shape the generated core expects.
func (t *IpcTransport) Execute(ctx context.Context, operation, _ string, variables map[string]any) (map[string]any, error) {
	result, err := t.dispatch(ctx, operation, variables)
	if err != nil {
		return nil, err
	}
	return map[string]any{operation: result}, nil
}

func (t *IpcTransport) dispatch(ctx context.Context, operation string, variables map[string]any) (any, error) {
	switch operation {
	case "OntologyEntities":
		entities, err := t.entityList(ctx)
		if err != nil {
			return nil, err
		}
		generic := make([]any, 0, len(entities))
		for _, entity := range entities {
			generic = append(generic, entity)
		}
		return generic, nil
	case "OntologyQuery":
		input, _ := variables["input"].(map[string]any)
		body := map[string]any{}
		for key, value := range input {
			if value == nil {
				continue
			}
			body[key] = value
		}
		if whereJSON, ok := body["whereJson"].(string); ok {
			delete(body, "whereJson")
			var where map[string]any
			if err := json.Unmarshal([]byte(whereJSON), &where); err == nil {
				body["where"] = where
			}
		}
		return t.post(ctx, "/query", body)
	case "OntologyDescribe":
		input, _ := variables["input"].(map[string]any)
		return t.post(ctx, "/describe", input)
	case "OntologyAddRow":
		entity, err := t.entityField(ctx, fmt.Sprint(variables["entityId"]), "apiName")
		if err != nil {
			return nil, err
		}
		if _, err := t.post(ctx, "/create", map[string]any{"entity": entity, "data": recordInputsToData(variables["values"])}); err != nil {
			return nil, err
		}
		return map[string]any{"Status": true}, nil
	case "OntologyAddRows":
		entity, err := t.entityField(ctx, fmt.Sprint(variables["entityId"]), "apiName")
		if err != nil {
			return nil, err
		}
		rows := []map[string]any{}
		entries, _ := variables["rows"].([]map[string]any)
		for _, entry := range entries {
			rows = append(rows, recordInputsToData(entry["values"]))
		}
		ids, err := t.post(ctx, "/create_many", map[string]any{"entity": entity, "rows": rows, "idempotencyKey": variables["idempotencyKey"]})
		if err != nil {
			return nil, err
		}
		idList, _ := ids.([]any)
		return map[string]any{"inserted": len(idList), "ids": idList}, nil
	case "OntologyUpdateRow":
		entityID := fmt.Sprint(variables["entityId"])
		entity, err := t.entityField(ctx, entityID, "apiName")
		if err != nil {
			return nil, err
		}
		pkKey, _ := t.entityField(ctx, entityID, "primaryKey")
		data := recordInputsToData(variables["values"])
		pk := ""
		if pkKey != "" {
			pk = fmt.Sprint(data[pkKey])
			delete(data, pkKey)
		}
		if _, err := t.post(ctx, "/update", map[string]any{"entity": entity, "pk": pk, "data": data}); err != nil {
			return nil, err
		}
		return map[string]any{"Status": true}, nil
	case "OntologyDeleteRow":
		entityID := fmt.Sprint(variables["entityId"])
		entity, err := t.entityField(ctx, entityID, "apiName")
		if err != nil {
			return nil, err
		}
		pkKey, _ := t.entityField(ctx, entityID, "primaryKey")
		data := recordInputsToData(variables["values"])
		pk := fmt.Sprint(data[pkKey])
		if _, err := t.post(ctx, "/delete", map[string]any{"entity": entity, "pk": pk}); err != nil {
			return nil, err
		}
		return map[string]any{"Status": true}, nil
	case "OntologyFollowLink":
		entity, err := t.entityField(ctx, fmt.Sprint(variables["entityId"]), "apiName")
		if err != nil {
			return nil, err
		}
		return t.post(ctx, "/follow_link", map[string]any{
			"entity":   entity,
			"pk":       variables["pk"],
			"link":     variables["linkApiName"],
			"pageSize": variables["pageSize"],
			"offset":   variables["pageOffset"],
		})
	default:
		return nil, fmt.Errorf("%w: %s is not available inside the function runtime in v1", ErrTransportCapability, operation)
	}
}
