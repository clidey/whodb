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
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolErrorMiddleware preserves structured recovery details while marking failed operations.
func toolErrorMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		result, err := next(ctx, method, req)
		if call, ok := result.(*mcp.CallToolResult); ok && call != nil && !call.IsError {
			data, marshalErr := json.Marshal(call.StructuredContent)
			var output struct {
				Error string `json:"error"`
			}
			if marshalErr == nil && json.Unmarshal(data, &output) == nil && output.Error != "" {
				call.IsError = true
			}
		}
		return result, err
	}
}
