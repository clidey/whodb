/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package mcp

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolErrorPreservesRecoveryPayload(t *testing.T) {
	for _, failure := range []bool{false, true} {
		payload := map[string]any{"request_id": "request-1", "scope": map[string]string{"host": "https://example.test"}, "next_commands": []string{"whodb login"}}
		if failure {
			payload["error"] = "login expired"
		}
		original := &mcp.CallToolResult{StructuredContent: payload, Content: []mcp.Content{&mcp.TextContent{Text: "original details"}}}
		handler := toolErrorMiddleware(func(context.Context, string, mcp.Request) (mcp.Result, error) { return original, nil })
		result, err := handler(context.Background(), "tools/call", nil)
		if err != nil || result != original || original.IsError != failure {
			t.Fatalf("result = %#v, error = %v", result, err)
		}
		if original.StructuredContent.(map[string]any)["request_id"] != "request-1" || original.Content[0].(*mcp.TextContent).Text != "original details" {
			t.Fatal("error details changed")
		}
	}
}
