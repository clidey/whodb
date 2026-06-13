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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/clidey/whodb/cli/pkg/identity"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common/config"
	"github.com/clidey/whodb/core/src/common/datadir"
	"github.com/clidey/whodb/core/src/env"
	"github.com/zalando/go-keyring"
)

var (
	globalUseKeyring bool
	keyringAvailable *bool // Cached result of keyring availability check
)

type Connection struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	Username    string            `json:"username"`
	Password    string            `json:"password,omitempty"`
	Database    string            `json:"database"`
	Schema      string            `json:"schema,omitempty"`
	Advanced    map[string]string `json:"advanced,omitempty"`
	IsProfile   bool              `json:"is_profile,omitempty"`
	SSHHost     string            `json:"ssh_host,omitempty"`
	SSHPort     int               `json:"ssh_port,omitempty"`
	SSHUser     string            `json:"ssh_user,omitempty"`
	SSHKeyFile  string            `json:"ssh_key_file,omitempty"`
	SSHPassword string            `json:"ssh_password,omitempty"`
}

type HistoryConfig struct {
	MaxEntries int  `json:"max_entries"`
	Persist    bool `json:"persist"`
}

type DisplayConfig struct {
	Theme    string `json:"theme"`
	PageSize int    `json:"page_size"`
}

type AIConfig struct {
	ConsentGiven bool   `json:"consent_given"`
	LastProvider string `json:"last_provider,omitempty"`
	LastModel    string `json:"last_model,omitempty"`
}

type QueryConfig struct {
	TimeoutSeconds          int `json:"timeout_seconds"`
	PreferredTimeoutSeconds int `json:"preferred_timeout_seconds,omitempty"`
}

// Profile bundles a connection name with display and query preferences
// for quick switching.
type Profile struct {
	Name           string `json:"name"`
	Connection     string `json:"connection"`
	Theme          string `json:"theme,omitempty"`
	PageSize       int    `json:"page_size,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// SavedQuery represents a bookmarked SQL query.
type SavedQuery struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

// PlatformHost stores non-secret hosted WhoDB account and workspace state.
type PlatformHost struct {
	URL                string                 `json:"url"`
	AccountID          string                 `json:"account_id,omitempty"`
	Email              string                 `json:"email,omitempty"`
	DefaultOrgID       string                 `json:"default_org_id,omitempty"`
	DefaultOrgName     string                 `json:"default_org_name,omitempty"`
	DefaultProjectID   string                 `json:"default_project_id,omitempty"`
	DefaultProjectName string                 `json:"default_project_name,omitempty"`
	Manifest           *PlatformManifestCache `json:"manifest,omitempty"`
}

// PlatformManifestCache stores non-secret hosted platform contract metadata.
type PlatformManifestCache struct {
	PlatformVersion         string          `json:"platform_version"`
	ManifestProtocolVersion string          `json:"manifest_protocol_version"`
	FetchedAt               string          `json:"fetched_at"`
	Raw                     json.RawMessage `json:"raw"`
}

// PlatformConfig stores hosted WhoDB CLI configuration.
type PlatformConfig struct {
	DefaultHost string         `json:"default_host,omitempty"`
	Hosts       []PlatformHost `json:"hosts,omitempty"`
}

// WorkspaceEditorBufferState stores the name and SQL text for one editor tab.
type WorkspaceEditorBufferState struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

// WorkspaceEditorState stores the editor tab set and active tab index.
type WorkspaceEditorState struct {
	Buffers   []WorkspaceEditorBufferState `json:"buffers,omitempty"`
	ActiveTab int                          `json:"active_tab,omitempty"`
}

// WorkspaceBrowserState stores lightweight browser selection and filter state.
type WorkspaceBrowserState struct {
	Schema string `json:"schema,omitempty"`
	Table  string `json:"table,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// WorkspaceResultsState stores reloadable table-results context.
type WorkspaceResultsState struct {
	Schema         string                `json:"schema,omitempty"`
	Table          string                `json:"table,omitempty"`
	CurrentPage    int                   `json:"current_page,omitempty"`
	PageSize       int                   `json:"page_size,omitempty"`
	ColumnOffset   int                   `json:"column_offset,omitempty"`
	VisibleColumns []string              `json:"visible_columns,omitempty"`
	Where          *model.WhereCondition `json:"where,omitempty"`
}

// WorkspaceDiffState stores the last schema diff selection inputs.
type WorkspaceDiffState struct {
	FromConnection string `json:"from_connection,omitempty"`
	ToConnection   string `json:"to_connection,omitempty"`
	FromSchema     string `json:"from_schema,omitempty"`
	ToSchema       string `json:"to_schema,omitempty"`
}

// WorkspaceState stores the lightweight interactive CLI session that can be
// restored on the next TUI launch.
type WorkspaceState struct {
	ConnectionName string                `json:"connection_name,omitempty"`
	ProfileName    string                `json:"profile_name,omitempty"`
	View           string                `json:"view,omitempty"`
	Layout         string                `json:"layout,omitempty"`
	FocusedPane    int                   `json:"focused_pane,omitempty"`
	Browser        WorkspaceBrowserState `json:"browser,omitempty"`
	Editor         WorkspaceEditorState  `json:"editor,omitempty"`
	Results        WorkspaceResultsState `json:"results,omitempty"`
	Diff           WorkspaceDiffState    `json:"diff,omitempty"`
	SavedAt        string                `json:"saved_at,omitempty"`
}

// CLISection is the structure stored in the "cli" section of config.json.
type CLISection struct {
	Connections  []Connection    `json:"connections"`
	History      HistoryConfig   `json:"history"`
	Display      DisplayConfig   `json:"display"`
	AI           AIConfig        `json:"ai"`
	Query        QueryConfig     `json:"query"`
	Platform     PlatformConfig  `json:"platform,omitempty"`
	SavedQueries []SavedQuery    `json:"saved_queries,omitempty"`
	Profiles     []Profile       `json:"profiles,omitempty"`
	ReadOnly     bool            `json:"read_only,omitempty"`
	Workspace    *WorkspaceState `json:"workspace,omitempty"`
}

type Config struct {
	CLISection
	useKeyring          bool // Not persisted
	keyringWarningShown bool // Track if warning was shown
}

func DefaultConfig() *Config {
	return &Config{
		CLISection: CLISection{
			Connections: []Connection{},
			History: HistoryConfig{
				MaxEntries: 1000,
				Persist:    true,
			},
			Display: DisplayConfig{
				Theme:    "dark",
				PageSize: 50,
			},
			AI: AIConfig{
				ConsentGiven: false,
			},
			Query: QueryConfig{
				TimeoutSeconds: 30,
			},
		},
		useKeyring: false,
	}
}

var (
	configDir     string
	configDirOnce sync.Once
	configDirErr  error
)

// ResetPathsForTesting resets cached CLI config path state after tests change
// environment variables such as HOME. It is intended for tests only.
func ResetPathsForTesting() {
	configDir = ""
	configDirErr = nil
	configDirOnce = sync.Once{}
	config.ResetConfigPath()
}

func getConfigOptions() datadir.Options {
	return datadir.Options{
		AppName:           "whodb",
		EnterpriseEdition: env.IsEnterpriseEdition,
		Development:       env.IsDevelopment,
	}
}

func GetConfigDir() (string, error) {
	configDirOnce.Do(func() {
		configDir, configDirErr = datadir.Get(getConfigOptions())
		if configDirErr != nil {
			return
		}

	})

	return configDir, configDirErr
}

func GetConfigPath() (string, error) {
	return config.GetPath(getConfigOptions())
}

// isKeyringAvailable tests if the OS keyring is accessible
func isKeyringAvailable() bool {
	if keyringAvailable != nil {
		return *keyringAvailable
	}

	_, err := keyring.Get(identity.Current().KeyringService, "whodb-cli-test-availability")
	available := err == nil || errors.Is(err, keyring.ErrNotFound)
	keyringAvailable = &available

	return available
}

func LoadConfig() (*Config, error) {
	return loadConfig(true, true)
}

// LoadConfigWithoutSecrets loads CLI configuration without resolving keyring
// secrets or printing keyring warnings. It is intended for metadata-only paths
// such as shell completion and connection discovery.
func LoadConfigWithoutSecrets() (*Config, error) {
	return loadConfig(false, false)
}

func loadConfig(includeSecrets, showWarnings bool) (*Config, error) {
	if _, err := GetConfigDir(); err != nil {
		return nil, err
	}

	useKeyring := isKeyringAvailable()
	globalUseKeyring = useKeyring

	cfg := DefaultConfig()
	cfg.useKeyring = useKeyring

	opts := getConfigOptions()
	if err := config.ReadSection(config.SectionCLI, &cfg.CLISection, opts); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	// Load passwords only when the caller needs secrets.
	if includeSecrets && cfg.useKeyring {
		for i := range cfg.Connections {
			if cfg.Connections[i].Name != "" {
				password, err := keyring.Get(identity.Current().KeyringService, "connection:"+cfg.Connections[i].Name)
				if err == nil {
					cfg.Connections[i].Password = password
				}
			}
		}
	}

	if showWarnings {
		cfg.showKeyringWarning()
	}
	return cfg, nil
}

// UsesKeyring returns whether this config instance can store secrets in the OS keyring.
func (c *Config) UsesKeyring() bool {
	return c.useKeyring
}

func (c *Config) showKeyringWarning() {
	if !c.useKeyring && !c.keyringWarningShown {
		fmt.Fprintf(os.Stderr, "\n⚠️  WARNING: OS keyring not available.\n")
		fmt.Fprintf(os.Stderr, "   Passwords will be stored in plaintext in config file.\n")
		fmt.Fprintf(os.Stderr, "   File permissions: 0600 (user read/write only)\n\n")
		c.keyringWarningShown = true
	}
}

func (c *Config) Save() error {
	globalUseKeyring = c.useKeyring

	if c.useKeyring {
		for _, conn := range c.Connections {
			if conn.Name != "" && conn.Password != "" {
				err := keyring.Set(identity.Current().KeyringService, "connection:"+conn.Name, conn.Password)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Could not save password to keyring for %s: %v\n", conn.Name, err)
					fmt.Fprintf(os.Stderr, "Password will be saved in config file.\n")
					globalUseKeyring = false
				}
			}
		}
	}

	// Prepare section for saving (strip passwords if using keyring)
	section := c.CLISection
	if globalUseKeyring {
		// Create a copy with passwords stripped
		section.Connections = make([]Connection, len(c.Connections))
		for i, conn := range c.Connections {
			section.Connections[i] = conn
			section.Connections[i].Password = ""
		}
	}

	opts := getConfigOptions()
	if err := config.WriteSection(config.SectionCLI, section, opts); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}

	return nil
}

// UpsertPlatformHost stores or replaces a hosted WhoDB account entry.
func (c *Config) UpsertPlatformHost(host PlatformHost) {
	for i, existing := range c.Platform.Hosts {
		if existing.URL == host.URL {
			c.Platform.Hosts[i] = host
			return
		}
	}
	c.Platform.Hosts = append(c.Platform.Hosts, host)
}

// SetOnlyPlatformHost replaces all hosted WhoDB account entries with one host.
func (c *Config) SetOnlyPlatformHost(host PlatformHost) {
	c.Platform.Hosts = []PlatformHost{host}
	c.Platform.DefaultHost = host.URL
}

// GetPlatformHost returns the hosted WhoDB account entry for the URL.
func (c *Config) GetPlatformHost(url string) (*PlatformHost, bool) {
	for i := range c.Platform.Hosts {
		if c.Platform.Hosts[i].URL == url {
			return &c.Platform.Hosts[i], true
		}
	}
	return nil, false
}

// RemovePlatformHost deletes the hosted WhoDB account entry for the URL.
func (c *Config) RemovePlatformHost(url string) bool {
	for i, host := range c.Platform.Hosts {
		if host.URL == url {
			c.Platform.Hosts = append(c.Platform.Hosts[:i], c.Platform.Hosts[i+1:]...)
			if c.Platform.DefaultHost == url {
				c.Platform.DefaultHost = ""
				if len(c.Platform.Hosts) > 0 {
					c.Platform.DefaultHost = c.Platform.Hosts[0].URL
				}
			}
			return true
		}
	}
	return false
}

// SetDefaultPlatformHost updates the hosted WhoDB host used by default.
func (c *Config) SetDefaultPlatformHost(url string) {
	c.Platform.DefaultHost = url
}

// SavePlatformRefreshToken stores a hosted WhoDB refresh token in the OS keyring.
func (c *Config) SavePlatformRefreshToken(hostURL, accountID, refreshToken string) error {
	if !c.useKeyring {
		return errors.New("OS keyring is required for hosted WhoDB refresh tokens")
	}
	return keyring.Set(identity.Current().KeyringService, platformRefreshTokenKey(hostURL, accountID), refreshToken)
}

// GetPlatformRefreshToken loads a hosted WhoDB refresh token from the OS keyring.
func (c *Config) GetPlatformRefreshToken(hostURL, accountID string) (string, error) {
	if !c.useKeyring {
		return "", errors.New("OS keyring is required for hosted WhoDB refresh tokens")
	}
	return keyring.Get(identity.Current().KeyringService, platformRefreshTokenKey(hostURL, accountID))
}

// DeletePlatformRefreshToken removes a hosted WhoDB refresh token from the OS keyring.
func (c *Config) DeletePlatformRefreshToken(hostURL, accountID string) error {
	if !c.useKeyring {
		return nil
	}
	err := keyring.Delete(identity.Current().KeyringService, platformRefreshTokenKey(hostURL, accountID))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// IsKeyringNotFound reports whether err means the requested keyring item is missing.
func IsKeyringNotFound(err error) bool {
	return errors.Is(err, keyring.ErrNotFound)
}

func platformRefreshTokenKey(hostURL, accountID string) string {
	return "platform:" + hostURL + ":" + accountID + ":refresh_token"
}

func (c *Config) AddConnection(conn Connection) {
	for i, existing := range c.Connections {
		if existing.Name == conn.Name {
			c.Connections[i] = conn
			return
		}
	}
	c.Connections = append(c.Connections, conn)
}

func (c *Config) RemoveConnection(name string) bool {
	for i, conn := range c.Connections {
		if conn.Name == name {
			c.Connections = append(c.Connections[:i], c.Connections[i+1:]...)

			if c.useKeyring {
				_ = keyring.Delete(identity.Current().KeyringService, "connection:"+name)
			}

			return true
		}
	}
	return false
}

func (c *Config) GetConnection(name string) (*Connection, error) {
	for _, conn := range c.Connections {
		if conn.Name == name {
			return &conn, nil
		}
	}
	return nil, fmt.Errorf("connection '%s' not found", name)
}

func (c *Config) SetAIConsent(consent bool) {
	c.AI.ConsentGiven = consent
}

func (c *Config) GetAIConsent() bool {
	return c.AI.ConsentGiven
}

func (c *Config) GetQueryTimeout() time.Duration {
	if c.Query.TimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.Query.TimeoutSeconds) * time.Second
}

func (c *Config) GetLastAIProvider() string {
	return c.AI.LastProvider
}

func (c *Config) SetLastAIProvider(provider string) {
	c.AI.LastProvider = provider
}

func (c *Config) GetLastAIModel() string {
	return c.AI.LastModel
}

func (c *Config) SetLastAIModel(model string) {
	c.AI.LastModel = model
}

func (c *Config) GetPreferredTimeout() int {
	return c.Query.PreferredTimeoutSeconds
}

func (c *Config) SetPreferredTimeout(seconds int) {
	c.Query.PreferredTimeoutSeconds = seconds
}

// GetPageSize returns the configured page size, defaulting to 50.
func (c *Config) GetPageSize() int {
	if c.Display.PageSize <= 0 {
		return 50
	}
	return c.Display.PageSize
}

// SetPageSize updates the configured page size.
func (c *Config) SetPageSize(size int) {
	c.Display.PageSize = size
}

// GetThemeName returns the configured theme name.
// Returns "default" if unset or set to the legacy "dark" value.
func (c *Config) GetThemeName() string {
	name := c.Display.Theme
	if name == "" || name == "dark" {
		return "default"
	}
	return name
}

// SetThemeName updates the configured theme name and persists to disk.
func (c *Config) SetThemeName(name string) {
	c.Display.Theme = name
}

// AddSavedQuery adds a bookmarked query. If a query with the same name
// already exists it is replaced.
func (c *Config) AddSavedQuery(name, query string) {
	for i, sq := range c.SavedQueries {
		if sq.Name == name {
			c.SavedQueries[i].Query = query
			return
		}
	}
	c.SavedQueries = append(c.SavedQueries, SavedQuery{Name: name, Query: query})
}

// DeleteSavedQuery removes a bookmarked query by name.
// Returns true if a query was found and removed.
func (c *Config) DeleteSavedQuery(name string) bool {
	for i, sq := range c.SavedQueries {
		if sq.Name == name {
			c.SavedQueries = append(c.SavedQueries[:i], c.SavedQueries[i+1:]...)
			return true
		}
	}
	return false
}

// GetSavedQueries returns all bookmarked queries.
func (c *Config) GetSavedQueries() []SavedQuery {
	return c.SavedQueries
}

// GetReadOnly returns whether read-only mode is enabled.
func (c *Config) GetReadOnly() bool {
	return c.ReadOnly
}

// SetReadOnly enables or disables read-only mode.
func (c *Config) SetReadOnly(readOnly bool) {
	c.ReadOnly = readOnly
}

// AddProfile adds a profile. If a profile with the same name already
// exists it is replaced.
func (c *Config) AddProfile(profile Profile) {
	for i, p := range c.Profiles {
		if p.Name == profile.Name {
			c.Profiles[i] = profile
			return
		}
	}
	c.Profiles = append(c.Profiles, profile)
}

// DeleteProfile removes a profile by name.
// Returns true if a profile was found and removed.
func (c *Config) DeleteProfile(name string) bool {
	for i, p := range c.Profiles {
		if p.Name == name {
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			return true
		}
	}
	return false
}

// GetProfile returns the profile with the given name, or nil if not found.
func (c *Config) GetProfile(name string) *Profile {
	for _, p := range c.Profiles {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

// GetProfiles returns all saved profiles.
func (c *Config) GetProfiles() []Profile {
	return c.Profiles
}

// GetWorkspace returns the saved interactive workspace, or nil if none exists.
func (c *Config) GetWorkspace() *WorkspaceState {
	return c.Workspace
}

// SetWorkspace replaces the saved interactive workspace snapshot.
func (c *Config) SetWorkspace(workspace *WorkspaceState) {
	c.Workspace = workspace
}

// ClearWorkspace removes any saved interactive workspace snapshot.
func (c *Config) ClearWorkspace() {
	c.Workspace = nil
}
