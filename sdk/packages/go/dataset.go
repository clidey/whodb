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

package whodb

import (
	"context"
	"fmt"
	"sync"

	"github.com/clidey/whodb/sdk/packages/go/gen"
)

// DatasetRowsOptions page dataset reads.
type DatasetRowsOptions struct {
	PageSize   int
	PageOffset int
}

// DatasetHandle is the client.Dataset("weekly_kpis") facade, addressed by
// dataset name.
type DatasetHandle struct {
	client *Client
	name   string

	mu      sync.Mutex
	dataset map[string]any
}

// Meta resolves and caches the dataset metadata backing this handle.
func (h *DatasetHandle) Meta(ctx context.Context) (map[string]any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.dataset != nil {
		return h.dataset, nil
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ProjectDatasets")
	result, err := h.client.execute(ctx, gen.NewProjectDatasetsRequest(map[string]any{"projectId": projectID}))
	if err != nil {
		return nil, err
	}
	for _, dataset := range asList(result) {
		if dataset["name"] == h.name {
			h.dataset = dataset
			return dataset, nil
		}
	}
	return nil, fmt.Errorf("%w: dataset %q not found in this project", ErrNotFound, h.name)
}

// Rows queries dataset rows with paging.
func (h *DatasetHandle) Rows(ctx context.Context, options DatasetRowsOptions) (RowsPage, error) {
	dataset, err := h.Meta(ctx)
	if err != nil {
		return RowsPage{}, err
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return RowsPage{}, err
	}
	pageSize := options.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	warnIfFlagged("QueryDataset")
	result, err := h.client.execute(ctx, gen.NewQueryDatasetRequest(map[string]any{
		"input": map[string]any{
			"projectId":  projectID,
			"datasetId":  dataset["id"],
			"pageSize":   pageSize,
			"pageOffset": options.PageOffset,
		},
	}))
	if err != nil {
		return RowsPage{}, err
	}
	rows, total := hydrateRows(asMap(result), nil)
	return RowsPage{Rows: rows, TotalCount: total}, nil
}

// Promote snapshots the draft dataset and makes it the active production version.
func (h *DatasetHandle) Promote(ctx context.Context, message string) (map[string]any, error) {
	obj, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	return h.client.Versions().Promote(ctx, ObjectTypeDataset, fmt.Sprint(obj["id"]), message)
}

// Versions lists the dataset's promoted versions, newest first.
func (h *DatasetHandle) Versions(ctx context.Context) ([]map[string]any, error) {
	obj, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	return h.client.Versions().List(ctx, ObjectTypeDataset, fmt.Sprint(obj["id"]))
}
