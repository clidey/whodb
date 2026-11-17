package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
)

func init() {
	edge := &cobra.Command{Use: "edge", Short: "Inspect edge devices and manage edge function runs and schedules"}
	sessions := &cobra.Command{Use: "agent-sessions", Short: "Inspect and cancel your platform AI sessions"}
	analysis := &cobra.Command{Use: "analysis", Short: "Inspect governed analysis artifacts and result pages"}
	scripts := &cobra.Command{Use: "scripts", Short: "Inspect curated script packages and run confirmed script previews"}
	usage := &cobra.Command{Use: "usage", Short: "Inspect organization usage and feature entitlements"}
	for _, group := range []*cobra.Command{edge, sessions, analysis, usage, scripts} {
		rootCmd.AddCommand(group)
		group.PersistentFlags().StringVar(&platformHost, "host", "", "hosted WhoDB URL")
		group.PersistentFlags().StringVarP(&platformFormat, "format", "f", "auto", "output format: auto, table, plain, json, ndjson, csv")
		group.PersistentFlags().BoolVarP(&platformQuiet, "quiet", "q", false, "suppress informational messages")
		group.PersistentFlags().StringVar(&platformResourceOrg, "org", "", "organization id, slug, or name")
		group.PersistentFlags().StringVar(&platformResourceProject, "project", "", "project id, slug, or name")
	}
	routes := map[string]struct {
		parent *cobra.Command
		use    string
	}{
		"MySubscription": {usage, "subscription"}, "AvailablePlans": {usage, "plans"}, "PlanFeatures": {usage, "plan-features <plan>"}, "PreviewSubscriptionChange": {usage, "preview-plan <plan> <interval>"},
		"ListScriptPackages": {scripts, "packages"}, "SearchScriptPackages": {scripts, "search"}, "GetScriptPackage": {scripts, "package <language> <name>"},
		"EdgeRunners": {edge, "runners"}, "EdgeFunctionRun": {edge, "run <id>"}, "EdgeFunctionRuns": {edge, "runs <function-id>"}, "EdgeSchedules": {edge, "schedules <function-id>"},
		"TransformRun":       {transformsCmd, "get-run <transform-id> <run-id>"},
		"AppAccessDirectory": {appsCmd, "access <app-id>"}, "ProjectTransformRuns": {transformsCmd, "history"},
		"AgentSessions": {sessions, "list"}, "AgentSession": {sessions, "get <session-id>"},
		"AnalysisArtifact": {analysis, "get <artifact-id>"}, "AnalysisArtifactPage": {analysis, "rows <artifact-id>"},
		"BillingCurrentUsage": {usage, "current"}, "BillingCurrentEntitlements": {usage, "entitlements"}, "CanAccessFeature": {usage, "feature <feature-key>"},
	}
	for _, c := range platformapi.CapabilityReads {
		r := routes[c.Operation]
		r.parent.AddCommand(capabilityReadCommand(r.use, c))
	}
	scripts.AddCommand(scriptExecuteCommand())
	cancel := typedResourceWriteCommand("cancel <session-id>", "Cancel your running AI session", "action", "agent_session", "cancel", emptyTypedPayload)
	cancel.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "approve this cancellation")
	sessions.AddCommand(cancel)
	var runner, input, function string
	var interval int
	run := typedResourceWriteCommand("start <function-id>", "Queue a promoted function on its assigned edge device", "action", "edge_function", "run", func(*cobra.Command) (map[string]any, error) {
		return map[string]any{"runnerId": runner, "input": input}, nil
	})
	run.Flags().StringVar(&runner, "runner-id", "", "assigned runner id")
	run.Flags().StringVar(&input, "input", "{}", "JSON object input")
	run.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "approve this execution")
	edge.AddCommand(run)
	schedule := typedResourceWriteCommand("schedule", "Create an interval schedule pinned to the promoted function version", "create", "edge_schedule", "", func(*cobra.Command) (map[string]any, error) {
		return map[string]any{"functionId": function, "runnerId": runner, "input": input, "intervalSeconds": interval}, nil
	})
	schedule.Flags().StringVar(&function, "function-id", "", "function id")
	schedule.Flags().StringVar(&runner, "runner-id", "", "assigned runner id")
	schedule.Flags().StringVar(&input, "input", "{}", "JSON object input")
	schedule.Flags().IntVar(&interval, "interval-seconds", 60, "schedule interval in seconds")
	schedule.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "approve this schedule")
	edge.AddCommand(schedule)
	stop := typedResourceWriteCommand("stop-schedule <schedule-id>", "Stop an edge interval schedule", "action", "edge_schedule", "stop", emptyTypedPayload)
	stop.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "approve stopping this schedule")
	edge.AddCommand(stop)
}

func capabilityReadCommand(use string, c platformapi.CapabilityRead) *cobra.Command {
	limit, offset, perTransform := 20, 0, 5
	var search, cursor, language, query, runtime string
	command := &cobra.Command{Use: use, Short: c.Description, Annotations: map[string]string{"platform_operation": c.Operation}, SilenceUsage: true, SilenceErrors: true, Args: cobra.ExactArgs(len(c.Required))}
	command.RunE = func(cmd *cobra.Command, args []string) error {
		if limit < 1 || limit > 1000 || offset < 0 || perTransform < 1 || perTransform > 20 {
			return fmt.Errorf("limit must be 1..1000, offset nonnegative, per-transform 1..20")
		}
		if c.Operation == "AppAccessDirectory" && limit > 100 {
			return fmt.Errorf("app access limit must be at most 100")
		}
		read := func(ctx context.Context, session *platformSession, project *platformapi.Project) (any, *output.QueryResult, error) {
			vars := map[string]any{"projectId": project.ID, "limit": limit, "pageSize": limit, "offset": offset, "perTransform": perTransform}
			for i, key := range c.Required {
				vars[key] = strings.TrimSpace(args[i])
			}
			if language != "" {
				vars["language"] = language
			}
			if query != "" {
				vars["query"] = query
			}
			if runtime != "" {
				vars["runtime"] = runtime
			}
			if search != "" {
				vars["search"] = search
			}
			if cursor != "" {
				vars["cursor"] = cursor
			}
			data, err := session.Client.PlatformQuery(ctx, c.Operation, vars)
			if err != nil {
				return nil, nil, err
			}
			if c.Operation == "EdgeRunners" || c.Operation == "EdgeSchedules" || c.Operation == "ListScriptPackages" || c.Operation == "SearchScriptPackages" || c.Operation == "PlanFeatures" || c.Operation == "AvailablePlans" {
				if items, ok := data.([]any); ok {
					start := min(offset, len(items))
					data = items[start:min(start+limit, len(items))]
				}
			}
			encoded, err := json.Marshal(data)
			if err != nil {
				return nil, nil, err
			}
			return data, tableResult([]string{"data"}, [][]any{{string(encoded)}}), nil
		}
		if c.OrganizationScoped() {
			return runPlatformOrgRead(cmd, func(ctx context.Context, session *platformSession, org *platformapi.Organization) (any, *output.QueryResult, error) {
				session.Client.SetWorkspaceContext(org.ID, "")
				return read(ctx, session, &platformapi.Project{})
			})
		}
		return runPlatformProjectRead(cmd, read)
	}

	switch c.Operation {
	case "AvailablePlans", "PlanFeatures", "ListScriptPackages", "SearchScriptPackages", "EdgeRunners", "EdgeSchedules", "EdgeFunctionRuns", "AgentSessions", "ProjectTransformRuns", "AppAccessDirectory", "AnalysisArtifactPage":
		command.Flags().IntVar(&limit, "limit", 20, "maximum results or rows (1..1000)")
	}
	switch c.Operation {
	case "AvailablePlans", "PlanFeatures", "ListScriptPackages", "SearchScriptPackages", "EdgeRunners", "EdgeSchedules", "AppAccessDirectory":
		command.Flags().IntVar(&offset, "offset", 0, "list or directory offset")
	}
	if c.Operation == "ProjectTransformRuns" {
		command.Flags().IntVar(&perTransform, "per-transform", 5, "maximum runs per transform (1..20)")
	}
	if c.Operation == "AppAccessDirectory" {
		command.Flags().StringVar(&search, "search", "", "directory search")
	}
	if c.Operation == "AnalysisArtifactPage" {
		command.Flags().StringVar(&cursor, "cursor", "", "nextCursor from the previous analysis page")
	}
	if c.Operation == "ListScriptPackages" || c.Operation == "SearchScriptPackages" {
		command.Flags().StringVar(&language, "language", "", "script language filter")
	}
	if c.Operation == "SearchScriptPackages" {
		command.Flags().StringVar(&query, "query", "", "package search text")
		command.Flags().StringVar(&runtime, "runtime", "", "script runtime filter")
	}
	return command
}
