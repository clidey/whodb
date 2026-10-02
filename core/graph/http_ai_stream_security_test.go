//go:build !arm && !riscv64

package graph

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/src/source"
)

type protectedChatRecorder struct{ calls int }

func (r *protectedChatRecorder) RunReadOnlyQuery(context.Context, string, ...any) (*source.RowsResult, error) {
	r.calls++
	return &source.RowsResult{}, nil
}

func TestStreamedChatUsesSQLPolicy(t *testing.T) {
	for _, query := range []string{"SELECT * INTO stolen FROM users", "SELECT 1; DELETE FROM users", "SELECT side_effect()"} {
		runner := &protectedChatRecorder{}
		op := types.OperationTypeGET
		response := processFinalResponse(t.Context(), &types.ChatResponse{Type: types.ChatMessageTypeSQL, Operation: &op, Text: query}, runner)
		if !response.RequiresConfirmation || runner.calls != 0 {
			t.Fatalf("model label bypassed streamed approval: %#v", response)
		}
	}
	runner := &protectedChatRecorder{}
	op := types.OperationTypeGET
	response := processFinalResponse(t.Context(), &types.ChatResponse{Type: types.ChatMessageTypeSQL, Operation: &op, Text: "SELECT 1"}, runner)
	if response.RequiresConfirmation || runner.calls != 1 {
		t.Fatal("read did not use protected executor")
	}
}

func TestChatRejectsToolPayloadAndMissingOperation(t *testing.T) {
	if !unsupportedChatTool(`{"toolName":"search_ontology_catalog","arguments":{}}`) {
		t.Fatal("tool payload was accepted as chat text")
	}
	if unsupportedChatTool(`{"name":"ordinary result"}`) {
		t.Fatal("ordinary JSON was mistaken for a tool action")
	}
	runner := &protectedChatRecorder{}
	response := processFinalResponse(t.Context(), &types.ChatResponse{Type: types.ChatMessageTypeSQL, Text: "SELECT 1"}, runner)
	if response.Type != "error" || runner.calls != 0 {
		t.Fatalf("missing operation was treated as executed SQL: %#v", response)
	}
}

func TestStreamedChatFailureMessages(t *testing.T) {
	response := httptest.NewRecorder()
	SendSSESQLFailure(response, response, "SELECT *\nFROM missing", "no such table: missing")
	var message struct {
		Type string
		Text string
	}
	stream := response.Body.String()
	if !strings.Contains(stream, "event: done") {
		t.Fatal("SQL failure did not complete the stream")
	}
	line := strings.Split(strings.Split(stream, "data: ")[1], "\n")[0]
	if err := json.Unmarshal([]byte(line), &message); err != nil {
		t.Fatal(err)
	}
	var details map[string]string
	if err := json.Unmarshal([]byte(message.Text), &details); err != nil {
		t.Fatal(err)
	}
	if message.Type != "sql:error" || details["sql"] != "SELECT *\nFROM missing" || details["error"] != "no such table: missing" {
		t.Fatalf("SQL failure lost query details: %#v", message)
	}

	response = httptest.NewRecorder()
	sendModelError(response, response, "localhost:1234: connection refused")
	if !strings.Contains(response.Body.String(), `"Type":"provider:error"`) || !strings.Contains(response.Body.String(), "event: done") {
		t.Fatalf("provider failure was not shown inline: %s", response.Body.String())
	}
}
