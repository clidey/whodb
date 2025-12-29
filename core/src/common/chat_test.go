package common

import (
	"errors"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

type rawExecuteStub struct {
	result *engine.GetRowsResult
	err    error
}

func (r rawExecuteStub) RawExecute(_ *engine.PluginConfig, _ string) (*engine.GetRowsResult, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}

func TestSQLChatParsesJsonAndMapsOperations(t *testing.T) {
	plugin := rawExecuteStub{
		result: &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "id", Type: "INT"}},
			Rows:    [][]string{{"1"}},
		},
	}

	response := "```json\n[{\"type\":\"sql\",\"operation\":\"get\",\"text\":\"select 1\"}]\n```"
	messages, err := SQLChat(response, &engine.PluginConfig{Credentials: &engine.Credentials{}}, plugin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected one chat message, got %d", len(messages))
	}
	if messages[0].Type != "sql:get" {
		t.Fatalf("expected operation to be mapped to sql:get, got %s", messages[0].Type)
	}
	if messages[0].Result == nil || len(messages[0].Result.Rows) != 1 {
		t.Fatalf("expected query result to be attached")
	}
}

func TestSQLChatHandlesExecutionErrors(t *testing.T) {
	plugin := rawExecuteStub{err: errors.New("boom")}
	response := "```json\n[{\"type\":\"sql\",\"operation\":\"get\",\"text\":\"select 1\"}]\n```"

	messages, err := SQLChat(response, &engine.PluginConfig{Credentials: &engine.Credentials{}}, plugin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 1 || messages[0].Type != "error" {
		t.Fatalf("expected error message type when execution fails")
	}
	if messages[0].Text != "boom" {
		t.Fatalf("expected error text to bubble up, got %s", messages[0].Text)
	}
}

func TestSQLChatRejectsMalformedResponse(t *testing.T) {
	plugin := rawExecuteStub{}
	if _, err := SQLChat("no json block here", &engine.PluginConfig{Credentials: &engine.Credentials{}}, plugin); err == nil {
		t.Fatalf("expected error when response lacks json block")
	}
}
