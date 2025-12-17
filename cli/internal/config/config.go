/*
 * Copyright 2025 Clidey, Inc.
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
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

const keyringService = "WhoDB-CLI"

var (
	globalUseKeyring bool
	keyringAvailable *bool // Cached result of keyring availability check
)

type Connection struct {
	Name     string            `json:"name" yaml:"name"`
	Type     string            `json:"type" yaml:"type"`
	Host     string            `json:"host" yaml:"host"`
	Port     int               `json:"port" yaml:"port"`
	Username string            `json:"username" yaml:"username"`
	Password string            `json:"password,omitempty" yaml:"password,omitempty"`
	Database string            `json:"database" yaml:"database"`
	Advanced map[string]string `json:"advanced,omitempty" yaml:"advanced,omitempty"`
}

// MarshalYAML implements custom YAML marshaling to exclude passwords when using keyring
func (c Connection) MarshalYAML() (interface{}, error) {
	// Create an alias to prevent infinite recursion
	type Alias Connection
	alias := (Alias)(c)

	// If using keyring, clear password from marshaled output
	if globalUseKeyring && c.Name != "" {
		alias.Password = ""
	}

	return alias, nil
}

type HistoryConfig struct {
	MaxEntries int  `json:"max_entries" yaml:"max_entries"`
	Persist    bool `json:"persist" yaml:"persist"`
}

type DisplayConfig struct {
	Theme    string `json:"theme" yaml:"theme"`
	PageSize int    `json:"page_size" yaml:"page_size"`
}

type AIConfig struct {
	ConsentGiven bool `json:"consent_given" yaml:"consent_given"`
}

type Config struct {
	Connections         []Connection  `json:"connections" yaml:"connections"`
	History             HistoryConfig `json:"history" yaml:"history"`
	Display             DisplayConfig `json:"display" yaml:"display"`
	AI                  AIConfig      `json:"ai" yaml:"ai"`
	useKeyring          bool          `json:"-" yaml:"-"` // Not persisted
	keyringWarningShown bool          `json:"-" yaml:"-"` // Track if warning was shown
}

func DefaultConfig() *Config {
	return &Config{
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
		// Don't check here - LoadConfig() will set this
		useKeyring: false,
	}
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}

	configDir := filepath.Join(home, ".whodb-cli")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("error creating config directory: %w", err)
	}

	return configDir, nil
}

func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// isKeyringAvailable tests if the OS keyring is accessible
// The result is cached after the first check
func isKeyringAvailable() bool {
	// Return cached result if available
	if keyringAvailable != nil {
		return *keyringAvailable
	}

	// Try to get a non-existent key to test availability
	_, err := keyring.Get(keyringService, "whodb-cli-test-availability")
	available := err == nil || errors.Is(err, keyring.ErrNotFound)

	// Cache the result
	keyringAvailable = &available

	return available
}

func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	useKeyring := isKeyringAvailable()
	globalUseKeyring = useKeyring // Set global for marshaling

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		cfg.useKeyring = useKeyring
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("error saving default config: %w", err)
		}
		cfg.showKeyringWarning()
		return cfg, nil
	}

	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	cfg.useKeyring = useKeyring

	// Load passwords based on storage method
	if cfg.useKeyring {
		// Load passwords from keyring
		for i := range cfg.Connections {
			if cfg.Connections[i].Name != "" {
				password, err := keyring.Get(keyringService, "connection:"+cfg.Connections[i].Name)
				if err == nil {
					cfg.Connections[i].Password = password
				}
				// Ignore errors - password may not exist yet
			}
		}
	}
	// If not using keyring, passwords are already loaded from YAML

	cfg.showKeyringWarning()
	return &cfg, nil
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
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Set global for marshaling behavior
	globalUseKeyring = c.useKeyring

	if c.useKeyring {
		// Save passwords to keyring
		for _, conn := range c.Connections {
			if conn.Name != "" && conn.Password != "" {
				// Save password to keyring
				err := keyring.Set(keyringService, "connection:"+conn.Name, conn.Password)
				if err != nil {
					// If keyring save fails, warn but continue
					// The custom marshaler will include the password in YAML as fallback
					fmt.Fprintf(os.Stderr, "Warning: Could not save password to keyring for %s: %v\n", conn.Name, err)
					fmt.Fprintf(os.Stderr, "Password will be saved in config file.\n")
					// Temporarily disable keyring for this save to allow password in file
					globalUseKeyring = false
				}
			}
		}
	}

	// Now just save directly - custom marshaler handles password exclusion
	viper.Set("connections", c.Connections)
	viper.Set("history", c.History)
	viper.Set("display", c.Display)
	viper.Set("ai", c.AI)

	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}

	// Enforce strict file permissions for the config file
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("error setting config file permissions: %w", err)
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

			// Also try to remove from keyring (ignore errors)
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

// SetAIConsent updates the AI consent preference
func (c *Config) SetAIConsent(consent bool) {
	c.AI.ConsentGiven = consent
}

// GetAIConsent returns the current AI consent preference
func (c *Config) GetAIConsent() bool {
	return c.AI.ConsentGiven
}
