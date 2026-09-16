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
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/spf13/cobra"
)

var (
	ontologyImportFile        string
	ontologyImportAction      string
	ontologyImportExclude     []string
	ontologyImportConcurrency int
	ontologyImportReplayState bool
	ontologyImportDryRun      bool
	ontologyImportYes         bool
)

type ontologyImportBehavior struct {
	Document ontologyImportBehaviorDocument `json:"document"`
}

type ontologyImportBehaviorDocument struct {
	StateProperty string                                    `json:"state_property"`
	InitialState  string                                    `json:"initial_state"`
	Actions       map[string]ontologyImportActionDefinition `json:"actions"`
}

type ontologyImportActionDefinition struct {
	Input map[string]any `json:"input"`
	From  []string       `json:"from"`
	Set   map[string]any `json:"set"`
}

type ontologyActionExecution struct {
	ExecutionID      string         `json:"executionId"`
	Allowed          bool           `json:"allowed"`
	DenialCode       *string        `json:"denialCode"`
	RecordChanges    map[string]any `json:"recordChanges"`
	RecordVersion    *int           `json:"recordVersion"`
	ValidationErrors []string       `json:"validationErrors"`
}

type ontologyImportResult struct {
	Ontology    string   `json:"ontology"`
	File        string   `json:"file"`
	Action      string   `json:"action"`
	Rows        int      `json:"rows"`
	Imported    int      `json:"imported"`
	Transitions int      `json:"transitions"`
	Failed      int      `json:"failed"`
	Errors      []string `json:"errors,omitempty"`
	DryRun      bool     `json:"dryRun,omitempty"`
}

var ontologyRecordsImportCmd = withExample(&cobra.Command{
	Use:           "import <ontology>",
	Short:         "Import CSV, JSON, NDJSON, or Parquet rows through ontology behavior actions",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runOntologyRecordsImport,
}, `  whodb ontologies records import purchase_order --file purchase_orders.parquet --action create --yes
  whodb ontologies records import workflow_task --file tasks.ndjson --action create --replay-state --concurrency 8 --yes`)

var ontologyBehaviorGetCmd = withExample(&cobra.Command{
	Use:           "behavior <ontology>",
	Short:         "Get the active behavior document for an ontology",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runOntologyBehaviorGet,
}, `  whodb ontologies behavior fulfillment_order --org retail --project fulfillment --format json`)

func runOntologyBehaviorGet(cmd *cobra.Command, args []string) error {
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
	ontology, ok := indexOntologies(ontologies)[strings.ToLower(args[0])]
	if !ok {
		return fmt.Errorf("ontology %q not found", args[0])
	}
	data, err := session.Client.PlatformQuery(ctx, "OntologyBehavior", map[string]any{"projectId": project.ID, "ontologyId": ontology.ID})
	if err != nil {
		return err
	}
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, "ontologies.behavior.get", data)
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
	return err
}

func runOntologyRecordsImport(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(ontologyImportFile) == "" {
		return fmt.Errorf("--file is required")
	}
	if strings.TrimSpace(ontologyImportAction) == "" {
		return fmt.Errorf("--action is required")
	}
	if !ontologyImportDryRun && !ontologyImportYes {
		return fmt.Errorf("--yes is required to execute ontology actions; use --dry-run to inspect")
	}
	rows, err := readOntologyImportRows(cmd.Context(), ontologyImportFile)
	if err != nil {
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
	ontologies, err := session.Client.Ontologies(ctx, project.ID)
	if err != nil {
		return err
	}
	ontology, ok := indexOntologies(ontologies)[strings.ToLower(args[0])]
	if !ok {
		return fmt.Errorf("ontology %q not found", args[0])
	}
	behavior, err := loadOntologyImportBehavior(ctx, session.Client, project.ID, ontology.ID)
	if err != nil {
		return err
	}
	action, ok := behavior.Document.Actions[ontologyImportAction]
	if !ok {
		return fmt.Errorf("behavior action %q is not defined for ontology %s", ontologyImportAction, ontology.APIName)
	}
	result := ontologyImportResult{Ontology: ontology.APIName, File: ontologyImportFile, Action: ontologyImportAction, Rows: len(rows), DryRun: ontologyImportDryRun}
	if ontologyImportDryRun {
		return writeOntologyImportResult(cmd, result)
	}

	concurrency := ontologyImportConcurrency
	if concurrency <= 0 {
		concurrency = min(8, max(2, runtime.GOMAXPROCS(0)))
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				transitions, err := importOntologyBehaviorRow(ctx, session.Client, project.ID, ontology, behavior.Document, ontologyImportAction, action, rows[index], index)
				mu.Lock()
				if err != nil {
					result.Failed++
					if len(result.Errors) < 50 {
						result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", index+1, err))
					}
				} else {
					result.Imported++
					result.Transitions += transitions
				}
				mu.Unlock()
			}
		}()
	}
	for index := range rows {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()
	if err := writeOntologyImportResult(cmd, result); err != nil {
		return err
	}
	if result.Failed > 0 {
		return fmt.Errorf("%d of %d ontology rows failed", result.Failed, result.Rows)
	}
	return nil
}

func loadOntologyImportBehavior(ctx context.Context, client *platform.Client, projectID, ontologyID string) (ontologyImportBehavior, error) {
	data, err := client.PlatformQuery(ctx, "OntologyBehavior", map[string]any{"projectId": projectID, "ontologyId": ontologyID})
	if err != nil {
		return ontologyImportBehavior{}, err
	}
	if data == nil {
		return ontologyImportBehavior{}, fmt.Errorf("ontology has no active behavior; use a transform or records add for direct ingestion")
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return ontologyImportBehavior{}, err
	}
	var behavior ontologyImportBehavior
	if err := json.Unmarshal(raw, &behavior); err != nil {
		return ontologyImportBehavior{}, fmt.Errorf("decode ontology behavior: %w", err)
	}
	return behavior, nil
}

func importOntologyBehaviorRow(ctx context.Context, client *platform.Client, projectID string, ontology platform.Ontology, behavior ontologyImportBehaviorDocument, actionName string, action ontologyImportActionDefinition, row map[string]any, index int) (int, error) {
	values := selectOntologyActionValues(row, action.Input, ontologyImportExclude)
	createResult, err := executeOntologyImportAction(ctx, client, projectID, ontology.ID, "", actionName, values, nil, ontologyImportKey(ontology.ID, actionName, row, index))
	if err != nil {
		return 0, err
	}
	if !ontologyImportReplayState || behavior.StateProperty == "" {
		return 0, nil
	}
	desiredState := strings.TrimSpace(fmt.Sprint(row[behavior.StateProperty]))
	if desiredState == "" || strings.EqualFold(desiredState, behavior.InitialState) {
		return 0, nil
	}
	path, err := ontologyImportTransitionPath(behavior, desiredState)
	if err != nil {
		return 0, err
	}
	recordKey := stringValue(createResult.RecordChanges[ontology.PrimaryKey])
	if recordKey == "" {
		recordKey = stringValue(row[ontology.PrimaryKey])
	}
	if recordKey == "" {
		return 0, fmt.Errorf("create action returned no primary key %q", ontology.PrimaryKey)
	}
	version := createResult.RecordVersion
	for step, transitionAction := range path {
		definition := behavior.Actions[transitionAction]
		transitionValues := selectOntologyActionValues(row, definition.Input, ontologyImportExclude)
		execution, err := executeOntologyImportAction(ctx, client, projectID, ontology.ID, recordKey, transitionAction, transitionValues, version, ontologyImportKey(ontology.ID, transitionAction, row, index*100+step+1))
		if err != nil {
			return step, err
		}
		version = execution.RecordVersion
	}
	return len(path), nil
}

func executeOntologyImportAction(ctx context.Context, client *platform.Client, projectID, ontologyID, recordKey, action string, values map[string]any, expectedVersion *int, idempotencyKey string) (ontologyActionExecution, error) {
	input := map[string]any{"projectId": projectID, "ontologyId": ontologyID, "action": action, "values": values, "idempotencyKey": idempotencyKey}
	if recordKey != "" {
		input["recordKey"] = recordKey
	}
	if expectedVersion != nil {
		input["expectedVersion"] = *expectedVersion
	}
	var mutation *platform.PlatformMutationResult
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		mutation, err = client.PlatformMutation(ctx, "ExecuteOntologyAction", map[string]any{"input": input})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "changed concurrently") {
			break
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 150 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ontologyActionExecution{}, ctx.Err()
		case <-timer.C:
		}
	}
	if err != nil {
		return ontologyActionExecution{}, err
	}
	var execution ontologyActionExecution
	if err := json.Unmarshal(mutation.Data, &execution); err != nil {
		return execution, fmt.Errorf("decode action result: %w", err)
	}
	if !execution.Allowed {
		parts := append([]string(nil), execution.ValidationErrors...)
		if execution.DenialCode != nil && *execution.DenialCode != "" {
			parts = append([]string{*execution.DenialCode}, parts...)
		}
		return execution, fmt.Errorf("action %s denied: %s", action, strings.Join(parts, "; "))
	}
	return execution, nil
}

func selectOntologyActionValues(row map[string]any, inputs map[string]any, excludes []string) map[string]any {
	excluded := make(map[string]struct{}, len(excludes))
	for _, field := range excludes {
		excluded[strings.ToLower(strings.TrimSpace(field))] = struct{}{}
	}
	values := make(map[string]any, len(inputs))
	for field := range inputs {
		if _, skip := excluded[strings.ToLower(field)]; skip {
			continue
		}
		if value, ok := row[field]; ok && value != nil {
			values[field] = normalizeOntologyImportValue(value)
		}
	}
	return values
}

func ontologyImportTransitionPath(behavior ontologyImportBehaviorDocument, desired string) ([]string, error) {
	type statePath struct {
		State string
		Path  []string
	}
	queue := []statePath{{State: behavior.InitialState}}
	visited := map[string]bool{strings.ToLower(behavior.InitialState): true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if strings.EqualFold(current.State, desired) {
			return current.Path, nil
		}
		names := make([]string, 0, len(behavior.Actions))
		for name := range behavior.Actions {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			action := behavior.Actions[name]
			if !containsFold(action.From, current.State) {
				continue
			}
			target := ontologyImportLiteralString(action.Set[behavior.StateProperty])
			if target == "" || strings.HasPrefix(target, "input.") {
				continue
			}
			key := strings.ToLower(target)
			if visited[key] {
				continue
			}
			visited[key] = true
			nextPath := append(append([]string(nil), current.Path...), name)
			queue = append(queue, statePath{State: target, Path: nextPath})
		}
	}
	return nil, fmt.Errorf("behavior has no transition path from %q to %q", behavior.InitialState, desired)
}

func ontologyImportLiteralString(value any) string {
	if literal, ok := value.(string); ok {
		return literal
	}
	expression, ok := value.(map[string]any)
	if !ok || !strings.EqualFold(stringValue(expression["kind"]), "literal") {
		return ""
	}
	return stringValue(expression["value"])
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func ontologyImportKey(ontologyID, action string, row map[string]any, index int) string {
	raw, _ := json.Marshal(row)
	sum := sha256.Sum256(append([]byte(fmt.Sprintf("%s:%s:%d:", ontologyID, action, index)), raw...))
	return "cli-import-" + hex.EncodeToString(sum[:16])
}

func readOntologyImportRows(ctx context.Context, path string) ([]map[string]any, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".parquet", ".pq":
		return readOntologyImportParquet(ctx, path)
	case ".csv":
		return readOntologyImportCSV(path)
	case ".json":
		return readOntologyImportJSON(path)
	case ".jsonl", ".ndjson":
		return readOntologyImportNDJSON(path)
	default:
		return nil, fmt.Errorf("unsupported import file type %q; use CSV, JSON, NDJSON, or Parquet", ext)
	}
}

func readOntologyImportParquet(ctx context.Context, path string) ([]map[string]any, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, "SELECT * FROM read_parquet(?)", path)
	if err != nil {
		return nil, fmt.Errorf("read parquet: %w", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		record := make(map[string]any, len(columns))
		for index, column := range columns {
			record[column] = normalizeOntologyImportValue(values[index])
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func readOntologyImportCSV(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	result := []map[string]any{}
	for {
		values, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		record := make(map[string]any, len(headers))
		for index, header := range headers {
			if index < len(values) && values[index] != "" {
				record[header] = values[index]
			}
		}
		result = append(result, record)
	}
	return result, nil
}

func readOntologyImportJSON(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.UseNumber()
	var rows []map[string]any
	if err := decoder.Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode JSON row array: %w", err)
	}
	return rows, nil
}

func readOntologyImportNDJSON(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.UseNumber()
	result := []map[string]any{}
	for {
		var row map[string]any
		if err := decoder.Decode(&row); err == io.EOF {
			return result, nil
		} else if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
}

func normalizeOntologyImportValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano)
	case json.Number:
		if integer, err := strconv.ParseInt(string(typed), 10, 64); err == nil {
			return integer
		}
		if decimal, err := strconv.ParseFloat(string(typed), 64); err == nil {
			return decimal
		}
		return string(typed)
	case fmt.Stringer:
		return typed.String()
	default:
		return value
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func writeOntologyImportResult(cmd *cobra.Command, result ontologyImportResult) error {
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, automationEnvelope{Command: "ontologies.records.import", Success: result.Failed == 0, Data: result})
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult(
		[]string{"ontology", "file", "action", "rows", "imported", "transitions", "failed", "errors"},
		[][]any{{result.Ontology, result.File, result.Action, result.Rows, result.Imported, result.Transitions, result.Failed, strings.Join(result.Errors, " | ")}},
	))
}
