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

package mcp

import (
	"os"
	"strings"
	"testing"
)

// setupTestEnv creates an isolated test environment
func setupTestEnv(t *testing.T) func() {
	t.Helper()
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	return func() {
		os.Setenv("HOME", origHome)
	}
}

func TestParseConnectionURI_Postgres(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantType string
		wantHost string
		wantPort int
		wantUser string
		wantDB   string
		wantErr  bool
	}{
		{
			name:     "full postgres URI",
			uri:      "postgres://myuser:mypass@localhost:5432/mydb",
			wantType: "Postgres",
			wantHost: "localhost",
			wantPort: 5432,
			wantUser: "myuser",
			wantDB:   "mydb",
			wantErr:  false,
		},
		{
			name:     "postgresql scheme",
			uri:      "postgresql://user:pass@host.example.com:5433/testdb",
			wantType: "Postgres",
			wantHost: "host.example.com",
			wantPort: 5433,
			wantUser: "user",
			wantDB:   "testdb",
			wantErr:  false,
		},
		{
			name:     "default port",
			uri:      "postgres://user:pass@localhost/mydb",
			wantType: "Postgres",
			wantHost: "localhost",
			wantPort: 5432,
			wantUser: "user",
			wantDB:   "mydb",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := ParseConnectionURI(tt.uri, "test")
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConnectionURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if conn.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", conn.Type, tt.wantType)
			}
			if conn.Host != tt.wantHost {
				t.Errorf("Host = %v, want %v", conn.Host, tt.wantHost)
			}
			if conn.Port != tt.wantPort {
				t.Errorf("Port = %v, want %v", conn.Port, tt.wantPort)
			}
			if conn.Username != tt.wantUser {
				t.Errorf("Username = %v, want %v", conn.Username, tt.wantUser)
			}
			if conn.Database != tt.wantDB {
				t.Errorf("Database = %v, want %v", conn.Database, tt.wantDB)
			}
		})
	}
}

func TestParseConnectionURI_MySQL(t *testing.T) {
	conn, err := ParseConnectionURI("mysql://root:secret@db.example.com:3306/production", "mysql-prod")
	if err != nil {
		t.Fatalf("ParseConnectionURI() error = %v", err)
	}

	if conn.Type != "MySQL" {
		t.Errorf("Type = %v, want MySQL", conn.Type)
	}
	if conn.Host != "db.example.com" {
		t.Errorf("Host = %v, want db.example.com", conn.Host)
	}
	if conn.Port != 3306 {
		t.Errorf("Port = %v, want 3306", conn.Port)
	}
	if conn.Name != "mysql-prod" {
		t.Errorf("Name = %v, want mysql-prod", conn.Name)
	}
}

func TestParseConnectionURI_OtherDatabases(t *testing.T) {
	tests := []struct {
		scheme      string
		wantType    string
		defaultPort int
	}{
		{"mariadb", "MariaDB", 3306},
		{"mongodb", "MongoDB", 27017},
		{"redis", "Redis", 6379},
		{"elasticsearch", "ElasticSearch", 9200},
		{"clickhouse", "ClickHouse", 9000},
	}

	for _, tt := range tests {
		t.Run(tt.scheme, func(t *testing.T) {
			uri := tt.scheme + "://user:pass@host/db"
			conn, err := ParseConnectionURI(uri, "test")
			if err != nil {
				t.Fatalf("ParseConnectionURI() error = %v", err)
			}
			if conn.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", conn.Type, tt.wantType)
			}
			if conn.Port != tt.defaultPort {
				t.Errorf("Port = %v, want %v", conn.Port, tt.defaultPort)
			}
		})
	}
}

func TestParseConnectionURI_WithSchema(t *testing.T) {
	conn, err := ParseConnectionURI("postgres://user:pass@localhost/mydb?schema=public", "test")
	if err != nil {
		t.Fatalf("ParseConnectionURI() error = %v", err)
	}
	if conn.Schema != "public" {
		t.Errorf("Schema = %v, want public", conn.Schema)
	}
}

func TestParseConnectionURI_UnsupportedScheme(t *testing.T) {
	_, err := ParseConnectionURI("unsupported://user:pass@localhost/db", "test")
	if err == nil {
		t.Error("Expected error for unsupported scheme")
	}
}

func TestParseConnectionURI_InvalidURI(t *testing.T) {
	_, err := ParseConnectionURI("not a valid uri", "test")
	if err == nil {
		t.Error("Expected error for invalid URI")
	}
}

func TestResolveConnection_FromEnvVar(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Set environment variable
	os.Setenv("WHODB_PROD_URI", "postgres://user:pass@localhost:5432/proddb")
	defer os.Unsetenv("WHODB_PROD_URI")

	conn, err := ResolveConnection("prod")
	if err != nil {
		t.Fatalf("ResolveConnection() error = %v", err)
	}

	if conn.Type != "Postgres" {
		t.Errorf("Type = %v, want Postgres", conn.Type)
	}
	if conn.Database != "proddb" {
		t.Errorf("Database = %v, want proddb", conn.Database)
	}
}

func TestResolveConnection_EnvVarWithDashes(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Set environment variable (dashes become underscores)
	os.Setenv("WHODB_MY_DB_URI", "mysql://user:pass@localhost:3306/mydb")
	defer os.Unsetenv("WHODB_MY_DB_URI")

	conn, err := ResolveConnection("my-db")
	if err != nil {
		t.Fatalf("ResolveConnection() error = %v", err)
	}

	if conn.Type != "MySQL" {
		t.Errorf("Type = %v, want MySQL", conn.Type)
	}
}

func TestResolveConnection_NotFound(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_, err := ResolveConnection("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent connection")
	}
}

func TestListAvailableConnections_FromEnvVars(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Set some environment variables
	os.Setenv("WHODB_DEV_URI", "postgres://user:pass@localhost/dev")
	os.Setenv("WHODB_STAGING_URI", "mysql://user:pass@localhost/staging")
	defer func() {
		os.Unsetenv("WHODB_DEV_URI")
		os.Unsetenv("WHODB_STAGING_URI")
	}()

	conns, err := ListAvailableConnections()
	if err != nil {
		t.Fatalf("ListAvailableConnections() error = %v", err)
	}

	// Should include both env var connections
	connMap := make(map[string]bool)
	for _, c := range conns {
		connMap[c] = true
	}

	if !connMap["dev"] {
		t.Error("Expected 'dev' connection from env var")
	}
	if !connMap["staging"] {
		t.Error("Expected 'staging' connection from env var")
	}
}

func TestResolveConnectionOrDefault_SingleConnection(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Set exactly one connection via env var
	os.Setenv("WHODB_MYDB_URI", "postgres://user:pass@localhost/mydb")
	defer os.Unsetenv("WHODB_MYDB_URI")

	// Should resolve without specifying name
	conn, err := ResolveConnectionOrDefault("")
	if err != nil {
		t.Fatalf("ResolveConnectionOrDefault() error = %v", err)
	}
	if conn.Database != "mydb" {
		t.Errorf("Database = %v, want mydb", conn.Database)
	}
}

func TestResolveConnectionOrDefault_MultipleConnections(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Set multiple connections
	os.Setenv("WHODB_DB1_URI", "postgres://user:pass@localhost/db1")
	os.Setenv("WHODB_DB2_URI", "mysql://user:pass@localhost/db2")
	defer func() {
		os.Unsetenv("WHODB_DB1_URI")
		os.Unsetenv("WHODB_DB2_URI")
	}()

	// Should error when name is empty
	_, err := ResolveConnectionOrDefault("")
	if err == nil {
		t.Error("Expected error for multiple connections without name")
	}
	if !strings.Contains(err.Error(), "multiple connections") {
		t.Errorf("Expected 'multiple connections' in error, got: %v", err)
	}
}

func TestResolveConnectionOrDefault_NoConnections(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// No connections configured
	_, err := ResolveConnectionOrDefault("")
	if err == nil {
		t.Error("Expected error for no connections")
	}
	if !strings.Contains(err.Error(), "no database connections") {
		t.Errorf("Expected 'no database connections' in error, got: %v", err)
	}
}

func TestResolveConnectionOrDefault_WithExplicitName(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Set multiple connections
	os.Setenv("WHODB_DB1_URI", "postgres://user:pass@localhost/db1")
	os.Setenv("WHODB_DB2_URI", "mysql://user:pass@localhost/db2")
	defer func() {
		os.Unsetenv("WHODB_DB1_URI")
		os.Unsetenv("WHODB_DB2_URI")
	}()

	// Should resolve when name is explicit
	conn, err := ResolveConnectionOrDefault("db2")
	if err != nil {
		t.Fatalf("ResolveConnectionOrDefault() error = %v", err)
	}
	if conn.Type != "MySQL" {
		t.Errorf("Type = %v, want MySQL", conn.Type)
	}
}

func TestDefaultPort(t *testing.T) {
	tests := []struct {
		dbType   string
		wantPort int
	}{
		{"Postgres", 5432},
		{"MySQL", 3306},
		{"MariaDB", 3306},
		{"MongoDB", 27017},
		{"Redis", 6379},
		{"ElasticSearch", 9200},
		{"ClickHouse", 9000},
		{"Unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			if got := defaultPort(tt.dbType); got != tt.wantPort {
				t.Errorf("defaultPort(%q) = %v, want %v", tt.dbType, got, tt.wantPort)
			}
		})
	}
}
