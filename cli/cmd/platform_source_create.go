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
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type sourceFieldPrompt func(platform.SourceConnectionField) (string, error)

func sourceTypeFromCreateArgs(args []string, flagValue string) (string, error) {
	argValue := ""
	if len(args) > 0 {
		argValue = strings.TrimSpace(args[0])
	}
	flagValue = strings.TrimSpace(flagValue)
	if argValue != "" && flagValue != "" && !strings.EqualFold(argValue, flagValue) {
		return "", fmt.Errorf("source type %q conflicts with --type %q", argValue, flagValue)
	}
	if argValue != "" {
		return argValue, nil
	}
	if flagValue != "" {
		return flagValue, nil
	}
	return "", fmt.Errorf("source type is required")
}

func collectSourceCreateInput(cmd *cobra.Command, sourceType *platform.SourceType) (platform.CreateSourceInput, error) {
	name, err := collectSourceName(cmd)
	if err != nil {
		return platform.CreateSourceInput{}, err
	}
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
	input := buildCreateSourceInput("", sourceType.ID, name, values, remainingAdvanced)
	return input, nil
}

func collectSourceName(cmd *cobra.Command) (string, error) {
	name := strings.TrimSpace(sourceName)
	if name != "" {
		return name, nil
	}
	if !isInteractiveInput() {
		return "", fmt.Errorf("--name is required")
	}
	fmt.Fprint(cmd.ErrOrStderr(), "Name: ")
	reader := bufio.NewReader(cmd.InOrStdin())
	value, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading source name: %w", err)
	}
	name = strings.TrimSpace(value)
	if name == "" {
		return "", fmt.Errorf("--name is required")
	}
	return name, nil
}

func explicitSourceCreateValues(cmd *cobra.Command) (map[string]string, error) {
	values := map[string]string{}
	for _, item := range []struct {
		flag  string
		key   string
		value string
	}{
		{flag: "hostname", key: "Hostname", value: sourceHostname},
		{flag: "port", key: "Port", value: sourcePort},
		{flag: "username", key: "Username", value: sourceUsername},
		{flag: "database", key: "Database", value: sourceDatabase},
	} {
		if cmd.Flags().Changed(item.flag) {
			values[item.key] = strings.TrimSpace(item.value)
		}
	}

	generic, err := parseSourceAdvanced(sourceFields)
	if err != nil {
		return nil, err
	}
	for key, value := range generic {
		values[key] = value
	}

	if cmd.Flags().Changed("password-env") || cmd.Flags().Changed("password-stdin") {
		password, err := readSourcePassword(cmd)
		if err != nil {
			return nil, err
		}
		values["Password"] = password
	}
	return values, nil
}

func collectSourceFieldValues(fields []platform.SourceConnectionField, explicitValues map[string]string, advanced map[string]string, prompt sourceFieldPrompt) (map[string]string, map[string]string, error) {
	fieldKeyByLower := make(map[string]string, len(fields))
	for _, field := range fields {
		fieldKeyByLower[strings.ToLower(field.Key)] = field.Key
	}

	values := map[string]string{}
	for key, value := range explicitValues {
		canonicalKey, ok := fieldKeyByLower[strings.ToLower(strings.TrimSpace(key))]
		if !ok {
			return nil, nil, fmt.Errorf("source type does not define connection field %q; run whodb-cli sources fields <source-type> to list valid fields", key)
		}
		values[canonicalKey] = value
	}

	remainingAdvanced := make(map[string]string, len(advanced))
	for key, value := range advanced {
		canonicalKey, ok := fieldKeyByLower[strings.ToLower(strings.TrimSpace(key))]
		if ok {
			values[canonicalKey] = value
			continue
		}
		remainingAdvanced[key] = value
	}

	for _, field := range fields {
		value, ok := values[field.Key]
		if !ok && field.DefaultValue != nil {
			value = *field.DefaultValue
			ok = true
		}
		if (!ok || strings.TrimSpace(value) == "") && field.Required {
			if prompt == nil || !isInteractiveInput() {
				return nil, nil, fmt.Errorf("source field %s is required; pass --field %s=value or run whodb-cli sources fields <source-type> to inspect required fields", field.Key, field.Key)
			}
			prompted, err := prompt(field)
			if err != nil {
				return nil, nil, err
			}
			value = prompted
			ok = true
		}
		if field.Required && strings.TrimSpace(value) == "" {
			return nil, nil, fmt.Errorf("source field %s is required; run whodb-cli sources fields <source-type> to inspect required fields", field.Key)
		}
		if ok {
			values[field.Key] = value
		}
	}

	return values, remainingAdvanced, nil
}

func promptSourceField(out io.Writer, field platform.SourceConnectionField) (string, error) {
	label := field.Key
	if field.DefaultValue != nil && *field.DefaultValue != "" && !platform.SourceFieldSecret(field) {
		label = fmt.Sprintf("%s [%s]", label, *field.DefaultValue)
	}
	fmt.Fprintf(out, "%s: ", label)
	if platform.SourceFieldSecret(field) {
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", field.Key, err)
		}
		return string(value), nil
	}
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading %s: %w", field.Key, err)
	}
	value = strings.TrimRight(value, "\r\n")
	if value == "" && field.DefaultValue != nil {
		return *field.DefaultValue, nil
	}
	return value, nil
}

func buildCreateSourceInput(projectID, sourceType, name string, values map[string]string, advanced map[string]string) platform.CreateSourceInput {
	input := platform.CreateSourceInput{
		ProjectID:    projectID,
		Name:         strings.TrimSpace(name),
		DatabaseType: sourceType,
		Advanced:     map[string]string{},
	}
	for key, value := range advanced {
		input.Advanced[key] = value
	}
	for key, value := range values {
		assignCreateSourceField(&input, key, value)
	}
	return input
}

func assignCreateSourceField(input *platform.CreateSourceInput, key, value string) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "hostname":
		input.Hostname = value
	case "port":
		input.Port = value
	case "username":
		input.Username = value
	case "password":
		input.Password = value
	case "database":
		input.Database = value
	default:
		input.Advanced[key] = value
	}
}

func resolveSourceType(types []platform.SourceType, value string) (*platform.SourceType, error) {
	needle := strings.TrimSpace(value)
	if needle == "" {
		return nil, fmt.Errorf("source type is required")
	}
	for _, sourceType := range types {
		if matchesPlatformIdentifier(needle, sourceType.ID, sourceType.Connector, sourceType.Label) {
			return &sourceType, nil
		}
	}
	return nil, fmt.Errorf("source type %q not found", needle)
}

func sortSourceTypes(types []platform.SourceType) {
	sort.SliceStable(types, func(i, j int) bool {
		return strings.ToLower(types[i].ID) < strings.ToLower(types[j].ID)
	})
}

func sourceFieldDefault(field platform.SourceConnectionField) string {
	if field.DefaultValue == nil {
		return ""
	}
	return *field.DefaultValue
}

func sourceFieldSecret(field platform.SourceConnectionField) bool {
	return platform.SourceFieldSecret(field)
}
