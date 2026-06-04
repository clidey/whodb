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

package gcp

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/clidey/whodb/core/src/log"
)

// LocalProject represents a GCP project discovered from local configuration.
type LocalProject struct {
	ProjectID string
	Name      string
	Source    string // "environment", "gcloud-config", "service-account"
	IsDefault bool
}

// DiscoverLocalProjects scans environment variables and gcloud CLI configuration
// for available GCP projects.
func DiscoverLocalProjects() ([]LocalProject, error) {
	projects := make(map[string]*LocalProject)

	// Check environment variables
	for _, envVar := range []string{"GOOGLE_CLOUD_PROJECT", "GCLOUD_PROJECT", "CLOUDSDK_CORE_PROJECT"} {
		if projectID := os.Getenv(envVar); projectID != "" {
			if _, exists := projects[projectID]; !exists {
				projects[projectID] = &LocalProject{
					ProjectID: projectID,
					Name:      projectID,
					Source:    "environment",
					IsDefault: true,
				}
			}
		}
	}

	// Check GOOGLE_APPLICATION_CREDENTIALS for project ID in service account JSON
	if saPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); saPath != "" {
		if project := parseServiceAccountProject(saPath); project != nil {
			if _, exists := projects[project.ProjectID]; !exists {
				projects[project.ProjectID] = project
			}
		}
	}

	// Check gcloud CLI configuration
	gcloudProjects := discoverGcloudProjects()
	for _, p := range gcloudProjects {
		if _, exists := projects[p.ProjectID]; !exists {
			projects[p.ProjectID] = &p
		}
	}

	result := make([]LocalProject, 0, len(projects))
	for _, project := range projects {
		result = append(result, *project)
	}

	return result, nil
}

// parseServiceAccountProject reads a service account JSON key file and extracts the project ID.
func parseServiceAccountProject(path string) *LocalProject {
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		log.Debugf("GCP projects: failed to read service account file %s: %v", path, err)
		return nil
	}

	var sa struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(data, &sa); err != nil {
		log.Debugf("GCP projects: failed to parse service account file %s: %v", path, err)
		return nil
	}

	if sa.ProjectID == "" {
		return nil
	}

	return &LocalProject{
		ProjectID: sa.ProjectID,
		Name:      sa.ProjectID,
		Source:    "service-account",
		IsDefault: false,
	}
}

// discoverGcloudProjects reads gcloud CLI configuration files for project settings.
func discoverGcloudProjects() []LocalProject {
	configDir := getGcloudConfigDir()
	if configDir == "" {
		return nil
	}

	var projects []LocalProject

	// Read the default properties file
	propertiesPath := filepath.Join(configDir, "properties")
	if projectID := parseGcloudProperties(propertiesPath); projectID != "" {
		projects = append(projects, LocalProject{
			ProjectID: projectID,
			Name:      projectID,
			Source:    "gcloud-config",
			IsDefault: true,
		})
	}

	// Read named configurations
	configurationsDir := filepath.Join(configDir, "configurations")
	entries, err := os.ReadDir(configurationsDir)
	if err != nil {
		return projects
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "config_") {
			continue
		}
		configPath := filepath.Join(configurationsDir, entry.Name())
		if projectID := parseGcloudProperties(configPath); projectID != "" {
			name := strings.TrimPrefix(entry.Name(), "config_")
			projects = append(projects, LocalProject{
				ProjectID: projectID,
				Name:      name,
				Source:    "gcloud-config",
				IsDefault: false,
			})
		}
	}

	return projects
}

// parseGcloudProperties reads a gcloud properties/config file and extracts the project ID.
func parseGcloudProperties(path string) string {
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	inCoreSection := false
	projectRegex := regexp.MustCompile(`^\s*project\s*=\s*(.+?)\s*$`)
	sectionRegex := regexp.MustCompile(`^\s*\[([^\]]+)\]\s*$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}

		if matches := sectionRegex.FindStringSubmatch(line); matches != nil {
			inCoreSection = matches[1] == "core"
			continue
		}

		if inCoreSection {
			if matches := projectRegex.FindStringSubmatch(line); matches != nil {
				return matches[1]
			}
		}
	}

	return ""
}

func getGcloudConfigDir() string {
	if dir := os.Getenv("CLOUDSDK_CONFIG"); dir != "" {
		return dir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Warnf("Unable to determine home directory for gcloud config: %v", err)
		return ""
	}
	return filepath.Join(homeDir, ".config", "gcloud")
}
