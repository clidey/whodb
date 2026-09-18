package whodb

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/clidey/whodb/sdk/packages/go/gen"
)

// DefaultHost is the hosted WhoDB platform endpoint.
const DefaultHost = "https://app.whodb.com"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Config configures the WhoDB client. Zero values fall back to the standard
// precedence: WHODB_API_KEY env → the whodb CLI's stored login; and inside
// the Functions runtime, the IPC transport is auto-detected.
type Config struct {
	// APIKey is a platform API key (whodb_sk_...). Highest-precedence credential.
	APIKey string
	// Token is a raw OIDC access token.
	Token string
	// TokenFunc is a caller-managed token callback.
	TokenFunc TokenFunc
	// Credentials is a fully custom provider (overrides the above).
	Credentials CredentialProvider
	// Org is the organization slug or ID. Optional with an API key.
	Org string
	// Project is the project slug or ID. Optional with a single-grant API key.
	Project string
	// Host overrides the platform endpoint (default https://app.whodb.com).
	Host string
	// Transport overrides the wire transport entirely (IPC, tests). When set,
	// org/project inputs are taken as IDs verbatim.
	Transport Transport
}

// Client is the WhoDB platform client.
type Client struct {
	transport      Transport
	http           *httpTransport // nil for custom transports
	cli            *cliCredentials
	usingAPIKey    bool
	skipResolution bool
	orgInput       string
	projectInput   string

	workspaceOnce sync.Once
	workspaceErr  error
	orgID         string
	project       string
}

// ManifestHash is the SHA-256 of the platform manifest this SDK release was
// generated from.
func ManifestHash() string { return gen.ManifestHash }

// New creates a client, applying the credential precedence from the SDK
// design: explicit Config → WHODB_API_KEY → CLI credentials; IPC transport
// auto-detected inside the Functions runtime.
func New(config Config) (*Client, error) {
	client := &Client{
		orgInput:     firstNonEmpty(config.Org, os.Getenv("WHODB_ORG")),
		projectInput: firstNonEmpty(config.Project, os.Getenv("WHODB_PROJECT")),
	}
	transport := config.Transport
	if transport == nil && config.Credentials == nil && config.APIKey == "" && config.Token == "" && config.TokenFunc == nil &&
		os.Getenv("WHODB_API_KEY") == "" && os.Getenv("WHODB_IPC_TOKEN") != "" {
		transport = NewIpcTransport(IpcConfig{})
	}
	if transport != nil {
		client.transport = transport
		client.skipResolution = true
		return client, nil
	}
	var credentials CredentialProvider
	switch {
	case config.Credentials != nil:
		credentials = config.Credentials
	case config.APIKey != "":
		credentials = staticCredentials{value: config.APIKey}
		client.usingAPIKey = true
	case config.Token != "":
		credentials = staticCredentials{value: config.Token}
	case config.TokenFunc != nil:
		credentials = config.TokenFunc
	case os.Getenv("WHODB_API_KEY") != "":
		credentials = staticCredentials{value: os.Getenv("WHODB_API_KEY")}
		client.usingAPIKey = true
	default:
		cli := newCliCredentials()
		credentials = cli
		client.cli = cli
	}
	host := firstNonEmpty(config.Host, os.Getenv("WHODB_HOST"), DefaultHost)
	client.http = newHTTPTransport(host, credentials)
	client.transport = client.http
	return client, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// execute runs one prepared operation and unwraps its payload, converting
// unknown-operation rejections into ErrVersion.
func (c *Client) execute(ctx context.Context, request gen.Request) (any, error) {
	data, err := c.transport.Execute(ctx, request.Operation, request.Document, request.Variables)
	if err != nil {
		return nil, interpretServerError(err)
	}
	return data[request.Operation], nil
}

// resolveWorkspace resolves org/project (slug or ID) to IDs once, stamping
// the workspace headers onto the HTTP transport. API keys may omit both —
// the platform discovers them via MyWorkspace.
func (c *Client) resolveWorkspace(ctx context.Context) error {
	c.workspaceOnce.Do(func() {
		orgInput := c.orgInput
		projectInput := c.projectInput
		if (orgInput == "" || projectInput == "") && c.cli != nil {
			if defaults, err := c.cli.defaults(ctx); err == nil {
				orgInput = firstNonEmpty(orgInput, defaults.OrgID)
				projectInput = firstNonEmpty(projectInput, defaults.ProjectID)
			}
		}
		if (orgInput == "" || projectInput == "") && c.usingAPIKey {
			warnIfFlagged("MyWorkspace")
			result, err := c.execute(ctx, gen.NewMyWorkspaceRequest(map[string]any{}))
			if err != nil {
				c.workspaceErr = err
				return
			}
			workspace, _ := result.(map[string]any)
			if orgID, ok := workspace["orgId"].(string); ok {
				orgInput = firstNonEmpty(orgInput, orgID)
			}
			if projectID, ok := workspace["projectId"].(string); ok {
				projectInput = firstNonEmpty(projectInput, projectID)
			}
			if projectInput == "" {
				c.workspaceErr = fmt.Errorf("%w: this API key has access to multiple (or zero) projects — set Project in Config or WHODB_PROJECT", ErrValidation)
				return
			}
		}
		if c.skipResolution {
			c.orgID, c.project = orgInput, projectInput
			return
		}
		if orgInput == "" || projectInput == "" {
			c.workspaceErr = fmt.Errorf("%w: org and project are required — set them in Config, WHODB_ORG/WHODB_PROJECT, or run: whodb use", ErrValidation)
			return
		}
		orgID := orgInput
		if !uuidPattern.MatchString(orgInput) {
			result, err := c.execute(ctx, gen.NewMyOrganizationsRequest(map[string]any{}))
			if err != nil {
				c.workspaceErr = err
				return
			}
			orgID = matchWorkspaceEntry(result, orgInput)
			if orgID == "" {
				c.workspaceErr = fmt.Errorf("%w: organization %q not found for this account", ErrValidation, orgInput)
				return
			}
		}
		if c.http != nil {
			c.http.setWorkspace(orgID, "")
		}
		projectID := projectInput
		if !uuidPattern.MatchString(projectInput) {
			result, err := c.execute(ctx, gen.NewProjectsRequest(map[string]any{"orgId": orgID}))
			if err != nil {
				c.workspaceErr = err
				return
			}
			projectID = matchWorkspaceEntry(result, projectInput)
			if projectID == "" {
				c.workspaceErr = fmt.Errorf("%w: project %q not found in this organization", ErrValidation, projectInput)
				return
			}
		}
		if c.http != nil {
			c.http.setWorkspace(orgID, projectID)
		}
		c.orgID, c.project = orgID, projectID
	})
	return c.workspaceErr
}

// matchWorkspaceEntry finds an entry by slug or name and returns its id.
func matchWorkspaceEntry(result any, input string) string {
	entries, _ := result.([]any)
	for _, entry := range entries {
		object, _ := entry.(map[string]any)
		if object["slug"] == input || object["name"] == input {
			id, _ := object["id"].(string)
			return id
		}
	}
	return ""
}

func (c *Client) projectID(ctx context.Context) (string, error) {
	if err := c.resolveWorkspace(ctx); err != nil {
		return "", err
	}
	return c.project, nil
}

// Ontology returns a handle for one ontology entity, addressed by apiName.
func (c *Client) Ontology(name string) *OntologyHandle {
	return &OntologyHandle{client: c, apiName: name}
}

// OntologyEntities lists all ontology entities in the project.
func (c *Client) OntologyEntities(ctx context.Context) ([]map[string]any, error) {
	projectID, err := c.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("OntologyEntities")
	result, err := c.execute(ctx, gen.NewOntologyEntitiesRequest(map[string]any{"projectId": projectID}))
	if err != nil {
		return nil, err
	}
	raw, _ := result.([]any)
	entities := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		entity, _ := entry.(map[string]any)
		entities = append(entities, entity)
	}
	return entities, nil
}
