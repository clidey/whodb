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

	"github.com/clidey/whodb/cli/internal/agentmanifest"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var agentSchemaFormat string

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent-facing metadata and helpers",
	Long:  `Agent-facing metadata and helpers for AI assistants and automation.`,
}

var agentSchemaCmd = &cobra.Command{
	Use:           "schema",
	Short:         "Emit the agent capability manifest",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example:       `  whodb agent schema --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(agentSchemaFormat)
		if err != nil {
			return err
		}
		if format != output.FormatJSON {
			return fmt.Errorf("agent schema supports --format json")
		}
		return writeCommandJSON(cmd, agentmanifest.Build())
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentSchemaCmd)

	agentSchemaCmd.Flags().StringVarP(&agentSchemaFormat, "format", "f", "json", "output format: json")
	agentSchemaCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}
