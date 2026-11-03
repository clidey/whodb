package plugins_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/clickhouse"
	"github.com/clidey/whodb/core/src/plugins/elasticsearch"
	"github.com/clidey/whodb/core/src/plugins/memcached"
	"github.com/clidey/whodb/core/src/plugins/mongodb"
	"github.com/clidey/whodb/core/src/plugins/mysql"
	"github.com/clidey/whodb/core/src/plugins/postgres"
	"github.com/clidey/whodb/core/src/plugins/redis"
	"github.com/clidey/whodb/core/src/sourcedial"
)

// testHost is a documentation address (RFC 5737); the refusing dialer means
// nothing is ever sent to it.
const testHost = "203.0.113.10"

var errRefused = errors.New("refused by the test dialer")

// refuseDials registers a dialer that records each address it is asked for
// and refuses the connection.
func refuseDials(t *testing.T) func() []string {
	t.Helper()
	var mu sync.Mutex
	var addrs []string
	sourcedial.SetDialer(func(_ context.Context, network, addr string) (net.Conn, error) {
		mu.Lock()
		defer mu.Unlock()
		addrs = append(addrs, network+" "+addr)
		return nil, errRefused
	})
	t.Cleanup(func() { sourcedial.SetDialer(nil) })
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), addrs...)
	}
}

// Every connector must open its connections through the registered source
// dialer, so a connection-time destination policy sees each real address.
// A connector that falls back to its driver's own dialing fails here.
func TestConnectorsDialThroughTheRegisteredSourceDialer(t *testing.T) {
	credentials := func(databaseType, port string, advanced ...engine.Record) *engine.PluginConfig {
		database := "db"
		if databaseType == engine.DatabaseType_Redis {
			database = "0"
		}
		return engine.NewPluginConfig(&engine.Credentials{
			Type:     databaseType,
			Hostname: testHost,
			Username: "user",
			Password: "password",
			Database: database,
			Advanced: append([]engine.Record{{Key: "Port", Value: port}}, advanced...),
		})
	}
	connectors := []struct {
		name    string
		port    string
		connect func() error
	}{
		{"postgres", "5432", func() error {
			_, err := postgres.NewPostgresPlugin().PluginFunctions.(*postgres.PostgresPlugin).DB(credentials(engine.DatabaseType_Postgres, "5432"))
			return err
		}},
		{"mysql", "3306", func() error {
			_, err := mysql.NewMySQLPlugin().PluginFunctions.(*mysql.MySQLPlugin).DB(credentials(engine.DatabaseType_MySQL, "3306"))
			return err
		}},
		{"mongodb", "27017", func() error {
			// Fail server selection quickly instead of after the 30s default.
			_, err := mongodb.DB(credentials(engine.DatabaseType_MongoDB, "27017", engine.Record{Key: "URL Params", Value: "?serverSelectionTimeoutMS=500"}))
			return err
		}},
		{"redis", "6379", func() error {
			_, err := redis.DB(credentials(engine.DatabaseType_Redis, "6379"))
			return err
		}},
		{"clickhouse over http", "8123", func() error {
			_, err := clickhouse.NewClickHousePlugin().PluginFunctions.(*clickhouse.ClickHousePlugin).DB(credentials(engine.DatabaseType_ClickHouse, "8123"))
			return err
		}},
		{"clickhouse native", "9000", func() error {
			_, err := clickhouse.NewClickHousePlugin().PluginFunctions.(*clickhouse.ClickHousePlugin).DB(credentials(engine.DatabaseType_ClickHouse, "9000", engine.Record{Key: "HTTP Protocol", Value: "disable"}))
			return err
		}},
		{"elasticsearch", "9200", func() error {
			_, err := elasticsearch.DB(credentials(engine.DatabaseType_ElasticSearch, "9200"))
			return err
		}},
		{"memcached", "11211", func() error {
			_, err := memcached.DB(credentials(engine.DatabaseType_Memcached, "11211"))
			return err
		}},
	}
	for _, connector := range connectors {
		t.Run(connector.name, func(t *testing.T) {
			dialed := refuseDials(t)
			if err := connector.connect(); err == nil {
				t.Fatal("connected although every dial was refused")
			}
			want := net.JoinHostPort(testHost, connector.port)
			for _, addr := range dialed() {
				if strings.HasSuffix(addr, " "+want) {
					return
				}
			}
			t.Fatalf("the source dialer was never asked for %s; dials: %v", want, dialed())
		})
	}
}
