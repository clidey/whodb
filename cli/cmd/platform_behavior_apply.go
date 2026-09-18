/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

type behaviorApplyManifest struct {
	Behaviors []behaviorApplyEntry `json:"behaviors"`
}

type behaviorApplyEntry struct {
	APIName          string         `json:"api_name"`
	ExpectedRevision int            `json:"expected_revision"`
	Document         map[string]any `json:"document"`
}

type behaviorApplyResult struct {
	APIName    string `json:"apiName"`
	OntologyID string `json:"ontologyId,omitempty"`
	Action     string `json:"action"`
	Error      string `json:"error,omitempty"`
}

var (
	ontologyBehaviorsCmd      = &cobra.Command{Use: "behaviors", Short: "Manage hosted WhoDB ontology behaviors"}
	ontologyBehaviorsApplyCmd = &cobra.Command{
		Use:           "apply",
		Short:         "Apply and activate ontology behaviors from a portable manifest",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runOntologyBehaviorsApply,
	}
	behaviorApplyManifestPath string
)

func registerOntologyBehaviorApplyCommand() {
	ontologyBehaviorsApplyCmd.Flags().StringVar(&behaviorApplyManifestPath, "manifest", "", "behavior manifest JSON file")
	ontologyBehaviorsApplyCmd.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "apply without first printing the plan")
	ontologyBehaviorsCmd.AddCommand(ontologyBehaviorsApplyCmd)
	ontologiesCmd.AddCommand(ontologyBehaviorsCmd)
}

func runOntologyBehaviorsApply(cmd *cobra.Command, _ []string) error {
	if strings.TrimSpace(behaviorApplyManifestPath) == "" {
		return fmt.Errorf("--manifest is required")
	}
	raw, err := os.ReadFile(behaviorApplyManifestPath)
	if err != nil {
		return err
	}
	var manifest behaviorApplyManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return fmt.Errorf("decode behavior manifest: %w", err)
	}
	if len(manifest.Behaviors) == 0 {
		return fmt.Errorf("behavior manifest has no behaviors")
	}
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
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	ontologies, err := session.Client.Ontologies(ctx, project.ID)
	if err != nil {
		return err
	}
	byExactName := make(map[string]platform.Ontology, len(ontologies))
	for _, ontology := range ontologies {
		byExactName[ontology.APIName] = ontology
	}
	results := make([]behaviorApplyResult, 0, len(manifest.Behaviors))
	for _, entry := range manifest.Behaviors {
		result := behaviorApplyResult{APIName: entry.APIName, Action: "skip"}
		ontology, ok := byExactName[entry.APIName]
		if !ok {
			result.Error = "ontology not found"
			results = append(results, result)
			continue
		}
		result.OntologyID = ontology.ID
		result.Action = "apply"
		if !platformWriteYes {
			results = append(results, result)
			continue
		}
		document := cloneBehaviorDocument(entry.Document)
		document["ontology_id"] = ontology.ID
		_, err := session.Client.PlatformMutation(ctx, "SaveBehavior", map[string]any{"input": map[string]any{
			"projectId":        project.ID,
			"ontologyId":       ontology.ID,
			"document":         document,
			"expectedRevision": entry.ExpectedRevision,
		}})
		if err != nil {
			result.Action = "failed"
			result.Error = err.Error()
		} else {
			result.Action = "applied"
		}
		results = append(results, result)
	}
	if err := writeBehaviorApplyResults(cmd, format, results); err != nil {
		return err
	}
	for _, result := range results {
		if result.Error != "" && result.Error != "ontology not found" {
			return fmt.Errorf("one or more ontology behaviors failed")
		}
	}
	return nil
}

func cloneBehaviorDocument(document map[string]any) map[string]any {
	raw, _ := json.Marshal(document)
	copy := map[string]any{}
	_ = json.Unmarshal(raw, &copy)
	return copy
}

func writeBehaviorApplyResults(cmd *cobra.Command, format output.Format, results []behaviorApplyResult) error {
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, automationEnvelope{Command: "ontologies.behaviors.apply", Success: true, Data: results})
	}
	rows := make([][]any, len(results))
	for index, result := range results {
		rows[index] = []any{result.APIName, result.OntologyID, result.Action, result.Error}
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"api_name", "ontology_id", "action", "error"}, rows))
}
