/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package platform

import (
	"context"
	"encoding/json"
	"fmt"
)

// platformQuerySpecs is the bounded read surface exposed to the platform MCP.
// It is deliberately not a general GraphQL passthrough: callers can select a
// known operation, while the CLI owns the query document and field set.
var platformQuerySpecs = map[string]string{
	"OAuthSecretProviders": `query CLIOAuthSecretProviders {
  OAuthSecretProviders { id name category scopes defaultScopes callbackURL documentationURL setupHint expiryOptional scopeOptional renewal }
}`,
	"OAuthSecretStatus": `query CLIOAuthSecretStatus($projectId: ID!, $id: ID!) {
  OAuthSecretStatus(projectId: $projectId, id: $id) { provider state scopes requestedScopes scopesVerified expiresAt lastRenewedAt renewable callbackURL }
}`,
	"ValidateOntologyBehaviors": `query CLIValidateOntologyBehaviors($projectId: ID!, $manifest: JSON!) {
  ValidateOntologyBehaviors(projectId: $projectId, manifest: $manifest)
 }`,
	"AppBuilderReference": `query CLIPlatformAppBuilderReference($kind: String!, $name: String) {
  AppBuilderReference(kind: $kind, name: $name)
}`,
	"AppLogicGraph": `query CLIPlatformAppLogicGraph($projectId: ID!, $appId: ID!, $page: String) {
  AppLogicGraph(projectId: $projectId, appId: $appId, page: $page)
}`,
	"ProjectApps": `query CLIPlatformProjectApps($projectId: ID!) {
  ProjectApps(projectId: $projectId) { id projectId name description thumbnailUrl useCases ontologyIds readOnlyOntologyIds functionIds createdBy createdAt updatedAt sourceHidden offlineCapable recoveryBindings }
}`,
	"AppDetail": `query CLIPlatformAppDetail($projectId: ID!, $id: ID!) {
  AppDetail(projectId: $projectId, id: $id) { id projectId name description thumbnailUrl useCases ontologyIds readOnlyOntologyIds functionIds createdBy createdAt updatedAt sourceHidden offlineCapable recoveryBindings }
}`,
	"AppFiles": `query CLIPlatformAppFiles($projectId: ID!, $appId: ID!) {
  AppFiles(projectId: $projectId, appId: $appId) { path content updatedAt }
}`,
	"AppView": `query CLIPlatformAppView($projectId: ID!, $id: ID!, $env: String, $page: String) {
  AppView(projectId: $projectId, id: $id, env: $env, page: $page) { version pages app { id projectId name description thumbnailUrl useCases html conversation ontologyIds readOnlyOntologyIds functionIds createdBy createdAt updatedAt sourceHidden offlineCapable recoveryBindings } files { path content updatedAt } }
}`,
	"AppVersionView": `query CLIPlatformAppVersionView($projectId: ID!, $appId: ID!, $version: Int!) {
  AppVersionView(projectId: $projectId, appId: $appId, version: $version) { version app { id projectId name description thumbnailUrl useCases html conversation ontologyIds readOnlyOntologyIds functionIds createdBy createdAt updatedAt sourceHidden offlineCapable recoveryBindings } files { path content updatedAt } }
}`,
	"OrganizationPackages": `query CLIPlatformOrganizationPackages($organizationId: ID!) {
  OrganizationPackages(organizationId: $organizationId) { id sourceProjectId name version channel description thumbnailUrl useCases installable importable visibility createdBy createdAt items { sourceProjectId sourceProjectName objectId objectType name thumbnailUrl version role visibility importable } requirements { key kind label description required metadata } stats { installCount importCount reviewCount averageRating } }
}`,
	"OrganizationPackage": `query CLIPlatformOrganizationPackage($organizationId: ID!, $packageId: ID!) {
  OrganizationPackage(organizationId: $organizationId, packageId: $packageId) { id sourceProjectId name version channel description thumbnailUrl useCases installable importable visibility createdBy createdAt
 items { sourceProjectId sourceProjectName objectId objectType name thumbnailUrl version role visibility importable }
 requirements { key kind label description required metadata }
 teams { teamName sourceTeamId permissions { sourceObjectId sourceObjectType relation } }
 stats { installCount importCount reviewCount averageRating }
 reviews { id rating body createdBy createdAt }
 media { id kind url caption sortOrder createdAt }
 changelog { id version title body createdAt }
 lineage { nodes { id nodeType name providerId } edges { sourceId sourceType targetId targetType createdAt } } }
}`,
	"PackageInstallations": `query CLIPlatformPackageInstallations($projectId: ID!) {
  PackageInstallations(projectId: $projectId) { id sourceOrgId sourceProjectId sourceScopeProjectId targetProjectId packageId packageName packageVersion packageChannel status installedBy installedAt updatedAt updateInfo { sourceAvailable sourceRemovedAt changes { action pruneAction sourceObjectId sourceObjectType sourceVersion targetObjectId targetObjectType } available currentPackageId currentVersion latestPackageId latestVersion latestCreatedAt channel } items { sourceObjectId sourceObjectType sourceVersion sourceVisibility sourceImportable targetObjectId targetObjectType } }
}`,
	"PackageInstallationUpdate": `query CLIPlatformPackageInstallationUpdate($projectId: ID!, $installationId: ID!) {
	  PackageInstallationUpdate(projectId: $projectId, installationId: $installationId) { available sourceAvailable sourceRemovedAt currentPackageId currentVersion latestPackageId latestVersion latestCreatedAt channel changes { action pruneAction sourceObjectId sourceObjectType sourceVersion targetObjectId targetObjectType } }
}`,
	"PackageInstallationCustomizations": `query CLIPlatformPackageInstallationCustomizations($projectId: ID!, $installationId: ID!) {
  PackageInstallationCustomizations(projectId: $projectId, installationId: $installationId) { id installationId sourceObjectId sourceObjectType targetObjectId path mode content updatedBy createdAt updatedAt }
}`,
	"PackageLibrary": `query CLIPlatformPackageLibrary($search: String) {
  PackageLibrary(search: $search) { sourceOrgName share { id packageId sourceOrgId sourceProjectId url createdBy createdAt revokedAt } package { id sourceProjectId name version channel description thumbnailUrl useCases installable importable visibility createdBy createdAt } }
}`,
	"SharedPackage": `query CLIPlatformSharedPackage($shareToken: String!) {
  SharedPackage(shareToken: $shareToken) { share { id packageId sourceOrgId sourceProjectId url createdBy createdAt revokedAt } package { id sourceProjectId name version channel description thumbnailUrl useCases installable importable visibility createdBy createdAt } }
}`,
	"PreviewCreatePackage": `query CLIPlatformPreviewCreatePackage($input: CreatePackageInput!) {
  PreviewCreatePackage(input: $input) { items { sourceProjectId objectId objectType name version role visibility importable included reason } teams { teamName sourceTeamId permissions { sourceObjectId sourceObjectType relation } } requirements { key kind label description required metadata } issues { severity message objectId objectType } conflicts { objectId objectType sourceName targetName resolution } }
}`,
	"PreviewInstallPackage": `query CLIPlatformPreviewInstallPackage($input: InstallPackageInput!) {
  PreviewInstallPackage(input: $input) { items { sourceProjectId objectId objectType name version role visibility importable included reason } teams { teamName sourceTeamId permissions { sourceObjectId sourceObjectType relation } } requirements { key kind label description required metadata } issues { severity message objectId objectType } conflicts { objectId objectType sourceName targetName resolution } }
}`,
	"PreviewImportPackage": `query CLIPlatformPreviewImportPackage($input: ImportPackageInput!) {
  PreviewImportPackage(input: $input) { items { sourceProjectId objectId objectType name version role visibility importable included reason } teams { teamName sourceTeamId permissions { sourceObjectId sourceObjectType relation } } requirements { key kind label description required metadata } issues { severity message objectId objectType } conflicts { objectId objectType sourceName targetName resolution } }
}`,
	"PreviewSharedPackageInstall": `query CLIPlatformPreviewSharedPackageInstall($input: SharedPackageOperationInput!) {
  PreviewSharedPackageInstall(input: $input) { items { sourceProjectId objectId objectType name version role visibility importable included reason } teams { teamName sourceTeamId permissions { sourceObjectId sourceObjectType relation } } requirements { key kind label description required metadata } issues { severity message objectId objectType } conflicts { objectId objectType sourceName targetName resolution } }
}`,
	"PreviewSharedPackageImport": `query CLIPlatformPreviewSharedPackageImport($input: SharedPackageOperationInput!) {
  PreviewSharedPackageImport(input: $input) { items { sourceProjectId objectId objectType name version role visibility importable included reason } teams { teamName sourceTeamId permissions { sourceObjectId sourceObjectType relation } } requirements { key kind label description required metadata } issues { severity message objectId objectType } conflicts { objectId objectType sourceName targetName resolution } }
}`,
	"ObjectVersions": `query CLIPlatformObjectVersions($projectId: ID!, $objectId: ID!, $objectType: VersionableObjectType!) {
  ObjectVersions(projectId: $projectId, objectId: $objectId, objectType: $objectType) { id objectId objectType version snapshot message promotedBy createdAt }
}`,
	"ActiveProdVersion": `query CLIPlatformActiveProdVersion($projectId: ID!, $objectId: ID!, $objectType: VersionableObjectType!) {
  ActiveProdVersion(projectId: $projectId, objectId: $objectId, objectType: $objectType) { objectId objectType version activatedAt activatedBy }
}`,
	"OntologyBehavior": `query CLIPlatformOntologyBehavior($projectId: ID!, $ontologyId: ID!) {
  OntologyBehavior(projectId: $projectId, ontologyId: $ontologyId) { id ontologyId document revision createdAt updatedAt }
}`,
	"ProjectActiveProdVersions": `query CLIPlatformProjectActiveProdVersions($projectId: ID!) {
  ProjectActiveProdVersions(projectId: $projectId) { objectId objectType version activatedAt activatedBy }
}`,
	"WhoHasAccess": `query CLIPlatformWhoHasAccess($resourceType: String!, $resourceId: ID!) {
  WhoHasAccess(resourceType: $resourceType, resourceId: $resourceId) { subject relation displayName }
}`,
	"MyPermissions": `query CLIPlatformMyPermissions($resourceType: String!, $resourceId: ID!) {
  MyPermissions(resourceType: $resourceType, resourceId: $resourceId)
}`,
	"WhatCanIAccess": `query CLIPlatformWhatCanIAccess($resourceType: String!) {
  WhatCanIAccess(resourceType: $resourceType)
}`,
	"OrgResources": `query CLIPlatformOrgResources($orgId: ID!) {
  OrgResources(orgId: $orgId) { id resourceType name owner createdAt }
}`,
	"OrgMembers": `query CLIPlatformOrgMembers($orgId: ID!) {
  OrgMembers(orgId: $orgId) { id email }
}`,
	"OrganizationDomains": `query CLIPlatformOrganizationDomains($orgId: ID!) {
  OrganizationDomains(orgId: $orgId) { id orgId domain isPrimary verifiedAt createdAt }
}`,
	"OrganizationSSOProviders": `query CLIPlatformOrganizationSSOProviders($orgId: ID!) {
  OrganizationSSOProviders(orgId: $orgId) { id orgId providerType displayName providerAlias enabled isDefault createdAt updatedAt }
}`,
	"Teams": `query CLIPlatformTeams($orgId: ID!) {
  Teams(orgId: $orgId) { id orgId name createdAt }
}`,
	"TeamMembers": `query CLIPlatformTeamMembers($teamId: ID!) {
  TeamMembers(teamId: $teamId) { userId email joinedAt }
}`,
	"DeletionImpact": `query CLIPlatformDeletionImpact($resourceType: String!, $resourceId: ID!, $projectId: ID, $orgId: ID) {
  DeletionImpact(resourceType: $resourceType, resourceId: $resourceId, projectId: $projectId, orgId: $orgId) { resource { id resourceType name } children { id resourceType name } references { id resourceType name } dependents { id resourceType name } sharedWith requiresConfirmation }
}`,
	"DeletedResources": `query CLIPlatformDeletedResources($orgId: ID, $projectId: ID) {
  DeletedResources(orgId: $orgId, projectId: $projectId) { id resourceType name projectId deletedAt deletedBy }
}`,
	"SharedWithMe": `query CLIPlatformSharedWithMe($orgId: ID!) {
  SharedWithMe(orgId: $orgId) { id resourceType name role sharedBy projectId projectName provenance }
}`,
	"SharedByMe": `query CLIPlatformSharedByMe($orgId: ID!) {
  SharedByMe(orgId: $orgId) { id resourceType name role sharedBy projectId projectName provenance }
}`,
	"MyGrants": `query CLIPlatformMyGrants($orgId: ID!) {
  MyGrants(orgId: $orgId) { resourceType resourceId resourceName role projectId projectName provenance }
}`,
	"ProjectResourceSummary": `query CLIPlatformProjectResourceSummary($projectId: ID!) {
  ProjectResourceSummary(projectId: $projectId) { sources { id name } datasets { id name } transforms { id name } functions { id name } apps { id name } ontologyTypes { id name } files { id name } }
}`,
	"ProjectAccessMatrix": `query CLIPlatformProjectAccessMatrix($projectId: ID!) {
  ProjectAccessMatrix(projectId: $projectId) { projectId projectName resources { resourceType resourceId resourceName accessList { subject displayName role } } }
}`,
}

// PlatformQuery executes one allow-listed hosted platform query and returns its
// decoded data object. Workspace identifiers are injected by the MCP layer.
func (c *Client) PlatformQuery(ctx context.Context, operation string, variables map[string]any) (any, error) {
	query, ok := platformQuerySpecs[operation]
	if !ok {
		return nil, fmt.Errorf("unsupported platform query %q", operation)
	}
	if err := c.RequireOperation("Query", operation, "platform read"); err != nil {
		return nil, err
	}
	var response map[string]json.RawMessage
	if err := c.graphQLWithRetry(ctx, func() string { return c.platformReadDocument(operation, query) }, variables, &response); err != nil {
		return nil, err
	}
	raw, ok := response[operation]
	if !ok {
		return nil, fmt.Errorf("platform returned no %s result", operation)
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("decode %s result: %w", operation, err)
	}
	return data, nil
}
