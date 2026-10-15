package duckdb

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/plugins"
)

func serverModeDuckDBPath(t *testing.T) string {
	t.Helper()
	t.Setenv("WHODB_CLI", "")
	t.Setenv("WHODB_DESKTOP", "")
	wasDevelopment := env.IsDevelopment
	env.IsDevelopment = true
	t.Cleanup(func() { env.IsDevelopment = wasDevelopment })
	if _, err := os.Stat("tmp"); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir("tmp", 0o755); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove("tmp") })
	} else if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("tmp", "duckdb-security-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "guard.duckdb")
}

func TestDuckDBSourceBlocksExternalAccess(t *testing.T) {
	path := serverModeDuckDBPath(t)
	seed, err := gorm.Open(Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.Exec("CREATE TABLE guard (id INTEGER PRIMARY KEY); INSERT INTO guard VALUES (1)").Error; err != nil {
		t.Fatal(err)
	}
	seedSQLDB, err := seed.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := seedSQLDB.Close(); err != nil {
		t.Fatal(err)
	}
	p := NewDuckDBPlugin().PluginFunctions.(*DuckDBPlugin)
	database, err := filepath.Rel("tmp", path)
	if err != nil {
		t.Fatal(err)
	}
	config := engine.NewPluginConfig(&engine.Credentials{Type: string(engine.DatabaseType_DuckDB), Database: database})
	config.Context = t.Context()
	t.Cleanup(func() { plugins.RemoveConnection(config) })
	db, err := p.DB(config)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	secret := filepath.Join(t.TempDir(), "secret.csv")
	if err := os.WriteFile(secret, []byte("name,value\nsecret,do-not-read\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	unrestricted, err := gorm.Open(Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var value string
	readQuery := "SELECT value FROM read_csv('" + strings.ReplaceAll(secret, "'", "''") + "')"
	if err := unrestricted.Raw(readQuery).Scan(&value).Error; err != nil || value != "do-not-read" {
		t.Fatalf("file-read test precondition failed: value=%q err=%v", value, err)
	}
	unrestrictedSQL, err := unrestricted.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := unrestrictedSQL.Close(); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "output.csv")
	attached := filepath.Join(t.TempDir(), "attached.duckdb")

	for _, query := range []string{
		readQuery,
		"COPY guard TO '" + strings.ReplaceAll(output, "'", "''") + "'",
		"ATTACH '" + strings.ReplaceAll(attached, "'", "''") + "' AS other",
		"SET enable_external_access=true",
		"INSTALL httpfs",
	} {
		t.Run(query, func(t *testing.T) {
			if _, err := p.RawExecute(config, query); err == nil {
				t.Fatalf("restricted DuckDB source executed %q", query)
			}
		})
	}
	for _, path := range []string{output, attached} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("restricted query created %s: %v", path, err)
		}
	}

	first, err := sqlDB.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := sqlDB.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	for _, conn := range []*sql.Conn{first, second} {
		var external, autoInstall, autoLoad, community, locked bool
		err := conn.QueryRowContext(t.Context(), `SELECT
			current_setting('enable_external_access'),
			current_setting('autoinstall_known_extensions'),
			current_setting('autoload_known_extensions'),
			current_setting('allow_community_extensions'),
			current_setting('lock_configuration')`).Scan(&external, &autoInstall, &autoLoad, &community, &locked)
		if err != nil || external || autoInstall || autoLoad || community || !locked {
			t.Fatalf("DuckDB connection lacks security settings: external=%t autoInstall=%t autoLoad=%t community=%t locked=%t err=%v", external, autoInstall, autoLoad, community, locked, err)
		}
	}
	if _, err := second.ExecContext(t.Context(), "COPY guard TO '"+strings.ReplaceAll(output, "'", "''")+"'"); err == nil {
		t.Fatal("second pooled connection exported a file")
	}

	if _, err := p.RawExecute(config, "INSERT INTO guard VALUES (2)"); err != nil {
		t.Fatalf("ordinary write failed: %v", err)
	}
	rows, err := p.RawExecute(config, `SELECT id, json_extract('{"value": 1}', '$.value') FROM guard ORDER BY id`)
	if err != nil || len(rows.Rows) != 2 {
		t.Fatalf("ordinary query failed: rows=%#v err=%v", rows, err)
	}
}

func TestDuckDBLocalModeRetainsExternalFileAccess(t *testing.T) {
	p, config, _ := readOnlyFixture(t)
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("local-file-access"), 0o600); err != nil {
		t.Fatal(err)
	}
	query := "SELECT content FROM read_text('" + strings.ReplaceAll(secret, "'", "''") + "')"
	rows, err := p.RawExecute(config, query)
	if err != nil || len(rows.Rows) != 1 || rows.Rows[0][0] != "local-file-access" {
		t.Fatalf("local DuckDB file read failed: rows=%#v err=%v", rows, err)
	}
}

func TestDuckDBSourceRejectsDSNPathCharacters(t *testing.T) {
	t.Setenv("WHODB_CLI", "true")
	p := NewDuckDBPlugin().PluginFunctions.(*DuckDBPlugin)
	for _, name := range []string{"hash#name.duckdb", "query?name.duckdb"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
				t.Fatal(err)
			}
			config := engine.NewPluginConfig(&engine.Credentials{Type: string(engine.DatabaseType_DuckDB), Database: path})
			if _, err := p.DB(config); err == nil || !strings.Contains(err.Error(), "unsupported character") {
				t.Fatalf("expected unsupported path rejection, got %v", err)
			}
		})
	}
}

func TestDuckDBSourceRejectsExistingUnrestrictedConnection(t *testing.T) {
	path := serverModeDuckDBPath(t)
	unrestricted, err := gorm.Open(Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	unrestrictedSQL, err := unrestricted.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = unrestrictedSQL.Close() })
	p := NewDuckDBPlugin().PluginFunctions.(*DuckDBPlugin)
	database, err := filepath.Rel("tmp", path)
	if err != nil {
		t.Fatal(err)
	}
	config := engine.NewPluginConfig(&engine.Credentials{Type: string(engine.DatabaseType_DuckDB), Database: database})
	if _, err := p.DB(config); err == nil {
		t.Fatal("restricted source reused an unrestricted database instance")
	}
}
