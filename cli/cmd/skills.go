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
	"fmt"

	"github.com/clidey/whodb/cli/internal/skillinstaller"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	skillsFormat        string
	skillsQuiet         bool
	skillsTarget        string
	skillsTargetDir     string
	skillsAgentsDir     string
	skillsIncludeAgents bool
	skillsForce         bool
	skillsDryRun        bool
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "List and install bundled assistant skills",
	Long:  `List and install the bundled WhoDB assistant skills and optional agents.`,
}

var skillsListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List bundled skills and agents",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := resolveSkillsFormat(skillsFormat)
		if err != nil {
			return err
		}
		items, err := skillinstaller.List()
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, items)
		}
		if !skillsQuiet {
			writeSkillItems(cmd, items)
		}
		return nil
	},
}

var skillsInstallCmd = &cobra.Command{
	Use:           "install [name]",
	Short:         "Install bundled skills or assistant integrations",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.MaximumNArgs(1),
	Example: `  # Install all skills to a specific directory
  whodb skills install --target-dir ~/.codex/skills

  # Install one skill
  whodb skills install query-builder --target-dir ~/.codex/skills

  # Install common skill and agent directories for a target
  whodb skills install --target claude-code --include-agents

  # Install MCP configuration for an assistant
  whodb skills install --target cursor
  whodb skills install --target gemini-cli

  # Preview files without writing
  whodb skills install --target cursor --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := resolveSkillsFormat(skillsFormat)
		if err != nil {
			return err
		}
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		result, err := skillinstaller.Install(skillinstaller.InstallOptions{
			Name:          name,
			Target:        skillsTarget,
			TargetDir:     skillsTargetDir,
			AgentsDir:     skillsAgentsDir,
			IncludeAgents: skillsIncludeAgents,
			Force:         skillsForce,
			DryRun:        skillsDryRun,
		})
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "skills.install", result)
		}
		if !skillsQuiet {
			writeSkillInstallResult(cmd, result)
		}
		return nil
	},
}

func resolveSkillsFormat(value string) (output.Format, error) {
	switch value {
	case "", "table", "auto":
		return output.FormatTable, nil
	case "json":
		return output.FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid --format %q (expected table or json)", value)
	}
}

func writeSkillItems(cmd *cobra.Command, items []skillinstaller.Item) {
	out := newCommandOutput(cmd, output.FormatTable, false)
	for _, item := range items {
		if item.Description != "" {
			out.Info("%s (%s): %s", item.Name, item.Type, item.Description)
		} else {
			out.Info("%s (%s)", item.Name, item.Type)
		}
	}
}

func writeSkillInstallResult(cmd *cobra.Command, result skillinstaller.InstallResult) {
	out := newCommandOutput(cmd, output.FormatTable, false)
	for _, item := range result.Skills {
		writeInstalledItem(out, result.DryRun, "skill", item)
	}
	for _, item := range result.Agents {
		writeInstalledItem(out, result.DryRun, "agent", item)
	}
	for _, item := range result.Configs {
		writeInstalledItem(out, result.DryRun, "config", item)
	}
	for _, item := range result.Rules {
		writeInstalledItem(out, result.DryRun, "rule", item)
	}
	for _, item := range result.Extensions {
		writeInstalledItem(out, result.DryRun, "extension", item)
	}
}

func writeInstalledItem(out *output.Writer, dryRun bool, label string, item skillinstaller.InstalledFile) {
	if !dryRun {
		out.Success("Installed %s %s to %s", label, item.Name, item.Path)
		return
	}

	action := item.Action
	if action == "" {
		action = "write"
	}
	out.Info("Would %s %s %s at %s", action, label, item.Name, item.Path)
	if item.BackupPath != "" {
		out.Info("Would back up %s %s to %s", label, item.Name, item.BackupPath)
	}
}

func completeSkillTargets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"codex",
		"claude-code",
		"cursor",
		"vscode",
		"github-copilot",
		"gemini-cli",
		"windsurf",
		"opencode",
		"cline",
		"zed",
		"continue",
		"aider",
	}, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(skillsCmd)
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsInstallCmd)

	skillsCmd.PersistentFlags().StringVarP(&skillsFormat, "format", "f", "table", "output format: table or json")
	skillsCmd.PersistentFlags().BoolVarP(&skillsQuiet, "quiet", "q", false, "suppress informational messages")

	skillsInstallCmd.Flags().StringVar(&skillsTarget, "target", "", "assistant target: codex, claude-code, cursor, vscode, github-copilot, gemini-cli, windsurf, opencode, cline, zed, continue, or aider")
	skillsInstallCmd.Flags().StringVar(&skillsTargetDir, "target-dir", "", "directory where skills should be installed")
	skillsInstallCmd.Flags().StringVar(&skillsAgentsDir, "agents-dir", "", "directory where agents should be installed")
	skillsInstallCmd.Flags().BoolVar(&skillsIncludeAgents, "include-agents", false, "install bundled agents as well as skills")
	skillsInstallCmd.Flags().BoolVar(&skillsForce, "force", false, "overwrite existing installed files")
	skillsInstallCmd.Flags().BoolVar(&skillsDryRun, "dry-run", false, "show files that would be written without modifying disk")

	skillsCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
	skillsInstallCmd.RegisterFlagCompletionFunc("target", completeSkillTargets)
}
