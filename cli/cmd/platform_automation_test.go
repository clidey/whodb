package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/platform"
)

func TestAutomationWorkspaceNeedsNoPersonalOrganizationDiscovery(t *testing.T) {
	// A nil client proves resolution performs no discovery or credential reads.
	session := &platformSession{Automation: true, Host: config.PlatformHost{DefaultOrgID: "org", DefaultProjectID: "project"}}
	org, project, err := resolvePlatformProject(context.Background(), session, "", "")
	if err != nil || org.ID != "org" || project.ID != "project" {
		t.Fatal(org, project, err)
	}
	if _, _, err := resolvePlatformProject(context.Background(), session, "org", "other"); err == nil {
		t.Fatal("workspace override escaped process binding")
	}
}

func TestAutomationManifestPreservesPersonalLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"PlatformManifest":{"platformVersion":"dev"}}}`))
	}))
	defer server.Close()
	cfg := config.DefaultConfig()
	cfg.UpsertPlatformHost(config.PlatformHost{URL: server.URL, AccountID: "personal-account"})
	client, err := platform.NewClient(server.URL, "service-key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := refreshPlatformManifest(context.Background(), cfg, &config.PlatformHost{URL: server.URL}, client); err != nil {
		t.Fatal(err)
	}
	entry, ok := cfg.GetPlatformHost(server.URL)
	if !ok || entry.AccountID != "personal-account" {
		t.Fatal("automation replaced saved identity")
	}
}

func TestCapabilityCommandsAreRegistered(t *testing.T) {
	found := map[string]bool{}
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if operation := command.Annotations["platform_operation"]; operation != "" {
			if command.RunE == nil {
				t.Errorf("%s has no handler", command.CommandPath())
			}
			found[operation] = true
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(rootCmd)
	for _, capability := range platform.CapabilityReads {
		if !found[capability.Operation] {
			t.Errorf("%s has no registered CLI command", capability.Operation)
		}
	}
}
