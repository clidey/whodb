//go:build integration

package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/clidey/whodb/core/src/plugins/duckdb"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDuckDBReadGuardTransport(t *testing.T) {
	binary := os.Getenv("WHODB_READGUARD_BINARY")
	for _, mode := range []string{"readonly", "confirm", "strict"} {
		t.Run(mode, func(t *testing.T) {
			setupTestEnv(t)
			t.Setenv("WHODB_CLI", "true")
			t.Setenv("WHODB_CLI_ANALYTICS_DISABLED", "true")
			dir := t.TempDir()
			path := filepath.Join(dir, "guard.duckdb")
			seed, err := sql.Open("duckdb", path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := seed.Exec("CREATE TABLE guard(id INTEGER PRIMARY KEY); INSERT INTO guard VALUES (1)"); err != nil {
				_ = seed.Close()
				t.Fatal(err)
			}
			// DuckDB file writers cannot span processes. Close the fixture before
			// launching a compiled CLI; all subsequent reads use its MCP endpoint.
			if err := seed.Close(); err != nil {
				t.Fatal(err)
			}
			profile, err := json.Marshal([]map[string]string{{"alias": "guard", "database": path}})
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("WHODB_DUCKDB", string(profile))
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()
			opts := &ServerOptions{ReadOnly: mode == "readonly", ConfirmWrites: mode != "readonly", SecurityLevel: SecurityLevelStandard, AllowMultiStatement: true}
			if mode == "strict" {
				opts.SecurityLevel = SecurityLevelStrict
			}
			endpoint := readGuardEndpoint(t, ctx, binary, opts)
			client := sdk.NewClient(&sdk.Implementation{Name: "duckdb-readguard-test", Version: "1"}, nil)
			session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: endpoint}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			call := func(tool string, args map[string]any) QueryOutput {
				t.Helper()
				response, err := session.CallTool(ctx, &sdk.CallToolParams{Name: tool, Arguments: args})
				if err != nil {
					t.Fatal(err)
				}
				data, err := json.Marshal(response.StructuredContent)
				if err != nil {
					t.Fatal(err)
				}
				var output QueryOutput
				if err := json.Unmarshal(data, &output); err != nil {
					t.Fatal(err)
				}
				return output
			}
			query := func(text string, params ...any) QueryOutput {
				return call("whodb_query", map[string]any{"connection": "guard", "query": text, "parameters": params})
			}
			assertValue := func(output QueryOutput, want string) {
				t.Helper()
				if output.Error != "" || output.ConfirmationRequired || len(output.Rows) != 1 || len(output.Rows[0]) != 1 || fmt.Sprint(output.Rows[0][0]) != want {
					t.Fatalf("expected %s: %#v", want, output)
				}
			}
			assertValue(query("SELECT id FROM guard WHERE id = ?", 1), "1")
			assertValue(query("WITH c AS (SELECT id FROM guard) SELECT count(*) FROM c"), "1")
			outputPath := filepath.Join(dir, "export.csv")
			for _, attack := range []struct {
				sql    string
				params []any
				file   bool
			}{
				{"CREATE TABLE created AS SELECT * FROM guard", nil, false},
				{"INSERT INTO guard VALUES (?)", []any{2}, false},
				{"COPY guard TO '" + strings.ReplaceAll(outputPath, "'", "''") + "' (HEADER)", nil, true},
			} {
				pending := query(attack.sql, attack.params...)
				if attack.file {
					if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("file created before approval: %v", err)
					}
				} else if len(attack.params) > 0 {
					assertValue(query("SELECT count(*) FROM guard"), "1")
				} else {
					assertValue(query("SELECT count(*) FROM information_schema.tables WHERE table_name = 'created'"), "0")
				}
				if mode == "readonly" || mode == "strict" && attack.file {
					if pending.Error == "" || pending.ConfirmationRequired {
						t.Fatalf("expected rejection: %#v", pending)
					}
					continue
				}
				if pending.Error != "" || !pending.ConfirmationRequired {
					t.Fatalf("expected approval: %#v", pending)
				}
				confirmed := call("whodb_confirm", map[string]any{"token": pending.ConfirmationToken})
				if confirmed.Error != "" {
					t.Fatalf("approved operation failed: %s", confirmed.Error)
				}
				if replay := call("whodb_confirm", map[string]any{"token": pending.ConfirmationToken}); replay.Error == "" {
					t.Fatal("approval replay accepted")
				}
				if attack.file {
					if _, err := os.Stat(outputPath); err != nil {
						t.Fatalf("approved export did not create a file: %v", err)
					}
				} else if len(attack.params) > 0 {
					assertValue(query("SELECT id FROM guard WHERE id = ?", 2), "2")
				} else {
					assertValue(query("SELECT count(*) FROM created"), "1")
				}
			}
			for _, text := range []string{"SELECT 1; COMMIT; DELETE FROM guard", "SELECT nextval('s')"} {
				out := query(text)
				if out.Error == "" && !out.ConfirmationRequired {
					t.Fatalf("unsafe SQL executed: %s", text)
				}
			}
			if out := query("SELECT * FROM missing_table"); out.Error == "" {
				t.Fatal("expected query error")
			}
			assertValue(query("SELECT id FROM guard WHERE id = ?", 1), "1")
		})
	}
}
