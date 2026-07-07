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
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	platformResourceOrg         string
	platformResourceProject     string
	platformFields              []string
	platformLimit               int
	platformOffset              int
	platformRootID              string
	platformRootType            string
	platformNodeID              string
	platformNodeType            string
	platformDirection           string
	platformMaxDepth            int
	platformFolderID            string
	platformEntityID            string
	platformPrimaryKey          string
	platformLinkAPIName         string
	platformSheetIndex          int
	platformIncludeRows         bool
	platformPayloadJSON         string
	platformPayloadStdin        bool
	platformWriteYes            bool
	platformFilterName          string
	platformFilterType          string
	platformFilterStatus        string
	platformFilterSchemaMode    string
	platformFilterKind          string
	platformFilterMIMEType      string
	platformFilterDeployed      string
	platformExportOutPath       string
	secretName                  string
	secretDescription           string
	secretValue                 string
	secretValueEnv              string
	secretValueStdin            bool
	aiProviderName              string
	aiProviderType              string
	aiProviderEndpoint          string
	aiProviderAPIKey            string
	aiProviderAPIKeyEnv         string
	aiProviderAPIKeyStdin       bool
	aiProviderModels            []string
	datasetName                 string
	datasetDescription          string
	datasetSourceID             string
	datasetSchemaMode           string
	datasetColumns              []string
	transformName               string
	transformDescription        string
	transformGraphJSON          string
	transformGraphFile          string
	transformScheduleCron       string
	transformTriggerMode        string
	functionName                string
	functionDescription         string
	functionLanguage            string
	functionEntryPoint          string
	functionTimeoutSeconds      int
	functionMemory              string
	functionCPU                 string
	functionFiles               []string
	functionDependencies        []string
	functionProviderIDs         []string
	functionOntologyIDs         []string
	functionReadOnlyOntologyIDs []string
	functionProviderConfigs     []string
	functionSecretBindings      []string
	functionDefaultMaxTokens    int
	functionDefaultTemperature  float64
	functionVersion             int
	functionPromoteMessage      string
	ontologyAPIName             string
	ontologyDisplayName         string
	ontologyPluralName          string
	ontologyDescription         string
	ontologyPrimaryKey          string
	ontologyTableName           string
	ontologySchemaName          string
	ontologyStatus              string
	ontologyIcon                string
	ontologyColor               string
	ontologyPropertiesJSON      []string
	ontologyLinksJSON           []string
	ontologyFastLookupFields    []string
	ontologyFastLookupReason    string
	ontologyRecordValues        []string
	ontologyRecordUpdateColumns []string
	folderName                  string
	folderParentID              string
	folderNewParentID           string
	filePath                    string
	fileOutPath                 string
	fileNewName                 string
	fileNewFolderID             string
	fileDatasetName             string
	fileDatasetDescription      string
	fileColumnMappings          []string
	functionInputJSON           string
	functionInputFile           string
	functionInputFileIDs        []string
)

var secretsCmd = &cobra.Command{Use: "secrets", Short: "Manage hosted WhoDB project secrets"}
var aiProvidersCmd = &cobra.Command{Use: "ai-providers", Short: "Manage hosted WhoDB AI providers"}
var ontologiesCmd = &cobra.Command{Use: "ontologies", Short: "Manage hosted WhoDB ontologies"}
var datasetsCmd = &cobra.Command{Use: "datasets", Short: "Manage hosted WhoDB datasets"}
var lineageCmd = &cobra.Command{Use: "lineage", Short: "Inspect hosted WhoDB lineage"}
var transformsCmd = &cobra.Command{Use: "transforms", Short: "Manage hosted WhoDB transforms"}
var functionsCmd = &cobra.Command{Use: "functions", Short: "Manage hosted WhoDB functions"}
var filesCmd = &cobra.Command{Use: "files", Short: "Manage hosted WhoDB project files"}
var foldersCmd = &cobra.Command{Use: "folders", Short: "Manage hosted WhoDB project folders"}
var resourcesCmd = &cobra.Command{Use: "resources", Short: "Run advanced hosted WhoDB platform resource writes"}

var capabilitiesCmd = &cobra.Command{
	Use:           "capabilities",
	Short:         "Show hosted WhoDB platform capabilities",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		manifest := manifestFromCache(session.Host.Manifest)
		if manifest == nil {
			manifest, err = refreshPlatformManifest(ctx, session.Config, &session.Host, session.Client)
			if err != nil {
				return err
			}
		}
		capabilities := platformStatusCapabilities(manifest)
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, capabilities)
		}
		rows := make([][]any, len(capabilities))
		for i, capability := range capabilities {
			rows[i] = []any{capability.Name, capability.Operation, capability.Supported}
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"name", "operation", "supported"}, rows))
	},
}

var secretsListCmd = platformProjectListCommand("list", "List hosted WhoDB secret metadata", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	secrets, err := session.Client.ProjectSecrets(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	secrets = filterSecrets(secrets)
	rows := make([][]any, len(secrets))
	for i, secret := range secrets {
		rows[i] = []any{secret.ID, secret.Name, secret.Description, len(secret.UsedBy), secret.UpdatedAt}
	}
	return secrets, tableResult([]string{"id", "name", "description", "used_by", "updated_at"}, rows), nil
})

var secretsGetCmd = platformIDCommand("get <secret>", "Show hosted WhoDB secret metadata", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	return readPlatformSecretDetail(ctx, session, id)
})
var secretDescribeCmd = platformIDCommand("describe <secret>", "Describe a hosted WhoDB secret", readPlatformSecretDetail)

func readPlatformSecretDetail(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	secrets, err := session.Client.ProjectSecrets(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	for _, secret := range secrets {
		if secret.ID == id || secret.Name == id {
			related := platformRelatedLineage(ctx, session, session.Host.DefaultProjectID, secret.ID, "secret")
			rows := [][]any{{"id", secret.ID}, {"name", secret.Name}, {"description", secret.Description}, {"used_by", len(secret.UsedBy)}, {"updated_at", secret.UpdatedAt}, {"upstream", len(related.Upstream)}, {"downstream", len(related.Downstream)}}
			return platformSecretDescribe{ProjectSecret: secret, Related: related}, tableResult([]string{"field", "value"}, rows), nil
		}
	}
	return nil, nil, fmt.Errorf("secret %q not found", id)
}

var aiProvidersListCmd = platformProjectListCommand("list", "List hosted WhoDB AI providers", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	providers, err := session.Client.AIProviders(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	providers = filterAIProviders(providers)
	rows := make([][]any, len(providers))
	for i, provider := range providers {
		rows[i] = []any{provider.ID, provider.Name, provider.ProviderType, provider.Endpoint, provider.UpdatedAt}
	}
	return providers, tableResult([]string{"id", "name", "type", "endpoint", "updated_at"}, rows), nil
})

var aiProviderGetCmd = platformIDCommand("get <provider>", "Show hosted WhoDB AI provider metadata", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	return readPlatformAIProviderDetail(ctx, session, id)
})
var aiProviderDescribeCmd = platformIDCommand("describe <provider>", "Describe a hosted WhoDB AI provider", readPlatformAIProviderDetail)

func readPlatformAIProviderDetail(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	providers, err := session.Client.AIProviders(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	for _, provider := range providers {
		if provider.ID == id || provider.Name == id {
			related := platformRelatedLineage(ctx, session, session.Host.DefaultProjectID, provider.ID, "ai_provider")
			rows := [][]any{{"id", provider.ID}, {"name", provider.Name}, {"type", provider.ProviderType}, {"endpoint", provider.Endpoint}, {"updated_at", provider.UpdatedAt}, {"upstream", len(related.Upstream)}, {"downstream", len(related.Downstream)}}
			return platformAIProviderDescribe{AIProvider: provider, Related: related}, tableResult([]string{"field", "value"}, rows), nil
		}
	}
	return nil, nil, fmt.Errorf("AI provider %q not found", id)
}

var aiProviderModelsCmd = &cobra.Command{
	Use:           "models <provider>",
	Short:         "List hosted WhoDB AI provider models",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			providerID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "ai_provider", args[0])
			if err != nil {
				return nil, nil, err
			}
			models, err := session.Client.AIProviderModels(ctx, session.Host.DefaultProjectID, providerID)
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
	ontologies = filterOntologies(ontologies)
	rows := make([][]any, len(ontologies))
	for i, ontology := range ontologies {
		rows[i] = []any{ontology.ID, ontology.APIName, ontology.DisplayName, ontology.TableName, ontology.Status}
	}
	return ontologies, tableResult([]string{"id", "api_name", "display_name", "table", "status"}, rows), nil
})

var ontologyGetCmd = platformIDCommand("get <ontology>", "Show hosted WhoDB ontology details", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	return readPlatformOntologyDetail(ctx, session, id)
})
var ontologyDescribeCmd = platformIDCommand("describe <ontology>", "Describe a hosted WhoDB ontology", readPlatformOntologyDetail)
var ontologyExportCmd = platformExportCommand("export <ontology>", "Export a hosted WhoDB ontology definition as JSON", func(ctx context.Context, session *platformSession, id string) (any, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "ontology", id)
	if err != nil {
		return nil, err
	}
	return session.Client.Ontology(ctx, session.Host.DefaultProjectID, resolvedID)
})

var ontologyFastLookupsCmd = platformIDCommand("fast-lookups <ontology>", "List hosted WhoDB ontology fast lookups", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "ontology", id)
	if err != nil {
		return nil, nil, err
	}
	lookups, err := session.Client.OntologyFastLookups(ctx, session.Host.DefaultProjectID, resolvedID)
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
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "ontology", id)
	if err != nil {
		return nil, nil, err
	}
	suggestions, err := session.Client.OntologyFastLookupSuggestions(ctx, session.Host.DefaultProjectID, resolvedID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(suggestions))
	for i, suggestion := range suggestions {
		rows[i] = []any{suggestion.DisplayName, strings.Join(suggestion.Fields, ","), suggestion.CanCreate, suggestion.Reason}
	}
	return suggestions, tableResult([]string{"display_name", "fields", "can_create", "reason"}, rows), nil
})

var ontologyFastLookupsCreateCmd = withExample(&cobra.Command{
	Use:           "create <ontology>",
	Short:         "Create a hosted WhoDB ontology fast lookup",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		payload, err := buildOntologyFastLookupCreatePayload(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		_, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
		if err != nil {
			return err
		}
		entityID, err := resolvePlatformResourceID(ctx, session, project.ID, "ontology", args[0])
		if err != nil {
			return err
		}
		payload["entityId"] = entityID
		return runPlatformResourceWrite(cmd, genericResourceWriteInput{Resource: "ontology_fast_lookup", Action: "create"}, "create", payload)
	},
}, `  whodb-cli ontologies fast-lookups create customers --field id --field email`)

var ontologyFastLookupsDeleteCmd = withExample(typedResourceWriteCommand("delete <lookup>", "Delete a hosted WhoDB ontology fast lookup", "delete", "ontology_fast_lookup", "", emptyTypedPayload), `  whodb-cli ontologies fast-lookups delete lookup_123 --yes`)

var ontologyRowsCmd = pagedIDRowsCommand("rows <ontology>", "Preview hosted WhoDB ontology rows", func(ctx context.Context, session *platformSession, id string) (*platform.DatasetQueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "ontology", id)
	if err != nil {
		return nil, err
	}
	return session.Client.OntologyRows(ctx, session.Host.DefaultProjectID, resolvedID, platformLimit, platformOffset)
})

var ontologyRecordsCmd = &cobra.Command{Use: "records", Short: "Manage hosted WhoDB ontology records"}
var ontologyRecordsAddCmd = withExample(typedResourceWriteCommand("add <ontology>", "Add a hosted WhoDB ontology record", "action", "ontology", "add_record", buildOntologyRecordAddPayload), `  whodb-cli ontologies records add customers --value id=1 --value name=Ada`)
var ontologyRecordsUpdateCmd = withExample(typedResourceWriteCommand("update <ontology>", "Update a hosted WhoDB ontology record", "action", "ontology", "update_record", buildOntologyRecordUpdatePayload), `  whodb-cli ontologies records update customers --value id=1 --value name=Grace --update-column name`)
var ontologyRecordsDeleteCmd = withExample(typedResourceWriteCommand("delete <ontology>", "Delete a hosted WhoDB ontology record", "action", "ontology", "delete_record", buildOntologyRecordDeletePayload), `  whodb-cli ontologies records delete customers --value id=1`)

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
	datasets = filterDatasets(datasets)
	rows := make([][]any, len(datasets))
	for i, dataset := range datasets {
		rows[i] = []any{dataset.ID, dataset.Name, dataset.SchemaMode, dataset.RowCount, dataset.SizeBytes, dataset.UpdatedAt}
	}
	return datasets, tableResult([]string{"id", "name", "schema_mode", "rows", "bytes", "updated_at"}, rows), nil
})

var datasetGetCmd = platformIDCommand("get <dataset>", "Show hosted WhoDB dataset details", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "dataset", id)
	if err != nil {
		return nil, nil, err
	}
	dataset, err := session.Client.Dataset(ctx, session.Host.DefaultProjectID, resolvedID)
	if err != nil {
		return nil, nil, err
	}
	rows := [][]any{{"id", dataset.ID}, {"name", dataset.Name}, {"schema_mode", dataset.SchemaMode}, {"row_count", dataset.RowCount}, {"size_bytes", dataset.SizeBytes}, {"columns", len(dataset.Schema)}}
	return dataset, tableResult([]string{"field", "value"}, rows), nil
})

var datasetSchemaCmd = platformIDCommand("schema <dataset>", "Show hosted WhoDB dataset schema", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "dataset", id)
	if err != nil {
		return nil, nil, err
	}
	dataset, err := session.Client.Dataset(ctx, session.Host.DefaultProjectID, resolvedID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(dataset.Schema))
	for i, column := range dataset.Schema {
		rows[i] = []any{column.Name, column.Type, column.IsNullable, column.IsPrimary}
	}
	return dataset.Schema, tableResult([]string{"name", "type", "nullable", "primary"}, rows), nil
})
var datasetDescribeCmd = platformIDCommand("describe <dataset>", "Describe a hosted WhoDB dataset", readPlatformDatasetDetail)
var datasetExportCmd = platformExportCommand("export <dataset>", "Export a hosted WhoDB dataset definition as JSON", func(ctx context.Context, session *platformSession, id string) (any, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "dataset", id)
	if err != nil {
		return nil, err
	}
	return session.Client.Dataset(ctx, session.Host.DefaultProjectID, resolvedID)
})

var datasetRowsCmd = pagedIDRowsCommand("rows <dataset>", "Preview hosted WhoDB dataset rows", func(ctx context.Context, session *platformSession, id string) (*platform.DatasetQueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "dataset", id)
	if err != nil {
		return nil, err
	}
	return session.Client.DatasetRows(ctx, session.Host.DefaultProjectID, resolvedID, platformLimit, platformOffset)
})
var datasetQueryCmd = pagedIDRowsCommand("query <dataset>", "Query hosted WhoDB dataset rows", func(ctx context.Context, session *platformSession, id string) (*platform.DatasetQueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "dataset", id)
	if err != nil {
		return nil, err
	}
	return session.Client.DatasetRows(ctx, session.Host.DefaultProjectID, resolvedID, platformLimit, platformOffset)
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
	transforms = filterTransforms(transforms)
	rows := make([][]any, len(transforms))
	for i, transform := range transforms {
		rows[i] = []any{transform.ID, transform.Name, transform.TriggerMode, transform.ScheduleCron, transform.UpdatedAt}
	}
	return transforms, tableResult([]string{"id", "name", "trigger_mode", "schedule", "updated_at"}, rows), nil
})

var transformGetCmd = platformIDCommand("get <transform>", "Show hosted WhoDB transform details", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	transforms, err := session.Client.Transforms(ctx, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	for _, transform := range transforms {
		if transform.ID == id || transform.Name == id {
			rows := [][]any{{"id", transform.ID}, {"name", transform.Name}, {"trigger_mode", transform.TriggerMode}, {"schedule", transform.ScheduleCron}, {"updated_at", transform.UpdatedAt}}
			return transform, tableResult([]string{"field", "value"}, rows), nil
		}
	}
	return nil, nil, fmt.Errorf("transform %q not found", id)
})
var transformDescribeCmd = platformIDCommand("describe <transform>", "Describe a hosted WhoDB transform", readPlatformTransformDetail)
var transformExportCmd = platformExportCommand("export <transform>", "Export a hosted WhoDB transform definition as JSON", func(ctx context.Context, session *platformSession, id string) (any, error) {
	transform, err := resolveTransform(ctx, session, session.Host.DefaultProjectID, id)
	if err != nil {
		return nil, err
	}
	return transform, nil
})

var transformRunsCmd = &cobra.Command{
	Use:           "runs <transform>",
	Short:         "List hosted WhoDB transform runs",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			transformID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "transform", args[0])
			if err != nil {
				return nil, nil, err
			}
			runs, err := session.Client.TransformRuns(ctx, session.Host.DefaultProjectID, transformID, platformLimit)
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
	functions = filterFunctions(functions)
	rows := make([][]any, len(functions))
	for i, fn := range functions {
		rows[i] = []any{fn.ID, fn.Name, fn.Language, fn.EntryPoint, fn.IsDeployed, fn.UpdatedAt}
	}
	return functions, tableResult([]string{"id", "name", "language", "entry_point", "deployed", "updated_at"}, rows), nil
})

var functionGetCmd = platformIDCommand("get <function>", "Show hosted WhoDB function details", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "function", id)
	if err != nil {
		return nil, nil, err
	}
	fn, err := session.Client.Function(ctx, session.Host.DefaultProjectID, resolvedID, platformFields)
	if err != nil {
		return nil, nil, err
	}
	rows := [][]any{{"id", fn.ID}, {"name", fn.Name}, {"language", fn.Language}, {"entry_point", fn.EntryPoint}, {"deployed", fn.IsDeployed}, {"files", len(fn.Files)}, {"dependencies", len(fn.Dependencies)}}
	return fn, tableResult([]string{"field", "value"}, rows), nil
})
var functionDescribeCmd = platformIDCommand("describe <function>", "Describe a hosted WhoDB function", readPlatformFunctionDetail)
var functionExportCmd = platformExportCommand("export <function>", "Export a hosted WhoDB function definition as JSON", func(ctx context.Context, session *platformSession, id string) (any, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "function", id)
	if err != nil {
		return nil, err
	}
	return session.Client.Function(ctx, session.Host.DefaultProjectID, resolvedID, platformFields)
})

var functionsVersionsCmd = platformIDCommand("versions <function>", "List promoted versions for a hosted WhoDB function", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	functionID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "function", id)
	if err != nil {
		return nil, nil, err
	}
	versions, err := session.Client.ObjectVersions(ctx, session.Host.DefaultProjectID, functionID, "function")
	if err != nil {
		return nil, nil, err
	}
	active, err := session.Client.ActiveProdVersion(ctx, session.Host.DefaultProjectID, functionID, "function")
	if err != nil {
		return nil, nil, err
	}
	activeVersion := 0
	if active != nil {
		activeVersion = active.Version
	}
	rows := make([][]any, len(versions))
	for i, version := range versions {
		rows[i] = []any{version.Version, version.Version == activeVersion, version.Message, version.PromotedBy, version.CreatedAt}
	}
	return versions, tableResult([]string{"version", "active", "message", "promoted_by", "created_at"}, rows), nil
})

var functionsActiveCmd = platformIDCommand("active <function>", "Show the active promoted version for a hosted WhoDB function", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	functionID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "function", id)
	if err != nil {
		return nil, nil, err
	}
	active, err := session.Client.ActiveProdVersion(ctx, session.Host.DefaultProjectID, functionID, "function")
	if err != nil {
		return nil, nil, err
	}
	if active == nil {
		value := map[string]any{"objectId": functionID, "objectType": "function", "active": false}
		return value, tableResult([]string{"field", "value"}, [][]any{{"object_id", functionID}, {"object_type", "function"}, {"active", false}}), nil
	}
	rows := [][]any{
		{"object_id", active.ObjectID},
		{"object_type", active.ObjectType},
		{"version", active.Version},
		{"activated_by", active.ActivatedBy},
		{"activated_at", active.ActivatedAt},
	}
	return active, tableResult([]string{"field", "value"}, rows), nil
})

var functionsPromoteCmd = &cobra.Command{
	Use:           "promote <function>",
	Short:         "Promote a hosted WhoDB function draft and make the new version active",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformFunctionLifecycleWrite(cmd, args[0], "promote")
	},
}

var functionsSetActiveCmd = &cobra.Command{
	Use:           "set-active <function>",
	Short:         "Set an existing hosted WhoDB function version active",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if functionVersion <= 0 {
			return fmt.Errorf("--version must be greater than 0")
		}
		return runPlatformFunctionLifecycleWrite(cmd, args[0], "set-active")
	},
}

var functionsRestoreDraftCmd = &cobra.Command{
	Use:           "restore-draft <function>",
	Short:         "Restore a promoted hosted WhoDB function version into the draft",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if functionVersion <= 0 {
			return fmt.Errorf("--version must be greater than 0")
		}
		return runPlatformFunctionLifecycleWrite(cmd, args[0], "restore-draft")
	},
}

var functionsRunCmd = &cobra.Command{
	Use:           "run <function>",
	Short:         "Run a hosted WhoDB function",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			input, err := readFunctionInput(cmd)
			if err != nil {
				return nil, nil, err
			}
			functionID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "function", args[0])
			if err != nil {
				return nil, nil, err
			}
			result, err := session.Client.ExecuteFunction(ctx, session.Host.DefaultProjectID, functionID, input, normalizedStringList(functionInputFileIDs))
			if err != nil {
				return nil, nil, err
			}
			return result, functionExecutionTable(result), nil
		})
	},
}

var functionsTestCmd = &cobra.Command{
	Use:           "test <function>",
	Short:         "Test a hosted WhoDB function draft",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			input, err := readFunctionInput(cmd)
			if err != nil {
				return nil, nil, err
			}
			files, err := parseFunctionFileInputs(functionFiles)
			if err != nil {
				return nil, nil, err
			}
			functionID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "function", args[0])
			if err != nil {
				return nil, nil, err
			}
			result, err := session.Client.TestFunction(ctx, session.Host.DefaultProjectID, functionID, input, files, normalizedStringList(functionInputFileIDs))
			if err != nil {
				return nil, nil, err
			}
			return result, functionExecutionTable(result), nil
		})
	},
}

var functionsPreviewCmd = &cobra.Command{
	Use:           "preview",
	Short:         "Preview an unsaved hosted WhoDB function definition",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			language := strings.TrimSpace(functionLanguage)
			entryPoint := strings.TrimSpace(functionEntryPoint)
			if language == "" || entryPoint == "" {
				return nil, nil, fmt.Errorf("--language and --entry-point are required")
			}
			input, err := readFunctionInput(cmd)
			if err != nil {
				return nil, nil, err
			}
			files, err := parseFunctionFileInputs(functionFiles)
			if err != nil {
				return nil, nil, err
			}
			if len(files) == 0 {
				return nil, nil, fmt.Errorf("--file is required at least once")
			}
			dependencies, err := parseFunctionDependencyNames(functionDependencies)
			if err != nil {
				return nil, nil, err
			}
			result, err := session.Client.PreviewFunction(ctx, session.Host.DefaultProjectID, language, entryPoint, input, files, dependencies)
			if err != nil {
				return nil, nil, err
			}
			return result, functionExecutionTable(result), nil
		})
	},
}

var filesListCmd = platformProjectListCommand("list", "List hosted WhoDB project files", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	contents, err := session.Client.FolderContents(ctx, session.Host.DefaultProjectID, platformFolderID, platformFields)
	if err != nil {
		return nil, nil, err
	}
	filtered := filterFolderContents(contents)
	rows := make([][]any, 0, len(filtered.Folders)+len(filtered.Files))
	for _, folder := range filtered.Folders {
		rows = append(rows, []any{folder.ID, "folder", folder.Name, "", "", folder.CreatedAt})
	}
	for _, file := range filtered.Files {
		rows = append(rows, []any{file.ID, "file", file.Name, file.MIMEType, file.SizeBytes, file.CreatedAt})
	}
	return filtered, tableResult([]string{"id", "kind", "name", "mime_type", "bytes", "created_at"}, rows), nil
})

var fileGetCmd = platformIDCommand("get <file>", "Show hosted WhoDB project file metadata", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	file, err := resolveProjectFile(ctx, session, session.Host.DefaultProjectID, id)
	if err != nil {
		return nil, nil, err
	}
	folderID := ""
	if file.FolderID != nil {
		folderID = *file.FolderID
	}
	datasetID := ""
	if file.DatasetID != nil {
		datasetID = *file.DatasetID
	}
	rows := [][]any{
		{"id", file.ID},
		{"name", file.Name},
		{"folder_id", folderID},
		{"mime_type", file.MIMEType},
		{"size_bytes", file.SizeBytes},
		{"is_tabular", file.IsTabular},
		{"dataset_id", datasetID},
		{"created_at", file.CreatedAt},
		{"updated_at", file.UpdatedAt},
	}
	return file, tableResult([]string{"field", "value"}, rows), nil
})
var fileDescribeCmd = platformIDCommand("describe <file>", "Describe hosted WhoDB project file metadata", readPlatformFileDetail)

var filePreviewCmd = &cobra.Command{
	Use:           "preview <file>",
	Short:         "Preview a hosted WhoDB project file",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			fileID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "file", args[0])
			if err != nil {
				return nil, nil, err
			}
			var sheetIndex *int
			if cmd.Flags().Changed("sheet-index") {
				sheetIndex = &platformSheetIndex
			}
			preview, err := session.Client.FilePreview(ctx, session.Host.DefaultProjectID, fileID, sheetIndex, platformFields)
			if err != nil {
				return nil, nil, err
			}
			rows := [][]any{{"mime_type", preview.MIMEType}, {"size_bytes", preview.SizeBytes}, {"is_tabular", preview.IsTabular}, {"has_text", preview.TextContent != nil}, {"has_tabular", preview.Tabular != nil}}
			return preview, tableResult([]string{"field", "value"}, rows), nil
		})
	},
}

var fileInspectCmd = &cobra.Command{
	Use:           "inspect <file>",
	Short:         "Inspect hosted WhoDB tabular file columns for dataset promotion",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			inspection, err := inspectPlatformFile(ctx, cmd, session, args[0])
			if err != nil {
				return nil, nil, err
			}
			if !platformIncludeRows {
				inspection.Rows = nil
			}
			return inspection, fileInspectionTable(inspection), nil
		})
	},
}

var fileColumnsCmd = &cobra.Command{
	Use:           "columns <file>",
	Short:         "Show hosted WhoDB tabular file columns and promotion mappings",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			inspection, err := inspectPlatformFile(ctx, cmd, session, args[0])
			if err != nil {
				return nil, nil, err
			}
			return inspection.Columns, fileInspectionColumnsTable(inspection), nil
		})
	},
}

var fileDownloadCmd = &cobra.Command{
	Use:           "download <file>",
	Short:         "Download previewable hosted WhoDB file content",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		_, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
		if err != nil {
			return err
		}
		file, err := resolveProjectFile(ctx, session, project.ID, args[0])
		if err != nil {
			return err
		}
		var sheetIndex *int
		if cmd.Flags().Changed("sheet-index") {
			sheetIndex = &platformSheetIndex
		}
		preview, err := session.Client.FilePreview(ctx, project.ID, file.ID, sheetIndex, nil)
		if err != nil {
			return err
		}
		content, err := previewFileDownloadContent(preview)
		if err != nil {
			return err
		}
		if strings.TrimSpace(fileOutPath) == "" {
			_, err = cmd.OutOrStdout().Write(content)
			return err
		}
		return os.WriteFile(filepath.Clean(fileOutPath), content, 0600)
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

var foldersListCmd = platformProjectListCommand("list", "List hosted WhoDB project folders", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	contents, err := session.Client.FolderContents(ctx, session.Host.DefaultProjectID, platformFolderID, platformFields)
	if err != nil {
		return nil, nil, err
	}
	folders := filterFolders(contents.Folders)
	rows := make([][]any, len(folders))
	for i, folder := range folders {
		parentID := ""
		if folder.ParentID != nil {
			parentID = *folder.ParentID
		}
		rows[i] = []any{folder.ID, folder.Name, parentID, folder.CreatedAt}
	}
	return folders, tableResult([]string{"id", "name", "parent_id", "created_at"}, rows), nil
})

var folderGetCmd = platformIDCommand("get <folder>", "Show hosted WhoDB project folder metadata", func(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	return readPlatformFolderDetail(ctx, session, id)
})
var folderDescribeCmd = platformIDCommand("describe <folder>", "Describe hosted WhoDB project folder metadata", readPlatformFolderDetail)

func readPlatformFolderDetail(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	folder, err := resolveProjectFolder(ctx, session, session.Host.DefaultProjectID, id)
	if err != nil {
		return nil, nil, err
	}
	parentID := ""
	if folder.ParentID != nil {
		parentID = *folder.ParentID
	}
	related := platformRelatedLineage(ctx, session, session.Host.DefaultProjectID, folder.ID, "project_folder")
	rows := [][]any{{"id", folder.ID}, {"name", folder.Name}, {"parent_id", parentID}, {"created_by", folder.CreatedBy}, {"created_at", folder.CreatedAt}, {"upstream", len(related.Upstream)}, {"downstream", len(related.Downstream)}}
	return platformFolderDescribe{ProjectFolder: *folder, Related: related}, tableResult([]string{"field", "value"}, rows), nil
}

var foldersTreeCmd = platformProjectListCommand("tree", "Show hosted WhoDB project folder tree", func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
	entries, err := loadProjectFolderTree(ctx, session, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(entries))
	for i, entry := range entries {
		rows[i] = []any{entry.ID, entry.Kind, entry.Name, entry.ParentID, entry.Depth, entry.Path}
	}
	return entries, tableResult([]string{"id", "kind", "name", "parent_id", "depth", "path"}, rows), nil
})

var resourcesCreateCmd = genericResourceWriteCommand("create <resource>", "Create a hosted WhoDB platform resource", "create")
var resourcesUpdateCmd = genericResourceWriteCommand("update <resource> <id>", "Update a hosted WhoDB platform resource", "update")
var resourcesDeleteCmd = genericResourceWriteCommand("delete <resource> <id>", "Delete a hosted WhoDB platform resource", "delete")
var resourcesActionCmd = genericResourceWriteCommand("action <resource> <action> [id]", "Run a hosted WhoDB platform resource action", "action")
var resourcesSpecsCmd = &cobra.Command{
	Use:           "specs",
	Short:         "List supported hosted WhoDB generic resource writes",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		specs := make([]platformKeyedGenericWriteSpec, 0, len(platform.GenericWriteSpecs))
		rows := make([][]any, 0, len(platform.GenericWriteSpecs))
		for _, key := range sortedPlatformWriteSpecKeys() {
			spec := platform.GenericWriteSpecs[key]
			specs = append(specs, platformKeyedGenericWriteSpec{Key: key, GenericWriteSpec: spec})
			rows = append(rows, []any{key, spec.Resource, spec.Action, spec.Mutation, spec.Mode, spec.NeedsID})
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, specs)
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"key", "resource", "action", "mutation", "mode", "needs_id"}, rows))
	},
}
var resourcesShapeCmd = &cobra.Command{
	Use:           "shape <write-key>",
	Short:         "Show payload shape for a hosted WhoDB generic resource write",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		key := normalizePlatformResourceKey(args[0])
		shape, ok := platform.PayloadShapes[key]
		if !ok {
			return fmt.Errorf("payload shape %q not found", args[0])
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, shape)
		}
		rows := make([][]any, len(shape.Fields))
		for i, field := range shape.Fields {
			rows[i] = []any{field.Name, field.Type, field.Required, field.Secret, field.Description}
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"field", "type", "required", "secret", "description"}, rows))
	},
}
var secretsCreateCmd = withExample(typedResourceWriteCommand("create", "Create a hosted WhoDB secret", "create", "secret", "", buildSecretCreatePayload), `  whodb-cli secrets create --name OPENAI_API_KEY --value-env OPENAI_API_KEY
  printf %s "$TOKEN" | whodb-cli secrets create --name SERVICE_TOKEN --value-stdin`)
var secretsUpdateCmd = withExample(typedResourceWriteCommand("update <secret>", "Update a hosted WhoDB secret", "update", "secret", "", buildSecretUpdatePayload), `  whodb-cli secrets update sec_123 --description "rotated key" --value-env OPENAI_API_KEY
  printf %s "$TOKEN" | whodb-cli secrets update sec_123 --value-stdin`)
var secretsDeleteCmd = withExample(typedResourceWriteCommand("delete <secret>", "Delete a hosted WhoDB secret", "delete", "secret", "", emptyTypedPayload), `  whodb-cli secrets delete sec_123
  whodb-cli secrets delete sec_123 --yes`)
var aiProvidersCreateCmd = withExample(typedResourceWriteCommand("create", "Create a hosted WhoDB AI provider", "create", "ai_provider", "", buildAIProviderCreatePayload), `  whodb-cli ai-providers create --name OpenAI --type openai --endpoint https://api.openai.com/v1 --api-key-env OPENAI_API_KEY --model gpt-4.1`)
var aiProvidersUpdateCmd = withExample(typedResourceWriteCommand("update <provider>", "Update a hosted WhoDB AI provider", "update", "ai_provider", "", buildAIProviderUpdatePayload), `  whodb-cli ai-providers update provider_123 --endpoint https://api.openai.com/v1 --model gpt-4.1 --model gpt-4.1-mini
  whodb-cli ai-providers update provider_123 --api-key-env OPENAI_API_KEY`)
var aiProvidersDeleteCmd = withExample(typedResourceWriteCommand("delete <provider>", "Delete a hosted WhoDB AI provider", "delete", "ai_provider", "", emptyTypedPayload), `  whodb-cli ai-providers delete provider_123 --yes`)
var datasetsCreateCmd = withExample(typedResourceWriteCommand("create", "Create a hosted WhoDB dataset", "create", "dataset", "", buildDatasetCreatePayload), `  whodb-cli datasets create --name Customers --schema-mode manual --column id:text:primary --column email:text:nullable`)
var datasetsUpdateCmd = withExample(typedResourceWriteCommand("update <dataset>", "Update a hosted WhoDB dataset", "update", "dataset", "", buildDatasetUpdatePayload), `  whodb-cli datasets update dataset_123 --description "Customer import"
  whodb-cli datasets update dataset_123 --column id:text:primary --column email:text:nullable`)
var datasetsDeleteCmd = withExample(typedResourceWriteCommand("delete <dataset>", "Delete a hosted WhoDB dataset", "delete", "dataset", "", emptyTypedPayload), `  whodb-cli datasets delete dataset_123 --yes`)
var ontologiesCreateCmd = withExample(typedResourceWriteCommand("create", "Create a hosted WhoDB ontology", "create", "ontology", "", buildOntologyCreatePayload), `  whodb-cli ontologies create --api-name customer --display-name Customer --plural-display-name Customers --primary-key id --table-name customers --schema-name public --property-json '{"apiName":"id","displayName":"ID","columnName":"id","dataType":"String","isRequired":true}'`)
var ontologiesUpdateCmd = withExample(typedResourceWriteCommand("update <ontology>", "Update a hosted WhoDB ontology", "update", "ontology", "", buildOntologyUpdatePayload), `  whodb-cli ontologies update ontology_123 --display-name "Customer Account" --status active`)
var ontologiesDeleteCmd = withExample(typedResourceWriteCommand("delete <ontology>", "Delete a hosted WhoDB ontology", "delete", "ontology", "", emptyTypedPayload), `  whodb-cli ontologies delete ontology_123 --yes`)
var transformsCreateCmd = withExample(typedResourceWriteCommand("create", "Create a hosted WhoDB transform", "create", "transform", "", buildTransformCreatePayload), `  whodb-cli transforms create --name daily-load --trigger-mode manual --graph-file ./transform.json`)
var transformsUpdateCmd = withExample(typedResourceWriteCommand("update <transform>", "Update a hosted WhoDB transform", "update", "transform", "", buildTransformUpdatePayload), `  whodb-cli transforms update transform_123 --description "Daily customer load" --graph-file ./transform.json`)
var transformsRunCmd = withExample(typedResourceWriteCommand("run <transform>", "Run a hosted WhoDB transform", "action", "transform", "run", emptyTypedPayload), `  whodb-cli transforms run transform_123`)
var transformsDeleteCmd = withExample(typedResourceWriteCommand("delete <transform>", "Delete a hosted WhoDB transform", "delete", "transform", "", emptyTypedPayload), `  whodb-cli transforms delete transform_123 --yes`)
var functionsCreateCmd = withExample(typedResourceWriteCommand("create", "Create a hosted WhoDB function", "create", "function", "", buildFunctionCreatePayload), `  whodb-cli functions create --name enrich-customer --language python --entry-point main --file main.py=./main.py`)
var functionsUpdateCmd = withExample(typedResourceWriteCommand("update <function>", "Update a hosted WhoDB function", "update", "function", "", buildFunctionUpdatePayload), `  whodb-cli functions update function_123 --description "Updated enrichment" --file main.py=./main.py`)
var functionsDeployCmd = withExample(platformFunctionDeployCommand("deploy <function>", "Deploy a hosted WhoDB function", "DeployFunction", "deploy"), `  whodb-cli functions deploy function_123`)
var functionsRedeployCmd = withExample(platformFunctionDeployCommand("redeploy <function>", "Redeploy a hosted WhoDB function", "RedeployFunction", "redeploy"), `  whodb-cli functions redeploy function_123`)
var functionsDeleteCmd = withExample(typedResourceWriteCommand("delete <function>", "Delete a hosted WhoDB function", "delete", "function", "", emptyTypedPayload), `  whodb-cli functions delete function_123 --yes`)
var foldersCreateCmd = withExample(typedResourceWriteCommand("create", "Create a hosted WhoDB project folder", "create", "folder", "", buildFolderCreatePayload), `  whodb-cli folders create --name imports
  whodb-cli folders create --name january --parent-id folder_123`)
var foldersRenameCmd = withExample(typedResourceWriteCommand("rename <folder>", "Rename a hosted WhoDB project folder", "action", "folder", "rename", buildFolderRenamePayload), `  whodb-cli folders rename folder_123 --name imports-2026`)
var foldersMoveCmd = withExample(typedResourceWriteCommand("move <folder>", "Move a hosted WhoDB project folder", "action", "folder", "move", buildFolderMovePayload), `  whodb-cli folders move folder_123 --parent-id folder_456
  whodb-cli folders move folder_123`)
var foldersDeleteCmd = withExample(typedResourceWriteCommand("delete <folder>", "Delete a hosted WhoDB project folder", "delete", "folder", "", emptyTypedPayload), `  whodb-cli folders delete folder_123 --yes`)
var filesUploadCmd = withExample(typedResourceWriteCommand("upload", "Upload a hosted WhoDB project file", "action", "file", "upload", buildFileUploadPayload), `  whodb-cli files upload --path ./customers.csv
  whodb-cli files upload --path ./customers.csv --folder-id folder_123`)
var filesDeleteCmd = withExample(typedResourceWriteCommand("delete <file>", "Delete a hosted WhoDB project file", "delete", "file", "", emptyTypedPayload), `  whodb-cli files delete file_123 --yes`)
var filesRenameCmd = withExample(typedResourceWriteCommand("rename <file>", "Rename a hosted WhoDB project file", "action", "file", "rename", buildFileRenamePayload), `  whodb-cli files rename file_123 --name customers-2026.csv`)
var filesMoveCmd = withExample(typedResourceWriteCommand("move <file>", "Move a hosted WhoDB project file", "action", "file", "move", buildFileMovePayload), `  whodb-cli files move file_123 --folder-id folder_123
  whodb-cli files move file_123`)
var filesPromoteDatasetCmd = withExample(typedResourceWriteCommand("promote-to-dataset <file>", "Promote a hosted WhoDB project file to a dataset", "action", "file", "promote_to_dataset", buildFilePromoteDatasetPayload), `  whodb-cli files promote-to-dataset customers.csv --name Customers --column-map id:id:text:primary --column-map name:name:text:nullable`)

func registerPlatformResourceCommands() {
	for _, command := range []*cobra.Command{secretsCmd, aiProvidersCmd, ontologiesCmd, datasetsCmd, lineageCmd, transformsCmd, functionsCmd, filesCmd, foldersCmd, resourcesCmd} {
		command.PersistentFlags().StringVar(&platformResourceOrg, "org", "", "organization id, slug, or name (defaults to selected organization)")
		command.PersistentFlags().StringVar(&platformResourceProject, "project", "", "project id, slug, or name (defaults to selected project)")
	}

	secretsCmd.AddCommand(secretsListCmd, secretsGetCmd, secretDescribeCmd, secretsCreateCmd, secretsUpdateCmd, secretsDeleteCmd)
	aiProvidersCmd.AddCommand(aiProvidersListCmd, aiProviderGetCmd, aiProviderDescribeCmd, aiProviderModelsCmd, aiProvidersCreateCmd, aiProvidersUpdateCmd, aiProvidersDeleteCmd)
	ontologyFastLookupsCmd.AddCommand(ontologyFastLookupsCreateCmd, ontologyFastLookupsDeleteCmd)
	ontologyRecordsCmd.AddCommand(ontologyRecordsAddCmd, ontologyRecordsUpdateCmd, ontologyRecordsDeleteCmd)
	ontologiesCmd.AddCommand(ontologiesListCmd, ontologyGetCmd, ontologyDescribeCmd, ontologyExportCmd, ontologyCloneCmd, ontologyFastLookupsCmd, ontologyFastLookupSuggestionsCmd, ontologyRowsCmd, ontologyFollowLinkCmd, ontologyRecordsCmd, ontologiesCreateCmd, ontologiesUpdateCmd, ontologiesDeleteCmd)
	datasetsCmd.AddCommand(datasetsListCmd, datasetGetCmd, datasetDescribeCmd, datasetSchemaCmd, datasetExportCmd, datasetCloneCmd, datasetRowsCmd, datasetQueryCmd, datasetsCreateCmd, datasetsUpdateCmd, datasetsDeleteCmd)
	lineageCmd.AddCommand(lineageProjectCmd, lineageRootCmd, lineageNeighborsCmd)
	transformsCmd.AddCommand(transformsListCmd, transformGetCmd, transformDescribeCmd, transformExportCmd, transformCloneCmd, transformRunsCmd, transformsCreateCmd, transformsUpdateCmd, transformsRunCmd, transformsDeleteCmd)
	functionsCmd.AddCommand(functionsListCmd, functionGetCmd, functionDescribeCmd, functionExportCmd, functionCloneCmd, functionsVersionsCmd, functionsActiveCmd, functionsPromoteCmd, functionsSetActiveCmd, functionsRestoreDraftCmd, functionsRunCmd, functionsTestCmd, functionsPreviewCmd, functionsCreateCmd, functionsUpdateCmd, functionsDeployCmd, functionsRedeployCmd, functionsDeleteCmd)
	filesCmd.AddCommand(filesListCmd, fileGetCmd, fileDescribeCmd, filePreviewCmd, fileInspectCmd, fileColumnsCmd, fileDownloadCmd, fileSearchCmd, tabularFilesCmd, storageUsageCmd, filesUploadCmd, filesPromoteDatasetCmd, filesDeleteCmd, filesRenameCmd, filesMoveCmd)
	foldersCmd.AddCommand(foldersListCmd, folderGetCmd, folderDescribeCmd, foldersTreeCmd, foldersCreateCmd, foldersRenameCmd, foldersMoveCmd, foldersDeleteCmd)
	resourcesCmd.AddCommand(resourcesSpecsCmd, resourcesShapeCmd, resourcesExportCmd, resourcesDiffCmd, resourcesImportCmd, resourcesCreateCmd, resourcesUpdateCmd, resourcesDeleteCmd, resourcesActionCmd)

	for _, command := range []*cobra.Command{functionsListCmd, functionGetCmd, functionDescribeCmd, functionExportCmd, filesListCmd, filePreviewCmd} {
		command.Flags().StringArrayVar(&platformFields, "field", nil, "top-level field to request; repeatable")
	}
	for _, command := range []*cobra.Command{ontologyExportCmd, datasetExportCmd, transformExportCmd, functionExportCmd} {
		command.Flags().StringVar(&platformExportOutPath, "out", "", "destination path; omitted writes JSON to stdout")
	}
	resourcesExportCmd.Flags().StringVar(&platformExportOutPath, "out", "", "destination path; omitted writes JSON to stdout")
	resourcesExportCmd.Flags().BoolVar(&platformBundleIncludeFiles, "include-files", false, "include previewable uploaded file content in the bundle")
	resourcesExportCmd.Flags().IntVar(&platformBundleMaxFileBytes, "max-file-bytes", 1<<20, "maximum bytes to include per uploaded file when --include-files is set")
	resourcesDiffCmd.Flags().StringVar(&platformBundlePath, "file", "", "project bundle JSON file")
	resourcesDiffCmd.Flags().StringVar(&platformBundlePrefix, "prefix", "", "prefix added to imported resource names in the plan")
	resourcesDiffCmd.Flags().BoolVar(&platformRenameConflicts, "rename-conflicts", false, "plan unique names for resources that conflict with existing resources")
	resourcesDiffCmd.Flags().BoolVar(&platformOverwriteConflicts, "overwrite-conflicts", false, "plan updates for resources that conflict with existing resources")
	resourcesImportCmd.Flags().StringVar(&platformBundlePath, "file", "", "project bundle JSON file")
	resourcesImportCmd.Flags().BoolVar(&platformImportDryRun, "dry-run", false, "show the import plan without writing")
	resourcesImportCmd.Flags().StringVar(&platformBundlePrefix, "prefix", "", "prefix added to imported resource names")
	resourcesImportCmd.Flags().BoolVar(&platformRenameConflicts, "rename-conflicts", false, "create unique names for resources that conflict with existing resources")
	resourcesImportCmd.Flags().BoolVar(&platformOverwriteConflicts, "overwrite-conflicts", false, "update resources that conflict with existing resources")
	resourcesImportCmd.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "run the import without prompting")
	for _, command := range []*cobra.Command{secretsListCmd, aiProvidersListCmd, ontologiesListCmd, datasetsListCmd, transformsListCmd, functionsListCmd, filesListCmd, foldersListCmd} {
		command.Flags().StringVar(&platformFilterName, "name", "", "case-insensitive name substring filter")
	}
	aiProvidersListCmd.Flags().StringVar(&platformFilterType, "type", "", "provider type filter")
	ontologiesListCmd.Flags().StringVar(&platformFilterStatus, "status", "", "ontology status filter")
	datasetsListCmd.Flags().StringVar(&platformFilterSchemaMode, "schema-mode", "", "dataset schema mode filter")
	transformsListCmd.Flags().StringVar(&platformFilterType, "trigger-mode", "", "transform trigger mode filter")
	functionsListCmd.Flags().StringVar(&platformFilterType, "language", "", "function language filter")
	functionsListCmd.Flags().StringVar(&platformFilterDeployed, "deployed", "", "function deployment filter: true or false")
	filesListCmd.Flags().StringVar(&platformFilterKind, "kind", "", "entry kind filter: file or folder")
	filesListCmd.Flags().StringVar(&platformFilterMIMEType, "mime-type", "", "file MIME type substring filter")
	for _, command := range []*cobra.Command{ontologyRowsCmd, datasetRowsCmd, datasetQueryCmd, ontologyFollowLinkCmd} {
		command.Flags().IntVar(&platformLimit, "limit", 50, "maximum rows to return")
		command.Flags().IntVar(&platformOffset, "offset", 0, "row offset")
	}
	transformRunsCmd.Flags().IntVar(&platformLimit, "limit", 20, "maximum runs to return")
	filesListCmd.Flags().StringVar(&platformFolderID, "folder-id", "", "folder id to list; omitted means project root")
	foldersListCmd.Flags().StringVar(&platformFolderID, "folder-id", "", "parent folder id to list; omitted means project root")
	filePreviewCmd.Flags().IntVar(&platformSheetIndex, "sheet-index", 0, "tabular sheet index to preview")
	fileInspectCmd.Flags().IntVar(&platformSheetIndex, "sheet-index", 0, "tabular sheet index to inspect")
	fileInspectCmd.Flags().BoolVar(&platformIncludeRows, "include-rows", false, "include preview rows in JSON output")
	fileColumnsCmd.Flags().IntVar(&platformSheetIndex, "sheet-index", 0, "tabular sheet index to inspect")
	fileDownloadCmd.Flags().StringVar(&fileOutPath, "out", "", "destination path; omitted writes content to stdout")
	fileDownloadCmd.Flags().IntVar(&platformSheetIndex, "sheet-index", 0, "tabular sheet index to download")
	functionsRunCmd.Flags().StringVar(&functionInputJSON, "input-json", "{}", "function input JSON string")
	functionsRunCmd.Flags().StringVar(&functionInputFile, "input-file", "", "path to function input JSON file")
	functionsRunCmd.Flags().StringArrayVar(&functionInputFileIDs, "input-file-id", nil, "hosted project file id to pass to the function; repeatable")
	functionsTestCmd.Flags().StringVar(&functionInputJSON, "input-json", "{}", "function input JSON string")
	functionsTestCmd.Flags().StringVar(&functionInputFile, "input-file", "", "path to function input JSON file")
	functionsTestCmd.Flags().StringArrayVar(&functionInputFileIDs, "input-file-id", nil, "hosted project file id to pass to the function; repeatable")
	functionsTestCmd.Flags().StringArrayVar(&functionFiles, "file", nil, "function file override as target-path=local-path; repeatable")
	functionsPreviewCmd.Flags().StringVar(&functionLanguage, "language", "", "function language")
	functionsPreviewCmd.Flags().StringVar(&functionEntryPoint, "entry-point", "", "function entry point")
	functionsPreviewCmd.Flags().StringArrayVar(&functionFiles, "file", nil, "function file as target-path=local-path; repeatable")
	functionsPreviewCmd.Flags().StringArrayVar(&functionDependencies, "dependency", nil, "function dependency name or name:version; repeatable")
	functionsPreviewCmd.Flags().StringVar(&functionInputJSON, "input-json", "{}", "function input JSON string")
	functionsPreviewCmd.Flags().StringVar(&functionInputFile, "input-file", "", "path to function input JSON file")
	functionsPromoteCmd.Flags().StringVar(&functionPromoteMessage, "message", "", "promotion message")
	functionsPromoteCmd.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "promote without prompting")
	functionsSetActiveCmd.Flags().IntVar(&functionVersion, "version", 0, "promoted version to set active")
	functionsSetActiveCmd.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "set active without prompting")
	functionsRestoreDraftCmd.Flags().IntVar(&functionVersion, "version", 0, "promoted version to restore into the draft")
	functionsRestoreDraftCmd.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "restore draft without prompting")
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
	registerTypedWriteFlags()
}

func registerTypedWriteFlags() {
	for _, command := range []*cobra.Command{
		secretsCreateCmd, secretsUpdateCmd, secretsDeleteCmd,
		aiProvidersCreateCmd, aiProvidersUpdateCmd, aiProvidersDeleteCmd,
		ontologiesCreateCmd, ontologiesUpdateCmd, ontologiesDeleteCmd,
		ontologyCloneCmd,
		ontologyFastLookupsCreateCmd, ontologyFastLookupsDeleteCmd,
		ontologyRecordsAddCmd, ontologyRecordsUpdateCmd, ontologyRecordsDeleteCmd,
		datasetsCreateCmd, datasetsUpdateCmd, datasetsDeleteCmd, datasetCloneCmd,
		transformsCreateCmd, transformsUpdateCmd, transformsRunCmd, transformsDeleteCmd, transformCloneCmd,
		functionsCreateCmd, functionsUpdateCmd, functionsDeployCmd, functionsRedeployCmd, functionsDeleteCmd, functionCloneCmd,
		foldersCreateCmd, foldersRenameCmd, foldersMoveCmd, foldersDeleteCmd,
		filesUploadCmd, filesPromoteDatasetCmd, filesDeleteCmd, filesRenameCmd, filesMoveCmd,
	} {
		command.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "run the write without prompting")
	}

	secretsCreateCmd.Flags().StringVar(&secretName, "name", "", "secret name")
	secretsCreateCmd.Flags().StringVar(&secretDescription, "description", "", "secret description")
	registerSecretValueFlags(secretsCreateCmd)
	secretsUpdateCmd.Flags().StringVar(&secretName, "name", "", "secret name")
	secretsUpdateCmd.Flags().StringVar(&secretDescription, "description", "", "secret description")
	registerSecretValueFlags(secretsUpdateCmd)

	aiProvidersCreateCmd.Flags().StringVar(&aiProviderName, "name", "", "AI provider name")
	aiProvidersCreateCmd.Flags().StringVar(&aiProviderType, "type", "", "AI provider type")
	aiProvidersCreateCmd.Flags().StringVar(&aiProviderEndpoint, "endpoint", "", "AI provider endpoint")
	aiProvidersCreateCmd.Flags().StringArrayVar(&aiProviderModels, "model", nil, "allowed model name; repeatable")
	registerAIProviderAPIKeyFlags(aiProvidersCreateCmd)
	aiProvidersUpdateCmd.Flags().StringVar(&aiProviderName, "name", "", "AI provider name")
	aiProvidersUpdateCmd.Flags().StringVar(&aiProviderEndpoint, "endpoint", "", "AI provider endpoint")
	aiProvidersUpdateCmd.Flags().StringArrayVar(&aiProviderModels, "model", nil, "allowed model name; repeatable")
	registerAIProviderAPIKeyFlags(aiProvidersUpdateCmd)

	datasetsCreateCmd.Flags().StringVar(&datasetName, "name", "", "dataset name")
	datasetsCreateCmd.Flags().StringVar(&datasetDescription, "description", "", "dataset description")
	datasetsCreateCmd.Flags().StringVar(&datasetSourceID, "source-id", "", "source id for source-backed dataset")
	datasetsCreateCmd.Flags().StringVar(&datasetSchemaMode, "schema-mode", "", "dataset schema mode")
	datasetsCreateCmd.Flags().StringArrayVar(&datasetColumns, "column", nil, "dataset column as name:type[:nullable][:primary]; repeatable")
	datasetsUpdateCmd.Flags().StringVar(&datasetName, "name", "", "dataset name")
	datasetsUpdateCmd.Flags().StringVar(&datasetDescription, "description", "", "dataset description")
	datasetsUpdateCmd.Flags().StringVar(&datasetSchemaMode, "schema-mode", "", "dataset schema mode")
	datasetsUpdateCmd.Flags().StringArrayVar(&datasetColumns, "column", nil, "dataset column as name:type[:nullable][:primary]; repeatable")

	registerOntologyWriteFlags(ontologiesCreateCmd)
	registerOntologyWriteFlags(ontologiesUpdateCmd)
	ontologyFastLookupsCreateCmd.Flags().StringArrayVar(&ontologyFastLookupFields, "field", nil, "ontology property to include in the fast lookup; repeatable")
	ontologyFastLookupsCreateCmd.Flags().StringVar(&ontologyFastLookupReason, "reason", "", "reason for the fast lookup")
	ontologyRecordsAddCmd.Flags().StringArrayVar(&ontologyRecordValues, "value", nil, "record value as key=value; repeatable")
	ontologyRecordsUpdateCmd.Flags().StringArrayVar(&ontologyRecordValues, "value", nil, "record value as key=value; repeatable")
	ontologyRecordsUpdateCmd.Flags().StringArrayVar(&ontologyRecordUpdateColumns, "update-column", nil, "ontology property to update; repeatable")
	ontologyRecordsDeleteCmd.Flags().StringArrayVar(&ontologyRecordValues, "value", nil, "record matcher as key=value; repeatable")

	transformsCreateCmd.Flags().StringVar(&transformName, "name", "", "transform name")
	registerTransformWriteFlags(transformsCreateCmd)
	transformsUpdateCmd.Flags().StringVar(&transformName, "name", "", "transform name")
	registerTransformWriteFlags(transformsUpdateCmd)

	functionsCreateCmd.Flags().StringVar(&functionName, "name", "", "function name")
	registerFunctionWriteFlags(functionsCreateCmd)
	functionsUpdateCmd.Flags().StringVar(&functionName, "name", "", "function name")
	registerFunctionWriteFlags(functionsUpdateCmd)

	foldersCreateCmd.Flags().StringVar(&folderName, "name", "", "folder name")
	foldersCreateCmd.Flags().StringVar(&folderParentID, "parent-id", "", "parent folder id")
	foldersRenameCmd.Flags().StringVar(&folderName, "name", "", "new folder name")
	foldersMoveCmd.Flags().StringVar(&folderNewParentID, "parent-id", "", "destination parent folder id; empty moves to project root")

	filesUploadCmd.Flags().StringVar(&filePath, "path", "", "local file path to upload")
	filesUploadCmd.Flags().StringVar(&platformFolderID, "folder-id", "", "destination folder id")
	filesPromoteDatasetCmd.Flags().StringVar(&datasetName, "name", "", "dataset name")
	filesPromoteDatasetCmd.Flags().StringVar(&datasetDescription, "description", "", "dataset description")
	filesPromoteDatasetCmd.Flags().IntVar(&platformSheetIndex, "sheet-index", 0, "tabular sheet index to promote")
	filesPromoteDatasetCmd.Flags().StringArrayVar(&fileColumnMappings, "column-map", nil, "column mapping as source:dataset:type[:nullable][:primary]; repeatable")
	filesRenameCmd.Flags().StringVar(&fileNewName, "name", "", "new file name")
	filesMoveCmd.Flags().StringVar(&fileNewFolderID, "folder-id", "", "destination folder id; empty moves to project root")
}

func registerSecretValueFlags(command *cobra.Command) {
	command.Flags().StringVar(&secretValue, "value", "", "secret value")
	command.Flags().StringVar(&secretValueEnv, "value-env", "", "environment variable containing the secret value")
	command.Flags().BoolVar(&secretValueStdin, "value-stdin", false, "read the secret value from stdin")
}

func registerAIProviderAPIKeyFlags(command *cobra.Command) {
	command.Flags().StringVar(&aiProviderAPIKey, "api-key", "", "AI provider API key")
	command.Flags().StringVar(&aiProviderAPIKeyEnv, "api-key-env", "", "environment variable containing the AI provider API key")
	command.Flags().BoolVar(&aiProviderAPIKeyStdin, "api-key-stdin", false, "read the AI provider API key from stdin")
}

func registerOntologyWriteFlags(command *cobra.Command) {
	command.Flags().StringVar(&ontologyAPIName, "api-name", "", "ontology API name")
	command.Flags().StringVar(&ontologyDisplayName, "display-name", "", "ontology display name")
	command.Flags().StringVar(&ontologyPluralName, "plural-display-name", "", "ontology plural display name")
	command.Flags().StringVar(&ontologyDescription, "description", "", "ontology description")
	command.Flags().StringVar(&ontologyPrimaryKey, "primary-key", "", "ontology primary key property")
	command.Flags().StringVar(&ontologyTableName, "table-name", "", "backing table name")
	command.Flags().StringVar(&ontologySchemaName, "schema-name", "", "backing schema name")
	command.Flags().StringVar(&ontologyStatus, "status", "", "ontology status")
	command.Flags().StringVar(&ontologyIcon, "icon", "", "ontology icon")
	command.Flags().StringVar(&ontologyColor, "color", "", "ontology color")
	command.Flags().StringArrayVar(&ontologyPropertiesJSON, "property-json", nil, "ontology property JSON object; repeatable")
	command.Flags().StringArrayVar(&ontologyLinksJSON, "link-json", nil, "ontology link JSON object; repeatable")
}

func registerTransformWriteFlags(command *cobra.Command) {
	command.Flags().StringVar(&transformDescription, "description", "", "transform description")
	command.Flags().StringVar(&transformGraphJSON, "graph-json", "", "transform graph JSON")
	command.Flags().StringVar(&transformGraphFile, "graph-file", "", "path to transform graph JSON file")
	command.Flags().StringVar(&transformScheduleCron, "schedule-cron", "", "transform schedule cron")
	command.Flags().StringVar(&transformTriggerMode, "trigger-mode", "", "transform trigger mode")
}

func registerFunctionWriteFlags(command *cobra.Command) {
	command.Flags().StringVar(&functionDescription, "description", "", "function description")
	command.Flags().StringVar(&functionLanguage, "language", "", "function language")
	command.Flags().StringVar(&functionEntryPoint, "entry-point", "", "function entry point")
	command.Flags().IntVar(&functionTimeoutSeconds, "timeout-seconds", 0, "function timeout in seconds")
	command.Flags().StringVar(&functionMemory, "memory", "", "function memory request")
	command.Flags().StringVar(&functionCPU, "cpu", "", "function CPU request")
	command.Flags().StringArrayVar(&functionFiles, "file", nil, "function file as target-path=local-path; repeatable")
	command.Flags().StringArrayVar(&functionDependencies, "dependency", nil, "function dependency as name[:version]; repeatable")
	command.Flags().StringArrayVar(&functionProviderIDs, "provider-id", nil, "AI provider id allowed for the function; repeatable")
	command.Flags().StringArrayVar(&functionOntologyIDs, "ontology-id", nil, "ontology id writable by the function; repeatable")
	command.Flags().StringArrayVar(&functionReadOnlyOntologyIDs, "read-only-ontology-id", nil, "ontology id readable by the function; repeatable")
	command.Flags().StringArrayVar(&functionProviderConfigs, "provider-config", nil, "provider model binding as provider-id=model; repeatable")
	command.Flags().StringArrayVar(&functionSecretBindings, "secret-binding", nil, "secret binding as NAME=secret-id[:target]; target defaults to ENV; repeatable")
	command.Flags().IntVar(&functionDefaultMaxTokens, "default-max-tokens", 0, "default max tokens for function AI calls")
	command.Flags().Float64Var(&functionDefaultTemperature, "default-temperature", 0, "default temperature for function AI calls")
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

func platformExportCommand(use, short string, read func(context.Context, *platformSession, string) (any, error)) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
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
			value, err := read(ctx, session, args[0])
			if err != nil {
				return err
			}
			raw, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				return err
			}
			raw = append(raw, '\n')
			if strings.TrimSpace(platformExportOutPath) == "" {
				_, err = cmd.OutOrStdout().Write(raw)
				return err
			}
			return os.WriteFile(filepath.Clean(platformExportOutPath), raw, 0600)
		},
	}
}

func readPlatformOntologyDetail(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "ontology", id)
	if err != nil {
		return nil, nil, err
	}
	ontology, err := session.Client.Ontology(ctx, session.Host.DefaultProjectID, resolvedID)
	if err != nil {
		return nil, nil, err
	}
	related := platformRelatedLineage(ctx, session, session.Host.DefaultProjectID, ontology.ID, "ontology_type")
	rows := [][]any{{"id", ontology.ID}, {"api_name", ontology.APIName}, {"display_name", ontology.DisplayName}, {"table", ontology.TableName}, {"status", ontology.Status}, {"properties", len(ontology.Properties)}, {"links", len(ontology.Links)}, {"upstream", len(related.Upstream)}, {"downstream", len(related.Downstream)}}
	return platformOntologyDescribe{Ontology: *ontology, Related: related}, tableResult([]string{"field", "value"}, rows), nil
}

func readPlatformDatasetDetail(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "dataset", id)
	if err != nil {
		return nil, nil, err
	}
	dataset, err := session.Client.Dataset(ctx, session.Host.DefaultProjectID, resolvedID)
	if err != nil {
		return nil, nil, err
	}
	related := platformRelatedLineage(ctx, session, session.Host.DefaultProjectID, dataset.ID, "dataset")
	rows := [][]any{{"id", dataset.ID}, {"name", dataset.Name}, {"schema_mode", dataset.SchemaMode}, {"row_count", dataset.RowCount}, {"size_bytes", dataset.SizeBytes}, {"columns", len(dataset.Schema)}, {"upstream", len(related.Upstream)}, {"downstream", len(related.Downstream)}}
	return platformDatasetDescribe{Dataset: *dataset, Related: related}, tableResult([]string{"field", "value"}, rows), nil
}

func readPlatformTransformDetail(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	transform, err := resolveTransform(ctx, session, session.Host.DefaultProjectID, id)
	if err != nil {
		return nil, nil, err
	}
	related := platformRelatedLineage(ctx, session, session.Host.DefaultProjectID, transform.ID, "transform")
	rows := [][]any{{"id", transform.ID}, {"name", transform.Name}, {"trigger_mode", transform.TriggerMode}, {"schedule", transform.ScheduleCron}, {"updated_at", transform.UpdatedAt}, {"upstream", len(related.Upstream)}, {"downstream", len(related.Downstream)}}
	return platformTransformDescribe{Transform: *transform, Related: related}, tableResult([]string{"field", "value"}, rows), nil
}

func readPlatformFunctionDetail(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	resolvedID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "function", id)
	if err != nil {
		return nil, nil, err
	}
	fn, err := session.Client.Function(ctx, session.Host.DefaultProjectID, resolvedID, platformFields)
	if err != nil {
		return nil, nil, err
	}
	related := platformRelatedLineage(ctx, session, session.Host.DefaultProjectID, fn.ID, "ontology_function")
	rows := [][]any{{"id", fn.ID}, {"name", fn.Name}, {"language", fn.Language}, {"entry_point", fn.EntryPoint}, {"deployed", fn.IsDeployed}, {"files", len(fn.Files)}, {"dependencies", len(fn.Dependencies)}, {"upstream", len(related.Upstream)}, {"downstream", len(related.Downstream)}}
	return platformFunctionDescribe{Function: *fn, Related: related}, tableResult([]string{"field", "value"}, rows), nil
}

func readPlatformFileDetail(ctx context.Context, session *platformSession, id string) (any, *output.QueryResult, error) {
	file, err := resolveProjectFile(ctx, session, session.Host.DefaultProjectID, id)
	if err != nil {
		return nil, nil, err
	}
	folderID := ""
	if file.FolderID != nil {
		folderID = *file.FolderID
	}
	datasetID := ""
	if file.DatasetID != nil {
		datasetID = *file.DatasetID
	}
	rows := [][]any{
		{"id", file.ID},
		{"name", file.Name},
		{"folder_id", folderID},
		{"mime_type", file.MIMEType},
		{"size_bytes", file.SizeBytes},
		{"is_tabular", file.IsTabular},
		{"dataset_id", datasetID},
		{"created_at", file.CreatedAt},
		{"updated_at", file.UpdatedAt},
	}
	related := platformRelatedLineage(ctx, session, session.Host.DefaultProjectID, file.ID, "project_file")
	rows = append(rows, []any{"upstream", len(related.Upstream)}, []any{"downstream", len(related.Downstream)})
	return platformFileDescribe{ProjectFile: *file, Related: related}, tableResult([]string{"field", "value"}, rows), nil
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
	org, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return err
	}
	session.Host.DefaultOrgID = org.ID
	session.Host.DefaultOrgName = org.Name
	session.Host.DefaultProjectID = project.ID
	session.Host.DefaultProjectName = project.Name
	session.Client.SetWorkspaceContext(org.ID, project.ID)
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

func typedResourceWriteCommand(use, short, operationKind, resource, action string, buildPayload func(*cobra.Command) (map[string]any, error)) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			switch operationKind {
			case "create":
				return cobra.NoArgs(cmd, args)
			case "update", "delete":
				return cobra.ExactArgs(1)(cmd, args)
			case "action":
				if action == "upload" {
					return cobra.NoArgs(cmd, args)
				}
				return cobra.ExactArgs(1)(cmd, args)
			default:
				return fmt.Errorf("unsupported resource write operation %q", operationKind)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := buildPayload(cmd)
			if err != nil {
				return err
			}
			input := genericResourceWriteInput{Resource: resource, Action: operationKind}
			if operationKind == "action" {
				input.Action = action
			}
			if len(args) > 0 {
				input.ID = args[0]
			}
			return runPlatformResourceWrite(cmd, input, operationKind, payload)
		},
	}
}

func platformFunctionDeployCommand(use, short, mutation, action string) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlatformFunctionDeployWrite(cmd, args[0], mutation, action)
		},
	}
}

func withExample(command *cobra.Command, example string) *cobra.Command {
	command.Example = example
	return command
}

func runGenericResourceWrite(cmd *cobra.Command, args []string, operationKind string) error {
	input := genericResourceInputFromArgs(args, operationKind)
	payload, err := readPlatformPayload(cmd)
	if err != nil {
		return err
	}
	return runPlatformResourceWrite(cmd, input, operationKind, payload)
}

func runPlatformResourceWrite(cmd *cobra.Command, input genericResourceWriteInput, operationKind string, payload map[string]any) error {
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
	if strings.TrimSpace(input.ID) != "" {
		if normalizePlatformResourceToken(input.Resource) == "file" && normalizePlatformResourceToken(input.Action) == "promote_to_dataset" {
			if resolvedID, err := resolvePlatformResourceID(ctx, session, project.ID, input.Resource, input.ID); err == nil {
				input.ID = resolvedID
			} else {
				input.ID = strings.TrimSpace(input.ID)
			}
		} else {
			resolvedID, err := resolvePlatformResourceID(ctx, session, project.ID, input.Resource, input.ID)
			if err != nil {
				return err
			}
			input.ID = resolvedID
		}
	}
	if err := prepareTypedResourcePayload(ctx, session, project.ID, input, payload); err != nil {
		return err
	}
	spec, variables, err := buildGenericResourceVariables(project.ID, input, payload)
	if err != nil {
		return err
	}
	if !platformWriteYes {
		approved, err := confirmPlatformResourceWrite(cmd.InOrStdin(), cmd.ErrOrStderr(), spec, project.Name, payload)
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

func runPlatformFunctionLifecycleWrite(cmd *cobra.Command, functionRef, action string) error {
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
	functionID, err := resolvePlatformResourceID(ctx, session, project.ID, "function", functionRef)
	if err != nil {
		return err
	}
	if !platformWriteYes {
		approved, err := confirmPlatformFunctionLifecycle(cmd.InOrStdin(), cmd.ErrOrStderr(), action, functionRef, project.Name, functionVersion)
		if err != nil {
			return err
		}
		if !approved {
			return fmt.Errorf("write cancelled")
		}
	}
	switch action {
	case "promote":
		version, err := session.Client.PromoteObject(ctx, project.ID, functionID, "function", functionPromoteMessage)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "functions.promote", version)
		}
		return out.WriteQueryResult(tableResult([]string{"field", "value"}, [][]any{
			{"function_id", version.ObjectID},
			{"version", version.Version},
			{"active", true},
			{"message", version.Message},
			{"promoted_by", version.PromotedBy},
			{"created_at", version.CreatedAt},
		}))
	case "set-active":
		active, err := session.Client.SetActiveObjectVersion(ctx, project.ID, functionID, "function", functionVersion)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "functions.set-active", active)
		}
		return out.WriteQueryResult(tableResult([]string{"field", "value"}, [][]any{
			{"function_id", active.ObjectID},
			{"version", active.Version},
			{"activated_by", active.ActivatedBy},
			{"activated_at", active.ActivatedAt},
		}))
	case "restore-draft":
		fn, err := session.Client.RestoreFunctionVersionToDraft(ctx, project.ID, functionID, functionVersion)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "functions.restore-draft", fn)
		}
		return out.WriteQueryResult(tableResult([]string{"field", "value"}, [][]any{
			{"function_id", fn.ID},
			{"name", fn.Name},
			{"restored_version", functionVersion},
			{"files", len(fn.Files)},
			{"dependencies", len(fn.Dependencies)},
			{"updated_at", fn.UpdatedAt},
		}))
	default:
		return fmt.Errorf("unsupported function lifecycle action %q", action)
	}
}

type platformFunctionDeployOutput struct {
	FunctionID    string `json:"functionId"`
	FunctionName  string `json:"functionName"`
	Operation     string `json:"operation"`
	ActiveVersion int    `json:"activeVersion"`
	Deployed      bool   `json:"deployed"`
}

func runPlatformFunctionDeployWrite(cmd *cobra.Command, functionRef, mutation, action string) error {
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
	functionID, err := resolvePlatformResourceID(ctx, session, project.ID, "function", functionRef)
	if err != nil {
		return err
	}
	if !platformWriteYes {
		approved, err := confirmPlatformFunctionLifecycle(cmd.InOrStdin(), cmd.ErrOrStderr(), action, functionRef, project.Name, 0)
		if err != nil {
			return err
		}
		if !approved {
			return fmt.Errorf("write cancelled")
		}
	}
	if _, err := session.Client.PlatformMutation(ctx, mutation, map[string]any{"projectId": project.ID, "id": functionID}); err != nil {
		return friendlyFunctionDeployError(functionRef, err)
	}
	active, err := session.Client.ActiveProdVersion(ctx, project.ID, functionID, "function")
	if err != nil {
		return err
	}
	fn, err := session.Client.Function(ctx, project.ID, functionID, nil)
	if err != nil {
		return err
	}
	result := platformFunctionDeployOutput{
		FunctionID:   functionID,
		FunctionName: fn.Name,
		Operation:    action,
		Deployed:     fn.IsDeployed,
	}
	if active != nil {
		result.ActiveVersion = active.Version
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, "functions."+action, result)
	}
	return out.WriteQueryResult(tableResult([]string{"field", "value"}, [][]any{
		{"function_id", result.FunctionID},
		{"name", result.FunctionName},
		{"operation", result.Operation},
		{"active_version", result.ActiveVersion},
		{"deployed", result.Deployed},
	}))
}

func friendlyFunctionDeployError(functionRef string, err error) error {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "no active version") || strings.Contains(message, "promote") && strings.Contains(message, "active") {
		return fmt.Errorf("function %s has no active version. Promote it first:\n  whodb-cli functions promote %s --message \"initial version\"\n  whodb-cli functions deploy %s", functionRef, functionRef, functionRef)
	}
	return err
}

func prepareTypedResourcePayload(ctx context.Context, session *platformSession, projectID string, input genericResourceWriteInput, payload map[string]any) error {
	resource := normalizePlatformResourceToken(input.Resource)
	action := normalizePlatformResourceToken(input.Action)
	if action != "update" || resource != "transform" {
		return nil
	}
	existing, err := resolveTransform(ctx, session, projectID, input.ID)
	if err != nil {
		return err
	}
	complete := map[string]any{
		"name":         existing.Name,
		"description":  existing.Description,
		"graphJson":    existing.GraphJSON,
		"scheduleCron": existing.ScheduleCron,
		"triggerMode":  existing.TriggerMode,
	}
	for key, value := range payload {
		complete[key] = value
	}
	for key, value := range complete {
		payload[key] = value
	}
	return nil
}

type genericResourceWriteInput struct {
	Resource string
	Action   string
	ID       string
}

type platformKeyedGenericWriteSpec struct {
	Key string `json:"key"`
	platform.GenericWriteSpec
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

func emptyTypedPayload(cmd *cobra.Command) (map[string]any, error) {
	return map[string]any{}, nil
}

func buildSecretCreatePayload(cmd *cobra.Command) (map[string]any, error) {
	name := strings.TrimSpace(secretName)
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	value, err := readSensitiveFlagValue(cmd, secretValue, secretValueEnv, secretValueStdin, "secret value")
	if err != nil {
		return nil, err
	}
	payload := map[string]any{"name": name, "value": value}
	if cmd.Flags().Changed("description") {
		payload["description"] = secretDescription
	}
	return payload, nil
}

func buildSecretUpdatePayload(cmd *cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	if cmd.Flags().Changed("name") {
		name := strings.TrimSpace(secretName)
		if name == "" {
			return nil, fmt.Errorf("--name cannot be empty")
		}
		payload["name"] = name
	}
	if cmd.Flags().Changed("description") {
		payload["description"] = secretDescription
	}
	if cmd.Flags().Changed("value") || secretValueEnv != "" || secretValueStdin {
		value, err := readSensitiveFlagValue(cmd, secretValue, secretValueEnv, secretValueStdin, "secret value")
		if err != nil {
			return nil, err
		}
		payload["value"] = value
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update; pass --name, --description, or a value flag")
	}
	return payload, nil
}

func buildAIProviderCreatePayload(cmd *cobra.Command) (map[string]any, error) {
	name := strings.TrimSpace(aiProviderName)
	providerType := strings.TrimSpace(aiProviderType)
	endpoint := strings.TrimSpace(aiProviderEndpoint)
	if name == "" || providerType == "" || endpoint == "" {
		return nil, fmt.Errorf("--name, --type, and --endpoint are required")
	}
	apiKey, err := readSensitiveFlagValue(cmd, aiProviderAPIKey, aiProviderAPIKeyEnv, aiProviderAPIKeyStdin, "AI provider API key")
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"name":         name,
		"providerType": providerType,
		"endpoint":     endpoint,
		"apiKey":       apiKey,
	}
	if len(aiProviderModels) > 0 {
		payload["models"] = normalizedStringList(aiProviderModels)
	}
	return payload, nil
}

func buildAIProviderUpdatePayload(cmd *cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	if cmd.Flags().Changed("name") {
		name := strings.TrimSpace(aiProviderName)
		if name == "" {
			return nil, fmt.Errorf("--name cannot be empty")
		}
		payload["name"] = name
	}
	if cmd.Flags().Changed("endpoint") {
		endpoint := strings.TrimSpace(aiProviderEndpoint)
		if endpoint == "" {
			return nil, fmt.Errorf("--endpoint cannot be empty")
		}
		payload["endpoint"] = endpoint
	}
	if cmd.Flags().Changed("model") {
		payload["models"] = normalizedStringList(aiProviderModels)
	}
	if cmd.Flags().Changed("api-key") || aiProviderAPIKeyEnv != "" || aiProviderAPIKeyStdin {
		apiKey, err := readSensitiveFlagValue(cmd, aiProviderAPIKey, aiProviderAPIKeyEnv, aiProviderAPIKeyStdin, "AI provider API key")
		if err != nil {
			return nil, err
		}
		payload["apiKey"] = apiKey
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update; pass --name, --endpoint, --model, or an API key flag")
	}
	return payload, nil
}

func buildDatasetCreatePayload(cmd *cobra.Command) (map[string]any, error) {
	name := strings.TrimSpace(datasetName)
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	payload := map[string]any{"name": name}
	if cmd.Flags().Changed("description") {
		payload["description"] = datasetDescription
	}
	if strings.TrimSpace(datasetSourceID) != "" {
		payload["sourceId"] = strings.TrimSpace(datasetSourceID)
	}
	if strings.TrimSpace(datasetSchemaMode) != "" {
		payload["schemaMode"] = strings.TrimSpace(datasetSchemaMode)
	}
	columns, err := parseDatasetColumns(datasetColumns)
	if err != nil {
		return nil, err
	}
	if len(columns) > 0 {
		payload["columns"] = columns
	}
	return payload, nil
}

func buildDatasetUpdatePayload(cmd *cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	if cmd.Flags().Changed("name") {
		name := strings.TrimSpace(datasetName)
		if name == "" {
			return nil, fmt.Errorf("--name cannot be empty")
		}
		payload["name"] = name
	}
	if cmd.Flags().Changed("description") {
		payload["description"] = datasetDescription
	}
	if cmd.Flags().Changed("schema-mode") {
		payload["schemaMode"] = strings.TrimSpace(datasetSchemaMode)
	}
	if cmd.Flags().Changed("column") {
		columns, err := parseDatasetColumns(datasetColumns)
		if err != nil {
			return nil, err
		}
		payload["columns"] = columns
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update; pass --name, --description, --schema-mode, or --column")
	}
	return payload, nil
}

func buildOntologyCreatePayload(cmd *cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	for flag, value := range map[string]string{
		"api-name":            ontologyAPIName,
		"display-name":        ontologyDisplayName,
		"plural-display-name": ontologyPluralName,
		"primary-key":         ontologyPrimaryKey,
		"table-name":          ontologyTableName,
		"schema-name":         ontologySchemaName,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("--%s is required", flag)
		}
	}
	payload["apiName"] = strings.TrimSpace(ontologyAPIName)
	payload["displayName"] = strings.TrimSpace(ontologyDisplayName)
	payload["pluralDisplayName"] = strings.TrimSpace(ontologyPluralName)
	payload["primaryKey"] = strings.TrimSpace(ontologyPrimaryKey)
	payload["tableName"] = strings.TrimSpace(ontologyTableName)
	payload["schemaName"] = strings.TrimSpace(ontologySchemaName)
	payload["description"] = ontologyDescription
	payload["icon"] = defaultString(strings.TrimSpace(ontologyIcon), "table")
	payload["color"] = defaultString(strings.TrimSpace(ontologyColor), "#3366ff")
	if strings.TrimSpace(ontologyStatus) != "" {
		payload["status"] = strings.TrimSpace(ontologyStatus)
	}
	properties, err := parseJSONObjectFlags(ontologyPropertiesJSON, "property-json")
	if err != nil {
		return nil, err
	}
	if len(properties) == 0 {
		return nil, fmt.Errorf("--property-json is required at least once")
	}
	links, err := parseJSONObjectFlags(ontologyLinksJSON, "link-json")
	if err != nil {
		return nil, err
	}
	payload["properties"] = properties
	payload["links"] = links
	return payload, nil
}

func buildOntologyUpdatePayload(cmd *cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	addChangedStringPayload(cmd, payload, "api-name", "apiName", ontologyAPIName)
	addChangedStringPayload(cmd, payload, "display-name", "displayName", ontologyDisplayName)
	addChangedStringPayload(cmd, payload, "plural-display-name", "pluralDisplayName", ontologyPluralName)
	addChangedStringPayload(cmd, payload, "description", "description", ontologyDescription)
	addChangedStringPayload(cmd, payload, "primary-key", "primaryKey", ontologyPrimaryKey)
	addChangedStringPayload(cmd, payload, "table-name", "tableName", ontologyTableName)
	addChangedStringPayload(cmd, payload, "schema-name", "schemaName", ontologySchemaName)
	addChangedStringPayload(cmd, payload, "status", "status", ontologyStatus)
	addChangedStringPayload(cmd, payload, "icon", "icon", ontologyIcon)
	addChangedStringPayload(cmd, payload, "color", "color", ontologyColor)
	if cmd.Flags().Changed("property-json") {
		properties, err := parseJSONObjectFlags(ontologyPropertiesJSON, "property-json")
		if err != nil {
			return nil, err
		}
		payload["properties"] = properties
	}
	if cmd.Flags().Changed("link-json") {
		links, err := parseJSONObjectFlags(ontologyLinksJSON, "link-json")
		if err != nil {
			return nil, err
		}
		payload["links"] = links
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update; pass ontology fields, --property-json, or --link-json")
	}
	return payload, nil
}

func buildTransformCreatePayload(cmd *cobra.Command) (map[string]any, error) {
	name := strings.TrimSpace(transformName)
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	graphJSON, err := readTransformGraphJSON(cmd, true)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"name":         name,
		"description":  transformDescription,
		"graphJson":    graphJSON,
		"scheduleCron": transformScheduleCron,
		"triggerMode":  defaultString(strings.TrimSpace(transformTriggerMode), "manual"),
	}
	return payload, nil
}

func buildTransformUpdatePayload(cmd *cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	addChangedStringPayload(cmd, payload, "name", "name", transformName)
	addChangedStringPayload(cmd, payload, "description", "description", transformDescription)
	addChangedStringPayload(cmd, payload, "schedule-cron", "scheduleCron", transformScheduleCron)
	addChangedStringPayload(cmd, payload, "trigger-mode", "triggerMode", transformTriggerMode)
	if cmd.Flags().Changed("graph-json") || cmd.Flags().Changed("graph-file") {
		graphJSON, err := readTransformGraphJSON(cmd, false)
		if err != nil {
			return nil, err
		}
		payload["graphJson"] = graphJSON
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update; pass --name, --description, --graph-json, --graph-file, --schedule-cron, or --trigger-mode")
	}
	return payload, nil
}

func buildFunctionCreatePayload(cmd *cobra.Command) (map[string]any, error) {
	name := strings.TrimSpace(functionName)
	language := strings.TrimSpace(functionLanguage)
	entryPoint := strings.TrimSpace(functionEntryPoint)
	if name == "" || language == "" || entryPoint == "" {
		return nil, fmt.Errorf("--name, --language, and --entry-point are required")
	}
	files, err := parseFunctionFiles(functionFiles)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("--file is required at least once")
	}
	deps, err := parseFunctionDependencies(functionDependencies)
	if err != nil {
		return nil, err
	}
	providerConfigs, err := parseFunctionProviderConfigs(functionProviderConfigs)
	if err != nil {
		return nil, err
	}
	secretBindings, err := parseFunctionSecretBindings(functionSecretBindings)
	if err != nil {
		return nil, err
	}
	timeoutSeconds := functionTimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 30
	}
	payload := map[string]any{
		"name":           name,
		"description":    functionDescription,
		"language":       language,
		"entryPoint":     entryPoint,
		"timeoutSeconds": timeoutSeconds,
		"memory":         defaultString(strings.TrimSpace(functionMemory), "128Mi"),
		"cpu":            defaultString(strings.TrimSpace(functionCPU), "100m"),
		"files":          files,
		"dependencies":   deps,
	}
	addFunctionConfigPayload(cmd, payload)
	if len(providerConfigs) > 0 {
		payload["providerConfigs"] = providerConfigs
	}
	if len(secretBindings) > 0 {
		payload["secretBindings"] = secretBindings
	}
	return payload, nil
}

func buildFunctionUpdatePayload(cmd *cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	addChangedStringPayload(cmd, payload, "name", "name", functionName)
	addChangedStringPayload(cmd, payload, "description", "description", functionDescription)
	addChangedStringPayload(cmd, payload, "language", "language", functionLanguage)
	addChangedStringPayload(cmd, payload, "entry-point", "entryPoint", functionEntryPoint)
	addChangedStringPayload(cmd, payload, "memory", "memory", functionMemory)
	addChangedStringPayload(cmd, payload, "cpu", "cpu", functionCPU)
	if cmd.Flags().Changed("timeout-seconds") {
		if functionTimeoutSeconds <= 0 {
			return nil, fmt.Errorf("--timeout-seconds must be positive")
		}
		payload["timeoutSeconds"] = functionTimeoutSeconds
	}
	if cmd.Flags().Changed("file") {
		files, err := parseFunctionFiles(functionFiles)
		if err != nil {
			return nil, err
		}
		payload["files"] = files
	}
	if cmd.Flags().Changed("dependency") {
		deps, err := parseFunctionDependencies(functionDependencies)
		if err != nil {
			return nil, err
		}
		payload["dependencies"] = deps
	}
	if cmd.Flags().Changed("provider-config") {
		configs, err := parseFunctionProviderConfigs(functionProviderConfigs)
		if err != nil {
			return nil, err
		}
		payload["providerConfigs"] = configs
	}
	if cmd.Flags().Changed("secret-binding") {
		bindings, err := parseFunctionSecretBindings(functionSecretBindings)
		if err != nil {
			return nil, err
		}
		payload["secretBindings"] = bindings
	}
	addFunctionConfigPayload(cmd, payload)
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update; pass function fields, --file, --dependency, or configuration flags")
	}
	return payload, nil
}

func addFunctionConfigPayload(cmd *cobra.Command, payload map[string]any) {
	if cmd.Flags().Changed("provider-id") {
		payload["providerIds"] = normalizedStringList(functionProviderIDs)
	}
	if cmd.Flags().Changed("ontology-id") {
		payload["ontologyIds"] = normalizedStringList(functionOntologyIDs)
	}
	if cmd.Flags().Changed("read-only-ontology-id") {
		payload["readOnlyOntologyIds"] = normalizedStringList(functionReadOnlyOntologyIDs)
	}
	if cmd.Flags().Changed("default-max-tokens") {
		payload["defaultMaxTokens"] = functionDefaultMaxTokens
	}
	if cmd.Flags().Changed("default-temperature") {
		payload["defaultTemperature"] = functionDefaultTemperature
	}
}

func buildFolderCreatePayload(cmd *cobra.Command) (map[string]any, error) {
	name := strings.TrimSpace(folderName)
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	payload := map[string]any{"name": name}
	if strings.TrimSpace(folderParentID) != "" {
		payload["parentId"] = strings.TrimSpace(folderParentID)
	}
	return payload, nil
}

func buildFolderRenamePayload(cmd *cobra.Command) (map[string]any, error) {
	name := strings.TrimSpace(folderName)
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	return map[string]any{"name": name}, nil
}

func buildFolderMovePayload(cmd *cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	if cmd.Flags().Changed("parent-id") {
		payload["newParentId"] = strings.TrimSpace(folderNewParentID)
	}
	return payload, nil
}

func buildFileUploadPayload(cmd *cobra.Command) (map[string]any, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("--path is required")
	}
	payload := map[string]any{"file_path": strings.TrimSpace(filePath)}
	if strings.TrimSpace(platformFolderID) != "" {
		payload["folderId"] = strings.TrimSpace(platformFolderID)
	}
	return payload, nil
}

func buildFilePromoteDatasetPayload(cmd *cobra.Command) (map[string]any, error) {
	name := strings.TrimSpace(datasetName)
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	columnMap, err := parseFileColumnMappings(fileColumnMappings)
	if err != nil {
		return nil, err
	}
	if len(columnMap) == 0 {
		return nil, fmt.Errorf("--column-map is required at least once")
	}
	payload := map[string]any{
		"datasetName": name,
		"description": datasetDescription,
		"columnMap":   columnMap,
	}
	if cmd.Flags().Changed("sheet-index") {
		payload["sheetIndex"] = platformSheetIndex
	}
	return payload, nil
}

func buildFileRenamePayload(cmd *cobra.Command) (map[string]any, error) {
	name := strings.TrimSpace(fileNewName)
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	return map[string]any{"name": name}, nil
}

func buildFileMovePayload(cmd *cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	if cmd.Flags().Changed("folder-id") {
		payload["newFolderId"] = strings.TrimSpace(fileNewFolderID)
	}
	return payload, nil
}

func readSensitiveFlagValue(cmd *cobra.Command, directValue, envName string, useStdin bool, label string) (string, error) {
	if useStdin {
		body, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("reading %s from stdin: %w", label, err)
		}
		value := strings.TrimRight(string(body), "\r\n")
		if value == "" {
			return "", fmt.Errorf("%s from stdin cannot be empty", label)
		}
		return value, nil
	}
	if strings.TrimSpace(envName) != "" {
		value := os.Getenv(strings.TrimSpace(envName))
		if value == "" {
			return "", fmt.Errorf("environment variable %s for %s is empty or unset", strings.TrimSpace(envName), label)
		}
		return value, nil
	}
	if directValue != "" {
		return directValue, nil
	}
	return "", fmt.Errorf("%s requires stdin, env, or direct value flag", label)
}

func normalizedStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func filterSecrets(secrets []platform.ProjectSecret) []platform.ProjectSecret {
	return filterSlice(secrets, func(secret platform.ProjectSecret) bool {
		return matchesSubstring(secret.Name, platformFilterName)
	})
}

func filterAIProviders(providers []platform.AIProvider) []platform.AIProvider {
	return filterSlice(providers, func(provider platform.AIProvider) bool {
		return matchesSubstring(provider.Name, platformFilterName) && matchesEqual(provider.ProviderType, platformFilterType)
	})
}

func filterOntologies(ontologies []platform.Ontology) []platform.Ontology {
	return filterSlice(ontologies, func(ontology platform.Ontology) bool {
		return (matchesSubstring(ontology.DisplayName, platformFilterName) || matchesSubstring(ontology.APIName, platformFilterName)) && matchesEqual(ontology.Status, platformFilterStatus)
	})
}

func filterDatasets(datasets []platform.Dataset) []platform.Dataset {
	return filterSlice(datasets, func(dataset platform.Dataset) bool {
		return matchesSubstring(dataset.Name, platformFilterName) && matchesEqual(dataset.SchemaMode, platformFilterSchemaMode)
	})
}

func filterTransforms(transforms []platform.Transform) []platform.Transform {
	return filterSlice(transforms, func(transform platform.Transform) bool {
		return matchesSubstring(transform.Name, platformFilterName) && matchesEqual(transform.TriggerMode, platformFilterType)
	})
}

func filterFunctions(functions []platform.Function) []platform.Function {
	return filterSlice(functions, func(fn platform.Function) bool {
		return matchesSubstring(fn.Name, platformFilterName) && matchesEqual(fn.Language, platformFilterType) && matchesBool(fn.IsDeployed, platformFilterDeployed)
	})
}

func filterFolderContents(contents *platform.FolderContents) *platform.FolderContents {
	if contents == nil {
		return nil
	}
	filtered := *contents
	filtered.Folders = nil
	filtered.Files = nil
	if matchesEntryKind("folder") {
		filtered.Folders = filterFolders(contents.Folders)
	}
	if matchesEntryKind("file") {
		filtered.Files = filterSlice(contents.Files, func(file platform.ProjectFile) bool {
			return matchesSubstring(file.Name, platformFilterName) && matchesSubstring(file.MIMEType, platformFilterMIMEType)
		})
	}
	return &filtered
}

func filterFolders(folders []platform.ProjectFolder) []platform.ProjectFolder {
	return filterSlice(folders, func(folder platform.ProjectFolder) bool {
		return matchesSubstring(folder.Name, platformFilterName)
	})
}

func filterSlice[T any](values []T, keep func(T) bool) []T {
	filtered := make([]T, 0, len(values))
	for _, value := range values {
		if keep(value) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func matchesSubstring(value, filter string) bool {
	needle := strings.TrimSpace(filter)
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func matchesEqual(value, filter string) bool {
	expected := strings.TrimSpace(filter)
	if expected == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(value), expected)
}

func matchesEntryKind(kind string) bool {
	return matchesEqual(kind, platformFilterKind)
}

func matchesBool(value bool, filter string) bool {
	expected := strings.TrimSpace(filter)
	if expected == "" {
		return true
	}
	switch strings.ToLower(expected) {
	case "true", "yes", "1":
		return value
	case "false", "no", "0":
		return !value
	default:
		return false
	}
}

func parseDatasetColumns(values []string) ([]map[string]any, error) {
	columns := make([]map[string]any, 0, len(values))
	for _, value := range values {
		parts := strings.Split(value, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("dataset column %q must be name:type[:nullable][:primary]", value)
		}
		name := strings.TrimSpace(parts[0])
		typ := strings.TrimSpace(parts[1])
		if name == "" || typ == "" {
			return nil, fmt.Errorf("dataset column %q must include a name and type", value)
		}
		column := map[string]any{
			"name":       name,
			"type":       typ,
			"isNullable": false,
			"isPrimary":  false,
		}
		for _, option := range parts[2:] {
			switch strings.ToLower(strings.TrimSpace(option)) {
			case "", "required", "notnull", "not_null":
				column["isNullable"] = false
			case "nullable", "null":
				column["isNullable"] = true
			case "primary", "pk":
				column["isPrimary"] = true
			default:
				return nil, fmt.Errorf("unsupported dataset column option %q in %q", option, value)
			}
		}
		columns = append(columns, column)
	}
	return columns, nil
}

func parseJSONObjectFlags(values []string, label string) ([]map[string]any, error) {
	objects := make([]map[string]any, 0, len(values))
	for _, value := range values {
		var object map[string]any
		if err := json.Unmarshal([]byte(value), &object); err != nil {
			return nil, fmt.Errorf("invalid --%s JSON object: %w", label, err)
		}
		if len(object) == 0 {
			return nil, fmt.Errorf("--%s cannot be an empty object", label)
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func readTransformGraphJSON(cmd *cobra.Command, useDefault bool) (string, error) {
	if cmd.Flags().Changed("graph-json") && cmd.Flags().Changed("graph-file") {
		return "", fmt.Errorf("pass only one of --graph-json or --graph-file")
	}
	if cmd.Flags().Changed("graph-file") {
		raw, err := os.ReadFile(strings.TrimSpace(transformGraphFile))
		if err != nil {
			return "", fmt.Errorf("read --graph-file: %w", err)
		}
		graph := strings.TrimSpace(string(raw))
		if graph == "" {
			return "", fmt.Errorf("--graph-file cannot be empty")
		}
		return graph, nil
	}
	if cmd.Flags().Changed("graph-json") {
		graph := strings.TrimSpace(transformGraphJSON)
		if graph == "" {
			return "", fmt.Errorf("--graph-json cannot be empty")
		}
		return graph, nil
	}
	if useDefault {
		return `{"nodes":[],"edges":[]}`, nil
	}
	return "", fmt.Errorf("--graph-json or --graph-file is required")
}

func parseFunctionFileInputs(values []string) ([]platform.FunctionFile, error) {
	files := make([]platform.FunctionFile, 0, len(values))
	for _, value := range values {
		target, localPath, ok := strings.Cut(value, "=")
		target = strings.TrimSpace(target)
		localPath = strings.TrimSpace(localPath)
		if !ok || target == "" || localPath == "" {
			return nil, fmt.Errorf("--file must be target-path=local-path")
		}
		raw, err := os.ReadFile(localPath)
		if err != nil {
			return nil, fmt.Errorf("read function file %q: %w", localPath, err)
		}
		files = append(files, platform.FunctionFile{Path: target, Content: string(raw)})
	}
	return files, nil
}

func parseFunctionFiles(values []string) ([]map[string]any, error) {
	inputs, err := parseFunctionFileInputs(values)
	if err != nil {
		return nil, err
	}
	files := make([]map[string]any, len(inputs))
	for i, input := range inputs {
		files[i] = map[string]any{"path": input.Path, "content": input.Content}
	}
	return files, nil
}

func parseFunctionDependencies(values []string) ([]map[string]any, error) {
	deps := make([]map[string]any, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value)
		version := ""
		if before, after, ok := strings.Cut(value, ":"); ok {
			name = strings.TrimSpace(before)
			version = strings.TrimSpace(after)
		}
		if name == "" {
			return nil, fmt.Errorf("--dependency must include a name")
		}
		deps = append(deps, map[string]any{"name": name, "version": version})
	}
	return deps, nil
}

func parseFunctionProviderConfigs(values []string) ([]map[string]any, error) {
	configs := make([]map[string]any, 0, len(values))
	for _, value := range values {
		providerID, model, ok := strings.Cut(value, "=")
		providerID = strings.TrimSpace(providerID)
		model = strings.TrimSpace(model)
		if !ok || providerID == "" || model == "" {
			return nil, fmt.Errorf("--provider-config must be provider-id=model")
		}
		configs = append(configs, map[string]any{"providerId": providerID, "model": model})
	}
	return configs, nil
}

func parseFunctionSecretBindings(values []string) ([]map[string]any, error) {
	bindings := make([]map[string]any, 0, len(values))
	for _, value := range values {
		name, rest, ok := strings.Cut(value, "=")
		name = strings.TrimSpace(name)
		rest = strings.TrimSpace(rest)
		if !ok || name == "" || rest == "" {
			return nil, fmt.Errorf("--secret-binding must be NAME=secret-id[:target]")
		}
		secretID := rest
		target := "ENV"
		if before, after, ok := strings.Cut(rest, ":"); ok {
			secretID = strings.TrimSpace(before)
			target = strings.ToUpper(strings.TrimSpace(after))
		}
		if secretID == "" {
			return nil, fmt.Errorf("--secret-binding must include a secret id")
		}
		if target == "" {
			target = "ENV"
		}
		bindings = append(bindings, map[string]any{"name": name, "secretId": secretID, "target": target})
	}
	return bindings, nil
}

func parseFunctionDependencyNames(values []string) ([]string, error) {
	deps, err := parseFunctionDependencies(values)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(deps))
	for i, dep := range deps {
		name, _ := dep["name"].(string)
		names[i] = name
	}
	return names, nil
}

func parseFileColumnMappings(values []string) ([]map[string]any, error) {
	mappings := make([]map[string]any, 0, len(values))
	for _, value := range values {
		parts := strings.Split(value, ":")
		if len(parts) < 3 {
			return nil, fmt.Errorf("--column-map %q must be source:dataset:type[:nullable][:primary]", value)
		}
		sourceColumn := strings.TrimSpace(parts[0])
		datasetColumn := strings.TrimSpace(parts[1])
		dataType := strings.TrimSpace(parts[2])
		if sourceColumn == "" || datasetColumn == "" || dataType == "" {
			return nil, fmt.Errorf("--column-map %q must include source, dataset, and type", value)
		}
		mapping := map[string]any{
			"sourceColumn":  sourceColumn,
			"datasetColumn": datasetColumn,
			"dataType":      dataType,
			"isNullable":    false,
			"isPrimary":     false,
		}
		for _, option := range parts[3:] {
			switch strings.ToLower(strings.TrimSpace(option)) {
			case "", "required", "notnull", "not_null":
				mapping["isNullable"] = false
			case "nullable", "null":
				mapping["isNullable"] = true
			case "primary", "pk":
				mapping["isPrimary"] = true
			default:
				return nil, fmt.Errorf("unsupported --column-map option %q in %q", option, value)
			}
		}
		mappings = append(mappings, mapping)
	}
	return mappings, nil
}

func buildOntologyFastLookupCreatePayload(cmd *cobra.Command) (map[string]any, error) {
	fields := normalizedStringList(ontologyFastLookupFields)
	if len(fields) == 0 {
		return nil, fmt.Errorf("--field is required")
	}
	payload := map[string]any{"fields": fields}
	if cmd.Flags().Changed("reason") {
		payload["reason"] = strings.TrimSpace(ontologyFastLookupReason)
	}
	return payload, nil
}

func buildOntologyRecordAddPayload(cmd *cobra.Command) (map[string]any, error) {
	values, err := parseOntologyRecordValues(ontologyRecordValues)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("--value is required")
	}
	return map[string]any{"values": values}, nil
}

func buildOntologyRecordUpdatePayload(cmd *cobra.Command) (map[string]any, error) {
	values, err := parseOntologyRecordValues(ontologyRecordValues)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("--value is required")
	}
	updatedColumns := make([]string, 0, len(ontologyRecordUpdateColumns))
	for _, column := range ontologyRecordUpdateColumns {
		column = strings.TrimSpace(column)
		if column == "" {
			return nil, fmt.Errorf("--update-column cannot be empty")
		}
		updatedColumns = append(updatedColumns, column)
	}
	if len(updatedColumns) == 0 {
		return nil, fmt.Errorf("--update-column is required")
	}
	return map[string]any{"values": values, "updatedColumns": updatedColumns}, nil
}

func buildOntologyRecordDeletePayload(cmd *cobra.Command) (map[string]any, error) {
	values, err := parseOntologyRecordValues(ontologyRecordValues)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("--value is required")
	}
	return map[string]any{"values": values}, nil
}

func parseOntologyRecordValues(values []string) ([]map[string]any, error) {
	records := make([]map[string]any, 0, len(values))
	for _, value := range values {
		key, recordValue, ok := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("--value must be key=value")
		}
		records = append(records, map[string]any{"key": key, "value": strings.TrimSpace(recordValue)})
	}
	return records, nil
}

func functionExecutionTable(result *platform.FunctionExecutionResult) *output.QueryResult {
	errorText := ""
	if result.Error != nil {
		errorText = *result.Error
	}
	outputText := ""
	if result.Output != nil {
		outputText = *result.Output
	}
	return tableResult([]string{"field", "value"}, [][]any{
		{"success", result.Success},
		{"duration_ms", result.DurationMS},
		{"output", outputText},
		{"logs", result.Logs},
		{"error", errorText},
	})
}

func addChangedStringPayload(cmd *cobra.Command, payload map[string]any, flag, key, value string) {
	if cmd.Flags().Changed(flag) {
		payload[key] = strings.TrimSpace(value)
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
				if strings.TrimSpace(firstResourceString(payload, "fileId")) == "" {
					payload["fileId"] = id
				}
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
		addConfirmDeletionVariable(spec.Mutation, variables)
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
		if spec.NeedsID {
			switch spec.Mutation {
			case "OntologyAddRow", "OntologyUpdateRow", "OntologyDeleteRow":
				variables["entityId"] = id
			default:
				variables["id"] = id
			}
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

func addConfirmDeletionVariable(mutation string, variables map[string]any) {
	switch mutation {
	case "DeleteOntology", "DeleteProjectFolder":
		variables["confirmDeletion"] = true
	}
}

func confirmPlatformResourceWrite(stdin io.Reader, stderr io.Writer, spec platform.GenericWriteSpec, projectName string, payload map[string]any) (bool, error) {
	summary := platformResourceWriteSummary(spec, payload)
	if summary == "" {
		summary = fmt.Sprintf("%s %s", titlePlatformAction(spec.Action), spec.Resource)
	}
	if _, err := fmt.Fprintf(stderr, "%s in project %s? [y/N]: ", summary, projectName); err != nil {
		return false, err
	}
	var answer string
	if _, err := fmt.Fscan(stdin, &answer); err != nil && err != io.EOF {
		return false, err
	}
	return isAffirmativeConfirmation(answer), nil
}

func platformResourceWriteSummary(spec platform.GenericWriteSpec, payload map[string]any) string {
	action := titlePlatformAction(spec.Action)
	resource := strings.ReplaceAll(spec.Resource, "_", " ")
	switch spec.Mutation {
	case "PromoteFileToDataset":
		if name := firstResourceString(payload, "datasetName", "name"); name != "" {
			return fmt.Sprintf("Promote file to dataset %q", name)
		}
	case "OntologyAddRow":
		return "Add ontology record"
	case "OntologyUpdateRow":
		return "Update ontology record"
	case "OntologyDeleteRow":
		return "Delete ontology record"
	}
	if name := firstResourceString(payload, "name", "displayName", "apiName", "datasetName"); name != "" {
		return fmt.Sprintf("%s %s %q", action, resource, name)
	}
	return fmt.Sprintf("%s %s", action, resource)
}

func confirmPlatformFunctionLifecycle(stdin io.Reader, stderr io.Writer, action, functionRef, projectName string, version int) (bool, error) {
	var prompt string
	switch action {
	case "promote":
		prompt = fmt.Sprintf("Promote function %s in project %s and make the new version active? [y/N]: ", functionRef, projectName)
	case "set-active":
		prompt = fmt.Sprintf("Set function %s version %d active in project %s? [y/N]: ", functionRef, version, projectName)
	case "restore-draft":
		prompt = fmt.Sprintf("Restore function %s version %d into the draft in project %s? [y/N]: ", functionRef, version, projectName)
	case "deploy":
		prompt = fmt.Sprintf("Deploy function %s in project %s? [y/N]: ", functionRef, projectName)
	case "redeploy":
		prompt = fmt.Sprintf("Redeploy function %s in project %s? [y/N]: ", functionRef, projectName)
	default:
		prompt = fmt.Sprintf("Run %s on function %s in project %s? [y/N]: ", action, functionRef, projectName)
	}
	if _, err := fmt.Fprint(stderr, prompt); err != nil {
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

func normalizePlatformResourceKey(value string) string {
	parts := strings.Split(strings.TrimSpace(value), ":")
	for i, part := range parts {
		parts[i] = normalizePlatformResourceToken(part)
	}
	return strings.Join(parts, ":")
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

func resolvePlatformResourceID(ctx context.Context, session *platformSession, projectID, resource, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s id or name is required", resource)
	}
	switch normalizePlatformResourceToken(resource) {
	case "secret":
		items, err := session.Client.ProjectSecrets(ctx, projectID)
		if err != nil {
			return "", err
		}
		return resolveNamedResource(value, "secret", items, func(item platform.ProjectSecret) (string, string) { return item.ID, item.Name })
	case "ai_provider":
		items, err := session.Client.AIProviders(ctx, projectID)
		if err != nil {
			return "", err
		}
		return resolveNamedResource(value, "AI provider", items, func(item platform.AIProvider) (string, string) { return item.ID, item.Name })
	case "ontology":
		items, err := session.Client.Ontologies(ctx, projectID)
		if err != nil {
			return "", err
		}
		var matches []string
		for _, item := range items {
			if item.ID == value {
				return item.ID, nil
			}
			if item.APIName == value || item.DisplayName == value {
				matches = append(matches, item.ID)
			}
		}
		switch len(matches) {
		case 0:
			return "", fmt.Errorf("ontology %q not found", value)
		case 1:
			return matches[0], nil
		default:
			return "", fmt.Errorf("ontology %q is ambiguous; use an id", value)
		}
	case "dataset":
		items, err := session.Client.Datasets(ctx, projectID)
		if err != nil {
			return "", err
		}
		return resolveNamedResource(value, "dataset", items, func(item platform.Dataset) (string, string) { return item.ID, item.Name })
	case "transform":
		items, err := session.Client.Transforms(ctx, projectID)
		if err != nil {
			return "", err
		}
		return resolveNamedResource(value, "transform", items, func(item platform.Transform) (string, string) { return item.ID, item.Name })
	case "function":
		items, err := session.Client.Functions(ctx, projectID, []string{"id", "name"})
		if err != nil {
			return "", err
		}
		return resolveNamedResource(value, "function", items, func(item platform.Function) (string, string) { return item.ID, item.Name })
	case "file":
		file, err := resolveProjectFile(ctx, session, projectID, value)
		if err != nil {
			return "", err
		}
		return file.ID, nil
	case "folder":
		folder, err := resolveProjectFolder(ctx, session, projectID, value)
		if err != nil {
			return "", err
		}
		return folder.ID, nil
	default:
		return value, nil
	}
}

func resolveTransform(ctx context.Context, session *platformSession, projectID, value string) (*platform.Transform, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("transform id or name is required")
	}
	items, err := session.Client.Transforms(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var matches []platform.Transform
	for _, item := range items {
		if item.ID == value {
			return &item, nil
		}
		if item.Name == value {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("transform %q not found", value)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("transform name %q is ambiguous; use an id", value)
	}
}

func resolveNamedResource[T any](value, label string, items []T, identity func(T) (string, string)) (string, error) {
	var matches []string
	for _, item := range items {
		id, name := identity(item)
		if id == value {
			return id, nil
		}
		if name == value {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%s %q not found", label, value)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%s name %q is ambiguous; use an id", label, value)
	}
}

func resolveProjectFile(ctx context.Context, session *platformSession, projectID, value string) (*platform.ProjectFile, error) {
	value = strings.TrimSpace(value)
	contents, err := session.Client.FolderContents(ctx, projectID, "", nil)
	if err != nil {
		return nil, err
	}
	if file, err := matchProjectFile(value, contents.Files); err == nil {
		return file, nil
	}
	files, err := session.Client.SearchProjectFiles(ctx, projectID, value)
	if err != nil {
		return nil, err
	}
	if file, err := matchProjectFile(value, files); err == nil {
		return file, nil
	}
	entries, err := loadProjectFolderTree(ctx, session, projectID)
	if err != nil {
		return nil, err
	}
	var matches []platform.ProjectFile
	for _, entry := range entries {
		if entry.Kind != "file" {
			continue
		}
		if entry.ID == value {
			file := entry.File
			return &file, nil
		}
		if entry.Name == value || entry.Path == value {
			matches = append(matches, entry.File)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("file %q not found", value)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("file %q is ambiguous; use an id or path", value)
	}
}

func matchProjectFile(value string, files []platform.ProjectFile) (*platform.ProjectFile, error) {
	var matches []platform.ProjectFile
	for _, file := range files {
		if file.ID == value {
			return &file, nil
		}
		if file.Name == value {
			matches = append(matches, file)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("file %q not found", value)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("file name %q is ambiguous; use an id", value)
	}
}

func resolveProjectFolder(ctx context.Context, session *platformSession, projectID, value string) (*platform.ProjectFolder, error) {
	entries, err := loadProjectFolderTree(ctx, session, projectID)
	if err != nil {
		return nil, err
	}
	var matches []platformFolderTreeEntry
	for _, entry := range entries {
		if entry.Kind != "folder" {
			continue
		}
		if entry.ID == value {
			folder := entry.Folder
			return &folder, nil
		}
		if entry.Name == value || entry.Path == value {
			matches = append(matches, entry)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("folder %q not found", value)
	case 1:
		folder := matches[0].Folder
		return &folder, nil
	default:
		return nil, fmt.Errorf("folder %q is ambiguous; use an id", value)
	}
}

type platformFolderTreeEntry struct {
	ID       string                 `json:"id"`
	Kind     string                 `json:"kind"`
	Name     string                 `json:"name"`
	ParentID string                 `json:"parentId,omitempty"`
	Depth    int                    `json:"depth"`
	Path     string                 `json:"path"`
	Folder   platform.ProjectFolder `json:"-"`
	File     platform.ProjectFile   `json:"-"`
}

func loadProjectFolderTree(ctx context.Context, session *platformSession, projectID string) ([]platformFolderTreeEntry, error) {
	var entries []platformFolderTreeEntry
	if err := appendProjectFolderTree(ctx, session, projectID, "", "", 0, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func appendProjectFolderTree(ctx context.Context, session *platformSession, projectID, folderID, basePath string, depth int, entries *[]platformFolderTreeEntry) error {
	contents, err := session.Client.FolderContents(ctx, projectID, folderID, nil)
	if err != nil {
		return err
	}
	for _, folder := range contents.Folders {
		parentID := ""
		if folder.ParentID != nil {
			parentID = *folder.ParentID
		}
		path := joinPlatformTreePath(basePath, folder.Name)
		*entries = append(*entries, platformFolderTreeEntry{
			ID:       folder.ID,
			Kind:     "folder",
			Name:     folder.Name,
			ParentID: parentID,
			Depth:    depth,
			Path:     path,
			Folder:   folder,
		})
		if err := appendProjectFolderTree(ctx, session, projectID, folder.ID, path, depth+1, entries); err != nil {
			return err
		}
	}
	for _, file := range contents.Files {
		parentID := folderID
		if file.FolderID != nil {
			parentID = *file.FolderID
		}
		*entries = append(*entries, platformFolderTreeEntry{
			ID:       file.ID,
			Kind:     "file",
			Name:     file.Name,
			ParentID: parentID,
			Depth:    depth,
			Path:     joinPlatformTreePath(basePath, file.Name),
			File:     file,
		})
	}
	return nil
}

func joinPlatformTreePath(basePath, name string) string {
	if strings.TrimSpace(basePath) == "" {
		return name
	}
	return strings.TrimRight(basePath, "/") + "/" + name
}

func previewFileDownloadContent(preview *platform.FilePreviewResult) ([]byte, error) {
	if preview == nil {
		return nil, fmt.Errorf("platform returned no file preview")
	}
	if preview.TextContent != nil {
		return []byte(*preview.TextContent), nil
	}
	if preview.Tabular != nil {
		var buffer bytes.Buffer
		writer := csv.NewWriter(&buffer)
		headers := make([]string, len(preview.Tabular.Columns))
		for i, column := range preview.Tabular.Columns {
			headers[i] = column.Name
		}
		if err := writer.Write(headers); err != nil {
			return nil, err
		}
		for _, row := range preview.Tabular.Rows {
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return nil, err
		}
		return buffer.Bytes(), nil
	}
	return nil, fmt.Errorf("file preview does not expose downloadable text or tabular content")
}

func readFunctionInput(cmd *cobra.Command) (string, error) {
	if cmd.Flags().Changed("input-json") && cmd.Flags().Changed("input-file") {
		return "", fmt.Errorf("pass only one of --input-json or --input-file")
	}
	raw := strings.TrimSpace(functionInputJSON)
	if cmd.Flags().Changed("input-file") {
		body, err := os.ReadFile(filepath.Clean(functionInputFile))
		if err != nil {
			return "", fmt.Errorf("read --input-file: %w", err)
		}
		raw = strings.TrimSpace(string(body))
	}
	if raw == "" {
		raw = "{}"
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return "", fmt.Errorf("function input must be valid JSON: %w", err)
	}
	return raw, nil
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

func inspectPlatformFile(ctx context.Context, cmd *cobra.Command, session *platformSession, fileRef string) (*platform.FileInspection, error) {
	fileID, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "file", fileRef)
	if err != nil {
		return nil, err
	}
	var sheetIndex *int
	if cmd.Flags().Changed("sheet-index") {
		sheetIndex = &platformSheetIndex
	}
	preview, err := session.Client.FilePreview(ctx, session.Host.DefaultProjectID, fileID, sheetIndex, nil)
	if err != nil {
		return nil, err
	}
	return platform.InspectFilePreview(fileID, preview), nil
}

func fileInspectionTable(inspection *platform.FileInspection) *output.QueryResult {
	if inspection == nil {
		return tableResult([]string{"field", "value"}, nil)
	}
	sheetName := ""
	if inspection.SheetName != nil {
		sheetName = *inspection.SheetName
	}
	sheetIndex := ""
	if inspection.SheetIndex != nil {
		sheetIndex = fmt.Sprint(*inspection.SheetIndex)
	}
	sheetCount := ""
	if inspection.SheetCount != nil {
		sheetCount = fmt.Sprint(*inspection.SheetCount)
	}
	return tableResult([]string{"field", "value"}, [][]any{
		{"file_id", inspection.FileID},
		{"mime_type", inspection.MIMEType},
		{"size_bytes", inspection.SizeBytes},
		{"is_tabular", inspection.IsTabular},
		{"sheet_name", sheetName},
		{"sheet_index", sheetIndex},
		{"sheet_count", sheetCount},
		{"columns", len(inspection.Columns)},
		{"rows", inspection.Total},
		{"column_map_example", inspection.ColumnMapExample},
	})
}

func fileInspectionColumnsTable(inspection *platform.FileInspection) *output.QueryResult {
	if inspection == nil {
		return tableResult([]string{"source_column", "dataset_column", "type", "nullable", "primary", "column_map"}, nil)
	}
	rows := make([][]any, len(inspection.Columns))
	for i, column := range inspection.Columns {
		rows[i] = []any{column.SourceColumn, column.DatasetColumn, column.DataType, column.IsNullable, column.IsPrimary, column.ColumnMapSnippet}
	}
	return tableResult([]string{"source_column", "dataset_column", "type", "nullable", "primary", "column_map"}, rows)
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
