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
	"fmt"
	"sort"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

const redactedValue = "********"

type sourceConfigOutput struct {
	Hostname string            `json:"hostname"`
	Port     string            `json:"port"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Database string            `json:"database"`
	Advanced map[string]string `json:"advanced"`
}

type sourceTestOutput struct {
	Status     string           `json:"status"`
	Source     *platform.Source `json:"source,omitempty"`
	SourceType string           `json:"sourceType,omitempty"`
}

var sourcesConfigCmd = &cobra.Command{
	Use:           "config <source>",
	Short:         "Show hosted WhoDB source connection config",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		_, project, source, sourceType, config, err := loadSourceConfigContext(ctx, session, args[0])
		if err != nil {
			return err
		}
		safe := redactSourceConfig(config, sourceType)
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, safe)
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{{Name: "field", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:    sourceConfigRows(source, project, safe),
		})
	},
}

var sourcesUpdateCmd = &cobra.Command{
	Use:           "update <source>",
	Short:         "Update a hosted WhoDB project source",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		quiet := platformQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)
		if !cmd.Flags().Changed("name") && !sourceConfigFlagsChanged(cmd) {
			return fmt.Errorf("nothing to update; pass --name or a connection config flag")
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		org, project, source, err := resolvePlatformSource(ctx, session, sourcesOrg, sourcesProject, args[0])
		if err != nil {
			return err
		}

		input := platform.UpdateSourceInput{OrgID: org.ID, ProjectID: project.ID, ID: source.ID}
		if cmd.Flags().Changed("name") {
			name := strings.TrimSpace(sourceName)
			if name == "" {
				return fmt.Errorf("--name cannot be empty")
			}
			input.Name = &name
		}
		if sourceConfigFlagsChanged(cmd) {
			types, err := session.Client.SourceTypes(ctx)
			if err != nil {
				return err
			}
			sourceType := findSourceType(types, source.DatabaseType)
			existing, err := session.Client.SourceConfig(ctx, org.ID, project.ID, source.ID)
			if err != nil {
				return err
			}
			values, advanced, err := explicitSourceConfigValues(cmd, sourceType)
			if err != nil {
				return err
			}
			config := mergeSourceConfig(existing, values, advanced)
			input.Config = &config
		}

		updated, err := session.Client.UpdateSource(ctx, input)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "sources.update", updated)
		}
		out.Success("Updated source %s in project %s", updated.Name, project.Name)
		return nil
	},
}

var sourcesTestCmd = &cobra.Command{
	Use:           "test [source]",
	Short:         "Test a hosted WhoDB source connection",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		quiet := platformQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}

		if len(args) == 1 {
			if strings.TrimSpace(sourceType) != "" || sourceConfigFlagsChanged(cmd) || cmd.Flags().Changed("name") {
				return fmt.Errorf("saved source tests do not accept draft config flags; omit <source> to test a draft config")
			}
			org, project, source, err := resolvePlatformSource(ctx, session, sourcesOrg, sourcesProject, args[0])
			if err != nil {
				return err
			}
			if _, err := session.Client.SourceObjects(ctx, org.ID, project.ID, source.ID, nil, nil, 1, 0); err != nil {
				return fmt.Errorf("saved source connection failed: %w", err)
			}
			data := sourceTestOutput{Status: "ok", Source: source}
			if format == output.FormatJSON {
				return writeAutomationEnvelope(cmd, "sources.test", data)
			}
			out.Success("Connection test passed for source %s", source.Name)
			return nil
		}

		sourceTypeValue := strings.TrimSpace(sourceType)
		if sourceTypeValue == "" {
			return fmt.Errorf("--type is required when testing a draft source config")
		}
		types, err := session.Client.SourceTypes(ctx)
		if err != nil {
			return err
		}
		selectedType, err := resolveSourceType(types, sourceTypeValue)
		if err != nil {
			return err
		}
		input, err := collectSourceDraftInput(cmd, selectedType)
		if err != nil {
			return err
		}
		if err := session.Client.TestSourceConnection(ctx, input); err != nil {
			return fmt.Errorf("draft source configuration failed: %w", err)
		}
		data := sourceTestOutput{Status: "ok", SourceType: selectedType.ID}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "sources.test", data)
		}
		out.Success("Connection test passed for %s", selectedType.ID)
		return nil
	},
}

func loadSourceConfigContext(ctx context.Context, session *platformSession, sourceValue string) (*platform.Organization, *platform.Project, *platform.Source, *platform.SourceType, *platform.SourceConfig, error) {
	org, project, source, err := resolvePlatformSource(ctx, session, sourcesOrg, sourcesProject, sourceValue)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	types, err := session.Client.SourceTypes(ctx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	sourceType := findSourceType(types, source.DatabaseType)
	config, err := session.Client.SourceConfig(ctx, org.ID, project.ID, source.ID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return org, project, source, sourceType, config, nil
}

func findSourceType(types []platform.SourceType, value string) *platform.SourceType {
	for i := range types {
		if matchesPlatformIdentifier(value, types[i].ID, types[i].Connector, types[i].Label) {
			return &types[i]
		}
	}
	return nil
}

func collectSourceDraftInput(cmd *cobra.Command, sourceType *platform.SourceType) (platform.CreateSourceInput, error) {
	explicitValues, err := explicitSourceCreateValues(cmd)
	if err != nil {
		return platform.CreateSourceInput{}, err
	}
	advanced, err := parseSourceAdvanced(sourceAdvanced)
	if err != nil {
		return platform.CreateSourceInput{}, err
	}
	values, remainingAdvanced, err := collectSourceFieldValues(sourceType.ConnectionFields, explicitValues, advanced, func(field platform.SourceConnectionField) (string, error) {
		return promptSourceField(cmd.ErrOrStderr(), field)
	})
	if err != nil {
		return platform.CreateSourceInput{}, err
	}
	return buildCreateSourceInput("", sourceType.ID, "connection-test", values, remainingAdvanced), nil
}

func explicitSourceConfigValues(cmd *cobra.Command, sourceType *platform.SourceType) (map[string]string, map[string]string, error) {
	explicitValues, err := explicitSourceCreateValues(cmd)
	if err != nil {
		return nil, nil, err
	}

	fieldKeyByLower := map[string]string{}
	if sourceType != nil {
		fieldKeyByLower = make(map[string]string, len(sourceType.ConnectionFields))
		for _, field := range sourceType.ConnectionFields {
			fieldKeyByLower[strings.ToLower(field.Key)] = field.Key
		}
	}

	values := map[string]string{}
	for key, value := range explicitValues {
		canonicalKey := key
		if sourceType != nil {
			var ok bool
			canonicalKey, ok = fieldKeyByLower[strings.ToLower(strings.TrimSpace(key))]
			if !ok {
				return nil, nil, fmt.Errorf("source type does not define connection field %q; run whodb-cli sources fields %s to list valid fields", key, sourceType.ID)
			}
		}
		values[canonicalKey] = value
	}

	parsedAdvanced, err := parseSourceAdvanced(sourceAdvanced)
	if err != nil {
		return nil, nil, err
	}
	advanced := map[string]string{}
	for key, value := range parsedAdvanced {
		if sourceType != nil {
			if canonicalKey, ok := fieldKeyByLower[strings.ToLower(strings.TrimSpace(key))]; ok {
				values[canonicalKey] = value
				continue
			}
		}
		advanced[key] = value
	}
	return values, advanced, nil
}

func sourceConfigFlagsChanged(cmd *cobra.Command) bool {
	for _, flag := range []string{"hostname", "port", "username", "database", "password-env", "password-stdin", "field", "advanced"} {
		if cmd.Flags().Changed(flag) {
			return true
		}
	}
	return false
}

func mergeSourceConfig(existing *platform.SourceConfig, values map[string]string, advanced map[string]string) platform.SourceConfig {
	merged := platform.SourceConfig{}
	if existing != nil {
		merged = *existing
		merged.Advanced = map[string]string{}
		for key, value := range existing.Advanced {
			merged.Advanced[key] = value
		}
	}
	if merged.Advanced == nil {
		merged.Advanced = map[string]string{}
	}
	for key, value := range values {
		assignSourceConfigField(&merged, key, value)
	}
	for key, value := range advanced {
		merged.Advanced[key] = value
	}
	return merged
}

func assignSourceConfigField(config *platform.SourceConfig, key, value string) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "hostname":
		config.Hostname = value
	case "port":
		config.Port = value
	case "username":
		config.Username = value
	case "password":
		config.Password = value
	case "database":
		config.Database = value
	default:
		if config.Advanced == nil {
			config.Advanced = map[string]string{}
		}
		config.Advanced[key] = value
	}
}

func redactSourceConfig(config *platform.SourceConfig, sourceType *platform.SourceType) sourceConfigOutput {
	if config == nil {
		return sourceConfigOutput{Advanced: map[string]string{}}
	}
	safe := sourceConfigOutput{
		Hostname: config.Hostname,
		Port:     config.Port,
		Username: config.Username,
		Password: redactValue("Password", config.Password, sourceType),
		Database: config.Database,
		Advanced: map[string]string{},
	}
	for key, value := range config.Advanced {
		safe.Advanced[key] = redactValue(key, value, sourceType)
	}
	return safe
}

func redactValue(key, value string, sourceType *platform.SourceType) string {
	if value == "" {
		return ""
	}
	if sourceConfigFieldSecret(key, sourceType) {
		return redactedValue
	}
	return value
}

func sourceConfigFieldSecret(key string, sourceType *platform.SourceType) bool {
	if strings.EqualFold(key, "Password") {
		return true
	}
	if sourceType != nil {
		for _, field := range sourceType.ConnectionFields {
			if strings.EqualFold(field.Key, key) {
				return sourceFieldSecret(field)
			}
		}
	}
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "_", " "))
	for _, part := range []string{"password", "secret", "token", "private key"} {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}

func sourceConfigRows(source *platform.Source, project *platform.Project, config sourceConfigOutput) [][]any {
	rows := [][]any{
		{"source_id", source.ID},
		{"source_name", source.Name},
		{"source_type", source.DatabaseType},
		{"project", project.Name},
		{"Hostname", config.Hostname},
		{"Port", config.Port},
		{"Username", config.Username},
		{"Password", config.Password},
		{"Database", config.Database},
	}
	keys := make([]string, 0, len(config.Advanced))
	for key := range config.Advanced {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rows = append(rows, []any{key, config.Advanced[key]})
	}
	return rows
}
