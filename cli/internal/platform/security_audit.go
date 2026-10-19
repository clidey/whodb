package platform

import "context"

// SecurityAuditInput selects an authorized security-ledger operation. Workspace
// IDs are supplied by the session, not by untrusted free-form GraphQL text.
type SecurityAuditInput struct {
	OrgID     string `json:"orgId"`
	Mode      string `json:"mode"`
	ProjectID string `json:"projectId,omitempty"`
	ActorID   string `json:"actorId,omitempty"`
	Action    string `json:"action,omitempty"`
	Outcome   string `json:"outcome,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	RunID     string `json:"runId,omitempty"`
	Since     string `json:"since,omitempty"`
	Until     string `json:"until,omitempty"`
	Limit     int    `json:"limit"`
	Cursor    string `json:"cursor,omitempty"`
}

// SecurityAudit reads canonical events, export health or chain verification.
func (c *Client) SecurityAudit(ctx context.Context, input SecurityAuditInput) (map[string]any, error) {
	if input.Mode == "" {
		input.Mode = "events"
	}
	if input.Limit == 0 {
		input.Limit = 50
	}
	if err := c.RequireOperation("Query", "SecurityAudit", "security ledger access"); err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]any `json:"SecurityAudit"`
	}
	err := c.graphQL(ctx, `query CLIPlatformSecurityAudit($input: SecurityAuditInput!) { SecurityAudit(input: $input) }`, map[string]any{"input": input}, &result)
	return result.Data, err
}
