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
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

// Core CLI commands for ontology analytics, bulk record inserts, run
// cancellation, project restore/purge and service accounts. Every other new
// platform capability is reachable through `whodb resources action` and
// `whodb resources query`.

var (
	ontologyQueryWhere        string
	ontologyQuerySearch       string
	ontologyQueryGroupBy      []string
	ontologyQueryMetrics      []string
	ontologyQuerySort         []string
	ontologyQueryScanLimit    int
	ontologyExplainInferred   bool
	ontologySimilarTopK       int
	ontologySimilarProperties []string
	ontologyRowsFile          string
	ontologyRowsIdempotency   string
	transformCancelRunID      string
	serviceAccountName        string
	serviceAccountDescription string
	serviceAccountKeyExpires  string
)

var ontologyQueryCmd = &cobra.Command{
	Use:           "query <entity>",
	Short:         "Query hosted WhoDB ontology rows with filters, grouping and metrics",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: `  whodb ontologies query orders --where '{"status":"open"}' --limit 20
  whodb ontologies query orders --group-by region --metric count --metric sum:amount:total --sort total:desc
  whodb ontologies query customers --search ada --search-field name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePlatformPage(platformLimit, platformOffset); err != nil {
			return err
		}
		where, err := parseJSONObjectFlag("--where", ontologyQueryWhere)
		if err != nil {
			return err
		}
		metrics, err := parseOntologyMetricFlags(ontologyQueryMetrics)
		if err != nil {
			return err
		}
		return runPlatformProjectRowsRead(cmd, func(ctx context.Context, session *platformSession) (*platform.DatasetQueryResult, error) {
			input := platform.OntologyQueryInput{
				Entity:       args[0],
				Where:        where,
				Search:       ontologyQuerySearch,
				SearchFields: ontologySimilarProperties,
				GroupBy:      ontologyQueryGroupBy,
				Metrics:      metrics,
				PageSize:     platformLimit,
				Offset:       platformOffset,
				ScanLimit:    ontologyQueryScanLimit,
			}
			for _, sort := range ontologyQuerySort {
				field, direction, _ := strings.Cut(sort, ":")
				input.Sort = append(input.Sort, platform.OntologyQuerySort{Field: strings.TrimSpace(field), Desc: strings.EqualFold(strings.TrimSpace(direction), "desc")})
			}
			return session.Client.OntologyQuery(ctx, session.Host.DefaultProjectID, input)
		})
	},
}

var ontologyAggregateCmd = &cobra.Command{
	Use:           "aggregate <ontology>",
	Short:         "Compute exact aggregates over a hosted WhoDB ontology",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: `  whodb ontologies aggregate orders --metric COUNT
  whodb ontologies aggregate orders --group-by region --metric SUM:amount:total --sort total:desc
  whodb ontologies aggregate orders --metric COUNT --where '{"Type":"Atomic","Atomic":{"Key":"status","Operator":"=","Value":"open","ColumnType":"String"}}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePlatformPage(platformLimit, platformOffset); err != nil {
			return err
		}
		where, err := parseJSONObjectFlag("--where", ontologyQueryWhere)
		if err != nil {
			return err
		}
		metrics, err := parseOntologyMetricFlags(ontologyQueryMetrics)
		if err != nil {
			return err
		}
		if len(metrics) == 0 {
			return fmt.Errorf("--metric is required, for example --metric COUNT")
		}
		return runPlatformProjectRowsRead(cmd, func(ctx context.Context, session *platformSession) (*platform.DatasetQueryResult, error) {
			id, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "ontology", args[0])
			if err != nil {
				return nil, err
			}
			input := platform.OntologyAggregateInput{ID: id, GroupBy: ontologyQueryGroupBy, Metrics: metrics, Where: where, PageSize: platformLimit, PageOffset: platformOffset}
			for _, sort := range ontologyQuerySort {
				column, direction, _ := strings.Cut(sort, ":")
				direction = strings.ToUpper(strings.TrimSpace(direction))
				if direction == "" {
					direction = "ASC"
				}
				input.Sort = append(input.Sort, platform.SortCondition{Column: strings.TrimSpace(column), Direction: direction})
			}
			return session.Client.OntologyAggregate(ctx, session.Host.DefaultProjectID, input)
		})
	},
}

var ontologyStatsCmd = &cobra.Command{
	Use:           "stats <ontology> <property>",
	Short:         "Summarize one hosted WhoDB ontology property",
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	Example:       `  whodb ontologies stats orders amount`,
	RunE: func(cmd *cobra.Command, args []string) error {
		where, err := parseJSONObjectFlag("--where", ontologyQueryWhere)
		if err != nil {
			return err
		}
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			id, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "ontology", args[0])
			if err != nil {
				return nil, nil, err
			}
			stats, err := session.Client.OntologyStats(ctx, session.Host.DefaultProjectID, id, args[1], where)
			if err != nil {
				return nil, nil, err
			}
			return stats, tableResult([]string{"property", "min", "max", "total", "non_null"}, [][]any{{stats.Property, optionalString(stats.Min), optionalString(stats.Max), stats.Total, stats.NonNull}}), nil
		})
	},
}

var ontologyExplainCmd = &cobra.Command{
	Use:           "explain [entity...]",
	Short:         "Describe hosted WhoDB ontologies, their fields and relationships",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: `  whodb ontologies explain
  whodb ontologies explain orders customers --include-inferred`,
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
		org, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
		if err != nil {
			return err
		}
		session.Client.SetWorkspaceContext(org.ID, project.ID)
		entities := args
		if len(entities) == 0 {
			ontologies, err := session.Client.Ontologies(ctx, project.ID)
			if err != nil {
				return err
			}
			for _, ontology := range ontologies {
				entities = append(entities, ontology.APIName)
			}
			if len(entities) == 0 {
				return fmt.Errorf("project %s has no ontologies", project.Name)
			}
		}
		description, err := session.Client.OntologyDescribe(ctx, project.ID, entities, ontologyExplainInferred)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, description)
		}
		if strings.TrimSpace(description.Text) != "" {
			fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(description.Text))
			fmt.Fprintln(cmd.OutOrStdout())
		}
		rows := make([][]any, 0)
		for _, entity := range description.Entities {
			for _, field := range entity.Fields {
				rows = append(rows, []any{entity.Name, field.Name, field.Type, field.Required, field.Description})
			}
		}
		for _, relationship := range description.Relationships {
			rows = append(rows, []any{relationship.Source, relationship.Name, "link:" + relationship.Cardinality, false, fmt.Sprintf("%s.%s -> %s.%s", relationship.Source, relationship.SourceField, relationship.Target, relationship.TargetField)})
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"entity", "field", "type", "required", "description"}, rows))
	},
}

var ontologySimilarCmd = &cobra.Command{
	Use:           "similar <ontology> <row-id>",
	Short:         "Find hosted WhoDB ontology rows similar to one row",
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	Example:       `  whodb ontologies similar products 42 --top-k 5 --property name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		where, err := parseJSONObjectFlag("--where", ontologyQueryWhere)
		if err != nil {
			return err
		}
		return runPlatformProjectRead(cmd, func(ctx context.Context, session *platformSession, _ *platform.Project) (any, *output.QueryResult, error) {
			id, err := resolvePlatformResourceID(ctx, session, session.Host.DefaultProjectID, "ontology", args[0])
			if err != nil {
				return nil, nil, err
			}
			result, err := session.Client.OntologySimilar(ctx, session.Host.DefaultProjectID, platform.OntologySimilarInput{EntityID: id, RowID: args[1], TopK: ontologySimilarTopK, Properties: ontologySimilarProperties, Where: where})
			if err != nil {
				return nil, nil, err
			}
			rows := make([][]any, 0, len(result.Rows))
			for _, row := range result.Rows {
				values := make([]string, 0, len(row.Values))
				for _, value := range row.Values {
					values = append(values, value.Key+"="+value.Value)
				}
				rows = append(rows, []any{fmt.Sprintf("%.4f", row.Score), strings.Join(values, ", ")})
			}
			return result, tableResult([]string{"score", "values"}, rows), nil
		})
	},
}

var ontologyRecordsAddManyCmd = withExample(typedResourceWriteCommand("add-many <ontology>", "Add many hosted WhoDB ontology records from a JSON file", "action", "ontology", "add_records", buildOntologyRecordsAddManyPayload), `  whodb ontologies records add-many customers --file rows.json --idempotency-key import-2026-10-12`)

func buildOntologyRecordsAddManyPayload(cmd *cobra.Command) (map[string]any, error) {
	if strings.TrimSpace(ontologyRowsFile) == "" {
		return nil, fmt.Errorf("--file is required")
	}
	raw, err := os.ReadFile(ontologyRowsFile)
	if err != nil {
		return nil, fmt.Errorf("read rows file: %w", err)
	}
	var objects []map[string]any
	if err := json.Unmarshal(raw, &objects); err != nil {
		return nil, fmt.Errorf("--file must contain a JSON array of objects keyed by ontology property: %w", err)
	}
	if len(objects) == 0 {
		return nil, fmt.Errorf("--file contains no rows")
	}
	rows := make([]map[string]any, 0, len(objects))
	for index, object := range objects {
		values := make([]map[string]any, 0, len(object))
		for key, value := range object {
			if strings.TrimSpace(key) == "" {
				continue
			}
			values = append(values, map[string]any{"Key": key, "Value": fmt.Sprint(value)})
		}
		if len(values) == 0 {
			return nil, fmt.Errorf("row %d has no values", index)
		}
		rows = append(rows, map[string]any{"values": values})
	}
	payload := map[string]any{"rows": rows}
	if strings.TrimSpace(ontologyRowsIdempotency) != "" {
		payload["idempotencyKey"] = strings.TrimSpace(ontologyRowsIdempotency)
	}
	return payload, nil
}

var functionsRunsCancelCmd = withExample(typedResourceWriteCommand("cancel <run-id>", "Cancel a queued or running hosted WhoDB function run", "action", "function_run", "cancel", func(*cobra.Command) (map[string]any, error) {
	return map[string]any{}, nil
}), `  whodb functions runs cancel 7c1f...`)

var transformsCancelRunCmd = withExample(typedResourceWriteCommand("cancel-run <transform>", "Cancel the running hosted WhoDB transform run", "action", "transform", "cancel_run", func(*cobra.Command) (map[string]any, error) {
	payload := map[string]any{}
	if strings.TrimSpace(transformCancelRunID) != "" {
		payload["runId"] = strings.TrimSpace(transformCancelRunID)
	}
	return payload, nil
}), `  whodb transforms cancel-run nightly-rollup`)

var projectsRestoreCmd = platformProjectLifecycleCommand("restore <project-id>", "Restore a soft-deleted hosted WhoDB project", "RestoreProject", "action:restore:project")
var projectsPurgeCmd = platformProjectLifecycleCommand("purge <project-id>", "Permanently delete a soft-deleted hosted WhoDB project", "PurgeProject", "action:purge:project")

func platformProjectLifecycleCommand(use, short, mutation, specKey string) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       "  whodb resources query DeletedResources   # find project ids\n  whodb projects " + strings.Fields(use)[0] + " <project-id>",
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
			org, err := resolveOrganization(ctx, session.Client, session.Host, projectsOrg)
			if err != nil {
				return err
			}
			projectID := strings.TrimSpace(args[0])
			if !platformWriteYes {
				approved, err := confirmPlatformResourceWrite(cmd.InOrStdin(), cmd.ErrOrStderr(), platform.GenericWriteSpecs[specKey], projectID, map[string]any{"orgId": org.ID})
				if err != nil {
					return err
				}
				if !approved {
					return fmt.Errorf("%s cancelled", strings.Fields(use)[0])
				}
			}
			result, err := session.Client.PlatformMutation(ctx, mutation, map[string]any{"id": projectID, "orgId": org.ID})
			if err != nil {
				return err
			}
			if format == output.FormatJSON {
				return writeAutomationEnvelope(cmd, "projects."+strings.Fields(use)[0], result)
			}
			newCommandOutput(cmd, format, platformQuiet).Success("%s project %s in %s", mutation, projectID, org.Name)
			return nil
		},
	}
}

var serviceAccountsCmd = &cobra.Command{Use: "service-accounts", Short: "Manage hosted WhoDB service accounts and API keys"}
var serviceAccountKeysCmd = &cobra.Command{Use: "keys", Short: "Manage hosted WhoDB service account keys"}

var serviceAccountsListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List service accounts in the selected organization",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformOrgRead(cmd, func(ctx context.Context, session *platformSession, org *platform.Organization) (any, *output.QueryResult, error) {
			data, err := session.Client.PlatformQuery(ctx, "ServiceAccounts", map[string]any{"orgId": org.ID})
			if err != nil {
				return nil, nil, err
			}
			items, _ := data.([]any)
			rows := make([][]any, 0, len(items))
			for _, item := range items {
				account, _ := item.(map[string]any)
				keys, _ := account["keys"].([]any)
				active := 0
				for _, key := range keys {
					if entry, _ := key.(map[string]any); entry["revokedAt"] == nil {
						active++
					}
				}
				rows = append(rows, []any{account["id"], account["name"], account["description"], len(keys), active, account["createdAt"]})
			}
			return data, tableResult([]string{"id", "name", "description", "keys", "active_keys", "created_at"}, rows), nil
		})
	},
}

var serviceAccountsCreateCmd = &cobra.Command{
	Use:           "create",
	Short:         "Create a service account in the selected organization",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	Example:       `  whodb service-accounts create --name ci-deployer --description "Deploys packages from CI"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(serviceAccountName) == "" {
			return fmt.Errorf("--name is required")
		}
		return runPlatformOrgMutation(cmd, "service-accounts.create", "CreateServiceAccount", func(org *platform.Organization) map[string]any {
			variables := map[string]any{"orgId": org.ID, "name": strings.TrimSpace(serviceAccountName)}
			if strings.TrimSpace(serviceAccountDescription) != "" {
				variables["description"] = strings.TrimSpace(serviceAccountDescription)
			}
			return variables
		}, func(result map[string]any) string {
			return fmt.Sprintf("Created service account %v (%v)", result["name"], result["id"])
		})
	},
}

var serviceAccountsDeleteCmd = &cobra.Command{
	Use:           "delete <service-account-id>",
	Short:         "Delete a service account and revoke all of its keys",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformOrgMutation(cmd, "service-accounts.delete", "DeleteServiceAccount", func(org *platform.Organization) map[string]any {
			return map[string]any{"orgId": org.ID, "id": strings.TrimSpace(args[0])}
		}, func(map[string]any) string {
			return fmt.Sprintf("Deleted service account %s", args[0])
		})
	},
}

var serviceAccountKeysCreateCmd = &cobra.Command{
	Use:           "create <service-account-id>",
	Short:         "Issue an API key for a service account; the key is shown once",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	Example:       `  whodb service-accounts keys create <service-account-id> --expires-at 2027-01-01T00:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformOrgMutation(cmd, "service-accounts.keys.create", "CreateServiceAccountKey", func(org *platform.Organization) map[string]any {
			variables := map[string]any{"orgId": org.ID, "serviceAccountId": strings.TrimSpace(args[0])}
			if strings.TrimSpace(serviceAccountKeyExpires) != "" {
				variables["expiresAt"] = strings.TrimSpace(serviceAccountKeyExpires)
			}
			return variables
		}, func(result map[string]any) string {
			key, _ := result["key"].(map[string]any)
			return fmt.Sprintf("Created key %v. Store this now; it is not shown again:\n%v", key["id"], result["rawKey"])
		})
	},
}

var serviceAccountKeysRevokeCmd = &cobra.Command{
	Use:           "revoke <key-id>",
	Short:         "Revoke one service account key",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformOrgMutation(cmd, "service-accounts.keys.revoke", "RevokeServiceAccountKey", func(org *platform.Organization) map[string]any {
			return map[string]any{"orgId": org.ID, "keyId": strings.TrimSpace(args[0])}
		}, func(map[string]any) string {
			return fmt.Sprintf("Revoked key %s", args[0])
		})
	},
}

func runPlatformOrgRead(cmd *cobra.Command, read func(context.Context, *platformSession, *platform.Organization) (any, *output.QueryResult, error)) error {
	ctx := context.Background()
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	org, err := resolveOrganization(ctx, session.Client, session.Host, platformResourceOrg)
	if err != nil {
		return err
	}
	value, result, err := read(ctx, session, org)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, value)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(result)
}

func runPlatformOrgMutation(cmd *cobra.Command, commandName, mutation string, variables func(*platform.Organization) map[string]any, success func(map[string]any) string) error {
	ctx := context.Background()
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	org, err := resolveOrganization(ctx, session.Client, session.Host, platformResourceOrg)
	if err != nil {
		return err
	}
	result, err := session.Client.PlatformMutation(ctx, mutation, variables(org))
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, commandName, result)
	}
	var data map[string]any
	_ = json.Unmarshal(result.Data, &data)
	newCommandOutput(cmd, format, platformQuiet).Success("%s", success(data))
	return nil
}

func parseJSONObjectFlag(flag, value string) (map[string]any, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return nil, fmt.Errorf("%s must be a JSON object: %w", flag, err)
	}
	return object, nil
}

// parseOntologyMetricFlags accepts op, op:field, or op:field:alias.
func parseOntologyMetricFlags(values []string) ([]platform.OntologyMetric, error) {
	metrics := make([]platform.OntologyMetric, 0, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, ":", 3)
		op := strings.TrimSpace(parts[0])
		if op == "" {
			return nil, fmt.Errorf("--metric must be op, op:field, or op:field:alias")
		}
		metric := platform.OntologyMetric{Op: op}
		if len(parts) > 1 {
			metric.Field = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			metric.As = strings.TrimSpace(parts[2])
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

func registerPlatformGapCommands() {
	ontologiesCmd.AddCommand(ontologyQueryCmd, ontologyAggregateCmd, ontologyStatsCmd, ontologyExplainCmd, ontologySimilarCmd)
	ontologyRecordsCmd.AddCommand(ontologyRecordsAddManyCmd)
	functionsRunsCmd.AddCommand(functionsRunsCancelCmd)
	transformsCmd.AddCommand(transformsCancelRunCmd)
	projectsCmd.AddCommand(projectsRestoreCmd, projectsPurgeCmd)
	serviceAccountKeysCmd.AddCommand(serviceAccountKeysCreateCmd, serviceAccountKeysRevokeCmd)
	serviceAccountsCmd.AddCommand(serviceAccountsListCmd, serviceAccountsCreateCmd, serviceAccountsDeleteCmd, serviceAccountKeysCmd)
	serviceAccountsCmd.PersistentFlags().StringVar(&platformResourceOrg, "org", "", "organization id, slug, or name (defaults to selected organization)")

	for _, command := range []*cobra.Command{ontologyQueryCmd, ontologyAggregateCmd} {
		command.Flags().IntVar(&platformLimit, "limit", 50, "maximum rows to return")
		command.Flags().IntVar(&platformOffset, "offset", 0, "row offset")
		command.Flags().StringArrayVar(&ontologyQueryGroupBy, "group-by", nil, "property to group by; repeatable")
		command.Flags().StringArrayVar(&ontologyQueryMetrics, "metric", nil, "metric as op, op:field, or op:field:alias; repeatable")
		command.Flags().StringArrayVar(&ontologyQuerySort, "sort", nil, "sort as field or field:desc; repeatable")
	}
	ontologyQueryCmd.Flags().StringVar(&ontologyQueryWhere, "where", "", `filter object, for example {"status":"open"} or {"amount":{"gte":100}}`)
	ontologyQueryCmd.Flags().StringVar(&ontologyQuerySearch, "search", "", "free-text search across searchable properties")
	ontologyQueryCmd.Flags().StringArrayVar(&ontologySimilarProperties, "search-field", nil, "restrict search to this property; repeatable")
	ontologyQueryCmd.Flags().IntVar(&ontologyQueryScanLimit, "scan-limit", 0, "rows scanned before filtering (default 1000, max 5000)")
	for _, command := range []*cobra.Command{ontologyAggregateCmd, ontologyStatsCmd, ontologySimilarCmd} {
		command.Flags().StringVar(&ontologyQueryWhere, "where", "", "platform WhereCondition JSON object")
	}
	ontologyExplainCmd.Flags().BoolVar(&ontologyExplainInferred, "include-inferred", false, "include relationships inferred from data")
	ontologySimilarCmd.Flags().IntVar(&ontologySimilarTopK, "top-k", 5, "number of similar rows to return")
	ontologySimilarCmd.Flags().StringArrayVar(&ontologySimilarProperties, "property", nil, "property to include in results; repeatable")
	ontologyRecordsAddManyCmd.Flags().StringVar(&ontologyRowsFile, "file", "", "JSON file containing an array of row objects keyed by ontology property")
	ontologyRecordsAddManyCmd.Flags().StringVar(&ontologyRowsIdempotency, "idempotency-key", "", "client-chosen key so a retried import does not insert twice")
	transformsCancelRunCmd.Flags().StringVar(&transformCancelRunID, "run-id", "", "specific run id to cancel (defaults to the active run)")
	for _, command := range []*cobra.Command{ontologyRecordsAddManyCmd, functionsRunsCancelCmd, transformsCancelRunCmd, projectsRestoreCmd, projectsPurgeCmd} {
		command.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "apply without prompting")
	}
	for _, command := range []*cobra.Command{projectsRestoreCmd, projectsPurgeCmd} {
		command.Flags().StringVar(&projectsOrg, "org", "", "organization id, slug, or name (defaults to selected organization)")
	}
	serviceAccountsCreateCmd.Flags().StringVar(&serviceAccountName, "name", "", "service account name")
	serviceAccountsCreateCmd.Flags().StringVar(&serviceAccountDescription, "description", "", "purpose of the account")
	serviceAccountKeysCreateCmd.Flags().StringVar(&serviceAccountKeyExpires, "expires-at", "", "RFC 3339 expiry; omit for no expiry")
}
