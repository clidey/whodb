/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	cloudruntime "github.com/clidey/whodb/cli/internal/cloud"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var cloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Inspect cloud providers and discovered resources",
	Long: `Inspect configured cloud providers and the resources they discover.

ALPHA WARNING:
  The cloud provider and discovered-resource workflow is still in testing.
  It is not ready for production use yet, and provider/resource coverage may
  still have rough edges. Mileage may vary.

This command works with the shared WhoDB cloud provider runtime. When provider
support is enabled, it can inspect configured providers and discover
cloud-managed databases and caches from the CLI.`,
	Example: `  # List configured providers
  whodb cloud providers list

  # Discover resources from all providers
  whodb cloud connections list

  # Discover resources from one provider
  whodb cloud connections list --provider aws-prod-us-west-2

  # Refresh provider discovery state
  whodb cloud providers refresh --all`,
}

func init() {
	rootCmd.AddCommand(cloudCmd)
	cloudCmd.AddCommand(
		newCloudProvidersCommand(),
		newCloudConnectionsCommand(),
	)
}

func newCloudProvidersCommand() *cobra.Command {
	cloudProvidersCmd := &cobra.Command{
		Use:   "providers",
		Short: "Inspect configured cloud providers",
		Long: `Inspect configured cloud providers from the shared WhoDB data directory.

Provider configuration can come from the app or from environment variables such
as WHODB_AWS_PROVIDER, WHODB_AZURE_PROVIDER, and WHODB_GCP_PROVIDER when cloud
provider support is enabled.`,
	}

	cloudProvidersCmd.AddCommand(
		newCloudProvidersListCommand(),
		newCloudProvidersTestCommand(),
		newCloudProvidersRefreshCommand(),
	)

	return cloudProvidersCmd
}

func newCloudProvidersListCommand() *cobra.Command {
	var formatFlag string
	var quiet bool

	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List configured cloud providers",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.ParseFormat(formatFlag)
			if err != nil {
				return err
			}

			writeCloudAlphaWarning(cmd, format, quiet)

			providers, err := cloudruntime.ListProviders()
			if err != nil {
				return err
			}

			return renderProviderList(cmd, providers, format, quiet)
		},
	}

	cmd.Flags().StringVar(&formatFlag, "format", "auto", "output format: auto, table, plain, json, ndjson, csv")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress informational output")

	return cmd
}

func newCloudProvidersTestCommand() *cobra.Command {
	var formatFlag string

	cmd := &cobra.Command{
		Use:           "test <provider-id>",
		Short:         "Test a configured cloud provider",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: `  whodb cloud providers test aws-prod-us-west-2
  whodb cloud providers test gcp-prod-us-central1 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.ParseFormat(formatFlag)
			if err != nil {
				return err
			}

			writeCloudAlphaWarning(cmd, format, false)

			provider, err := cloudruntime.TestProvider(args[0])
			if err != nil {
				return err
			}

			return renderProviderAction(cmd, []cloudruntime.ProviderSummary{provider}, format, "cloud.providers.test")
		},
	}

	cmd.Flags().StringVar(&formatFlag, "format", "auto", "output format: auto, table, plain, json, ndjson, csv")

	return cmd
}

func newCloudProvidersRefreshCommand() *cobra.Command {
	var formatFlag string
	var refreshAll bool

	cmd := &cobra.Command{
		Use:           "refresh [provider-id]",
		Short:         "Refresh provider discovery state",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: `  whodb cloud providers refresh aws-prod-us-west-2
  whodb cloud providers refresh --all
  whodb cloud providers refresh --all --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.ParseFormat(formatFlag)
			if err != nil {
				return err
			}

			writeCloudAlphaWarning(cmd, format, false)

			if refreshAll && len(args) > 0 {
				return fmt.Errorf("provider ID cannot be used with --all")
			}
			if !refreshAll && len(args) != 1 {
				return fmt.Errorf("provide a provider ID or pass --all")
			}

			var providers []cloudruntime.ProviderSummary
			if refreshAll {
				providers, err = cloudruntime.RefreshAllProviders()
			} else {
				var provider cloudruntime.ProviderSummary
				provider, err = cloudruntime.RefreshProvider(args[0])
				providers = []cloudruntime.ProviderSummary{provider}
			}
			if err != nil {
				return err
			}

			return renderProviderAction(cmd, providers, format, "cloud.providers.refresh")
		},
	}

	cmd.Flags().BoolVar(&refreshAll, "all", false, "refresh all configured providers")
	cmd.Flags().StringVar(&formatFlag, "format", "auto", "output format: auto, table, plain, json, ndjson, csv")

	return cmd
}

func newCloudConnectionsCommand() *cobra.Command {
	cloudConnectionsCmd := &cobra.Command{
		Use:   "connections",
		Short: "Inspect discovered cloud resources",
		Long: `Inspect discovered cloud-managed databases and caches from the CLI.

This command runs discovery directly through the shared provider registry, so
it returns whatever the current WhoDB build knows how to discover.`,
	}

	cloudConnectionsCmd.AddCommand(newCloudConnectionsListCommand())
	return cloudConnectionsCmd
}

func newCloudConnectionsListCommand() *cobra.Command {
	var formatFlag string
	var quiet bool
	var providerID string

	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List discovered cloud resources",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: `  whodb cloud connections list
  whodb cloud connections list --provider aws-prod-us-west-2
  whodb cloud connections list --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.ParseFormat(formatFlag)
			if err != nil {
				return err
			}

			writeCloudAlphaWarning(cmd, format, quiet)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			connections, err := cloudruntime.ListConnections(ctx, providerID)
			if err != nil && len(connections) == 0 {
				return err
			}
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: partial discovery failure: %v\n", err)
			}

			return renderConnectionList(cmd, connections, format, quiet, providerID)
		},
	}

	cmd.Flags().StringVar(&providerID, "provider", "", "limit discovery to a single provider ID")
	cmd.Flags().StringVar(&formatFlag, "format", "auto", "output format: auto, table, plain, json, ndjson, csv")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress informational output")

	return cmd
}

func writeCloudAlphaWarning(cmd *cobra.Command, format output.Format, quiet bool) {
	if quiet || shouldSuppressInformationalOutput(cmd, format) {
		return
	}

	writeAlphaWarningBlock(
		cmd.ErrOrStderr(),
		"CLOUD IS ALPHA / IN TESTING",
		"The cloud provider and discovered-resource workflow is not ready for production use yet.",
		"Expect rough edges, incomplete provider/resource coverage, and behavior changes.",
		"Mileage may vary.",
	)
}

func writeAlphaWarningBlock(w io.Writer, title string, lines ...string) {
	if w == nil {
		return
	}

	fmt.Fprintf(w, "\n========== %s ==========\n", title)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	fmt.Fprintln(w, "========================================")
	fmt.Fprintln(w)
}

func renderProviderList(cmd *cobra.Command, providers []cloudruntime.ProviderSummary, format output.Format, quiet bool) error {
	out := newCommandOutput(cmd, format, quiet || shouldSuppressInformationalOutput(cmd, format))

	if len(providers) == 0 {
		switch effectiveCommandOutputFormat(cmd, format) {
		case output.FormatJSON:
			return writeCommandJSON(cmd, []cloudruntime.ProviderSummary{})
		case output.FormatNDJSON, output.FormatCSV:
			return nil
		default:
			out.Info("No cloud providers configured.")
			out.Info("Create providers in WhoDB or define WHODB_AWS_PROVIDER, WHODB_AZURE_PROVIDER, or WHODB_GCP_PROVIDER.")
			return nil
		}
	}

	switch effectiveCommandOutputFormat(cmd, format) {
	case output.FormatJSON:
		return writeCommandJSON(cmd, providers)
	case output.FormatNDJSON:
		return writeCommandNDJSON(cmd, providers)
	default:
		return out.WriteQueryResult(providerTableResult(providers))
	}
}

func renderProviderAction(cmd *cobra.Command, providers []cloudruntime.ProviderSummary, format output.Format, command string) error {
	out := newCommandOutput(cmd, format, true)

	if len(providers) == 0 {
		switch effectiveCommandOutputFormat(cmd, format) {
		case output.FormatJSON:
			return writeAutomationEnvelope(cmd, command, []cloudruntime.ProviderSummary{})
		case output.FormatNDJSON, output.FormatCSV:
			return nil
		default:
			out.Info("No cloud providers configured.")
			return nil
		}
	}

	switch effectiveCommandOutputFormat(cmd, format) {
	case output.FormatJSON:
		if len(providers) == 1 {
			return writeAutomationEnvelope(cmd, command, providers[0])
		}
		return writeAutomationEnvelope(cmd, command, providers)
	case output.FormatNDJSON:
		return writeCommandNDJSON(cmd, providers)
	default:
		return out.WriteQueryResult(providerTableResult(providers))
	}
}

func renderConnectionList(cmd *cobra.Command, connections []cloudruntime.ConnectionSummary, format output.Format, quiet bool, providerID string) error {
	out := newCommandOutput(cmd, format, quiet || shouldSuppressInformationalOutput(cmd, format))

	if len(connections) == 0 {
		switch effectiveCommandOutputFormat(cmd, format) {
		case output.FormatJSON:
			return writeCommandJSON(cmd, []cloudruntime.ConnectionSummary{})
		case output.FormatNDJSON, output.FormatCSV:
			return nil
		default:
			if providerID != "" {
				out.Info("No discovered connections found for provider %s.", providerID)
				return nil
			}

			providers, err := cloudruntime.ListProviders()
			if errors.Is(err, cloudruntime.ErrCloudProvidersDisabled) {
				return err
			}
			if err == nil && len(providers) == 0 {
				out.Info("No cloud providers configured.")
				out.Info("Create providers in WhoDB or define WHODB_AWS_PROVIDER, WHODB_AZURE_PROVIDER, or WHODB_GCP_PROVIDER.")
				return nil
			}

			out.Info("No discovered connections found.")
			return nil
		}
	}

	switch effectiveCommandOutputFormat(cmd, format) {
	case output.FormatJSON:
		return writeCommandJSON(cmd, connections)
	case output.FormatNDJSON:
		return writeCommandNDJSON(cmd, connections)
	default:
		return out.WriteQueryResult(connectionTableResult(connections))
	}
}

func providerTableResult(providers []cloudruntime.ProviderSummary) *output.QueryResult {
	rows := make([][]any, 0, len(providers))
	for _, provider := range providers {
		rows = append(rows, []any{
			provider.ID,
			provider.ProviderType,
			provider.Name,
			provider.Scope,
			provider.Region,
			provider.Status,
			provider.DiscoveredCount,
			provider.LastDiscoveryAt,
		})
	}

	return &output.QueryResult{
		Columns: []output.Column{
			{Name: "id", Type: "string"},
			{Name: "type", Type: "string"},
			{Name: "name", Type: "string"},
			{Name: "scope", Type: "string"},
			{Name: "region", Type: "string"},
			{Name: "status", Type: "string"},
			{Name: "discovered_count", Type: "int"},
			{Name: "last_discovery_at", Type: "string"},
		},
		Rows: rows,
	}
}

func connectionTableResult(connections []cloudruntime.ConnectionSummary) *output.QueryResult {
	rows := make([][]any, 0, len(connections))
	for _, connection := range connections {
		rows = append(rows, []any{
			connection.ID,
			connection.ProviderID,
			connection.ProviderType,
			connection.Name,
			connection.SourceType,
			connection.Region,
			connection.Status,
			metadataSummary(connection.Metadata),
		})
	}

	return &output.QueryResult{
		Columns: []output.Column{
			{Name: "id", Type: "string"},
			{Name: "provider_id", Type: "string"},
			{Name: "provider_type", Type: "string"},
			{Name: "name", Type: "string"},
			{Name: "source_type", Type: "string"},
			{Name: "region", Type: "string"},
			{Name: "status", Type: "string"},
			{Name: "metadata", Type: "string"},
		},
		Rows: rows,
	}
}

func metadataSummary(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}

	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, metadata[key]))
	}
	return strings.Join(parts, ", ")
}
