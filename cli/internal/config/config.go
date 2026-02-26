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
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/clidey/whodb/core/src/common/config"
	"github.com/clidey/whodb/core/src/common/datadir"
	"github.com/clidey/whodb/core/src/env"
	"github.com/zalando/go-keyring"
)

const keyringService = "WhoDB-CLI"

var (
	globalUseKeyring bool
	keyringAvailable *bool // Cached result of keyring availability check
)

type Connection struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Host      string            `json:"host"`
	Port      int               `json:"port"`
	Username  string            `json:"username"`
	Password  string            `json:"password,omitempty"`
	Database  string            `json:"database"`
	Schema    string            `json:"schema,omitempty"`
	Advanced  map[string]string `json:"advanced,omitempty"`
	IsProfile bool              `json:"is_profile,omitempty"`
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

// CLISection is the structure stored in the "cli" section of config.json.
type CLISection struct {
	Connections []Connection  `json:"connections"`
	History     HistoryConfig `json:"history"`
	Display     DisplayConfig `json:"display"`
	AI          AIConfig      `json:"ai"`
	Query       QueryConfig   `json:"query"`
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

	_, err := keyring.Get(keyringService, "whodb-cli-test-availability")
	available := err == nil || errors.Is(err, keyring.ErrNotFound)
	keyringAvailable = &available

	return available
}

func LoadConfig() (*Config, error) {
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

	// Load passwords
	if cfg.useKeyring {
		for i := range cfg.Connections {
			if cfg.Connections[i].Name != "" {
				password, err := keyring.Get(keyringService, "connection:"+cfg.Connections[i].Name)
				if err == nil {
					cfg.Connections[i].Password = password
				}
			}
		}
	}

	cfg.showKeyringWarning()
	return cfg, nil
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
				err := keyring.Set(keyringService, "connection:"+conn.Name, conn.Password)
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
				_ = keyring.Delete(keyringService, "connection:"+name)
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
