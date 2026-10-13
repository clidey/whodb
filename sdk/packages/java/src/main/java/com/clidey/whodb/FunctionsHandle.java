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

/** The {@code client.functions()} facade: the project's function collection. */
public final class FunctionsHandle {
    private final WhoDB client;

    FunctionsHandle(WhoDB client) {
        this.client = client;
    }

    /** Lists the project's functions. */
    public List<Map<String, Object>> list() {
        ManifestCheck.warnIfFlagged("ProjectFunctions");
        return Facades.asList(client.execute(Operations.projectFunctionsRequest(Map.of("projectId", client.projectId()))));
    }

    /** Creates a function (CreateFunctionInput fields without projectId). */
    public Map<String, Object> create(Map<String, Object> definition) {
        ManifestCheck.warnIfFlagged("CreateFunction");
        return Facades.asMap(client.execute(Operations.createFunctionRequest(Map.of(
            "input", Facades.mergeInput(definition, "projectId", client.projectId())))));
    }
}
