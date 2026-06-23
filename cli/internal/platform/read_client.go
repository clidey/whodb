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

import (
	"context"
	"fmt"
)

// ProjectSecrets returns hosted project secret metadata and usage without values.
func (c *Client) ProjectSecrets(ctx context.Context, projectID string) ([]ProjectSecret, error) {
	if err := c.RequireOperation("Query", "ProjectSecrets", "secret metadata listing"); err != nil {
		return nil, err
	}
	var resp struct {
		ProjectSecrets []ProjectSecret `json:"ProjectSecrets"`
	}
	err := c.graphQL(ctx, operationProjectSecrets, map[string]any{"projectId": projectID}, &resp)
	return resp.ProjectSecrets, err
}

// SourceFieldConstraints returns field constraints for one hosted source object.
func (c *Client) SourceFieldConstraints(ctx context.Context, projectID, sourceID string, ref SourceObjectRefInput) ([]SourceFieldConstraints, error) {
	if err := c.RequireOperation("Query", "PlatformSourceFieldConstraints", "source field constraints"); err != nil {
		return nil, err
	}
	var resp struct {
		PlatformSourceFieldConstraints []SourceFieldConstraints `json:"PlatformSourceFieldConstraints"`
	}
	variables := map[string]any{"projectId": projectID, "sourceId": sourceID, "ref": ref.graphQLInput()}
	err := c.graphQL(ctx, operationPlatformSourceFieldConstraints, variables, &resp)
	return resp.PlatformSourceFieldConstraints, err
}

// SourceContent returns content for one hosted source object.
func (c *Client) SourceContent(ctx context.Context, projectID, sourceID string, ref SourceObjectRefInput, fields []string) (*SourceContent, error) {
	if err := c.RequireOperation("Query", "PlatformSourceContent", "source content reads"); err != nil {
		return nil, err
	}
	var resp struct {
		PlatformSourceContent *SourceContent `json:"PlatformSourceContent"`
	}
	variables := map[string]any{"projectId": projectID, "sourceId": sourceID, "ref": ref.graphQLInput()}
	if err := c.graphQL(ctx, operationPlatformSourceContentForFields(fields), variables, &resp); err != nil {
		return nil, err
	}
	return resp.PlatformSourceContent, nil
}

// AIProviders returns hosted AI provider metadata without API keys.
func (c *Client) AIProviders(ctx context.Context, projectID string) ([]AIProvider, error) {
	if err := c.RequireOperation("Query", "PlatformAIProviders", "AI provider listing"); err != nil {
		return nil, err
	}
	var resp struct {
		PlatformAIProviders []AIProvider `json:"PlatformAIProviders"`
	}
	err := c.graphQL(ctx, operationPlatformAIProviders, map[string]any{"projectId": projectID}, &resp)
	return resp.PlatformAIProviders, err
}

// AIProviderModels returns model names advertised by one hosted AI provider.
func (c *Client) AIProviderModels(ctx context.Context, projectID, providerID string) ([]string, error) {
	if err := c.RequireOperation("Query", "AIProviderModels", "AI provider model listing"); err != nil {
		return nil, err
	}
	var resp struct {
		AIProviderModels []string `json:"AIProviderModels"`
	}
	err := c.graphQL(ctx, operationAIProviderModels, map[string]any{"projectId": projectID, "providerId": providerID}, &resp)
	return resp.AIProviderModels, err
}

// Ontologies returns hosted ontology object types in one project.
func (c *Client) Ontologies(ctx context.Context, projectID string) ([]Ontology, error) {
	if err := c.RequireOperation("Query", "OntologyEntities", "ontology listing"); err != nil {
		return nil, err
	}
	var resp struct {
		OntologyEntities []Ontology `json:"OntologyEntities"`
	}
	err := c.graphQL(ctx, operationOntologyEntities, map[string]any{"projectId": projectID}, &resp)
	return resp.OntologyEntities, err
}

// Ontology returns one hosted ontology object type.
func (c *Client) Ontology(ctx context.Context, projectID, id string) (*Ontology, error) {
	if err := c.RequireOperation("Query", "OntologyEntity", "ontology detail"); err != nil {
		return nil, err
	}
	var resp struct {
		OntologyEntity *Ontology `json:"OntologyEntity"`
	}
	if err := c.graphQL(ctx, operationOntologyEntity, map[string]any{"projectId": projectID, "id": id}, &resp); err != nil {
		return nil, err
	}
	if resp.OntologyEntity == nil {
		return nil, fmt.Errorf("platform returned no ontology")
	}
	return resp.OntologyEntity, nil
}

// OntologyFastLookups returns saved fast lookups for one ontology.
func (c *Client) OntologyFastLookups(ctx context.Context, projectID, entityID string) ([]OntologyFastLookup, error) {
	if err := c.RequireOperation("Query", "OntologyFastLookups", "ontology fast lookups"); err != nil {
		return nil, err
	}
	var resp struct {
		OntologyFastLookups []OntologyFastLookup `json:"OntologyFastLookups"`
	}
	variables := map[string]any{"projectId": projectID, "entityId": entityID}
	err := c.graphQL(ctx, operationOntologyFastLookups, variables, &resp)
	return resp.OntologyFastLookups, err
}

// OntologyFastLookupSuggestions returns suggested fast lookups for one ontology.
func (c *Client) OntologyFastLookupSuggestions(ctx context.Context, projectID, entityID string) ([]OntologyFastLookupSuggestion, error) {
	if err := c.RequireOperation("Query", "OntologyFastLookupSuggestions", "ontology fast lookup suggestions"); err != nil {
		return nil, err
	}
	var resp struct {
		OntologyFastLookupSuggestions []OntologyFastLookupSuggestion `json:"OntologyFastLookupSuggestions"`
	}
	variables := map[string]any{"projectId": projectID, "entityId": entityID}
	err := c.graphQL(ctx, operationOntologyFastLookupSuggestions, variables, &resp)
	return resp.OntologyFastLookupSuggestions, err
}

// OntologyRows returns row previews for one ontology.
func (c *Client) OntologyRows(ctx context.Context, projectID, id string, pageSize, pageOffset int) (*DatasetQueryResult, error) {
	if err := c.RequireOperation("Query", "OntologyRows", "ontology rows"); err != nil {
		return nil, err
	}
	var resp struct {
		OntologyRows *DatasetQueryResult `json:"OntologyRows"`
	}
	variables := map[string]any{"projectId": projectID, "id": id, "pageSize": pageSize, "pageOffset": pageOffset}
	if err := c.graphQL(ctx, operationOntologyRows, variables, &resp); err != nil {
		return nil, err
	}
	if resp.OntologyRows == nil {
		return nil, fmt.Errorf("platform returned no ontology rows")
	}
	return resp.OntologyRows, nil
}

// OntologyFollowLink returns linked rows for one ontology row.
func (c *Client) OntologyFollowLink(ctx context.Context, projectID, entityID, pk, linkAPIName string, pageSize, pageOffset int) (*DatasetQueryResult, error) {
	if err := c.RequireOperation("Query", "OntologyFollowLink", "ontology link traversal"); err != nil {
		return nil, err
	}
	var resp struct {
		OntologyFollowLink *DatasetQueryResult `json:"OntologyFollowLink"`
	}
	variables := map[string]any{
		"projectId":   projectID,
		"entityId":    entityID,
		"pk":          pk,
		"linkApiName": linkAPIName,
		"pageSize":    pageSize,
		"pageOffset":  pageOffset,
	}
	if err := c.graphQL(ctx, operationOntologyFollowLink, variables, &resp); err != nil {
		return nil, err
	}
	if resp.OntologyFollowLink == nil {
		return nil, fmt.Errorf("platform returned no ontology linked rows")
	}
	return resp.OntologyFollowLink, nil
}

// Datasets returns hosted datasets in one project.
func (c *Client) Datasets(ctx context.Context, projectID string) ([]Dataset, error) {
	if err := c.RequireOperation("Query", "ProjectDatasets", "dataset listing"); err != nil {
		return nil, err
	}
	var resp struct {
		ProjectDatasets []Dataset `json:"ProjectDatasets"`
	}
	err := c.graphQL(ctx, operationProjectDatasets, map[string]any{"projectId": projectID}, &resp)
	return resp.ProjectDatasets, err
}

// Dataset returns one hosted dataset.
func (c *Client) Dataset(ctx context.Context, projectID, id string) (*Dataset, error) {
	if err := c.RequireOperation("Query", "DatasetDetail", "dataset detail"); err != nil {
		return nil, err
	}
	var resp struct {
		DatasetDetail *Dataset `json:"DatasetDetail"`
	}
	if err := c.graphQL(ctx, operationDatasetDetail, map[string]any{"projectId": projectID, "id": id}, &resp); err != nil {
		return nil, err
	}
	if resp.DatasetDetail == nil {
		return nil, fmt.Errorf("platform returned no dataset")
	}
	return resp.DatasetDetail, nil
}

// DatasetRows returns row previews for one hosted dataset.
func (c *Client) DatasetRows(ctx context.Context, projectID, datasetID string, pageSize, pageOffset int) (*DatasetQueryResult, error) {
	if err := c.RequireOperation("Query", "QueryDataset", "dataset rows"); err != nil {
		return nil, err
	}
	var resp struct {
		QueryDataset *DatasetQueryResult `json:"QueryDataset"`
	}
	variables := map[string]any{"input": map[string]any{
		"projectId":  projectID,
		"datasetId":  datasetID,
		"pageSize":   pageSize,
		"pageOffset": pageOffset,
	}}
	if err := c.graphQL(ctx, operationQueryDataset, variables, &resp); err != nil {
		return nil, err
	}
	if resp.QueryDataset == nil {
		return nil, fmt.Errorf("platform returned no dataset rows")
	}
	return resp.QueryDataset, nil
}

// Lineage returns lineage around one root node.
func (c *Client) Lineage(ctx context.Context, projectID, rootID, rootType, direction string, maxDepth int) (*LineageGraph, error) {
	if err := c.RequireOperation("Query", "LineageGraph", "lineage graph"); err != nil {
		return nil, err
	}
	var resp struct {
		LineageGraph *LineageGraph `json:"LineageGraph"`
	}
	variables := map[string]any{
		"projectId": projectID,
		"rootId":    rootID,
		"rootType":  rootType,
		"direction": optionalString(direction),
		"maxDepth":  optionalPositiveInt(maxDepth),
	}
	if err := c.graphQL(ctx, operationLineageGraph, variables, &resp); err != nil {
		return nil, err
	}
	if resp.LineageGraph == nil {
		return nil, fmt.Errorf("platform returned no lineage graph")
	}
	return resp.LineageGraph, nil
}

// LineageNeighbors returns immediate lineage neighbors for one node.
func (c *Client) LineageNeighbors(ctx context.Context, projectID, nodeID, nodeType string) (*LineageGraph, error) {
	if err := c.RequireOperation("Query", "LineageNeighbors", "lineage neighbors"); err != nil {
		return nil, err
	}
	var resp struct {
		LineageNeighbors *LineageGraph `json:"LineageNeighbors"`
	}
	variables := map[string]any{"projectId": projectID, "nodeId": nodeID, "nodeType": nodeType}
	if err := c.graphQL(ctx, operationLineageNeighbors, variables, &resp); err != nil {
		return nil, err
	}
	if resp.LineageNeighbors == nil {
		return nil, fmt.Errorf("platform returned no lineage neighbors")
	}
	return resp.LineageNeighbors, nil
}

// ProjectLineage returns the hosted project lineage graph.
func (c *Client) ProjectLineage(ctx context.Context, projectID string) (*LineageGraph, error) {
	if err := c.RequireOperation("Query", "ProjectLineage", "project lineage"); err != nil {
		return nil, err
	}
	var resp struct {
		ProjectLineage *LineageGraph `json:"ProjectLineage"`
	}
	if err := c.graphQL(ctx, operationProjectLineage, map[string]any{"projectId": projectID}, &resp); err != nil {
		return nil, err
	}
	if resp.ProjectLineage == nil {
		return nil, fmt.Errorf("platform returned no project lineage")
	}
	return resp.ProjectLineage, nil
}

// Transforms returns hosted transforms in one project.
func (c *Client) Transforms(ctx context.Context, projectID string) ([]Transform, error) {
	if err := c.RequireOperation("Query", "ProjectTransforms", "transform listing"); err != nil {
		return nil, err
	}
	var resp struct {
		ProjectTransforms []Transform `json:"ProjectTransforms"`
	}
	err := c.graphQL(ctx, operationProjectTransforms, map[string]any{"projectId": projectID}, &resp)
	return resp.ProjectTransforms, err
}

// TransformRuns returns recent runs for one hosted transform.
func (c *Client) TransformRuns(ctx context.Context, projectID, transformID string, limit int) ([]TransformRun, error) {
	if err := c.RequireOperation("Query", "TransformRuns", "transform runs"); err != nil {
		return nil, err
	}
	var resp struct {
		TransformRuns []TransformRun `json:"TransformRuns"`
	}
	variables := map[string]any{"projectId": projectID, "transformId": transformID, "limit": optionalPositiveInt(limit)}
	err := c.graphQL(ctx, operationTransformRuns, variables, &resp)
	return resp.TransformRuns, err
}

// Functions returns hosted ontology functions in one project.
func (c *Client) Functions(ctx context.Context, projectID string, fields []string) ([]Function, error) {
	if err := c.RequireOperation("Query", "ProjectFunctions", "function listing"); err != nil {
		return nil, err
	}
	var resp struct {
		ProjectFunctions []Function `json:"ProjectFunctions"`
	}
	err := c.graphQL(ctx, operationProjectFunctionsForFields(fields), map[string]any{"projectId": projectID}, &resp)
	return resp.ProjectFunctions, err
}

// Function returns one hosted ontology function.
func (c *Client) Function(ctx context.Context, projectID, id string, fields []string) (*Function, error) {
	if err := c.RequireOperation("Query", "FunctionDetail", "function detail"); err != nil {
		return nil, err
	}
	var resp struct {
		FunctionDetail *Function `json:"FunctionDetail"`
	}
	if err := c.graphQL(ctx, operationFunctionDetailForFields(fields), map[string]any{"projectId": projectID, "id": id}, &resp); err != nil {
		return nil, err
	}
	if resp.FunctionDetail == nil {
		return nil, fmt.Errorf("platform returned no function")
	}
	return resp.FunctionDetail, nil
}

// FolderContents returns hosted project folder contents.
func (c *Client) FolderContents(ctx context.Context, projectID, folderID string, fields []string) (*FolderContents, error) {
	if err := c.RequireOperation("Query", "FolderContents", "project files"); err != nil {
		return nil, err
	}
	var resp struct {
		FolderContents *FolderContents `json:"FolderContents"`
	}
	variables := map[string]any{"projectId": projectID, "folderId": optionalString(folderID)}
	if err := c.graphQL(ctx, operationFolderContentsForFields(fields), variables, &resp); err != nil {
		return nil, err
	}
	if resp.FolderContents == nil {
		return nil, fmt.Errorf("platform returned no folder contents")
	}
	return resp.FolderContents, nil
}

// FilePreview returns a hosted project file preview.
func (c *Client) FilePreview(ctx context.Context, projectID, fileID string, sheetIndex *int, fields []string) (*FilePreviewResult, error) {
	if err := c.RequireOperation("Query", "FilePreview", "file preview"); err != nil {
		return nil, err
	}
	var resp struct {
		FilePreview *FilePreviewResult `json:"FilePreview"`
	}
	variables := map[string]any{"projectId": projectID, "fileId": fileID, "sheetIndex": sheetIndex}
	if err := c.graphQL(ctx, operationFilePreviewForFields(fields), variables, &resp); err != nil {
		return nil, err
	}
	if resp.FilePreview == nil {
		return nil, fmt.Errorf("platform returned no file preview")
	}
	return resp.FilePreview, nil
}

// SearchProjectFiles returns hosted project files matching a query.
func (c *Client) SearchProjectFiles(ctx context.Context, projectID, query string) ([]ProjectFile, error) {
	if err := c.RequireOperation("Query", "SearchProjectFiles", "file search"); err != nil {
		return nil, err
	}
	var resp struct {
		SearchProjectFiles []ProjectFile `json:"SearchProjectFiles"`
	}
	variables := map[string]any{"projectId": projectID, "query": query}
	err := c.graphQL(ctx, operationSearchProjectFiles, variables, &resp)
	return resp.SearchProjectFiles, err
}

// ProjectTabularFiles returns hosted project files that can back datasets.
func (c *Client) ProjectTabularFiles(ctx context.Context, projectID string) ([]ProjectFile, error) {
	if err := c.RequireOperation("Query", "ProjectTabularFiles", "tabular file listing"); err != nil {
		return nil, err
	}
	var resp struct {
		ProjectTabularFiles []ProjectFile `json:"ProjectTabularFiles"`
	}
	err := c.graphQL(ctx, operationProjectTabularFiles, map[string]any{"projectId": projectID}, &resp)
	return resp.ProjectTabularFiles, err
}

// ProjectStorageUsage returns hosted project storage usage in bytes.
func (c *Client) ProjectStorageUsage(ctx context.Context, projectID string) (int, error) {
	if err := c.RequireOperation("Query", "ProjectStorageUsage", "project storage usage"); err != nil {
		return 0, err
	}
	var resp struct {
		ProjectStorageUsage int `json:"ProjectStorageUsage"`
	}
	err := c.graphQL(ctx, operationProjectStorageUsage, map[string]any{"projectId": projectID}, &resp)
	return resp.ProjectStorageUsage, err
}

func optionalString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func optionalPositiveInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}
