/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package mcp

import (
	"errors"

	"github.com/google/uuid"
)

// Organization backup tools cover status and capture only. Recovery keys, archive
// transfer and restore stay out of MCP so recovery material never enters a tool conversation.
func init() {
	RegisterPlatformExtension(PlatformExtensionOperation{
		Name: "whodb_platform_org_backup_status", Description: "List organization backup jobs and permitted settings. Admins see status without private recovery configuration. For key handling, archive transfer or restore, open the recovery_url in the WhoDB UI.",
		Operation: "OrganizationBackups", Document: `query OrganizationBackups($orgId: ID!) { OrganizationBackups(orgId: $orgId) { canCreate isOwner storedBytes maxArchiveBytes emailConfigured key { id recipient confirmedAt } policy { id projectId enabled schedule timeZone retentionDays selection notificationUsers nextRunAt } projectPolicies { id projectId enabled schedule timeZone retentionDays selection notificationUsers nextRunAt } jobs { id keyId scheduleId projectId deletionRequestedAt deleteAfter kind state createdAt capturedAt scheduledFor completedAt expiresAt retentionDays sizeBytes sha256 errorCode } } }`,
		Variables: func(orgID string, _ PlatformExtensionInput) (map[string]any, error) {
			return map[string]any{"orgId": orgID}, nil
		},
	})
	RegisterPlatformExtension(PlatformExtensionOperation{
		Name: "whodb_platform_org_backup_scope", Description: "Preview the owner-selected organization content and automatic dependencies. Empty selection means all WhoDB-owned organization content. Owner only.",
		Operation: "OrganizationBackupScope", Document: `query OrganizationBackupScope($orgId: ID!, $selection: OrganizationBackupSelectionInput!) { OrganizationBackupScope(orgId: $orgId, selection: $selection) { full projects projectNames resources { projectId type id name automatic } requirements { kind projectId id version } } }`,
		Variables: func(orgID string, input PlatformExtensionInput) (map[string]any, error) {
			return map[string]any{"orgId": orgID, "selection": backupSelection(input.Selection)}, nil
		},
	})
	RegisterPlatformExtension(PlatformExtensionOperation{
		Name: "whodb_platform_org_backup_notifications", Description: "List failure notices addressed to the caller for an organization.",
		Operation: "OrganizationBackupNotifications", Document: `query OrganizationBackupNotifications($orgId: ID!) { OrganizationBackupNotifications(orgId: $orgId) { id backupId errorCode createdAt readAt } }`,
		Variables: func(orgID string, _ PlatformExtensionInput) (map[string]any, error) {
			return map[string]any{"orgId": orgID}, nil
		},
	})
	RegisterPlatformExtension(PlatformExtensionOperation{
		Name: "whodb_platform_org_backup_create", Description: "Request a manual encrypted organization backup with write confirmation. Admins use the owner-saved selection; a custom selection requires owner authority. Supply request_id to make retries idempotent.",
		Operation: "CreateOrganizationBackup", Write: true, Document: `mutation CreateOrganizationBackup($orgId: ID!, $requestId: ID!, $selection: OrganizationBackupSelectionInput) { CreateOrganizationBackup(orgId: $orgId, requestId: $requestId, selection: $selection) { id state createdAt } }`,
		Variables: func(orgID string, input PlatformExtensionInput) (map[string]any, error) {
			requestID := input.RequestID
			if requestID == "" {
				requestID = uuid.NewString()
			}
			variables := map[string]any{"orgId": orgID, "requestId": requestID}
			if input.Selection != nil {
				variables["selection"] = backupSelection(input.Selection)
			}
			return variables, nil
		},
	})
	RegisterPlatformExtension(PlatformExtensionOperation{
		Name: "whodb_platform_org_backup_policy_set", Description: "Set an organization backup schedule, retention, scope and notification recipients. Owner only; requires write confirmation. Supply a complete policy object.",
		Operation: "SaveOrganizationBackupPolicy", Write: true, Document: `mutation SaveOrganizationBackupPolicy($input: OrganizationBackupPolicyInput!) { SaveOrganizationBackupPolicy(input: $input) { Status } }`,
		Variables: func(orgID string, input PlatformExtensionInput) (map[string]any, error) {
			if input.Policy == nil {
				return nil, errors.New("policy is required")
			}
			policy := make(map[string]any, len(input.Policy)+1)
			for key, value := range input.Policy {
				policy[key] = value
			}
			policy["orgId"] = orgID
			return map[string]any{"input": policy}, nil
		},
	})
}

func backupSelection(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{"projects": []string{}, "resources": []any{}}
	}
	return value
}
