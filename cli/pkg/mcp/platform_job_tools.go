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

// PlatformFunctionRunsInput is the input for the whodb_platform_function_runs tool.
type PlatformFunctionRunsInput struct {
	FunctionID string   `json:"function_id,omitempty" jsonschema:"Function id; lists its recent runs"`
	RunID      string   `json:"run_id,omitempty" jsonschema:"Function run id; returns that one run instead of a list"`
	Limit      int      `json:"limit,omitempty" jsonschema:"Maximum runs to return"`
	Fields     []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// HandlePlatformFunctionRuns returns one persisted function run, or the recent runs of one function.
func HandlePlatformFunctionRuns(ctx context.Context, req *mcp.CallToolRequest, input PlatformFunctionRunsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.FunctionID) == "" && strings.TrimSpace(input.RunID) == "" {
		return nil, PlatformReadOutput{Error: "function_id or run_id is required", RequestID: generateRequestID("platform_function_runs")}, nil
	}
	return platformProjectRead(ctx, "platform_function_runs", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		if strings.TrimSpace(input.RunID) != "" {
			run, err := session.Client.FunctionRun(ctx, session.Host.DefaultProjectID, input.RunID)
			return run, 1, false, err
		}
		runs, err := session.Client.FunctionRuns(ctx, session.Host.DefaultProjectID, input.FunctionID, input.Limit)
		return runs, len(runs), false, err
	})
}

// PlatformFunctionWaitInput waits for one persisted function run to finish.
type PlatformFunctionWaitInput struct {
	RunID       string   `json:"run_id" jsonschema:"Function run id returned by the start_run action or function_runs tool"`
	TimeoutSecs int      `json:"timeout_seconds,omitempty" jsonschema:"Maximum wait time, default 60 seconds"`
	PollSecs    int      `json:"poll_seconds,omitempty" jsonschema:"Polling interval, default 2 seconds"`
	Fields      []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// HandlePlatformFunctionWait polls one persisted function run until it completes, fails or is canceled.
func HandlePlatformFunctionWait(ctx context.Context, req *mcp.CallToolRequest, input PlatformFunctionWaitInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	requestID := generateRequestID("platform_function_wait")
	if strings.TrimSpace(input.RunID) == "" {
		return nil, PlatformReadOutput{Error: "run_id is required", RequestID: requestID}, nil
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
		run, err := session.Client.FunctionRun(ctx, session.Host.DefaultProjectID, input.RunID)
		if err != nil {
			return nil, platformReadErrorOutput(err, requestID), nil
		}
		if run.Finished() {
			return nil, platformReadOutput(session, "platform_function_wait", run, 1, false, requestID, input.Fields), nil
		}
		select {
		case <-ctx.Done():
			return nil, platformReadErrorOutput(ctx.Err(), requestID), nil
		case <-deadline.C:
			return nil, PlatformReadOutput{Error: "function run is still " + run.Status + " after the timeout; call again with the same run_id", ErrorCode: string(PlatformErrorRateLimited), Retryable: true, RequestID: requestID}, nil
		case <-time.After(poll):
		}
	}
}

func platformFunctionWaitToolDefinition() *mcp.Tool {
	return &mcp.Tool{Name: "whodb_platform_function_wait", Description: "Wait for a persisted hosted function run to complete, fail or be canceled, then return its output, logs and error. Use the run id returned by the function start_run action and keep the timeout bounded; a run keeps going after a timed-out wait.", Annotations: platformReadOnlyAnnotations("Wait For Hosted Function Run")}
}
