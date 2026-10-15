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

/** The {@code client.source("src_...")} facade: browse and read a connected data source by ID. */
public final class SourceHandle {
    private final WhoDB client;
    private final String sourceId;

    /**
     * Scope and paging for a source object listing.
     *
     * @param parent optional SourceObjectRefInput ({Kind, Locator, Path})
     * @param pageSize page size; 0 leaves the platform default
     * @param pageOffset page offset
     */
    public record ObjectsOptions(Map<String, Object> parent, int pageSize, int pageOffset) {
        /** No parent, platform default paging. */
        public static ObjectsOptions defaults() {
            return new ObjectsOptions(null, 0, 0);
        }
    }

    /**
     * Filter, sort and paging for a source object read.
     *
     * @param where platform WhereCondition object
     * @param sort list of SortCondition objects
     * @param pageSize page size (0 = 100)
     * @param offset page offset
     */
    public record RowsOptions(Map<String, Object> where, List<Map<String, Object>> sort, int pageSize, int offset) {
        /** No filter, default paging. */
        public static RowsOptions defaults() {
            return new RowsOptions(null, null, 0, 0);
        }
    }

    SourceHandle(WhoDB client, String sourceId) {
        this.client = client;
        this.sourceId = sourceId;
    }

    /** Lists browsable objects (schemas, tables, collections...). */
    public List<Map<String, Object>> objects(ObjectsOptions options) {
        ObjectsOptions effective = options == null ? ObjectsOptions.defaults() : options;
        ManifestCheck.warnIfFlagged("PlatformSourceObjects");
        return Facades.asList(client.execute(Operations.platformSourceObjectsRequest(Facades.vars(
            "projectId", client.projectId(),
            "sourceId", sourceId,
            "parent", effective.parent(),
            "kinds", null,
            "pageSize", effective.pageSize() > 0 ? effective.pageSize() : null,
            "pageOffset", effective.pageOffset() > 0 ? effective.pageOffset() : null))));
    }

    /** Lists the columns of one object (table/collection). */
    public List<Map<String, Object>> columns(Map<String, Object> ref) {
        ManifestCheck.warnIfFlagged("PlatformSourceColumns");
        return Facades.asList(client.execute(Operations.platformSourceColumnsRequest(Facades.vars(
            "projectId", client.projectId(), "sourceId", sourceId, "ref", ref))));
    }

    /** Reads one page of rows from one object. */
    public List<Map<String, Object>> rows(Map<String, Object> ref, RowsOptions options) {
        RowsOptions effective = options == null ? RowsOptions.defaults() : options;
        int pageSize = effective.pageSize() == 0 ? Facades.DEFAULT_PAGE_SIZE : effective.pageSize();
        ManifestCheck.warnIfFlagged("PlatformSourceRows");
        Map<String, Object> result = Facades.asMap(client.execute(Operations.platformSourceRowsRequest(Facades.vars(
            "projectId", client.projectId(),
            "sourceId", sourceId,
            "ref", ref,
            "where", effective.where(),
            "sort", effective.sort(),
            "pageSize", pageSize,
            "pageOffset", effective.offset()))));
        return Hydrator.hydrateRows(result, Map.of());
    }
}
