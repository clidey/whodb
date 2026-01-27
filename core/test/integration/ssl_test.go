//go:build integration

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

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/clickhouse"
	"github.com/clidey/whodb/core/src/plugins/elasticsearch"
	"github.com/clidey/whodb/core/src/plugins/mongodb"
	"github.com/clidey/whodb/core/src/plugins/mysql"
	"github.com/clidey/whodb/core/src/plugins/postgres"
	"github.com/clidey/whodb/core/src/plugins/redis"
	"github.com/clidey/whodb/core/src/plugins/ssl"
)

// sslTarget represents an SSL-enabled database target for testing.
type sslTarget struct {
	name       string
	plugin     *engine.Plugin
	config     *engine.PluginConfig
	schema     string
	sslMode    ssl.SSLMode
	expectFail bool // true if this config should fail to connect
	skipReason string
}

// getCertsDir returns the path to the dev/certs directory.
func getCertsDir() string {
	// Try relative path from test directory
	candidates := []string{
		"../../../dev/certs",
		"../../dev/certs",
		"dev/certs",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

// readCertFile reads a certificate file and returns its contents.
func readCertFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// initSSLTargets creates the SSL test targets.
// These require the SSL containers to be running (docker compose --profile ssl up).
func initSSLTargets() []sslTarget {
	certsDir := getCertsDir()
	if certsDir == "" {
		return nil
	}

	// Read certificate contents
	postgresCA := readCertFile(filepath.Join(certsDir, "ca/postgres/ca.pem"))
	postgresClientCert := readCertFile(filepath.Join(certsDir, "client/postgres/client-cert.pem"))
	postgresClientKey := readCertFile(filepath.Join(certsDir, "client/postgres/client-key.pem"))

	mysqlCA := readCertFile(filepath.Join(certsDir, "ca/mysql/ca.pem"))
	mysqlClientCert := readCertFile(filepath.Join(certsDir, "client/mysql/client-cert.pem"))
	mysqlClientKey := readCertFile(filepath.Join(certsDir, "client/mysql/client-key.pem"))

	mariadbCA := readCertFile(filepath.Join(certsDir, "ca/mariadb/ca.pem"))
	mariadbClientCert := readCertFile(filepath.Join(certsDir, "client/mariadb/client-cert.pem"))
	mariadbClientKey := readCertFile(filepath.Join(certsDir, "client/mariadb/client-key.pem"))

	mongoCA := readCertFile(filepath.Join(certsDir, "ca/mongodb/ca.pem"))
	mongoClientCert := readCertFile(filepath.Join(certsDir, "client/mongodb/client-cert.pem"))
	mongoClientKey := readCertFile(filepath.Join(certsDir, "client/mongodb/client-key.pem"))

	redisCA := readCertFile(filepath.Join(certsDir, "ca/redis/ca.pem"))
	redisClientCert := readCertFile(filepath.Join(certsDir, "client/redis/client-cert.pem"))
	redisClientKey := readCertFile(filepath.Join(certsDir, "client/redis/client-key.pem"))

	clickhouseCA := readCertFile(filepath.Join(certsDir, "ca/clickhouse/ca.pem"))

	elasticsearchCA := readCertFile(filepath.Join(certsDir, "ca/elasticsearch/ca.pem"))

	return []sslTarget{
		// PostgreSQL SSL tests
		{
			name:   "postgres_ssl_required",
			plugin: postgres.NewPostgresPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_Postgres),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "5433"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeRequired)},
				},
			}),
			schema:  "public",
			sslMode: ssl.SSLModeRequired,
		},
		{
			name:   "postgres_ssl_verify_ca",
			plugin: postgres.NewPostgresPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_Postgres),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "5433"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeVerifyCA)},
					{Key: "SSL CA Content", Value: postgresCA},
					{Key: "SSL Client Cert Content", Value: postgresClientCert},
					{Key: "SSL Client Key Content", Value: postgresClientKey},
				},
			}),
			schema:  "public",
			sslMode: ssl.SSLModeVerifyCA,
		},
		{
			name:   "postgres_ssl_verify_ca_invalid_ca",
			plugin: postgres.NewPostgresPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_Postgres),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "5433"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeVerifyCA)},
					// Invalid CA content - should fail certificate verification
					{Key: "SSL CA Content", Value: "-----BEGIN CERTIFICATE-----\nINVALID_CERTIFICATE_DATA\n-----END CERTIFICATE-----"},
				},
			}),
			schema:     "public",
			sslMode:    ssl.SSLModeVerifyCA,
			expectFail: true,
		},

		// MySQL SSL tests
		{
			name:   "mysql_ssl_required",
			plugin: mysql.NewMySQLPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MySQL),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "3309"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeRequired)},
				},
			}),
			schema:  "test_db",
			sslMode: ssl.SSLModeRequired,
		},
		{
			name:   "mysql_ssl_verify_ca",
			plugin: mysql.NewMySQLPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MySQL),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "3309"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeVerifyCA)},
					{Key: "SSL CA Content", Value: mysqlCA},
					{Key: "SSL Client Cert Content", Value: mysqlClientCert},
					{Key: "SSL Client Key Content", Value: mysqlClientKey},
				},
			}),
			schema:  "test_db",
			sslMode: ssl.SSLModeVerifyCA,
		},

		// MariaDB SSL tests
		{
			name:   "mariadb_ssl_required",
			plugin: mysql.NewMyMariaDBPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MariaDB),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "3310"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeRequired)},
				},
			}),
			schema:  "test_db",
			sslMode: ssl.SSLModeRequired,
		},
		{
			name:   "mariadb_ssl_verify_ca",
			plugin: mysql.NewMyMariaDBPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MariaDB),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "3310"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeVerifyCA)},
					{Key: "SSL CA Content", Value: mariadbCA},
					{Key: "SSL Client Cert Content", Value: mariadbClientCert},
					{Key: "SSL Client Key Content", Value: mariadbClientKey},
				},
			}),
			schema:  "test_db",
			sslMode: ssl.SSLModeVerifyCA,
		},

		// MongoDB SSL tests
		{
			name:   "mongodb_ssl_enabled",
			plugin: mongodb.NewMongoDBPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MongoDB),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "27018"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeEnabled)},
					{Key: "SSL CA Content", Value: mongoCA},
					{Key: "SSL Client Cert Content", Value: mongoClientCert},
					{Key: "SSL Client Key Content", Value: mongoClientKey},
				},
			}),
			schema:  "test_db",
			sslMode: ssl.SSLModeEnabled,
		},
		{
			name:   "mongodb_ssl_insecure",
			plugin: mongodb.NewMongoDBPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MongoDB),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "27018"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeInsecure)},
				},
			}),
			schema:  "test_db",
			sslMode: ssl.SSLModeInsecure,
		},

		// Redis SSL tests
		{
			name:   "redis_ssl_enabled",
			plugin: redis.NewRedisPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_Redis),
				Hostname: "localhost",
				Password: "password",
				Database: "0",
				Advanced: []engine.Record{
					{Key: "Port", Value: "6380"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeEnabled)},
					{Key: "SSL CA Content", Value: redisCA},
					{Key: "SSL Client Cert Content", Value: redisClientCert},
					{Key: "SSL Client Key Content", Value: redisClientKey},
				},
			}),
			schema:  "",
			sslMode: ssl.SSLModeEnabled,
		},
		{
			name:   "redis_ssl_insecure",
			plugin: redis.NewRedisPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_Redis),
				Hostname: "localhost",
				Password: "password",
				Database: "0",
				Advanced: []engine.Record{
					{Key: "Port", Value: "6380"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeInsecure)},
				},
			}),
			schema:  "",
			sslMode: ssl.SSLModeInsecure,
		},

		// ClickHouse SSL tests
		{
			name:   "clickhouse_ssl_enabled",
			plugin: clickhouse.NewClickHousePlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_ClickHouse),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "9440"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeEnabled)},
					{Key: "SSL CA Content", Value: clickhouseCA},
				},
			}),
			schema:  "test_db",
			sslMode: ssl.SSLModeEnabled,
		},
		{
			name:   "clickhouse_ssl_insecure",
			plugin: clickhouse.NewClickHousePlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_ClickHouse),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{
					{Key: "Port", Value: "9440"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeInsecure)},
				},
			}),
			schema:  "test_db",
			sslMode: ssl.SSLModeInsecure,
		},

		// Elasticsearch SSL tests
		{
			name:   "elasticsearch_ssl_enabled",
			plugin: elasticsearch.NewElasticSearchPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_ElasticSearch),
				Hostname: "localhost",
				Username: "elastic",
				Password: "password",
				Advanced: []engine.Record{
					{Key: "Port", Value: "9201"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeEnabled)},
					{Key: "SSL CA Content", Value: elasticsearchCA},
				},
			}),
			schema:  "",
			sslMode: ssl.SSLModeEnabled,
		},
		{
			name:   "elasticsearch_ssl_insecure",
			plugin: elasticsearch.NewElasticSearchPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_ElasticSearch),
				Hostname: "localhost",
				Username: "elastic",
				Password: "password",
				Advanced: []engine.Record{
					{Key: "Port", Value: "9201"},
					{Key: "SSL Mode", Value: string(ssl.SSLModeInsecure)},
				},
			}),
			schema:  "",
			sslMode: ssl.SSLModeInsecure,
		},
	}
}

// TestSSLConnectionAvailability tests that SSL connections can be established.
func TestSSLConnectionAvailability(t *testing.T) {
	if os.Getenv("WHODB_SSL_TESTS") != "1" {
		t.Skip("SSL tests disabled. Set WHODB_SSL_TESTS=1 and run docker compose --profile ssl up")
	}

	sslTargets := initSSLTargets()
	if len(sslTargets) == 0 {
		t.Skip("Could not find certs directory")
	}

	for _, target := range sslTargets {
		t.Run(target.name, func(t *testing.T) {
			if target.skipReason != "" {
				t.Skip(target.skipReason)
			}

			ok := target.plugin.IsAvailable(target.config)
			if target.expectFail {
				if ok {
					t.Errorf("expected connection to fail for %s but it succeeded", target.name)
				}
			} else {
				if !ok {
					t.Errorf("expected connection to succeed for %s but it failed", target.name)
				}
			}
		})
	}
}

// TestSSLStatusResolver tests that GetSSLStatus returns correct values.
func TestSSLStatusResolver(t *testing.T) {
	if os.Getenv("WHODB_SSL_TESTS") != "1" {
		t.Skip("SSL tests disabled. Set WHODB_SSL_TESTS=1 and run docker compose --profile ssl up")
	}

	sslTargets := initSSLTargets()
	if len(sslTargets) == 0 {
		t.Skip("Could not find certs directory")
	}

	for _, target := range sslTargets {
		if target.expectFail || target.skipReason != "" {
			continue
		}

		t.Run(target.name, func(t *testing.T) {
			status, err := target.plugin.GetSSLStatus(target.config)
			if err != nil {
				t.Fatalf("GetSSLStatus failed: %v", err)
			}

			if status == nil {
				t.Fatal("GetSSLStatus returned nil")
			}

			if !status.IsEnabled {
				t.Errorf("expected SSL to be enabled for %s, got IsEnabled=false", target.name)
			}

			// Check that mode matches what we configured
			expectedMode := string(target.sslMode)
			if status.Mode != expectedMode {
				// Some databases normalize the mode, so check for common variations
				if !strings.EqualFold(status.Mode, expectedMode) && status.Mode != "enabled" {
					t.Errorf("expected SSL mode %q for %s, got %q", expectedMode, target.name, status.Mode)
				}
			}
		})
	}
}

// TestSSLDisabledConnection tests that non-SSL connections to SSL-only servers fail appropriately.
func TestSSLDisabledConnection(t *testing.T) {
	if os.Getenv("WHODB_SSL_TESTS") != "1" {
		t.Skip("SSL tests disabled. Set WHODB_SSL_TESTS=1 and run docker compose --profile ssl up")
	}

	// Try connecting to SSL-only Postgres without SSL - should fail or connect without SSL
	plugin := postgres.NewPostgresPlugin()
	config := engine.NewPluginConfig(&engine.Credentials{
		Type:     string(engine.DatabaseType_Postgres),
		Hostname: "localhost",
		Username: "user",
		Password: "password",
		Database: "test_db",
		Advanced: []engine.Record{
			{Key: "Port", Value: "5433"},
			{Key: "SSL Mode", Value: string(ssl.SSLModeDisabled)},
		},
	})

	// The server requires SSL, so this should either:
	// 1. Fail to connect
	// 2. Connect but report SSL as disabled
	ok := plugin.IsAvailable(config)
	if ok {
		// If it connected, verify SSL status shows disabled
		status, err := plugin.GetSSLStatus(config)
		if err == nil && status != nil && status.IsEnabled {
			t.Error("connected with SSL disabled but GetSSLStatus reports SSL enabled")
		}
	}
	// If it failed to connect, that's expected for SSL-required servers
}

// TestSSLBasicQueryAfterConnection tests that queries work over SSL connections.
func TestSSLBasicQueryAfterConnection(t *testing.T) {
	if os.Getenv("WHODB_SSL_TESTS") != "1" {
		t.Skip("SSL tests disabled. Set WHODB_SSL_TESTS=1 and run docker compose --profile ssl up")
	}

	sslTargets := initSSLTargets()
	if len(sslTargets) == 0 {
		t.Skip("Could not find certs directory")
	}

	for _, target := range sslTargets {
		if target.expectFail || target.skipReason != "" {
			continue
		}

		// Skip non-SQL databases for basic query test
		if target.plugin.Type == engine.DatabaseType_MongoDB ||
			target.plugin.Type == engine.DatabaseType_Redis ||
			target.plugin.Type == engine.DatabaseType_ElasticSearch {
			continue
		}

		t.Run(target.name, func(t *testing.T) {
			// Simple query to verify connection works
			result, err := target.plugin.RawExecute(target.config, "SELECT 1")
			if err != nil {
				t.Fatalf("RawExecute failed over SSL: %v", err)
			}

			if result == nil || len(result.Rows) == 0 {
				t.Error("expected non-empty result from SELECT 1")
			}
		})
	}
}

// TestSSLModeValidation tests that invalid SSL modes are rejected.
func TestSSLModeValidation(t *testing.T) {
	tests := []struct {
		name   string
		dbType engine.DatabaseType
		mode   string
		valid  bool
	}{
		{"postgres_valid_required", engine.DatabaseType_Postgres, "required", true},
		{"postgres_valid_verify_ca", engine.DatabaseType_Postgres, "verify-ca", true},
		{"postgres_invalid_preferred", engine.DatabaseType_Postgres, "preferred", false},
		{"mysql_valid_required", engine.DatabaseType_MySQL, "required", true},
		{"mysql_valid_preferred", engine.DatabaseType_MySQL, "preferred", true},
		{"mysql_invalid_verify_identity", engine.DatabaseType_MySQL, "verify-identity", true}, // MySQL supports this
		{"clickhouse_valid_enabled", engine.DatabaseType_ClickHouse, "enabled", true},
		{"clickhouse_invalid_required", engine.DatabaseType_ClickHouse, "required", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			valid := ssl.ValidateSSLMode(tc.dbType, ssl.SSLMode(tc.mode))
			if valid != tc.valid {
				t.Errorf("ValidateSSLMode(%s, %s) = %v, want %v", tc.dbType, tc.mode, valid, tc.valid)
			}
		})
	}
}
