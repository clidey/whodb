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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// Ontology analytics, sharing, service account, lineage provider and package
// polish tools. Typed reads go through the platform client; list-shaped reads
// go through the PlatformQuery allow-list like the other extended read tools.

// PlatformOntologyMetricInput is one metric for ontology query and aggregate tools.
type PlatformOntologyMetricInput struct {
	Op    string `json:"op" jsonschema:"Metric operation. Query: count, count_distinct, sum, avg, min, max. Aggregate: COUNT, SUM, AVG, MIN, MAX."`
	Field string `json:"field,omitempty" jsonschema:"Ontology property to measure; omit for count(*)"`
	As    string `json:"as,omitempty" jsonschema:"Output column alias"`
}

// PlatformOntologyQueryJoinInput joins a second ontology into an ontology query.
type PlatformOntologyQueryJoinInput struct {
	Entity     string `json:"entity" jsonschema:"Ontology apiName to join"`
	LeftField  string `json:"left_field" jsonschema:"Property on the base entity"`
	RightField string `json:"right_field" jsonschema:"Property on the joined entity"`
	As         string `json:"as,omitempty" jsonschema:"Alias prefix for joined columns"`
	Kind       string `json:"kind,omitempty" jsonschema:"Join kind, for example inner or left"`
}

// PlatformOntologyQuerySortInput orders ontology query results.
type PlatformOntologyQuerySortInput struct {
	Field string `json:"field" jsonschema:"Property or metric alias to sort by"`
	Desc  bool   `json:"desc,omitempty" jsonschema:"Sort descending"`
}

// PlatformOntologyQueryInput is the input for whodb_platform_ontology_query.
type PlatformOntologyQueryInput struct {
	Entity       string                           `json:"entity" jsonschema:"Ontology apiName (not id). List with whodb_platform_ontologies."`
	Where        map[string]any                   `json:"where,omitempty" jsonschema:"Filter object: {prop: value}, {prop: {op: value}}, or {_and: [...]} / {_or: [...]}. Ops: eq, neq, gt, gte, lt, lte, contains, icontains, isnull."`
	Search       string                           `json:"search,omitempty" jsonschema:"Free-text search across searchable properties"`
	SearchFields []string                         `json:"search_fields,omitempty" jsonschema:"Restrict search to these properties"`
	Joins        []PlatformOntologyQueryJoinInput `json:"joins,omitempty" jsonschema:"Join other ontologies by matching properties"`
	GroupBy      []string                         `json:"group_by,omitempty" jsonschema:"Properties to group by; pair with metrics"`
	Metrics      []PlatformOntologyMetricInput    `json:"metrics,omitempty" jsonschema:"Metrics to compute, usually with group_by"`
	Sort         []PlatformOntologyQuerySortInput `json:"sort,omitempty" jsonschema:"Result ordering"`
	Limit        int                              `json:"limit,omitempty" jsonschema:"Maximum rows to return"`
	Offset       int                              `json:"offset,omitempty" jsonschema:"Row offset"`
	ScanLimit    int                              `json:"scan_limit,omitempty" jsonschema:"Rows scanned before filtering (default 1000, max 5000). Raise for large entities; use whodb_platform_ontology_aggregate for exact totals."`
	Fields       []string                         `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformSortConditionInput is a platform SortCondition for aggregate tools.
type PlatformSortConditionInput struct {
	Column    string `json:"column" jsonschema:"Group-by property or metric alias"`
	Direction string `json:"direction,omitempty" jsonschema:"ASC or DESC (default ASC)"`
}

// PlatformOntologyAggregateInput is the input for whodb_platform_ontology_aggregate.
type PlatformOntologyAggregateInput struct {
	ID      string                        `json:"id" jsonschema:"Ontology id"`
	GroupBy []string                      `json:"group_by,omitempty" jsonschema:"Properties to group by; empty for a single-row total"`
	Metrics []PlatformOntologyMetricInput `json:"metrics" jsonschema:"At least one metric: COUNT, SUM, AVG, MIN, MAX"`
	Where   map[string]any                `json:"where,omitempty" jsonschema:"Filter object using the same grammar as whodb_platform_ontology_query: {prop: value}, {prop: {op: value}}, _and/_or lists. Ops: eq, neq, gt, gte, lt, lte, contains, icontains, startswith, in, nin, between, isnull. A raw platform WhereCondition object is also accepted."`
	Sort    []PlatformSortConditionInput  `json:"sort,omitempty" jsonschema:"Result ordering by group column or metric alias"`
	Limit   int                           `json:"limit,omitempty" jsonschema:"Maximum groups to return"`
	Offset  int                           `json:"offset,omitempty" jsonschema:"Group offset"`
	Fields  []string                      `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformOntologyStatsInput is the input for whodb_platform_ontology_stats.
type PlatformOntologyStatsInput struct {
	ID       string         `json:"id" jsonschema:"Ontology id"`
	Property string         `json:"property" jsonschema:"Ontology property apiName"`
	Where    map[string]any `json:"where,omitempty" jsonschema:"Optional filter in the shared grammar, for example {\"status\":\"open\"}"`
	Fields   []string       `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformOntologyDescribeInput is the input for whodb_platform_ontology_describe.
type PlatformOntologyDescribeInput struct {
	Entities        []string `json:"entities,omitempty" jsonschema:"Ontology apiNames to describe; empty describes every ontology in the project"`
	IncludeInferred bool     `json:"include_inferred,omitempty" jsonschema:"Include relationships inferred from data, not just declared links"`
	Fields          []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformOntologySimilarInput is the input for whodb_platform_ontology_similar.
type PlatformOntologySimilarInput struct {
	EntityID   string         `json:"entity_id" jsonschema:"Ontology id with embeddingEnabled"`
	RowID      string         `json:"row_id" jsonschema:"Primary key of the reference row"`
	TopK       int            `json:"top_k,omitempty" jsonschema:"Number of neighbours (default 5)"`
	Properties []string       `json:"properties,omitempty" jsonschema:"Properties to include in results"`
	Where      map[string]any `json:"where,omitempty" jsonschema:"Optional filter in the shared grammar to pre-filter candidates"`
	Fields     []string       `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformOntologyRecordCapabilitiesInput is the input for whodb_platform_ontology_record_capabilities.
type PlatformOntologyRecordCapabilitiesInput struct {
	OntologyID string   `json:"ontology_id" jsonschema:"Ontology id"`
	RecordKey  string   `json:"record_key,omitempty" jsonschema:"Primary key of one record; omit for type-level capabilities"`
	Fields     []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformOntologyActionPreviewInput is the input for whodb_platform_ontology_action_preview.
type PlatformOntologyActionPreviewInput struct {
	OntologyID string         `json:"ontology_id" jsonschema:"Ontology id"`
	RecordKey  string         `json:"record_key,omitempty" jsonschema:"Primary key of the target record; omit for create-style actions"`
	Action     string         `json:"action" jsonschema:"Behavior action apiName"`
	Values     map[string]any `json:"values,omitempty" jsonschema:"Action input values"`
	Fields     []string       `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformOntologyActionExecutionsInput is the input for whodb_platform_ontology_action_executions.
type PlatformOntologyActionExecutionsInput struct {
	OntologyID string   `json:"ontology_id" jsonschema:"Ontology id"`
	RecordKey  string   `json:"record_key" jsonschema:"Primary key of the record"`
	Limit      int      `json:"limit,omitempty" jsonschema:"Maximum executions to return (default 20)"`
	Fields     []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformTransformPreviewInput is the input for whodb_platform_transform_preview.
type PlatformTransformPreviewInput struct {
	GraphJSON string   `json:"graph_json" jsonschema:"Transform graph JSON, same shape as the transform graphJson field"`
	Fields    []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformUserInput selects one organization member.
type PlatformUserInput struct {
	UserID string   `json:"user_id" jsonschema:"User id from whodb_platform_org_members"`
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformAppPageFilesInput selects one page of a hosted app.
type PlatformAppPageFilesInput struct {
	ID     string   `json:"id" jsonschema:"App id"`
	Page   string   `json:"page" jsonschema:"Page name from whodb_platform_app_views"`
	Env    string   `json:"env,omitempty" jsonschema:"Optional app environment"`
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformOntologyRecordsInput is the input for whodb_platform_add_ontology_records.
type PlatformOntologyRecordsInput struct {
	EntityID       string              `json:"entity_id" jsonschema:"Ontology id"`
	Rows           []map[string]string `json:"rows" jsonschema:"Rows as objects keyed by ontology property"`
	IdempotencyKey string              `json:"idempotency_key,omitempty" jsonschema:"Client-chosen key so a retried call does not insert the rows twice"`
}

func platformOntologyAnalyticsToolDefinitions() []*mcp.Tool {
	read := func(name, description string) *mcp.Tool {
		return &mcp.Tool{Name: name, Description: description, Annotations: platformReadOnlyAnnotations(description)}
	}
	return []*mcp.Tool{
		{Name: "whodb_platform_ontology_query", Description: descPlatformOntologyQuery, Annotations: platformReadOnlyAnnotations("Query Hosted Ontology Rows")},
		{Name: "whodb_platform_ontology_aggregate", Description: descPlatformOntologyAggregate, Annotations: platformReadOnlyAnnotations("Aggregate Hosted Ontology Rows")},
		{Name: "whodb_platform_ontology_stats", Description: descPlatformOntologyStats, Annotations: platformReadOnlyAnnotations("Summarize Hosted Ontology Property")},
		{Name: "whodb_platform_ontology_describe", Description: descPlatformOntologyDescribe, Annotations: platformReadOnlyAnnotations("Describe Hosted Ontologies")},
		{Name: "whodb_platform_ontology_similar", Description: descPlatformOntologySimilar, Annotations: platformReadOnlyAnnotations("Find Similar Hosted Ontology Rows")},
		{Name: "whodb_platform_ontology_record_capabilities", Description: descPlatformOntologyRecordCapabilities, Annotations: platformReadOnlyAnnotations("Inspect Hosted Ontology Record Capabilities")},
		{Name: "whodb_platform_ontology_action_preview", Description: descPlatformOntologyActionPreview, Annotations: platformReadOnlyAnnotations("Preview Hosted Ontology Action")},
		{Name: "whodb_platform_ontology_action_executions", Description: descPlatformOntologyActionExecutions, Annotations: platformReadOnlyAnnotations("List Hosted Ontology Action Executions")},
		{Name: "whodb_platform_ontology_embedding_jobs", Description: descPlatformOntologyEmbeddingJobs, Annotations: platformReadOnlyAnnotations("List Hosted Ontology Embedding Jobs")},
		{Name: "whodb_platform_transform_preview", Description: descPlatformTransformPreview, Annotations: platformReadOnlyAnnotations("Preview Hosted Transform Output")},
		read("whodb_platform_service_accounts", "List service accounts and key metadata in the selected organization. Keys never include secrets."),
		read("whodb_platform_lineage_providers", "List OpenLineage providers configured for the selected project."),
		read("whodb_platform_shared_ontologies", "List ontologies the selected project shares with other projects, including their consumers."),
		read("whodb_platform_discover_shared_ontologies", "List ontologies other projects share that the selected project may reference."),
		read("whodb_platform_ontology_references", "List shared ontologies referenced by the selected project and their status."),
		read("whodb_platform_ontology_reference_impact", "Count links, apps, functions, transforms and behaviors that depend on one ontology reference before removing it."),
		read("whodb_platform_user_owned_resources", "List resources owned by one organization member, for example before removing them."),
		read("whodb_platform_user_resource_access", "List resources one organization member can access and how that access was granted."),
		read("whodb_platform_share_dependents", "List resources that must be shared alongside a resource for it to work, with the default role for each."),
		read("whodb_platform_org_package_installations", "List package installations across the selected organization, optionally for one package."),
		read("whodb_platform_app_page_files", "Read the generated files for one page of a hosted app."),
	}
}

func registerPlatformOntologyAnalyticsTool(server *mcp.Server, tool *mcp.Tool, secOpts *SecurityOptions) bool {
	switch tool.Name {
	case "whodb_platform_ontology_query":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyQueryInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyQuery(ctx, req, input, secOpts)
		})
	case "whodb_platform_ontology_aggregate":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyAggregateInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyAggregate(ctx, req, input, secOpts)
		})
	case "whodb_platform_ontology_stats":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyStatsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyStats(ctx, req, input)
		})
	case "whodb_platform_ontology_describe":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyDescribeInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyDescribe(ctx, req, input)
		})
	case "whodb_platform_ontology_similar":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologySimilarInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologySimilar(ctx, req, input)
		})
	case "whodb_platform_ontology_record_capabilities":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyRecordCapabilitiesInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyRecordCapabilities(ctx, req, input)
		})
	case "whodb_platform_ontology_action_preview":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyActionPreviewInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyActionPreview(ctx, req, input)
		})
	case "whodb_platform_ontology_action_executions":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyActionExecutionsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyActionExecutions(ctx, req, input)
		})
	case "whodb_platform_ontology_embedding_jobs":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyEmbeddingJobs(ctx, req, input)
		})
	case "whodb_platform_transform_preview":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformTransformPreviewInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformTransformPreview(ctx, req, input, secOpts)
		})
	case "whodb_platform_service_accounts":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_service_accounts", "ServiceAccounts", map[string]any{}, input.Fields)
		})
	case "whodb_platform_lineage_providers":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_lineage_providers", "ProjectLineageProviders", map[string]any{}, input.Fields)
		})
	case "whodb_platform_shared_ontologies":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_shared_ontologies", "SharedOntologies", map[string]any{}, input.Fields)
		})
	case "whodb_platform_discover_shared_ontologies":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_discover_shared_ontologies", "DiscoverSharedOntologies", map[string]any{}, input.Fields)
		})
	case "whodb_platform_ontology_references":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_ontology_references", "OntologyReferences", map[string]any{}, input.Fields)
		})
	case "whodb_platform_ontology_reference_impact":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_ontology_reference_impact", "OntologyReferenceImpact", map[string]any{"referenceId": strings.TrimSpace(input.ID)}, input.Fields)
		})
	case "whodb_platform_user_owned_resources":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformUserInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_user_owned_resources", "UserOwnedResources", map[string]any{"userId": strings.TrimSpace(input.UserID)}, input.Fields)
		})
	case "whodb_platform_user_resource_access":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformUserInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_user_resource_access", "UserResourceAccess", map[string]any{"userId": strings.TrimSpace(input.UserID)}, input.Fields)
		})
	case "whodb_platform_share_dependents":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_share_dependents", "ShareDependents", map[string]any{"resourceType": strings.TrimSpace(input.ResourceType), "resourceId": strings.TrimSpace(input.ResourceID)}, input.Fields)
		})
	case "whodb_platform_org_package_installations":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPackageInput) (*mcp.CallToolResult, any, error) {
			variables := map[string]any{}
			if strings.TrimSpace(input.ID) != "" {
				variables["packageId"] = strings.TrimSpace(input.ID)
			}
			return handlePlatformExtendedQuery(ctx, "platform_org_package_installations", "OrganizationPackageInstallations", variables, input.Fields)
		})
	case "whodb_platform_app_page_files":
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAppPageFilesInput) (*mcp.CallToolResult, any, error) {
			if strings.TrimSpace(input.Page) == "" {
				return nil, PlatformReadOutput{Error: "page is required", RequestID: generateRequestID("platform_app_page_files")}, nil
			}
			return handlePlatformExtendedQuery(ctx, "platform_app_page_files", "AppPageFiles", map[string]any{"id": strings.TrimSpace(input.ID), "page": strings.TrimSpace(input.Page), "env": optionalPlatformString(input.Env)}, input.Fields)
		})
	default:
		return false
	}
	return true
}

// HandlePlatformOntologyQuery runs a bounded ontology query with filters, search, joins, grouping and metrics.
func HandlePlatformOntologyQuery(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyQueryInput, secOpts *SecurityOptions) (*mcp.CallToolResult, PlatformReadOutput, error) {
	entity := strings.TrimSpace(input.Entity)
	if entity == "" {
		return nil, PlatformReadOutput{Error: "entity is required", RequestID: generateRequestID("platform_ontology_query")}, nil
	}
	if input.Offset < 0 {
		return nil, PlatformReadOutput{Error: "offset must be non-negative", RequestID: generateRequestID("platform_ontology_query")}, nil
	}
	if input.ScanLimit > platformapi.OntologyQueryMaxScanLimit {
		return nil, PlatformReadOutput{Error: fmt.Sprintf("scan_limit must be at most %d", platformapi.OntologyQueryMaxScanLimit), RequestID: generateRequestID("platform_ontology_query")}, nil
	}
	if err := validateOntologyMetrics(input.Metrics, platformapi.OntologyQueryMetricOps, strings.ToLower); err != nil {
		return nil, platformReadErrorOutput(err, generateRequestID("platform_ontology_query")), nil
	}
	var partial *bool
	result, output, err := platformProjectRead(ctx, "platform_ontology_query", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		query := platformapi.OntologyQueryInput{
			Entity:       entity,
			Where:        input.Where,
			Search:       input.Search,
			SearchFields: input.SearchFields,
			GroupBy:      input.GroupBy,
			Metrics:      ontologyMetrics(input.Metrics),
			PageSize:     platformRowsLimit(input.Limit, secOpts),
			Offset:       input.Offset,
			ScanLimit:    input.ScanLimit,
		}
		for _, join := range input.Joins {
			query.Joins = append(query.Joins, platformapi.OntologyQueryJoin{Entity: join.Entity, LeftField: join.LeftField, RightField: join.RightField, As: join.As, Kind: join.Kind})
		}
		for _, sort := range input.Sort {
			query.Sort = append(query.Sort, platformapi.OntologyQuerySort{Field: sort.Field, Desc: sort.Desc})
		}
		rows, err := session.Client.OntologyQuery(ctx, session.Host.DefaultProjectID, query)
		if rows == nil {
			return rows, 0, false, err
		}
		partial = rows.Partial
		return rows, len(rows.Rows), partial == nil || *partial || rows.Total > len(rows.Rows), err
	})
	if output.Error == "" {
		if partial == nil {
			output.Warnings = append(output.Warnings, "the server did not report query completeness; results and totals may be partial.")
		} else if *partial {
			output.Warnings = append(output.Warnings, fmt.Sprintf("one or more source scans reached the scan limit; results and totals are partial. Raise scan_limit (max %d) or use whodb_platform_ontology_aggregate for exact counts.", platformapi.OntologyQueryMaxScanLimit))
		}
	}
	return result, output, err
}

// HandlePlatformOntologyAggregate runs a SQL aggregate over one ontology.
func HandlePlatformOntologyAggregate(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyAggregateInput, secOpts *SecurityOptions) (*mcp.CallToolResult, PlatformReadOutput, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return nil, PlatformReadOutput{Error: "id is required", RequestID: generateRequestID("platform_ontology_aggregate")}, nil
	}
	if len(input.Metrics) == 0 {
		return nil, PlatformReadOutput{Error: "metrics is required; use [{\"op\":\"COUNT\"}] for a row count", RequestID: generateRequestID("platform_ontology_aggregate")}, nil
	}
	if input.Offset < 0 {
		return nil, PlatformReadOutput{Error: "offset must be non-negative", RequestID: generateRequestID("platform_ontology_aggregate")}, nil
	}
	if err := validateOntologyMetrics(input.Metrics, platformapi.OntologyAggregateMetricOps, strings.ToUpper); err != nil {
		return nil, platformReadErrorOutput(err, generateRequestID("platform_ontology_aggregate")), nil
	}
	where, err := platformapi.NormalizeWhereCondition(input.Where)
	if err != nil {
		return nil, platformReadErrorOutput(err, generateRequestID("platform_ontology_aggregate")), nil
	}
	return platformProjectRead(ctx, "platform_ontology_aggregate", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		aggregate := platformapi.OntologyAggregateInput{
			ID:         id,
			GroupBy:    input.GroupBy,
			Metrics:    ontologyMetrics(input.Metrics),
			Where:      where,
			PageSize:   platformRowsLimit(input.Limit, secOpts),
			PageOffset: input.Offset,
		}
		for _, sort := range input.Sort {
			direction := strings.ToUpper(strings.TrimSpace(sort.Direction))
			if direction == "" {
				direction = "ASC"
			}
			aggregate.Sort = append(aggregate.Sort, platformapi.SortCondition{Column: sort.Column, Direction: direction})
		}
		rows, err := session.Client.OntologyAggregate(ctx, session.Host.DefaultProjectID, aggregate)
		if rows == nil {
			return rows, 0, false, err
		}
		return rows, len(rows.Rows), rows.Total > len(rows.Rows), err
	})
}

func validateOntologyMetrics(metrics []PlatformOntologyMetricInput, allowed []string, normalize func(string) string) error {
	for _, metric := range metrics {
		op := normalize(strings.TrimSpace(metric.Op))
		if op == "" {
			return fmt.Errorf("metrics[].op is required")
		}
		found := false
		for _, candidate := range allowed {
			if op == candidate {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unsupported metric op %q; use one of %s", metric.Op, strings.Join(allowed, ", "))
		}
	}
	return nil
}

func ontologyMetrics(metrics []PlatformOntologyMetricInput) []platformapi.OntologyMetric {
	out := make([]platformapi.OntologyMetric, 0, len(metrics))
	for _, metric := range metrics {
		out = append(out, platformapi.OntologyMetric{Op: metric.Op, Field: metric.Field, As: metric.As})
	}
	return out
}

// HandlePlatformOntologyStats summarises one ontology property.
func HandlePlatformOntologyStats(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyStatsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	property := strings.TrimSpace(input.Property)
	if property == "" {
		return nil, PlatformReadOutput{Error: "property is required", RequestID: generateRequestID("platform_ontology_stats")}, nil
	}
	where, err := platformapi.NormalizeWhereCondition(input.Where)
	if err != nil {
		return nil, platformReadErrorOutput(err, generateRequestID("platform_ontology_stats")), nil
	}
	return platformIDRead(ctx, "platform_ontology_stats", input.ID, input.Fields, func(ctx context.Context, session *platformToolSession, id string) (any, int, bool, error) {
		stats, err := session.Client.OntologyStats(ctx, session.Host.DefaultProjectID, id, property, where)
		if stats == nil {
			return stats, 0, false, err
		}
		return stats, 1, false, err
	})
}

// HandlePlatformOntologyDescribe returns a narrative and structured description of ontologies.
func HandlePlatformOntologyDescribe(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyDescribeInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_ontology_describe", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		entities := normalizedPlatformWriteStrings(input.Entities)
		if len(entities) == 0 {
			// The hosted resolver requires explicit names, so an empty request
			// describes every ontology in the project.
			ontologies, err := session.Client.Ontologies(ctx, session.Host.DefaultProjectID)
			if err != nil {
				return nil, 0, false, err
			}
			for _, ontology := range ontologies {
				entities = append(entities, ontology.APIName)
			}
			if len(entities) == 0 {
				return &platformapi.OntologyDescription{Entities: []platformapi.OntologyDescriptionEntity{}, Relationships: []platformapi.OntologyDescriptionRelationship{}}, 0, false, nil
			}
		}
		description, err := session.Client.OntologyDescribe(ctx, session.Host.DefaultProjectID, entities, input.IncludeInferred)
		if description == nil {
			return description, 0, false, err
		}
		return description, len(description.Entities), false, err
	})
}

// HandlePlatformOntologySimilar finds rows similar to one row using ontology embeddings.
func HandlePlatformOntologySimilar(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologySimilarInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.RowID) == "" {
		return nil, PlatformReadOutput{Error: "row_id is required", RequestID: generateRequestID("platform_ontology_similar")}, nil
	}
	where, err := platformapi.NormalizeWhereCondition(input.Where)
	if err != nil {
		return nil, platformReadErrorOutput(err, generateRequestID("platform_ontology_similar")), nil
	}
	return platformIDRead(ctx, "platform_ontology_similar", input.EntityID, input.Fields, func(ctx context.Context, session *platformToolSession, entityID string) (any, int, bool, error) {
		result, err := session.Client.OntologySimilar(ctx, session.Host.DefaultProjectID, platformapi.OntologySimilarInput{
			EntityID:   entityID,
			RowID:      strings.TrimSpace(input.RowID),
			TopK:       input.TopK,
			Properties: input.Properties,
			Where:      where,
		})
		if result == nil {
			return result, 0, false, err
		}
		return result, len(result.Rows), false, err
	})
}

// HandlePlatformOntologyRecordCapabilities reports readable and writable properties and allowed actions.
func HandlePlatformOntologyRecordCapabilities(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyRecordCapabilitiesInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformIDRead(ctx, "platform_ontology_record_capabilities", input.OntologyID, input.Fields, func(ctx context.Context, session *platformToolSession, ontologyID string) (any, int, bool, error) {
		capabilities, err := session.Client.OntologyRecordCapabilities(ctx, session.Host.DefaultProjectID, ontologyID, input.RecordKey)
		if capabilities == nil {
			return capabilities, 0, false, err
		}
		return capabilities, len(capabilities.Actions), false, err
	})
}

// HandlePlatformOntologyActionPreview evaluates a behavior action without executing it.
func HandlePlatformOntologyActionPreview(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyActionPreviewInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.Action) == "" {
		return nil, PlatformReadOutput{Error: "action is required", RequestID: generateRequestID("platform_ontology_action_preview")}, nil
	}
	return platformIDRead(ctx, "platform_ontology_action_preview", input.OntologyID, input.Fields, func(ctx context.Context, session *platformToolSession, ontologyID string) (any, int, bool, error) {
		preview, err := session.Client.PreviewOntologyAction(ctx, platformapi.PreviewOntologyActionInput{
			ProjectID:  session.Host.DefaultProjectID,
			OntologyID: ontologyID,
			RecordKey:  input.RecordKey,
			Action:     strings.TrimSpace(input.Action),
			Values:     input.Values,
		})
		if preview == nil {
			return preview, 0, false, err
		}
		return preview, 1, false, err
	})
}

// HandlePlatformOntologyActionExecutions lists recorded behavior action executions for one record.
func HandlePlatformOntologyActionExecutions(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyActionExecutionsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.RecordKey) == "" {
		return nil, PlatformReadOutput{Error: "record_key is required", RequestID: generateRequestID("platform_ontology_action_executions")}, nil
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	return platformIDRead(ctx, "platform_ontology_action_executions", input.OntologyID, input.Fields, func(ctx context.Context, session *platformToolSession, ontologyID string) (any, int, bool, error) {
		executions, err := session.Client.OntologyActionExecutions(ctx, session.Host.DefaultProjectID, ontologyID, strings.TrimSpace(input.RecordKey), limit)
		if executions == nil {
			executions = []platformapi.OntologyActionExecution{}
		}
		return executions, len(executions), false, err
	})
}

// HandlePlatformOntologyEmbeddingJobs lists embedding jobs for one ontology.
func HandlePlatformOntologyEmbeddingJobs(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformIDRead(ctx, "platform_ontology_embedding_jobs", input.ID, input.Fields, func(ctx context.Context, session *platformToolSession, entityID string) (any, int, bool, error) {
		jobs, err := session.Client.OntologyEmbeddingJobs(ctx, session.Host.DefaultProjectID, entityID)
		if jobs == nil {
			jobs = []platformapi.OntologyEmbeddingJob{}
		}
		return jobs, len(jobs), false, err
	})
}

// HandlePlatformTransformPreview executes a transform graph without saving it.
func HandlePlatformTransformPreview(ctx context.Context, req *mcp.CallToolRequest, input PlatformTransformPreviewInput, secOpts *SecurityOptions) (*mcp.CallToolResult, PlatformReadOutput, error) {
	graph := strings.TrimSpace(input.GraphJSON)
	if graph == "" {
		return nil, PlatformReadOutput{Error: "graph_json is required", RequestID: generateRequestID("platform_transform_preview")}, nil
	}
	if !json.Valid([]byte(graph)) {
		return nil, PlatformReadOutput{Error: "graph_json must be valid JSON", RequestID: generateRequestID("platform_transform_preview")}, nil
	}
	return platformProjectRead(ctx, "platform_transform_preview", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		rows, err := session.Client.PreviewTransform(ctx, session.Host.DefaultProjectID, graph)
		if rows == nil {
			return rows, 0, false, err
		}
		limit := platformRowsLimit(0, secOpts)
		truncated := false
		if len(rows.Rows) > limit {
			rows.Rows = rows.Rows[:limit]
			truncated = true
		}
		return rows, len(rows.Rows), truncated, err
	})
}

func handlePlatformOntologyRecordsWrite(ctx context.Context, input PlatformOntologyRecordsInput, confirmWrites bool) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	if strings.TrimSpace(input.EntityID) == "" {
		return typedWriteValidationError("platform_add_ontology_records", "entity_id is required")
	}
	if len(input.Rows) == 0 {
		return typedWriteValidationError("platform_add_ontology_records", "rows is required")
	}
	rows := make([]map[string]any, 0, len(input.Rows))
	for index, row := range input.Rows {
		values := recordInputPayload(row)
		if len(values) == 0 {
			return typedWriteValidationError("platform_add_ontology_records", fmt.Sprintf("rows[%d] has no values", index))
		}
		rows = append(rows, map[string]any{"values": values})
	}
	payload := map[string]any{"rows": rows}
	if strings.TrimSpace(input.IdempotencyKey) != "" {
		payload["idempotencyKey"] = strings.TrimSpace(input.IdempotencyKey)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: generateRequestID("platform_add_ontology_records")}, nil
	}
	return handlePlatformTypedGenericWrite(ctx, "whodb_platform_add_ontology_records", PlatformGenericWriteInput{Resource: "ontology", Action: "add_records", ID: strings.TrimSpace(input.EntityID), PayloadJSON: string(raw), IdempotencyKey: strings.TrimSpace(input.IdempotencyKey)}, "action", confirmWrites)
}

const descPlatformOntologyQuery = `Query ontology rows with filters, search, joins, grouping and metrics. Use this to answer data questions instead of paging through whodb_platform_ontology_rows.

where grammar: {"status":"open"}, {"amount":{"gte":100}}, {"_or":[{"region":"EU"},{"region":"UK"}]}. Operators: eq, neq, gt, gte, lt, lte, contains, icontains, isnull. group_by + metrics (count, count_distinct, sum, avg, min, max) return one row per group. The server scans at most scan_limit rows (default 1000, max 5000) before filtering, so totals on large entities may be partial; use whodb_platform_ontology_aggregate for exact SQL totals. Policy-protected ontologies reject this query; fall back to whodb_platform_ontology_rows. Results are capped by limit and the MCP --max-rows setting.`

const descPlatformOntologyAggregate = `Compute exact SQL aggregates (COUNT, SUM, AVG, MIN, MAX) over one ontology, optionally grouped and filtered.

Pass group_by for one row per group and metrics for the values; omit group_by for a single total row. where uses the same grammar as whodb_platform_ontology_query (for example {"status":"open","amount":{"gte":100}}); a raw platform WhereCondition object is also accepted. Prefer this over whodb_platform_ontology_query when exact totals matter or the entity is large. Results are capped by limit and the MCP --max-rows setting.`

const descPlatformOntologyStats = `Return min, max, total and non-null counts for one ontology property, optionally filtered with the shared where grammar. Cheap way to profile a column before querying.`

const descPlatformOntologyDescribe = `Describe ontologies in plain text plus structured entities, fields and relationships. Call this first when answering a data question so property names and links are known. Empty entities describes the whole project.`

const descPlatformOntologySimilar = `Find rows similar to one row using the ontology's embeddings. Requires embeddingEnabled on the ontology and a completed embedding job; otherwise the platform returns an error explaining what is missing. Start a backfill with whodb_platform_action reembed.`

const descPlatformOntologyRecordCapabilities = `Report which properties the signed-in user can read and write on an ontology record and which behavior actions are allowed, with denial codes. Call before writing or executing actions.`

const descPlatformOntologyActionPreview = `Preview a behavior action without executing it: whether it is allowed, the proposed changes and the events it would emit. Show the preview to the user, then execute with whodb_platform_action execute_behavior.`

const descPlatformOntologyActionExecutions = `List recorded behavior action executions for one record, newest first, including status and result summary.`

const descPlatformOntologyEmbeddingJobs = `List embedding backfill jobs for one ontology with progress and errors. Use after whodb_platform_action reembed and before whodb_platform_ontology_similar.`

const descPlatformTransformPreview = `Run a transform graph without saving it and return the resulting rows. Use to validate a graph before whodb_platform_create transform or whodb_platform_action run. Results are capped by the MCP --max-rows setting.`

const descPlatformAddOntologyRecords = `Add many rows to a hosted ontology in one call.

rows are plain objects keyed by ontology property. Pass idempotency_key so a retried call never inserts duplicates. Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`
