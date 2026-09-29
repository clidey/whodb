//go:build integration

package clickhouse

import (
	"fmt"
	"os"
	"testing"
	"time"

	driver "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/clidey/whodb/core/src/engine"
	_ "github.com/clidey/whodb/core/src/sources/database"
)

func TestClickHouseProtectedExecution(t *testing.T) {
	if os.Getenv("WHODB_READGUARD_LOCAL") != "1" {
		t.Fatal("requires disposable local readguard ClickHouse and WHODB_READGUARD_LOCAL=1")
	}
	plugin := NewClickHousePlugin().PluginFunctions.(*ClickHousePlugin)
	config := engine.NewPluginConfig(&engine.Credentials{Type: "ClickHouse", Hostname: "127.0.0.1", Username: "guard", Password: "guard-local", Database: "guard", Advanced: []engine.Record{{Key: "Port", Value: "59009"}}})
	config.Context = t.Context()
	db, err := plugin.DB(config)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	table := fmt.Sprintf("readguard_%d", time.Now().UnixNano())
	if err := db.Exec("CREATE TABLE " + table + " (id Int32) ENGINE=MergeTree ORDER BY id").Error; err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP TABLE " + table)
	if err := db.Exec("INSERT INTO " + table + " VALUES (1)").Error; err != nil {
		t.Fatal(err)
	}
	// Exercise the actual native setting independently of classification.
	protected := db.WithContext(driver.Context(t.Context(), driver.WithSettings(driver.Settings{"readonly": 1})))
	if err := protected.Exec("INSERT INTO " + table + " VALUES (999)").Error; err == nil {
		t.Fatal("native readonly setting allowed a write")
	}
	config.ReadOnly = true
	for _, query := range []string{"INSERT INTO " + table + " VALUES (2)", "SELECT * FROM " + table + " INTO OUTFILE '/tmp/readguard'", "SELECT * FROM file('/tmp/readguard')", "SELECT 1 SETTINGS readonly=0"} {
		if _, err := plugin.RawExecute(config, query); err == nil {
			t.Fatalf("protected query accepted: %s", query)
		}
	}
	result, err := plugin.RawExecute(config, "SELECT count(*) FROM "+table)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "1" {
		t.Fatalf("unexpected protected result: %#v", result)
	}
}
