package duckdb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
)

func readOnlyFixture(t *testing.T) (*DuckDBPlugin, *engine.PluginConfig, *gorm.DB) {
	t.Helper()
	t.Setenv("WHODB_CLI", "true")
	path := filepath.Join(t.TempDir(), "guard.duckdb")
	db, err := gorm.Open(Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("CREATE TABLE guard (id INTEGER PRIMARY KEY); INSERT INTO guard VALUES (1)").Error; err != nil {
		t.Fatal(err)
	}
	p := NewDuckDBPlugin().PluginFunctions.(*DuckDBPlugin)
	config := engine.NewPluginConfig(&engine.Credentials{Type: string(engine.DatabaseType_DuckDB), Database: path})
	config.Context = t.Context()
	t.Cleanup(func() { plugins.RemoveConnection(config) })
	return p, config, db
}

func TestDuckDBProtectedReads(t *testing.T) {
	p, config, _ := readOnlyFixture(t)
	// Keep the ordinary cached connection open while protected connections run.
	if _, err := p.RawExecute(config, "INSERT INTO guard VALUES (?)", 2); err != nil {
		t.Fatal(err)
	}
	protected := *config
	protected.ReadOnly = true
	for _, query := range []string{
		"SELECT id FROM guard WHERE id = ?",
		"WITH c AS (SELECT id FROM guard WHERE id = ?) SELECT * FROM c",
	} {
		rows, err := p.RawExecute(&protected, query, 2)
		if err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		if len(rows.Rows) != 1 || rows.Rows[0][0] != "2" {
			t.Fatalf("%s: unexpected result: %#v", query, rows.Rows)
		}
	}
	if _, err := p.RawExecute(&protected, "SELECT 1; SELECT 2"); !errors.Is(err, engine.ErrMultiStatementUnsupported) {
		t.Fatalf("expected existing batch restriction: %v", err)
	}
	if _, err := p.RawExecute(&protected, "SELECT * FROM missing_table"); err == nil {
		t.Fatal("expected query error")
	}
	if _, err := p.RawExecute(&protected, "SELECT count(*) FROM guard"); err != nil {
		t.Fatalf("read after error: %v", err)
	}
	if _, err := p.RawExecute(config, "INSERT INTO guard VALUES (?)", 3); err != nil {
		t.Fatalf("ordinary connection became read-only: %v", err)
	}
	rows, err := p.RawExecute(&protected, "SELECT count(*) FROM guard")
	if err != nil || len(rows.Rows) != 1 || rows.Rows[0][0] != "3" {
		t.Fatalf("protected read did not see committed writes: %#v, %v", rows, err)
	}
	protected.Transaction = &gorm.DB{}
	if _, err := p.RawExecute(&protected, "SELECT 1"); err == nil {
		t.Fatal("protected read joined an existing transaction")
	}
}

func TestDuckDBNativeReadOnlyTransaction(t *testing.T) {
	p, _, db := readOnlyFixture(t)
	// Bypass the classifier to prove the engine enforces the transaction mode.
	for _, query := range []string{
		"INSERT INTO guard VALUES (2)", "UPDATE guard SET id = 2",
		"DELETE FROM guard", "CREATE TABLE bad AS SELECT * FROM guard",
		"ALTER TABLE guard ADD COLUMN extra INTEGER", "DROP TABLE guard",
	} {
		t.Run(query, func(t *testing.T) {
			if _, err := p.executeReadOnly(db.WithContext(t.Context()), query); err == nil || !strings.Contains(err.Error(), "read-only mode") {
				t.Fatalf("expected native read-only rejection: %v", err)
			}
		})
	}
	rows, err := p.executeReadOnly(db.WithContext(t.Context()), "SELECT id FROM guard")
	if err != nil || len(rows.Rows) != 1 || rows.Rows[0][0] != "1" {
		t.Fatalf("guard changed or transaction leaked: %#v, %v", rows, err)
	}
}

func TestDuckDBProtectedConcurrentRequests(t *testing.T) {
	p, config, _ := readOnlyFixture(t)
	protected := *config
	protected.ReadOnly = true
	results := make(chan error, 9)
	for range 8 {
		go func() {
			for range 5 {
				rows, err := p.RawExecute(&protected, "SELECT id FROM guard WHERE id = ?", 1)
				if err != nil {
					results <- err
					return
				}
				if len(rows.Rows) != 1 || rows.Rows[0][0] != "1" {
					results <- fmt.Errorf("unexpected protected result: %#v", rows.Rows)
					return
				}
			}
			results <- nil
		}()
	}
	go func() {
		for id := 2; id <= 6; id++ {
			if _, err := p.RawExecute(config, "INSERT INTO guard VALUES (?)", id); err != nil {
				results <- err
				return
			}
		}
		results <- nil
	}()
	for range 9 {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	rows, err := p.RawExecute(&protected, "SELECT count(*) FROM guard")
	if err != nil || len(rows.Rows) != 1 || rows.Rows[0][0] != "6" {
		t.Fatalf("concurrent writes not visible: %#v, %v", rows, err)
	}
}

func TestDuckDBProtectedSideEffects(t *testing.T) {
	p, config, db := readOnlyFixture(t)
	output := filepath.Join(t.TempDir(), "output.csv")
	copyQuery := "COPY guard TO '" + strings.ReplaceAll(output, "'", "''") + "' (HEADER)"
	// COPY TO is allowed even by the native read-only transaction. The classifier
	// is necessary, and a successful export proves the negative check is meaningful.
	if _, err := p.executeReadOnly(db.WithContext(t.Context()), copyQuery); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}
	config.ReadOnly = true
	for _, query := range []string{
		copyQuery, "INSERT INTO guard VALUES (2)",
		"WITH c AS (SELECT 1) DELETE FROM guard",
		"SELECT 1; COMMIT; INSERT INTO guard VALUES (2)",
		"SELECT 1; " + copyQuery, "SELECT nextval('sequence')",
		"ATTACH ':memory:' AS other", "SET enable_external_access = true",
		"SELECT * FROM read_csv('" + strings.ReplaceAll(output, "'", "''") + "')",
	} {
		if _, err := p.RawExecute(config, query); err == nil || !strings.Contains(err.Error(), "read-only query rejected") {
			t.Fatalf("expected classifier rejection for %s: %v", query, err)
		}
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("export created a file: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	config.Context = ctx
	if _, err := p.RawExecute(config, "SELECT 1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	config.Context = t.Context()
	if _, err := p.RawExecute(config, "SELECT 1"); err != nil {
		t.Fatalf("read after cancellation: %v", err)
	}
	deadline, stop := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer stop()
	config.Context = deadline
	if _, err := p.RawExecute(config, "WITH RECURSIVE c(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM c WHERE n < 1000000000) SELECT sum(n) FROM c"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected long query to be interrupted: %v", err)
	}
	config.Context = t.Context()
	if _, err := p.RawExecute(config, "SELECT 1"); err != nil {
		t.Fatalf("read after query timeout: %v", err)
	}
}
