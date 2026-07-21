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
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/clickhouse"
	"github.com/clidey/whodb/core/src/plugins/elasticsearch"
	"github.com/clidey/whodb/core/src/plugins/mongodb"
	"github.com/clidey/whodb/core/src/plugins/mysql"
	"github.com/clidey/whodb/core/src/plugins/postgres"
	"github.com/clidey/whodb/core/src/plugins/redis"
	"github.com/clidey/whodb/core/src/query"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

type target struct {
	name             string
	plugin           *engine.Plugin
	config           *engine.PluginConfig
	schema           string
	enabled          bool
	readySchema      string
	readyStorageUnit string
}

var targets []target

func TestMain(m *testing.M) {
	if os.Getenv("WHODB_START_COMPOSE") == "1" {
		if err := runComposeUp(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to start docker-compose: %v\n", err)
			os.Exit(1)
		}
	}

	waitForServices()
	initTargets()
	if err := waitForSeededTargets(); err != nil {
		fmt.Fprintf(os.Stderr, "seeded data did not become ready: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func runComposeUp() error {
	cmd := exec.Command("docker", "compose", "-f", "dev/docker-compose.yml", "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForServices() {
	ports := []string{"5432", "3306", "3307", "3308", "27017", "9000", "6379", "8123", "9200"}
	for _, p := range ports {
		waitForPort("127.0.0.1", p, 2*time.Minute)
	}
}

func waitForPort(host, port string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 3*time.Second)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func initTargets() {
	targets = []target{
		{
			name:   "postgres",
			plugin: postgres.NewPostgresPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_Postgres),
				Hostname: "localhost",
				Username: "user",
				Password: "jio53$*(@nfe)",
				Database: "test_db",
			}),
			schema:           "public",
			enabled:          true,
			readySchema:      "test_schema",
			readyStorageUnit: "orders",
		},
		{
			name:   "mysql",
			plugin: mysql.NewMySQLPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MySQL),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
			}),
			schema:           "test_db",
			enabled:          true,
			readySchema:      "test_db",
			readyStorageUnit: "orders",
		},
		{
			name:   "mariadb",
			plugin: mysql.NewMyMariaDBPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MariaDB),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{{Key: "Port", Value: "3307"}},
			}),
			schema:           "test_db",
			enabled:          true,
			readySchema:      "test_db",
			readyStorageUnit: "orders",
		},
		{
			name:   "mysql842",
			plugin: mysql.NewMySQLPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MySQL),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{{Key: "Port", Value: "3308"}},
			}),
			schema:           "test_db",
			enabled:          true,
			readySchema:      "test_db",
			readyStorageUnit: "orders",
		},
		{
			name:   "clickhouse",
			plugin: clickhouse.NewClickHousePlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_ClickHouse),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
				Advanced: []engine.Record{{Key: "Port", Value: "9000"}},
			}),
			schema:           "test_db",
			enabled:          true,
			readySchema:      "test_db",
			readyStorageUnit: "orders",
		},
		{
			name:   "mongo",
			plugin: mongodb.NewMongoDBPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MongoDB),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db",
			}),
			schema:           "test_db",
			enabled:          true,
			readySchema:      "test_db",
			readyStorageUnit: "orders",
		},
		{
			name:   "elasticsearch",
			plugin: elasticsearch.NewElasticSearchPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_ElasticSearch),
				Hostname: "localhost",
				Advanced: []engine.Record{{Key: "Port", Value: "9200"}},
			}),
			schema:           "",
			enabled:          true,
			readyStorageUnit: "orders",
		},
		{
			name:   "redis",
			plugin: redis.NewRedisPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_Redis),
				Hostname: "localhost",
				Password: "password",
				Database: "0",
				Advanced: []engine.Record{{Key: "Port", Value: "6379"}},
			}),
			schema:           "",
			enabled:          true,
			readyStorageUnit: "orders:recent",
		},
	}
}

func sessionMetadataForTarget(target target) *source.TypeSessionMetadata {
	ids := []string{}
	if target.config != nil && target.config.Credentials != nil {
		ids = append(ids, target.config.Credentials.Type)
	}
	if target.plugin != nil {
		ids = append(ids, string(target.plugin.Type))
	}

	metadata, ok := sourcecatalog.ResolveSessionMetadata(ids...)
	if !ok {
		return nil
	}
	return metadata
}

func engineTypeDefinition(td source.TypeDefinition) engine.TypeDefinition {
	return engine.TypeDefinition{
		ID:               td.ID,
		Label:            td.Label,
		HasLength:        td.HasLength,
		HasPrecision:     td.HasPrecision,
		DefaultLength:    td.DefaultLength,
		DefaultPrecision: td.DefaultPrecision,
		Category:         engine.TypeCategory(td.Category),
		InsertFunc:       td.InsertFunc,
		TableModel:       td.TableModel,
		DDLSuffix:        td.DDLSuffix,
	}
}

func waitForSeededTargets() error {
	for _, target := range targets {
		if !target.enabled {
			continue
		}
		if err := waitForSeededTarget(target, 5*time.Minute); err != nil {
			return err
		}
	}
	return nil
}

func waitForSeededTarget(target target, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	readySchema := target.readySchema
	if readySchema == "" {
		readySchema = target.schema
	}

	for time.Now().Before(deadline) {
		if !target.plugin.IsAvailable(context.Background(), target.config) {
			lastErr = fmt.Errorf("%s plugin is not available yet", target.name)
			time.Sleep(2 * time.Second)
			continue
		}

		if target.readyStorageUnit == "" {
			return nil
		}

		exists, err := target.plugin.StorageUnitExists(target.config, readySchema, target.readyStorageUnit)
		if err == nil && exists {
			rows, rowsErr := target.plugin.GetRows(target.config, &engine.GetRowsRequest{
				Schema:      readySchema,
				StorageUnit: target.readyStorageUnit,
				Sort:        []*query.SortCondition{},
				PageSize:    1,
			})
			if rowsErr == nil && len(rows.Rows) > 0 {
				return nil
			}
			if rowsErr != nil {
				lastErr = rowsErr
			}
		} else if err != nil {
			lastErr = err
		}

		time.Sleep(2 * time.Second)
	}

	if lastErr != nil {
		return fmt.Errorf("%s readiness check failed: %w", target.name, lastErr)
	}
	return fmt.Errorf("%s readiness check timed out", target.name)
}
