package gorm_plugin

import (
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("failed to open sqlite dry-run db: %v", err)
	}
	return db
}

func varsContain(vars []any, value any) bool {
	for _, v := range vars {
		if v == value {
			return true
		}
	}
	return false
}

func TestSQLBuilderUsesParameterizedQueries(t *testing.T) {
	db := newDryRunDB(t)
	sb := NewSQLBuilder(db, nil)

	injected := "Robert'); DROP TABLE users;--"

	selectQuery := sb.SelectQuery("", "users", []string{"id", "name"}, map[string]any{"name": injected})
	selectResult := selectQuery.Find(&[]map[string]any{})
	selectSQL := selectResult.Statement.SQL.String()

	if strings.Contains(selectSQL, injected) {
		t.Fatalf("expected SELECT SQL to not include raw user input, got %q", selectSQL)
	}
	if !strings.Contains(selectSQL, "?") {
		t.Fatalf("expected SELECT SQL to use placeholders, got %q", selectSQL)
	}
	if !varsContain(selectResult.Statement.Vars, injected) {
		t.Fatalf("expected SELECT vars to include injected value, got %#v", selectResult.Statement.Vars)
	}

	updateResult := sb.UpdateQuery("", "users", map[string]any{"name": injected}, map[string]any{"id": 1})
	updateSQL := updateResult.Statement.SQL.String()
	if strings.Contains(updateSQL, injected) {
		t.Fatalf("expected UPDATE SQL to not include raw user input, got %q", updateSQL)
	}
	if !strings.Contains(updateSQL, "?") {
		t.Fatalf("expected UPDATE SQL to use placeholders, got %q", updateSQL)
	}
	if !varsContain(updateResult.Statement.Vars, injected) {
		t.Fatalf("expected UPDATE vars to include injected value, got %#v", updateResult.Statement.Vars)
	}

	deleteResult := sb.DeleteQuery("", "users", map[string]any{"id": injected})
	deleteSQL := deleteResult.Statement.SQL.String()
	if strings.Contains(deleteSQL, injected) {
		t.Fatalf("expected DELETE SQL to not include raw user input, got %q", deleteSQL)
	}
	if !strings.Contains(deleteSQL, "?") {
		t.Fatalf("expected DELETE SQL to use placeholders, got %q", deleteSQL)
	}
	if !varsContain(deleteResult.Statement.Vars, injected) {
		t.Fatalf("expected DELETE vars to include injected value, got %#v", deleteResult.Statement.Vars)
	}
}

func TestSQLBuilderBuildOrderByAddsClause(t *testing.T) {
	db := newDryRunDB(t)
	sb := NewSQLBuilder(db, nil)

	query := sb.SelectQuery("", "users", []string{"id"}, nil)
	query = sb.BuildOrderBy(query, []plugins.Sort{
		{Column: "name", Direction: plugins.Down},
	})

	result := query.Find(&[]map[string]any{})
	sql := result.Statement.SQL.String()
	if !strings.Contains(strings.ToUpper(sql), "ORDER BY") {
		t.Fatalf("expected ORDER BY in SQL, got %q", sql)
	}
	if !strings.Contains(strings.ToUpper(sql), "DESC") {
		t.Fatalf("expected DESC in SQL, got %q", sql)
	}
}

func TestSQLBuilderCreateTableQuotesIdentifiers(t *testing.T) {
	db := newDryRunDB(t)
	sb := NewSQLBuilder(db, nil)

	ddl := sb.CreateTableQuery("public", "users", []ColumnDef{
		{Name: "id", Type: "INTEGER", Primary: true},
		{Name: "name", Type: "TEXT", Nullable: false},
	})

	if !strings.HasPrefix(ddl, "CREATE TABLE ") {
		t.Fatalf("expected CREATE TABLE prefix, got %q", ddl)
	}

	quotedTable := sb.QuoteIdentifier("public") + "." + sb.QuoteIdentifier("users")
	if !strings.Contains(ddl, quotedTable) {
		t.Fatalf("expected quoted schema/table identifiers %q, got %q", quotedTable, ddl)
	}

	expectedPK := "PRIMARY KEY (" + sb.QuoteIdentifier("id") + ")"
	if !strings.Contains(ddl, expectedPK) {
		t.Fatalf("expected PRIMARY KEY clause %q, got %q", expectedPK, ddl)
	}

	expectedNotNull := sb.QuoteIdentifier("name") + " TEXT NOT NULL"
	if !strings.Contains(ddl, expectedNotNull) {
		t.Fatalf("expected NOT NULL column clause %q, got %q", expectedNotNull, ddl)
	}
}
