// Package appfolder reads and checks a version-controlled WhoDB app workspace.
package appfolder

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// File is one text file sent to the app runtime.
type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Manifest is the app.json metadata used by the local CLI.
type Manifest struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	DefaultPage        string   `json:"defaultPage"`
	Ontologies         []string `json:"ontologies"`
	ReadOnlyOntologies []string `json:"readOnlyOntologies"`
	Functions          []string `json:"functions"`
}

// Workspace is a checked app folder and its publishable files.
type Workspace struct {
	Manifest Manifest `json:"manifest"`
	Files    []File   `json:"files"`
	Pages    []string `json:"pages"`
}

type pageTree struct {
	Version int       `json:"version"`
	Root    *treeNode `json:"root"`
	Effects []struct {
		Handler string `json:"handler"`
	} `json:"effects"`
}

type treeNode struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	On       map[string]string `json:"on"`
	Children []*treeNode       `json:"children"`
}

// Read loads a WhoDB app folder and reports invalid references before upload.
func Read(dir string) (*Workspace, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	workspace := &Workspace{Files: []File{}, Pages: []string{}}
	contents := map[string]string{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if relative != "app.json" && relative != "pages" && relative != "shared" && !strings.HasPrefix(relative, "pages/") && !strings.HasPrefix(relative, "shared/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink; app files must be regular files", relative)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("%s is not a regular file", relative)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.Valid(data) {
			return fmt.Errorf("%s is not UTF-8 text", relative)
		}
		contents[relative] = string(data)
		workspace.Files = append(workspace.Files, File{Path: relative, Content: string(data)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := jsonFile(contents, "app.json", &workspace.Manifest); err != nil {
		return nil, err
	}
	if strings.TrimSpace(workspace.Manifest.Name) == "" {
		return nil, fmt.Errorf("app.json: name is required")
	}
	if strings.TrimSpace(workspace.Manifest.DefaultPage) == "" {
		return nil, fmt.Errorf("app.json: defaultPage is required")
	}
	if strings.Contains(workspace.Manifest.DefaultPage, "/") || strings.HasPrefix(workspace.Manifest.DefaultPage, ".") {
		return nil, fmt.Errorf("app.json: invalid defaultPage %q", workspace.Manifest.DefaultPage)
	}
	for path := range contents {
		parts := strings.Split(path, "/")
		if len(parts) == 3 && parts[0] == "pages" && parts[2] == "tree.json" {
			workspace.Pages = append(workspace.Pages, parts[1])
		}
		if strings.HasSuffix(path, "/state.json") {
			var state any
			if err := jsonFile(contents, path, &state); err != nil {
				return nil, err
			}
		}
	}
	sort.Strings(workspace.Pages)
	if len(workspace.Pages) == 0 {
		return nil, fmt.Errorf("pages/<page>/tree.json is required")
	}
	if _, ok := contents["pages/"+workspace.Manifest.DefaultPage+"/tree.json"]; !ok {
		return nil, fmt.Errorf("app.json: defaultPage %q has no tree.json", workspace.Manifest.DefaultPage)
	}
	for _, page := range workspace.Pages {
		path := "pages/" + page + "/tree.json"
		var tree pageTree
		if err := jsonFile(contents, path, &tree); err != nil {
			return nil, err
		}
		if tree.Version != 1 || tree.Root == nil {
			return nil, fmt.Errorf("%s: expected version 1 and a root node", path)
		}
		seen := map[string]bool{}
		var visit func(*treeNode) error
		visit = func(node *treeNode) error {
			if node == nil || node.ID == "" || node.Type == "" {
				return fmt.Errorf("%s: every node needs id and type", path)
			}
			if seen[node.ID] {
				return fmt.Errorf("%s: duplicate node id %q", path, node.ID)
			}
			seen[node.ID] = true
			for event, handler := range node.On {
				if err := checkHandler(contents, page, handler); err != nil {
					return fmt.Errorf("%s: node %q event %q: %w", path, node.ID, event, err)
				}
			}
			for _, child := range node.Children {
				if err := visit(child); err != nil {
					return err
				}
			}
			return nil
		}
		if err := visit(tree.Root); err != nil {
			return nil, err
		}
		for _, effect := range tree.Effects {
			if err := checkHandler(contents, page, effect.Handler); err != nil {
				return nil, fmt.Errorf("%s: effect: %w", path, err)
			}
		}
	}
	sort.Slice(workspace.Files, func(i, j int) bool { return workspace.Files[i].Path < workspace.Files[j].Path })
	return workspace, nil
}

func jsonFile(files map[string]string, path string, target any) error {
	content, ok := files[path]
	if !ok {
		return fmt.Errorf("%s is required", path)
	}
	if err := json.Unmarshal([]byte(content), target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func checkHandler(files map[string]string, page, handler string) error {
	if handler == "" {
		return fmt.Errorf("handler name is empty")
	}
	for _, prefix := range []string{"open_", "close_", "toggle_"} {
		if strings.HasPrefix(handler, prefix) && len(handler) > len(prefix) {
			return nil
		}
	}
	if strings.HasPrefix(handler, "set_") {
		parts := strings.SplitN(strings.TrimPrefix(handler, "set_"), "_", 2)
		if len(parts) == 2 && parts[0] != "" {
			return nil
		}
	}
	if strings.HasPrefix(handler, "nav_") && len(handler) > len("nav_") {
		if _, ok := files["pages/"+strings.TrimPrefix(handler, "nav_")+"/tree.json"]; !ok {
			return fmt.Errorf("navigation target %q has no tree.json", strings.TrimPrefix(handler, "nav_"))
		}
		return nil
	}
	if _, ok := files["pages/"+page+"/scripts/"+handler+".js"]; !ok {
		return fmt.Errorf("script %q is missing at pages/%s/scripts/%s.js", handler, page, handler)
	}
	return nil
}
