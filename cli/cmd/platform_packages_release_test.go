package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNormalizePackageReleaseManifest(t *testing.T) {
	manifest := map[string]any{"packages": []any{
		map[string]any{"packageId": "pkg-1", "targets": []any{map[string]any{"projectId": "project-1"}}},
		map[string]any{"shareToken": "secret", "targets": []any{map[string]any{"projectId": "project-2"}}},
	}}
	deployments, err := normalizePackageReleaseManifest(manifest, "customer-org")
	if err != nil {
		t.Fatalf("normalizePackageReleaseManifest returned error: %v", err)
	}
	if len(deployments) != 2 {
		t.Fatalf("deployment count = %d, want 2", len(deployments))
	}
	for index, deployment := range deployments {
		if deployment["organizationId"] != "customer-org" {
			t.Fatalf("deployment %d organizationId = %#v", index, deployment["organizationId"])
		}
	}
}

func TestNormalizePackageReleaseManifestRejectsAmbiguousRelease(t *testing.T) {
	manifest := map[string]any{"packages": []any{map[string]any{
		"packageId": "pkg-1", "shareToken": "secret", "targets": []any{map[string]any{"projectId": "project-1"}},
	}}}
	if _, err := normalizePackageReleaseManifest(manifest, "customer-org"); err == nil {
		t.Fatal("expected packageId plus shareToken to be rejected")
	}
}

func TestNormalizePackageReleaseManifestAcceptsSinglePackageID(t *testing.T) {
	manifest := map[string]any{"packages": []any{map[string]any{
		"packageId": "pkg-1", "targets": []any{map[string]any{"projectId": "project-1"}},
	}}}
	if _, err := normalizePackageReleaseManifest(manifest, "customer-org"); err != nil {
		t.Fatalf("packageId-only release returned error: %v", err)
	}
}

func TestPackageManifestCommandsDoNotConflictWithFormatShorthand(t *testing.T) {
	for _, command := range []*cobra.Command{packagesPreviewCreateCmd, packagesCreateCmd, packagesDeployCmd, packagesDeployReleaseCmd} {
		fileFlag := command.Flags().Lookup("file")
		if fileFlag == nil {
			t.Fatalf("packages %s has no --file flag", command.Name())
		}
		if fileFlag.Shorthand != "" {
			t.Fatalf("packages %s --file shorthand = %q; parent packages command reserves -f for --format", command.Name(), fileFlag.Shorthand)
		}
	}
}
