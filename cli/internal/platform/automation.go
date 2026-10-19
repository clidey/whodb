package platform

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// APIKeyEnv supplies a process-only service-account credential. It is never saved.
const APIKeyEnv = "WHODB_PLATFORM_API_KEY"

// AutomationClient loads an explicitly host-bound automation credential. A nil
// client means no API key was supplied. Invalid automation settings never fall
// back to a saved personal login. Workspace values must be canonical IDs.
func AutomationClient(requestedHost string) (*Client, SessionScope, error) {
	key, present := os.LookupEnv(APIKeyEnv)
	if !present {
		return nil, SessionScope{}, nil
	}
	scope := SessionScopeFromEnvironment()
	if strings.TrimSpace(key) == "" {
		return nil, scope, fmt.Errorf("%s is empty", APIKeyEnv)
	}
	if scope.Host == "" || !scope.HasWorkspace() {
		return nil, scope, fmt.Errorf("API-key authentication requires %s, %s and %s (canonical workspace IDs)", SessionHostEnv, SessionOrgEnv, SessionProjectEnv)
	}
	host, err := NormalizeHost(scope.Host)
	if err != nil {
		return nil, scope, err
	}
	if requestedHost != "" {
		target, err := NormalizeHost(requestedHost)
		if err != nil {
			return nil, scope, err
		}
		if target != host {
			return nil, scope, fmt.Errorf("requested host differs from %s; refusing to forward the API key", SessionHostEnv)
		}
	}
	client, err := NewClient(host, strings.TrimSpace(key))
	if err != nil {
		return nil, scope, err
	}
	client.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	scope.Host = host
	client.SetWorkspaceContext(scope.Org, scope.Project)
	return client, scope, nil
}
