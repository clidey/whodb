package whodb

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestMapGraphQLErrors(t *testing.T) {
	build := func(code string) []graphQLError {
		var e graphQLError
		e.Message = "x"
		e.Extensions.Code = code
		return []graphQLError{e}
	}
	if !errors.Is(mapGraphQLErrors(build("UNAUTHENTICATED")), ErrAuth) {
		t.Error("UNAUTHENTICATED must map to ErrAuth")
	}
	if !errors.Is(mapGraphQLErrors(build("FORBIDDEN")), ErrAuth) {
		t.Error("FORBIDDEN must map to ErrAuth")
	}
	if !errors.Is(mapGraphQLErrors(build("NOT_FOUND")), ErrNotFound) {
		t.Error("NOT_FOUND must map to ErrNotFound")
	}
	if !errors.Is(mapGraphQLErrors(build("BAD_USER_INPUT")), ErrValidation) {
		t.Error("BAD_USER_INPUT must map to ErrValidation")
	}
	var platformErr *PlatformError
	if !errors.As(mapGraphQLErrors(build("OTHER")), &platformErr) || platformErr.Code != "OTHER" {
		t.Error("unknown codes must map to PlatformError with the code preserved")
	}
	if mapGraphQLErrors(nil) == nil {
		t.Error("empty errors must still produce an error")
	}
}

func TestInterpretServerError(t *testing.T) {
	converted := interpretServerError(errors.New(`Cannot query field "New" on type "Query"`))
	if !errors.Is(converted, ErrVersion) {
		t.Error("unknown-operation rejections must become ErrVersion")
	}
	passthrough := errors.New("connection refused")
	if interpretServerError(passthrough) != passthrough {
		t.Error("other errors must pass through unchanged")
	}
	if interpretServerError(nil) != nil {
		t.Error("nil must pass through")
	}
}

func TestCoerceValue(t *testing.T) {
	if got := coerceValue("42", "bigint"); got != int64(42) {
		t.Errorf("int coercion: got %v (%T)", got, got)
	}
	if got := coerceValue("3.14", "numeric"); got != 3.14 {
		t.Errorf("float coercion: got %v", got)
	}
	if got := coerceValue("true", "boolean"); got != true {
		t.Errorf("bool coercion: got %v", got)
	}
	if got := coerceValue("f", "boolean"); got != false {
		t.Errorf("bool false coercion: got %v", got)
	}
	if _, ok := coerceValue("2026-01-05T00:00:00Z", "timestamptz").(time.Time); !ok {
		t.Error("timestamp coercion must produce time.Time")
	}
	var want any
	_ = json.Unmarshal([]byte(`{"a":1}`), &want)
	got := coerceValue(`{"a": 1}`, "jsonb")
	if gotJSON, _ := json.Marshal(got); string(gotJSON) != `{"a":1}` {
		t.Errorf("json coercion: got %v", got)
	}
	if got := coerceValue("hello", "text"); got != "hello" {
		t.Errorf("string passthrough: got %v", got)
	}
	if got := coerceValue(nil, "bigint"); got != nil {
		t.Errorf("nil passthrough: got %v", got)
	}
	if got := coerceValue("not-a-number", "bigint"); got != "not-a-number" {
		t.Errorf("unparseable int must pass through: got %v", got)
	}
	// Non-string values pass through (IPC delivers native types).
	if got := coerceValue(float64(7), "bigint"); got != float64(7) {
		t.Errorf("native value passthrough: got %v", got)
	}
}

func TestHydrateRowsMetadataOverride(t *testing.T) {
	result := map[string]any{"columns": []any{"age"}, "rows": []any{[]any{"42"}}, "total": float64(1)}
	rows, _ := hydrateRows(result, map[string]string{"age": "Integer"})
	if rows[0]["age"] != int64(42) {
		t.Errorf("property metadata must override column type: got %v (%T)", rows[0]["age"], rows[0]["age"])
	}
	untyped, _ := hydrateRows(result, nil)
	if untyped[0]["age"] != "42" {
		t.Errorf("no type info must yield string: got %v", untyped[0]["age"])
	}
}

func TestHydrateRowsPascalCase(t *testing.T) {
	result := map[string]any{
		"Columns":    []any{map[string]any{"Name": "n", "Type": "int4"}},
		"Rows":       []any{[]any{"7"}},
		"TotalCount": float64(1),
	}
	rows, total := hydrateRows(result, nil)
	if rows[0]["n"] != int64(7) {
		t.Errorf("PascalCase RowsResult: got %v", rows[0]["n"])
	}
	if total != float64(1) {
		t.Errorf("total: got %v", total)
	}
}

func TestToRecordInputs(t *testing.T) {
	records := toRecordInputs(map[string]any{"flag": true, "nested": map[string]any{"a": 1}, "empty": nil})
	byKey := map[string]string{}
	for _, record := range records {
		byKey[record["Key"]] = record["Value"]
	}
	if byKey["flag"] != "true" {
		t.Errorf("bool must lowercase: got %q", byKey["flag"])
	}
	if byKey["nested"] != `{"a":1}` {
		t.Errorf("objects must JSON-encode: got %q", byKey["nested"])
	}
	if byKey["empty"] != "" {
		t.Errorf("nil must encode empty: got %q", byKey["empty"])
	}
}

func TestConfigPrecedence(t *testing.T) {
	t.Setenv("WHODB_API_KEY", "")
	t.Setenv("WHODB_IPC_TOKEN", "")
	client, err := New(Config{APIKey: "whodb_sk_x", Org: "o", Project: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if !client.usingAPIKey {
		t.Error("explicit APIKey must mark usingAPIKey")
	}
	t.Setenv("WHODB_API_KEY", "whodb_sk_env")
	envClient, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !envClient.usingAPIKey {
		t.Error("WHODB_API_KEY env must mark usingAPIKey")
	}
	t.Setenv("WHODB_API_KEY", "")
	cliClient, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if cliClient.cli == nil {
		t.Error("no credentials must fall back to CLI provider")
	}
}

func TestIpcAutodetect(t *testing.T) {
	t.Setenv("WHODB_API_KEY", "")
	t.Setenv("WHODB_IPC_TOKEN", "tok")
	t.Setenv("WHODB_IPC_ADDRESS", "127.0.0.1:9999")
	client, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := client.transport.(*IpcTransport); !ok {
		t.Errorf("WHODB_IPC_TOKEN must autodetect IpcTransport, got %T", client.transport)
	}
}
