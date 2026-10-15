package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/clidey/whodb/cli/internal/appfolder"
)

func initAppFromHosted(ctx context.Context, session *platformSession, projectID, ref, folder string) (*platformApp, int, error) {
	apps, err := readPlatformApps(ctx, session, projectID)
	if err != nil {
		return nil, 0, err
	}
	var app *platformApp
	for i := range apps {
		if apps[i].ID == ref || strings.EqualFold(apps[i].Name, ref) {
			if app != nil {
				return nil, 0, fmt.Errorf("multiple apps match %q; pass an app ID with --from", ref)
			}
			app = &apps[i]
		}
	}
	if app == nil {
		return nil, 0, fmt.Errorf("app %q not found", ref)
	}
	data, err := session.Client.PlatformQuery(ctx, "AppFiles", map[string]any{"projectId": projectID, "appId": app.ID})
	if err != nil {
		return nil, 0, err
	}
	var remote []platformAppFile
	if err := remarshalPlatformData(data, &remote); err != nil {
		return nil, 0, fmt.Errorf("decode app files: %w", err)
	}
	if len(remote) == 0 {
		return nil, 0, fmt.Errorf("app %q has no readable draft files", ref)
	}
	files := make(map[string]string, len(remote))
	for _, file := range remote {
		if !localAppFilePath(file.Path) {
			return nil, 0, fmt.Errorf("app contains unsupported or unsafe file path %q", file.Path)
		}
		if _, exists := files[file.Path]; exists {
			return nil, 0, fmt.Errorf("app contains duplicate file path %q", file.Path)
		}
		files[file.Path] = file.Content
	}
	manifest := map[string]json.RawMessage{}
	if content, exists := files["app.json"]; exists {
		if err := json.Unmarshal([]byte(content), &manifest); err != nil || manifest == nil {
			return nil, 0, fmt.Errorf("app.json is not a JSON object")
		}
	}
	page := ""
	if raw := manifest["defaultPage"]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &page)
	}
	if page == "" {
		pages := []string{}
		for path := range files {
			parts := strings.Split(path, "/")
			if len(parts) == 3 && parts[0] == "pages" && parts[2] == "tree.json" {
				pages = append(pages, parts[1])
			}
		}
		sort.Strings(pages)
		if len(pages) == 0 {
			return nil, 0, fmt.Errorf("app %q has no page tree", ref)
		}
		page = pages[0]
		if _, hasMain := files["pages/main/tree.json"]; hasMain {
			page = "main"
		}
	}
	ontologyNames, readOnlyNames, functionNames, err := hostedAppBindingNames(ctx, session, projectID, app)
	if err != nil {
		return nil, 0, err
	}
	for key, value := range map[string]any{
		"name": app.Name, "description": app.Description, "defaultPage": page,
		"ontologies": ontologyNames, "readOnlyOntologies": readOnlyNames, "functions": functionNames,
	} {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, 0, err
		}
		manifest[key] = encoded
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, 0, err
	}
	files["app.json"] = string(encoded) + "\n"
	if err := writeImportedAppFolder(folder, files); err != nil {
		return nil, 0, err
	}
	return app, len(files), nil
}

func hostedAppBindingNames(ctx context.Context, session *platformSession, projectID string, app *platformApp) ([]string, []string, []string, error) {
	ontologyNames := map[string]string{}
	if len(app.OntologyIDs)+len(app.ReadOnlyOntologyIDs) > 0 {
		entities, err := session.Client.Ontologies(ctx, projectID)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, entity := range entities {
			ontologyNames[entity.ID] = entity.APIName
		}
	}
	functionNames := map[string]string{}
	if len(app.FunctionIDs) > 0 {
		functions, err := session.Client.Functions(ctx, projectID, []string{"id", "name"})
		if err != nil {
			return nil, nil, nil, err
		}
		for _, function := range functions {
			functionNames[function.ID] = function.Name
		}
	}
	mapNames := func(ids []string, names map[string]string) []string {
		result := make([]string, 0, len(ids))
		for _, id := range ids {
			name := names[id]
			if name == "" {
				name = id
			}
			result = append(result, name)
		}
		return result
	}
	return mapNames(app.OntologyIDs, ontologyNames), mapNames(app.ReadOnlyOntologyIDs, ontologyNames), mapNames(app.FunctionIDs, functionNames), nil
}

func localAppFilePath(path string) bool {
	if path == "" || strings.Contains(path, "\\") || !filepath.IsLocal(filepath.FromSlash(path)) {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || strings.HasPrefix(part, ".") {
			return false
		}
	}
	return path == "app.json" || strings.HasPrefix(path, "pages/") || strings.HasPrefix(path, "shared/")
}

func writeImportedAppFolder(folder string, files map[string]string) error {
	if _, err := os.Lstat(folder); err == nil {
		return fmt.Errorf("%s already exists", folder)
	} else if !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(folder)
	base := filepath.Base(filepath.Clean(folder))
	staging, err := os.MkdirTemp(parent, "."+base+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	for path, content := range files {
		local := filepath.Join(staging, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(local, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if _, err := appfolder.Read(staging); err != nil {
		return fmt.Errorf("hosted app cannot be used as a local app: %w", err)
	}
	if _, err := os.Lstat(folder); err == nil {
		return fmt.Errorf("%s already exists", folder)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(staging, folder)
}
