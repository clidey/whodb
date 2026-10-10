package mcp

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformState holds one principal's confirmations and workflow plans. Embedders
// must persist it encrypted and serialize access across their replicas.
type PlatformState struct {
	// Checkpoint persists an execution claim before a confirmed mutation begins.
	Checkpoint         func() error `json:"-"`
	Pending            map[string]*PendingPlatformAction
	IdempotencyPending map[string]string
	IdempotencyDone    map[string]time.Time
	Workflows          []platformWorkflowPlan
	mu                 sync.RWMutex
}

var defaultPlatformState = NewPlatformState()

// NewPlatformState creates isolated platform execution state.
func NewPlatformState() *PlatformState {
	return &PlatformState{Pending: map[string]*PendingPlatformAction{}, IdempotencyPending: map[string]string{}, IdempotencyDone: map[string]time.Time{}}
}

// PlatformRuntime supplies an embedding's identity, transport and principal-owned
// state. It never reads local login configuration or follows a different host.
type PlatformRuntime struct {
	Host      string
	UserID    string
	Email     string
	Transport http.RoundTripper
	State     *PlatformState
}

type platformRuntimeKey struct{}
type generatedPlatformUploadKey struct{}

// WithPlatformRuntime attaches the authenticated runtime for one request.
func WithPlatformRuntime(ctx context.Context, runtime *PlatformRuntime) context.Context {
	return context.WithValue(ctx, platformRuntimeKey{}, runtime)
}

func platformRuntimeFromContext(ctx context.Context) *PlatformRuntime {
	runtime, _ := ctx.Value(platformRuntimeKey{}).(*PlatformRuntime)
	return runtime
}

func hasPlatformRuntime(contexts ...context.Context) bool {
	return len(contexts) > 0 && platformRuntimeFromContext(contexts[0]) != nil
}

func platformStateFor(contexts ...context.Context) *PlatformState {
	if hasPlatformRuntime(contexts...) {
		return platformRuntimeFromContext(contexts[0]).State
	}
	return defaultPlatformState
}

// NewPlatformServer registers the shared platform tools, prompts and resources
// with the embedding application's version and without local database,
// configuration or browser-capture tools.
func NewPlatformServer(readOnly bool, applicationVersion string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "whodb", Version: applicationVersion}, &mcp.ServerOptions{
		Instructions: mcpUsageInstructions + "\n\n" + platformInstructions + "\nThis server uses your OAuth identity. Supply workspace {org, project} on workspace calls. Local login settings, local paths and browser capture are unavailable. Writes require preview and confirmation.",
	})
	options := &SecurityOptions{ReadOnly: readOnly, ConfirmWrites: true, SecurityLevel: SecurityLevelStandard, MaxRows: 100, QueryTimeout: 30 * time.Second,
		ToolEnablement: &ToolEnablement{DisabledTools: []string{"whodb_platform_app_screenshot"}}}
	server.AddReceivingMiddleware(toolErrorMiddleware)
	registerPlatformSurface(server, options)
	return server
}

func (runtime *PlatformRuntime) loadSession(ctx context.Context) (*platformToolSession, error) {
	if runtime.UserID == "" || runtime.Transport == nil || runtime.State == nil {
		return nil, fmt.Errorf("authenticated platform runtime is required")
	}
	scope, _, err := platformRequestScope(ctx)
	if err != nil {
		return nil, err
	}
	client, err := platformapi.NewClient(runtime.Host, "")
	if err != nil {
		return nil, err
	}
	client.SetHTTPClient(&http.Client{Transport: runtime.Transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if _, err := client.PlatformManifest(ctx); err != nil {
		return nil, err
	}
	session := &platformToolSession{Host: config.PlatformHost{URL: runtime.Host, AccountID: runtime.UserID, Email: runtime.Email}, Client: &embeddedPlatformClient{Client: client, runtime: runtime}}
	if scope.Org != "" {
		org, err := resolvePlatformToolOrg(ctx, session, scope.Org)
		if err != nil {
			return nil, err
		}
		session.Host.DefaultOrgID, session.Host.DefaultOrgName = org.ID, org.Name
		client.SetWorkspaceContext(org.ID, "")
		if scope.Project != "" {
			if _, err := applyPlatformToolSessionScope(ctx, session, scope); err != nil {
				return nil, err
			}
		}
	}
	recordPlatformScope(ctx, session)
	return session, nil
}

type embeddedPlatformClient struct {
	*platformapi.Client
	runtime *PlatformRuntime
}

// Me returns the identity verified by the embedding, without a local login.
func (c *embeddedPlatformClient) Me(context.Context) (*platformapi.User, error) {
	return &platformapi.User{ID: c.runtime.UserID, Email: c.runtime.Email}, nil
}

// UploadProjectFile accepts only temporary files created from inline bundle content.
func (c *embeddedPlatformClient) UploadProjectFile(ctx context.Context, project string, folder *string, path string) (*platformapi.ProjectFile, error) {
	if approved, _ := ctx.Value(generatedPlatformUploadKey{}).(string); approved == "" || approved != path {
		return nil, fmt.Errorf("local file paths are unavailable over remote MCP; upload through the application or import inline bundle content")
	}
	return c.Client.UploadProjectFile(ctx, project, folder, path)
}

func localPlatformConfig(ctx context.Context) (*config.Config, error) {
	if platformRuntimeFromContext(ctx) != nil {
		return nil, nil
	}
	return config.LoadConfigWithoutSecrets()
}

func embeddedPlatformSetupStatus(ctx context.Context, requestID string) PlatformSetupStatusOutput {
	runtime := platformRuntimeFromContext(ctx)
	output := PlatformSetupStatusOutput{Host: runtime.Host, Authenticated: true, AccountID: runtime.UserID, Email: runtime.Email, Status: "needs_workspace", RequestID: requestID}
	session, err := runtime.loadSession(ctx)
	if err != nil {
		output.Error = err.Error()
		return output
	}
	output.OrgID, output.OrgName = session.Host.DefaultOrgID, session.Host.DefaultOrgName
	output.ProjectID, output.ProjectName = session.Host.DefaultProjectID, session.Host.DefaultProjectName
	output.WorkspaceSelected = hasPlatformWorkspace(session)
	if output.WorkspaceSelected {
		output.Status = "ready"
	} else {
		output.NextSteps = []string{"List organizations and projects, then pass workspace {org, project}."}
	}
	return output
}

func registerPlatformSurface(server *mcp.Server, options *SecurityOptions) {
	server.AddReceivingMiddleware(platformPolicyMiddleware(options.PlatformPolicy))
	registerPlatformTools(server, options)
	registerPlatformPrompts(server)
	registerPlatformResources(server, options)
}
