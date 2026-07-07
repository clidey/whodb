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
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// BundleClient is the platform API surface needed to export and plan project bundles.
type BundleClient interface {
	ProjectSecrets(context.Context, string) ([]ProjectSecret, error)
	AIProviders(context.Context, string) ([]AIProvider, error)
	Datasets(context.Context, string) ([]Dataset, error)
	Ontologies(context.Context, string) ([]Ontology, error)
	Transforms(context.Context, string) ([]Transform, error)
	Functions(context.Context, string, []string) ([]Function, error)
	FolderContents(context.Context, string, string, []string) (*FolderContents, error)
	FilePreview(context.Context, string, string, *int, []string) (*FilePreviewResult, error)
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
	AIProviders   []AIProvider    `json:"aiProviders"`
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
	SourceID string         `json:"sourceId,omitempty"`
	TargetID string         `json:"targetId,omitempty"`
	Action   string         `json:"action"`
	Reason   string         `json:"reason,omitempty"`
	Impacts  []string       `json:"impacts,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

// BundlePlan describes how a bundle would apply to a selected project.
type BundlePlan struct {
	Host              string         `json:"host"`
	SourceProjectID   string         `json:"sourceProjectId,omitempty"`
	SourceProjectName string         `json:"sourceProjectName,omitempty"`
	TargetProjectID   string         `json:"targetProjectId"`
	TargetProjectName string         `json:"targetProjectName"`
	DryRun            bool           `json:"dryRun"`
	Actions           []BundleAction `json:"actions"`
}

// BundleImportOptions controls how a project bundle is planned.
type BundleImportOptions struct {
	DryRun             bool
	Prefix             string
	RenameConflicts    bool
	OverwriteConflicts bool
	Getenv             func(string) string
}

// BundleExportOptions controls project bundle export behavior.
type BundleExportOptions struct {
	IncludeFiles bool
	MaxFileBytes int
}

type bundleTreeEntry struct {
	Folder ProjectFolder
	File   ProjectFile
	Kind   string
}

// BuildProjectBundle exports project metadata into a portable bundle.
func BuildProjectBundle(ctx context.Context, client BundleClient, host, orgID, orgName string, project *Project) (*ProjectBundle, error) {
	return BuildProjectBundleWithOptions(ctx, client, host, orgID, orgName, project, BundleExportOptions{})
}

// BuildProjectBundleWithOptions exports project metadata into a portable bundle.
func BuildProjectBundleWithOptions(ctx context.Context, client BundleClient, host, orgID, orgName string, project *Project, options BundleExportOptions) (*ProjectBundle, error) {
	secrets, err := client.ProjectSecrets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	aiProviders, err := client.AIProviders(ctx, project.ID)
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
			file := entry.File
			if options.IncludeFiles {
				file = exportProjectFileContent(ctx, client, project.ID, file, options.MaxFileBytes)
			}
			files = append(files, file)
		}
	}
	notes := []string{
		"Secret values are not exported. Import reads secret values from WHODB_IMPORT_SECRET_<SECRET_NAME> environment variables.",
		"AI provider API keys are not exported. Import reads provider keys from WHODB_IMPORT_AI_PROVIDER_KEY_<PROVIDER_NAME> environment variables.",
	}
	if options.IncludeFiles {
		notes = append(notes, "Previewable uploaded file content is included up to the configured size cap. Imported files are uploaded into the target project root.")
	} else {
		notes = append(notes, "Uploaded file bytes are not exported. File metadata is included for planning only.")
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
		AIProviders:   aiProviders,
		Datasets:      datasets,
		Ontologies:    ontologies,
		Transforms:    transforms,
		Functions:     functions,
		Folders:       folders,
		Files:         files,
		Notes:         notes,
	}, nil
}

// PlanBundleImport returns create/skip actions for applying a bundle to a project.
func PlanBundleImport(ctx context.Context, client BundleClient, host string, project *Project, bundle *ProjectBundle, dryRun bool, getenv func(string) string) (*BundlePlan, error) {
	return PlanBundleImportWithOptions(ctx, client, host, project, bundle, BundleImportOptions{DryRun: dryRun, Getenv: getenv})
}

// PlanBundleImportWithOptions returns actions for applying a bundle to a project.
func PlanBundleImportWithOptions(ctx context.Context, client BundleClient, host string, project *Project, bundle *ProjectBundle, options BundleImportOptions) (*BundlePlan, error) {
	if options.RenameConflicts && options.OverwriteConflicts {
		return nil, fmt.Errorf("rename conflicts and overwrite conflicts cannot both be enabled")
	}
	currentSecrets, err := client.ProjectSecrets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	currentAIProviders, err := client.AIProviders(ctx, project.ID)
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
	currentTree, err := loadBundleFolderTree(ctx, client, project.ID)
	if err != nil {
		return nil, err
	}

	plan := &BundlePlan{
		Host:              host,
		SourceProjectID:   bundle.ProjectID,
		SourceProjectName: bundle.ProjectName,
		TargetProjectID:   project.ID,
		TargetProjectName: project.Name,
		DryRun:            options.DryRun,
	}
	secretNames := resourceNamesBy(currentSecrets, func(secret ProjectSecret) string { return secret.Name })
	aiProviderNames := resourceNamesBy(currentAIProviders, func(provider AIProvider) string { return provider.Name })
	datasetNames := resourceNamesBy(currentDatasets, func(dataset Dataset) string { return dataset.Name })
	ontologyNames := resourceNamesBy(currentOntologies, func(ontology Ontology) string { return ontology.APIName })
	transformNames := resourceNamesBy(currentTransforms, func(transform Transform) string { return transform.Name })
	functionNames := resourceNamesBy(currentFunctions, func(fn Function) string { return fn.Name })
	fileNames := resourceNames{}
	for _, entry := range currentTree {
		if entry.Kind == "file" {
			fileNames.add(entry.File.Name, entry.File.ID)
		}
	}
	if options.Getenv == nil {
		options.Getenv = func(string) string { return "" }
	}

	for _, secret := range bundle.Secrets {
		name := bundleTargetName(secret.Name, options.Prefix)
		action := BundleAction{Resource: "secret", SourceID: secret.ID, Name: name}
		envName := ImportSecretEnvName(secret.Name)
		if existing, ok := secretNames.lookup(name); ok && !options.RenameConflicts && !options.OverwriteConflicts {
			action.Action = "skip"
			action.TargetID = existing
			action.Reason = "secret already exists"
			action.Impacts = append(action.Impacts, "Existing secret will be reused by dependent imported resources.")
		} else if existing, ok := secretNames.lookup(name); ok && options.OverwriteConflicts {
			action.Action = "update"
			action.TargetID = existing
			action.Payload = map[string]any{"name": name, "description": secret.Description}
			if value := options.Getenv(envName); value != "" {
				action.Payload["value"] = value
				action.Impacts = append(action.Impacts, "Secret value will be replaced from "+envName+".")
			} else {
				action.Impacts = append(action.Impacts, "Secret metadata will be updated; value is unchanged because "+envName+" is not set.")
			}
		} else if options.Getenv(envName) == "" {
			action.Action = "skip"
			action.Reason = "missing " + envName
			action.Impacts = append(action.Impacts, "Set "+envName+" to import this secret and bind dependent resources to it.")
		} else {
			if options.RenameConflicts {
				name = uniqueResourceName(name, secretNames)
				action.Name = name
				action.Impacts = append(action.Impacts, "Secret will be renamed to avoid a target-project conflict.")
			}
			action.Action = "create"
			action.Payload = map[string]any{"name": name, "description": secret.Description, "value": options.Getenv(envName)}
			action.Impacts = append(action.Impacts, "Secret value will be read from "+envName+".")
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, provider := range bundle.AIProviders {
		name := bundleTargetName(provider.Name, options.Prefix)
		action := BundleAction{Resource: "ai_provider", SourceID: provider.ID, Name: name}
		envName := ImportAIProviderKeyEnvName(provider.Name)
		payload := AIProviderCreatePayloadFromExport(provider)
		payload["name"] = name
		if existing, ok := aiProviderNames.lookup(name); ok && !options.RenameConflicts && !options.OverwriteConflicts {
			action.Action = "skip"
			action.TargetID = existing
			action.Reason = "AI provider already exists"
			action.Impacts = append(action.Impacts, "Existing AI provider will be reused by dependent imported functions.")
		} else if existing, ok := aiProviderNames.lookup(name); ok && options.OverwriteConflicts {
			action.Action = "update"
			action.TargetID = existing
			action.Payload = payload
			if value := options.Getenv(envName); value != "" {
				action.Payload["apiKey"] = value
				action.Impacts = append(action.Impacts, "AI provider key will be replaced from "+envName+".")
			} else {
				action.Impacts = append(action.Impacts, "AI provider metadata will be updated; key is unchanged because "+envName+" is not set.")
			}
		} else if options.Getenv(envName) == "" {
			action.Action = "skip"
			action.Reason = "missing " + envName
			action.Impacts = append(action.Impacts, "Set "+envName+" to import this AI provider and bind dependent functions to it.")
		} else {
			if options.RenameConflicts {
				name = uniqueResourceName(name, aiProviderNames)
				action.Name = name
				payload["name"] = name
				action.Impacts = append(action.Impacts, "AI provider will be renamed to avoid a target-project conflict.")
			}
			action.Action = "create"
			payload["apiKey"] = options.Getenv(envName)
			action.Payload = payload
			action.Impacts = append(action.Impacts, "AI provider key will be read from "+envName+".")
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, dataset := range bundle.Datasets {
		name := bundleTargetName(dataset.Name, options.Prefix)
		action := BundleAction{Resource: "dataset", SourceID: dataset.ID, Name: name}
		payload := DatasetCreatePayloadFromExport(dataset)
		payload["name"] = name
		if existing, ok := datasetNames.lookup(name); ok && !options.RenameConflicts && !options.OverwriteConflicts {
			action.Action = "skip"
			action.TargetID = existing
			action.Reason = "dataset already exists"
			action.Impacts = append(action.Impacts, "Existing dataset will be reused by dependent imported resources.")
		} else if existing, ok := datasetNames.lookup(name); ok && options.OverwriteConflicts {
			action.Action = "update"
			action.TargetID = existing
			action.Payload = payload
			action.Impacts = append(action.Impacts, "Dataset metadata/schema will be updated in place.")
		} else {
			if options.RenameConflicts {
				name = uniqueResourceName(name, datasetNames)
				action.Name = name
				payload["name"] = name
				action.Impacts = append(action.Impacts, "Dataset will be renamed to avoid a target-project conflict.")
			}
			action.Action = "create"
			action.Payload = payload
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, ontology := range bundle.Ontologies {
		name := SafeIdentifier(bundleTargetName(ontology.APIName, options.Prefix))
		action := BundleAction{Resource: "ontology", SourceID: ontology.ID, Name: name}
		payload := OntologyCreatePayloadFromExport(ontology)
		payload["apiName"] = name
		if existing, ok := ontologyNames.lookup(name); ok && !options.RenameConflicts && !options.OverwriteConflicts {
			action.Action = "skip"
			action.TargetID = existing
			action.Reason = "ontology already exists"
			action.Impacts = append(action.Impacts, "Existing ontology will be reused by dependent imported functions.")
		} else if existing, ok := ontologyNames.lookup(name); ok && options.OverwriteConflicts {
			action.Action = "update"
			action.TargetID = existing
			action.Payload = payload
			action.Impacts = append(action.Impacts, "Ontology metadata/properties will be updated in place.")
		} else {
			if options.RenameConflicts {
				name = uniqueResourceName(name, ontologyNames)
				action.Name = name
				payload["apiName"] = name
				payload["tableName"] = name
				action.Impacts = append(action.Impacts, "Ontology API name will be renamed to avoid a target-project conflict.")
			}
			action.Action = "create"
			action.Payload = payload
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, transform := range bundle.Transforms {
		name := bundleTargetName(transform.Name, options.Prefix)
		action := BundleAction{Resource: "transform", SourceID: transform.ID, Name: name}
		payload := TransformCreatePayloadFromExport(transform)
		payload["name"] = name
		if existing, ok := transformNames.lookup(name); ok && !options.RenameConflicts && !options.OverwriteConflicts {
			action.Action = "skip"
			action.TargetID = existing
			action.Reason = "transform already exists"
			action.Impacts = append(action.Impacts, "Existing transform will be reused where referenced.")
		} else if existing, ok := transformNames.lookup(name); ok && options.OverwriteConflicts {
			action.Action = "update"
			action.TargetID = existing
			action.Payload = payload
			action.Impacts = append(action.Impacts, "Transform graph/configuration will be updated in place.")
		} else {
			if options.RenameConflicts {
				name = uniqueResourceName(name, transformNames)
				action.Name = name
				payload["name"] = name
				action.Impacts = append(action.Impacts, "Transform will be renamed to avoid a target-project conflict.")
			}
			action.Action = "create"
			action.Payload = payload
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, fn := range bundle.Functions {
		name := bundleTargetName(fn.Name, options.Prefix)
		action := BundleAction{Resource: "function", SourceID: fn.ID, Name: name}
		payload := FunctionCreatePayloadFromExport(fn, true)
		payload["name"] = name
		if existing, ok := functionNames.lookup(name); ok && !options.RenameConflicts && !options.OverwriteConflicts {
			action.Action = "skip"
			action.TargetID = existing
			action.Reason = "function already exists"
			action.Impacts = append(action.Impacts, "Existing function will remain unchanged.")
		} else if existing, ok := functionNames.lookup(name); ok && options.OverwriteConflicts {
			action.Action = "update"
			action.TargetID = existing
			action.Payload = payload
			action.Impacts = append(action.Impacts, bundleFunctionImpacts(fn)...)
			action.Impacts = append(action.Impacts, "Function code/configuration will be updated in place.")
		} else {
			if options.RenameConflicts {
				name = uniqueResourceName(name, functionNames)
				action.Name = name
				payload["name"] = name
				action.Impacts = append(action.Impacts, "Function will be renamed to avoid a target-project conflict.")
			}
			action.Action = "create"
			action.Payload = payload
			action.Impacts = append(action.Impacts, bundleFunctionImpacts(fn)...)
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, folder := range bundle.Folders {
		plan.Actions = append(plan.Actions, BundleAction{Resource: "folder", Name: folder.Name, Action: "skip", Reason: "folder metadata import is not implemented in v1"})
	}
	for _, file := range bundle.Files {
		name := bundleTargetName(file.Name, options.Prefix)
		action := BundleAction{Resource: "file", SourceID: file.ID, Name: name}
		if strings.TrimSpace(file.Content) == "" {
			action.Action = "skip"
			action.Reason = "file bytes are not included in bundle"
		} else if existing, ok := fileNames.lookup(name); ok && !options.RenameConflicts {
			action.Action = "skip"
			action.TargetID = existing
			action.Reason = "file already exists"
			if options.OverwriteConflicts {
				action.Impacts = append(action.Impacts, "File overwrite is not supported by bundle import; existing file will remain unchanged.")
			} else {
				action.Impacts = append(action.Impacts, "Existing file will remain unchanged.")
			}
		} else {
			if options.RenameConflicts {
				name = uniqueResourceName(name, fileNames)
				action.Name = name
				action.Impacts = append(action.Impacts, "File will be renamed to avoid a target-project conflict.")
			}
			action.Action = "create"
			action.Payload = map[string]any{"name": name, "content": file.Content}
			action.Impacts = append(action.Impacts, "File content will be uploaded into the target project root.")
			if file.Truncated {
				action.Impacts = append(action.Impacts, "Exported file content was truncated by the bundle size cap.")
			}
		}
		plan.Actions = append(plan.Actions, action)
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

// AIProviderCreatePayloadFromExport converts exported AI provider metadata to a create payload.
func AIProviderCreatePayloadFromExport(provider AIProvider) map[string]any {
	return map[string]any{
		"name":         provider.Name,
		"providerType": provider.ProviderType,
		"endpoint":     provider.Endpoint,
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
	return importEnvName("WHODB_IMPORT_SECRET_", name)
}

// ImportAIProviderKeyEnvName returns the environment variable used for importing an AI provider API key.
func ImportAIProviderKeyEnvName(name string) string {
	return importEnvName("WHODB_IMPORT_AI_PROVIDER_KEY_", name)
}

func importEnvName(prefix, name string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9]+`)
	sanitized := strings.Trim(re.ReplaceAllString(strings.ToUpper(name), "_"), "_")
	if sanitized == "" {
		sanitized = "VALUE"
	}
	return prefix + sanitized
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

// BundleDependencyMap maps exported resource ids to target resource ids during import.
type BundleDependencyMap map[string]string

// ApplyBundleDependencyMap rewrites project-local ids in a bundle action payload.
func ApplyBundleDependencyMap(action *BundleAction, dependencies BundleDependencyMap) {
	if action == nil || len(dependencies) == 0 || len(action.Payload) == 0 {
		return
	}
	switch action.Resource {
	case "transform":
		if graphJSON, ok := action.Payload["graphJson"].(string); ok && graphJSON != "" {
			action.Payload["graphJson"] = replaceKnownIDs(graphJSON, dependencies)
		}
	case "function":
		action.Payload["providerIds"] = remapStringSlice(action.Payload["providerIds"], dependencies)
		action.Payload["ontologyIds"] = remapStringSlice(action.Payload["ontologyIds"], dependencies)
		action.Payload["readOnlyOntologyIds"] = remapStringSlice(action.Payload["readOnlyOntologyIds"], dependencies)
		action.Payload["providerConfigs"] = remapProviderConfigs(action.Payload["providerConfigs"], dependencies)
		action.Payload["secretBindings"] = remapSecretBindings(action.Payload["secretBindings"], dependencies)
	}
}

// AddBundleDependencyMapping records a source to target id mapping.
func AddBundleDependencyMapping(dependencies BundleDependencyMap, sourceID, targetID string) {
	sourceID = strings.TrimSpace(sourceID)
	targetID = strings.TrimSpace(targetID)
	if sourceID != "" && targetID != "" {
		dependencies[sourceID] = targetID
	}
}

func replaceKnownIDs(value string, dependencies BundleDependencyMap) string {
	for sourceID, targetID := range dependencies {
		value = strings.ReplaceAll(value, sourceID, targetID)
	}
	return value
}

func remapStringSlice(value any, dependencies BundleDependencyMap) []string {
	values := anyStringSlice(value)
	out := make([]string, 0, len(values))
	for _, item := range values {
		if targetID := dependencies[item]; targetID != "" {
			out = append(out, targetID)
		}
	}
	return out
}

func remapProviderConfigs(value any, dependencies BundleDependencyMap) []map[string]any {
	items := anyMapSlice(value)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		providerID, _ := item["providerId"].(string)
		targetID := dependencies[providerID]
		if targetID == "" {
			continue
		}
		next := copyMap(item)
		next["providerId"] = targetID
		out = append(out, next)
	}
	return out
}

func remapSecretBindings(value any, dependencies BundleDependencyMap) []map[string]any {
	items := anyMapSlice(value)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		secretID, _ := item["secretId"].(string)
		targetID := dependencies[secretID]
		if targetID == "" {
			continue
		}
		next := copyMap(item)
		next["secretId"] = targetID
		out = append(out, next)
	}
	return out
}

func anyStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func anyMapSlice(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []FunctionProviderConfig:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, map[string]any{"providerId": item.ProviderID, "model": item.Model})
		}
		return out
	case []FunctionSecretBinding:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, map[string]any{"name": item.Name, "secretId": item.SecretID, "target": item.Target})
		}
		return out
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if mapped, ok := item.(map[string]any); ok {
				out = append(out, mapped)
			}
		}
		return out
	default:
		return nil
	}
}

func copyMap(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

type resourceNames map[string]string

func resourceNamesBy[T any](values []T, name func(T) string) resourceNames {
	names := resourceNames{}
	for _, value := range values {
		id, resourceName := namedResourceIdentity(value, name)
		if resourceName != "" {
			names[strings.ToLower(resourceName)] = id
		}
	}
	return names
}

func namedResourceIdentity[T any](value T, name func(T) string) (string, string) {
	resourceName := strings.TrimSpace(name(value))
	switch typed := any(value).(type) {
	case ProjectSecret:
		return typed.ID, resourceName
	case AIProvider:
		return typed.ID, resourceName
	case Dataset:
		return typed.ID, resourceName
	case Ontology:
		return typed.ID, resourceName
	case Transform:
		return typed.ID, resourceName
	case Function:
		return typed.ID, resourceName
	default:
		return "", resourceName
	}
}

func (names resourceNames) lookup(name string) (string, bool) {
	id, ok := names[strings.ToLower(strings.TrimSpace(name))]
	return id, ok
}

func (names resourceNames) add(name, id string) {
	name = strings.TrimSpace(name)
	if name != "" {
		names[strings.ToLower(name)] = id
	}
}

func bundleTargetName(name, prefix string) string {
	name = strings.TrimSpace(name)
	if prefix = strings.TrimSpace(prefix); prefix != "" {
		return prefix + name
	}
	return name
}

func uniqueResourceName(base string, names resourceNames) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "imported"
	}
	if _, ok := names.lookup(base); !ok {
		names.add(base, "")
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := names.lookup(candidate); !ok {
			names.add(candidate, "")
			return candidate
		}
	}
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

func exportProjectFileContent(ctx context.Context, client BundleClient, projectID string, file ProjectFile, maxBytes int) ProjectFile {
	preview, err := client.FilePreview(ctx, projectID, file.ID, nil, nil)
	if err != nil {
		file.Truncated = true
		return file
	}
	content, contentType, ok := bundleFilePreviewContent(preview)
	if !ok {
		return file
	}
	if maxBytes > 0 && len([]byte(content)) > maxBytes {
		content = string([]byte(content)[:maxBytes])
		file.Truncated = true
	}
	file.Content = content
	file.ContentType = contentType
	return file
}

func bundleFilePreviewContent(preview *FilePreviewResult) (string, string, bool) {
	if preview == nil {
		return "", "", false
	}
	if preview.TextContent != nil {
		return *preview.TextContent, "text", true
	}
	if preview.Tabular == nil {
		return "", "", false
	}
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	headers := make([]string, len(preview.Tabular.Columns))
	for i, column := range preview.Tabular.Columns {
		headers[i] = column.Name
	}
	if err := writer.Write(headers); err != nil {
		return "", "", false
	}
	for _, row := range preview.Tabular.Rows {
		if err := writer.Write(row); err != nil {
			return "", "", false
		}
	}
	writer.Flush()
	if writer.Error() != nil {
		return "", "", false
	}
	return buffer.String(), "csv", true
}

func bundleFunctionImpacts(fn Function) []string {
	var impacts []string
	if len(fn.ProviderIDs) > 0 || len(fn.ProviderConfigs) > 0 {
		impacts = append(impacts, "Function AI provider references will be remapped when matching imported or existing providers are available.")
	}
	if len(fn.OntologyIDs) > 0 || len(fn.ReadOnlyOntologyIDs) > 0 {
		impacts = append(impacts, "Function ontology references will be remapped when matching imported or existing ontologies are available.")
	}
	if len(fn.SecretBindings) > 0 {
		impacts = append(impacts, "Function secret bindings will be remapped when matching imported or existing secrets are available.")
	}
	return impacts
}
