package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// PlatformCapabilityReadInput selects bounded operational metadata or result pages.
type PlatformCapabilityReadInput struct {
	Language     string   `json:"language,omitempty" jsonschema:"Script language filter, or required language for script_package"`
	Name         string   `json:"name,omitempty" jsonschema:"Curated package name for script_package"`
	Query        string   `json:"query,omitempty" jsonschema:"Script package search text"`
	Runtime      string   `json:"runtime,omitempty" jsonschema:"Script runtime filter"`
	Plan         string   `json:"plan,omitempty" jsonschema:"Plan key for plan_features or subscription_preview"`
	Interval     string   `json:"interval,omitempty" jsonschema:"Billing interval for subscription_preview"`
	TransformID  string   `json:"transform_id,omitempty" jsonschema:"Owning transform id; required for transform_run"`
	ID           string   `json:"id,omitempty" jsonschema:"Run or analysis artifact id; required for edge_run and analysis tools"`
	FunctionID   string   `json:"function_id,omitempty" jsonschema:"Function id; required for edge_runs and edge_schedules"`
	AppID        string   `json:"app_id,omitempty" jsonschema:"App id; required for app_access"`
	SessionID    string   `json:"session_id,omitempty" jsonschema:"Owned AI session id; required for agent_session"`
	FeatureKey   string   `json:"feature_key,omitempty" jsonschema:"Platform feature key; required for feature_access"`
	Search       string   `json:"search,omitempty" jsonschema:"App access directory search"`
	Limit        int      `json:"limit,omitempty" jsonschema:"Maximum results or rows; default 20, capped by server max rows"`
	Offset       int      `json:"offset,omitempty" jsonschema:"Offset for edge runner/schedule lists or app access"`
	PerTransform int      `json:"per_transform,omitempty" jsonschema:"Runs per transform; default 5, maximum 20"`
	Cursor       string   `json:"cursor,omitempty" jsonschema:"Opaque nextCursor from the previous analysis artifact page"`
	Fields       []string `json:"fields,omitempty" jsonschema:"Optional top-level output projection"`
}

func platformCapabilityReadDefinitions() []*mcp.Tool {
	tools := make([]*mcp.Tool, 0, len(platformapi.CapabilityReads))
	for _, c := range platformapi.CapabilityReads {
		tools = append(tools, &mcp.Tool{Name: "whodb_platform_" + c.Tool, Description: c.Description, Annotations: platformReadOnlyAnnotations(c.Tool), InputSchema: capabilityReadSchema(c)})
	}
	return tools
}
func registerPlatformCapabilityRead(server *mcp.Server, tool *mcp.Tool, opts *SecurityOptions) bool {
	for _, c := range platformapi.CapabilityReads {
		if tool.Name != "whodb_platform_"+c.Tool {
			continue
		}
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, in PlatformCapabilityReadInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformCapabilityRead(ctx, c, in, opts)
		})
		return true
	}
	return false
}
func handlePlatformCapabilityRead(ctx context.Context, c platformapi.CapabilityRead, in PlatformCapabilityReadInput, opts *SecurityOptions) (*mcp.CallToolResult, PlatformReadOutput, error) {
	fail := func(err error) (*mcp.CallToolResult, PlatformReadOutput, error) {
		return nil, platformReadErrorOutput(err, generateRequestID(c.Tool)), nil
	}
	if in.Limit < 0 || in.Offset < 0 || in.PerTransform < 0 || in.PerTransform > 20 {
		return fail(fmt.Errorf("limit and offset must be nonnegative; per_transform must be between 0 and 20"))
	}
	if c.Operation == "AppAccessDirectory" && in.Limit > 100 {
		return fail(fmt.Errorf("app access limit must be at most 100"))
	}
	vars := map[string]any{"language": strings.TrimSpace(in.Language), "name": strings.TrimSpace(in.Name), "plan": strings.TrimSpace(in.Plan), "interval": strings.TrimSpace(in.Interval), "transformId": strings.TrimSpace(in.TransformID), "id": strings.TrimSpace(in.ID), "functionId": strings.TrimSpace(in.FunctionID), "appId": strings.TrimSpace(in.AppID), "sessionId": strings.TrimSpace(in.SessionID), "featureKey": strings.TrimSpace(in.FeatureKey)}
	for _, key := range c.Required {
		if vars[key] == "" {
			return fail(fmt.Errorf("%s is required", snakeCasePlatformKey(key)))
		}
	}
	for k, v := range vars {
		if v == "" {
			delete(vars, k)
		}
	}
	requested := in.Limit
	if requested == 0 {
		requested = 20
		if c.Operation == "OAuthSecretProviders" {
			requested = 50
		}
	}
	if requested > 1000 {
		requested = 1000
	}
	limit := platformRowsLimit(requested, opts)
	vars["limit"] = limit
	vars["pageSize"] = limit
	vars["offset"] = in.Offset
	if in.PerTransform > 0 {
		vars["perTransform"] = in.PerTransform
	} else {
		vars["perTransform"] = 5
	}
	if in.Cursor != "" {
		vars["cursor"] = in.Cursor
	}
	if in.Query != "" {
		vars["query"] = in.Query
	}
	if in.Runtime != "" {
		vars["runtime"] = in.Runtime
	}
	if in.Search != "" {
		vars["search"] = in.Search
	}
	read := func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		vars["projectId"] = session.Host.DefaultProjectID
		data, err := session.Client.PlatformQuery(ctx, c.Operation, vars)
		count := platformQueryCount(data)
		truncated := false
		if list, ok := data.([]any); ok {
			start := 0
			if c.Operation == "EdgeRunners" || c.Operation == "EdgeSchedules" || c.Operation == "ListScriptPackages" || c.Operation == "SearchScriptPackages" || c.Operation == "PlanFeatures" || c.Operation == "AvailablePlans" {
				start = min(in.Offset, len(list))
			}
			end := min(start+limit, len(list))
			data = list[start:end]
			count = end - start
			truncated = end < len(list)
		}
		if page, ok := data.(map[string]any); ok {
			if rows, ok := page["rows"].([]any); ok {
				count = len(rows)
			}
			if users, ok := page["users"].([]any); ok {
				count = len(users)
				if total, ok := page["total"].(float64); ok {
					truncated = float64(in.Offset+count) < total
				}
			}
			if cursor, ok := page["nextCursor"].(string); ok {
				truncated = cursor != ""
			}
		}
		return data, count, truncated, err
	}
	if c.OrganizationScoped() {
		session, err := loadPlatformToolSession(ctx)
		if err != nil {
			return fail(err)
		}
		if session.Host.DefaultOrgID == "" {
			if _, err := resolvePlatformToolOrg(ctx, session, ""); err != nil {
				return fail(err)
			}
		}
		session.Host.DefaultProjectID = ""
		session.Host.DefaultProjectName = ""
		session.Client.SetWorkspaceContext(session.Host.DefaultOrgID, "")
		recordPlatformScope(ctx, session)
		data, count, truncated, err := read(ctx, session)
		if err != nil {
			return fail(err)
		}
		return nil, platformReadOutput(session, c.Tool, data, count, truncated, generateRequestID(c.Tool), in.Fields), nil
	}
	return platformProjectRead(ctx, c.Tool, in.Fields, read)
}

func capabilityReadSchema(c platformapi.CapabilityRead) *jsonschema.Schema {
	schema, err := jsonschema.For[PlatformCapabilityReadInput](nil)
	if err != nil {
		panic(err)
	}
	allowed := map[string]bool{"fields": true}
	for _, key := range c.Required {
		name := snakeCasePlatformKey(key)
		allowed[name] = true
		schema.Required = append(schema.Required, name)
	}
	switch c.Operation {
	case "AvailablePlans", "PlanFeatures":
		allowed["limit"] = true
		allowed["offset"] = true
	case "ListScriptPackages":
		allowed["language"] = true
		allowed["limit"] = true
		allowed["offset"] = true
	case "SearchScriptPackages":
		allowed["language"] = true
		allowed["query"] = true
		allowed["runtime"] = true
		allowed["limit"] = true
		allowed["offset"] = true
	case "EdgeRunners", "EdgeSchedules":
		allowed["limit"] = true
		allowed["offset"] = true
	case "EdgeFunctionRuns", "AgentSessions":
		allowed["limit"] = true
	case "ProjectTransformRuns":
		allowed["limit"] = true
		allowed["per_transform"] = true
	case "AppAccessDirectory":
		allowed["limit"] = true
		allowed["offset"] = true
		allowed["search"] = true
	case "AnalysisArtifactPage":
		allowed["limit"] = true
		allowed["cursor"] = true
	}
	for key := range schema.Properties {
		if !allowed[key] {
			delete(schema.Properties, key)
		}
	}
	return schema
}
