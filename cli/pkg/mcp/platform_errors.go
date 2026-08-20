/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package mcp

import "strings"

// PlatformErrorCode is a stable category an agent can use to decide whether
// to retry, ask for setup, or change the requested operation.
type PlatformErrorCode string

const (
	PlatformErrorAuth        PlatformErrorCode = "authentication_required"
	PlatformErrorWorkspace   PlatformErrorCode = "workspace_required"
	PlatformErrorValidation  PlatformErrorCode = "invalid_input"
	PlatformErrorNotFound    PlatformErrorCode = "not_found"
	PlatformErrorPermission  PlatformErrorCode = "permission_denied"
	PlatformErrorConflict    PlatformErrorCode = "conflict"
	PlatformErrorRateLimited PlatformErrorCode = "rate_limited"
	PlatformErrorBackend     PlatformErrorCode = "platform_error"
)

func platformErrorFields(err error) (string, bool, []string) {
	if err == nil {
		return "", false, nil
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "login") || strings.Contains(message, "token") || strings.Contains(message, "authenticated") || strings.Contains(message, "401"):
		return string(PlatformErrorAuth), false, []string{"whodb_platform_setup_status", "whodb_platform_status"}
	case strings.Contains(message, "workspace") || strings.Contains(message, "organization") || strings.Contains(message, "project") && strings.Contains(message, "selected"):
		return string(PlatformErrorWorkspace), false, []string{"whodb_platform_orgs", "whodb_platform_projects", "whodb_platform_use"}
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

// PlatformRecoveryAdvice is deterministic guidance an agent can follow after a failed call.
type PlatformRecoveryAdvice struct {
	LikelyCause string
	NextSteps   []string
}

func platformRecoveryAdvice(code string, message string) PlatformRecoveryAdvice {
	switch PlatformErrorCode(code) {
	case PlatformErrorAuth:
		return PlatformRecoveryAdvice{"the hosted session is missing or expired", []string{"run whodb_platform_setup_status", "ask the user to run whodb login", "retry the original tool"}}
	case PlatformErrorWorkspace:
		return PlatformRecoveryAdvice{"no usable organization or project is selected", []string{"run whodb_platform_orgs", "run whodb_platform_projects", "run whodb_platform_use with the intended workspace"}}
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
