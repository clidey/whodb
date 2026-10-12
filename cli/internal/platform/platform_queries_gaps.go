/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package platform

// Allow-listed read operations for service accounts, lineage providers,
// ontology sharing, admin visibility and package polish. Exposed through
// Client.PlatformQuery and the MCP extended read tools.
func init() {
	queries := map[string]string{
		"ServiceAccounts": `query CLIPlatformServiceAccounts($orgId: ID!) {
  ServiceAccounts(orgId: $orgId) { id orgId name description createdBy createdAt disabledAt keys { id keyPrefix createdAt expiresAt revokedAt lastUsedAt } }
}`,
		"ProjectLineageProviders": `query CLIPlatformProjectLineageProviders($projectId: ID!) {
  ProjectLineageProviders(projectId: $projectId) { id projectId name endpoint readEndpoint namespace authType enabled createdAt updatedAt }
}`,
		"SharedOntologies": `query CLIPlatformSharedOntologies($projectId: ID!) {
  SharedOntologies(projectId: $projectId) { id projectId ontologyId visibility ontology { id apiName displayName description } consumers { projectId referenceId alias status } createdAt }
}`,
		"DiscoverSharedOntologies": `query CLIPlatformDiscoverSharedOntologies($projectId: ID!) {
  DiscoverSharedOntologies(consumerProjectId: $projectId) { id projectId ontologyId visibility ontology { id apiName displayName description } consumers { projectId referenceId alias status } createdAt }
}`,
		"OntologyReferences": `query CLIPlatformOntologyReferences($projectId: ID!) {
  OntologyReferences(projectId: $projectId) { id consumerProjectId alias ownerProjectId ontologyId status ontology { id apiName displayName description } createdAt }
}`,
		"OntologyReferenceImpact": `query CLIPlatformOntologyReferenceImpact($referenceId: ID!) {
  OntologyReferenceImpact(referenceId: $referenceId) { referenceId dependentLinks dependentApps dependentFuncs dependentXfms dependentBehavs }
}`,
		"UserOwnedResources": `query CLIPlatformUserOwnedResources($userId: ID!, $orgId: ID!) {
  UserOwnedResources(userId: $userId, orgId: $orgId) { id resourceType name owner createdAt }
}`,
		"UserResourceAccess": `query CLIPlatformUserResourceAccess($userId: ID!, $orgId: ID!) {
  UserResourceAccess(userId: $userId, orgId: $orgId) { resourceType resourceId resourceName role projectId projectName provenance }
}`,
		"ShareDependents": `query CLIPlatformShareDependents($resourceType: String!, $resourceId: ID!) {
  ShareDependents(resourceType: $resourceType, resourceId: $resourceId) { resourceType resourceId resourceName defaultRole reason grantable }
}`,
		"OrganizationPackageInstallations": `query CLIPlatformOrganizationPackageInstallations($organizationId: ID!, $packageId: ID) {
  OrganizationPackageInstallations(organizationId: $organizationId, packageId: $packageId) { id sourceOrgId sourceProjectId targetProjectId packageId packageName packageVersion packageChannel status installedBy installedAt updatedAt updateInfo { available latestVersion } }
}`,
		"AppPageFiles": `query CLIPlatformAppPageFiles($projectId: ID!, $id: ID!, $env: String, $page: String!) {
  AppPageFiles(projectId: $projectId, id: $id, env: $env, page: $page) { path content updatedAt }
}`,
	}
	for name, query := range queries {
		platformQuerySpecs[name] = query
	}
}
