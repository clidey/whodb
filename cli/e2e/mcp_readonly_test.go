//go:build e2e_postgres

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

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/jackc/pgx/v5"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type queryToolOutput struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	Error   string     `json:"error"`
}

func TestPostgres_MCPReadOnlyRejectsAdvisoryBypasses(t *testing.T) {
	cfg := testharness.DefaultPostgresConfig()
	cleanupEnv := testharness.SetupEnv(t, cfg)
	t.Cleanup(cleanupEnv)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close(context.Background()) })

	const table = "public.mcp_read_only_advisory"
	if _, err := connection.Exec(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
		t.Fatalf("drop stale advisory table: %v", err)
	}
	if _, err := connection.Exec(ctx, "CREATE TABLE "+table+" (id integer PRIMARY KEY, v text NOT NULL)"); err != nil {
		t.Fatalf("create advisory table: %v", err)
	}
	t.Cleanup(func() { _, _ = connection.Exec(context.Background(), "DROP TABLE IF EXISTS "+table) })
	if _, err := connection.Exec(ctx, "INSERT INTO "+table+" VALUES (1, 'orig'), (2, 'orig'), (3, 'orig')"); err != nil {
		t.Fatalf("seed advisory table: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate MCP port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release MCP port: %v", err)
	}

	var serverLog bytes.Buffer
	command := exec.CommandContext(context.Background(), testharness.CLIBinaryPath(t),
		"mcp", "serve", "--read-only", "--transport", "http",
		"--host", "127.0.0.1", "--port", strconv.Itoa(port),
	)
	command.Env = os.Environ()
	command.Stdout = &serverLog
	command.Stderr = &serverLog
	if err := command.Start(); err != nil {
		t.Fatalf("start MCP server: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	var stopOnce sync.Once
	stopServer := func() {
		stopOnce.Do(func() {
			_ = command.Process.Signal(os.Interrupt)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = command.Process.Kill()
				<-done
			}
		})
	}
	t.Cleanup(stopServer)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForMCPHealth(t, baseURL+"/health", stopServer, &serverLog)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "advisory-e2e", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcpsdk.StreamableClientTransport{Endpoint: baseURL + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	control := callQueryTool(t, ctx, session, "SELECT count(*) FROM "+table)
	if control.Error != "" || len(control.Rows) != 1 || len(control.Rows[0]) != 1 || control.Rows[0][0] != "3" {
		t.Fatalf("unexpected read-only control result: %#v", control)
	}

	queries := []string{
		"DELETE FROM " + table + " WHERE id=1",
		"WITH x AS (DELETE FROM " + table + " WHERE id=2 RETURNING *) SELECT * FROM x",
		"EXPLAIN ANALYZE INSERT INTO " + table + " VALUES (99, 'pwned')",
	}
	for _, query := range queries {
		output := callQueryTool(t, ctx, session, query)
		if !strings.Contains(output.Error, "write operations are not allowed in read-only mode") {
			t.Errorf("expected read-only rejection for %q, got %#v", query, output)
		}
	}

	var count, inserted99 int
	var originalValues bool
	if err := connection.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE id = 99), bool_and(v = 'orig')
		FROM public.mcp_read_only_advisory
	`).Scan(&count, &inserted99, &originalValues); err != nil {
		t.Fatalf("verify advisory table state: %v", err)
	}
	if count != 3 || inserted99 != 0 || !originalValues {
		t.Fatalf("read-only MCP queries modified PostgreSQL: count=%d id99=%d original_values=%t", count, inserted99, originalValues)
	}
}

func waitForMCPHealth(t *testing.T, healthURL string, stopServer func(), serverLog *bytes.Buffer) {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(healthURL)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	stopServer()
	t.Fatalf("MCP server did not become healthy: %s", serverLog.String())
}

func callQueryTool(t *testing.T, ctx context.Context, session *mcpsdk.ClientSession, query string) queryToolOutput {
	t.Helper()
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "whodb_query",
		Arguments: map[string]any{
			"connection": "test-pg",
			"query":      query,
		},
	})
	if err != nil {
		t.Fatalf("call whodb_query with %q: %v", query, err)
	}
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal whodb_query result for %q: %v", query, err)
	}
	var output queryToolOutput
	if err := json.Unmarshal(payload, &output); err != nil {
		t.Fatalf("decode whodb_query result for %q: %v", query, err)
	}
	return output
}
