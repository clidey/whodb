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
	"slices"

	"github.com/clidey/whodb/cli/internal/config"
	connresolver "github.com/clidey/whodb/cli/internal/connections"
	"github.com/clidey/whodb/cli/internal/sourcetypes"
	"github.com/spf13/cobra"
)

// completeConnectionNames returns saved and env-based connection names.
func completeConnectionNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	resolver, err := connresolver.NewResolver(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	conns := resolver.List()
	names := make([]string, len(conns))
	for i, c := range conns {
		names[i] = c.Name
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeDatabaseTypes returns known database type identifiers and their synonyms.
func completeDatabaseTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ids := sourcetypes.IDs()
	synonyms := sourcetypes.Synonyms()
	types := make([]string, 0, len(ids)+len(synonyms))
	types = append(types, ids...)
	types = append(types, synonyms...)
	return types, cobra.ShellCompDirectiveNoFileComp
}

// completeOutputFormats returns the standard output format options.
func completeOutputFormats(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"auto", "table", "plain", "json", "ndjson", "csv"}, cobra.ShellCompDirectiveNoFileComp
}

// completeExportFormats returns export-specific format options.
func completeExportFormats(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"csv", "excel"}, cobra.ShellCompDirectiveNoFileComp
}

// completeAuditFormats returns audit-specific format options.
func completeAuditFormats(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp
}

// completeShellNames returns supported shell names for completion scripts.
func completeShellNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"bash", "zsh", "fish", "powershell"}, cobra.ShellCompDirectiveNoFileComp
}

// completeProfileNames returns saved profile names.
func completeProfileNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	profiles := cfg.GetProfiles()
	names := make([]string, len(profiles))
	for i, p := range profiles {
		names[i] = p.Name
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeMCPTransports returns supported MCP transport types.
func completeMCPTransports(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"stdio", "http"}, cobra.ShellCompDirectiveNoFileComp
}

// completeMCPSecurityLevels returns supported MCP security levels.
func completeMCPSecurityLevels(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"strict", "standard", "minimal"}, cobra.ShellCompDirectiveNoFileComp
}

// completeSSLModes returns source-backed SSL mode values for the selected
// database type. If no type is selected yet, it returns the union of all known
// SSL modes.
func completeSSLModes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	dbType, _ := cmd.Flags().GetString("type")
	if spec, ok := lookupDatabaseType(dbType); ok {
		modes := make([]string, 0, len(spec.SSLModes))
		for _, mode := range spec.SSLModes {
			modes = append(modes, string(mode.Value))
		}
		return modes, cobra.ShellCompDirectiveNoFileComp
	}

	modeSet := map[string]struct{}{}
	for _, id := range sourcetypes.IDs() {
		for _, mode := range sourcetypes.SSLModes(id) {
			modeSet[string(mode.Value)] = struct{}{}
		}
	}

	modes := make([]string, 0, len(modeSet))
	for mode := range modeSet {
		modes = append(modes, mode)
	}
	slices.Sort(modes)
	return modes, cobra.ShellCompDirectiveNoFileComp
}
