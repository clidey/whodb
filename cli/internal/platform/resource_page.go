package platform

import "context"

// ResourcePageInput selects a bounded platform metadata collection.
type ResourcePageInput struct {
	Resource   string `json:"resource"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	Search     string `json:"search,omitempty"`
	Name       string `json:"name,omitempty"`
	Type       string `json:"type,omitempty"`
	Status     string `json:"status,omitempty"`
	SchemaMode string `json:"schemaMode,omitempty"`
	Deployed   *bool  `json:"deployed,omitempty"`
}

// ResourcePage contains metadata and explicit continuation information.
type ResourcePage struct {
	Items   []map[string]any `json:"items"`
	HasMore bool             `json:"hasMore"`
}

// SupportsResourcePages reports whether the host advertises bounded collections.
func (c *Client) SupportsResourcePages() bool {
	return c.manifest != nil && c.manifest.RequireOperation("Query", "PlatformResources", "bounded collections") == nil
}

// ResourcePage retrieves only the requested metadata page from the host.
func (c *Client) ResourcePage(ctx context.Context, projectID string, input ResourcePageInput) (*ResourcePage, error) {
	var resp struct{ PlatformResources *ResourcePage }
	err := c.graphQL(ctx, `query CLIResourcePage($projectId: ID!, $input: PlatformResourcePageInput!) {
 PlatformResources(projectId:$projectId,input:$input) { items { id name description type status schemaMode isDeployed apiName } hasMore }
 }`, map[string]any{"projectId": projectID, string(GenericWriteModeInput): input}, &resp) //nolint:goconst // GraphQL variable names follow the operation schema.
	return resp.PlatformResources, err
}
