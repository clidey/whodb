/*
 * Copyright 2026 Clidey, Inc.
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

type behaviorApplyManifest struct {
	Version   int                  `json:"version"`
	Behaviors []behaviorApplyEntry `json:"behaviors"`
}

type behaviorApplyEntry struct {
	APIName          string         `json:"api_name"`
	ExpectedRevision *int           `json:"expected_revision,omitempty"`
	Document         map[string]any `json:"document"`
}

var ontologyBehaviorsCmd = &cobra.Command{Use: "behaviors", Short: "Pull, validate, plan and atomically activate ontology behaviors"}

func registerOntologyBehaviorApplyCommand() {
	for _, name := range []string{"plan", "diff", "validate", "apply"} {
		command := &cobra.Command{Use: name, Args: cobra.NoArgs, SilenceUsage: true, SilenceErrors: true, RunE: runOntologyBehaviorsManifest}
		command.Short = "Validate the complete behavior manifest without activation"
		if name == "apply" {
			command.Short = "Atomically activate a behavior manifest; without --yes, preview only"
		}
		command.Flags().String("manifest", "", "portable behavior manifest JSON file")
		if name == "apply" {
			command.Flags().BoolP("yes", "y", false, "activate all changed definitions atomically (otherwise preview)")
		}
		ontologyBehaviorsCmd.AddCommand(command)
	}
	pull := &cobra.Command{Use: "pull [ontology...]", Short: "Export portable behavior definitions; omit names for all configured ontologies", RunE: runOntologyBehaviorsPull, SilenceUsage: true, SilenceErrors: true}
	pull.Flags().String("output", "", "write manifest to a new file instead of stdout")
	ontologyBehaviorsCmd.AddCommand(pull)
	ontologiesCmd.AddCommand(ontologyBehaviorsCmd)
}

func decodeBehaviorManifest(raw []byte) (behaviorApplyManifest, error) {
	var manifest behaviorApplyManifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("decode behavior manifest: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return manifest, fmt.Errorf("behavior manifest must contain exactly one JSON object")
	}
	// Keep the original apply-manifest format readable.
	if manifest.Version == 0 {
		manifest.Version = 1
	}
	if manifest.Version != 1 {
		return manifest, fmt.Errorf("unsupported behavior manifest version %d", manifest.Version)
	}
	if len(manifest.Behaviors) == 0 || len(manifest.Behaviors) > 100 {
		return manifest, fmt.Errorf("manifest must contain between 1 and 100 behaviors")
	}
	names := map[string]bool{}
	for _, entry := range manifest.Behaviors {
		if strings.TrimSpace(entry.APIName) == "" || len(entry.Document) == 0 {
			return manifest, fmt.Errorf("each behavior requires api_name and document")
		}
		if names[entry.APIName] {
			return manifest, fmt.Errorf("duplicate behavior %q", entry.APIName)
		}
		names[entry.APIName] = true
		if entry.ExpectedRevision != nil && *entry.ExpectedRevision < 0 {
			return manifest, fmt.Errorf("%s: expected_revision cannot be negative", entry.APIName)
		}
	}
	return manifest, nil
}

func behaviorManifestMap(manifest behaviorApplyManifest) (map[string]any, error) {
	raw, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	err = decoder.Decode(&value)
	return value, err
}

type behaviorManifestClient interface {
	PlatformQuery(context.Context, string, map[string]any) (any, error)
	PlatformMutation(context.Context, string, map[string]any) (*platform.PlatformMutationResult, error)
}

func prepareBehaviorManifest(ctx context.Context, client behaviorManifestClient, projectID string, manifest *behaviorApplyManifest, ontologies []platform.Ontology) error {
	byName := map[string]platform.Ontology{}
	for _, entity := range ontologies {
		byName[entity.APIName] = entity
	}
	for index, entry := range manifest.Behaviors {
		entity, ok := byName[entry.APIName]
		if !ok {
			return fmt.Errorf("ontology %q not found; no behaviors applied", entry.APIName)
		}
		current, err := client.PlatformQuery(ctx, "OntologyBehavior", map[string]any{"projectId": projectID, "ontologyId": entity.ID})
		if err != nil {
			return err
		}
		revision := 0
		if current != nil {
			raw, err := json.Marshal(current)
			if err != nil {
				return err
			}
			var data struct {
				Revision int `json:"revision"`
			}
			if err := json.Unmarshal(raw, &data); err != nil {
				return err
			}
			revision = data.Revision
		}
		if entry.ExpectedRevision != nil && *entry.ExpectedRevision != revision {
			return fmt.Errorf("%s: revision conflict (expected %d, found %d)", entry.APIName, *entry.ExpectedRevision, revision)
		}
		manifest.Behaviors[index].ExpectedRevision = &revision
	}
	return nil
}

func executeBehaviorManifest(ctx context.Context, client behaviorManifestClient, projectID string, manifest behaviorApplyManifest, apply bool) (any, error) {
	encoded, err := behaviorManifestMap(manifest)
	if err != nil {
		return nil, err
	}
	variables := map[string]any{"projectId": projectID, "manifest": encoded}
	if !apply {
		return client.PlatformQuery(ctx, "ValidateOntologyBehaviors", variables)
	}
	// The server validates every definition and commits all of them together.
	result, err := client.PlatformMutation(ctx, "ApplyOntologyBehaviors", variables)
	if err != nil {
		return nil, err
	}
	var data any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func runOntologyBehaviorsManifest(cmd *cobra.Command, _ []string) error {
	file, _ := cmd.Flags().GetString("manifest")
	if strings.TrimSpace(file) == "" {
		return fmt.Errorf("--manifest is required")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	manifest, err := decodeBehaviorManifest(raw)
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
	entities, err := session.Client.Ontologies(ctx, project.ID)
	if err != nil {
		return err
	}
	if err := prepareBehaviorManifest(ctx, session.Client, project.ID, &manifest, entities); err != nil {
		return err
	}
	apply := false
	if cmd.Name() == "apply" {
		apply, _ = cmd.Flags().GetBool("yes")
	}
	result, err := executeBehaviorManifest(ctx, session.Client, project.ID, manifest, apply)
	if err != nil {
		return err
	}
	return writeBehaviorManifestResult(cmd, "ontologies.behaviors."+cmd.Name(), result)
}

func runOntologyBehaviorsPull(cmd *cobra.Command, args []string) error {
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
	entities, err := session.Client.Ontologies(ctx, project.ID)
	if err != nil {
		return err
	}
	selected := map[string]bool{}
	for _, name := range args {
		selected[name] = true
	}
	known := map[string]bool{}
	for _, entity := range entities {
		known[entity.APIName] = true
	}
	for name := range selected {
		if !known[name] {
			return fmt.Errorf("ontology %q not found", name)
		}
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].APIName < entities[j].APIName })
	manifest := behaviorApplyManifest{Version: 1, Behaviors: []behaviorApplyEntry{}}
	for _, entity := range entities {
		if len(selected) > 0 && !selected[entity.APIName] {
			continue
		}
		result, err := session.Client.PlatformQuery(ctx, "OntologyBehavior", map[string]any{"projectId": project.ID, "ontologyId": entity.ID})
		if err != nil {
			return err
		}
		if result == nil {
			if selected[entity.APIName] {
				return fmt.Errorf("ontology %q has no behavior", entity.APIName)
			}
			continue
		}
		raw, err := json.Marshal(result)
		if err != nil {
			return err
		}
		var data struct {
			Document map[string]any `json:"document"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return err
		}
		if data.Document == nil {
			return fmt.Errorf("%s: empty behavior document", entity.APIName)
		}
		delete(data.Document, "ontology_id")
		manifest.Behaviors = append(manifest.Behaviors, behaviorApplyEntry{APIName: entity.APIName, Document: data.Document})
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	file, _ := cmd.Flags().GetString("output")
	if file != "" {
		handle, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, writeErr := handle.Write(raw)
		closeErr := handle.Close()
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func writeBehaviorManifestResult(cmd *cobra.Command, command string, data any) error {
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, automationEnvelope{Command: command, Success: true, Data: data})
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
	return err
}
