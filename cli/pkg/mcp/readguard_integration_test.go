//go:build integration

package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/clidey/whodb/core/src/plugins/mysql"
	_ "github.com/clidey/whodb/core/src/plugins/postgres"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// This suite deliberately targets disposable localhost containers, never saved
// connections. See the read-only security test instructions for their setup.
func TestReadGuardTransport(t *testing.T) {
	if os.Getenv("WHODB_READGUARD_LOCAL") != "1" {
		t.Fatal("requires WHODB_READGUARD_LOCAL=1 and disposable local readguard databases")
	}
	binary := os.Getenv("WHODB_READGUARD_BINARY")
	cases := []struct {
		name, driver, dsn, profile, env, container string
		postgres                                   bool
	}{
		{"Postgres", "pgx", "postgres://guard:guard-local@127.0.0.1:55439/guard?sslmode=disable", `[{"alias":"guard","host":"127.0.0.1","port":"55439","user":"guard","password":"guard-local","database":"guard"}]`, "WHODB_POSTGRES", "whodb-readguard-pg", true},
		{"MySQL", "mysql", "root:guard-local@tcp(127.0.0.1:53319)/guard", `[{"alias":"guard","host":"127.0.0.1","port":"53319","user":"root","password":"guard-local","database":"guard"}]`, "WHODB_MYSQL", "whodb-readguard-mysql", false},
		{"MariaDB", "mysql", "root:guard-local@tcp(127.0.0.1:53329)/guard", `[{"alias":"guard","host":"127.0.0.1","port":"53329","user":"root","password":"guard-local","database":"guard"}]`, "WHODB_MARIADB", "whodb-readguard-maria", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupTestEnv(t)
			t.Setenv(tc.env, tc.profile)
			t.Setenv("WHODB_CLI_ANALYTICS_DISABLED", "true")
			observer, err := sql.Open(tc.driver, tc.dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer observer.Close()
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()
			if err := observer.PingContext(ctx); err != nil {
				t.Fatal(err)
			}
			table := fmt.Sprintf("readguard_%d", time.Now().UnixNano())
			if _, err := observer.ExecContext(ctx, "CREATE TABLE "+table+" (id INTEGER PRIMARY KEY)"); err != nil {
				t.Fatal(err)
			}
			defer observer.ExecContext(context.Background(), "DROP TABLE "+table)
			if _, err := observer.ExecContext(ctx, "INSERT INTO "+table+" VALUES (1)"); err != nil {
				t.Fatal(err)
			}
			t.Run("native read-only transaction", func(t *testing.T) {
				tx, err := observer.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback()
				var count int
				if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 1 {
					t.Fatalf("native read failed: %d %v", count, err)
				}
				if _, err := tx.ExecContext(ctx, "INSERT INTO "+table+" VALUES (999)"); err == nil {
					t.Fatal("driver read-only transaction allowed a write")
				}
			})
			for _, mode := range []string{"readonly", "confirm", "strict"} {
				t.Run(mode, func(t *testing.T) {
					opts := &ServerOptions{ReadOnly: mode == "readonly", ConfirmWrites: mode != "readonly", SecurityLevel: SecurityLevelStandard, AllowMultiStatement: true}
					if mode == "strict" {
						opts.SecurityLevel = SecurityLevelStrict
					}
					endpoint := readGuardEndpoint(t, ctx, binary, opts)

					client := sdk.NewClient(&sdk.Implementation{Name: "readguard-test", Version: "1"}, nil)
					session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: endpoint}, nil)
					if err != nil {
						t.Fatal(err)
					}
					defer session.Close()
					call := func(tool string, args map[string]any) map[string]any {
						t.Helper()
						result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: tool, Arguments: args})
						if err != nil {
							t.Fatal(err)
						}
						data, err := json.Marshal(result.StructuredContent)
						if err != nil {
							t.Fatal(err)
						}
						var out map[string]any
						if err = json.Unmarshal(data, &out); err != nil {
							t.Fatal(err)
						}
						if out == nil {
							t.Fatalf("missing structured response: %#v", result)
						}
						return out
					}
					query := func(sql string, params ...any) map[string]any {
						return call("whodb_query", map[string]any{"connection": "guard", "query": sql, "parameters": params})
					}
					assertOK := func(out map[string]any) {
						t.Helper()
						if out["error"] != nil && out["error"] != "" {
							t.Fatalf("operation failed: %#v", out)
						}
					}
					read := query("SELECT count(*) FROM " + table)
					assertOK(read)
					if read["confirmation_required"] == true {
						t.Fatal("read requires approval")
					}
					placeholder := "?"
					if tc.postgres {
						placeholder = "$1"
					}
					assertOK(query("SELECT id FROM "+table+" WHERE id = "+placeholder, 1))
					outputForms := []string{"OUTFILE", "DUMPFILE"}
					if tc.postgres {
						outputForms = []string{"table"}
					}
					for _, outputForm := range outputForms {
						destination := table + "_" + mode + "_" + strings.ToLower(outputForm)
						file := "/tmp/" + destination
						attack := "SELECT * FROM " + table + " INTO " + outputForm + " '" + file + "'"
						if tc.postgres {
							attack = "SELECT * INTO " + destination + " FROM " + table
						}
						pending := query(attack)
						// A separate observer proves there was no side effect before approval.
						if tc.postgres {
							var exists bool
							if err := observer.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name=$1)", destination).Scan(&exists); err != nil {
								t.Fatal(err)
							}
							if exists {
								t.Fatal("table created without approval")
							}
						} else if err := exec.Command("docker", "exec", tc.container, "test", "!", "-e", file).Run(); err != nil {
							t.Fatal("output file exists before approval")
						}
						if mode == "readonly" || mode == "strict" && !tc.postgres {
							if pending["error"] == nil || pending["error"] == "" {
								t.Fatalf("expected rejection: %#v", pending)
							}
						} else {
							if pending["confirmation_required"] != true {
								t.Fatalf("expected approval: %#v", pending)
							}
							confirmed := call("whodb_confirm", map[string]any{"token": pending["confirmation_token"]})
							assertOK(confirmed)
							replay := call("whodb_confirm", map[string]any{"token": pending["confirmation_token"]})
							if replay["error"] == nil || replay["error"] == "" {
								t.Fatal("approval replay accepted")
							}
							if tc.postgres {
								var count int
								if err := observer.QueryRowContext(ctx, "SELECT count(*) FROM "+destination).Scan(&count); err != nil || count != 1 {
									t.Fatalf("confirmed SELECT INTO failed: %d %v", count, err)
								}
								if _, err := observer.ExecContext(ctx, "DROP TABLE "+destination); err != nil {
									t.Fatal(err)
								}
							} else if err := exec.Command("docker", "exec", tc.container, "test", "-f", file).Run(); err != nil {
								t.Fatal("positive control did not create output file")
							}
						}
					}

					// Unknown function and batch suffixes cannot dispatch automatically.
					for _, q := range []string{"SELECT side_effect()", "SELECT 1; USE another_database", "PRAGMA query_only(OFF)"} {
						out := query(q)
						if out["confirmation_required"] != true && (out["error"] == nil || out["error"] == "") {
							t.Fatalf("unsafe query executed: %s", q)
						}
					}
					if mode == "confirm" {
						pending := query("INSERT INTO "+table+" VALUES ("+placeholder+")", 2)
						if pending["confirmation_required"] != true {
							t.Fatalf("missing parameterized approval: %#v", pending)
						}
						assertOK(call("whodb_confirm", map[string]any{"token": pending["confirmation_token"]}))
						var count int
						if err := observer.QueryRowContext(ctx, "SELECT count(*) FROM "+table+" WHERE id=2").Scan(&count); err != nil || count != 1 {
							t.Fatalf("parameters lost: %d %v", count, err)
						}
						// A failed dispatch consumes approval as well (unique-key error here).
						pending = query("INSERT INTO "+table+" VALUES ("+placeholder+")", 2)
						failed := call("whodb_confirm", map[string]any{"token": pending["confirmation_token"]})
						if failed["error"] == nil || failed["error"] == "" {
							t.Fatal("expected duplicate-key failure")
						}
						replay := call("whodb_confirm", map[string]any{"token": pending["confirmation_token"]})
						if !strings.Contains(fmt.Sprint(replay["error"]), "token") {
							t.Fatalf("failed dispatch approval was reusable: %#v", replay)
						}
						if _, err := observer.ExecContext(ctx, "DELETE FROM "+table+" WHERE id=2"); err != nil {
							t.Fatal(err)
						}
					}
				})
			}
		})
	}
}

func readGuardEndpoint(t *testing.T, ctx context.Context, binary string, opts *ServerOptions) string {
	t.Helper()
	if binary == "" {
		server := NewServer(opts)
		httpServer := httptest.NewServer(sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, nil))
		t.Cleanup(httpServer.Close)
		return httpServer.URL
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	args := []string{"mcp", "serve", "--transport", "http", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--allow-multi-statement"}
	if opts.ReadOnly {
		args = append(args, "--read-only")
	}
	if opts.SecurityLevel == SecurityLevelStrict {
		args = append(args, "--security", "strict")
	}
	command := exec.CommandContext(ctx, binary, args...)
	logFile, err := os.CreateTemp(t.TempDir(), "server-*.log")
	if err != nil {
		t.Fatal(err)
	}
	command.Stdout = logFile
	command.Stderr = logFile
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	t.Cleanup(func() {
		_ = command.Process.Signal(os.Interrupt)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = command.Process.Kill()
			<-done
		}
		_ = logFile.Close()
	})
	endpoint := "http://127.0.0.1:" + strconv.Itoa(port)
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(endpoint + "/health")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return endpoint + "/mcp"
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	logs, _ := os.ReadFile(logFile.Name())
	t.Fatalf("MCP process did not become ready: %s", logs)
	return ""
}
