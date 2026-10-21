/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package mcp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// PlatformErrorCode is a stable category an agent can use to decide whether
// to retry, ask for setup, or change the requested operation.
type PlatformErrorCode string

const (
	PlatformErrorAuth         PlatformErrorCode = "authentication_required"
	PlatformErrorWorkspace    PlatformErrorCode = "workspace_required"
	PlatformErrorValidation   PlatformErrorCode = "invalid_input"
	PlatformErrorNotFound     PlatformErrorCode = "not_found"
	PlatformErrorPermission   PlatformErrorCode = "permission_denied"
	PlatformErrorConflict     PlatformErrorCode = "conflict"
	PlatformErrorRateLimited  PlatformErrorCode = "rate_limited"
	PlatformErrorBackend      PlatformErrorCode = "platform_error"
	PlatformErrorQuota        PlatformErrorCode = "quota_exceeded"
	PlatformErrorTimeout      PlatformErrorCode = "timeout"
	PlatformErrorUnavailable  PlatformErrorCode = "service_unavailable"
	PlatformErrorConfirmation PlatformErrorCode = "confirmation_expired"
)

func platformErrorFields(err error) (string, bool, []string) {
	if err == nil {
		return "", false, nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return string(PlatformErrorTimeout), true, nil
	}
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		return string(PlatformErrorTimeout), true, nil
	}
	var gql platformapi.GraphQLError
	if errors.As(err, &gql) && gql.Code != "" {
		return platformTypedErrorFields(gql.Code)
	}
	var httpErr platformapi.PlatformHTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case http.StatusUnauthorized:
			return platformTypedErrorFields("UNAUTHENTICATED")
		case http.StatusForbidden:
			return platformTypedErrorFields("FORBIDDEN")
		case http.StatusNotFound:
			return platformTypedErrorFields("NOT_FOUND")
		case http.StatusConflict:
			return platformTypedErrorFields("CONFLICT")
		case http.StatusTooManyRequests:
			return platformTypedErrorFields("RATE_LIMITED")
		case http.StatusBadGateway, http.StatusServiceUnavailable:
			return platformTypedErrorFields("SERVICE_UNAVAILABLE")
		case http.StatusRequestTimeout, http.StatusGatewayTimeout:
			return platformTypedErrorFields("DEADLINE_EXCEEDED")
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			return platformTypedErrorFields("BAD_USER_INPUT")
		}
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "confirmation") && (strings.Contains(message, "expired") || strings.Contains(message, "not found")):
		return string(PlatformErrorConfirmation), false, []string{"whodb_platform_pending"}
	case strings.Contains(message, "quota") && strings.Contains(message, "exceeded"):
		return platformTypedErrorFields("RESOURCE_EXHAUSTED")
	case strings.Contains(message, "access denied"):
		return platformTypedErrorFields("FORBIDDEN")
	case strings.Contains(message, "deadline exceeded"):
		return platformTypedErrorFields("DEADLINE_EXCEEDED")
	case strings.Contains(message, "temporarily unavailable"):
		return platformTypedErrorFields("SERVICE_UNAVAILABLE")
	case strings.Contains(message, "authentication") || strings.Contains(message, "login") || strings.Contains(message, "refresh token") || strings.Contains(message, "access token") || strings.Contains(message, "authenticated") || strings.Contains(message, "401"):
		return string(PlatformErrorAuth), false, []string{"whodb_platform_setup_status", "whodb_platform_status"}
	case (strings.Contains(message, "workspace") || strings.Contains(message, "organization") || strings.Contains(message, "project")) && (strings.Contains(message, "selected") || strings.Contains(message, "required")):
		return string(PlatformErrorWorkspace), false, []string{"whodb_platform_orgs", "whodb_platform_projects", "whodb_platform_workspace_resolve", "whodb_platform_session_use"}
	case strings.Contains(message, "embedding"):
		return string(PlatformErrorValidation), false, []string{"whodb_platform_ontology", "whodb_platform_ontology_embedding_jobs"}
	case strings.Contains(message, "policy-protected"):
		return string(PlatformErrorValidation), false, []string{"whodb_platform_ontology_rows", "whodb_platform_ontology_record_capabilities"}
	case strings.Contains(message, "permission") || strings.Contains(message, "forbidden") || strings.Contains(message, "403"):
		return string(PlatformErrorPermission), false, []string{"whodb_platform_resource_permissions"}
	case strings.Contains(message, "not found") || strings.Contains(message, "does not exist") || strings.Contains(message, "404"):
		return string(PlatformErrorNotFound), false, nil
	case strings.Contains(message, "already exists") || strings.Contains(message, "conflict") || strings.Contains(message, "409"):
		return string(PlatformErrorConflict), false, nil
	case strings.Contains(message, "rate limit") || strings.Contains(message, "429") || strings.Contains(message, "too many"):
		return string(PlatformErrorRateLimited), true, nil
	case strings.Contains(message, "required") || strings.Contains(message, "invalid") || strings.Contains(message, "unsupported") || strings.Contains(message, "does not support") || strings.Contains(message, "not supported"):
		return string(PlatformErrorValidation), false, []string{"whodb_platform_status"}
	default:
		return string(PlatformErrorBackend), false, nil
	}
}

func platformTypedErrorFields(code string) (string, bool, []string) {
	switch code {
	case "UNAUTHENTICATED":
		return string(PlatformErrorAuth), false, []string{"whodb_platform_setup_status", "whodb_platform_status"}
	case "FORBIDDEN":
		return string(PlatformErrorPermission), false, []string{"whodb_platform_resource_permissions"}
	case "BAD_USER_INPUT", "GRAPHQL_VALIDATION_FAILED":
		return string(PlatformErrorValidation), false, []string{"whodb_platform_status"}
	case "NOT_FOUND":
		return string(PlatformErrorNotFound), false, nil
	case "CONFLICT", "VERSION_CONFLICT":
		return string(PlatformErrorConflict), false, nil
	case "RESOURCE_EXHAUSTED":
		return string(PlatformErrorQuota), false, nil
	case "TOO_MANY_REQUESTS", "RATE_LIMITED":
		return string(PlatformErrorRateLimited), true, nil
	case "SERVICE_UNAVAILABLE":
		return string(PlatformErrorUnavailable), true, nil
	case "DEADLINE_EXCEEDED":
		return string(PlatformErrorTimeout), true, nil
	default:
		return string(PlatformErrorBackend), false, nil
	}
}

// PlatformRecoveryAdvice is deterministic guidance an agent can follow after a failed call.
type PlatformRecoveryAdvice struct {
	LikelyCause string
	NextSteps   []string
}

func platformRecoveryAdvice(code string, message string) PlatformRecoveryAdvice {
	switch PlatformErrorCode(code) {
	case PlatformErrorConfirmation:
		return PlatformRecoveryAdvice{"the write approval has expired", []string{"request a new preview for the same intended operation", "obtain approval again", "confirm the new token"}}
	case PlatformErrorQuota:
		return PlatformRecoveryAdvice{"the workspace resource quota is exhausted", []string{"inspect usage and plan limits", "free capacity or change the plan before retrying"}}
	case PlatformErrorTimeout, PlatformErrorUnavailable:
		return PlatformRecoveryAdvice{"the operation timed out or its service is temporarily unavailable", []string{"inspect the resource or run status before retrying a write", "retry reads after a short delay", "reuse the original idempotency key when retrying a write"}}
	case PlatformErrorAuth:
		return PlatformRecoveryAdvice{"the hosted session is missing or expired", []string{"run whodb_platform_setup_status", "ask the user to run whodb login", "retry the original tool"}}
	case PlatformErrorWorkspace:
		return PlatformRecoveryAdvice{"no usable organization or project is selected", []string{"run whodb_platform_orgs", "run whodb_platform_projects", "run whodb_platform_workspace_resolve with the intended workspace", "or run whodb_platform_session_use to make it the default for this session"}}
	case PlatformErrorPermission:
		return PlatformRecoveryAdvice{"the signed-in user does not have the required platform permission", []string{"run whodb_platform_resource_permissions for the target", "ask the user to use the platform to grant access", "do not retry unchanged"}}
	case PlatformErrorNotFound:
		return PlatformRecoveryAdvice{"the resource id or name is not visible in the selected project", []string{"run whodb_platform_resolve_resource", "run whodb_platform_workspace_map", "verify the selected workspace"}}
	case PlatformErrorConflict:
		return PlatformRecoveryAdvice{"the requested state conflicts with an existing resource or pending change", []string{"resolve the target with whodb_platform_resolve_resource", "read the target and change impact", "retry with an explicit id or adjusted payload"}}
	case PlatformErrorValidation:
		return PlatformRecoveryAdvice{"the request does not match the hosted operation contract", []string{"run whodb_platform_status to refresh capabilities", "run whodb_platform_write_plan", "correct the named payload field and retry"}}
	case PlatformErrorRateLimited:
		return PlatformRecoveryAdvice{"the hosted platform asked the caller to slow down", []string{"wait briefly", "retry once", "use a bounded polling interval for long-running operations"}}
	default:
		if strings.Contains(strings.ToLower(message), "transform") || strings.Contains(strings.ToLower(message), "function") {
			return PlatformRecoveryAdvice{"the hosted runtime rejected or could not complete the operation", []string{"run whodb_platform_runtime_readiness", "read the target resource and recent runs", "retry only after fixing the reported runtime issue"}}
		}
		return PlatformRecoveryAdvice{"the hosted platform returned an unexpected error", []string{"run whodb_platform_status", "read the target resource", "retry only if the error is transient"}}
	}
}

func normalizePlatformErrorOutput(output map[string]any, message string) {
	code, _ := output["error_code"].(string)
	if code == "" {
		var retry bool
		var suggested []string
		code, retry, suggested = platformErrorFields(errors.New(message))
		output["error_code"] = code
		output["retryable"] = retry
		if len(suggested) > 0 {
			output["suggested_tools"] = suggested
		}
	}
	if output["request_id"] == nil || output["request_id"] == "" {
		output["request_id"] = generateRequestID("platform")
	}
	if output["recovery"] == nil {
		output["recovery"] = platformRecoveryAdvice(code, message)
	}
}
