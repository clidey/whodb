package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

type readinessClient struct {
	*fakePlatformClient
	sourceReads int
}

func (c *readinessClient) AIProviders(context.Context, string) ([]platformapi.AIProvider, error) {
	return nil, errors.New("permission denied")
}
func (c *readinessClient) ProjectSources(ctx context.Context, org, project string) ([]platformapi.Source, error) {
	c.sourceReads++
	return c.fakePlatformClient.ProjectSources(ctx, org, project)
}
func (c *readinessClient) FunctionRuns(context.Context, string, string, int) ([]platformapi.FunctionRun, error) {
	at := "2026-10-19T10:00:00Z"
	return []platformapi.FunctionRun{{ID: "run", Status: "completed", CompletedAt: &at}}, nil
}

func TestRuntimeReadinessPartialInventoryAndHistoricalEvidence(t *testing.T) {
	client := &readinessClient{fakePlatformClient: &fakePlatformClient{}}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) { return testPlatformSession(client), nil })
	_, out, err := HandlePlatformRuntimeReadiness(context.Background(), nil, PlatformNextActionsInput{FunctionID: "function"})
	if err != nil || out.Error != "" {
		t.Fatalf("readiness failed: %v %s", err, out.Error)
	}
	readiness := out.Data.(PlatformRuntimeReadiness)
	if client.sourceReads != 0 {
		t.Fatal("runtime readiness fetched unrelated source inventory")
	}
	for _, warning := range readiness.Warnings {
		if strings.Contains(warning, "no hosted sources") || strings.Contains(warning, "no datasets") {
			t.Fatalf("unfetched inventory reported as absent: %s", warning)
		}
	}
	if readiness.ObservedAt == "" || len(readiness.Evidence) != 5 {
		t.Fatal(readiness)
	}
	for _, evidence := range readiness.Evidence {
		if evidence.Reachable != "unknown" {
			t.Fatal("metadata/history claimed current reachability")
		}
		if evidence.Resource == "ai_providers" && evidence.Configured != "unknown" {
			t.Fatal("failed inventory claimed absence")
		}
	}
	if readiness.Evidence[4].Verified != "verified" || readiness.Evidence[4].ObservedAt != "2026-10-19T10:00:00Z" {
		t.Fatal("lost execution timestamp")
	}
	snapshot, err := loadPlatformWorkspaceSnapshot(context.Background(), testPlatformSession(client), true, true)
	if err != nil || len(snapshot.functions) == 0 {
		t.Fatal("one unavailable collection discarded others")
	}
	if _, ok := platformWorkspaceCounts(snapshot)["ai_providers"]; ok {
		t.Fatal("unknown inventory reported as zero")
	}
}
