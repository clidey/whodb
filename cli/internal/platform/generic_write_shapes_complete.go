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

// Payload shapes for every remaining write spec so whodb://platform/schema
// documents each action, required fields are validated before a confirmation
// preview is issued, and agents get an example for every write.
func init() {
	str := func(name, description string) PayloadField {
		return PayloadField{Name: name, Type: "string", Description: description}
	}
	req := func(name, description string) PayloadField {
		return PayloadField{Name: name, Type: "string", Required: true, Description: description}
	}
	orgID := str("orgId", "Organization id; defaults to the selected organization")
	idOnly := func(key, resource, action, description string) PayloadShape {
		return PayloadShape{Key: key, Resource: resource, Action: action, Description: description}
	}
	functionFiles := PayloadField{Name: "files", Type: "array", Description: "Function files as [{\"path\":\"main.py\",\"content\":\"...\"}]"}
	functionDependencies := PayloadField{Name: "dependencies", Type: "array", Description: "Dependencies as [{\"name\":\"requests\",\"version\":\"2.32\"}]"}
	functionOptional := []PayloadField{
		str("description", "Function description"),
		str("language", "python, javascript, r, julia, rust, or go"),
		str("entryPoint", "Entry function name"),
		{Name: "timeoutSeconds", Type: "integer", Description: "Execution timeout"},
		str("memory", "Memory limit, for example 512Mi"),
		str("cpu", "CPU limit, for example 500m"),
		functionFiles,
		functionDependencies,
		{Name: "providerIds", Type: "array", Description: "AI provider ids the function may use"},
		{Name: "ontologyIds", Type: "array", Description: "Ontologies the function may read and write"},
		{Name: "readOnlyOntologyIds", Type: "array", Description: "Ontologies the function may only read"},
		{Name: "providerConfigs", Type: "array", Description: "Provider defaults as [{\"providerId\":\"...\",\"model\":\"gpt-4o-mini\"}]"},
		{Name: "secretBindings", Type: "array", Description: "Secrets exposed to the function as [{\"name\":\"API_KEY\",\"secretId\":\"...\",\"target\":\"ENV\"}]"},
		{Name: "defaultMaxTokens", Type: "integer", Description: "Default max tokens for provider calls"},
		{Name: "defaultTemperature", Type: "number", Description: "Default temperature for provider calls"},
		str("scheduleCron", "Five-field cron schedule for automatic runs; empty string clears it"),
		str("scheduleInput", "JSON object passed as input to every scheduled run"),
	}
	bindings := PayloadField{Name: "bindings", Type: "array", Description: "Requirement bindings as [{\"key\":\"source\",\"targetId\":\"...\"}] or [{\"key\":\"region\",\"value\":\"EU\"}]"}

	shapes := []PayloadShape{
		// Organization administration.
		{Key: "action:create_team:organization", Resource: "organization", Action: "create_team", Description: "Create a team in the organization.", Fields: []PayloadField{orgID, req("name", "Team name")}, Examples: []string{`{"name":"Data platform"}`}},
		idOnly("action:delete_team:organization", "organization", "delete_team", "Delete a team. id is the team id."),
		{Key: "action:add_team_member:organization", Resource: "organization", Action: "add_team_member", Description: "Add a member to a team.", Fields: []PayloadField{req("teamId", "Team id"), req("userId", "User id from whodb_platform_org_members")}, Examples: []string{`{"teamId":"team-id","userId":"user-id"}`}},
		{Key: "action:remove_team_member:organization", Resource: "organization", Action: "remove_team_member", Description: "Remove a member from a team.", Fields: []PayloadField{req("teamId", "Team id"), req("userId", "User id")}, Examples: []string{`{"teamId":"team-id","userId":"user-id"}`}},
		{Key: "action:rename:organization", Resource: "organization", Action: "rename", Description: "Rename the organization and optionally change its slug.", Fields: []PayloadField{orgID, req("name", "New organization name"), str("slug", "New URL slug")}, Examples: []string{`{"name":"Acme Analytics"}`}},
		{Key: "action:delete:organization", Resource: "organization", Action: "delete", Description: "Delete the organization and everything in it. Irreversible.", Fields: []PayloadField{orgID, {Name: "confirmDeletion", Type: "boolean", Description: "Must be true to delete an organization with projects"}}, Examples: []string{`{"confirmDeletion":true}`}},
		{Key: "action:upsert_domain:organization", Resource: "organization", Action: "upsert_domain", Description: "Add or update a verified email domain for the organization.", Fields: []PayloadField{orgID, req("domain", "Email domain, for example acme.com"), {Name: "isPrimary", Type: "boolean", Description: "Make this the primary domain"}}, Examples: []string{`{"domain":"acme.com","isPrimary":true}`}},
		idOnly("action:delete_domain:organization", "organization", "delete_domain", "Remove an organization domain. id is the domain id from whodb_platform_org_domains."),
		{Key: "action:create_sso:organization", Resource: "organization", Action: "create_sso", Description: "Register an SSO identity provider for the organization.", Fields: []PayloadField{orgID, req("providerType", "oidc or saml"), req("displayName", "Name shown on the login page"), req("providerAlias", "Identity provider alias in the auth server"), {Name: "enabled", Type: "boolean", Description: "Enable immediately"}, {Name: "isDefault", Type: "boolean", Description: "Use as the default login"}}, Examples: []string{`{"providerType":"oidc","displayName":"Okta","providerAlias":"okta","enabled":true}`}},
		{Key: "action:update_sso:organization", Resource: "organization", Action: "update_sso", Description: "Update an SSO provider. id is the provider id from whodb_platform_org_sso.", Fields: []PayloadField{str("providerType", "oidc or saml"), str("displayName", "Name shown on the login page"), str("providerAlias", "Identity provider alias"), {Name: "enabled", Type: "boolean", Description: "Enable or disable"}, {Name: "isDefault", Type: "boolean", Description: "Use as the default login"}}, Examples: []string{`{"enabled":false}`}},
		idOnly("action:delete_sso:organization", "organization", "delete_sso", "Remove an SSO provider. id is the provider id."),
		{Key: "action:reinvite_user:organization", Resource: "organization", Action: "reinvite_user", Description: "Resend an organization invitation.", Fields: []PayloadField{orgID, req("userId", "Invited user id")}, Examples: []string{`{"userId":"user-id"}`}},
		{Key: "action:remove_user:organization", Resource: "organization", Action: "remove_user", Description: "Remove a member from the organization. Check whodb_platform_user_owned_resources first; force transfers nothing and fails if the member still owns resources unless true.", Fields: []PayloadField{orgID, req("userId", "User id"), {Name: "force", Type: "boolean", Description: "Remove even if the member owns resources"}}, Examples: []string{`{"userId":"user-id"}`}},
		idOnly("action:sync_app_permissions:organization", "organization", "sync_app_permissions", "Recompute app permission grants across the organization. No payload."),
		// Ownership.
		{Key: "action:transfer_ownership:resource", Resource: "resource", Action: "transfer_ownership", Description: "Transfer ownership of a resource to another member.", Fields: []PayloadField{req("resourceType", "dataset, app, function, transform, ontology, source, or secret"), req("resourceId", "Resource id"), req("newOwnerId", "User id of the new owner")}, Examples: []string{`{"resourceType":"dataset","resourceId":"dataset-id","newOwnerId":"user-id"}`}},
		{Key: "action:transfer_ownership_by_email:resource", Resource: "resource", Action: "transfer_ownership_by_email", Description: "Transfer ownership of a resource to a member identified by email.", Fields: []PayloadField{req("email", "New owner's email"), req("resourceType", "Resource type"), req("resourceId", "Resource id")}, Examples: []string{`{"email":"owner@acme.test","resourceType":"app","resourceId":"app-id"}`}},
		{Key: "action:claim_orphan:resource", Resource: "resource", Action: "claim_orphan", Description: "Claim ownership of a resource whose owner left the organization.", Fields: []PayloadField{req("resourceType", "Resource type"), req("resourceId", "Resource id")}, Examples: []string{`{"resourceType":"dataset","resourceId":"dataset-id"}`}},
		// Functions.
		{Key: "create:function", Resource: "function", Action: "create", Description: "Create a function. projectId is injected.", Fields: append([]PayloadField{req("name", "Function name"), req("description", "Function description"), req("language", "python, javascript, r, julia, rust, or go"), req("entryPoint", "Entry function name"), {Name: "files", Type: "array", Required: true, Description: "Function files as [{\"path\":\"main.py\",\"content\":\"...\"}]"}, {Name: "dependencies", Type: "array", Required: true, Description: "Dependencies as [{\"name\":\"requests\",\"version\":\"2.32\"}]; use [] for none"}, str("executionMode", "SYNC, ASYNC, or LIVE (default SYNC)"), str("liveBehavior", "REQUEST or SERVICE for LIVE functions")}, functionOptional[3:]...), Examples: []string{`{"name":"normalize","description":"Normalize names","language":"python","entryPoint":"main","files":[{"path":"main.py","content":"def main(input):\n    return input\n"}],"dependencies":[]}`}},
		{Key: "update:function", Resource: "function", Action: "update", Description: "Update a function. id and projectId are injected; send only changed fields.", Fields: append([]PayloadField{str("name", "Function name")}, functionOptional...)},
		{Key: "action:test:function", Resource: "function", Action: "test", Description: "Run a function once with test input without recording a persisted run. projectId is injected.", Fields: []PayloadField{req("functionId", "Function id"), req("input", "JSON input as a string"), functionFiles, {Name: "inputFileIds", Type: "array", Description: "Project file ids to mount as inputs"}}, Examples: []string{`{"functionId":"function-id","input":"{\"name\":\"Ada\"}"}`}},
		{Key: "action:preview:function", Resource: "function", Action: "preview", Description: "Run inline function code that is not saved yet. projectId is injected.", Fields: []PayloadField{req("language", "Runtime language"), req("entryPoint", "Entry function name"), req("input", "JSON input as a string"), {Name: "files", Type: "array", Required: true, Description: "Files as [{\"path\":\"main.py\",\"content\":\"...\"}]"}, {Name: "dependencies", Type: "array", Description: "Dependency names"}}, Examples: []string{`{"language":"python","entryPoint":"main","input":"{}","files":[{"path":"main.py","content":"def main(input):\n    return input\n"}]}`}},
		{Key: "action:execute:function", Resource: "function", Action: "execute", Description: "Execute a deployed function synchronously and return its output. projectId is injected. Prefer start_run for work that may outlast one request.", Fields: []PayloadField{req("functionId", "Function id"), req("input", "JSON input as a string"), {Name: "inputFileIds", Type: "array", Description: "Project file ids to mount as inputs"}}, Examples: []string{`{"functionId":"function-id","input":"{}"}`}},
		// Transforms.
		{Key: "create:transform", Resource: "transform", Action: "create", Description: "Create a transform from a pipeline graph. projectId is injected. Validate the graph first with whodb_platform_transform_preview.", Fields: []PayloadField{req("name", "Transform name"), str("description", "Transform description"), req("graphJson", "Pipeline graph JSON with nodes and edges"), str("scheduleCron", "Cron schedule; empty for none"), str("triggerMode", "manual (default) or scheduled")}, Examples: []string{`{"name":"nightly-rollup","description":"Roll up orders","graphJson":"{\"nodes\":[],\"edges\":[]}","scheduleCron":"","triggerMode":"manual"}`}},
		{Key: "update:transform", Resource: "transform", Action: "update", Description: "Update a transform. id and projectId are injected; send the full definition.", Fields: []PayloadField{req("name", "Transform name"), str("description", "Transform description"), req("graphJson", "Pipeline graph JSON"), str("scheduleCron", "Cron schedule; empty for none"), str("triggerMode", "manual or scheduled")}},
		// Folders and source objects.
		{Key: "create:folder", Resource: "folder", Action: "create", Description: "Create a project folder. projectId is injected.", Fields: []PayloadField{req("name", "Folder name"), str("parentId", "Parent folder id; omit for the project root")}, Examples: []string{`{"name":"imports"}`}},
		{Key: "action:move:folder", Resource: "folder", Action: "move", Description: "Move a folder. id and projectId are injected.", Fields: []PayloadField{str("newParentId", "Destination folder id; omit or null for the project root")}, Examples: []string{`{"newParentId":"folder-id"}`}},
		{Key: "action:rename:folder", Resource: "folder", Action: "rename", Description: "Rename a folder. id and projectId are injected.", Fields: []PayloadField{req("name", "New folder name")}, Examples: []string{`{"name":"archive"}`}},
		{Key: "create:source_object", Resource: "source_object", Action: "create", Description: "Create an object such as a table in a hosted source. projectId is injected.", Fields: []PayloadField{req("sourceId", "Source id"), {Name: "parent", Type: "SourceObjectRefInput", Description: "Parent ref such as {\"Kind\":\"schema\",\"Path\":[\"public\"]}"}, req("name", "Object name"), {Name: "fields", Type: "[RecordInput]", Required: true, Description: "Columns as [{\"Key\":\"id\",\"Value\":\"INTEGER\"}]"}}, Examples: []string{`{"sourceId":"source-id","parent":{"Kind":"schema","Path":["public"]},"name":"events","fields":[{"Key":"id","Value":"INTEGER"},{"Key":"name","Value":"TEXT"}]}`}},
		{Key: "update:source_object", Resource: "source_object", Action: "update", Description: "Update a row in a hosted source object. projectId is injected.", Fields: []PayloadField{req("sourceId", "Source id"), {Name: "ref", Type: "SourceObjectRefInput", Required: true, Description: "Object ref such as {\"Kind\":\"table\",\"Path\":[\"public\",\"events\"]}"}, {Name: "values", Type: "[RecordInput]", Required: true, Description: "Matcher plus replacement values as key/value entries"}, {Name: "updatedColumns", Type: "array", Required: true, Description: "Columns being changed"}}},
		{Key: "delete:source_object", Resource: "source_object", Action: "delete", Description: "Delete a row from a hosted source object. projectId is injected.", Fields: []PayloadField{req("sourceId", "Source id"), {Name: "ref", Type: "SourceObjectRefInput", Required: true, Description: "Object ref"}, {Name: "values", Type: "[RecordInput]", Required: true, Description: "Values identifying the row"}}},
		// Packages.
		{Key: "action:create_share:package", Resource: "package", Action: "create_share", Description: "Create a share token for a package so another organization can install it.", Fields: []PayloadField{req("packageId", "Package id"), req("sourceProjectId", "Project that owns the package")}, Examples: []string{`{"packageId":"package-id","sourceProjectId":"project-id"}`}},
		{Key: "action:import:package", Resource: "package", Action: "import", Description: "Import a package's objects into the selected project as editable copies. targetProjectId is injected.", Fields: []PayloadField{req("packageId", "Package id"), bindings}, Examples: []string{`{"packageId":"package-id"}`}},
		{Key: "action:install_shared:package", Resource: "package", Action: "install_shared", Description: "Install a package shared from another organization. targetProjectId is injected.", Fields: []PayloadField{req("shareToken", "Share token"), bindings, str("sourceScopeProjectId", "Scope project when the share covers several")}, Examples: []string{`{"shareToken":"share-token"}`}},
		{Key: "action:import_shared:package", Resource: "package", Action: "import_shared", Description: "Import a package shared from another organization as editable copies. targetProjectId is injected.", Fields: []PayloadField{req("shareToken", "Share token"), bindings, str("sourceScopeProjectId", "Scope project when the share covers several")}, Examples: []string{`{"shareToken":"share-token"}`}},
		{Key: "action:uninstall:package", Resource: "package", Action: "uninstall", Description: "Uninstall a package installation. targetProjectId is injected.", Fields: []PayloadField{req("installationId", "Installation id from whodb_platform_package_installations"), {Name: "deleteObjects", Type: "boolean", Description: "Also delete the installed objects"}}, Examples: []string{`{"installationId":"installation-id","deleteObjects":false}`}},
		{Key: "action:update_installation:package", Resource: "package", Action: "update_installation", Description: "Update an installation to a newer package version. targetProjectId is injected.", Fields: []PayloadField{req("installationId", "Installation id"), str("packageId", "Target package version id; defaults to the latest"), bindings, {Name: "pruneRemoved", Type: "boolean", Description: "Delete objects the new version no longer contains"}}, Examples: []string{`{"installationId":"installation-id"}`}},
		// Versioning.
		{Key: "action:promote:object", Resource: "object", Action: "promote", Description: "Promote the current draft of a versioned object to production. projectId is injected.", Fields: []PayloadField{req("objectId", "Object id"), req("objectType", "app, function, transform, dataset, or ontology"), str("message", "Release note")}, Examples: []string{`{"objectId":"app-id","objectType":"app","message":"Ship v2"}`}},
		{Key: "action:rollback:object", Resource: "object", Action: "rollback", Description: "Make an earlier promoted version active again. projectId is injected.", Fields: []PayloadField{req("objectId", "Object id"), req("objectType", "app, function, transform, dataset, or ontology"), {Name: "version", Type: "integer", Required: true, Description: "Version number from whodb_platform_object_versions"}}, Examples: []string{`{"objectId":"app-id","objectType":"app","version":3}`}},
		{Key: "action:restore_version_to_draft:app", Resource: "app", Action: "restore_version_to_draft", Description: "Copy a promoted app version back into the draft. projectId is injected.", Fields: []PayloadField{req("appId", "App id"), {Name: "version", Type: "integer", Required: true, Description: "Promoted version number"}}, Examples: []string{`{"appId":"app-id","version":2}`}},
		{Key: "action:restore_version_to_draft:dataset", Resource: "dataset", Action: "restore_version_to_draft", Description: "Copy a promoted dataset version back into the draft. projectId is injected.", Fields: []PayloadField{req("datasetId", "Dataset id"), {Name: "version", Type: "integer", Required: true, Description: "Promoted version number"}}},
		{Key: "action:restore_version_to_draft:ontology", Resource: "ontology", Action: "restore_version_to_draft", Description: "Copy a promoted ontology version back into the draft. projectId is injected.", Fields: []PayloadField{req("ontologyId", "Ontology id"), {Name: "version", Type: "integer", Required: true, Description: "Promoted version number"}}},
		{Key: "action:restore_version_to_draft:transform", Resource: "transform", Action: "restore_version_to_draft", Description: "Copy a promoted transform version back into the draft. projectId is injected.", Fields: []PayloadField{req("transformId", "Transform id"), {Name: "version", Type: "integer", Required: true, Description: "Promoted version number"}}},
		// Soft delete, restore and purge: id only.
		idOnly("delete:secret", "secret", "delete", "Delete a secret. id is the secret id; projectId is injected."),
		idOnly("delete:ai_provider", "ai_provider", "delete", "Delete an AI provider. id is the provider id."),
		idOnly("delete:ontology", "ontology", "delete", "Soft-delete an ontology and its links. id is the ontology id; projectId is injected. Restore with action restore."),
		idOnly("delete:ontology_fast_lookup", "ontology_fast_lookup", "delete", "Remove a fast lookup. id is the lookup id; projectId is injected."),
		idOnly("delete:dataset", "dataset", "delete", "Soft-delete a dataset. id is the dataset id; projectId is injected."),
		idOnly("delete:transform", "transform", "delete", "Soft-delete a transform. id is the transform id; projectId is injected."),
		idOnly("delete:function", "function", "delete", "Soft-delete a function. id is the function id; projectId is injected."),
		idOnly("delete:app", "app", "delete", "Soft-delete an app. id is the app id; projectId is injected."),
		idOnly("delete:file", "file", "delete", "Soft-delete a project file. id is the file id; projectId is injected."),
		idOnly("delete:folder", "folder", "delete", "Soft-delete a folder and its contents. id is the folder id; projectId is injected."),
		idOnly("action:restore:source", "source", "restore", "Restore a soft-deleted source. id is the source id; projectId is injected."),
		idOnly("action:purge:source", "source", "purge", "Permanently delete a soft-deleted source. Irreversible."),
		idOnly("action:restore:ontology", "ontology", "restore", "Restore a soft-deleted ontology. id is the ontology id; projectId is injected."),
		idOnly("action:purge:ontology", "ontology", "purge", "Permanently delete a soft-deleted ontology and its rows. Irreversible."),
		idOnly("action:restore:dataset", "dataset", "restore", "Restore a soft-deleted dataset."),
		idOnly("action:purge:dataset", "dataset", "purge", "Permanently delete a soft-deleted dataset and its rows. Irreversible."),
		idOnly("action:restore:transform", "transform", "restore", "Restore a soft-deleted transform."),
		idOnly("action:purge:transform", "transform", "purge", "Permanently delete a soft-deleted transform. Irreversible."),
		idOnly("action:restore:function", "function", "restore", "Restore a soft-deleted function."),
		idOnly("action:purge:function", "function", "purge", "Permanently delete a soft-deleted function. Irreversible."),
		idOnly("action:restore:app", "app", "restore", "Restore a soft-deleted app."),
		idOnly("action:purge:app", "app", "purge", "Permanently delete a soft-deleted app. Irreversible."),
	}
	for _, shape := range shapes {
		if _, exists := PayloadShapes[shape.Key]; exists {
			continue
		}
		PayloadShapes[shape.Key] = shape
	}
}
