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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	transformRebindNodeID      string
	transformRebindSourceID    string
	transformRebindSourceKind  string
	transformRebindSourcePath  []string
	transformRebindObjectKind  string
	transformRebindFileID      string
	transformRebindFilePath    string
	transformRebindCloneName   string
	transformRebindDescription string
	transformRebindDryRun      bool
	transformRebindYes         bool
)

type transformRebindResult struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Action    string `json:"action"`
	NodeID    string `json:"nodeId"`
	GraphJSON string `json:"graphJson"`
}

type transformSourceBinding struct {
	NodeType string
	Config   map[string]any
}

var transformsRebindSourceCmd = withExample(&cobra.Command{
	Use:           "rebind-source <transform>",
	Short:         "Replace a transform template source without raw API requests",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runTransformRebindSource,
}, `  whodb transforms rebind-source "Onboard Product (Canonical Contract)" --source-id SOURCE_ID --source-path public --source-path products --clone-name "Load customer products" --yes
  whodb transforms rebind-source TRANSFORM_ID --file ./products.parquet --node customer_source --dry-run --format json`)

func runTransformRebindSource(cmd *cobra.Command, args []string) error {
	if err := validateTransformRebindFlags(); err != nil {
		return err
	}
	ctx := cmd.Context()
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	org, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return err
	}
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	transforms, err := session.Client.Transforms(ctx, project.ID)
	if err != nil {
		return err
	}
	transform, ok := findTransform(transforms, args[0])
	if !ok {
		return fmt.Errorf("transform %q not found", args[0])
	}

	binding := transformSourceBinding{}
	if strings.TrimSpace(transformRebindFileID) != "" || strings.TrimSpace(transformRebindFilePath) != "" {
		fileID := strings.TrimSpace(transformRebindFileID)
		if transformRebindFilePath != "" {
			if transformRebindDryRun {
				fileID = "<upload:" + filepath.Base(transformRebindFilePath) + ">"
			} else {
				if !transformRebindYes {
					return fmt.Errorf("--yes is required to upload and save; use --dry-run to preview")
				}
				uploaded, err := session.Client.UploadProjectFile(ctx, project.ID, nil, transformRebindFilePath)
				if err != nil {
					return err
				}
				fileID = uploaded.ID
			}
		}
		binding = transformSourceBinding{NodeType: "file", Config: map[string]any{"sourceKind": "file", "fileId": fileID}}
	} else {
		path := append([]string(nil), transformRebindSourcePath...)
		binding = transformSourceBinding{NodeType: "source", Config: map[string]any{
			"sourceKind": defaultString(transformRebindSourceKind, "database"),
			"sourceId":   transformRebindSourceID,
			"sourceObjectRef": map[string]any{
				"Kind":    defaultString(transformRebindObjectKind, "Table"),
				"Path":    path,
				"Locator": path[len(path)-1],
			},
		}}
	}
	graphJSON, nodeID, err := rebindTransformGraph(transform.GraphJSON, transformRebindNodeID, binding)
	if err != nil {
		return err
	}
	name := transform.Name
	action := "updated"
	if strings.TrimSpace(transformRebindCloneName) != "" {
		name = transformRebindCloneName
		action = "cloned"
	}
	result := transformRebindResult{Name: name, Action: action, NodeID: nodeID, GraphJSON: graphJSON}
	if transformRebindDryRun {
		result.Action = "preview-" + action
		return writeTransformRebindResult(cmd, result)
	}
	if !transformRebindYes {
		return fmt.Errorf("--yes is required to save the rebound transform; use --dry-run to preview")
	}
	input := map[string]any{
		"projectId":    project.ID,
		"name":         name,
		"description":  defaultString(transformRebindDescription, transform.Description),
		"graphJson":    graphJSON,
		"scheduleCron": transform.ScheduleCron,
		"triggerMode":  defaultString(transform.TriggerMode, "manual"),
	}
	if action == "updated" {
		input["id"] = transform.ID
	}
	mutation, err := session.Client.PlatformMutation(ctx, "SaveTransform", map[string]any{"input": input})
	if err != nil {
		return err
	}
	var saved platform.Transform
	if err := json.Unmarshal(mutation.Data, &saved); err != nil {
		return fmt.Errorf("decode saved transform: %w", err)
	}
	result.ID = saved.ID
	return writeTransformRebindResult(cmd, result)
}

func validateTransformRebindFlags() error {
	fileBinding := strings.TrimSpace(transformRebindFileID) != "" || strings.TrimSpace(transformRebindFilePath) != ""
	sourceBinding := strings.TrimSpace(transformRebindSourceID) != "" || len(transformRebindSourcePath) > 0
	if fileBinding == sourceBinding {
		return fmt.Errorf("provide either --file-id/--file or --source-id with --source-path")
	}
	if transformRebindFileID != "" && transformRebindFilePath != "" {
		return fmt.Errorf("--file-id and --file are mutually exclusive")
	}
	if sourceBinding && (strings.TrimSpace(transformRebindSourceID) == "" || len(transformRebindSourcePath) == 0) {
		return fmt.Errorf("database source binding requires --source-id and at least one --source-path")
	}
	return nil
}

func findTransform(transforms []platform.Transform, ref string) (platform.Transform, bool) {
	ref = strings.ToLower(strings.TrimSpace(ref))
	for _, transform := range transforms {
		if strings.ToLower(transform.ID) == ref || strings.ToLower(transform.Name) == ref {
			return transform, true
		}
	}
	return platform.Transform{}, false
}

func rebindTransformGraph(raw, requestedNodeID string, binding transformSourceBinding) (string, string, error) {
	var graph map[string]any
	if err := json.Unmarshal([]byte(raw), &graph); err != nil {
		return "", "", fmt.Errorf("decode transform graph: %w", err)
	}
	nodes, ok := graph["nodes"].([]any)
	if !ok {
		return "", "", fmt.Errorf("transform graph has no nodes")
	}
	for _, value := range nodes {
		node, ok := value.(map[string]any)
		if !ok {
			continue
		}
		nodeID, _ := node["id"].(string)
		nodeType, _ := node["type"].(string)
		if requestedNodeID != "" && nodeID != requestedNodeID {
			continue
		}
		if requestedNodeID == "" && nodeType != "source" && nodeType != "file" && nodeType != "dataset" {
			continue
		}
		node["type"] = binding.NodeType
		node["config"] = binding.Config
		encoded, err := json.Marshal(graph)
		return string(encoded), nodeID, err
	}
	if requestedNodeID != "" {
		return "", "", fmt.Errorf("source node %q not found", requestedNodeID)
	}
	return "", "", fmt.Errorf("transform graph has no source, file, or dataset node")
}

func writeTransformRebindResult(cmd *cobra.Command, result transformRebindResult) error {
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, "transforms.rebind-source", result)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult(
		[]string{"id", "name", "action", "node_id", "graph_json"},
		[][]any{{result.ID, result.Name, result.Action, result.NodeID, result.GraphJSON}},
	))
}
