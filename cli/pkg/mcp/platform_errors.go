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
	case strings.Contains(message, "rate") || strings.Contains(message, "429") || strings.Contains(message, "too many"):
		return string(PlatformErrorRateLimited), true, nil
	case strings.Contains(message, "required") || strings.Contains(message, "invalid") || strings.Contains(message, "unsupported"):
		return string(PlatformErrorValidation), false, []string{"whodb_platform_status"}
	default:
		return string(PlatformErrorBackend), false, nil
	}
}
