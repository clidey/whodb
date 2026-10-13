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

/** The {@code client.transforms()} facade: the project's transform collection. */
public final class TransformsHandle {
    private final WhoDB client;

    TransformsHandle(WhoDB client) {
        this.client = client;
    }

    /** Lists the project's transforms. */
    public List<Map<String, Object>> list() {
        ManifestCheck.warnIfFlagged("ProjectTransforms");
        return Facades.asList(client.execute(Operations.projectTransformsRequest(Map.of("projectId", client.projectId()))));
    }

    /** Creates a transform from a pipeline graph (SaveTransformInput fields without id/projectId). */
    public Map<String, Object> create(Map<String, Object> definition) {
        ManifestCheck.warnIfFlagged("SaveTransform");
        return Facades.asMap(client.execute(Operations.saveTransformRequest(Map.of(
            "input", Facades.mergeInput(definition, "projectId", client.projectId())))));
    }

    /** Runs a pipeline graph without saving it and returns the resulting rows. */
    public RowsPage preview(String graphJson) {
        ManifestCheck.warnIfFlagged("PreviewTransform");
        Map<String, Object> result = Facades.asMap(client.execute(Operations.previewTransformRequest(Map.of(
            "projectId", client.projectId(), "graphJson", graphJson))));
        return new RowsPage(Hydrator.hydrateRows(result, Map.of()), Facades.totalOf(result));
    }
}
