package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	packageInputFile       string
	packageBindings        []string
	packageSourceProjectID string
	packageDeleteObjects   bool
	packageRollbackID      string
)

var packagesGetCmd = &cobra.Command{Use: "get <package>", Short: "Inspect an organization package release", Args: cobra.ExactArgs(1), RunE: runPlatformPackageGet}
var packagesPreviewCreateCmd = &cobra.Command{Use: "preview-create", Short: "Validate and preview a package manifest", RunE: runPlatformPackageCreatePreview}
var packagesCreateCmd = &cobra.Command{Use: "create", Short: "Create an immutable package release from a manifest", RunE: runPlatformPackageCreate}
var packagesPreviewInstallCmd = &cobra.Command{Use: "preview-install <package>", Short: "Validate package requirements and installation conflicts", Args: cobra.ExactArgs(1), RunE: runPlatformPackageInstallPreview}
var packagesInstallCmd = &cobra.Command{Use: "install <package>", Short: "Install a package into the selected project", Args: cobra.ExactArgs(1), RunE: runPlatformPackageInstall}
var packagesImportCmd = &cobra.Command{Use: "import <package>", Short: "Import a package as customer-owned objects", Args: cobra.ExactArgs(1), RunE: runPlatformPackageImport}
var packagesPreviewSharedCmd = &cobra.Command{Use: "preview-shared <share-token>", Short: "Validate a shared package installation", Args: cobra.ExactArgs(1), RunE: runPlatformSharedPackagePreview}
var packagesInstallSharedCmd = &cobra.Command{Use: "install-shared <share-token>", Short: "Install a cross-organization shared package", Args: cobra.ExactArgs(1), RunE: runPlatformSharedPackageInstall}
var packagesImportSharedCmd = &cobra.Command{Use: "import-shared <share-token>", Short: "Import a cross-organization shared package", Args: cobra.ExactArgs(1), RunE: runPlatformSharedPackageImport}
var packagesDeployCmd = &cobra.Command{Use: "deploy", Short: "Install one multi-project package deployment manifest", RunE: runPlatformPackageDeploy}
var packagesDeployReleaseCmd = &cobra.Command{Use: "deploy-release", Short: "Converge a resumable multi-package release manifest", RunE: runPlatformPackageReleaseDeploy}
var packagesShareCmd = &cobra.Command{Use: "share <package>", Short: "Create a governed cross-organization package share", Args: cobra.ExactArgs(1), RunE: runPlatformPackageShare}
var packagesUninstallCmd = &cobra.Command{Use: "uninstall <installation>", Short: "Uninstall a managed package installation", Args: cobra.ExactArgs(1), RunE: runPlatformPackageUninstall}
var packagesDeleteCmd = &cobra.Command{Use: "delete <package>", Short: "Remove a package release from distribution", Args: cobra.ExactArgs(1), RunE: runPlatformPackageDelete}
var packagesRevokeShareCmd = &cobra.Command{Use: "revoke-share <share>", Short: "Revoke a package share", Args: cobra.ExactArgs(1), RunE: runPlatformPackageRevokeShare}

func registerPlatformPackageLifecycleCommands() {
	for _, command := range []*cobra.Command{packagesPreviewCreateCmd, packagesCreateCmd, packagesDeployCmd, packagesDeployReleaseCmd} {
		command.Flags().StringVar(&packageInputFile, "file", "", "JSON manifest file")
		_ = command.MarkFlagRequired("file")
	}
	for _, command := range []*cobra.Command{packagesPreviewInstallCmd, packagesInstallCmd, packagesImportCmd, packagesPreviewSharedCmd, packagesInstallSharedCmd, packagesImportSharedCmd} {
		command.Flags().StringArrayVar(&packageBindings, "binding", nil, "requirement-key=target-id; repeatable")
		command.Flags().StringVar(&packageSourceProjectID, "source-project", "", "source project scope for a multi-project package")
	}
	for _, command := range []*cobra.Command{packagesCreateCmd, packagesInstallCmd, packagesImportCmd, packagesInstallSharedCmd, packagesImportSharedCmd, packagesDeployCmd, packagesDeployReleaseCmd, packagesShareCmd, packagesUninstallCmd, packagesDeleteCmd, packagesRevokeShareCmd} {
		command.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "execute the mutation")
	}
	packagesShareCmd.Flags().StringVar(&packageSourceProjectID, "source-project", "", "source project represented by this share (defaults to selected project)")
	packagesUninstallCmd.Flags().BoolVar(&packageDeleteObjects, "delete-objects", false, "delete package-owned objects; customer data remains protected by the server")
	packagesUpdateCmd.Flags().StringVar(&packageRollbackID, "package-id", "", "specific package release id to apply, including an older rollback release")
	packagesCmd.AddCommand(packagesGetCmd, packagesPreviewCreateCmd, packagesCreateCmd, packagesPreviewInstallCmd, packagesInstallCmd, packagesImportCmd, packagesPreviewSharedCmd, packagesInstallSharedCmd, packagesImportSharedCmd, packagesDeployCmd, packagesDeployReleaseCmd, packagesShareCmd, packagesUninstallCmd, packagesDeleteCmd, packagesRevokeShareCmd)
}

func packageLifecycleSession(ctx context.Context) (*platformSession, string, string, error) {
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return nil, "", "", err
	}
	org, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return nil, "", "", err
	}
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	return session, org.ID, project.ID, nil
}

func readPackageManifest(path string) (map[string]any, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("--file is required")
	}
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read package manifest: %w", err)
	}
	var input map[string]any
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("parse package manifest: %w", err)
	}
	return input, nil
}

func runPlatformPackageGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	session, orgID, _, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	data, err := session.Client.PlatformQuery(ctx, "OrganizationPackage", map[string]any{"organizationId": orgID, "packageId": args[0]})
	if err != nil {
		return err
	}
	return writePackageResult(cmd, "packages.get", data)
}

func runPlatformPackageCreatePreview(cmd *cobra.Command, _ []string) error {
	return runPackageManifestQuery(cmd, "PreviewCreatePackage", "packages.create.preview")
}

func runPackageManifestQuery(cmd *cobra.Command, operation, envelope string) error {
	ctx := context.Background()
	input, err := readPackageManifest(packageInputFile)
	if err != nil {
		return err
	}
	session, _, _, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	data, err := session.Client.PlatformQuery(ctx, operation, map[string]any{"input": input})
	if err != nil {
		return err
	}
	return writePackageResult(cmd, envelope, data)
}

func runPlatformPackageCreate(cmd *cobra.Command, _ []string) error {
	input, err := readPackageManifest(packageInputFile)
	if err != nil {
		return err
	}
	return runPackageMutation(cmd, "CreatePackage", map[string]any{"input": input}, "packages.create")
}

func packageOperationInput(packageID, projectID string) (map[string]any, error) {
	bindings, err := parsePackageBindings(packageBindings)
	if err != nil {
		return nil, err
	}
	input := map[string]any{"targetProjectId": projectID, "packageId": packageID}
	if len(bindings) > 0 {
		input["bindings"] = bindings
	}
	if strings.TrimSpace(packageSourceProjectID) != "" {
		input["sourceScopeProjectId"] = strings.TrimSpace(packageSourceProjectID)
	}
	return input, nil
}

func runPlatformPackageInstallPreview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	session, _, projectID, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	input, err := packageOperationInput(args[0], projectID)
	if err != nil {
		return err
	}
	data, err := session.Client.PlatformQuery(ctx, "PreviewInstallPackage", map[string]any{"input": input})
	if err != nil {
		return err
	}
	return writePackageResult(cmd, "packages.install.preview", data)
}

func runPlatformPackageInstall(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_, _, projectID, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	input, err := packageOperationInput(args[0], projectID)
	if err != nil {
		return err
	}
	return runPackageMutation(cmd, "InstallPackage", map[string]any{"input": input}, "packages.install")
}

func runPlatformPackageImport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_, _, projectID, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	input, err := packageOperationInput(args[0], projectID)
	if err != nil {
		return err
	}
	delete(input, "sourceScopeProjectId")
	return runPackageMutation(cmd, "ImportPackage", map[string]any{"input": input}, "packages.import")
}

func sharedPackageOperationInput(shareToken, projectID string) (map[string]any, error) {
	bindings, err := parsePackageBindings(packageBindings)
	if err != nil {
		return nil, err
	}
	input := map[string]any{"shareToken": shareToken, "targetProjectId": projectID}
	if len(bindings) > 0 {
		input["bindings"] = bindings
	}
	if strings.TrimSpace(packageSourceProjectID) != "" {
		input["sourceScopeProjectId"] = strings.TrimSpace(packageSourceProjectID)
	}
	return input, nil
}

func runPlatformSharedPackagePreview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	session, _, projectID, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	input, err := sharedPackageOperationInput(args[0], projectID)
	if err != nil {
		return err
	}
	data, err := session.Client.PlatformQuery(ctx, "PreviewSharedPackageInstall", map[string]any{"input": input})
	if err != nil {
		return err
	}
	return writePackageResult(cmd, "packages.shared.preview", data)
}

func runPlatformSharedPackageInstall(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_, _, projectID, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	input, err := sharedPackageOperationInput(args[0], projectID)
	if err != nil {
		return err
	}
	return runPackageMutation(cmd, "InstallSharedPackage", map[string]any{"input": input}, "packages.shared.install")
}

func runPlatformSharedPackageImport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_, _, projectID, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	input, err := sharedPackageOperationInput(args[0], projectID)
	if err != nil {
		return err
	}
	return runPackageMutation(cmd, "ImportSharedPackage", map[string]any{"input": input}, "packages.shared.import")
}

func runPlatformPackageDeploy(cmd *cobra.Command, _ []string) error {
	input, err := readPackageManifest(packageInputFile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, orgID, _, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	if manifestString(input, "organizationId") == "" {
		input["organizationId"] = orgID
	} else if manifestString(input, "organizationId") != orgID {
		return errors.New("manifest organizationId does not match the selected --org; select the target organization explicitly")
	}
	return runPackageMutation(cmd, "InstallPackageDeployment", map[string]any{"input": input}, "packages.deploy")
}

type packageReleaseDeploymentResult struct {
	Index      int    `json:"index"`
	PackageID  string `json:"packageId,omitempty"`
	Successes  int    `json:"successCount"`
	Failures   int    `json:"failureCount"`
	Deployment any    `json:"deployment"`
}

type packageReleaseResult struct {
	OrganizationID string                           `json:"organizationId"`
	Releases       []packageReleaseDeploymentResult `json:"releases"`
	SuccessCount   int                              `json:"successCount"`
	FailureCount   int                              `json:"failureCount"`
}

func normalizePackageReleaseManifest(input map[string]any, organizationID string) ([]map[string]any, error) {
	rawPackages, ok := input["packages"].([]any)
	if !ok || len(rawPackages) == 0 {
		return nil, errors.New("release manifest requires a non-empty packages array")
	}
	deployments := make([]map[string]any, 0, len(rawPackages))
	for index, raw := range rawPackages {
		deployment, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("packages[%d] must be an object", index)
		}
		packageID := manifestString(deployment, "packageId")
		shareToken := manifestString(deployment, "shareToken")
		if (packageID == "") == (shareToken == "") {
			return nil, fmt.Errorf("packages[%d] must provide exactly one of packageId or shareToken", index)
		}
		targets, ok := deployment["targets"].([]any)
		if !ok || len(targets) == 0 {
			return nil, fmt.Errorf("packages[%d] requires a non-empty targets array", index)
		}
		deployment["organizationId"] = organizationID
		deployments = append(deployments, deployment)
	}
	return deployments, nil
}

func runPlatformPackageReleaseDeploy(cmd *cobra.Command, _ []string) error {
	if !platformWriteYes {
		return errors.New("refusing package release deployment without --yes")
	}
	manifest, err := readPackageManifest(packageInputFile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	session, orgID, _, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	if configured := manifestString(manifest, "organizationId"); configured != "" && configured != orgID {
		return errors.New("manifest organizationId does not match the selected --org; select the target organization explicitly")
	}
	deployments, err := normalizePackageReleaseManifest(manifest, orgID)
	if err != nil {
		return err
	}
	result := packageReleaseResult{OrganizationID: orgID, Releases: make([]packageReleaseDeploymentResult, 0, len(deployments))}
	for index, deployment := range deployments {
		mutation, mutationErr := session.Client.PlatformMutation(ctx, "InstallPackageDeployment", map[string]any{"input": deployment})
		if mutationErr != nil {
			return fmt.Errorf("deploy packages[%d]: %w", index, mutationErr)
		}
		var decoded map[string]any
		if err := json.Unmarshal(mutation.Data, &decoded); err != nil {
			return fmt.Errorf("decode packages[%d] deployment: %w", index, err)
		}
		successes := jsonNumberAsInt(decoded["successCount"])
		failures := jsonNumberAsInt(decoded["failureCount"])
		packageID := manifestString(deployment, "packageId")
		result.Releases = append(result.Releases, packageReleaseDeploymentResult{Index: index, PackageID: packageID, Successes: successes, Failures: failures, Deployment: decoded})
		result.SuccessCount += successes
		result.FailureCount += failures
	}
	if err := writePackageResult(cmd, "packages.release.deploy", result); err != nil {
		return err
	}
	if result.FailureCount > 0 {
		return fmt.Errorf("release deployment completed with %d failed target(s); rerun the same manifest after correcting the reported errors", result.FailureCount)
	}
	return nil
}

func manifestString(input map[string]any, key string) string {
	value, ok := input[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func jsonNumberAsInt(value any) int {
	switch number := value.(type) {
	case float64:
		return int(number)
	case int:
		return number
	case json.Number:
		parsed, _ := number.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func runPlatformPackageShare(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_, _, projectID, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(packageSourceProjectID) != "" {
		projectID = strings.TrimSpace(packageSourceProjectID)
	}
	return runPackageMutation(cmd, "CreatePackageShare", map[string]any{"input": map[string]any{"packageId": args[0], "sourceProjectId": projectID}}, "packages.share")
}

func runPlatformPackageUninstall(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_, _, projectID, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	input := map[string]any{"targetProjectId": projectID, "installationId": args[0], "deleteObjects": packageDeleteObjects}
	return runPackageMutation(cmd, "UninstallPackage", map[string]any{"input": input}, "packages.uninstall")
}

func runPlatformPackageDelete(cmd *cobra.Command, args []string) error {
	return runPackageMutation(cmd, "DeletePackage", map[string]any{"packageId": args[0]}, "packages.delete")
}

func runPlatformPackageRevokeShare(cmd *cobra.Command, args []string) error {
	return runPackageMutation(cmd, "RevokePackageShare", map[string]any{"shareId": args[0]}, "packages.share.revoke")
}

func runPackageMutation(cmd *cobra.Command, operation string, variables map[string]any, envelope string) error {
	if !platformWriteYes {
		return errors.New("refusing package mutation without --yes; run the matching preview first")
	}
	ctx := context.Background()
	session, _, _, err := packageLifecycleSession(ctx)
	if err != nil {
		return err
	}
	result, err := session.Client.PlatformMutation(ctx, operation, variables)
	if err != nil {
		return err
	}
	return writePackageResult(cmd, envelope, result.Data)
}

func writePackageResult(cmd *cobra.Command, envelope string, response any) error {
	value := response
	if raw, ok := response.(json.RawMessage); ok {
		value = nil
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &value); err != nil {
				return fmt.Errorf("decode package response: %w", err)
			}
		}
	}
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, envelope, value)
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
	return err
}
