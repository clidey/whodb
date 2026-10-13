// Copyright 2026 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//! Organization and governance facades: service accounts, lineage providers,
//! and ontology sharing.

use serde_json::{json, Map, Value};

use crate::client::Client;
use crate::error::Result;
use crate::facade_util::{as_list, merge_input, require, vars};
use crate::gen::operations as ops;
use crate::manifest_check::warn_if_flagged;

fn opt(value: Option<&str>) -> Value {
    value
        .filter(|text| !text.is_empty())
        .map(|text| json!(text))
        .unwrap_or(Value::Null)
}

/// API access principals for the organization the client is scoped to.
pub struct ServiceAccountsHandle<'client> {
    pub(crate) client: &'client Client,
}

impl<'client> ServiceAccountsHandle<'client> {
    /// Lists service accounts with key metadata; raw keys are never returned.
    pub fn list(&self) -> Result<Vec<Value>> {
        let org_id = self.client.org_id()?;
        warn_if_flagged("ServiceAccounts");
        Ok(as_list(self.client.execute(
            ops::service_accounts_request(vars(&[("orgId", json!(org_id))])),
        )?))
    }

    /// Creates a service account. Issue a key with `create_key`.
    pub fn create(&self, name: &str, description: Option<&str>) -> Result<Value> {
        require(name, "name")?;
        let org_id = self.client.org_id()?;
        warn_if_flagged("CreateServiceAccount");
        self.client
            .execute(ops::create_service_account_request(vars(&[
                ("orgId", json!(org_id)),
                ("name", json!(name.trim())),
                ("description", opt(description)),
            ])))
    }

    /// Issues an API key. The returned rawKey is shown exactly once; store it
    /// securely and never log it. `expires_at` is RFC 3339.
    pub fn create_key(&self, service_account_id: &str, expires_at: Option<&str>) -> Result<Value> {
        require(service_account_id, "service_account_id")?;
        let org_id = self.client.org_id()?;
        warn_if_flagged("CreateServiceAccountKey");
        self.client
            .execute(ops::create_service_account_key_request(vars(&[
                ("orgId", json!(org_id)),
                ("serviceAccountId", json!(service_account_id)),
                ("expiresAt", opt(expires_at)),
            ])))
    }

    /// Revokes one key.
    pub fn revoke_key(&self, key_id: &str) -> Result<()> {
        require(key_id, "key_id")?;
        let org_id = self.client.org_id()?;
        warn_if_flagged("RevokeServiceAccountKey");
        self.client
            .execute(ops::revoke_service_account_key_request(vars(&[
                ("orgId", json!(org_id)),
                ("keyId", json!(key_id)),
            ])))?;
        Ok(())
    }

    /// Deletes a service account and revokes all of its keys.
    pub fn delete(&self, service_account_id: &str) -> Result<()> {
        require(service_account_id, "service_account_id")?;
        let org_id = self.client.org_id()?;
        warn_if_flagged("DeleteServiceAccount");
        self.client
            .execute(ops::delete_service_account_request(vars(&[
                ("orgId", json!(org_id)),
                ("id", json!(service_account_id)),
            ])))?;
        Ok(())
    }
}

/// External OpenLineage providers feeding the project's lineage graph.
pub struct LineageProvidersHandle<'client> {
    pub(crate) client: &'client Client,
}

impl<'client> LineageProvidersHandle<'client> {
    /// Lists the project's providers; API keys are never returned.
    pub fn list(&self) -> Result<Vec<Value>> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("ProjectLineageProviders");
        Ok(as_list(self.client.execute(
            ops::project_lineage_providers_request(vars(&[("projectId", json!(project_id))])),
        )?))
    }

    /// Creates a provider, or updates the one whose id is in the definition.
    pub fn save(&self, definition: &Map<String, Value>) -> Result<Value> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("SaveLineageProvider");
        let input = merge_input(definition, &[("projectId", json!(project_id))]);
        self.client
            .execute(ops::save_lineage_provider_request(vars(&[(
                "input", input,
            )])))
    }

    /// Deletes a provider.
    pub fn delete(&self, provider_id: &str) -> Result<()> {
        require(provider_id, "provider_id")?;
        let project_id = self.client.project_id()?;
        warn_if_flagged("DeleteLineageProvider");
        self.client
            .execute(ops::delete_lineage_provider_request(vars(&[
                ("projectId", json!(project_id)),
                ("providerId", json!(provider_id)),
            ])))?;
        Ok(())
    }
}

/// Share ontologies with other projects in the organization and reference
/// shared ontologies locally.
pub struct SharingHandle<'client> {
    pub(crate) client: &'client Client,
}

impl<'client> SharingHandle<'client> {
    /// Lists ontologies this project shares, with their consumers.
    pub fn shared(&self) -> Result<Vec<Value>> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("SharedOntologies");
        Ok(as_list(self.client.execute(
            ops::shared_ontologies_request(vars(&[("projectId", json!(project_id))])),
        )?))
    }

    /// Lists ontologies other projects share that this project may reference.
    pub fn discover(&self) -> Result<Vec<Value>> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("DiscoverSharedOntologies");
        Ok(as_list(self.client.execute(
            ops::discover_shared_ontologies_request(vars(&[(
                "consumerProjectId",
                json!(project_id),
            )])),
        )?))
    }

    /// Lists shared ontologies this project references.
    pub fn references(&self) -> Result<Vec<Value>> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("OntologyReferences");
        Ok(as_list(self.client.execute(
            ops::ontology_references_request(vars(&[("projectId", json!(project_id))])),
        )?))
    }

    /// Counts dependents of a reference before removing it.
    pub fn reference_impact(&self, reference_id: &str) -> Result<Value> {
        require(reference_id, "reference_id")?;
        warn_if_flagged("OntologyReferenceImpact");
        self.client
            .execute(ops::ontology_reference_impact_request(vars(&[(
                "referenceId",
                json!(reference_id),
            )])))
    }

    /// Shares one of this project's ontologies. `visibility` is
    /// "organization" or "allowlist"; `allowed_project_ids` applies to
    /// allowlist sharing.
    pub fn share(
        &self,
        ontology_id: &str,
        visibility: &str,
        allowed_project_ids: Option<&[String]>,
    ) -> Result<Value> {
        require(ontology_id, "ontology_id")?;
        let project_id = self.client.project_id()?;
        warn_if_flagged("ShareOntology");
        let input = json!({
            "projectId": project_id,
            "ontologyId": ontology_id,
            "visibility": visibility,
            "allowedProjectIds": allowed_project_ids.map(|ids| json!(ids)).unwrap_or(Value::Null),
        });
        self.client
            .execute(ops::share_ontology_request(vars(&[("input", input)])))
    }

    /// Stops sharing an ontology; existing references become broken.
    pub fn unshare(&self, ontology_id: &str) -> Result<()> {
        require(ontology_id, "ontology_id")?;
        let project_id = self.client.project_id()?;
        warn_if_flagged("UnshareOntology");
        self.client.execute(ops::unshare_ontology_request(vars(&[
            ("projectId", json!(project_id)),
            ("ontologyId", json!(ontology_id)),
        ])))?;
        Ok(())
    }

    /// References a shared ontology from this project under a local alias.
    pub fn create_reference(
        &self,
        owner_project_id: &str,
        ontology_id: &str,
        alias: &str,
    ) -> Result<Value> {
        require(owner_project_id, "owner_project_id")?;
        require(ontology_id, "ontology_id")?;
        require(alias, "alias")?;
        let project_id = self.client.project_id()?;
        warn_if_flagged("CreateOntologyReference");
        let input = json!({
            "consumerProjectId": project_id,
            "ownerProjectId": owner_project_id,
            "ontologyId": ontology_id,
            "alias": alias,
        });
        self.client
            .execute(ops::create_ontology_reference_request(vars(&[(
                "input", input,
            )])))
    }

    /// Removes a reference. Check `reference_impact` first.
    pub fn remove_reference(&self, reference_id: &str) -> Result<()> {
        require(reference_id, "reference_id")?;
        warn_if_flagged("RemoveOntologyReference");
        self.client
            .execute(ops::remove_ontology_reference_request(vars(&[(
                "referenceId",
                json!(reference_id),
            )])))?;
        Ok(())
    }
}
