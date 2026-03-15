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

package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	goversion "github.com/hashicorp/go-version"
)

const (
	githubReleasesURL = "https://api.github.com/repos/clidey/whodb/releases/latest"
	cacheTTL          = 24 * time.Hour
	httpTimeout       = 5 * time.Second
)

// UpdateInfo holds the result of an update check.
type UpdateInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
}

type cachedResult struct {
	info      UpdateInfo
	checkedAt time.Time
}

var (
	cache   *cachedResult
	cacheMu sync.Mutex
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdate checks whether a newer version of WhoDB is available.
// Results are cached in memory for 24 hours. When disabled is true, no
// network request is made and UpdateAvailable is always false.
// On any error, returns UpdateAvailable: false
func CheckForUpdate(currentVersion string, disabled bool) UpdateInfo {
	noUpdate := UpdateInfo{
		CurrentVersion:  currentVersion,
		LatestVersion:   currentVersion,
		UpdateAvailable: false,
	}

	if disabled || currentVersion == "" || currentVersion == "development" {
		return noUpdate
	}

	cacheMu.Lock()
	if cache != nil && time.Since(cache.checkedAt) < cacheTTL {
		result := cache.info
		cacheMu.Unlock()
		return result
	}
	cacheMu.Unlock()

	result := noUpdate

	if release, err := fetchLatestRelease(); err == nil {
		latestTag := strings.TrimPrefix(release.TagName, "v")
		currentClean := strings.TrimPrefix(currentVersion, "v")

		current, currErr := goversion.NewVersion(currentClean)
		latest, latErr := goversion.NewVersion(latestTag)

		if currErr == nil && latErr == nil && latest.GreaterThan(current) {
			result = UpdateInfo{
				CurrentVersion:  currentVersion,
				LatestVersion:   release.TagName,
				UpdateAvailable: true,
				ReleaseURL:      fmt.Sprintf("https://github.com/clidey/whodb/releases/tag/%s", release.TagName),
			}
		}
	}

	cacheMu.Lock()
	cache = &cachedResult{info: result, checkedAt: time.Now()}
	cacheMu.Unlock()

	return result
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
