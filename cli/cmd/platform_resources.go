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
	"io"
	"sort"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	platformResourceOrg     string
	platformResourceProject string
	platformFields          []string
	platformLimit           int
	platformOffset          int
	platformRootID          string
	platformRootType        string
	platformNodeID          string
	platformNodeType        string
	platformDirection       string
	platformMaxDepth        int
	platformFolderID        string
	platformEntityID        string
	platformPrimaryKey      string
	platformLinkAPIName     string
	platformSheetIndex      int
	platformPayloadJSON     string
	platformPayloadStdin    bool
	platformWriteYes        bool
)

var secretsCmd = &cobra.Command{Use: "secrets", Short: "Manage hosted WhoDB project secrets"}
var aiProvidersCmd = &cobra.Command{Use: "ai-providers", Short: "Manage hosted WhoDB AI providers"}
var ontologiesCmd = &cobra.Command{Use: "ontologies", Short: "Manage hosted WhoDB ontologies"}
var datasetsCmd = &cobra.Command{Use: "datasets", Short: "Manage hosted WhoDB datasets"}
var lineageCmd = &cobra.Command{Use: "lineage", Short: "Inspect hosted WhoDB lineage"}
var transformsCmd = &cobra.Command{Use: "transforms", Short: "Manage hosted WhoDB transforms"}
var functionsCmd = &cobra.Command{Use: "functions", Short: "Manage hosted WhoDB functions"}
var filesCmd = &cobra.Command{Use: "files", Short: "Manage hosted WhoDB project files"}
var resourcesCmd = &cobra.Command{Use: "resources", Short: "Run advanced hosted WhoDB platform resource writes"}

var secretsListCmd = platformProjectListCommand("list", "List hosted WhoDB secret metadata", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	secrets, err := session.Client.ProjectSecrets(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(secrets))
	for i, secret := range secrets {
		rows[i] = []any{secret.ID, secret.Name, secret.Description, len(secret.UsedBy), secret.UpdatedAt}
	}
	return secrets, tableResult([]string{"id", "name", "description", "used_by", "updated_at"}, rows), nil
})

var aiProvidersListCmd = platformProjectListCommand("list", "List hosted WhoDB AI providers", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	providers, err := session.Client.AIProviders(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(providers))
	for i, provider := range providers {
		rows[i] = []any{provider.ID, provider.Name, provider.ProviderType, provider.Endpoint, provider.UpdatedAt}
	}
	return providers, tableResult([]string{"id", "name", "type", "endpoint", "updated_at"}, rows), nil
})

var aiProviderModelsCmd = &cobra.Command{
	Use:           "models <provider>",
	Short:         "List hosted WhoDB AI provider models",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			models, err := session.Client.AIProviderModels(ctx, session.Host.DefaultProjectID, args[0])
			if err != nil {
				return nil, nil, err
			}
			rows := make([][]any, len(models))
			for i, model := range models {
				rows[i] = []any{model}
			}
			return models, tableResult([]string{"model"}, rows), nil
		})
	},
}

var ontologiesListCmd = platformProjectListCommand("list", "List hosted WhoDB ontologies", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	ontologies, err := session.Client.Ontologies(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(ontologies))
	for i, ontology := range ontologies {
		rows[i] = []any{ontology.ID, ontology.APIName, ontology.DisplayName, ontology.TableName, ontology.Status}
	}
	return ontologies, tableResult([]string{"id", "api_name", "display_name", "table", "status"}, rows), nil
})

var ontologyGetCmd = platformIDCommand("get <ontology>", "Show hosted WhoDB ontology details", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	ontology, err := session.Client.Ontology(ctx, session.Host.DefaultProjectID, id)
	if err != nil {
		return nil, nil, err
	}
	rows := [][]any{{"id", ontology.ID}, {"api_name", ontology.APIName}, {"display_name", ontology.DisplayName}, {"table", ontology.TableName}, {"status", ontology.Status}, {"properties", len(ontology.Properties)}, {"links", len(ontology.Links)}}
	return ontology, tableResult([]string{"field", "value"}, rows), nil
})

var ontologyFastLookupsCmd = platformIDCommand("fast-lookups <ontology>", "List hosted WhoDB ontology fast lookups", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	lookups, err := session.Client.OntologyFastLookups(ctx, session.Host.DefaultProjectID, id)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(lookups))
	for i, lookup := range lookups {
		rows[i] = []any{lookup.ID, lookup.DisplayName, strings.Join(lookup.Fields, ","), lookup.Status}
	}
	return lookups, tableResult([]string{"id", "display_name", "fields", "status"}, rows), nil
})

var ontologyFastLookupSuggestionsCmd = platformIDCommand("fast-lookup-suggestions <ontology>", "Suggest hosted WhoDB ontology fast lookups", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	suggestions, err := session.Client.OntologyFastLookupSuggestions(ctx, session.Host.DefaultProjectID, id)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(suggestions))
	for i, suggestion := range suggestions {
		rows[i] = []any{suggestion.DisplayName, strings.Join(suggestion.Fields, ","), suggestion.CanCreate, suggestion.Reason}
	}
	return suggestions, tableResult([]string{"display_name", "fields", "can_create", "reason"}, rows), nil
})

var ontologyRowsCmd = pagedIDRowsCommand("rows <ontology>", "Preview hosted WhoDB ontology rows", func(ctx context.Context, session *platformSession, id string) (*platform.DatasetQueryResult, error) {
	return session.Client.OntologyRows(ctx, session.Host.DefaultProjectID, id, platformLimit, platformOffset)
})

var ontologyFollowLinkCmd = &cobra.Command{
	Use:           "follow-link",
	Short:         "Follow a hosted WhoDB ontology link",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(platformEntityID) == "" || strings.TrimSpace(platformPrimaryKey) == "" || strings.TrimSpace(platformLinkAPIName) == "" {
			return fmt.Errorf("--entity-id, --primary-key, and --link-api-name are required")
		}
		return runPlatformProjectRowsRead(cmd, func(ctx context.Context, session *platformSession) (*platform.DatasetQueryResult, error) {
			return session.Client.OntologyFollowLink(ctx, session.Host.DefaultProjectID, platformEntityID, platformPrimaryKey, platformLinkAPIName, platformLimit, platformOffset)
		})
	},
}

var datasetsListCmd = platformProjectListCommand("list", "List hosted WhoDB datasets", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	datasets, err := session.Client.Datasets(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(datasets))
	for i, dataset := range datasets {
		rows[i] = []any{dataset.ID, dataset.Name, dataset.SchemaMode, dataset.RowCount, dataset.SizeBytes, dataset.UpdatedAt}
	}
	return datasets, tableResult([]string{"id", "name", "schema_mode", "rows", "bytes", "updated_at"}, rows), nil
})

var datasetGetCmd = platformIDCommand("get <dataset>", "Show hosted WhoDB dataset details", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	dataset, err := session.Client.Dataset(ctx, session.Host.DefaultProjectID, id)
	if err != nil {
		return nil, nil, err
	}
	rows := [][]any{{"id", dataset.ID}, {"name", dataset.Name}, {"schema_mode", dataset.SchemaMode}, {"row_count", dataset.RowCount}, {"size_bytes", dataset.SizeBytes}, {"columns", len(dataset.Schema)}}
	return dataset, tableResult([]string{"field", "value"}, rows), nil
})

var datasetRowsCmd = pagedIDRowsCommand("rows <dataset>", "Preview hosted WhoDB dataset rows", func(ctx context.Context, session *platformSession, id string) (*platform.DatasetQueryResult, error) {
	return session.Client.DatasetRows(ctx, session.Host.DefaultProjectID, id, platformLimit, platformOffset)
})

var lineageProjectCmd = platformProjectListCommand("project", "Show hosted WhoDB project lineage", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	graph, err := session.Client.ProjectLineage(ctx, session.Host.DefaultProjectID)
	return graph, lineageTable(graph), err
})

var lineageRootCmd = &cobra.Command{
	Use:           "root",
	Short:         "Show hosted WhoDB lineage from a root node",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(platformRootID) == "" || strings.TrimSpace(platformRootType) == "" {
			return fmt.Errorf("--root-id and --root-type are required")
		}
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			graph, err := session.Client.Lineage(ctx, session.Host.DefaultProjectID, platformRootID, platformRootType, platformDirection, platformMaxDepth)
			return graph, lineageTable(graph), err
		})
	},
}

var lineageNeighborsCmd = &cobra.Command{
	Use:           "neighbors",
	Short:         "Show hosted WhoDB lineage neighbors",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(platformNodeID) == "" || strings.TrimSpace(platformNodeType) == "" {
			return fmt.Errorf("--node-id and --node-type are required")
		}
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			graph, err := session.Client.LineageNeighbors(ctx, session.Host.DefaultProjectID, platformNodeID, platformNodeType)
			return graph, lineageTable(graph), err
		})
	},
}

var transformsListCmd = platformProjectListCommand("list", "List hosted WhoDB transforms", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	transforms, err := session.Client.Transforms(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(transforms))
	for i, transform := range transforms {
		rows[i] = []any{transform.ID, transform.Name, transform.TriggerMode, transform.ScheduleCron, transform.UpdatedAt}
	}
	return transforms, tableResult([]string{"id", "name", "trigger_mode", "schedule", "updated_at"}, rows), nil
})

var transformRunsCmd = &cobra.Command{
	Use:           "runs <transform>",
	Short:         "List hosted WhoDB transform runs",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			runs, err := session.Client.TransformRuns(ctx, session.Host.DefaultProjectID, args[0], platformLimit)
			if err != nil {
				return nil, nil, err
			}
			rows := make([][]any, len(runs))
			for i, run := range runs {
				rows[i] = []any{run.ID, run.Status, run.TriggeredBy, run.StartedAt, run.CompletedAt, run.ErrorMessage}
			}
			return runs, tableResult([]string{"id", "status", "triggered_by", "started_at", "completed_at", "error"}, rows), nil
		})
	},
}

var functionsListCmd = platformProjectListCommand("list", "List hosted WhoDB functions", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	functions, err := session.Client.Functions(ctx, session.Host.DefaultProjectID, platformFields)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(functions))
	for i, fn := range functions {
		rows[i] = []any{fn.ID, fn.Name, fn.Language, fn.EntryPoint, fn.IsDeployed, fn.UpdatedAt}
	}
	return functions, tableResult([]string{"id", "name", "language", "entry_point", "deployed", "updated_at"}, rows), nil
})

var functionGetCmd = platformIDCommand("get <function>", "Show hosted WhoDB function details", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	fn, err := session.Client.Function(ctx, session.Host.DefaultProjectID, id, platformFields)
	if err != nil {
		return nil, nil, err
	}
	rows := [][]any{{"id", fn.ID}, {"name", fn.Name}, {"language", fn.Language}, {"entry_point", fn.EntryPoint}, {"deployed", fn.IsDeployed}, {"files", len(fn.Files)}, {"dependencies", len(fn.Dependencies)}}
	return fn, tableResult([]string{"field", "value"}, rows), nil
})

var filesListCmd = platformProjectListCommand("list", "List hosted WhoDB project files", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	contents, err := session.Client.FolderContents(ctx, session.Host.DefaultProjectID, platformFolderID, platformFields)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, 0, len(contents.Folders)+len(contents.Files))
	for _, folder := range contents.Folders {
		rows = append(rows, []any{folder.ID, "folder", folder.Name, "", "", folder.CreatedAt})
	}
	for _, file := range contents.Files {
		rows = append(rows, []any{file.ID, "file", file.Name, file.MIMEType, file.SizeBytes, file.CreatedAt})
	}
	return contents, tableResult([]string{"id", "kind", "name", "mime_type", "bytes", "created_at"}, rows), nil
})

var filePreviewCmd = &cobra.Command{
	Use:           "preview <file>",
	Short:         "Preview a hosted WhoDB project file",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			var sheetIndex *int
			if cmd.Flags().Changed("sheet-index") {
				sheetIndex = &platformSheetIndex
			}
			preview, err := session.Client.FilePreview(ctx, session.Host.DefaultProjectID, args[0], sheetIndex, platformFields)
			if err != nil {
				return nil, nil, err
			}
			rows := [][]any{{"mime_type", preview.MIMEType}, {"size_bytes", preview.SizeBytes}, {"is_tabular", preview.IsTabular}, {"has_text", preview.TextContent != nil}, {"has_tabular", preview.Tabular != nil}}
			return preview, tableResult([]string{"field", "value"}, rows), nil
		})
	},
}

var fileSearchCmd = &cobra.Command{
	Use:           "search <query>",
	Short:         "Search hosted WhoDB project files",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			files, err := session.Client.SearchProjectFiles(ctx, session.Host.DefaultProjectID, args[0])
			return files, projectFilesTable(files), err
		})
	},
}

var tabularFilesCmd = platformProjectListCommand("tabular", "List hosted WhoDB tabular project files", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	files, err := session.Client.ProjectTabularFiles(ctx, session.Host.DefaultProjectID)
	return files, projectFilesTable(files), err
})

var storageUsageCmd = platformProjectListCommand("storage-usage", "Show hosted WhoDB project storage usage", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	usage, err := session.Client.ProjectStorageUsage(ctx, session.Host.DefaultProjectID)
	return map[string]int{"storage_used": usage}, tableResult([]string{"field", "value"}, [][]any{{"storage_used", usage}}), err
})

var resourcesCreateCmd = genericResourceWriteCommand("create <resource>", "Create a hosted WhoDB platform resource", "create")
var resourcesUpdateCmd = genericResourceWriteCommand("update <resource> <id>", "Update a hosted WhoDB platform resource", "update")
var resourcesDeleteCmd = genericResourceWriteCommand("delete <resource> <id>", "Delete a hosted WhoDB platform resource", "delete")
var resourcesActionCmd = genericResourceWriteCommand("action <resource> <action> [id]", "Run a hosted WhoDB platform resource action", "action")

func registerPlatformResourceCommands() {
	for _, command := range []*cobra.Command{secretsCmd, aiProvidersCmd, ontologiesCmd, datasetsCmd, lineageCmd, transformsCmd, functionsCmd, filesCmd, resourcesCmd} {
		command.PersistentFlags().StringVar(&platformResourceOrg, "org", "", "organization id, slug, or name (defaults to selected organization)")
		command.PersistentFlags().StringVar(&platformResourceProject, "project", "", "project id, slug, or name (defaults to selected project)")
	}

	secretsCmd.AddCommand(secretsListCmd)
	aiProvidersCmd.AddCommand(aiProvidersListCmd, aiProviderModelsCmd)
	ontologiesCmd.AddCommand(ontologiesListCmd, ontologyGetCmd, ontologyFastLookupsCmd, ontologyFastLookupSuggestionsCmd, ontologyRowsCmd, ontologyFollowLinkCmd)
	datasetsCmd.AddCommand(datasetsListCmd, datasetGetCmd, datasetRowsCmd)
	lineageCmd.AddCommand(lineageProjectCmd, lineageRootCmd, lineageNeighborsCmd)
	transformsCmd.AddCommand(transformsListCmd, transformRunsCmd)
	functionsCmd.AddCommand(functionsListCmd, functionGetCmd)
	filesCmd.AddCommand(filesListCmd, filePreviewCmd, fileSearchCmd, tabularFilesCmd, storageUsageCmd)
	resourcesCmd.AddCommand(resourcesCreateCmd, resourcesUpdateCmd, resourcesDeleteCmd, resourcesActionCmd)

	for _, command := range []*cobra.Command{functionsListCmd, functionGetCmd, filesListCmd, filePreviewCmd} {
		command.Flags().StringArrayVar(&platformFields, "field", nil, "top-level field to request; repeatable")
	}
	for _, command := range []*cobra.Command{ontologyRowsCmd, datasetRowsCmd, ontologyFollowLinkCmd} {
		command.Flags().IntVar(&platformLimit, "limit", 50, "maximum rows to return")
		command.Flags().IntVar(&platformOffset, "offset", 0, "row offset")
	}
	transformRunsCmd.Flags().IntVar(&platformLimit, "limit", 20, "maximum runs to return")
	filesListCmd.Flags().StringVar(&platformFolderID, "folder-id", "", "folder id to list; omitted means project root")
	filePreviewCmd.Flags().IntVar(&platformSheetIndex, "sheet-index", 0, "tabular sheet index to preview")
	lineageRootCmd.Flags().StringVar(&platformRootID, "root-id", "", "root node id")
	lineageRootCmd.Flags().StringVar(&platformRootType, "root-type", "", "root node type")
	lineageRootCmd.Flags().StringVar(&platformDirection, "direction", "", "lineage direction")
	lineageRootCmd.Flags().IntVar(&platformMaxDepth, "max-depth", 0, "maximum lineage depth")
	lineageNeighborsCmd.Flags().StringVar(&platformNodeID, "node-id", "", "node id")
	lineageNeighborsCmd.Flags().StringVar(&platformNodeType, "node-type", "", "node type")
	ontologyFollowLinkCmd.Flags().StringVar(&platformEntityID, "entity-id", "", "ontology entity id")
	ontologyFollowLinkCmd.Flags().StringVar(&platformPrimaryKey, "primary-key", "", "source row primary key")
	ontologyFollowLinkCmd.Flags().StringVar(&platformLinkAPIName, "link-api-name", "", "ontology link API name")
	for _, command := range []*cobra.Command{resourcesCreateCmd, resourcesUpdateCmd, resourcesDeleteCmd, resourcesActionCmd} {
		command.Flags().StringVar(&platformPayloadJSON, "payload-json", "", "JSON object payload for the hosted write")
		command.Flags().BoolVar(&platformPayloadStdin, "payload-stdin", false, "read JSON object payload from stdin")
		command.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "run the write without prompting")
	}
}

func platformProjectListCommand(use, short string, read func(context.Context, *platformSession, *platform.Project) (any, *output.QueryResult, error)) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlatformProjectRead(cmd, read)
		},
	}
}

func platformIDCommand(use, short string, read func(context.Context, *platformSession, string) (any, *output.QueryResult, error)) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
				return read(ctx, session, args[0])
			})
		},
	}
}

func pagedIDRowsCommand(use, short string, read func(context.Context, *platformSession, string) (*platform.DatasetQueryResult, error)) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validatePlatformPage(platformLimit, platformOffset); err != nil {
				return err
			}
			return runPlatformProjectRowsRead(cmd, func(ctx context.Context, session *platformSession) (*platform.DatasetQueryResult, error) {
				return read(ctx, session, args[0])
			})
		},
	}
}

func runPlatformProjectRead(cmd *cobra.Command, read func(context.Context, *platformSession, *platform.Project) (any, *output.QueryResult, error)) error {
	ctx := context.Background()
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	_, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return err
	}
	value, result, err := read(ctx, session, project)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, value)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(result)
}

func runPlatformProjectRowsRead(cmd *cobra.Command, read func(context.Context, *platformSession) (*platform.DatasetQueryResult, error)) error {
	return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
		result, err := read(ctx, session)
		if err != nil {
			return nil, nil, err
		}
		return result, datasetRowsTable(result), nil
	})
}

func genericResourceWriteCommand(use, short, operationKind string) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			switch operationKind {
			case "create":
				return cobra.ExactArgs(1)(cmd, args)
			case "update", "delete":
				return cobra.ExactArgs(2)(cmd, args)
			case "action":
				return cobra.RangeArgs(2, 3)(cmd, args)
			default:
				return fmt.Errorf("unsupported resource write operation %q", operationKind)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenericResourceWrite(cmd, args, operationKind)
		},
	}
}

func runGenericResourceWrite(cmd *cobra.Command, args []string, operationKind string) error {
	ctx := context.Background()
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	quiet := platformQuiet || format == output.FormatJSON
	out := newCommandOutput(cmd, format, quiet)
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	_, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return err
	}
	input := genericResourceInputFromArgs(args, operationKind)
	payload, err := readPlatformPayload(cmd)
	if err != nil {
		return err
	}
	spec, variables, err := buildGenericResourceVariables(project.ID, input, payload)
	if err != nil {
		return err
	}
	if !platformWriteYes {
		approved, err := confirmPlatformResourceWrite(cmd.InOrStdin(), cmd.ErrOrStderr(), spec, project.Name)
		if err != nil {
			return err
		}
		if !approved {
			return fmt.Errorf("write cancelled")
		}
	}
	if spec.Mutation == "UploadProjectFile" {
		filePath, _ := variables["filePath"].(string)
		folderID, _ := variables["folderId"].(*string)
		uploaded, err := session.Client.UploadProjectFile(ctx, project.ID, folderID, filePath)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "resources."+operationKind, uploaded)
		}
		out.Success("Uploaded file %s to project %s", uploaded.Name, project.Name)
		return nil
	}
	result, err := session.Client.PlatformMutation(ctx, spec.Mutation, variables)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, "resources."+operationKind, result)
	}
	out.Success("%s %s %s in project %s", titlePlatformAction(spec.Action), spec.Resource, spec.Mutation, project.Name)
	return nil
}

type genericResourceWriteInput struct {
	Resource string
	Action   string
	ID       string
}

func genericResourceInputFromArgs(args []string, operationKind string) genericResourceWriteInput {
	input := genericResourceWriteInput{Resource: args[0], Action: operationKind}
	switch operationKind {
	case "update", "delete":
		input.ID = args[1]
	case "action":
		input.Action = args[1]
		if len(args) == 3 {
			input.ID = args[2]
		}
	}
	return input
}

func readPlatformPayload(cmd *cobra.Command) (map[string]any, error) {
	raw := strings.TrimSpace(platformPayloadJSON)
	if platformPayloadStdin {
		body, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, err
		}
		raw = strings.TrimSpace(string(body))
	}
	if raw == "" {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}
	return payload, nil
}

func buildGenericResourceVariables(projectID string, input genericResourceWriteInput, payload map[string]any) (platform.GenericWriteSpec, map[string]any, error) {
	resource := normalizePlatformResourceToken(input.Resource)
	action := normalizePlatformResourceToken(input.Action)
	key := action + ":" + resource
	if action != "create" && action != "update" && action != "delete" {
		key = "action:" + action + ":" + resource
	}
	spec, ok := platform.GenericWriteSpecs[key]
	if !ok {
		return platform.GenericWriteSpec{}, nil, fmt.Errorf("unsupported platform %s for resource %q", action, resource)
	}
	id := strings.TrimSpace(input.ID)
	if spec.NeedsID && id == "" {
		return platform.GenericWriteSpec{}, nil, fmt.Errorf("id is required for %s %s", spec.Action, spec.Resource)
	}
	variables := map[string]any{}
	switch spec.Mode {
	case platform.GenericWriteModeInput:
		if spec.InjectProjectID {
			payload["projectId"] = projectID
		}
		if spec.NeedsID {
			if spec.Mutation == "PromoteFileToDataset" {
				payload["fileId"] = firstResourceString(payload, "fileId", id)
			} else {
				payload["id"] = id
			}
		}
		if spec.Action == "move" && spec.Resource == "file" {
			payload["newFolderId"] = nullableResourceString(payload, "newFolderId")
		}
		if spec.Action == "move" && spec.Resource == "folder" {
			payload["newParentId"] = nullableResourceString(payload, "newParentId")
		}
		variables["input"] = payload
	case platform.GenericWriteModeProjectID:
		variables["projectId"] = projectID
		variables["id"] = id
	case platform.GenericWriteModeID:
		variables["id"] = id
	case platform.GenericWriteModeProjectIDName:
		name := firstResourceString(payload, "name")
		if strings.TrimSpace(name) == "" {
			return platform.GenericWriteSpec{}, nil, fmt.Errorf("payload_json.name is required")
		}
		variables["projectId"] = projectID
		variables["id"] = id
		variables["name"] = strings.TrimSpace(name)
	case platform.GenericWriteModeDirect:
		for key, value := range payload {
			variables[key] = value
		}
		if spec.InjectProjectID {
			variables["projectId"] = projectID
		}
	case platform.GenericWriteModeFileUpload:
		filePath := firstResourceString(payload, "file_path", "filePath", "path")
		if strings.TrimSpace(filePath) == "" {
			return platform.GenericWriteSpec{}, nil, fmt.Errorf("payload_json.file_path is required")
		}
		variables["filePath"] = strings.TrimSpace(filePath)
		variables["folderId"] = nullableResourceString(payload, "folderId")
	default:
		return platform.GenericWriteSpec{}, nil, fmt.Errorf("unsupported write mode %q", spec.Mode)
	}
	return spec, variables, nil
}

func confirmPlatformResourceWrite(stdin io.Reader, stderr io.Writer, spec platform.GenericWriteSpec, projectName string) (bool, error) {
	if _, err := fmt.Fprintf(stderr, "%s %s in project %s? [y/N]: ", titlePlatformAction(spec.Action), spec.Resource, projectName); err != nil {
		return false, err
	}
	var answer string
	if _, err := fmt.Fscan(stdin, &answer); err != nil && err != io.EOF {
		return false, err
	}
	return isAffirmativeConfirmation(answer), nil
}

func normalizePlatformResourceToken(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, "-", "_")))
}

func titlePlatformAction(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func firstResourceString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nullableResourceString(payload map[string]any, key string) *string {
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(text)
	return &trimmed
}

func tableResult(columnNames []string, rows [][]any) *output.QueryResult {
	columns := make([]output.Column, len(columnNames))
	for i, name := range columnNames {
		columns[i] = output.Column{Name: name, Type: "string"}
	}
	return &output.QueryResult{Columns: columns, Rows: rows}
}

func datasetRowsTable(result *platform.DatasetQueryResult) *output.QueryResult {
	columns := make([]output.Column, len(result.Columns))
	for i, column := range result.Columns {
		columns[i] = output.Column{Name: column, Type: "string"}
	}
	rows := make([][]any, len(result.Rows))
	for i, row := range result.Rows {
		values := make([]any, len(row))
		for j, value := range row {
			values[j] = value
		}
		rows[i] = values
	}
	return &output.QueryResult{Columns: columns, Rows: rows}
}

func lineageTable(graph *platform.LineageGraph) *output.QueryResult {
	if graph == nil {
		return tableResult([]string{"kind", "id", "type", "name"}, nil)
	}
	rows := make([][]any, 0, len(graph.Nodes)+len(graph.Edges))
	for _, node := range graph.Nodes {
		rows = append(rows, []any{"node", node.ID, node.NodeType, node.Name})
	}
	for _, edge := range graph.Edges {
		rows = append(rows, []any{"edge", edge.SourceID + " -> " + edge.TargetID, edge.SourceType + " -> " + edge.TargetType, edge.CreatedAt})
	}
	return tableResult([]string{"kind", "id", "type", "name"}, rows)
}

func projectFilesTable(files []platform.ProjectFile) *output.QueryResult {
	rows := make([][]any, len(files))
	for i, file := range files {
		rows[i] = []any{file.ID, file.Name, file.MIMEType, file.SizeBytes, file.IsTabular, file.CreatedAt}
	}
	return tableResult([]string{"id", "name", "mime_type", "bytes", "tabular", "created_at"}, rows)
}

func sortedPlatformWriteSpecKeys() []string {
	keys := make([]string, 0, len(platform.GenericWriteSpecs))
	for key := range platform.GenericWriteSpecs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
