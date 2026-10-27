package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// PlatformExtensionInput is the bounded input for edition-owned hosted tools.
type PlatformExtensionInput struct {
	Org       string         `json:"org,omitempty" jsonschema:"Organization ID, slug or name; omit for the selected organization"`
	Selection map[string]any `json:"selection,omitempty" jsonschema:"Optional project/resource selection"`
	RequestID string         `json:"request_id,omitempty" jsonschema:"Stable idempotency ID for a manual request"`
	Policy    map[string]any `json:"policy,omitempty" jsonschema:"Complete policy for an owner-authorized configuration change"`
}

// PlatformExtensionOperation declares one edition-owned static GraphQL tool.
type PlatformExtensionOperation struct {
	Name        string
	Description string
	Operation   string
	Document    string
	Write       bool
	Variables   func(string, PlatformExtensionInput) (map[string]any, error)
}

var platformExtensions []PlatformExtensionOperation

// RegisterPlatformExtension adds one static tool before MCP servers are constructed.
func RegisterPlatformExtension(operation PlatformExtensionOperation) {
	if !strings.HasPrefix(operation.Name, "whodb_platform_") || operation.Operation == "" || operation.Document == "" || operation.Variables == nil {
		panic("invalid platform extension")
	}
	for _, existing := range platformExtensions {
		if existing.Name == operation.Name {
			panic("duplicate platform extension tool")
		}
	}
	if operation.Write {
		platformapi.RegisterPlatformMutation(operation.Operation, operation.Document)
	}
	platformExtensions = append(platformExtensions, operation)
}

func platformExtensionDefinitions() []*mcp.Tool {
	tools := make([]*mcp.Tool, 0, len(platformExtensions))
	for _, extension := range platformExtensions {
		annotations := platformReadOnlyAnnotations(extension.Name)
		if extension.Write {
			annotations = platformDestructiveAnnotations(extension.Name)
		}
		tools = append(tools, &mcp.Tool{Name: extension.Name, Description: extension.Description, Annotations: annotations})
	}
	return tools
}

func registerPlatformExtension(server *mcp.Server, tool *mcp.Tool, options *SecurityOptions) bool {
	for _, extension := range platformExtensions {
		if extension.Name != tool.Name {
			continue
		}
		if extension.Write && options.ReadOnly {
			return true
		}
		addPlatformTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformExtensionInput) (*mcp.CallToolResult, any, error) {
			session, err := loadPlatformToolSession(ctx)
			if err != nil {
				return nil, nil, err
			}
			org, err := resolvePlatformToolOrg(ctx, session, input.Org)
			if err != nil {
				return nil, nil, err
			}
			variables, err := extension.Variables(org.ID, input)
			if err != nil {
				return nil, nil, err
			}
			if variables == nil {
				variables = map[string]any{}
			}
			if !extension.Write {
				client, ok := session.Client.(interface {
					RequireOperation(string, string, string) error
					ExecuteGraphQL(context.Context, string, any, any) error
				})
				if !ok {
					return nil, nil, errors.New("platform extension client is unavailable")
				}
				if err := client.RequireOperation("Query", extension.Operation, "platform read"); err != nil {
					return nil, nil, err
				}
				var result map[string]json.RawMessage
				if err := client.ExecuteGraphQL(ctx, extension.Document, variables, &result); err != nil {
					return nil, nil, err
				}
				value, ok := result[extension.Operation]
				if !ok {
					return nil, nil, errors.New("platform response omitted requested operation")
				}
				output := map[string]any{"org_id": org.ID, "result": value}
				if org.Slug != "" {
					output["recovery_url"] = session.Host.URL + "/" + url.PathEscape(org.Slug) + "/platform/organization?tab=backups"
				}
				return nil, output, nil
			}
			action := &PendingPlatformAction{AccountID: session.Host.AccountID, Operation: extension.Operation, Resource: "organization", Action: extension.Operation, Host: session.Host.URL, OrgID: org.ID, Mutation: extension.Operation, Variables: variables, Summary: extension.Description, Changes: []string{extension.Description + " in " + org.Name}}
			return platformMutationWriteOutput(ctx, generateRequestID(extension.Name), extension.Name, action, options.ConfirmWrites)
		})
		return true
	}
	return false
}
