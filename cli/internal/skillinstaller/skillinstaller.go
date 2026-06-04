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

// Package skillinstaller installs bundled WhoDB assistant skills and integrations.
package skillinstaller

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	whodbplugin "github.com/clidey/whodb/cli/external-plugin/whodb"
	"go.yaml.in/yaml/v3"
)

// Item describes one bundled skill or agent.
type Item struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// InstallOptions configures a skill or assistant integration installation.
type InstallOptions struct {
	Name          string
	Target        string
	TargetDir     string
	AgentsDir     string
	IncludeAgents bool
	Force         bool
	DryRun        bool
}

// InstallResult describes files written by an install operation.
type InstallResult struct {
	DryRun     bool            `json:"dryRun,omitempty"`
	Skills     []InstalledFile `json:"skills,omitempty"`
	Agents     []InstalledFile `json:"agents,omitempty"`
	Configs    []InstalledFile `json:"configs,omitempty"`
	Rules      []InstalledFile `json:"rules,omitempty"`
	Extensions []InstalledFile `json:"extensions,omitempty"`
}

// InstalledFile records one installed asset.
type InstalledFile struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Action     string `json:"action,omitempty"`
	BackupPath string `json:"backupPath,omitempty"`
}

// List returns all bundled skills and agents.
func List() ([]Item, error) {
	items := make([]Item, 0)

	skills, err := skillNames()
	if err != nil {
		return nil, err
	}
	for _, name := range skills {
		description, _ := readSkillDescription(name)
		items = append(items, Item{Name: name, Type: "skill", Description: description})
	}

	agents, err := agentNames()
	if err != nil {
		return nil, err
	}
	for _, name := range agents {
		items = append(items, Item{Name: name, Type: "agent"})
	}

	slices.SortFunc(items, func(a, b Item) int {
		if a.Type != b.Type {
			return strings.Compare(a.Type, b.Type)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return items, nil
}

// Install writes bundled skills, agents, or assistant integration files.
func Install(opts InstallOptions) (InstallResult, error) {
	if isIntegrationTarget(opts.Target) {
		return installIntegrationTarget(opts)
	}

	targetDir, agentsDir, err := resolveTargetDirs(opts)
	if err != nil {
		return InstallResult{}, err
	}
	if opts.IncludeAgents && strings.TrimSpace(agentsDir) == "" {
		return InstallResult{}, fmt.Errorf("--include-agents requires --target claude-code or --agents-dir")
	}

	names, err := selectedSkillNames(opts.Name)
	if err != nil {
		return InstallResult{}, err
	}

	result := InstallResult{DryRun: opts.DryRun, Skills: make([]InstalledFile, 0, len(names))}
	for _, name := range names {
		path, err := installSkill(name, targetDir, opts.Force, opts.DryRun)
		if err != nil {
			return result, err
		}
		item, err := installedFile(name, path, opts.DryRun, false)
		if err != nil {
			return result, err
		}
		result.Skills = append(result.Skills, item)
	}

	if opts.IncludeAgents {
		agents, err := agentNames()
		if err != nil {
			return result, err
		}
		for _, name := range agents {
			path, err := installAgent(name, agentsDir, opts.Force, opts.DryRun)
			if err != nil {
				return result, err
			}
			item, err := installedFile(name, path, opts.DryRun, false)
			if err != nil {
				return result, err
			}
			result.Agents = append(result.Agents, item)
		}
	}

	return result, nil
}

func resolveTargetDirs(opts InstallOptions) (string, string, error) {
	if strings.TrimSpace(opts.TargetDir) != "" {
		agentsDir := opts.AgentsDir
		if opts.IncludeAgents && strings.TrimSpace(agentsDir) == "" {
			return "", "", fmt.Errorf("--agents-dir is required when --include-agents is used with --target-dir")
		}
		return opts.TargetDir, agentsDir, nil
	}

	target := strings.TrimSpace(opts.Target)
	if target == "" {
		return "", "", fmt.Errorf("provide --target or --target-dir")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	switch target {
	case "codex":
		return filepath.Join(home, ".codex", "skills"), opts.AgentsDir, nil
	case "claude-code":
		agentsDir := opts.AgentsDir
		if strings.TrimSpace(agentsDir) == "" {
			agentsDir = filepath.Join(home, ".claude", "agents")
		}
		return filepath.Join(home, ".claude", "skills"), agentsDir, nil
	default:
		return "", "", fmt.Errorf("unsupported target %q", target)
	}
}

func isIntegrationTarget(target string) bool {
	switch strings.TrimSpace(target) {
	case "cursor", "vscode", "github-copilot", "gemini-cli", "windsurf", "opencode", "cline", "zed", "continue", "aider":
		return true
	default:
		return false
	}
}

func installIntegrationTarget(opts InstallOptions) (InstallResult, error) {
	if strings.TrimSpace(opts.TargetDir) != "" {
		return InstallResult{}, fmt.Errorf("--target-dir installs skills directly; omit it when installing --target %s", opts.Target)
	}
	if strings.TrimSpace(opts.Name) != "" && opts.Name != "all" {
		return InstallResult{}, fmt.Errorf("--target %s installs the whole WhoDB integration; omit the skill name", opts.Target)
	}
	if opts.IncludeAgents && strings.TrimSpace(opts.AgentsDir) == "" {
		return InstallResult{}, fmt.Errorf("--include-agents is only supported for claude-code or a custom --agents-dir")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return InstallResult{}, err
	}

	result := InstallResult{DryRun: opts.DryRun}
	switch opts.Target {
	case "cursor":
		path := filepath.Join(home, ".cursor", "mcp.json")
		item, err := installedFile("cursor-mcp", path, opts.DryRun, true)
		if err != nil {
			return result, err
		}
		if err := mergeJSONSection(path, "mcpServers", "whodb", stdioMCPServer(false), opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		result.Configs = append(result.Configs, item)
	case "vscode":
		path, err := vscodeMCPConfigPath()
		if err != nil {
			return result, err
		}
		item, err := installedFile("vscode-mcp", path, opts.DryRun, true)
		if err != nil {
			return result, err
		}
		if err := mergeJSONSection(path, "servers", "whodb", stdioMCPServer(true), opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		result.Configs = append(result.Configs, item)
	case "github-copilot":
		path := filepath.Join(home, ".copilot", "mcp-config.json")
		server := stdioMCPServer(true)
		server["tools"] = []string{"*"}
		item, err := installedFile("github-copilot-mcp", path, opts.DryRun, true)
		if err != nil {
			return result, err
		}
		if err := mergeJSONSection(path, "mcpServers", "whodb", server, opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		result.Configs = append(result.Configs, item)
	case "gemini-cli":
		extensionDir := filepath.Join(home, ".gemini", "extensions", "whodb")
		manifestPath := filepath.Join(extensionDir, "gemini-extension.json")
		contextPath := filepath.Join(extensionDir, "GEMINI.md")
		item, err := installedFile("gemini-cli-extension", extensionDir, opts.DryRun, false)
		if err != nil {
			return result, err
		}
		if err := writeJSONFile(manifestPath, geminiExtensionManifest(), opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		if err := writeFile(contextPath, []byte(geminiContext()), opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		result.Extensions = append(result.Extensions, item)
	case "windsurf":
		path := filepath.Join(home, ".codeium", "mcp_config.json")
		item, err := installedFile("windsurf-mcp", path, opts.DryRun, true)
		if err != nil {
			return result, err
		}
		if err := mergeJSONSection(path, "mcpServers", "whodb", stdioMCPServer(false), opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		result.Configs = append(result.Configs, item)
	case "opencode":
		path := filepath.Join(home, ".config", "opencode", "opencode.json")
		item, err := installedFile("opencode-mcp", path, opts.DryRun, true)
		if err != nil {
			return result, err
		}
		if err := mergeOpenCodeConfig(path, opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		result.Configs = append(result.Configs, item)
	case "cline":
		mcpPath := filepath.Join(home, ".cline", "data", "settings", "cline_mcp_settings.json")
		configItem, err := installedFile("cline-mcp", mcpPath, opts.DryRun, true)
		if err != nil {
			return result, err
		}
		if err := mergeJSONSection(mcpPath, "mcpServers", "whodb", clineMCPServer(), opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		rulePath := filepath.Join(home, "Documents", "Cline", "Rules", "whodb.md")
		ruleItem, err := installedFile("cline-rule", rulePath, opts.DryRun, false)
		if err != nil {
			return result, err
		}
		if err := writeFile(rulePath, []byte(assistantRuleMarkdown()), opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		result.Configs = append(result.Configs, configItem)
		result.Rules = append(result.Rules, ruleItem)
	case "zed":
		path, err := zedSettingsPath()
		if err != nil {
			return result, err
		}
		item, err := installedFile("zed-context-server", path, opts.DryRun, true)
		if err != nil {
			return result, err
		}
		if err := mergeJSONSection(path, "context_servers", "whodb", stdioMCPServer(false), opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		result.Configs = append(result.Configs, item)
	case "continue":
		path := filepath.Join(home, ".continue", "config.yaml")
		item, err := installedFile("continue-config", path, opts.DryRun, true)
		if err != nil {
			return result, err
		}
		if err := mergeContinueConfig(path, opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		result.Configs = append(result.Configs, item)
	case "aider":
		conventionsPath := filepath.Join(home, ".aider", "whodb-conventions.md")
		ruleItem, err := installedFile("aider-conventions", conventionsPath, opts.DryRun, false)
		if err != nil {
			return result, err
		}
		if err := writeFile(conventionsPath, []byte(assistantRuleMarkdown()), opts.Force, opts.DryRun); err != nil {
			return result, err
		}
		configPath := filepath.Join(home, ".aider.conf.yml")
		configItem, err := installedFile("aider-config", configPath, opts.DryRun, true)
		if err != nil {
			return result, err
		}
		if err := mergeAiderConfig(configPath, conventionsPath, opts.DryRun); err != nil {
			return result, err
		}
		result.Configs = append(result.Configs, configItem)
		result.Rules = append(result.Rules, ruleItem)
	default:
		return result, fmt.Errorf("unsupported target %q", opts.Target)
	}

	if opts.IncludeAgents {
		agents, err := agentNames()
		if err != nil {
			return result, err
		}
		for _, name := range agents {
			path, err := installAgent(name, opts.AgentsDir, opts.Force, opts.DryRun)
			if err != nil {
				return result, err
			}
			item, err := installedFile(name, path, opts.DryRun, false)
			if err != nil {
				return result, err
			}
			result.Agents = append(result.Agents, item)
		}
	}

	return result, nil
}

func stdioMCPServer(includeType bool) map[string]any {
	server := map[string]any{
		"command": "npx",
		"args":    []string{"-y", "@clidey/whodb-cli", "mcp", "serve"},
	}
	if includeType {
		server["type"] = "stdio"
	}
	return server
}

func clineMCPServer() map[string]any {
	server := stdioMCPServer(false)
	server["disabled"] = false
	return server
}

func geminiExtensionManifest() map[string]any {
	return map[string]any{
		"name":            "whodb",
		"version":         "1.0.0",
		"contextFileName": "GEMINI.md",
		"mcpServers": map[string]any{
			"whodb": stdioMCPServer(false),
		},
	}
}

func geminiContext() string {
	return assistantRuleMarkdown()
}

func assistantRuleMarkdown() string {
	return `# WhoDB Database Assistance

Use the WhoDB MCP server for database work. Start with ` + "`whodb_connections`" + ` to find available connections, then inspect schemas with ` + "`whodb_schemas`" + `, ` + "`whodb_tables`" + `, and ` + "`whodb_columns`" + ` before writing queries.

Prefer read-only exploration queries with explicit limits while investigating data. Use ` + "`whodb_explain`" + ` for query plans, ` + "`whodb_erd`" + ` for relationship metadata, ` + "`whodb_audit`" + ` for data-quality checks, and ` + "`whodb_diff`" + ` for schema comparisons.
`
}

func vscodeMCPConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "Code", "User", "mcp.json"), nil
}

func zedSettingsPath() (string, error) {
	if runtime.GOOS == "linux" && os.Getenv("XDG_CONFIG_HOME") != "" {
		configHome := os.Getenv("XDG_CONFIG_HOME")
		return filepath.Join(configHome, "zed", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zed", "settings.json"), nil
}

func installedFile(name, path string, dryRun, includeBackup bool) (InstalledFile, error) {
	item := InstalledFile{Name: name, Path: path}
	if !dryRun {
		return item, nil
	}

	exists, err := fileExists(path)
	if err != nil {
		return item, err
	}
	if exists {
		item.Action = "update"
		if includeBackup {
			item.BackupPath = path + ".whodb.bak"
		}
	} else {
		item.Action = "create"
	}
	return item, nil
}

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func mergeJSONSection(path, section, name string, value map[string]any, force, dryRun bool) error {
	config, err := readJSONObject(path)
	if err != nil {
		return err
	}

	sectionValue, ok := config[section]
	if !ok {
		sectionValue = map[string]any{}
		config[section] = sectionValue
	}
	sectionMap, ok := sectionValue.(map[string]any)
	if !ok {
		return fmt.Errorf("%s contains non-object %q", path, section)
	}
	if _, exists := sectionMap[name]; exists && !force {
		return fmt.Errorf("%s already contains %s.%s; use --force to overwrite", path, section, name)
	}
	sectionMap[name] = value

	return writeJSONFile(path, config, true, dryRun)
}

func mergeOpenCodeConfig(path string, force, dryRun bool) error {
	config, err := readJSONObject(path)
	if err != nil {
		return err
	}
	if _, ok := config["$schema"]; !ok {
		config["$schema"] = "https://opencode.ai/config.json"
	}

	mcpValue, ok := config["mcp"]
	if !ok {
		mcpValue = map[string]any{}
		config["mcp"] = mcpValue
	}
	mcpMap, ok := mcpValue.(map[string]any)
	if !ok {
		return fmt.Errorf("%s contains non-object %q", path, "mcp")
	}
	if _, exists := mcpMap["whodb"]; exists && !force {
		return fmt.Errorf("%s already contains mcp.whodb; use --force to overwrite", path)
	}
	mcpMap["whodb"] = map[string]any{
		"type":    "local",
		"command": []string{"npx", "-y", "@clidey/whodb-cli", "mcp", "serve"},
		"enabled": true,
	}

	return writeJSONFile(path, config, true, dryRun)
}

func readJSONObject(path string) (map[string]any, error) {
	config := map[string]any{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return config, nil
	}
	if err := json.Unmarshal(data, &config); err != nil {
		data, err = stripJSONC(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s as JSON or JSONC: %w", path, err)
		}
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse %s as JSON or JSONC: %w", path, err)
		}
	}
	return config, nil
}

func stripJSONC(data []byte) ([]byte, error) {
	stripped := make([]byte, 0, len(data))
	inString := false
	escaped := false

	for index := 0; index < len(data); index++ {
		character := data[index]
		if inString {
			stripped = append(stripped, character)
			if escaped {
				escaped = false
				continue
			}
			switch character {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch character {
		case '"':
			inString = true
			stripped = append(stripped, character)
		case '/':
			if index+1 >= len(data) {
				stripped = append(stripped, character)
				continue
			}
			switch data[index+1] {
			case '/':
				index += 2
				for index < len(data) && data[index] != '\n' && data[index] != '\r' {
					index++
				}
				if index < len(data) {
					stripped = append(stripped, data[index])
				}
			case '*':
				index += 2
				for index+1 < len(data) && (data[index] != '*' || data[index+1] != '/') {
					if data[index] == '\n' || data[index] == '\r' {
						stripped = append(stripped, data[index])
					}
					index++
				}
				if index+1 >= len(data) {
					return nil, fmt.Errorf("unterminated JSONC block comment")
				}
				index++
			default:
				stripped = append(stripped, character)
			}
		default:
			stripped = append(stripped, character)
		}
	}

	return stripTrailingJSONCommas(stripped), nil
}

func stripTrailingJSONCommas(data []byte) []byte {
	stripped := make([]byte, 0, len(data))
	inString := false
	escaped := false

	for index := 0; index < len(data); index++ {
		character := data[index]
		if inString {
			stripped = append(stripped, character)
			if escaped {
				escaped = false
				continue
			}
			switch character {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch character {
		case '"':
			inString = true
			stripped = append(stripped, character)
		case ',':
			next := index + 1
			for next < len(data) && isJSONWhitespace(data[next]) {
				next++
			}
			if next < len(data) && (data[next] == '}' || data[next] == ']') {
				continue
			}
			stripped = append(stripped, character)
		default:
			stripped = append(stripped, character)
		}
	}

	return stripped
}

func isJSONWhitespace(character byte) bool {
	switch character {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func writeJSONFile(path string, value any, force, dryRun bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeConfigFile(path, data, true, dryRun)
}

func mergeContinueConfig(path string, force, dryRun bool) error {
	config, err := readYAMLObject(path)
	if err != nil {
		return err
	}
	if _, ok := config["name"]; !ok {
		config["name"] = "WhoDB"
	}
	if _, ok := config["version"]; !ok {
		config["version"] = "1.0.0"
	}
	if _, ok := config["schema"]; !ok {
		config["schema"] = "v1"
	}
	if err := mergeYAMLNamedList(config, "mcpServers", map[string]any{
		"name":    "whodb",
		"command": "npx",
		"args":    []string{"-y", "@clidey/whodb-cli", "mcp", "serve"},
	}, force); err != nil {
		return err
	}
	appendYAMLStringList(config, "rules", "Use the WhoDB MCP server for database schema exploration, SQL querying, explain plans, data-quality audits, and schema comparisons.")
	return writeYAMLFile(path, config, dryRun)
}

func mergeAiderConfig(path, conventionsPath string, dryRun bool) error {
	config, err := readYAMLObject(path)
	if err != nil {
		return err
	}
	appendYAMLStringList(config, "read", conventionsPath)
	return writeYAMLFile(path, config, dryRun)
}

func readYAMLObject(path string) (map[string]any, error) {
	config := map[string]any{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return config, nil
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s as YAML: %w", path, err)
	}
	return config, nil
}

func writeYAMLFile(path string, value any, dryRun bool) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return writeConfigFile(path, data, true, dryRun)
}

func mergeYAMLNamedList(config map[string]any, key string, item map[string]any, force bool) error {
	name, _ := item["name"].(string)
	items := yamlList(config[key])
	for index, existing := range items {
		existingMap, ok := existing.(map[string]any)
		if !ok {
			continue
		}
		if existingMap["name"] == name {
			if !force {
				return fmt.Errorf("%s already contains %s; use --force to overwrite", key, name)
			}
			items[index] = item
			config[key] = items
			return nil
		}
	}
	config[key] = append(items, item)
	return nil
}

func appendYAMLStringList(config map[string]any, key, value string) {
	items := yamlList(config[key])
	for _, item := range items {
		if item == value {
			config[key] = items
			return
		}
	}
	config[key] = append(items, value)
}

func yamlList(value any) []any {
	switch typed := value.(type) {
	case nil:
		return []any{}
	case []any:
		return typed
	case []string:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, item)
		}
		return result
	default:
		return []any{typed}
	}
}

func selectedSkillNames(name string) ([]string, error) {
	all, err := skillNames()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" || name == "all" {
		return all, nil
	}
	for _, candidate := range all {
		if candidate == name {
			return []string{name}, nil
		}
	}
	return nil, fmt.Errorf("skill %q not found", name)
}

func skillNames() ([]string, error) {
	entries, err := fs.ReadDir(whodbplugin.FS, "skills")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

func agentNames() ([]string, error) {
	entries, err := fs.ReadDir(whodbplugin.FS, "agents")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, strings.TrimSuffix(entry.Name(), ".md"))
		}
	}
	slices.Sort(names)
	return names, nil
}

func installSkill(name, targetDir string, force, dryRun bool) (string, error) {
	sourcePath := filepath.ToSlash(filepath.Join("skills", name, "SKILL.md"))
	data, err := whodbplugin.FS.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}

	destPath := filepath.Join(targetDir, name, "SKILL.md")
	if err := writeFile(destPath, data, force, dryRun); err != nil {
		return "", err
	}
	return destPath, nil
}

func installAgent(name, targetDir string, force, dryRun bool) (string, error) {
	sourcePath := filepath.ToSlash(filepath.Join("agents", name+".md"))
	data, err := whodbplugin.FS.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}

	destPath := filepath.Join(targetDir, name+".md")
	if err := writeFile(destPath, data, force, dryRun); err != nil {
		return "", err
	}
	return destPath, nil
}

func writeFile(path string, data []byte, force, dryRun bool) error {
	if err := validateFileWrite(path, force); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func validateFileWrite(path string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func writeConfigFile(path string, data []byte, force, dryRun bool) error {
	if dryRun {
		return nil
	}
	if err := backupExistingFile(path); err != nil {
		return err
	}
	return writeFile(path, data, force, false)
}

func backupExistingFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(path+".whodb.bak", data, 0o644)
}

func readSkillDescription(name string) (string, error) {
	path := filepath.ToSlash(filepath.Join("skills", name, "SKILL.md"))
	data, err := whodbplugin.FS.ReadFile(path)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "description:")), `"`), nil
		}
	}
	return "", nil
}
