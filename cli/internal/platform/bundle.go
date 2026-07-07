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

package platform

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// BundleClient is the platform API surface needed to export and plan project bundles.
type BundleClient interface {
	ProjectSecrets(context.Context, string) ([]ProjectSecret, error)
	Datasets(context.Context, string) ([]Dataset, error)
	Ontologies(context.Context, string) ([]Ontology, error)
	Transforms(context.Context, string) ([]Transform, error)
	Functions(context.Context, string, []string) ([]Function, error)
	FolderContents(context.Context, string, string, []string) (*FolderContents, error)
}

// CloneClient is the platform API surface needed to clone project resources.
type CloneClient interface {
	Datasets(context.Context, string) ([]Dataset, error)
	Dataset(context.Context, string, string) (*Dataset, error)
	Ontologies(context.Context, string) ([]Ontology, error)
	Ontology(context.Context, string, string) (*Ontology, error)
	Transforms(context.Context, string) ([]Transform, error)
	Functions(context.Context, string, []string) ([]Function, error)
	Function(context.Context, string, string, []string) (*Function, error)
}

// ProjectBundle is a portable metadata bundle for one hosted WhoDB project.
type ProjectBundle struct {
	BundleVersion int             `json:"bundleVersion"`
	ExportedAt    string          `json:"exportedAt"`
	Host          string          `json:"host"`
	OrgID         string          `json:"orgId"`
	OrgName       string          `json:"orgName"`
	ProjectID     string          `json:"projectId"`
	ProjectName   string          `json:"projectName"`
	Secrets       []ProjectSecret `json:"secrets"`
	Datasets      []Dataset       `json:"datasets"`
	Ontologies    []Ontology      `json:"ontologies"`
	Transforms    []Transform     `json:"transforms"`
	Functions     []Function      `json:"functions"`
	Folders       []ProjectFolder `json:"folders"`
	Files         []ProjectFile   `json:"files"`
	Notes         []string        `json:"notes,omitempty"`
}

// BundleAction describes one import-plan action for a project bundle item.
type BundleAction struct {
	Resource string         `json:"resource"`
	Name     string         `json:"name"`
	Action   string         `json:"action"`
	Reason   string         `json:"reason,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

// BundlePlan describes how a bundle would apply to a selected project.
type BundlePlan struct {
	Host        string         `json:"host"`
	ProjectID   string         `json:"projectId"`
	ProjectName string         `json:"projectName"`
	DryRun      bool           `json:"dryRun"`
	Actions     []BundleAction `json:"actions"`
}

type bundleTreeEntry struct {
	Folder ProjectFolder
	File   ProjectFile
	Kind   string
}

// BuildProjectBundle exports project metadata into a portable bundle.
func BuildProjectBundle(ctx context.Context, client BundleClient, host, orgID, orgName string, project *Project) (*ProjectBundle, error) {
	secrets, err := client.ProjectSecrets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	datasets, err := client.Datasets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	ontologies, err := client.Ontologies(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	transforms, err := client.Transforms(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	functions, err := client.Functions(ctx, project.ID, nil)
	if err != nil {
		return nil, err
	}
	tree, err := loadBundleFolderTree(ctx, client, project.ID)
	if err != nil {
		return nil, err
	}
	folders := make([]ProjectFolder, 0)
	files := make([]ProjectFile, 0)
	for _, entry := range tree {
		switch entry.Kind {
		case "folder":
			folders = append(folders, entry.Folder)
		case "file":
			files = append(files, entry.File)
		}
	}
	return &ProjectBundle{
		BundleVersion: 1,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Host:          host,
		OrgID:         orgID,
		OrgName:       orgName,
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		Secrets:       secrets,
		Datasets:      datasets,
		Ontologies:    ontologies,
		Transforms:    transforms,
		Functions:     functions,
		Folders:       folders,
		Files:         files,
		Notes: []string{
			"Secret values are not exported. Import reads secret values from WHODB_IMPORT_SECRET_<SECRET_NAME> environment variables.",
			"Uploaded file bytes are not exported. File metadata is included for planning only.",
		},
	}, nil
}

// PlanBundleImport returns create/skip actions for applying a bundle to a project.
func PlanBundleImport(ctx context.Context, client BundleClient, host string, project *Project, bundle *ProjectBundle, dryRun bool, getenv func(string) string) (*BundlePlan, error) {
	currentSecrets, err := client.ProjectSecrets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	currentDatasets, err := client.Datasets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	currentOntologies, err := client.Ontologies(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	currentTransforms, err := client.Transforms(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	currentFunctions, err := client.Functions(ctx, project.ID, []string{"id", "name"})
	if err != nil {
		return nil, err
	}

	plan := &BundlePlan{Host: host, ProjectID: project.ID, ProjectName: project.Name, DryRun: dryRun}
	secretNames := NamesBy(currentSecrets, func(secret ProjectSecret) string { return secret.Name })
	datasetNames := NamesBy(currentDatasets, func(dataset Dataset) string { return dataset.Name })
	ontologyNames := NamesBy(currentOntologies, func(ontology Ontology) string { return ontology.APIName })
	transformNames := NamesBy(currentTransforms, func(transform Transform) string { return transform.Name })
	functionNames := NamesBy(currentFunctions, func(fn Function) string { return fn.Name })
	if getenv == nil {
		getenv = func(string) string { return "" }
	}

	for _, secret := range bundle.Secrets {
		action := BundleAction{Resource: "secret", Name: secret.Name}
		envName := ImportSecretEnvName(secret.Name)
		if secretNames[secret.Name] {
			action.Action = "skip"
			action.Reason = "secret already exists"
		} else if getenv(envName) == "" {
			action.Action = "skip"
			action.Reason = "missing " + envName
		} else {
			action.Action = "create"
			action.Payload = map[string]any{"name": secret.Name, "description": secret.Description, "value": getenv(envName)}
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, dataset := range bundle.Datasets {
		action := BundleAction{Resource: "dataset", Name: dataset.Name}
		if datasetNames[dataset.Name] {
			action.Action = "skip"
			action.Reason = "dataset already exists"
		} else {
			action.Action = "create"
			action.Payload = DatasetCreatePayloadFromExport(dataset)
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, ontology := range bundle.Ontologies {
		action := BundleAction{Resource: "ontology", Name: ontology.APIName}
		if ontologyNames[ontology.APIName] {
			action.Action = "skip"
			action.Reason = "ontology already exists"
		} else {
			action.Action = "create"
			action.Payload = OntologyCreatePayloadFromExport(ontology)
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, transform := range bundle.Transforms {
		action := BundleAction{Resource: "transform", Name: transform.Name}
		if transformNames[transform.Name] {
			action.Action = "skip"
			action.Reason = "transform already exists"
		} else {
			action.Action = "create"
			action.Payload = TransformCreatePayloadFromExport(transform)
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, fn := range bundle.Functions {
		action := BundleAction{Resource: "function", Name: fn.Name}
		if functionNames[fn.Name] {
			action.Action = "skip"
			action.Reason = "function already exists"
		} else {
			action.Action = "create"
			action.Payload = FunctionCreatePayloadFromExport(fn, false)
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, folder := range bundle.Folders {
		plan.Actions = append(plan.Actions, BundleAction{Resource: "folder", Name: folder.Name, Action: "skip", Reason: "folder metadata import is not implemented in v1"})
	}
	for _, file := range bundle.Files {
		plan.Actions = append(plan.Actions, BundleAction{Resource: "file", Name: file.Name, Action: "skip", Reason: "file bytes are not included in bundles"})
	}
	return plan, nil
}

// BuildClonePayload returns a create payload for cloning one resource in the same project.
func BuildClonePayload(ctx context.Context, client CloneClient, projectID, resource, sourceRef, newName string) (map[string]any, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return nil, fmt.Errorf("new name cannot be empty")
	}
	switch resource {
	case "dataset":
		id, err := ResolveResourceID(ctx, client, projectID, "dataset", sourceRef)
		if err != nil {
			return nil, err
		}
		dataset, err := client.Dataset(ctx, projectID, id)
		if err != nil {
			return nil, err
		}
		payload := DatasetCreatePayloadFromExport(*dataset)
		payload["name"] = newName
		return payload, nil
	case "ontology":
		id, err := ResolveResourceID(ctx, client, projectID, "ontology", sourceRef)
		if err != nil {
			return nil, err
		}
		ontology, err := client.Ontology(ctx, projectID, id)
		if err != nil {
			return nil, err
		}
		payload := OntologyCreatePayloadFromExport(*ontology)
		identifier := SafeIdentifier(newName)
		payload["apiName"] = identifier
		payload["displayName"] = newName
		payload["pluralDisplayName"] = newName + "s"
		payload["tableName"] = identifier
		return payload, nil
	case "transform":
		transform, err := ResolveTransform(ctx, client, projectID, sourceRef)
		if err != nil {
			return nil, err
		}
		payload := TransformCreatePayloadFromExport(*transform)
		payload["name"] = newName
		return payload, nil
	case "function":
		id, err := ResolveResourceID(ctx, client, projectID, "function", sourceRef)
		if err != nil {
			return nil, err
		}
		fn, err := client.Function(ctx, projectID, id, nil)
		if err != nil {
			return nil, err
		}
		payload := FunctionCreatePayloadFromExport(*fn, true)
		payload["name"] = newName
		return payload, nil
	default:
		return nil, fmt.Errorf("unsupported clone resource %q", resource)
	}
}

// DatasetCreatePayloadFromExport converts exported dataset metadata to a create payload.
func DatasetCreatePayloadFromExport(dataset Dataset) map[string]any {
	payload := map[string]any{"name": dataset.Name}
	if dataset.Description != "" {
		payload["description"] = dataset.Description
	}
	if dataset.SchemaMode != "" {
		payload["schemaMode"] = dataset.SchemaMode
	}
	if len(dataset.Schema) > 0 {
		columns := make([]map[string]any, len(dataset.Schema))
		for i, column := range dataset.Schema {
			columns[i] = map[string]any{
				"name":       column.Name,
				"type":       column.Type,
				"isNullable": column.IsNullable,
				"isPrimary":  column.IsPrimary,
			}
		}
		payload["columns"] = columns
	}
	return payload
}

// OntologyCreatePayloadFromExport converts exported ontology metadata to a create payload.
func OntologyCreatePayloadFromExport(ontology Ontology) map[string]any {
	properties := make([]map[string]any, len(ontology.Properties))
	for i, property := range ontology.Properties {
		properties[i] = map[string]any{
			"apiName":          property.APIName,
			"displayName":      property.DisplayName,
			"description":      property.Description,
			"columnName":       property.ColumnName,
			"dataType":         property.DataType,
			"arrayElementType": property.ArrayElementType,
			"isRequired":       property.IsRequired,
			"visibility":       property.Visibility,
			"isSearchable":     property.IsSearchable,
			"isSortable":       property.IsSortable,
			"isEditOnly":       property.IsEditOnly,
		}
	}
	links := make([]map[string]any, len(ontology.Links))
	for i, link := range ontology.Links {
		links[i] = map[string]any{
			"apiName":                  link.APIName,
			"targetOntologyApiName":    link.TargetOntologyAPIName,
			"cardinality":              link.Cardinality,
			"foreignKeyProperty":       link.ForeignKeyProperty,
			"targetForeignKeyProperty": link.TargetForeignKeyProperty,
			"joinTable":                link.JoinTable,
			"sourceColumnInJoinTable":  link.SourceColumnInJoinTable,
			"targetColumnInJoinTable":  link.TargetColumnInJoinTable,
			"displayName":              link.DisplayName,
			"pluralDisplayName":        link.PluralDisplayName,
			"reverseDisplayName":       link.ReverseDisplayName,
		}
	}
	return map[string]any{
		"apiName":           ontology.APIName,
		"displayName":       ontology.DisplayName,
		"pluralDisplayName": ontology.PluralDisplayName,
		"description":       ontology.Description,
		"primaryKey":        ontology.PrimaryKey,
		"tableName":         ontology.TableName,
		"schemaName":        ontology.SchemaName,
		"icon":              DefaultString(ontology.Icon, "table"),
		"color":             DefaultString(ontology.Color, "#3366ff"),
		"properties":        properties,
		"links":             links,
	}
}

// TransformCreatePayloadFromExport converts exported transform metadata to a create payload.
func TransformCreatePayloadFromExport(transform Transform) map[string]any {
	return map[string]any{
		"name":         transform.Name,
		"description":  transform.Description,
		"graphJson":    DefaultString(transform.GraphJSON, `{"nodes":[],"edges":[]}`),
		"scheduleCron": transform.ScheduleCron,
		"triggerMode":  DefaultString(transform.TriggerMode, "manual"),
	}
}

// FunctionCreatePayloadFromExport converts exported function metadata to a create payload.
func FunctionCreatePayloadFromExport(fn Function, keepProjectReferences bool) map[string]any {
	files := make([]map[string]any, len(fn.Files))
	for i, file := range fn.Files {
		files[i] = map[string]any{"path": file.Path, "content": file.Content}
	}
	dependencies := make([]map[string]any, len(fn.Dependencies))
	for i, dep := range fn.Dependencies {
		dependencies[i] = map[string]any{"name": dep.Name, "version": dep.Version}
	}
	payload := map[string]any{
		"name":           fn.Name,
		"description":    fn.Description,
		"language":       fn.Language,
		"entryPoint":     fn.EntryPoint,
		"timeoutSeconds": fn.TimeoutSeconds,
		"memory":         DefaultString(fn.Memory, "128Mi"),
		"cpu":            DefaultString(fn.CPU, "100m"),
		"files":          files,
		"dependencies":   dependencies,
	}
	if fn.TimeoutSeconds <= 0 {
		payload["timeoutSeconds"] = 30
	}
	if keepProjectReferences {
		payload["providerIds"] = fn.ProviderIDs
		payload["ontologyIds"] = fn.OntologyIDs
		payload["readOnlyOntologyIds"] = fn.ReadOnlyOntologyIDs
		payload["providerConfigs"] = fn.ProviderConfigs
		payload["secretBindings"] = fn.SecretBindings
	}
	if fn.DefaultMaxTokens > 0 {
		payload["defaultMaxTokens"] = fn.DefaultMaxTokens
	}
	if fn.DefaultTemperature > 0 {
		payload["defaultTemperature"] = fn.DefaultTemperature
	}
	return payload
}

// ImportSecretEnvName returns the environment variable used for importing a secret value.
func ImportSecretEnvName(name string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9]+`)
	sanitized := strings.Trim(re.ReplaceAllString(strings.ToUpper(name), "_"), "_")
	if sanitized == "" {
		sanitized = "VALUE"
	}
	return "WHODB_IMPORT_SECRET_" + sanitized
}

// SafeIdentifier returns a lower-case API-safe identifier for cloned resources.
func SafeIdentifier(value string) string {
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

// DefaultString returns fallback when value is empty.
func DefaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// NamesBy indexes values by a derived non-empty name.
func NamesBy[T any](values []T, name func(T) string) map[string]bool {
	names := make(map[string]bool, len(values))
	for _, value := range values {
		if key := strings.TrimSpace(name(value)); key != "" {
			names[key] = true
		}
	}
	return names
}

// ResolveResourceID resolves a project resource by id or common display name.
func ResolveResourceID(ctx context.Context, client CloneClient, projectID, resource, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", resource)
	}
	switch resource {
	case "dataset":
		datasets, err := client.Datasets(ctx, projectID)
		if err != nil {
			return "", err
		}
		return resolveNamedResource(value, "dataset", datasets, func(dataset Dataset) (string, string) {
			return dataset.ID, dataset.Name
		})
	case "ontology":
		ontologies, err := client.Ontologies(ctx, projectID)
		if err != nil {
			return "", err
		}
		return resolveNamedResource(value, "ontology", ontologies, func(ontology Ontology) (string, string) {
			return ontology.ID, ontology.APIName
		})
	case "function":
		functions, err := client.Functions(ctx, projectID, []string{"id", "name"})
		if err != nil {
			return "", err
		}
		return resolveNamedResource(value, "function", functions, func(fn Function) (string, string) {
			return fn.ID, fn.Name
		})
	default:
		return "", fmt.Errorf("unsupported resource %q", resource)
	}
}

// ResolveTransform resolves a transform by id or name.
func ResolveTransform(ctx context.Context, client CloneClient, projectID, value string) (*Transform, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("transform is required")
	}
	transforms, err := client.Transforms(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, transform := range transforms {
		if transform.ID == value || strings.EqualFold(transform.Name, value) {
			found := transform
			return &found, nil
		}
	}
	return nil, fmt.Errorf("transform %q not found", value)
}

func resolveNamedResource[T any](value, resource string, values []T, identity func(T) (string, string)) (string, error) {
	for _, item := range values {
		id, name := identity(item)
		if id == value || strings.EqualFold(name, value) {
			return id, nil
		}
	}
	return "", fmt.Errorf("%s %q not found", resource, value)
}

func loadBundleFolderTree(ctx context.Context, client BundleClient, projectID string) ([]bundleTreeEntry, error) {
	var tree []bundleTreeEntry
	var walk func(string) error
	walk = func(folderID string) error {
		contents, err := client.FolderContents(ctx, projectID, folderID, nil)
		if err != nil {
			return err
		}
		if contents == nil {
			return nil
		}
		for _, folder := range contents.Folders {
			tree = append(tree, bundleTreeEntry{Kind: "folder", Folder: folder})
			if err := walk(folder.ID); err != nil {
				return err
			}
		}
		for _, file := range contents.Files {
			tree = append(tree, bundleTreeEntry{Kind: "file", File: file})
		}
		return nil
	}
	if err := walk(""); err != nil {
		return nil, err
	}
	return tree, nil
}
