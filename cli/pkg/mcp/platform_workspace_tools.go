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
	"sort"
	"strconv"
	"strings"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformWorkspaceMapInput is the input for the whodb_platform_workspace_map tool.
type PlatformWorkspaceMapInput struct {
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder summaries. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit project lineage summary. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example counts, sources, datasets, warnings."`
}

// PlatformResourceGraphInput is the input for the whodb_platform_resource_graph tool.
type PlatformResourceGraphInput struct {
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder nodes. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit hosted lineage edges. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example nodes, edges, warnings."`
}

// PlatformNextActionsInput is the input for the whodb_platform_next_actions tool.
type PlatformNextActionsInput struct {
	Goal        string   `json:"goal,omitempty" jsonschema:"Optional user goal used to keep suggested actions relevant."`
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder summaries. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit project lineage summary. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example actions, warnings, goal."`
}

// PlatformWorkspaceItem is a compact platform resource summary for agents.
type PlatformWorkspaceItem struct {
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type"`
	Name     string            `json:"name,omitempty"`
	Status   string            `json:"status,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PlatformLineageSummary is a compact lineage overview.
type PlatformLineageSummary struct {
	NodeCount int `json:"node_count"`
	EdgeCount int `json:"edge_count"`
}

// PlatformWorkspaceMap is a project-level map of hosted platform resources.
type PlatformWorkspaceMap struct {
	Host        string                  `json:"host,omitempty"`
	OrgID       string                  `json:"org_id,omitempty"`
	OrgName     string                  `json:"org_name,omitempty"`
	ProjectID   string                  `json:"project_id,omitempty"`
	ProjectName string                  `json:"project_name,omitempty"`
	Counts      map[string]int          `json:"counts"`
	Sources     []PlatformWorkspaceItem `json:"sources"`
	Secrets     []PlatformWorkspaceItem `json:"secrets"`
	AIProviders []PlatformWorkspaceItem `json:"ai_providers"`
	Datasets    []PlatformWorkspaceItem `json:"datasets"`
	Ontologies  []PlatformWorkspaceItem `json:"ontologies"`
	Transforms  []PlatformWorkspaceItem `json:"transforms"`
	Functions   []PlatformWorkspaceItem `json:"functions"`
	Files       []PlatformWorkspaceItem `json:"files,omitempty"`
	Folders     []PlatformWorkspaceItem `json:"folders,omitempty"`
	StorageUsed int                     `json:"storage_used"`
	Lineage     *PlatformLineageSummary `json:"lineage,omitempty"`
	Warnings    []string                `json:"warnings"`
}

// PlatformResourceGraphNode is one node in the hosted platform resource graph.
type PlatformResourceGraphNode struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Name     string            `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PlatformResourceGraphEdge is one relationship in the hosted platform resource graph.
type PlatformResourceGraphEdge struct {
	FromID   string `json:"from_id"`
	FromType string `json:"from_type"`
	ToID     string `json:"to_id"`
	ToType   string `json:"to_type"`
	Kind     string `json:"kind"`
}

// PlatformResourceGraph is a normalized relationship graph for hosted platform resources.
type PlatformResourceGraph struct {
	Nodes    []PlatformResourceGraphNode `json:"nodes"`
	Edges    []PlatformResourceGraphEdge `json:"edges"`
	Counts   map[string]int              `json:"counts"`
	Warnings []string                    `json:"warnings"`
}

// PlatformNextAction describes one deterministic suggested next step for an agent.
type PlatformNextAction struct {
	Priority       int      `json:"priority"`
	Area           string   `json:"area"`
	Title          string   `json:"title"`
	Reason         string   `json:"reason"`
	SuggestedTools []string `json:"suggested_tools"`
	ReadOnly       bool     `json:"read_only"`
}

// PlatformNextActions describes suggested next steps based on the selected project.
type PlatformNextActions struct {
	Goal     string               `json:"goal,omitempty"`
	Actions  []PlatformNextAction `json:"actions"`
	Warnings []string             `json:"warnings"`
}

type platformWorkspaceSnapshot struct {
	session     *platformToolSession
	sources     []platformapi.Source
	secrets     []platformapi.ProjectSecret
	providers   []platformapi.AIProvider
	datasets    []platformapi.Dataset
	ontologies  []platformapi.Ontology
	transforms  []platformapi.Transform
	functions   []platformapi.Function
	folders     []platformapi.ProjectFolder
	files       []platformapi.ProjectFile
	storageUsed int
	lineage     *platformapi.LineageGraph
	warnings    []string
	omitFiles   bool
	omitLineage bool
}

// HandlePlatformWorkspaceMap returns a compact selected-project workspace map.
func HandlePlatformWorkspaceMap(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkspaceMapInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_workspace_map", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		workspaceMap := buildPlatformWorkspaceMap(snapshot)
		return workspaceMap, sumPlatformCounts(workspaceMap.Counts), false, nil
	})
}

// HandlePlatformResourceGraph returns a normalized selected-project resource graph.
func HandlePlatformResourceGraph(ctx context.Context, req *mcp.CallToolRequest, input PlatformResourceGraphInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_resource_graph", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		graph := buildPlatformResourceGraph(snapshot)
		return graph, len(graph.Nodes), false, nil
	})
}

// HandlePlatformNextActions returns deterministic next steps for the selected project.
func HandlePlatformNextActions(ctx context.Context, req *mcp.CallToolRequest, input PlatformNextActionsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_next_actions", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		next := buildPlatformNextActions(snapshot, input.Goal)
		return next, len(next.Actions), false, nil
	})
}

func loadPlatformWorkspaceSnapshot(ctx context.Context, session *platformToolSession, omitFiles, omitLineage bool) (*platformWorkspaceSnapshot, error) {
	projectID := session.Host.DefaultProjectID
	snapshot := &platformWorkspaceSnapshot{
		session:     session,
		omitFiles:   omitFiles,
		omitLineage: omitLineage,
	}

	var err error
	if snapshot.sources, err = session.Client.ProjectSources(ctx, session.Host.DefaultOrgID, projectID); err != nil {
		return nil, err
	}
	if snapshot.secrets, err = session.Client.ProjectSecrets(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.providers, err = session.Client.AIProviders(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.datasets, err = session.Client.Datasets(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.ontologies, err = session.Client.Ontologies(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.transforms, err = session.Client.Transforms(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.functions, err = session.Client.Functions(ctx, projectID, []string{"id", "name", "language", "entryPoint", "isDeployed", "providerIds", "ontologyIds", "readOnlyOntologyIds", "providerConfigs", "secretBindings"}); err != nil {
		return nil, err
	}
	if !omitFiles {
		contents, err := session.Client.FolderContents(ctx, projectID, "", []string{"folders", "files", "storageUsed"})
		if err != nil {
			return nil, err
		}
		if contents != nil {
			snapshot.folders = contents.Folders
			snapshot.files = contents.Files
			snapshot.storageUsed = contents.StorageUsed
		}
	}
	usage, err := session.Client.ProjectStorageUsage(ctx, projectID)
	if err != nil {
		return nil, err
	}
	snapshot.storageUsed = usage
	if !omitLineage {
		snapshot.lineage, err = session.Client.ProjectLineage(ctx, projectID)
		if err != nil {
			return nil, err
		}
	}
	snapshot.warnings = platformWorkspaceWarnings(snapshot)
	return snapshot, nil
}

func buildPlatformWorkspaceMap(snapshot *platformWorkspaceSnapshot) PlatformWorkspaceMap {
	scope := platformScope(snapshot.session)
	workspaceMap := PlatformWorkspaceMap{
		Host:        scope.Host,
		OrgID:       scope.OrgID,
		OrgName:     scope.OrgName,
		ProjectID:   scope.ProjectID,
		ProjectName: scope.ProjectName,
		Counts:      platformWorkspaceCounts(snapshot),
		Sources:     sourceItems(snapshot.sources),
		Secrets:     secretItems(snapshot.secrets),
		AIProviders: providerItems(snapshot.providers),
		Datasets:    datasetItems(snapshot.datasets),
		Ontologies:  ontologyItems(snapshot.ontologies),
		Transforms:  transformItems(snapshot.transforms),
		Functions:   functionItems(snapshot.functions),
		Files:       fileItems(snapshot.files),
		Folders:     folderItems(snapshot.folders),
		StorageUsed: snapshot.storageUsed,
		Warnings:    append([]string(nil), snapshot.warnings...),
	}
	if snapshot.lineage != nil {
		workspaceMap.Lineage = &PlatformLineageSummary{NodeCount: len(snapshot.lineage.Nodes), EdgeCount: len(snapshot.lineage.Edges)}
	}
	return workspaceMap
}

func buildPlatformResourceGraph(snapshot *platformWorkspaceSnapshot) PlatformResourceGraph {
	nodesByKey := map[string]PlatformResourceGraphNode{}
	var edges []PlatformResourceGraphEdge
	addNode := func(id, nodeType, name string, metadata map[string]string) {
		if id == "" {
			return
		}
		key := nodeType + ":" + id
		if _, ok := nodesByKey[key]; ok {
			return
		}
		nodesByKey[key] = PlatformResourceGraphNode{ID: id, Type: nodeType, Name: name, Metadata: metadata}
	}
	addEdge := func(fromID, fromType, toID, toType, kind string) {
		if fromID == "" || toID == "" {
			return
		}
		edges = append(edges, PlatformResourceGraphEdge{FromID: fromID, FromType: fromType, ToID: toID, ToType: toType, Kind: kind})
	}

	for _, source := range snapshot.sources {
		addNode(source.ID, "source", source.Name, map[string]string{"database_type": source.DatabaseType})
	}
	for _, secret := range snapshot.secrets {
		addNode(secret.ID, "secret", secret.Name, nil)
		for _, usage := range secret.UsedBy {
			addEdge(secret.ID, "secret", usage.ConsumerID, normalizeGraphType(usage.ConsumerType), "secret_binding")
		}
	}
	for _, provider := range snapshot.providers {
		addNode(provider.ID, "ai_provider", provider.Name, map[string]string{"provider_type": provider.ProviderType})
	}
	for _, dataset := range snapshot.datasets {
		addNode(dataset.ID, "dataset", dataset.Name, map[string]string{"schema_mode": dataset.SchemaMode})
		addEdge(dataset.SourceID, "source", dataset.ID, "dataset", "source_dataset")
	}
	ontologyIDByAPIName := map[string]string{}
	for _, ontology := range snapshot.ontologies {
		if strings.TrimSpace(ontology.APIName) != "" {
			ontologyIDByAPIName[ontology.APIName] = ontology.ID
		}
	}
	for _, ontology := range snapshot.ontologies {
		addNode(ontology.ID, "ontology", ontologyDisplayName(ontology), map[string]string{"api_name": ontology.APIName, "status": ontology.Status})
		if ontology.SourceID != nil {
			addEdge(*ontology.SourceID, "source", ontology.ID, "ontology", "source_ontology")
		}
		for _, link := range ontology.Links {
			targetID := ontologyIDByAPIName[link.TargetOntologyAPIName]
			if targetID == "" {
				targetID = link.TargetOntologyAPIName
			}
			addEdge(ontology.ID, "ontology", targetID, "ontology", "ontology_link:"+link.APIName)
		}
	}
	for _, transform := range snapshot.transforms {
		addNode(transform.ID, "transform", transform.Name, map[string]string{"trigger_mode": transform.TriggerMode})
	}
	for _, fn := range snapshot.functions {
		addNode(fn.ID, "function", fn.Name, map[string]string{"language": fn.Language, "deployed": boolString(fn.IsDeployed)})
		for _, providerID := range fn.ProviderIDs {
			addEdge(fn.ID, "function", providerID, "ai_provider", "uses_provider")
		}
		for _, config := range fn.ProviderConfigs {
			addEdge(fn.ID, "function", config.ProviderID, "ai_provider", "uses_provider_model")
		}
		for _, ontologyID := range fn.OntologyIDs {
			addEdge(fn.ID, "function", ontologyID, "ontology", "writes_ontology")
		}
		for _, ontologyID := range fn.ReadOnlyOntologyIDs {
			addEdge(fn.ID, "function", ontologyID, "ontology", "reads_ontology")
		}
		for _, binding := range fn.SecretBindings {
			addEdge(fn.ID, "function", binding.SecretID, "secret", "uses_secret")
		}
	}
	for _, folder := range snapshot.folders {
		addNode(folder.ID, "folder", folder.Name, map[string]string{"path": folder.Path})
		if folder.ParentID != nil {
			addEdge(*folder.ParentID, "folder", folder.ID, "folder", "contains")
		}
	}
	for _, file := range snapshot.files {
		addNode(file.ID, "file", file.Name, map[string]string{"mime_type": file.MIMEType, "tabular": boolString(file.IsTabular)})
		if file.FolderID != nil {
			addEdge(*file.FolderID, "folder", file.ID, "file", "contains")
		}
		if file.DatasetID != nil {
			addEdge(file.ID, "file", *file.DatasetID, "dataset", "promoted_to_dataset")
		}
	}
	if snapshot.lineage != nil {
		for _, node := range snapshot.lineage.Nodes {
			addNode(node.ID, normalizeGraphType(node.NodeType), node.Name, nil)
		}
		for _, edge := range snapshot.lineage.Edges {
			addEdge(edge.SourceID, normalizeGraphType(edge.SourceType), edge.TargetID, normalizeGraphType(edge.TargetType), "lineage")
		}
	}

	nodes := make([]PlatformResourceGraphNode, 0, len(nodesByKey))
	for _, node := range nodesByKey {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Type == nodes[j].Type {
			return nodes[i].ID < nodes[j].ID
		}
		return nodes[i].Type < nodes[j].Type
	})
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Kind == edges[j].Kind {
			if edges[i].FromType == edges[j].FromType {
				if edges[i].FromID == edges[j].FromID {
					return edges[i].ToID < edges[j].ToID
				}
				return edges[i].FromID < edges[j].FromID
			}
			return edges[i].FromType < edges[j].FromType
		}
		return edges[i].Kind < edges[j].Kind
	})
	return PlatformResourceGraph{Nodes: nodes, Edges: edges, Counts: platformWorkspaceCounts(snapshot), Warnings: append([]string(nil), snapshot.warnings...)}
}

func buildPlatformNextActions(snapshot *platformWorkspaceSnapshot, goal string) PlatformNextActions {
	goal = strings.TrimSpace(goal)
	actions := make([]PlatformNextAction, 0)
	add := func(priority int, area, title, reason string, tools ...string) {
		actions = append(actions, PlatformNextAction{Priority: priority, Area: area, Title: title, Reason: reason, SuggestedTools: tools, ReadOnly: allReadOnlyPlatformTools(tools)})
	}

	if len(snapshot.sources) == 0 {
		add(10, "sources", "Connect a source", "The project has no hosted sources, so agents cannot browse source schemas or sample source data.", "whodb_platform_source_types", "whodb_platform_source_fields", "whodb_platform_source_create")
	} else {
		add(20, "sources", "Inspect source structure", "The project has hosted sources. Browse objects and columns before creating datasets or ontologies from source data.", "whodb_platform_sources", "whodb_platform_source_objects", "whodb_platform_source_columns")
	}
	if len(snapshot.files) > 0 && len(snapshot.datasets) == 0 {
		add(30, "files", "Inspect tabular files for dataset promotion", "The project has files but no datasets. Inspect tabular files before promoting one to a dataset.", "whodb_platform_files", "whodb_platform_tabular_files", "whodb_platform_file_inspect", "whodb_platform_promote_file_to_dataset")
	}
	if len(snapshot.datasets) == 0 {
		add(40, "datasets", "Create or promote a dataset", "The project has no datasets, so dataset rows, ontology backing data, and downstream transforms have nothing durable to operate on.", "whodb_platform_create_dataset", "whodb_platform_promote_file_to_dataset")
	} else {
		add(50, "datasets", "Review dataset shape", "Datasets exist. Inspect schema and preview rows before modeling ontologies or transforms.", "whodb_platform_datasets", "whodb_platform_dataset", "whodb_platform_dataset_rows")
	}
	if len(snapshot.ontologies) == 0 && len(snapshot.datasets) > 0 {
		add(60, "ontology", "Model datasets as ontologies", "Datasets exist but no ontology object types were returned for this project.", "whodb_platform_datasets", "whodb_platform_create")
	} else if len(snapshot.ontologies) > 0 {
		add(70, "ontology", "Inspect ontology relationships", "Ontology object types exist. Inspect links and fast lookup suggestions before adding records or functions.", "whodb_platform_ontologies", "whodb_platform_ontology", "whodb_platform_ontology_fast_lookup_suggestions")
	}
	if len(snapshot.providers) == 0 {
		add(80, "ai_providers", "Configure an AI provider if AI-backed functions are needed", "No AI provider metadata was returned. Functions that call model providers need a configured provider.", "whodb_platform_ai_providers", "whodb_platform_create")
	}
	if len(snapshot.functions) == 0 && len(snapshot.ontologies) > 0 {
		add(90, "functions", "Add ontology functions when behavior is needed", "Ontologies exist but no functions were returned. Functions can add behavior around ontology data.", "whodb_platform_functions", "whodb_platform_create")
	} else if undeployedFunctionCount(snapshot.functions) > 0 {
		add(100, "functions", "Review undeployed functions", "At least one function is not deployed. Inspect it before deploying or redeploying.", "whodb_platform_functions", "whodb_platform_function", "whodb_platform_action")
	}
	if len(snapshot.transforms) == 0 && len(snapshot.datasets) > 0 {
		add(110, "transforms", "Add transforms for repeatable data movement", "Datasets exist but no transforms were returned for this project.", "whodb_platform_transforms", "whodb_platform_create")
	} else if len(snapshot.transforms) > 0 {
		add(120, "transforms", "Check transform runs", "Transforms exist. Inspect recent runs before changing or running a transform.", "whodb_platform_transforms", "whodb_platform_transform_runs")
	}
	if snapshot.lineage != nil && len(snapshot.lineage.Edges) > 0 {
		add(130, "lineage", "Use lineage before changing resources", "Project lineage has relationships. Check affected upstream and downstream resources before writes.", "whodb_platform_project_lineage", "whodb_platform_lineage_neighbors")
	}
	if len(actions) == 0 {
		add(140, "workspace", "Inspect workspace map", "The project has little returned metadata. Start with a compact map and status before deciding on writes.", "whodb_platform_status", "whodb_platform_workspace_map")
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Priority < actions[j].Priority })
	return PlatformNextActions{Goal: goal, Actions: actions, Warnings: append([]string(nil), snapshot.warnings...)}
}

func platformWorkspaceCounts(snapshot *platformWorkspaceSnapshot) map[string]int {
	counts := map[string]int{
		"sources":      len(snapshot.sources),
		"secrets":      len(snapshot.secrets),
		"ai_providers": len(snapshot.providers),
		"datasets":     len(snapshot.datasets),
		"ontologies":   len(snapshot.ontologies),
		"transforms":   len(snapshot.transforms),
		"functions":    len(snapshot.functions),
		"folders":      len(snapshot.folders),
		"files":        len(snapshot.files),
	}
	if snapshot.lineage != nil {
		counts["lineage_nodes"] = len(snapshot.lineage.Nodes)
		counts["lineage_edges"] = len(snapshot.lineage.Edges)
	}
	return counts
}

func sumPlatformCounts(counts map[string]int) int {
	total := 0
	for key, count := range counts {
		if strings.HasPrefix(key, "lineage_") {
			continue
		}
		total += count
	}
	return total
}

func platformWorkspaceWarnings(snapshot *platformWorkspaceSnapshot) []string {
	warnings := make([]string, 0)
	if snapshot.omitFiles {
		warnings = append(warnings, "file summaries omitted by request")
	}
	if snapshot.omitLineage {
		warnings = append(warnings, "lineage summary omitted by request")
	}
	if len(snapshot.sources) == 0 {
		warnings = append(warnings, "no hosted sources returned for the selected project")
	}
	if len(snapshot.datasets) == 0 && len(snapshot.files) == 0 {
		warnings = append(warnings, "no datasets or root files returned for the selected project")
	}
	return warnings
}

func sourceItems(sources []platformapi.Source) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(sources))
	for _, source := range sources {
		items = append(items, PlatformWorkspaceItem{ID: source.ID, Type: "source", Name: source.Name, Metadata: map[string]string{"database_type": source.DatabaseType}})
	}
	return items
}

func secretItems(secrets []platformapi.ProjectSecret) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(secrets))
	for _, secret := range secrets {
		items = append(items, PlatformWorkspaceItem{ID: secret.ID, Type: "secret", Name: secret.Name, Metadata: map[string]string{"used_by_count": intString(len(secret.UsedBy))}})
	}
	return items
}

func providerItems(providers []platformapi.AIProvider) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(providers))
	for _, provider := range providers {
		items = append(items, PlatformWorkspaceItem{ID: provider.ID, Type: "ai_provider", Name: provider.Name, Metadata: map[string]string{"provider_type": provider.ProviderType}})
	}
	return items
}

func datasetItems(datasets []platformapi.Dataset) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(datasets))
	for _, dataset := range datasets {
		items = append(items, PlatformWorkspaceItem{ID: dataset.ID, Type: "dataset", Name: dataset.Name, Metadata: map[string]string{"schema_mode": dataset.SchemaMode, "rows": intString(dataset.RowCount)}})
	}
	return items
}

func ontologyItems(ontologies []platformapi.Ontology) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(ontologies))
	for _, ontology := range ontologies {
		items = append(items, PlatformWorkspaceItem{ID: ontology.ID, Type: "ontology", Name: ontologyDisplayName(ontology), Status: ontology.Status, Metadata: map[string]string{"api_name": ontology.APIName, "properties": intString(len(ontology.Properties)), "links": intString(len(ontology.Links))}})
	}
	return items
}

func transformItems(transforms []platformapi.Transform) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(transforms))
	for _, transform := range transforms {
		items = append(items, PlatformWorkspaceItem{ID: transform.ID, Type: "transform", Name: transform.Name, Metadata: map[string]string{"trigger_mode": transform.TriggerMode}})
	}
	return items
}

func functionItems(functions []platformapi.Function) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(functions))
	for _, fn := range functions {
		status := "draft"
		if fn.IsDeployed {
			status = "deployed"
		}
		items = append(items, PlatformWorkspaceItem{ID: fn.ID, Type: "function", Name: fn.Name, Status: status, Metadata: map[string]string{"language": fn.Language, "entry_point": fn.EntryPoint}})
	}
	return items
}

func fileItems(files []platformapi.ProjectFile) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(files))
	for _, file := range files {
		items = append(items, PlatformWorkspaceItem{ID: file.ID, Type: "file", Name: file.Name, Metadata: map[string]string{"mime_type": file.MIMEType, "tabular": boolString(file.IsTabular), "size_bytes": intString(file.SizeBytes)}})
	}
	return items
}

func folderItems(folders []platformapi.ProjectFolder) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(folders))
	for _, folder := range folders {
		items = append(items, PlatformWorkspaceItem{ID: folder.ID, Type: "folder", Name: folder.Name, Metadata: map[string]string{"path": folder.Path}})
	}
	return items
}

func normalizeGraphType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func ontologyDisplayName(ontology platformapi.Ontology) string {
	if strings.TrimSpace(ontology.DisplayName) != "" {
		return ontology.DisplayName
	}
	return ontology.APIName
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func undeployedFunctionCount(functions []platformapi.Function) int {
	count := 0
	for _, fn := range functions {
		if !fn.IsDeployed {
			count++
		}
	}
	return count
}

func allReadOnlyPlatformTools(tools []string) bool {
	for _, tool := range platformToolDefinitions() {
		for _, name := range tools {
			if tool.Name == name && tool.Annotations != nil && !tool.Annotations.ReadOnlyHint {
				return false
			}
		}
	}
	return true
}
