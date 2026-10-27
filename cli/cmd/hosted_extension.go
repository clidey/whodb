package cmd

import (
	"context"
	"net/http"

	"github.com/spf13/cobra"
)

// HostedSession gives an added command the authenticated hosted connection.
type HostedSession struct {
	Host       string
	OrgID      string
	Automation bool
	query      func(context.Context, string, any, any) error
	request    func(*http.Request) (*http.Response, error)
}

// GraphQL executes a fixed document supplied by the caller.
func (s *HostedSession) GraphQL(ctx context.Context, document string, variables, target any) error {
	return s.query(ctx, document, variables, target)
}

// Do sends a request to the authenticated host.
func (s *HostedSession) Do(request *http.Request) (*http.Response, error) {
	return s.request(request)
}

// OpenHostedSession resolves an organization for an added hosted command.
func OpenHostedSession(ctx context.Context, host, organization string) (*HostedSession, error) {
	session, err := loadPlatformSession(ctx, host)
	if err != nil {
		return nil, err
	}
	org, err := resolveOrganization(ctx, session.Client, session.Host, organization)
	if err != nil {
		return nil, err
	}
	session.Client.SetWorkspaceContext(org.ID, "")
	return &HostedSession{Host: session.Host.URL, OrgID: org.ID, Automation: session.Automation,
		query: session.Client.ExecuteGraphQL, request: session.Client.DoAuthenticated}, nil
}

// RegisterHostedCommand adds a command supplied by the embedding edition.
func RegisterHostedCommand(command *cobra.Command) {
	rootCmd.AddCommand(command)
}
