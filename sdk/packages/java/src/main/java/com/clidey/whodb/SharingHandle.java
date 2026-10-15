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

package com.clidey.whodb;

import com.clidey.whodb.gen.Operations;
import java.util.List;
import java.util.Map;

/**
 * The {@code client.sharing()} facade: share ontologies with other projects in
 * the organization and reference shared ontologies locally.
 */
public final class SharingHandle {
    private final WhoDB client;

    SharingHandle(WhoDB client) {
        this.client = client;
    }

    /** Lists ontologies this project shares, with their consumers. */
    public List<Map<String, Object>> shared() {
        ManifestCheck.warnIfFlagged("SharedOntologies");
        return Facades.asList(client.execute(Operations.sharedOntologiesRequest(Map.of("projectId", client.projectId()))));
    }

    /** Lists ontologies other projects share that this project may reference. */
    public List<Map<String, Object>> discover() {
        ManifestCheck.warnIfFlagged("DiscoverSharedOntologies");
        return Facades.asList(client.execute(Operations.discoverSharedOntologiesRequest(Map.of("consumerProjectId", client.projectId()))));
    }

    /** Lists shared ontologies this project references. */
    public List<Map<String, Object>> references() {
        ManifestCheck.warnIfFlagged("OntologyReferences");
        return Facades.asList(client.execute(Operations.ontologyReferencesRequest(Map.of("projectId", client.projectId()))));
    }

    /** Counts dependents of a reference before removing it. */
    public Map<String, Object> referenceImpact(String referenceId) {
        Facades.require(referenceId, "referenceId");
        ManifestCheck.warnIfFlagged("OntologyReferenceImpact");
        return Facades.asMap(client.execute(Operations.ontologyReferenceImpactRequest(Map.of("referenceId", referenceId))));
    }

    /**
     * Shares one of this project's ontologies. {@code visibility} is
     * "organization" or "allowlist"; {@code allowedProjectIds} applies to
     * allowlist sharing.
     */
    public Map<String, Object> share(String ontologyId, String visibility, List<String> allowedProjectIds) {
        Facades.require(ontologyId, "ontologyId");
        ManifestCheck.warnIfFlagged("ShareOntology");
        return Facades.asMap(client.execute(Operations.shareOntologyRequest(Map.of("input", Facades.vars(
            "projectId", client.projectId(), "ontologyId", ontologyId,
            "visibility", visibility, "allowedProjectIds", allowedProjectIds)))));
    }

    /** Stops sharing an ontology; existing references become broken. */
    public void unshare(String ontologyId) {
        Facades.require(ontologyId, "ontologyId");
        ManifestCheck.warnIfFlagged("UnshareOntology");
        client.execute(Operations.unshareOntologyRequest(Map.of("projectId", client.projectId(), "ontologyId", ontologyId)));
    }

    /** References a shared ontology from this project under a local alias. */
    public Map<String, Object> createReference(String ownerProjectId, String ontologyId, String alias) {
        Facades.require(ownerProjectId, "ownerProjectId");
        Facades.require(ontologyId, "ontologyId");
        Facades.require(alias, "alias");
        ManifestCheck.warnIfFlagged("CreateOntologyReference");
        return Facades.asMap(client.execute(Operations.createOntologyReferenceRequest(Map.of("input", Map.of(
            "consumerProjectId", client.projectId(), "ownerProjectId", ownerProjectId,
            "ontologyId", ontologyId, "alias", alias)))));
    }

    /** Removes a reference. Check {@link #referenceImpact} first. */
    public void removeReference(String referenceId) {
        Facades.require(referenceId, "referenceId");
        ManifestCheck.warnIfFlagged("RemoveOntologyReference");
        client.execute(Operations.removeOntologyReferenceRequest(Map.of("referenceId", referenceId)));
    }
}
