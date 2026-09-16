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
	"sort"
	"sync"

	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	ontologyDataStatusEmptyOnly   bool
	ontologyDataStatusConcurrency int
)

type ontologyDataStatus struct {
	APIName     string `json:"apiName"`
	DisplayName string `json:"displayName"`
	StorageMode string `json:"storageMode"`
	Rows        int    `json:"rows"`
	Error       string `json:"error,omitempty"`
}

var ontologyDataStatusCmd = withExample(&cobra.Command{
	Use:           "data-status",
	Short:         "Audit row availability across every ontology in a project",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runOntologyDataStatus,
}, `  whodb ontologies data-status --org retail --project fulfillment --format json
  whodb ontologies data-status --empty-only`)

func runOntologyDataStatus(cmd *cobra.Command, _ []string) error {
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
	concurrency := ontologyDataStatusConcurrency
	if concurrency <= 0 {
		concurrency = 6
	}
	statuses := make([]ontologyDataStatus, len(ontologies))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				ontology := ontologies[index]
				status := ontologyDataStatus{APIName: ontology.APIName, DisplayName: ontology.DisplayName, StorageMode: ontology.StorageMode}
				rows, queryErr := session.Client.OntologyRows(ctx, project.ID, ontology.ID, 1, 0)
				if queryErr != nil {
					status.Error = queryErr.Error()
				} else {
					status.Rows = rows.Total
				}
				statuses[index] = status
			}
		}()
	}
	for index := range ontologies {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].APIName < statuses[j].APIName })
	if ontologyDataStatusEmptyOnly {
		filtered := statuses[:0]
		for _, status := range statuses {
			if status.Rows == 0 || status.Error != "" {
				filtered = append(filtered, status)
			}
		}
		statuses = filtered
	}
	failed := 0
	for _, status := range statuses {
		if status.Error != "" {
			failed++
		}
	}
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		if err := writeCommandJSON(cmd, automationEnvelope{Command: "ontologies.data-status", Success: failed == 0, Data: statuses}); err != nil {
			return err
		}
	} else {
		rows := make([][]any, len(statuses))
		for index, status := range statuses {
			rows[index] = []any{status.APIName, status.DisplayName, status.StorageMode, status.Rows, status.Error}
		}
		if err := newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"api_name", "display_name", "storage_mode", "rows", "error"}, rows)); err != nil {
			return err
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d ontology row queries failed", failed)
	}
	return nil
}
