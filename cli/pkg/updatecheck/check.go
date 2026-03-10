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

package updatecheck

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	goversion "github.com/hashicorp/go-version"
)

const (
	githubReleasesURL = "https://api.github.com/repos/clidey/whodb/releases/latest"
	cacheTTL          = 24 * time.Hour
	httpTimeout       = 5 * time.Second
)

// Result holds the outcome of an update check.
type Result struct {
	LatestVersion   string
	UpdateAvailable bool
}

type cacheFile struct {
	LastCheck     time.Time `json:"lastCheck"`
	LatestVersion string    `json:"latestVersion"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// Check looks for a newer version of whodb-cli using a file-based 24h cache.
// Returns nil if no update is available, the check is suppressed, or any error occurs.
func Check(currentVersion string) *Result {
	if currentVersion == "" || currentVersion == "dev" {
		return nil
	}

	if os.Getenv("WHODB_DISABLE_UPDATE_CHECK") == "true" {
		return nil
	}

	cacheDir := getCacheDir()
	cachePath := filepath.Join(cacheDir, "update-check.json")

	// Try to read cache
	if cached, err := readCache(cachePath); err == nil {
		if time.Since(cached.LastCheck) < cacheTTL {
			return compareVersions(currentVersion, cached.LatestVersion)
		}
	}

	// Cache miss or stale — fetch from GitHub
	latestVersion := currentVersion
	if release, err := fetchLatestRelease(); err == nil {
		latestVersion = release.TagName
	}

	// Write cache (best-effort), even on failure to avoid retrying
	_ = os.MkdirAll(cacheDir, 0700)
	_ = writeCache(cachePath, &cacheFile{
		LastCheck:     time.Now(),
		LatestVersion: latestVersion,
	})

	return compareVersions(currentVersion, latestVersion)
}

func compareVersions(currentVersion, latestTag string) *Result {
	latestClean := strings.TrimPrefix(latestTag, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	current, err := goversion.NewVersion(currentClean)
	if err != nil {
		return nil
	}

	latest, err := goversion.NewVersion(latestClean)
	if err != nil {
		return nil
	}

	if latest.GreaterThan(current) {
		return &Result{
			LatestVersion:   latestTag,
			UpdateAvailable: true,
		}
	}

	return nil
}

func getCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	return filepath.Join(home, ".whodb-cli")
}

func readCache(path string) (*cacheFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c cacheFile
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func writeCache(path string, c *cacheFile) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func fetchLatestRelease() (*githubRelease, error) {
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(githubReleasesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status")
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}
