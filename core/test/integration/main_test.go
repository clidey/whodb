//go:build integration

package integration

import (
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
)

type target struct {
	name    string
	plugin  *engine.Plugin
	config  *engine.PluginConfig
	schema  string
	enabled bool
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

	os.Exit(m.Run())
}

func runComposeUp() error {
	cmd := exec.Command("docker", "compose", "-f", "dev/docker-compose.e2e.yaml", "up", "-d")
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
			schema:  "public",
			enabled: true,
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
			schema:  "test_db",
			enabled: true,
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
			schema:  "test_db",
			enabled: true,
		},
		{
			name:   "mysql842",
			plugin: mysql.NewMySQLPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_MySQL),
				Hostname: "localhost",
				Username: "user",
				Password: "password",
				Database: "test_db_842",
				Advanced: []engine.Record{{Key: "Port", Value: "3308"}},
			}),
			schema:  "test_db_842",
			enabled: true,
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
			schema:  "test_db",
			enabled: true,
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
			schema:  "test_db",
			enabled: true,
		},
		{
			name:   "elasticsearch",
			plugin: elasticsearch.NewElasticSearchPlugin(),
			config: engine.NewPluginConfig(&engine.Credentials{
				Type:     string(engine.DatabaseType_ElasticSearch),
				Hostname: "localhost",
				Advanced: []engine.Record{{Key: "Port", Value: "9200"}},
			}),
			schema:  "",
			enabled: true,
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
			schema:  "",
			enabled: true,
		},
	}
}
