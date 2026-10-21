//go:build e2e_platform

package e2e_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
)

func TestPlatformCLI_ConcurrentRefreshProcesses(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(os.Getenv("HOME"), "tokens"))
	config.ResetPathsForTesting()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	host := envOrDefault("WHODB_PLATFORM_E2E_HOST", defaultPlatformE2EHost)
	refresh := mintDevRefreshToken(t, ctx, envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultPlatformE2EKeycloakURL), envOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_HOST_HEADER", "127.0.0.1:4001"), defaultPlatformE2EUser, defaultPlatformE2EPassword)
	restore := seedPlatformLogin(t, ctx, host, refresh)
	defer restore()
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, stderr, code := testharness.RunCLI(t, "status", "--host", host, "--format", "json", "--quiet")
			if code != 0 {
				t.Errorf("status exit=%d: %s", code, stderr)
			}
		}()
	}
	wg.Wait()
	runEnvelope(t, "status", "--host", host, "--format", "json", "--quiet")
}
