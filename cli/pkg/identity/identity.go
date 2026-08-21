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

// Package identity defines the edition-specific runtime identity used by the
// CLI for display text, local storage paths, and service names.
package identity

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	defaultCommandName        = "whodb"
	defaultDisplayName        = "WhoDB CLI"
	defaultHomeDirName        = ".whodb-cli"
	defaultViperEnvPrefix     = "WHODB_CLI"
	defaultAnalyticsDisabled  = "WHODB_CLI_ANALYTICS_DISABLED"
	defaultRuntimeEnv         = "WHODB_CLI"
	defaultKeyringService     = "WhoDB-CLI"
	defaultIssueURL           = "https://github.com/clidey/whodb/issues/new?template=bug_report.md"
	defaultUpdateCheckAPIURL  = "https://api.github.com/repos/clidey/whodb/releases/latest"
	defaultUpdateCheckPageURL = "https://github.com/clidey/whodb/releases/latest"
)

// Config describes the edition-specific CLI identity.
type Config struct {
	Edition              string
	CommandName          string
	DisplayName          string
	VersionName          string
	HomeDirName          string
	ViperEnvPrefix       string
	AnalyticsDisabledEnv string
	RuntimeEnv           string
	KeyringService       string
	IssueURL             string
	UpdateCheckAPIURL    string
	UpdateCheckPageURL   string
	RootLongAppend       string
}

var (
	currentMu sync.RWMutex
	current   = CE()
)

// CE returns the default Community Edition CLI identity.
func CE() Config {
	return Config{
		Edition:              "ce",
		CommandName:          defaultCommandName,
		DisplayName:          defaultDisplayName,
		VersionName:          defaultCommandName,
		HomeDirName:          defaultHomeDirName,
		ViperEnvPrefix:       defaultViperEnvPrefix,
		AnalyticsDisabledEnv: defaultAnalyticsDisabled,
		RuntimeEnv:           defaultRuntimeEnv,
		KeyringService:       defaultKeyringService,
		IssueURL:             defaultIssueURL,
		UpdateCheckAPIURL:    defaultUpdateCheckAPIURL,
		UpdateCheckPageURL:   defaultUpdateCheckPageURL,
	}
}

// Current returns the active CLI identity.
func Current() Config {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return current
}

// SetCurrent sets the active CLI identity for the current process.
func SetCurrent(cfg Config) {
	currentMu.Lock()
	defer currentMu.Unlock()
	current = normalize(cfg)
}

// ReplaceText applies the active CLI identity to user-facing text.
func ReplaceText(text string) string {
	cfg := Current()
	replacer := strings.NewReplacer(
		"WhoDB CLI", cfg.DisplayName,
		".whodb-cli", cfg.HomeDirName,
	)
	return replacer.Replace(text)
}

// HomePath returns a path rooted under the edition-specific CLI home
// directory in the user's home folder.
func HomePath(parts ...string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	allParts := []string{homeDir, Current().HomeDirName}
	allParts = append(allParts, parts...)
	return filepath.Join(allParts...), nil
}

func normalize(cfg Config) Config {
	base := CE()

	if strings.TrimSpace(cfg.Edition) != "" {
		base.Edition = cfg.Edition
	}
	if strings.TrimSpace(cfg.CommandName) != "" {
		base.CommandName = cfg.CommandName
	}
	if strings.TrimSpace(cfg.DisplayName) != "" {
		base.DisplayName = cfg.DisplayName
	}
	if strings.TrimSpace(cfg.VersionName) != "" {
		base.VersionName = cfg.VersionName
	}
	if strings.TrimSpace(cfg.HomeDirName) != "" {
		base.HomeDirName = cfg.HomeDirName
	}
	if strings.TrimSpace(cfg.ViperEnvPrefix) != "" {
		base.ViperEnvPrefix = cfg.ViperEnvPrefix
	}
	if strings.TrimSpace(cfg.AnalyticsDisabledEnv) != "" {
		base.AnalyticsDisabledEnv = cfg.AnalyticsDisabledEnv
	}
	if strings.TrimSpace(cfg.RuntimeEnv) != "" {
		base.RuntimeEnv = cfg.RuntimeEnv
	}
	if strings.TrimSpace(cfg.KeyringService) != "" {
		base.KeyringService = cfg.KeyringService
	}
	if strings.TrimSpace(cfg.IssueURL) != "" {
		base.IssueURL = cfg.IssueURL
	}
	if strings.TrimSpace(cfg.UpdateCheckAPIURL) != "" {
		base.UpdateCheckAPIURL = cfg.UpdateCheckAPIURL
	}
	if strings.TrimSpace(cfg.UpdateCheckPageURL) != "" {
		base.UpdateCheckPageURL = cfg.UpdateCheckPageURL
	}
	if strings.TrimSpace(cfg.RootLongAppend) != "" {
		base.RootLongAppend = cfg.RootLongAppend
	}

	return base
}
