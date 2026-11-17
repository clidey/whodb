package appfolder

import (
	"fmt"
	"path"
	"strings"
)

// RuntimeMetadata declares explicitly loaded shared resources for each page.
// A nil SharedAssets map keeps the existing all-shared-resources behavior.
type RuntimeMetadata struct {
	SharedAssets map[string][]string `json:"sharedAssets,omitempty"`
}

func checkRuntimeAssets(workspace *Workspace, files map[string]string) error {
	if workspace.Manifest.Runtime == nil || workspace.Manifest.Runtime.SharedAssets == nil {
		return nil
	}
	declarations := workspace.Manifest.Runtime.SharedAssets
	pages := make(map[string]bool, len(workspace.Pages))
	for _, page := range workspace.Pages {
		pages[page] = true
	}
	for page, assets := range declarations {
		if page != "*" && !pages[page] {
			return fmt.Errorf("app.json: shared assets refer to unknown page %q", page)
		}
		for _, asset := range assets {
			if _, ok := files[asset]; !ok || !strings.HasPrefix(asset, "shared/") || path.Clean(asset) != asset {
				return fmt.Errorf("app.json: shared asset %q for %q must identify an existing shared file", asset, page)
			}
		}
	}
	for _, page := range workspace.Pages {
		if _, ok := declarations[page]; !ok {
			return fmt.Errorf("app.json: shared assets must explicitly declare page %q", page)
		}
	}
	return nil
}
