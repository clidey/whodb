/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	platformapi "github.com/clidey/whodb/cli/internal/platform"

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
	return waitForPlatformRun(ctx, session, "platform_transform_wait", requestID, input.Fields, timeout, poll, func(ctx context.Context) (*platformapi.TransformRun, bool, error) {
		runs, err := session.Client.TransformRuns(ctx, session.Host.DefaultProjectID, input.TransformID, 100)
		if err != nil {
			return nil, false, err
		}
		for _, run := range runs {
			if run.ID == input.RunID {
				return &run, isTerminalTransformRunStatus(run.Status), nil
			}
		}
		return nil, false, nil
	})
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
	return waitForPlatformRun(ctx, session, "platform_function_wait", requestID, input.Fields, timeout, poll, func(ctx context.Context) (*platformapi.FunctionRun, bool, error) {
		run, err := session.Client.FunctionRun(ctx, session.Host.DefaultProjectID, input.RunID)
		if err != nil {
			return nil, false, err
		}
		return run, run.Finished(), nil
	})
}

func platformFunctionWaitToolDefinition() *mcp.Tool {
	return &mcp.Tool{Name: "whodb_platform_function_wait", Description: "Wait for a persisted hosted function run to complete, fail or be canceled, then return its output, logs and error. Use the run id returned by the function start_run action and keep the timeout bounded; a run keeps going after a timed-out wait.", Annotations: platformReadOnlyAnnotations("Wait For Hosted Function Run")}
}

// Leave response-serialization time inside the transport deadline, including
// when a polling request itself stalls. A wait never cancels the hosted job.
func waitForPlatformRun[T any](ctx context.Context, session *platformToolSession, operation, requestID string, fields []string, timeout, poll time.Duration, fetch func(context.Context) (*T, bool, error)) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if end, ok := ctx.Deadline(); ok {
		remaining := time.Until(end)
		reserve := min(time.Second, remaining/10)
		timeout = min(timeout, remaining-reserve)
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var latest *T
	finish := func() (*mcp.CallToolResult, PlatformReadOutput, error) {
		if ctx.Err() != nil {
			return nil, platformReadErrorOutput(ctx.Err(), requestID), nil //nolint:nilerr // MCP errors are returned in the structured result.
		}
		if latest == nil {
			return nil, platformReadErrorOutput(fmt.Errorf("no run status available before wait deadline: %w", context.DeadlineExceeded), requestID), nil
		}
		out := platformReadOutput(session, operation, latest, 1, false, requestID, fields)
		out.Warnings = append(out.Warnings, "The run was still in progress at the last poll. Call again with the same run_id to check its status.")
		return nil, out, nil
	}
	for {
		run, terminal, err := fetch(waitCtx)
		if err != nil {
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return finish()
			}
			return nil, platformReadErrorOutput(err, requestID), nil
		}
		if run != nil {
			latest = run
		}
		if terminal {
			return nil, platformReadOutput(session, operation, run, 1, false, requestID, fields), nil
		}
		timer := time.NewTimer(poll)
		select {
		case <-waitCtx.Done():
			timer.Stop()
			return finish()
		case <-timer.C:
		}
	}
}
