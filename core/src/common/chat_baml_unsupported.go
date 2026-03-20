//go:build arm || riscv64

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

// chat_baml_unsupported.go provides stub implementations for armv7 platforms
// where BAML is not supported. AI features will return user-friendly errors
// instead of crashing.

package common

import (
	"context"
	"errors"

	"github.com/clidey/whodb/core/src/engine"
)

// RawExecutePlugin defines the interface for executing raw queries
type RawExecutePlugin interface {
	RawExecute(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error)
}

// IsNoSQLDatabase returns true if the database type should use the database-agnostic
// BAML prompt instead of the SQL-specific prompt.
func IsNoSQLDatabase(dbType string) bool {
	switch engine.DatabaseType(dbType) {
	case engine.DatabaseType_MongoDB,
		engine.DatabaseType_Redis,
		engine.DatabaseType_ElasticSearch:
		return true
	}
	return false
}

// ErrBAMLNotSupported is returned when AI features are used on unsupported platforms
var ErrBAMLNotSupported = errors.New("AI features are not supported on this platform (arm/riscv64). BAML requires amd64 or arm64 architecture")

// ExecuteChatQuery returns an error on unsupported platforms
func ExecuteChatQuery(
	ctx context.Context,
	databaseType string,
	schema string,
	tableDetails string,
	previousConversation string,
	userQuery string,
	config *engine.PluginConfig,
	plugin RawExecutePlugin,
) ([]*engine.ChatMessage, error) {
	return nil, ErrBAMLNotSupported
}

// SetupAIClient returns nil on unsupported platforms
func SetupAIClient(externalModel *engine.ExternalModel) []any {
	return nil
}

// CreateDynamicBAMLClient returns nil on unsupported platforms
func CreateDynamicBAMLClient(externalModel *engine.ExternalModel) any {
	return nil
}

// BAMLConfigResolver resolves BAML provider string + options for a given provider type.
type BAMLConfigResolver func(providerType, apiKey, endpoint, model string) (string, map[string]any, error)

// RegisterBAMLConfigResolver is a no-op on unsupported platforms.
func RegisterBAMLConfigResolver(resolver BAMLConfigResolver) {}
