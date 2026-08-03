/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package mcp

import (
	"context"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformTransformWaitInput waits for one hosted transform run to finish.
type PlatformTransformWaitInput struct {
	TransformID string   `json:"transform_id" jsonschema:"Hosted transform id"`
	RunID       string   `json:"run_id" jsonschema:"Transform run id returned by the run action or transform_runs tool"`
	TimeoutSecs int      `json:"timeout_seconds,omitempty" jsonschema:"Maximum wait time, default 60 seconds"`
	PollSecs    int      `json:"poll_seconds,omitempty" jsonschema:"Polling interval, default 2 seconds"`
	Fields      []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// HandlePlatformTransformWait polls the real hosted transform-run history and returns a terminal run.
func HandlePlatformTransformWait(ctx context.Context, req *mcp.CallToolRequest, input PlatformTransformWaitInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	requestID := generateRequestID("platform_transform_wait")
	if strings.TrimSpace(input.TransformID) == "" || strings.TrimSpace(input.RunID) == "" {
		return nil, PlatformReadOutput{Error: "transform_id and run_id are required", RequestID: requestID}, nil
	}
	timeout := time.Duration(input.TimeoutSecs) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	poll := time.Duration(input.PollSecs) * time.Second
	if poll <= 0 {
		poll = 2 * time.Second
	}
	if poll > timeout {
		poll = timeout
	}
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		return nil, platformReadErrorOutput(err, requestID), nil
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		runs, err := session.Client.TransformRuns(ctx, session.Host.DefaultProjectID, input.TransformID, 100)
		if err != nil {
			return nil, platformReadErrorOutput(err, requestID), nil
		}
		for _, run := range runs {
			if run.ID != input.RunID {
				continue
			}
			if isTerminalTransformRunStatus(run.Status) {
				return nil, platformReadOutput(session, "platform_transform_wait", run, 1, false, requestID, input.Fields), nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, platformReadErrorOutput(ctx.Err(), requestID), nil
		case <-deadline.C:
			return nil, PlatformReadOutput{Error: "transform run did not reach a terminal state before timeout", ErrorCode: string(PlatformErrorRateLimited), Retryable: true, RequestID: requestID}, nil
		case <-time.After(poll):
		}
	}
}

func isTerminalTransformRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "succeeded", "completed", "complete", "failed", "failure", "cancelled", "canceled", "error":
		return true
	default:
		return false
	}
}

func platformTransformWaitToolDefinition() *mcp.Tool {
	return &mcp.Tool{Name: "whodb_platform_transform_wait", Description: "Wait for a real hosted transform run to reach success or failure. Use the run id returned by whodb_platform_action and keep the timeout bounded.", Annotations: platformReadOnlyAnnotations("Wait For Hosted Transform Run")}
}
