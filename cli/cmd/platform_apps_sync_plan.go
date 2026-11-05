package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/spf13/cobra"
)

type appSyncPlan struct {
	Upserts []string `json:"upserts"`
	Deletes []string `json:"deletes"`
	SHA256  string   `json:"sha256"`
}

type appSyncJournal struct {
	SHA256  string   `json:"sha256"`
	AppID   string   `json:"app_id"`
	Upserts []string `json:"upserts"`
	Deletes []string `json:"deletes"`
}

var (
	appSyncPlanPath    string
	appSyncJournalPath string
	appSyncApply       bool
	appSyncBatchSize   int
)

var appsSyncPlanCmd = &cobra.Command{
	Use:           "sync-plan <folder>",
	Short:         "Apply a reviewed app file migration to the hosted draft",
	Long:          "Apply only the paths in a SHA-256-checked plan. A journal makes interrupted batches safe to retry. The published version is unchanged.",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runAppSyncPlan,
}

func registerAppSyncPlanCommand() {
	appsSyncPlanCmd.Flags().StringVar(&appLocalRef, "app", "", "existing hosted app ID or name")
	appsSyncPlanCmd.Flags().StringVar(&appSyncPlanPath, "plan", "", "reviewed JSON plan with upserts, deletes, and sha256")
	appsSyncPlanCmd.Flags().StringVar(&appSyncJournalPath, "journal", "", "progress journal; defaults beside the plan")
	appsSyncPlanCmd.Flags().BoolVar(&appSyncApply, "apply", false, "stage the plan in the hosted draft")
	appsSyncPlanCmd.Flags().IntVar(&appSyncBatchSize, "batch-size", 20, "mutations per GraphQL request (1-20)")
	appsCmd.AddCommand(appsSyncPlanCmd)
}

func runAppSyncPlan(cmd *cobra.Command, args []string) error {
	if appSyncPlanPath == "" || appLocalRef == "" {
		return fmt.Errorf("--plan and --app are required")
	}
	if appSyncBatchSize < 1 || appSyncBatchSize > 20 {
		return fmt.Errorf("--batch-size must be between 1 and 20")
	}
	planBytes, err := os.ReadFile(appSyncPlanPath)
	if err != nil {
		return err
	}
	var plan appSyncPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		return fmt.Errorf("decode plan: %w", err)
	}
	contents, err := validateAppSyncPlan(args[0], plan)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Plan %s: %d upserts, %d deletes\n", plan.SHA256, len(plan.Upserts), len(plan.Deletes))
	if !appSyncApply {
		return nil
	}

	session, err := loadPlatformSession(cmd.Context(), platformHost)
	if err != nil {
		return err
	}
	org, project, err := resolvePlatformProject(cmd.Context(), session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return err
	}
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	apps, err := readPlatformApps(cmd.Context(), session, project.ID)
	if err != nil {
		return err
	}
	appID := ""
	for _, app := range apps {
		if app.ID == appLocalRef || strings.EqualFold(app.Name, appLocalRef) {
			if appID != "" {
				return fmt.Errorf("multiple apps match %q; pass an app ID", appLocalRef)
			}
			appID = app.ID
		}
	}
	if appID == "" {
		return fmt.Errorf("app %q does not exist", appLocalRef)
	}
	journalPath := appSyncJournalPath
	if journalPath == "" {
		journalPath = appSyncPlanPath + ".progress.json"
	}
	journal := appSyncJournal{SHA256: plan.SHA256, AppID: appID, Upserts: []string{}, Deletes: []string{}}
	if saved, err := os.ReadFile(journalPath); err == nil {
		if err := json.Unmarshal(saved, &journal); err != nil {
			return fmt.Errorf("decode journal: %w", err)
		}
		if journal.SHA256 != plan.SHA256 || journal.AppID != appID {
			return fmt.Errorf("journal belongs to a different plan or app")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, phase := range []struct {
		name   string
		paths  []string
		done   *[]string
		upsert bool
	}{
		{"upserts", plan.Upserts, &journal.Upserts, true},
		{"deletes", plan.Deletes, &journal.Deletes, false},
	} {
		completed := make(map[string]bool, len(*phase.done))
		for _, path := range *phase.done {
			completed[path] = true
		}
		pending := make([]string, 0, len(phase.paths))
		for _, path := range phase.paths {
			if !completed[path] {
				pending = append(pending, path)
			}
		}
		for start := 0; start < len(pending); {
			end := appSyncBatchEnd(pending, start, appSyncBatchSize, contents, phase.upsert)
			batch := pending[start:end]
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			err := executeAppSyncBatch(ctx, session.Client, project.ID, appID, batch, contents, phase.upsert)
			cancel()
			if err != nil {
				return fmt.Errorf("%s batch starting %q: %w; retry with the same journal", phase.name, batch[0], err)
			}
			*phase.done = append(*phase.done, batch...)
			if err := writeAppSyncJournal(journalPath, journal); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %d/%d\n", phase.name, len(*phase.done), len(phase.paths))
			start = end
		}
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Draft staged. Verify before promoting.")
	return nil
}

// appSyncBatchEnd keeps uploads below the hosted request-body limit while
// retaining the larger batch size for path-only deletions.
func appSyncBatchEnd(paths []string, start, limit int, contents map[string]string, upsert bool) int {
	end := min(start+limit, len(paths))
	if !upsert {
		return end
	}
	const maxContentJSONBytes = 400 << 10
	used := 0
	for i := start; i < end; i++ {
		encoded, _ := json.Marshal(contents[paths[i]])
		if i > start && used+len(encoded) > maxContentJSONBytes {
			return i
		}
		used += len(encoded)
	}
	return end
}

func validateAppSyncPlan(folder string, plan appSyncPlan) (map[string]string, error) {
	if len(plan.Upserts) == 0 || len(plan.Deletes) == 0 || len(plan.SHA256) != 64 {
		return nil, fmt.Errorf("plan must have upserts, deletes, and a SHA-256 digest")
	}
	contents := make(map[string]string, len(plan.Upserts))
	seen := make(map[string]bool, len(plan.Upserts)+len(plan.Deletes))
	hash := sha256.New()
	for _, path := range plan.Upserts {
		if !localAppFilePath(path) || seen[path] {
			return nil, fmt.Errorf("unsafe or duplicate upsert path %q", path)
		}
		seen[path] = true
		content, err := os.ReadFile(filepath.Join(folder, filepath.FromSlash(path)))
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", path, err)
		}
		contents[path] = string(content)
		_, _ = hash.Write([]byte(path))
		_, _ = hash.Write(content)
	}
	for _, path := range plan.Deletes {
		if !localAppFilePath(path) || seen[path] {
			return nil, fmt.Errorf("unsafe or duplicate delete path %q", path)
		}
		if !strings.HasPrefix(path, "pages/") || !strings.Contains(path, "/scripts/") {
			return nil, fmt.Errorf("delete path %q is outside page scripts", path)
		}
		seen[path] = true
		_, _ = hash.Write([]byte(path))
	}
	if !sort.StringsAreSorted(plan.Upserts) || !sort.StringsAreSorted(plan.Deletes) {
		return nil, fmt.Errorf("plan paths must be sorted")
	}
	if hex.EncodeToString(hash.Sum(nil)) != plan.SHA256 {
		return nil, fmt.Errorf("plan digest does not match local files")
	}
	return contents, nil
}

func executeAppSyncBatch(ctx context.Context, client *platform.Client, projectID, appID string, paths []string, contents map[string]string, upsert bool) error {
	variables := map[string]any{"projectId": projectID, "appId": appID}
	definitions := []string{"$projectId: ID!", "$appId: ID!"}
	fields := make([]string, 0, len(paths))
	for i, path := range paths {
		pathVar := fmt.Sprintf("path%d", i)
		variables[pathVar] = path
		definitions = append(definitions, "$"+pathVar+": String!")
		if upsert {
			contentVar := fmt.Sprintf("content%d", i)
			variables[contentVar] = contents[path]
			definitions = append(definitions, "$"+contentVar+": String!")
			fields = append(fields, fmt.Sprintf("a%d: UpsertAppFile(projectId:$projectId,appId:$appId,path:$%s,content:$%s){path}", i, pathVar, contentVar))
		} else {
			fields = append(fields, fmt.Sprintf("a%d: DeleteAppFile(projectId:$projectId,appId:$appId,path:$%s){Status}", i, pathVar))
		}
	}
	document := "mutation CLIAppSyncBatch(" + strings.Join(definitions, ",") + "){\n" + strings.Join(fields, "\n") + "\n}"
	var result map[string]json.RawMessage
	if err := client.ExecuteGraphQL(ctx, document, variables, &result); err != nil {
		return err
	}
	for i := range paths {
		if len(result[fmt.Sprintf("a%d", i)]) == 0 {
			return fmt.Errorf("missing response for item %d", i)
		}
	}
	return nil
}

func writeAppSyncJournal(path string, journal appSyncJournal) error {
	data, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
