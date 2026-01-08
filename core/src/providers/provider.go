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

// Package providers defines the ConnectionProvider interface for database connection sources.
//
// A ConnectionProvider is a source of database connections. It abstracts where connection
// credentials come from - whether manually entered by users or auto-discovered from cloud
// platforms like AWS.
//
// The key insight is that a provider is a "credential factory". It discovers available
// databases and produces standard engine.Credentials that existing database plugins understand.
// A MySQL plugin doesn't know or care if credentials came from AWS RDS or manual entry -
// it just receives standard credentials and connects.
//
// Built-in providers:
//   - Manual: Wraps user-entered credentials (the default behavior)
//   - AWS: Discovers RDS, ElastiCache, DocumentDB instances and generates credentials
//
// Example flow:
//
//  1. User configures AWS provider with region and auth method
//  2. AWS provider discovers RDS MySQL instance "prod-db"
//  3. User clicks "prod-db" in UI
//  4. AWS provider builds engine.Credentials with hostname, IAM auth token, etc.
//  5. Existing MySQL plugin receives credentials and connects normally
package providers

import (
	"context"

	"github.com/clidey/whodb/core/src/engine"
)

// ProviderType identifies a connection provider type.
type ProviderType string

const (
	// ProviderTypeManual represents manually-entered connections.
	ProviderTypeManual ProviderType = "manual"

	// ProviderTypeAWS represents AWS-discovered connections.
	ProviderTypeAWS ProviderType = "aws"
)

// ConnectionProvider discovers and builds credentials for database connections.
// Implementations must be safe for concurrent use.
type ConnectionProvider interface {
	// Type returns the provider type (e.g., "manual", "aws").
	Type() ProviderType

	// ID returns the unique identifier for this provider instance.
	// For manual provider, this might be "default".
	// For AWS provider, this identifies the specific AWS account/region combo.
	ID() string

	// Name returns a human-readable name for this provider.
	Name() string

	// DiscoverConnections returns all available connections from this provider.
	// The result should be cached by the caller - implementations may make
	// expensive API calls each time.
	DiscoverConnections(ctx context.Context) ([]DiscoveredConnection, error)

	// TestConnection verifies the provider can access its source.
	// For AWS, this tests AWS API access. For manual, this always succeeds.
	TestConnection(ctx context.Context) error

	// RefreshConnection refreshes credentials for a connection if needed.
	// Some providers (like AWS with IAM auth) need periodic credential refresh.
	// Returns true if credentials were refreshed, false if no refresh needed.
	RefreshConnection(ctx context.Context, connectionID string) (bool, error)

	// Close releases any resources held by the provider.
	Close(ctx context.Context) error
}

// DiscoveredConnection represents a database found by a provider.
type DiscoveredConnection struct {
	// ID uniquely identifies this connection within the provider.
	// Format: providerID/connectionID (e.g., "aws-us-west-2/prod-mysql")
	ID string

	// ProviderType identifies the provider that discovered this connection.
	ProviderType ProviderType

	// ProviderID identifies the specific provider instance.
	ProviderID string

	// Name is the display name for the connection (e.g., "prod-mysql").
	Name string

	// DatabaseType maps to existing engine.DatabaseType constants.
	// This determines which plugin handles the connection.
	DatabaseType engine.DatabaseType

	// Region indicates the geographic location (for cloud resources).
	Region string

	// Status indicates the current state of the resource.
	Status ConnectionStatus

	// Metadata contains provider-specific details.
	// For AWS RDS: engine version, instance class, endpoint, port, etc.
	Metadata map[string]string
}

// ConnectionStatus represents the health/availability of a discovered connection.
type ConnectionStatus string

const (
	// ConnectionStatusAvailable means the connection is ready to use.
	ConnectionStatusAvailable ConnectionStatus = "available"

	// ConnectionStatusStarting means the resource is starting up.
	ConnectionStatusStarting ConnectionStatus = "starting"

	// ConnectionStatusStopped means the resource is stopped but can be started.
	ConnectionStatusStopped ConnectionStatus = "stopped"

	// ConnectionStatusDeleting means the resource is being deleted.
	ConnectionStatusDeleting ConnectionStatus = "deleting"

	// ConnectionStatusFailed means the resource is in a failed state.
	ConnectionStatusFailed ConnectionStatus = "failed"

	// ConnectionStatusUnknown means the status couldn't be determined.
	ConnectionStatusUnknown ConnectionStatus = "unknown"
)

// IsAvailable returns true if the connection status indicates it can be connected to.
func (s ConnectionStatus) IsAvailable() bool {
	return s == ConnectionStatusAvailable
}

// ProviderConfig is the base configuration for all providers.
type ProviderConfig struct {
	// ID uniquely identifies this provider instance.
	ID string

	// Name is a human-readable name for this provider.
	Name string

	// Enabled controls whether this provider is active.
	Enabled bool
}
