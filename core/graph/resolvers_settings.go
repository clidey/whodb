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

package graph

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/analytics"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/mockdata"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins/ssl"
	"github.com/clidey/whodb/core/src/settings"
	"github.com/clidey/whodb/core/src/version"
)

// UpdateSettings is the resolver for the UpdateSettings field.
func (r *mutationResolver) UpdateSettings(ctx context.Context, newSettings model.SettingsConfigInput) (*model.StatusResponse, error) {
	var fields []settings.ISettingsField

	if newSettings.MetricsEnabled != nil {
		metricsEnabled := common.StrPtrToBool(newSettings.MetricsEnabled)
		fields = append(fields, settings.MetricsEnabledField(metricsEnabled))

		analytics.TrackMutation(ctx, "UpdateSettings.metrics", map[string]any{
			"metrics_enabled": metricsEnabled,
		})
	}

	updated := settings.UpdateSettings(fields...)
	return &model.StatusResponse{
		Status: updated,
	}, nil
}

// SettingsConfig is the resolver for the SettingsConfig field.
func (r *queryResolver) SettingsConfig(ctx context.Context) (*model.SettingsConfig, error) {
	currentSettings := settings.Get()
	return &model.SettingsConfig{
		MetricsEnabled:        &currentSettings.MetricsEnabled,
		CloudProvidersEnabled: env.IsAWSProviderEnabled,
		DisableCredentialForm: env.DisableCredentialForm,
		MaxPageSize:           env.MaxPageSize,
	}, nil
}

// MockDataMaxRowCount is the resolver for the MockDataMaxRowCount field.
func (r *queryResolver) MockDataMaxRowCount(ctx context.Context) (int, error) {
	return mockdata.GetMockDataGenerationMaxRowCount(), nil
}

// DatabaseMetadata is the resolver for the DatabaseMetadata field.
func (r *queryResolver) DatabaseMetadata(ctx context.Context) (*model.DatabaseMetadata, error) {
	plugin, _ := GetPluginForContext(ctx)
	if plugin == nil {
		return nil, nil
	}
	metadata := plugin.GetDatabaseMetadata()

	// Return nil if plugin doesn't implement metadata (default GormPlugin behavior)
	if metadata == nil {
		return nil, nil
	}

	// Convert engine.TypeDefinition to model.TypeDefinition
	typeDefinitions := make([]*model.TypeDefinition, 0, len(metadata.TypeDefinitions))
	for _, td := range metadata.TypeDefinitions {
		typeDefinitions = append(typeDefinitions, &model.TypeDefinition{
			ID:               td.ID,
			Label:            td.Label,
			HasLength:        td.HasLength,
			HasPrecision:     td.HasPrecision,
			DefaultLength:    td.DefaultLength,
			DefaultPrecision: td.DefaultPrecision,
			Category:         model.TypeCategory(td.Category),
		})
	}

	// Convert map[string]string to []*model.Record
	aliasMap := make([]*model.Record, 0, len(metadata.AliasMap))
	for key, value := range metadata.AliasMap {
		aliasMap = append(aliasMap, &model.Record{
			Key:   key,
			Value: value,
		})
	}

	return &model.DatabaseMetadata{
		DatabaseType:    string(metadata.DatabaseType),
		TypeDefinitions: typeDefinitions,
		Operators:       metadata.Operators,
		AliasMap:        aliasMap,
	}, nil
}

// SSLStatus is the resolver for the SSLStatus field.
func (r *queryResolver) SSLStatus(ctx context.Context) (*model.SSLStatus, error) {
	plugin, config := GetPluginForContext(ctx)
	if plugin == nil {
		log.Debug("[SSL] SSLStatus resolver: no plugin context")
		return nil, nil
	}

	log.Debugf("[SSL] SSLStatus resolver: querying SSL status for %s", config.Credentials.Type)
	status, err := plugin.GetSSLStatus(config)
	if err != nil {
		log.Warnf("[SSL] SSLStatus resolver: error getting SSL status: %v", err)
		return nil, err
	}

	// Return nil if SSL status is not applicable (e.g., SQLite)
	if status == nil {
		log.Debugf("[SSL] SSLStatus resolver: SSL not applicable for %s", config.Credentials.Type)
		return nil, nil
	}

	log.Infof("[SSL] SSLStatus resolver: %s connection SSL enabled=%t, mode=%s",
		config.Credentials.Type, status.IsEnabled, status.Mode)

	return &model.SSLStatus{
		IsEnabled: status.IsEnabled,
		Mode:      status.Mode,
	}, nil
}

// DatabaseQuerySuggestions is the resolver for the DatabaseQuerySuggestions field.
func (r *queryResolver) DatabaseQuerySuggestions(ctx context.Context, schema string) ([]*model.DatabaseQuerySuggestion, error) {
	plugin, config := GetPluginForContext(ctx)

	log.WithFields(log.Fields{
		"operation": "DatabaseQuerySuggestions",
		"schema":    schema,
	}).Info("Fetching database suggestions")

	// Get storage units (tables) from the schema
	units, err := plugin.GetStorageUnits(config, schema)
	if err != nil {
		log.WithFields(log.Fields{
			"operation": "DatabaseQuerySuggestions",
			"schema":    schema,
		}).WithError(err).Error("Failed to get storage units for suggestions")
		return nil, err
	}

	log.WithFields(log.Fields{
		"operation":   "DatabaseQuerySuggestions",
		"schema":      schema,
		"units_count": len(units),
	}).Info("Retrieved storage units for suggestions")

	suggestions := []*model.DatabaseQuerySuggestion{}

	// Generate suggestions based on actual tables in the database
	// Limit to 3 suggestions
	maxSuggestions := 3
	if len(units) > maxSuggestions {
		units = units[:maxSuggestions]
	}

	for i, unit := range units {
		var description string
		var category string

		tableName := unit.Name

		// TODO: These hardcoded English strings need localization. There is currently no
		// backend localization function available. When one is added, replace these with
		// localized equivalents.
		// Generate natural, conversational queries that someone would actually ask
		switch i % 3 {
		case 0:
			description = fmt.Sprintf("What are the most recent records in %s?", tableName)
			category = "SELECT"
		case 1:
			description = fmt.Sprintf("How many total entries are in %s?", tableName)
			category = "AGGREGATE"
		case 2:
			description = fmt.Sprintf("Show me all the data in %s", tableName)
			category = "SELECT"
		}

		suggestions = append(suggestions, &model.DatabaseQuerySuggestion{
			Description: description,
			Category:    category,
		})
	}

	// If no tables found, return empty array
	if len(suggestions) == 0 {
		log.WithFields(log.Fields{
			"operation": "DatabaseQuerySuggestions",
			"schema":    schema,
		}).Warn("No suggestions generated - no tables found")
		return []*model.DatabaseQuerySuggestion{}, nil
	}

	log.WithFields(log.Fields{
		"operation":         "DatabaseQuerySuggestions",
		"schema":            schema,
		"suggestions_count": len(suggestions),
	}).Info("Successfully generated database suggestions")

	return suggestions, nil
}

// Version is the resolver for the Version field.
func (r *queryResolver) Version(ctx context.Context) (string, error) {
	if env.ApplicationVersion != "" {
		return env.ApplicationVersion, nil
	}
	// Default fallback for development
	return "development", nil
}

// UpdateInfo is the resolver for the UpdateInfo field.
func (r *queryResolver) UpdateInfo(ctx context.Context) (*model.UpdateInfo, error) {
	currentVersion := env.ApplicationVersion
	if currentVersion == "" {
		currentVersion = "development"
	}

	disabled := env.GetDisableUpdateCheck() || os.Getenv("SNAP") != ""

	info := version.CheckForUpdate(currentVersion, disabled)
	return &model.UpdateInfo{
		CurrentVersion:  info.CurrentVersion,
		LatestVersion:   info.LatestVersion,
		UpdateAvailable: info.UpdateAvailable,
		ReleaseURL:      info.ReleaseURL,
	}, nil
}

// Health is the resolver for the Health field.
func (r *queryResolver) Health(ctx context.Context) (*model.HealthStatus, error) {
	status := &model.HealthStatus{
		Server:   "healthy",
		Database: "unavailable",
	}

	// Check if user is authenticated and has credentials
	credentials := auth.GetCredentials(ctx)
	if credentials != nil && credentials.Type != "" {
		config := engine.NewPluginConfig(credentials)
		plugin := src.MainEngine.Choose(engine.DatabaseType(config.Credentials.Type))

		if plugin != nil {
			// Create a context with 5 second timeout (Oracle connections can take 3-8s)
			healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			done := make(chan bool, 1)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("Panic during health check for database: %v", r)
						status.Database = "error"
					}
					done <- true
				}()

				if plugin.IsAvailable(healthCtx, config) {
					status.Database = "healthy"
				} else {
					status.Database = "error"
				}
			}()

			select {
			case <-done:
				// Health check completed
			case <-healthCtx.Done():
				// Timeout - database is not responding
				status.Database = "error"
			}
		}
	}

	return status, nil
}

// Profiles is the resolver for the Profiles field.
func (r *queryResolver) Profiles(ctx context.Context) ([]*model.LoginProfile, error) {
	var profiles []*model.LoginProfile
	for i, profile := range src.GetLoginProfiles() {
		profileName := src.GetLoginProfileId(i, profile)

		// Check if SSL is configured (mode is set and not "disabled")
		sslConfigured := false
		if mode, ok := profile.Advanced[ssl.KeySSLMode]; ok && mode != "" && mode != string(ssl.SSLModeDisabled) {
			sslConfigured = true
		}

		loginProfile := &model.LoginProfile{
			ID:                   profileName,
			Type:                 model.DatabaseType(profile.Type),
			Hostname:             &profile.Hostname,
			Database:             &profile.Database,
			IsEnvironmentDefined: true,
			Source:               profile.Source,
			SSLConfigured:        sslConfigured,
		}
		if len(profile.Alias) > 0 {
			loginProfile.Alias = &profile.Alias
		}
		if len(profile.CustomId) > 0 {
			loginProfile.ID = profile.CustomId
		}
		profiles = append(profiles, loginProfile)
	}
	return profiles, nil
}
