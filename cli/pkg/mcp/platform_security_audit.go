package mcp

import (
	"context"
	"errors"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformSecurityAuditInput filters organization-scoped security events.
type PlatformSecurityAuditInput struct {
	Mode      string `json:"mode,omitempty" jsonschema:"events (default), exports, or verify. Requires organization management permission."`
	ProjectID string `json:"project_id,omitempty"`
	ActorID   string `json:"actor_id,omitempty"`
	Action    string `json:"action,omitempty"`
	Outcome   string `json:"outcome,omitempty" jsonschema:"Exact event outcome: SUCCESS, FAILURE, or DENIED."`
	RequestID string `json:"request_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	Since     string `json:"since,omitempty" jsonschema:"Inclusive RFC3339 start time."`
	Until     string `json:"until,omitempty" jsonschema:"Inclusive RFC3339 end time."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Events per page, 1-200; default 50."`
	Cursor    string `json:"cursor,omitempty" jsonschema:"Continue event pagination with nextCursor from the preceding response."`
}

// HandlePlatformSecurityAudit reads the security ledger through authorized GraphQL.
func HandlePlatformSecurityAudit(ctx context.Context, req *mcp.CallToolRequest, input PlatformSecurityAuditInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_security_audit", nil, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		client, ok := session.Client.(interface {
			SecurityAudit(context.Context, platformapi.SecurityAuditInput) (map[string]any, error)
		})
		if !ok {
			return nil, 0, false, errors.New("security ledger access is unavailable")
		}
		data, err := client.SecurityAudit(ctx, platformapi.SecurityAuditInput{OrgID: session.Host.DefaultOrgID, Mode: input.Mode, ProjectID: input.ProjectID, ActorID: input.ActorID, Action: input.Action, Outcome: input.Outcome, RequestID: input.RequestID, RunID: input.RunID, Since: input.Since, Until: input.Until, Limit: input.Limit, Cursor: input.Cursor})
		return data, 1, false, err
	})
}
