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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

const transformPlanKind = "WhoDBTransformPlan"

var (
	transformApplyManifest   string
	transformApplyCatalogs   []string
	transformApplyBaseDir    string
	transformApplyDryRun     bool
	transformApplyOverwrite  bool
	transformApplyRun        bool
	transformApplyWait       bool
	transformApplyTimeout    time.Duration
	transformApplyPoll       time.Duration
	transformApplyOnlyEmpty  bool
	transformApplyNamePrefix string
	transformApplyYes        bool
)

type transformPlan struct {
	APIVersion string               `json:"apiVersion" yaml:"apiVersion"`
	Kind       string               `json:"kind" yaml:"kind"`
	Transforms []transformPlanEntry `json:"transforms" yaml:"transforms"`
}

type transformPlanEntry struct {
	Name         string `json:"name" yaml:"name"`
	Description  string `json:"description" yaml:"description"`
	Ontology     string `json:"ontology" yaml:"ontology"`
	File         string `json:"file" yaml:"file"`
	WriteMode    string `json:"writeMode" yaml:"writeMode"`
	TriggerMode  string `json:"triggerMode" yaml:"triggerMode"`
	ScheduleCron string `json:"scheduleCron" yaml:"scheduleCron"`
	Run          *bool  `json:"run,omitempty" yaml:"run,omitempty"`
}

type transformApplyResult struct {
	Name      string `json:"name"`
	Ontology  string `json:"ontology"`
	File      string `json:"file"`
	Action    string `json:"action"`
	RunStatus string `json:"runStatus,omitempty"`
	Error     string `json:"error,omitempty"`
}

var transformsApplyCmd = withExample(&cobra.Command{
	Use:           "apply (--manifest <path> | --catalog <path>...)",
	Short:         "Apply and optionally run a declarative transform plan",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runTransformApply,
}, `  whodb transforms apply --manifest fixtures.yaml --dry-run
  whodb transforms apply --manifest fixtures.yaml --run --yes
  whodb transforms apply --manifest fixtures.yaml --overwrite --run --yes --format json
  whodb transforms apply --catalog seeds.json --catalog analytics.json --only-empty --run --yes`)

func runTransformApply(cmd *cobra.Command, _ []string) error {
	if strings.TrimSpace(transformApplyManifest) == "" && len(transformApplyCatalogs) == 0 {
		return fmt.Errorf("--manifest or at least one --catalog is required")
	}
	if strings.TrimSpace(transformApplyManifest) != "" && len(transformApplyCatalogs) > 0 {
		return fmt.Errorf("--manifest and --catalog are mutually exclusive")
	}
	var plan transformPlan
	manifestDir := ""
	var err error
	if strings.TrimSpace(transformApplyManifest) != "" {
		plan, manifestDir, err = readTransformPlan(transformApplyManifest)
		if err != nil {
			return err
		}
	}
	if !transformApplyDryRun && !transformApplyYes {
		return fmt.Errorf("--yes is required to upload files, save transforms, or run them; use --dry-run to preview")
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

	ontologies, err := session.Client.Ontologies(ctx, project.ID)
	if err != nil {
		return err
	}
	if len(transformApplyCatalogs) > 0 {
		plan, err = readTransformCatalogs(transformApplyCatalogs, ontologies)
		if err != nil {
			return err
		}
		manifestDir = "."
	}
	if err := validateTransformPlan(plan); err != nil {
		return err
	}
	transforms, err := session.Client.Transforms(ctx, project.ID)
	if err != nil {
		return err
	}
	files, err := session.Client.ProjectTabularFiles(ctx, project.ID)
	if err != nil {
		return err
	}

	ontologyByRef := indexOntologies(ontologies)
	transformByName := make(map[string]platform.Transform, len(transforms))
	for _, transform := range transforms {
		transformByName[strings.ToLower(transform.Name)] = transform
	}
	fileByName := make(map[string]platform.ProjectFile, len(files))
	for _, file := range files {
		fileByName[strings.ToLower(file.Name)] = file
	}

	baseDir := manifestDir
	if strings.TrimSpace(transformApplyBaseDir) != "" {
		baseDir, err = filepath.Abs(transformApplyBaseDir)
		if err != nil {
			return err
		}
	}
	results := make([]transformApplyResult, 0, len(plan.Transforms))
	for _, entry := range plan.Transforms {
		result := transformApplyResult{Name: entry.Name, Ontology: entry.Ontology, File: entry.File}
		ontology, ok := ontologyByRef[strings.ToLower(entry.Ontology)]
		if !ok {
			result.Error = "ontology not found"
			results = append(results, result)
			continue
		}
		if transformApplyOnlyEmpty {
			rows, queryErr := session.Client.OntologyRows(ctx, project.ID, ontology.ID, 1, 0)
			if queryErr != nil {
				result.Error = fmt.Sprintf("query existing rows: %v", queryErr)
				results = append(results, result)
				continue
			}
			if rows.Total > 0 {
				result.Action = "skipped-nonempty"
				results = append(results, result)
				continue
			}
		}
		filePath := entry.File
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(baseDir, filePath)
		}
		fileName := filepath.Base(filePath)
		file, fileExists := fileByName[strings.ToLower(fileName)]
		transform, transformExists := transformByName[strings.ToLower(entry.Name)]
		if transformApplyDryRun {
			result.Action = previewTransformApplyAction(fileExists, transformExists, transformApplyOverwrite, shouldRunTransform(entry))
			results = append(results, result)
			continue
		}
		if !fileExists || transformApplyOverwrite {
			if _, err := os.Stat(filePath); err != nil {
				result.Error = fmt.Sprintf("file unavailable: %v", err)
				results = append(results, result)
				continue
			}
			uploaded, err := session.Client.UploadProjectFile(ctx, project.ID, nil, filePath)
			if err != nil {
				result.Error = err.Error()
				results = append(results, result)
				continue
			}
			file = *uploaded
			fileByName[strings.ToLower(fileName)] = file
			result.Action = "uploaded"
		}

		graphJSON, err := buildFileOntologyGraph(file.ID, ontology, entry.WriteMode)
		if err != nil {
			return err
		}
		if !transformExists || transformApplyOverwrite {
			input := map[string]any{
				"projectId":    project.ID,
				"name":         entry.Name,
				"description":  entry.Description,
				"graphJson":    graphJSON,
				"scheduleCron": entry.ScheduleCron,
				"triggerMode":  defaultString(entry.TriggerMode, "manual"),
			}
			if transformExists {
				input["id"] = transform.ID
			}
			mutation, err := session.Client.PlatformMutation(ctx, "SaveTransform", map[string]any{"input": input})
			if err != nil {
				result.Error = err.Error()
				results = append(results, result)
				continue
			}
			if err := json.Unmarshal(mutation.Data, &transform); err != nil {
				result.Error = fmt.Sprintf("decode saved transform: %v", err)
				results = append(results, result)
				continue
			}
			transformByName[strings.ToLower(entry.Name)] = transform
			if transformExists {
				if result.Action == "uploaded" {
					result.Action = "uploaded+updated"
				} else {
					result.Action = "updated"
				}
			} else if result.Action == "uploaded" {
				result.Action = "uploaded+created"
			} else {
				result.Action = "created"
			}
		} else if result.Action == "" {
			result.Action = "reused"
		}
		if shouldRunTransform(entry) {
			mutation, err := session.Client.PlatformMutation(ctx, "RunTransform", map[string]any{"projectId": project.ID, "id": transform.ID})
			if err != nil {
				result.Error = err.Error()
				results = append(results, result)
				continue
			}
			var run platform.TransformRun
			if err := json.Unmarshal(mutation.Data, &run); err != nil {
				result.Error = fmt.Sprintf("decode transform run: %v", err)
			} else {
				if transformApplyWait && !isTerminalTransformRunStatus(run.Status) {
					run, err = waitForTransformRun(ctx, session.Client, project.ID, transform.ID, run.ID, transformApplyTimeout, transformApplyPoll)
					if err != nil {
						result.Error = err.Error()
					}
				}
				result.RunStatus = run.Status
				if isFailedTransformRunStatus(run.Status) && result.Error == "" {
					result.Error = run.ErrorMessage
				}
			}
		}
		results = append(results, result)
	}
	return writeTransformApplyResults(cmd, results)
}

type transformCatalogFile struct {
	APIName string `json:"api_name" yaml:"api_name"`
	File    string `json:"file" yaml:"file"`
}

type transformCatalogOntology struct {
	APIName string `json:"apiName" yaml:"apiName"`
	Source  string `json:"source" yaml:"source"`
}

type transformCatalog struct {
	Files          []transformCatalogFile     `json:"files" yaml:"files"`
	ReferenceFiles []transformCatalogFile     `json:"reference_files" yaml:"reference_files"`
	Ontologies     []transformCatalogOntology `json:"ontologies" yaml:"ontologies"`
}

func readTransformCatalogs(paths []string, ontologies []platform.Ontology) (transformPlan, error) {
	filesByAPIName := map[string]string{}
	for _, path := range paths {
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return transformPlan{}, err
		}
		file, err := os.Open(absolutePath)
		if err != nil {
			return transformPlan{}, err
		}
		var catalog transformCatalog
		decodeErr := yaml.NewDecoder(file).Decode(&catalog)
		closeErr := file.Close()
		if decodeErr != nil {
			return transformPlan{}, fmt.Errorf("decode transform catalog %s: %w", path, decodeErr)
		}
		if closeErr != nil {
			return transformPlan{}, closeErr
		}
		baseDir := filepath.Dir(absolutePath)
		for _, entry := range append(catalog.Files, catalog.ReferenceFiles...) {
			if entry.APIName != "" && entry.File != "" {
				filesByAPIName[strings.ToLower(entry.APIName)] = absoluteCatalogFile(baseDir, entry.File)
			}
		}
		for _, entry := range catalog.Ontologies {
			if entry.APIName != "" && entry.Source != "" {
				filesByAPIName[strings.ToLower(entry.APIName)] = absoluteCatalogFile(baseDir, entry.Source)
			}
		}
	}
	prefix := strings.TrimSpace(transformApplyNamePrefix)
	if prefix == "" {
		prefix = "WhoDB Fixture"
	}
	sortedOntologies := append([]platform.Ontology(nil), ontologies...)
	sort.Slice(sortedOntologies, func(i, j int) bool { return sortedOntologies[i].APIName < sortedOntologies[j].APIName })
	entries := make([]transformPlanEntry, 0, len(sortedOntologies))
	for _, ontology := range sortedOntologies {
		path, ok := filesByAPIName[strings.ToLower(ontology.APIName)]
		if !ok {
			continue
		}
		entries = append(entries, transformPlanEntry{
			Name:      prefix + " " + ontology.DisplayName,
			Ontology:  ontology.APIName,
			File:      path,
			WriteMode: "truncate_and_insert",
		})
	}
	if len(entries) == 0 {
		return transformPlan{}, fmt.Errorf("catalogs contain no file mappings for project ontologies")
	}
	return transformPlan{APIVersion: "whodb.com/v1alpha1", Kind: transformPlanKind, Transforms: entries}, nil
}

func absoluteCatalogFile(baseDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

func waitForTransformRun(ctx context.Context, client *platform.Client, projectID, transformID, runID string, timeout, pollInterval time.Duration) (platform.TransformRun, error) {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	deadline := time.Now().Add(timeout)
	latest := platform.TransformRun{ID: runID, TransformID: transformID, Status: "running"}
	var lastQueryError error
	for {
		runs, err := client.TransformRuns(ctx, projectID, transformID, 20)
		if err != nil {
			lastQueryError = err
		} else {
			for _, candidate := range runs {
				if candidate.ID == runID {
					latest = candidate
					break
				}
			}
			if isTerminalTransformRunStatus(latest.Status) {
				return latest, nil
			}
		}
		if time.Now().After(deadline) {
			if lastQueryError != nil {
				return latest, fmt.Errorf("timed out waiting for transform run %s after %s (last query error: %v)", runID, timeout, lastQueryError)
			}
			return latest, fmt.Errorf("timed out waiting for transform run %s after %s", runID, timeout)
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return latest, ctx.Err()
		case <-timer.C:
		}
	}
}

func isTerminalTransformRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "succeeded", "success", "failed", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func isFailedTransformRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func readTransformPlan(path string) (transformPlan, string, error) {
	var plan transformPlan
	absPath, err := filepath.Abs(path)
	if err != nil {
		return plan, "", err
	}
	file, err := os.Open(absPath)
	if err != nil {
		return plan, "", err
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&plan); err != nil {
		return plan, "", fmt.Errorf("decode transform plan: %w", err)
	}
	return plan, filepath.Dir(absPath), nil
}

func validateTransformPlan(plan transformPlan) error {
	if plan.Kind != transformPlanKind {
		return fmt.Errorf("manifest kind must be %q", transformPlanKind)
	}
	if len(plan.Transforms) == 0 {
		return fmt.Errorf("manifest must contain at least one transform")
	}
	seen := map[string]struct{}{}
	for index, entry := range plan.Transforms {
		if strings.TrimSpace(entry.Name) == "" || strings.TrimSpace(entry.Ontology) == "" || strings.TrimSpace(entry.File) == "" {
			return fmt.Errorf("transform %d requires name, ontology, and file", index+1)
		}
		key := strings.ToLower(entry.Name)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate transform name %q", entry.Name)
		}
		seen[key] = struct{}{}
		switch defaultString(entry.WriteMode, "upsert") {
		case "upsert", "insert", "truncate_and_insert", "create_or_replace":
		default:
			return fmt.Errorf("transform %q has unsupported writeMode %q", entry.Name, entry.WriteMode)
		}
	}
	return nil
}

func indexOntologies(ontologies []platform.Ontology) map[string]platform.Ontology {
	result := make(map[string]platform.Ontology, len(ontologies)*3)
	for _, ontology := range ontologies {
		for _, ref := range []string{ontology.ID, ontology.APIName, ontology.DisplayName} {
			result[strings.ToLower(ref)] = ontology
		}
	}
	return result
}

func buildFileOntologyGraph(fileID string, ontology platform.Ontology, writeMode string) (string, error) {
	graph := map[string]any{
		"nodes": []map[string]any{
			{"id": "file", "type": "file", "config": map[string]any{"fileId": fileID, "sourceKind": "file"}},
			{"id": "ontology", "type": "ontology", "config": map[string]any{"ontologyId": ontology.ID, "ontologyName": ontology.DisplayName, "writeMode": defaultString(writeMode, "upsert")}},
		},
		"edges": []map[string]string{{"source": "file", "target": "ontology"}},
	}
	raw, err := json.Marshal(graph)
	return string(raw), err
}

func shouldRunTransform(entry transformPlanEntry) bool {
	if entry.Run != nil {
		return *entry.Run
	}
	return transformApplyRun
}

func previewTransformApplyAction(fileExists, transformExists, overwrite, run bool) string {
	actions := make([]string, 0, 3)
	if !fileExists || overwrite {
		actions = append(actions, "upload")
	}
	if !transformExists {
		actions = append(actions, "create")
	} else if overwrite {
		actions = append(actions, "update")
	} else {
		actions = append(actions, "reuse")
	}
	if run {
		actions = append(actions, "run")
	}
	return strings.Join(actions, "+")
}

func writeTransformApplyResults(cmd *cobra.Command, results []transformApplyResult) error {
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	failed := 0
	for _, result := range results {
		if result.Error != "" {
			failed++
		}
	}
	if format == output.FormatJSON {
		if err := writeCommandJSON(cmd, automationEnvelope{Command: "transforms.apply", Success: failed == 0, Data: results}); err != nil {
			return err
		}
		if failed > 0 {
			return fmt.Errorf("%d transform plan entries failed", failed)
		}
		return nil
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	rows := make([][]any, len(results))
	for index, result := range results {
		rows[index] = []any{result.Name, result.Ontology, result.File, result.Action, result.RunStatus, result.Error}
	}
	if err := newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"name", "ontology", "file", "action", "run_status", "error"}, rows)); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d transform plan entries failed", failed)
	}
	return nil
}
