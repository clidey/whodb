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
	"regexp"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var datasetCloneCmd = platformCloneCommand("clone <dataset> <new-name>", "Clone a hosted WhoDB dataset definition", "dataset")
var ontologyCloneCmd = platformCloneCommand("clone <ontology> <new-api-name>", "Clone a hosted WhoDB ontology definition", "ontology")
var transformCloneCmd = platformCloneCommand("clone <transform> <new-name>", "Clone a hosted WhoDB transform definition", "transform")
var functionCloneCmd = platformCloneCommand("clone <function> <new-name>", "Clone a hosted WhoDB function draft", "function")

func platformCloneCommand(use, short, resource string) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Args:          cobra.ExactArgs(2),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlatformClone(cmd, resource, args[0], args[1])
		},
	}
}

func runPlatformClone(cmd *cobra.Command, resource, sourceRef, newName string) error {
	ctx := context.Background()
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	org, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return err
	}
	session.Host.DefaultOrgID = org.ID
	session.Host.DefaultOrgName = org.Name
	session.Host.DefaultProjectID = project.ID
	session.Host.DefaultProjectName = project.Name
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	payload, err := buildPlatformClonePayload(ctx, session, project.ID, resource, sourceRef, newName)
	if err != nil {
		return err
	}
	spec := platform.GenericWriteSpecs["create:"+resource]
	if !platformWriteYes {
		approved, err := confirmPlatformResourceWrite(cmd.InOrStdin(), cmd.ErrOrStderr(), spec, project.Name, payload)
		if err != nil {
			return err
		}
		if !approved {
			return fmt.Errorf("write cancelled")
		}
	}
	variablesPayload := clonePayload(payload)
	_, variables, err := buildGenericResourceVariables(project.ID, genericResourceWriteInput{Resource: resource, Action: "create"}, variablesPayload)
	if err != nil {
		return err
	}
	result, err := session.Client.PlatformMutation(ctx, spec.Mutation, variables)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, resource+".clone", result)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"field", "value"}, [][]any{
		{"resource", resource},
		{"source", sourceRef},
		{"new_name", newName},
		{"mutation", spec.Mutation},
	}))
}

func buildPlatformClonePayload(ctx context.Context, session *platformSession, projectID, resource, sourceRef, newName string) (map[string]any, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return nil, fmt.Errorf("new name cannot be empty")
	}
	switch resource {
	case "dataset":
		id, err := resolvePlatformResourceID(ctx, session, projectID, "dataset", sourceRef)
		if err != nil {
			return nil, err
		}
		dataset, err := session.Client.Dataset(ctx, projectID, id)
		if err != nil {
			return nil, err
		}
		payload := datasetCreatePayloadFromExport(*dataset)
		payload["name"] = newName
		return payload, nil
	case "ontology":
		id, err := resolvePlatformResourceID(ctx, session, projectID, "ontology", sourceRef)
		if err != nil {
			return nil, err
		}
		ontology, err := session.Client.Ontology(ctx, projectID, id)
		if err != nil {
			return nil, err
		}
		payload := ontologyCreatePayloadFromExport(*ontology)
		identifier := safePlatformIdentifier(newName)
		payload["apiName"] = identifier
		payload["displayName"] = newName
		payload["pluralDisplayName"] = newName + "s"
		payload["tableName"] = identifier
		return payload, nil
	case "transform":
		transform, err := resolveTransform(ctx, session, projectID, sourceRef)
		if err != nil {
			return nil, err
		}
		payload := transformCreatePayloadFromExport(*transform)
		payload["name"] = newName
		return payload, nil
	case "function":
		id, err := resolvePlatformResourceID(ctx, session, projectID, "function", sourceRef)
		if err != nil {
			return nil, err
		}
		fn, err := session.Client.Function(ctx, projectID, id, nil)
		if err != nil {
			return nil, err
		}
		payload := functionCreatePayloadFromExport(*fn, true)
		payload["name"] = newName
		return payload, nil
	default:
		return nil, fmt.Errorf("unsupported clone resource %q", resource)
	}
}

func safePlatformIdentifier(value string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	identifier := strings.Trim(re.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "_"), "_")
	if identifier == "" {
		return "clone"
	}
	if identifier[0] >= '0' && identifier[0] <= '9' {
		return "x_" + identifier
	}
	return identifier
}

func clonePayload(payload map[string]any) map[string]any {
	copy := make(map[string]any, len(payload))
	for key, value := range payload {
		copy[key] = value
	}
	return copy
}
