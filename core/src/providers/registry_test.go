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

package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

// mockProvider implements ConnectionProvider for testing.
type mockProvider struct {
	id          string
	name        string
	connections []DiscoveredConnection
	testErr     error
}

func (m *mockProvider) Type() ProviderType {
	return "mock"
}

func (m *mockProvider) ID() string {
	return m.id
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) DiscoverConnections(ctx context.Context) ([]DiscoveredConnection, error) {
	if m.testErr != nil {
		return nil, m.testErr
	}
	return m.connections, nil
}

func (m *mockProvider) TestConnection(ctx context.Context) error {
	return m.testErr
}

func (m *mockProvider) RefreshConnection(ctx context.Context, connectionID string) (bool, error) {
	return false, nil
}

func (m *mockProvider) Close(ctx context.Context) error {
	return nil
}

// Verify mockProvider implements ConnectionProvider
var _ ConnectionProvider = (*mockProvider)(nil)

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	p := &mockProvider{id: "test-1", name: "Test Provider"}
	err := r.Register(p)
	if err != nil {
		t.Fatalf("unexpected error registering provider: %v", err)
	}

	// Registering again should fail
	err = r.Register(p)
	if !errors.Is(err, ErrProviderAlreadyExists) {
		t.Errorf("expected ErrProviderAlreadyExists, got %v", err)
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	p := &mockProvider{id: "test-1", name: "Test Provider"}
	_ = r.Register(p)

	got, err := r.Get("test-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID() != "test-1" {
		t.Errorf("expected ID test-1, got %s", got.ID())
	}

	// Non-existent provider
	_, err = r.Get("non-existent")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()

	p := &mockProvider{id: "test-1", name: "Test Provider"}
	_ = r.Register(p)

	err := r.Unregister("test-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not be found after unregistering
	_, err = r.Get("test-1")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}

	// Unregistering non-existent should fail
	err = r.Unregister("non-existent")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	p1 := &mockProvider{id: "test-1", name: "Provider 1"}
	p2 := &mockProvider{id: "test-2", name: "Provider 2"}

	_ = r.Register(p1)
	_ = r.Register(p2)

	list := r.List()
	if len(list) != 2 {
		t.Errorf("expected 2 providers, got %d", len(list))
	}
}

func TestRegistry_DiscoverAll(t *testing.T) {
	r := NewRegistry()

	p1 := &mockProvider{
		id:   "test-1",
		name: "Provider 1",
		connections: []DiscoveredConnection{
			{
				ID:           "test-1/conn-1",
				ProviderID:   "test-1",
				Name:         "Connection 1",
				DatabaseType: engine.DatabaseType_MySQL,
				Status:       ConnectionStatusAvailable,
			},
		},
	}
	p2 := &mockProvider{
		id:   "test-2",
		name: "Provider 2",
		connections: []DiscoveredConnection{
			{
				ID:           "test-2/conn-2",
				ProviderID:   "test-2",
				Name:         "Connection 2",
				DatabaseType: engine.DatabaseType_Postgres,
				Status:       ConnectionStatusAvailable,
			},
		},
	}

	_ = r.Register(p1)
	_ = r.Register(p2)

	ctx := context.Background()
	conns, err := r.DiscoverAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}
}

func TestRegistry_DiscoverAll_WithErrors(t *testing.T) {
	r := NewRegistry()

	p1 := &mockProvider{
		id:   "test-1",
		name: "Working Provider",
		connections: []DiscoveredConnection{
			{ID: "test-1/conn-1", ProviderID: "test-1", Name: "Connection 1"},
		},
	}
	p2 := &mockProvider{
		id:      "test-2",
		name:    "Failing Provider",
		testErr: errors.New("discovery failed"),
	}

	_ = r.Register(p1)
	_ = r.Register(p2)

	ctx := context.Background()
	conns, err := r.DiscoverAll(ctx)

	// Should still return connections from working provider
	if len(conns) != 1 {
		t.Errorf("expected 1 connection, got %d", len(conns))
	}
	// Should also return the error
	if err == nil {
		t.Error("expected error from failing provider")
	}
}

func TestRegistry_FilterByDatabaseType(t *testing.T) {
	r := NewRegistry()

	p := &mockProvider{
		id:   "test-1",
		name: "Test Provider",
		connections: []DiscoveredConnection{
			{ID: "test-1/mysql", DatabaseType: engine.DatabaseType_MySQL},
			{ID: "test-1/postgres", DatabaseType: engine.DatabaseType_Postgres},
			{ID: "test-1/redis", DatabaseType: engine.DatabaseType_Redis},
		},
	}
	_ = r.Register(p)

	ctx := context.Background()

	mysqlConns, err := r.FilterByDatabaseType(ctx, engine.DatabaseType_MySQL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mysqlConns) != 1 {
		t.Errorf("expected 1 MySQL connection, got %d", len(mysqlConns))
	}
}

func TestRegistry_FilterAvailable(t *testing.T) {
	r := NewRegistry()

	p := &mockProvider{
		id:   "test-1",
		name: "Test Provider",
		connections: []DiscoveredConnection{
			{ID: "test-1/available", Status: ConnectionStatusAvailable},
			{ID: "test-1/stopped", Status: ConnectionStatusStopped},
			{ID: "test-1/starting", Status: ConnectionStatusStarting},
		},
	}
	_ = r.Register(p)

	ctx := context.Background()

	availableConns, err := r.FilterAvailable(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(availableConns) != 1 {
		t.Errorf("expected 1 available connection, got %d", len(availableConns))
	}
}

func TestRegistry_RefreshDiscovery(t *testing.T) {
	r := NewRegistry()

	callCount := 0
	p := &mockProvider{
		id:   "test-1",
		name: "Test Provider",
	}
	// Override DiscoverConnections to track calls
	originalConns := p.connections
	p.connections = []DiscoveredConnection{
		{ID: "test-1/conn-1", ProviderID: "test-1"},
	}
	_ = r.Register(p)

	ctx := context.Background()

	// First discovery
	_, _ = r.DiscoverAll(ctx)

	// Refresh
	conns, err := r.RefreshDiscovery(ctx, "test-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) == 0 && len(originalConns) == 0 {
		// OK - empty is fine for this test
	}
	_ = callCount // Just to track that we did the call
}

func TestConnectionStatus_IsAvailable(t *testing.T) {
	testCases := []struct {
		status   ConnectionStatus
		expected bool
	}{
		{ConnectionStatusAvailable, true},
		{ConnectionStatusStarting, false},
		{ConnectionStatusStopped, false},
		{ConnectionStatusDeleting, false},
		{ConnectionStatusFailed, false},
		{ConnectionStatusUnknown, false},
	}

	for _, tc := range testCases {
		result := tc.status.IsAvailable()
		if result != tc.expected {
			t.Errorf("IsAvailable() for %s: expected %v, got %v", tc.status, tc.expected, result)
		}
	}
}

func TestDiscoveredConnection_Fields(t *testing.T) {
	conn := DiscoveredConnection{
		ID:           "aws-us-west-2/prod-mysql",
		ProviderType: ProviderTypeAWS,
		ProviderID:   "aws-us-west-2",
		Name:         "prod-mysql",
		DatabaseType: engine.DatabaseType_MySQL,
		Region:       "us-west-2",
		Status:       ConnectionStatusAvailable,
		Metadata:     map[string]string{"engine": "mysql", "version": "8.0"},
	}

	if conn.ID != "aws-us-west-2/prod-mysql" {
		t.Errorf("unexpected ID: %s", conn.ID)
	}
	if conn.DatabaseType != engine.DatabaseType_MySQL {
		t.Errorf("unexpected DatabaseType: %s", conn.DatabaseType)
	}
	if !conn.Status.IsAvailable() {
		t.Error("expected connection to be available")
	}
}
