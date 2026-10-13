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
 * The {@code client.lineageProviders()} facade: external OpenLineage providers
 * feeding the project's lineage graph.
 */
public final class LineageProvidersHandle {
    private final WhoDB client;

    LineageProvidersHandle(WhoDB client) {
        this.client = client;
    }

    /** Lists the project's providers; API keys are never returned. */
    public List<Map<String, Object>> list() {
        ManifestCheck.warnIfFlagged("ProjectLineageProviders");
        return Facades.asList(client.execute(Operations.projectLineageProvidersRequest(Map.of("projectId", client.projectId()))));
    }

    /** Creates a provider, or updates the one whose id is in the definition. */
    public Map<String, Object> save(Map<String, Object> definition) {
        ManifestCheck.warnIfFlagged("SaveLineageProvider");
        return Facades.asMap(client.execute(Operations.saveLineageProviderRequest(Map.of(
            "input", Facades.mergeInput(definition, "projectId", client.projectId())))));
    }

    /** Deletes a provider. */
    public void delete(String providerId) {
        Facades.require(providerId, "providerId");
        ManifestCheck.warnIfFlagged("DeleteLineageProvider");
        client.execute(Operations.deleteLineageProviderRequest(Map.of("projectId", client.projectId(), "providerId", providerId)));
    }
}
