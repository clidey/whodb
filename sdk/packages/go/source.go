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

	"github.com/clidey/whodb/sdk/packages/go/gen"
)

// SourceObjectsOptions scope and page a source object listing.
type SourceObjectsOptions struct {
	// Parent is an optional SourceObjectRefInput ({Kind, Locator, Path}).
	Parent     map[string]any
	PageSize   int
	PageOffset int
}

// SourceRowsOptions filter, sort and page a source object read. Where is a
// platform WhereCondition object and Sort a list of SortCondition objects.
type SourceRowsOptions struct {
	Where    map[string]any
	Sort     []map[string]any
	PageSize int
	Offset   int
}

// SourceHandle is the client.Source("src_...") facade: browse and read a
// connected data source by ID.
type SourceHandle struct {
	client   *Client
	sourceID string
}

// Objects lists browsable objects (schemas, tables, collections...).
func (h *SourceHandle) Objects(ctx context.Context, options SourceObjectsOptions) ([]map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("PlatformSourceObjects")
	variables := map[string]any{"projectId": projectID, "sourceId": h.sourceID, "parent": nil, "kinds": nil, "pageSize": nil, "pageOffset": nil}
	if options.Parent != nil {
		variables["parent"] = options.Parent
	}
	if options.PageSize > 0 {
		variables["pageSize"] = options.PageSize
	}
	if options.PageOffset > 0 {
		variables["pageOffset"] = options.PageOffset
	}
	result, err := h.client.execute(ctx, gen.NewPlatformSourceObjectsRequest(variables))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Columns lists the columns of one object (table/collection).
func (h *SourceHandle) Columns(ctx context.Context, ref map[string]any) ([]map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("PlatformSourceColumns")
	result, err := h.client.execute(ctx, gen.NewPlatformSourceColumnsRequest(map[string]any{"projectId": projectID, "sourceId": h.sourceID, "ref": ref}))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Rows reads one page of rows from one object.
func (h *SourceHandle) Rows(ctx context.Context, ref map[string]any, options SourceRowsOptions) ([]Row, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	pageSize := options.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	warnIfFlagged("PlatformSourceRows")
	variables := map[string]any{
		"projectId":  projectID,
		"sourceId":   h.sourceID,
		"ref":        ref,
		"where":      nil,
		"sort":       nil,
		"pageSize":   pageSize,
		"pageOffset": options.Offset,
	}
	if options.Where != nil {
		variables["where"] = options.Where
	}
	if options.Sort != nil {
		variables["sort"] = options.Sort
	}
	result, err := h.client.execute(ctx, gen.NewPlatformSourceRowsRequest(variables))
	if err != nil {
		return nil, err
	}
	rows, _ := hydrateRows(asMap(result), nil)
	return rows, nil
}
