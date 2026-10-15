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

/** The {@code client.dataset("weekly_kpis")} facade: metadata and paged reads for one dataset. */
public final class DatasetHandle {
    private final WhoDB client;
    private final String name;
    private Map<String, Object> dataset;

    /**
     * Paging for dataset reads.
     *
     * @param pageSize page size (0 = 100)
     * @param pageOffset page offset
     */
    public record RowsOptions(int pageSize, int pageOffset) {
        /** Default paging. */
        public static RowsOptions defaults() {
            return new RowsOptions(0, 0);
        }
    }

    DatasetHandle(WhoDB client, String name) {
        this.client = client;
        this.name = name;
    }

    /** Resolves and caches the dataset metadata backing this handle. */
    public synchronized Map<String, Object> meta() {
        if (dataset != null) {
            return dataset;
        }
        ManifestCheck.warnIfFlagged("ProjectDatasets");
        List<Map<String, Object>> datasets = Facades.asList(
            client.execute(Operations.projectDatasetsRequest(Map.of("projectId", client.projectId()))));
        for (Map<String, Object> candidate : datasets) {
            if (name.equals(candidate.get("name"))) {
                dataset = candidate;
                return dataset;
            }
        }
        throw new WhoDBException(WhoDBException.Kind.NOT_FOUND, "dataset \"" + name + "\" not found in this project");
    }

    /** Queries dataset rows with paging. */
    public RowsPage rows(RowsOptions options) {
        Map<String, Object> meta = meta();
        RowsOptions effective = options == null ? RowsOptions.defaults() : options;
        int pageSize = effective.pageSize() == 0 ? Facades.DEFAULT_PAGE_SIZE : effective.pageSize();
        ManifestCheck.warnIfFlagged("QueryDataset");
        Map<String, Object> result = Facades.asMap(client.execute(Operations.queryDatasetRequest(Map.of(
            "input", Facades.vars(
                "projectId", client.projectId(),
                "datasetId", meta.get("id"),
                "pageSize", pageSize,
                "pageOffset", effective.pageOffset())))));
        return new RowsPage(Hydrator.hydrateRows(result, Map.of()), Facades.totalOf(result));
    }

    /** Promotes the draft dataset: snapshots it and makes it the active production version. {@code message} may be null. */
    public Map<String, Object> promote(String message) {
        return client.versions().promote("dataset", String.valueOf(meta().get("id")), message);
    }

    /** Lists the dataset's promoted versions, newest first. */
    public List<Map<String, Object>> versions() {
        return client.versions().list("dataset", String.valueOf(meta().get("id")));
    }
}
