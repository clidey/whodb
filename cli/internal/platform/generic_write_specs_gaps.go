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

// Write specs for bulk ontology inserts, run cancellation, project and file
// restore/purge, embeddings, ontology sharing, lineage providers, service
// accounts, function test input generation, team rename and package polish.
// Each entry pairs a GenericWriteSpec with its PayloadShape and mutation
// document so the generic MCP write tools and `whodb resources action` expose
// it without new tool registrations.

const sharedOntologyFields = `
  id
  projectId
  ontologyId
  visibility
  consumers { projectId referenceId alias status }
  createdAt
`

const ontologyReferenceFields = `
  id
  consumerProjectId
  alias
  ownerProjectId
  ontologyId
  status
  createdAt
`

const lineageProviderFields = `
  id
  projectId
  name
  endpoint
  readEndpoint
  namespace
  authType
  enabled
  createdAt
  updatedAt
`

const serviceAccountFields = `
  id
  orgId
  name
  description
  createdBy
  createdAt
  disabledAt
  keys { id keyPrefix createdAt expiresAt revokedAt lastUsedAt }
`

const ontologyEmbeddingJobFields = `
  id
  entityId
  status
  model
  dimensions
  batchSize
  totalRows
  processedRows
  errorMessage
  createdAt
  startedAt
  completedAt
`

func init() {
	specs := map[string]GenericWriteSpec{
		"action:add_records:ontology":                    {Resource: "ontology", Action: "add_records", Mutation: "OntologyAddRows", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "entityId", InjectProjectID: true},
		"action:purge_data:ontology":                     {Resource: "ontology", Action: "purge_data", Mutation: "OntologyPurgeData", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "entityId", InjectProjectID: true},
		"action:reembed:ontology":                        {Resource: "ontology", Action: "reembed", Mutation: "ReembedOntology", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "entityId", InjectProjectID: true},
		"action:share:ontology":                          {Resource: "ontology", Action: "share", Mutation: "ShareOntology", Mode: GenericWriteModeInput, NeedsID: true, IdentityField: "ontologyId", InjectProjectID: true},
		"action:unshare:ontology":                        {Resource: "ontology", Action: "unshare", Mutation: "UnshareOntology", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "ontologyId", InjectProjectID: true},
		"create:ontology_reference":                      {Resource: "ontology_reference", Action: "create", Mutation: "CreateOntologyReference", Mode: GenericWriteModeDirect, InjectProjectID: true},
		"delete:ontology_reference":                      {Resource: "ontology_reference", Action: "delete", Mutation: "RemoveOntologyReference", Mode: GenericWriteModeID, NeedsID: true},
		"action:purge_data:dataset":                      {Resource: "dataset", Action: "purge_data", Mutation: "DatasetPurgeData", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "datasetId", InjectProjectID: true},
		"action:cancel:function_run":                     {Resource: "function_run", Action: "cancel", Mutation: "CancelFunctionRun", Mode: GenericWriteModeProjectID, NeedsID: true},
		"action:generate_test_input:function":            {Resource: "function", Action: "generate_test_input", Mutation: "GenerateFunctionTestInput", Mode: GenericWriteModeInput, NeedsID: true, IdentityField: "functionId", InjectProjectID: true},
		"action:cancel_run:transform":                    {Resource: "transform", Action: "cancel_run", Mutation: "CancelTransformRun", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "transformId", InjectProjectID: true},
		"action:cancel_generation:app":                   {Resource: "app", Action: "cancel_generation", Mutation: "CancelAppGeneration", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "appId"},
		"action:restore:project":                         {Resource: "project", Action: "restore", Mutation: "RestoreProject", Mode: GenericWriteModeDirect, NeedsID: true, InjectOrgID: true},
		"action:purge:project":                           {Resource: "project", Action: "purge", Mutation: "PurgeProject", Mode: GenericWriteModeDirect, NeedsID: true, InjectOrgID: true},
		"action:restore:file":                            {Resource: "file", Action: "restore", Mutation: "RestoreProjectFile", Mode: GenericWriteModeProjectID, NeedsID: true},
		"action:purge:file":                              {Resource: "file", Action: "purge", Mutation: "PurgeProjectFile", Mode: GenericWriteModeProjectID, NeedsID: true},
		"action:restore:folder":                          {Resource: "folder", Action: "restore", Mutation: "RestoreProjectFolder", Mode: GenericWriteModeProjectID, NeedsID: true},
		"action:purge:folder":                            {Resource: "folder", Action: "purge", Mutation: "PurgeProjectFolder", Mode: GenericWriteModeProjectID, NeedsID: true},
		"create:lineage_provider":                        {Resource: "lineage_provider", Action: "create", Mutation: "SaveLineageProvider", Mode: GenericWriteModeInput, InjectProjectID: true},
		"update:lineage_provider":                        {Resource: "lineage_provider", Action: "update", Mutation: "SaveLineageProvider", Mode: GenericWriteModeInput, NeedsID: true, InjectProjectID: true},
		"delete:lineage_provider":                        {Resource: "lineage_provider", Action: "delete", Mutation: "DeleteLineageProvider", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "providerId", InjectProjectID: true},
		"action:create_service_account:organization":     {Resource: "organization", Action: "create_service_account", Mutation: "CreateServiceAccount", Mode: GenericWriteModeDirect, InjectOrgID: true},
		"action:delete_service_account:organization":     {Resource: "organization", Action: "delete_service_account", Mutation: "DeleteServiceAccount", Mode: GenericWriteModeDirect, NeedsID: true, InjectOrgID: true},
		"action:create_service_account_key:organization": {Resource: "organization", Action: "create_service_account_key", Mutation: "CreateServiceAccountKey", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "serviceAccountId", InjectOrgID: true},
		"action:revoke_service_account_key:organization": {Resource: "organization", Action: "revoke_service_account_key", Mutation: "RevokeServiceAccountKey", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "keyId", InjectOrgID: true},
		"action:rename_team:organization":                {Resource: "organization", Action: "rename_team", Mutation: "RenameTeam", Mode: GenericWriteModeDirect, NeedsID: true, IdentityField: "teamId"},
		"action:add_changelog:package":                   {Resource: "package", Action: "add_changelog", Mutation: "AddPackageChangelog", Mode: GenericWriteModeInput, NeedsID: true, IdentityField: "packageId"},
		"action:add_media:package":                       {Resource: "package", Action: "add_media", Mutation: "AddPackageMedia", Mode: GenericWriteModeInput, NeedsID: true, IdentityField: "packageId"},
		"action:review:package":                          {Resource: "package", Action: "review", Mutation: "ReviewPackage", Mode: GenericWriteModeInput, NeedsID: true, IdentityField: "packageId"},
	}
	for key, spec := range specs {
		GenericWriteSpecs[key] = spec
	}

	orgID := PayloadField{Name: "orgId", Type: "string", Description: "Organization id; defaults to the selected organization"}
	shapes := []PayloadShape{
		{
			Key: "action:add_records:ontology", Resource: "ontology", Action: "add_records", Description: "Insert many ontology rows in one call. id becomes entityId and projectId is injected. Pass idempotencyKey so a retry does not insert twice. Prefer whodb_platform_add_ontology_records, which accepts plain objects.",
			Fields: []PayloadField{
				{Name: "rows", Type: "array", Required: true, Description: "Rows as [{\"values\":[{\"Key\":\"prop\",\"Value\":\"text\"}]}]"},
				{Name: "idempotencyKey", Type: "string", Description: "Client-chosen key that makes the insert replay-safe"},
			},
			Examples: []string{`{"rows":[{"values":[{"Key":"name","Value":"Ada"}]},{"values":[{"Key":"name","Value":"Grace"}]}],"idempotencyKey":"import-2026-10-12"}`},
		},
		{Key: "action:purge_data:ontology", Resource: "ontology", Action: "purge_data", Description: "Permanently delete every row of an ontology while keeping its definition. id becomes entityId and projectId is injected. Irreversible."},
		{
			Key: "action:reembed:ontology", Resource: "ontology", Action: "reembed", Description: "Start an embedding backfill for an ontology with embeddingEnabled. id becomes entityId and projectId is injected. Track it with whodb_platform_ontology_embedding_jobs.",
			Fields:   []PayloadField{{Name: "batchSize", Type: "integer", Description: "Rows embedded per batch (default 4)"}},
			Examples: []string{`{"batchSize":16}`},
		},
		{
			Key: "action:share:ontology", Resource: "ontology", Action: "share", Description: "Share an ontology with other projects in the organization. id becomes ontologyId and projectId is injected.",
			Fields: []PayloadField{
				{Name: "visibility", Type: "string", Required: true, Description: "organization or allowlist"},
				{Name: "allowedProjectIds", Type: "array", Description: "Project ids allowed to reference the ontology when visibility is allowlist"},
			},
			Examples: []string{`{"visibility":"organization"}`, `{"visibility":"allowlist","allowedProjectIds":["project-id"]}`},
		},
		{Key: "action:unshare:ontology", Resource: "ontology", Action: "unshare", Description: "Stop sharing an ontology. Existing references in other projects become broken. id becomes ontologyId and projectId is injected."},
		{
			Key: "create:ontology_reference", Resource: "ontology_reference", Action: "create", Description: "Reference a shared ontology from the selected project under an alias. The selected project becomes consumerProjectId. Discover candidates with whodb_platform_discover_shared_ontologies.",
			Fields: []PayloadField{
				{Name: "ownerProjectId", Type: "string", Required: true, Description: "Project that owns the shared ontology"},
				{Name: "ontologyId", Type: "string", Required: true, Description: "Shared ontology id"},
				{Name: "alias", Type: "string", Required: true, Description: "Local apiName for the referenced ontology"},
			},
			Examples: []string{`{"ownerProjectId":"owner-project-id","ontologyId":"ontology-id","alias":"shared_customers"}`},
		},
		{Key: "delete:ontology_reference", Resource: "ontology_reference", Action: "delete", Description: "Remove an ontology reference. id is the reference id. Check whodb_platform_ontology_reference_impact first."},
		{Key: "action:purge_data:dataset", Resource: "dataset", Action: "purge_data", Description: "Permanently delete every row of a dataset while keeping its definition. id becomes datasetId and projectId is injected. Irreversible."},
		{Key: "action:cancel:function_run", Resource: "function_run", Action: "cancel", Description: "Request cancellation of a queued or running function run. id is the run id; projectId is injected. Poll whodb_platform_function_wait for CANCELED."},
		{
			Key: "action:generate_test_input:function", Resource: "function", Action: "generate_test_input", Description: "Ask an AI provider to draft a test input for a function. id becomes functionId and projectId is injected.",
			Fields: []PayloadField{
				{Name: "ModelType", Type: "string", Required: true, Description: "Provider type, for example OpenAI"},
				{Name: "Model", Type: "string", Required: true, Description: "Model name"},
				{Name: "ProviderId", Type: "string", Description: "Hosted AI provider id"},
				{Name: "Token", Type: "string", Secret: true, Description: "Optional provider token when no hosted provider is used"},
			},
			Examples: []string{`{"ModelType":"OpenAI","Model":"gpt-4o-mini","ProviderId":"provider-id"}`},
		},
		{
			Key: "action:cancel_run:transform", Resource: "transform", Action: "cancel_run", Description: "Cancel the running transform run. id becomes transformId and projectId is injected; runId defaults to the active run.",
			Fields:   []PayloadField{{Name: "runId", Type: "string", Description: "Specific run id to cancel"}},
			Examples: []string{`{}`, `{"runId":"run-id"}`},
		},
		{Key: "action:cancel_generation:app", Resource: "app", Action: "cancel_generation", Description: "Stop an in-progress AI app generation. id becomes appId."},
		{
			Key: "action:restore:project", Resource: "project", Action: "restore", Description: "Restore a soft-deleted project. id is the project id. Find candidates with whodb_platform_deleted_resources.",
			Fields: []PayloadField{{Name: "orgId", Type: "string", Description: "Organization id; defaults to the selected organization"}},
		},
		{
			Key: "action:purge:project", Resource: "project", Action: "purge", Description: "Permanently delete a soft-deleted project and all of its data. id is the project id. Irreversible.",
			Fields: []PayloadField{{Name: "orgId", Type: "string", Description: "Organization id; defaults to the selected organization"}},
		},
		{Key: "action:restore:file", Resource: "file", Action: "restore", Description: "Restore a soft-deleted project file. id is the file id; projectId is injected."},
		{Key: "action:purge:file", Resource: "file", Action: "purge", Description: "Permanently delete a soft-deleted project file. id is the file id; projectId is injected. Irreversible."},
		{Key: "action:restore:folder", Resource: "folder", Action: "restore", Description: "Restore a soft-deleted project folder. id is the folder id; projectId is injected."},
		{Key: "action:purge:folder", Resource: "folder", Action: "purge", Description: "Permanently delete a soft-deleted project folder. id is the folder id; projectId is injected. Irreversible."},
		{
			Key: "create:lineage_provider", Resource: "lineage_provider", Action: "create", Description: "Register an OpenLineage provider for the selected project. projectId is injected.",
			Fields: []PayloadField{
				{Name: "name", Type: "string", Required: true, Description: "Provider name"},
				{Name: "endpoint", Type: "string", Required: true, Description: "OpenLineage write endpoint URL"},
				{Name: "readEndpoint", Type: "string", Required: true, Description: "OpenLineage read endpoint URL"},
				{Name: "namespace", Type: "string", Required: true, Description: "OpenLineage namespace"},
				{Name: "authType", Type: "string", Required: true, Description: "none or api_key"},
				{Name: "apiKey", Type: "string", Secret: true, Description: "API key when authType is api_key"},
				{Name: "enabled", Type: "boolean", Required: true, Description: "Whether the provider is active"},
			},
			Examples: []string{`{"name":"Marquez","endpoint":"https://marquez.example.com/api/v1/lineage","readEndpoint":"https://marquez.example.com/api/v1","namespace":"whodb","authType":"none","enabled":true}`},
		},
		{
			Key: "update:lineage_provider", Resource: "lineage_provider", Action: "update", Description: "Update an OpenLineage provider. id and projectId are injected; send the full provider definition.",
			Fields: []PayloadField{
				{Name: "name", Type: "string", Required: true, Description: "Provider name"},
				{Name: "endpoint", Type: "string", Required: true, Description: "OpenLineage write endpoint URL"},
				{Name: "readEndpoint", Type: "string", Required: true, Description: "OpenLineage read endpoint URL"},
				{Name: "namespace", Type: "string", Required: true, Description: "OpenLineage namespace"},
				{Name: "authType", Type: "string", Required: true, Description: "none or api_key"},
				{Name: "apiKey", Type: "string", Secret: true, Description: "API key when authType is api_key"},
				{Name: "enabled", Type: "boolean", Required: true, Description: "Whether the provider is active"},
			},
		},
		{Key: "delete:lineage_provider", Resource: "lineage_provider", Action: "delete", Description: "Delete an OpenLineage provider. id becomes providerId and projectId is injected."},
		{
			Key: "action:create_service_account:organization", Resource: "organization", Action: "create_service_account", Description: "Create a service account for API access. Then create a key with action create_service_account_key.",
			Fields: []PayloadField{
				orgID,
				{Name: "name", Type: "string", Required: true, Description: "Service account name"},
				{Name: "description", Type: "string", Description: "Purpose of the account"},
			},
			Examples: []string{`{"orgId":"org-id","name":"ci-deployer","description":"Deploys packages from CI"}`},
		},
		{
			Key: "action:delete_service_account:organization", Resource: "organization", Action: "delete_service_account", Description: "Delete a service account and revoke all of its keys. id is the service account id.",
			Fields: []PayloadField{orgID},
		},
		{
			Key: "action:create_service_account_key:organization", Resource: "organization", Action: "create_service_account_key", Description: "Issue an API key for a service account. id becomes serviceAccountId. The response rawKey is shown exactly once; hand it to the user and never log or store it.",
			Fields: []PayloadField{
				orgID,
				{Name: "expiresAt", Type: "string", Description: "RFC 3339 expiry; omit for no expiry"},
			},
			Examples: []string{`{"orgId":"org-id","expiresAt":"2027-01-01T00:00:00Z"}`},
		},
		{
			Key: "action:revoke_service_account_key:organization", Resource: "organization", Action: "revoke_service_account_key", Description: "Revoke one service account key. id becomes keyId.",
			Fields: []PayloadField{orgID},
		},
		{
			Key: "action:rename_team:organization", Resource: "organization", Action: "rename_team", Description: "Rename a team. id becomes teamId.",
			Fields:   []PayloadField{{Name: "name", Type: "string", Required: true, Description: "New team name"}},
			Examples: []string{`{"name":"Data platform"}`},
		},
		{
			Key: "action:add_changelog:package", Resource: "package", Action: "add_changelog", Description: "Add a changelog entry to a package version. id becomes packageId.",
			Fields: []PayloadField{
				{Name: "version", Type: "string", Required: true, Description: "Package version the entry describes"},
				{Name: "title", Type: "string", Required: true, Description: "Short title"},
				{Name: "body", Type: "string", Description: "Markdown body"},
			},
			Examples: []string{`{"version":"1.2.0","title":"Adds invoice ontology","body":"- new ontology\n- fixed transform"}`},
		},
		{
			Key: "action:add_media:package", Resource: "package", Action: "add_media", Description: "Attach a screenshot or video URL to a package. id becomes packageId.",
			Fields: []PayloadField{
				{Name: "kind", Type: "string", Required: true, Description: "screenshot or video"},
				{Name: "url", Type: "string", Required: true, Description: "Public media URL"},
				{Name: "caption", Type: "string", Description: "Caption"},
			},
			Examples: []string{`{"kind":"screenshot","url":"https://example.com/shot.png","caption":"Dashboard"}`},
		},
		{
			Key: "action:review:package", Resource: "package", Action: "review", Description: "Leave a rating and review on a package. id becomes packageId.",
			Fields: []PayloadField{
				{Name: "rating", Type: "integer", Required: true, Description: "1 to 5"},
				{Name: "body", Type: "string", Description: "Review text"},
			},
			Examples: []string{`{"rating":5,"body":"Installed cleanly."}`},
		},
	}
	for _, shape := range shapes {
		PayloadShapes[shape.Key] = shape
	}

	mutations := map[string]platformMutationSpec{
		"OntologyAddRows":           mutationSpecWithDirect("OntologyAddRows", "$projectId: ID!, $entityId: ID!, $rows: [OntologyCreateRowInput!]!, $idempotencyKey: String", "projectId: $projectId, entityId: $entityId, rows: $rows, idempotencyKey: $idempotencyKey", "inserted ids"),
		"OntologyPurgeData":         mutationSpecWithDirect("OntologyPurgeData", "$projectId: ID!, $entityId: ID!", "projectId: $projectId, entityId: $entityId", statusResponseFields),
		"ReembedOntology":           mutationSpecWithDirect("ReembedOntology", "$projectId: ID!, $entityId: ID!, $batchSize: Int", "projectId: $projectId, entityId: $entityId, batchSize: $batchSize", ontologyEmbeddingJobFields),
		"ShareOntology":             mutationSpecWithInput("ShareOntology", "ShareOntologyInput", sharedOntologyFields),
		"UnshareOntology":           mutationSpecWithDirect("UnshareOntology", "$projectId: ID!, $ontologyId: ID!", "projectId: $projectId, ontologyId: $ontologyId", statusResponseFields),
		"CreateOntologyReference":   mutationSpecWithDirect("CreateOntologyReference", "$projectId: ID!, $ownerProjectId: ID!, $ontologyId: ID!, $alias: String!", "input: {consumerProjectId: $projectId, ownerProjectId: $ownerProjectId, ontologyId: $ontologyId, alias: $alias}", ontologyReferenceFields),
		"RemoveOntologyReference":   mutationSpecWithDirect("RemoveOntologyReference", "$id: ID!", "referenceId: $id", statusResponseFields),
		"DatasetPurgeData":          mutationSpecWithDirect("DatasetPurgeData", "$projectId: ID!, $datasetId: ID!", "projectId: $projectId, datasetId: $datasetId", statusResponseFields),
		"CancelFunctionRun":         mutationSpecWithProjectID("CancelFunctionRun", functionRunFields),
		"GenerateFunctionTestInput": mutationSpecWithInput("GenerateFunctionTestInput", "GenerateFunctionTestInputInput", "json"),
		"CancelTransformRun":        mutationSpecWithDirect("CancelTransformRun", "$projectId: ID!, $transformId: ID!, $runId: ID", "projectId: $projectId, transformId: $transformId, runId: $runId", statusResponseFields),
		"CancelAppGeneration":       mutationSpecWithDirect("CancelAppGeneration", "$appId: ID!", "appId: $appId", statusResponseFields),
		"RestoreProject":            mutationSpecWithDirect("RestoreProject", "$id: ID!, $orgId: ID", "id: $id, orgId: $orgId", statusResponseFields),
		"PurgeProject":              mutationSpecWithDirect("PurgeProject", "$id: ID!, $orgId: ID", "id: $id, orgId: $orgId", statusResponseFields),
		"RestoreProjectFile":        mutationSpecWithProjectID("RestoreProjectFile", statusResponseFields),
		"PurgeProjectFile":          mutationSpecWithProjectID("PurgeProjectFile", statusResponseFields),
		"RestoreProjectFolder":      mutationSpecWithProjectID("RestoreProjectFolder", statusResponseFields),
		"PurgeProjectFolder":        mutationSpecWithProjectID("PurgeProjectFolder", statusResponseFields),
		"SaveLineageProvider":       mutationSpecWithInput("SaveLineageProvider", "SaveLineageProviderInput", lineageProviderFields),
		"DeleteLineageProvider":     mutationSpecWithDirect("DeleteLineageProvider", "$projectId: ID!, $providerId: ID!", "projectId: $projectId, providerId: $providerId", statusResponseFields),
		"CreateServiceAccount":      mutationSpecWithDirect("CreateServiceAccount", "$orgId: ID!, $name: String!, $description: String", "orgId: $orgId, name: $name, description: $description", serviceAccountFields),
		"DeleteServiceAccount":      mutationSpecWithDirect("DeleteServiceAccount", "$orgId: ID!, $id: ID!", "orgId: $orgId, id: $id", statusResponseFields),
		"CreateServiceAccountKey":   mutationSpecWithDirect("CreateServiceAccountKey", "$orgId: ID!, $serviceAccountId: ID!, $expiresAt: String", "orgId: $orgId, serviceAccountId: $serviceAccountId, expiresAt: $expiresAt", "key { id keyPrefix createdAt expiresAt revokedAt lastUsedAt } rawKey"),
		"RevokeServiceAccountKey":   mutationSpecWithDirect("RevokeServiceAccountKey", "$orgId: ID!, $keyId: ID!", "orgId: $orgId, keyId: $keyId", statusResponseFields),
		"RenameTeam":                mutationSpecWithDirect("RenameTeam", "$teamId: ID!, $name: String!", "teamId: $teamId, name: $name", statusResponseFields),
		"AddPackageChangelog":       mutationSpecWithInput("AddPackageChangelog", "AddPackageChangelogInput", "id version title body createdAt"),
		"AddPackageMedia":           mutationSpecWithInput("AddPackageMedia", "AddPackageMediaInput", "id kind url caption sortOrder createdAt"),
		"ReviewPackage":             mutationSpecWithInput("ReviewPackage", "ReviewPackageInput", "id rating body createdBy createdAt"),
	}
	for name, spec := range mutations {
		platformMutationSpecs[name] = spec
	}
}
