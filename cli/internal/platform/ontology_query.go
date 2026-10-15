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
	"encoding/json"
	"fmt"
	"strings"
)

// Ontology analytics operations: bounded queries, SQL aggregates, statistics,
// descriptions, similarity search, record capabilities, action previews and
// execution history. These sit next to the row previews in read_client.go.

const operationOntologyQuery = `
query CLIPlatformOntologyQuery($projectId: ID!, $input: OntologyQueryInput!) {
  OntologyQuery(projectId: $projectId, input: $input) {
` + datasetQueryResultFields + `
  }
}`

const operationOntologyAggregate = `
query CLIPlatformOntologyAggregate($projectId: ID!, $id: ID!, $groupBy: [String!]!, $metrics: [OntologyAggregateMetricInput!]!, $where: WhereCondition, $sort: [SortCondition!], $pageSize: Int!, $pageOffset: Int!) {
  OntologyAggregate(projectId: $projectId, id: $id, groupBy: $groupBy, metrics: $metrics, where: $where, sort: $sort, pageSize: $pageSize, pageOffset: $pageOffset) {
` + datasetQueryResultFields + `
  }
}`

const operationOntologyStats = `
query CLIPlatformOntologyStats($projectId: ID!, $id: ID!, $property: String!, $where: WhereCondition) {
  OntologyStats(projectId: $projectId, id: $id, property: $property, where: $where) {
    property
    min
    max
    total
    nonNull
  }
}`

const operationOntologyDescribe = `
query CLIPlatformOntologyDescribe($projectId: ID!, $input: OntologyDescribeInput!) {
  OntologyDescribe(projectId: $projectId, input: $input) {
    text
    entities {
      name
      displayName
      description
      primaryKey
      fields { name displayName description type required searchable sortable }
    }
    relationships { name source sourceField target targetField cardinality kind }
  }
}`

const operationOntologySimilar = `
query CLIPlatformOntologySimilar($projectId: ID!, $input: OntologySimilarInput!) {
  OntologySimilar(projectId: $projectId, input: $input) {
    rows {
      score
      values { key value }
    }
  }
}`

const operationOntologyRecordCapabilities = `
query CLIPlatformOntologyRecordCapabilities($projectId: ID!, $ontologyId: ID!, $recordKey: String) {
  OntologyRecordCapabilities(projectId: $projectId, ontologyId: $ontologyId, recordKey: $recordKey) {
    behaviorVersion
    recordVersion
    currentState
    readableProperties
    writableProperties
    actions { apiName allowed denialCode }
  }
}`

const operationPreviewOntologyAction = `
query CLIPlatformPreviewOntologyAction($input: PreviewOntologyActionInput!) {
  PreviewOntologyAction(input: $input) {
    allowed
    denialCode
    proposedChanges
    emittedEvents
  }
}`

const operationOntologyActionExecutions = `
query CLIPlatformOntologyActionExecutions($projectId: ID!, $ontologyId: ID!, $recordKey: String!, $limit: Int) {
  OntologyActionExecutions(projectId: $projectId, ontologyId: $ontologyId, recordKey: $recordKey, limit: $limit) {
    id
    ontologyId
    recordKey
    actionName
    behaviorVersionId
    actorId
    status
    resultSummary
    createdAt
  }
}`

const operationOntologyEmbeddingJobs = `
query CLIPlatformOntologyEmbeddingJobs($projectId: ID!, $entityId: ID!) {
  OntologyEmbeddingJobs(projectId: $projectId, entityId: $entityId) {
    id
    entityId
    status
    model
    dimensions
    batchSize
    totalRows
    processedRows
    errorMessage
    createdAt
    startedAt
    completedAt
  }
}`

const operationPreviewTransform = `
query CLIPlatformPreviewTransform($projectId: ID!, $graphJson: String!) {
  PreviewTransform(projectId: $projectId, graphJson: $graphJson) {
` + datasetQueryResultFields + `
  }
}`

func init() {
	platformOperations["ontology_query"] = operationOntologyQuery
	platformOperations["ontology_aggregate"] = operationOntologyAggregate
	platformOperations["ontology_stats"] = operationOntologyStats
	platformOperations["ontology_describe"] = operationOntologyDescribe
	platformOperations["ontology_similar"] = operationOntologySimilar
	platformOperations["ontology_record_capabilities"] = operationOntologyRecordCapabilities
	platformOperations["ontology_action_preview"] = operationPreviewOntologyAction
	platformOperations["ontology_action_executions"] = operationOntologyActionExecutions
	platformOperations["ontology_embedding_jobs"] = operationOntologyEmbeddingJobs
	platformOperations["transform_preview"] = operationPreviewTransform
}

// OntologyQueryOperators lists the comparison operators accepted inside an
// OntologyQuery where filter, mirroring the hosted bounded query service.
var OntologyQueryOperators = []string{"eq", "neq", "gt", "gte", "lt", "lte", "contains", "icontains", "isnull"}

// OntologyQueryMetricOps lists the metric operations accepted by OntologyQuery.
var OntologyQueryMetricOps = []string{"count", "count_distinct", "sum", "avg", "min", "max"}

// OntologyAggregateMetricOps lists the metric operations accepted by OntologyAggregate.
var OntologyAggregateMetricOps = []string{"COUNT", "SUM", "AVG", "MIN", "MAX"}

// OntologyQueryMaxScanLimit is the largest scan window the hosted bounded query accepts.
const OntologyQueryMaxScanLimit = 5000

// OntologyQueryDefaultScanLimit is the scan window used when none is requested.
const OntologyQueryDefaultScanLimit = 1000

// OntologyMetric is one metric for OntologyQuery or OntologyAggregate.
type OntologyMetric struct {
	Op    string `json:"op"`
	Field string `json:"field,omitempty"`
	As    string `json:"as,omitempty"`
}

// OntologyQueryJoin joins a second ontology into a bounded ontology query.
type OntologyQueryJoin struct {
	Entity     string `json:"entity"`
	LeftField  string `json:"leftField"`
	RightField string `json:"rightField"`
	As         string `json:"as,omitempty"`
	Kind       string `json:"kind,omitempty"`
}

// OntologyQuerySort orders bounded ontology query results.
type OntologyQuerySort struct {
	Field string `json:"field"`
	Desc  bool   `json:"desc,omitempty"`
}

// OntologyQueryInput is the CLI shape of the hosted OntologyQueryInput.
// Where uses the bounded query grammar: {field: value}, {field: {op: value}},
// and _and / _or lists.
type OntologyQueryInput struct {
	Entity       string
	Where        map[string]any
	Search       string
	SearchFields []string
	Joins        []OntologyQueryJoin
	GroupBy      []string
	Metrics      []OntologyMetric
	Sort         []OntologyQuerySort
	PageSize     int
	Offset       int
	ScanLimit    int
}

// SortCondition is the shared platform sort input used by OntologyAggregate.
type SortCondition struct {
	Column    string `json:"Column"`
	Direction string `json:"Direction"`
}

// OntologyAggregateInput is the CLI shape of the hosted OntologyAggregate arguments.
// Where is a platform WhereCondition object passed through unchanged.
type OntologyAggregateInput struct {
	ID         string
	GroupBy    []string
	Metrics    []OntologyMetric
	Where      map[string]any
	Sort       []SortCondition
	PageSize   int
	PageOffset int
}

// OntologyStatsResult summarises one ontology property.
type OntologyStatsResult struct {
	Property string  `json:"property"`
	Min      *string `json:"min"`
	Max      *string `json:"max"`
	Total    int     `json:"total"`
	NonNull  int     `json:"nonNull"`
}

// OntologyDescriptionField is one property in an ontology description.
type OntologyDescriptionField struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Searchable  bool   `json:"searchable"`
	Sortable    bool   `json:"sortable"`
}

// OntologyDescriptionEntity is one ontology in an ontology description.
type OntologyDescriptionEntity struct {
	Name        string                     `json:"name"`
	DisplayName string                     `json:"displayName"`
	Description string                     `json:"description"`
	PrimaryKey  string                     `json:"primaryKey"`
	Fields      []OntologyDescriptionField `json:"fields"`
}

// OntologyDescriptionRelationship is one link in an ontology description.
type OntologyDescriptionRelationship struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	SourceField string `json:"sourceField"`
	Target      string `json:"target"`
	TargetField string `json:"targetField"`
	Cardinality string `json:"cardinality"`
	Kind        string `json:"kind"`
}

// OntologyDescription is a human-readable plus structured ontology summary.
type OntologyDescription struct {
	Text          string                            `json:"text"`
	Entities      []OntologyDescriptionEntity       `json:"entities"`
	Relationships []OntologyDescriptionRelationship `json:"relationships"`
}

// OntologySimilarInput is the CLI shape of the hosted OntologySimilarInput.
type OntologySimilarInput struct {
	EntityID   string
	RowID      string
	TopK       int
	Properties []string
	Where      map[string]any
}

// OntologySimilarityValue is one key/value pair of a similar row.
type OntologySimilarityValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// OntologySimilarityRow is one similarity hit.
type OntologySimilarityRow struct {
	Score  float64                   `json:"score"`
	Values []OntologySimilarityValue `json:"values"`
}

// OntologySimilarityResult is the hosted similarity search response.
type OntologySimilarityResult struct {
	Rows []OntologySimilarityRow `json:"rows"`
}

// OntologyActionCapability reports whether one behavior action is allowed.
type OntologyActionCapability struct {
	APIName    string  `json:"apiName"`
	Allowed    bool    `json:"allowed"`
	DenialCode *string `json:"denialCode"`
}

// OntologyRecordCapabilities reports what the caller may read, write and do on a record.
type OntologyRecordCapabilities struct {
	BehaviorVersion    int                        `json:"behaviorVersion"`
	RecordVersion      *int                       `json:"recordVersion"`
	CurrentState       *string                    `json:"currentState"`
	ReadableProperties []string                   `json:"readableProperties"`
	WritableProperties []string                   `json:"writableProperties"`
	Actions            []OntologyActionCapability `json:"actions"`
}

// PreviewOntologyActionInput is the CLI shape of the hosted PreviewOntologyActionInput.
type PreviewOntologyActionInput struct {
	ProjectID  string
	OntologyID string
	RecordKey  string
	Action     string
	Values     map[string]any
}

// OntologyActionPreview describes what an action would do without executing it.
type OntologyActionPreview struct {
	Allowed         bool     `json:"allowed"`
	DenialCode      *string  `json:"denialCode"`
	ProposedChanges any      `json:"proposedChanges"`
	EmittedEvents   []string `json:"emittedEvents"`
}

// OntologyActionExecution is one recorded behavior action execution.
type OntologyActionExecution struct {
	ID                string `json:"id"`
	OntologyID        string `json:"ontologyId"`
	RecordKey         string `json:"recordKey"`
	ActionName        string `json:"actionName"`
	BehaviorVersionID string `json:"behaviorVersionId"`
	ActorID           string `json:"actorId"`
	Status            string `json:"status"`
	ResultSummary     any    `json:"resultSummary"`
	CreatedAt         string `json:"createdAt"`
}

// OntologyEmbeddingJob is one embedding backfill job for an ontology.
type OntologyEmbeddingJob struct {
	ID            string  `json:"id"`
	EntityID      string  `json:"entityId"`
	Status        string  `json:"status"`
	Model         string  `json:"model"`
	Dimensions    int     `json:"dimensions"`
	BatchSize     int     `json:"batchSize"`
	TotalRows     int     `json:"totalRows"`
	ProcessedRows int     `json:"processedRows"`
	ErrorMessage  string  `json:"errorMessage"`
	CreatedAt     string  `json:"createdAt"`
	StartedAt     *string `json:"startedAt"`
	CompletedAt   *string `json:"completedAt"`
}

// OntologyQuery runs a bounded, in-memory ontology query with filters, search,
// joins, grouping and metrics.
func (c *Client) OntologyQuery(ctx context.Context, projectID string, input OntologyQueryInput) (*DatasetQueryResult, error) {
	if err := c.RequireOperation("Query", "OntologyQuery", "ontology query"); err != nil {
		return nil, err
	}
	gqlInput := map[string]any{"entity": strings.TrimSpace(input.Entity)}
	if len(input.Where) > 0 {
		raw, err := json.Marshal(input.Where)
		if err != nil {
			return nil, fmt.Errorf("encode where filter: %w", err)
		}
		gqlInput["whereJson"] = string(raw)
	}
	if strings.TrimSpace(input.Search) != "" {
		gqlInput["search"] = strings.TrimSpace(input.Search)
	}
	if len(input.SearchFields) > 0 {
		gqlInput["searchFields"] = input.SearchFields
	}
	if len(input.Joins) > 0 {
		joins := make([]map[string]any, 0, len(input.Joins))
		for _, join := range input.Joins {
			entry := map[string]any{"entity": join.Entity, "leftField": join.LeftField, "rightField": join.RightField}
			if join.As != "" {
				entry["as"] = join.As
			}
			if join.Kind != "" {
				entry["kind"] = join.Kind
			}
			joins = append(joins, entry)
		}
		gqlInput["joins"] = joins
	}
	if len(input.GroupBy) > 0 {
		gqlInput["groupBy"] = input.GroupBy
	}
	if len(input.Metrics) > 0 {
		gqlInput["metrics"] = ontologyMetricVariables(input.Metrics, strings.ToLower)
	}
	if len(input.Sort) > 0 {
		sorts := make([]map[string]any, 0, len(input.Sort))
		for _, sort := range input.Sort {
			sorts = append(sorts, map[string]any{"field": sort.Field, "desc": sort.Desc})
		}
		gqlInput["sort"] = sorts
	}
	if input.PageSize > 0 {
		gqlInput["pageSize"] = input.PageSize
	}
	if input.Offset > 0 {
		gqlInput["offset"] = input.Offset
	}
	if input.ScanLimit > 0 {
		gqlInput["scanLimit"] = input.ScanLimit
	}
	var resp struct {
		OntologyQuery *DatasetQueryResult `json:"OntologyQuery"`
	}
	if err := c.graphQL(ctx, operationOntologyQuery, map[string]any{"projectId": projectID, "input": gqlInput}, &resp); err != nil {
		return nil, err
	}
	if resp.OntologyQuery == nil {
		return nil, fmt.Errorf("platform returned no ontology query result")
	}
	return resp.OntologyQuery, nil
}

// OntologyAggregate runs a SQL-pushdown aggregate over one ontology.
func (c *Client) OntologyAggregate(ctx context.Context, projectID string, input OntologyAggregateInput) (*DatasetQueryResult, error) {
	if err := c.RequireOperation("Query", "OntologyAggregate", "ontology aggregate"); err != nil {
		return nil, err
	}
	groupBy := input.GroupBy
	if groupBy == nil {
		groupBy = []string{}
	}
	variables := map[string]any{
		"projectId":  projectID,
		"id":         input.ID,
		"groupBy":    groupBy,
		"metrics":    ontologyMetricVariables(input.Metrics, strings.ToUpper),
		"pageSize":   input.PageSize,
		"pageOffset": input.PageOffset,
	}
	if len(input.Where) > 0 {
		variables["where"] = input.Where
	}
	if len(input.Sort) > 0 {
		variables["sort"] = input.Sort
	}
	var resp struct {
		OntologyAggregate *DatasetQueryResult `json:"OntologyAggregate"`
	}
	if err := c.graphQL(ctx, operationOntologyAggregate, variables, &resp); err != nil {
		return nil, err
	}
	if resp.OntologyAggregate == nil {
		return nil, fmt.Errorf("platform returned no ontology aggregate result")
	}
	return resp.OntologyAggregate, nil
}

func ontologyMetricVariables(metrics []OntologyMetric, normalizeOp func(string) string) []map[string]any {
	out := make([]map[string]any, 0, len(metrics))
	for _, metric := range metrics {
		entry := map[string]any{"op": normalizeOp(strings.TrimSpace(metric.Op))}
		if strings.TrimSpace(metric.Field) != "" {
			entry["field"] = strings.TrimSpace(metric.Field)
		}
		if strings.TrimSpace(metric.As) != "" {
			entry["as"] = strings.TrimSpace(metric.As)
		}
		out = append(out, entry)
	}
	return out
}

// OntologyStats returns min, max, total and non-null counts for one property.
func (c *Client) OntologyStats(ctx context.Context, projectID, id, property string, where map[string]any) (*OntologyStatsResult, error) {
	if err := c.RequireOperation("Query", "OntologyStats", "ontology stats"); err != nil {
		return nil, err
	}
	variables := map[string]any{"projectId": projectID, "id": id, "property": property}
	if len(where) > 0 {
		variables["where"] = where
	}
	var resp struct {
		OntologyStats *OntologyStatsResult `json:"OntologyStats"`
	}
	if err := c.graphQL(ctx, operationOntologyStats, variables, &resp); err != nil {
		return nil, err
	}
	if resp.OntologyStats == nil {
		return nil, fmt.Errorf("platform returned no ontology stats")
	}
	return resp.OntologyStats, nil
}

// OntologyDescribe returns a narrative and structured description of ontologies.
func (c *Client) OntologyDescribe(ctx context.Context, projectID string, entities []string, includeInferred bool) (*OntologyDescription, error) {
	if err := c.RequireOperation("Query", "OntologyDescribe", "ontology describe"); err != nil {
		return nil, err
	}
	if entities == nil {
		entities = []string{}
	}
	variables := map[string]any{"projectId": projectID, "input": map[string]any{"entities": entities, "includeInferred": includeInferred}}
	var resp struct {
		OntologyDescribe *OntologyDescription `json:"OntologyDescribe"`
	}
	if err := c.graphQL(ctx, operationOntologyDescribe, variables, &resp); err != nil {
		return nil, err
	}
	if resp.OntologyDescribe == nil {
		return nil, fmt.Errorf("platform returned no ontology description")
	}
	return resp.OntologyDescribe, nil
}

// OntologySimilar returns rows similar to one row using the ontology's embeddings.
func (c *Client) OntologySimilar(ctx context.Context, projectID string, input OntologySimilarInput) (*OntologySimilarityResult, error) {
	if err := c.RequireOperation("Query", "OntologySimilar", "ontology similarity"); err != nil {
		return nil, err
	}
	gqlInput := map[string]any{"entityId": input.EntityID, "rowId": input.RowID}
	if input.TopK > 0 {
		gqlInput["topK"] = input.TopK
	}
	if len(input.Properties) > 0 {
		gqlInput["properties"] = input.Properties
	}
	if len(input.Where) > 0 {
		gqlInput["where"] = input.Where
	}
	var resp struct {
		OntologySimilar *OntologySimilarityResult `json:"OntologySimilar"`
	}
	if err := c.graphQL(ctx, operationOntologySimilar, map[string]any{"projectId": projectID, "input": gqlInput}, &resp); err != nil {
		return nil, err
	}
	if resp.OntologySimilar == nil {
		return nil, fmt.Errorf("platform returned no similarity result")
	}
	return resp.OntologySimilar, nil
}

// OntologyRecordCapabilities reports readable/writable properties and allowed actions.
func (c *Client) OntologyRecordCapabilities(ctx context.Context, projectID, ontologyID, recordKey string) (*OntologyRecordCapabilities, error) {
	if err := c.RequireOperation("Query", "OntologyRecordCapabilities", "ontology record capabilities"); err != nil {
		return nil, err
	}
	variables := map[string]any{"projectId": projectID, "ontologyId": ontologyID}
	if strings.TrimSpace(recordKey) != "" {
		variables["recordKey"] = strings.TrimSpace(recordKey)
	}
	var resp struct {
		OntologyRecordCapabilities *OntologyRecordCapabilities `json:"OntologyRecordCapabilities"`
	}
	if err := c.graphQL(ctx, operationOntologyRecordCapabilities, variables, &resp); err != nil {
		return nil, err
	}
	if resp.OntologyRecordCapabilities == nil {
		return nil, fmt.Errorf("platform returned no record capabilities")
	}
	return resp.OntologyRecordCapabilities, nil
}

// PreviewOntologyAction evaluates a behavior action without executing it.
func (c *Client) PreviewOntologyAction(ctx context.Context, input PreviewOntologyActionInput) (*OntologyActionPreview, error) {
	if err := c.RequireOperation("Query", "PreviewOntologyAction", "ontology action preview"); err != nil {
		return nil, err
	}
	values := input.Values
	if values == nil {
		values = map[string]any{}
	}
	gqlInput := map[string]any{"projectId": input.ProjectID, "ontologyId": input.OntologyID, "action": input.Action, "values": values}
	if strings.TrimSpace(input.RecordKey) != "" {
		gqlInput["recordKey"] = strings.TrimSpace(input.RecordKey)
	}
	var resp struct {
		PreviewOntologyAction *OntologyActionPreview `json:"PreviewOntologyAction"`
	}
	if err := c.graphQL(ctx, operationPreviewOntologyAction, map[string]any{"input": gqlInput}, &resp); err != nil {
		return nil, err
	}
	if resp.PreviewOntologyAction == nil {
		return nil, fmt.Errorf("platform returned no action preview")
	}
	return resp.PreviewOntologyAction, nil
}

// OntologyActionExecutions lists recorded action executions for one record.
func (c *Client) OntologyActionExecutions(ctx context.Context, projectID, ontologyID, recordKey string, limit int) ([]OntologyActionExecution, error) {
	if err := c.RequireOperation("Query", "OntologyActionExecutions", "ontology action executions"); err != nil {
		return nil, err
	}
	variables := map[string]any{"projectId": projectID, "ontologyId": ontologyID, "recordKey": recordKey}
	if limit > 0 {
		variables["limit"] = limit
	}
	var resp struct {
		OntologyActionExecutions []OntologyActionExecution `json:"OntologyActionExecutions"`
	}
	err := c.graphQL(ctx, operationOntologyActionExecutions, variables, &resp)
	return resp.OntologyActionExecutions, err
}

// OntologyEmbeddingJobs lists embedding jobs for one ontology.
func (c *Client) OntologyEmbeddingJobs(ctx context.Context, projectID, entityID string) ([]OntologyEmbeddingJob, error) {
	if err := c.RequireOperation("Query", "OntologyEmbeddingJobs", "ontology embedding jobs"); err != nil {
		return nil, err
	}
	var resp struct {
		OntologyEmbeddingJobs []OntologyEmbeddingJob `json:"OntologyEmbeddingJobs"`
	}
	err := c.graphQL(ctx, operationOntologyEmbeddingJobs, map[string]any{"projectId": projectID, "entityId": entityID}, &resp)
	return resp.OntologyEmbeddingJobs, err
}

// PreviewTransform executes a transform graph without saving it and returns the result rows.
func (c *Client) PreviewTransform(ctx context.Context, projectID, graphJSON string) (*DatasetQueryResult, error) {
	if err := c.RequireOperation("Query", "PreviewTransform", "transform preview"); err != nil {
		return nil, err
	}
	var resp struct {
		PreviewTransform *DatasetQueryResult `json:"PreviewTransform"`
	}
	if err := c.graphQL(ctx, operationPreviewTransform, map[string]any{"projectId": projectID, "graphJson": graphJSON}, &resp); err != nil {
		return nil, err
	}
	if resp.PreviewTransform == nil {
		return nil, fmt.Errorf("platform returned no transform preview")
	}
	return resp.PreviewTransform, nil
}
