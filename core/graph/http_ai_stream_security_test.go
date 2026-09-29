//go:build !arm && !riscv64

package graph

import (
	"context"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/src/source"
	"testing"
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
