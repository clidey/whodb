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

// Package doctor runs database connection and metadata diagnostics for the CLI.
package doctor

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
)

const (
	statusOK    = "ok"
	statusError = "error"
	statusWarn  = "warn"
)

// Options configures a doctor run.
type Options struct {
	Connection string
	Schema     string
}

// Report is the JSON-serializable doctor output.
type Report struct {
	Connection     ConnectionSummary `json:"connection"`
	Checks         []Check           `json:"checks"`
	SuggestedTools []string          `json:"suggested_tools,omitempty"`
}

// ConnectionSummary is a redacted connection description.
type ConnectionSummary struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Source   string `json:"source,omitempty"`
}

// Check describes one diagnostic step.
type Check struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	DurationMS int64  `json:"duration_ms,omitempty"`
	Count      int    `json:"count,omitempty"`
	Schema     string `json:"schema,omitempty"`
}

// Run executes the doctor checks for one connection.
func Run(ctx context.Context, opts Options) (Report, error) {
	_ = ctx

	report := Report{}
	start := time.Now()
	mgr, err := dbmgr.NewManager()
	if err != nil {
		return report, fmt.Errorf("cannot initialize database manager: %w", err)
	}
	report.Checks = append(report.Checks, Check{
		Name:       "config_load",
		Status:     statusOK,
		DurationMS: elapsed(start),
	})

	conn, sourceName, err := resolveConnection(mgr, opts.Connection)
	if err != nil {
		report.Checks = append(report.Checks, Check{Name: "connection_resolve", Status: statusError, Message: err.Error()})
		return report, err
	}
	report.Connection = summarizeConnection(conn, sourceName)
	report.Checks = append(report.Checks, Check{Name: "connection_resolve", Status: statusOK})

	connectStart := time.Now()
	if err := mgr.Connect(conn); err != nil {
		report.Checks = append(report.Checks, Check{
			Name:       "connect",
			Status:     statusError,
			Message:    err.Error(),
			DurationMS: elapsed(connectStart),
		})
		return report, nil
	}
	report.Checks = append(report.Checks, Check{
		Name:       "connect",
		Status:     statusOK,
		DurationMS: elapsed(connectStart),
	})
	defer mgr.Disconnect() //nolint:errcheck

	schemaStart := time.Now()
	resolvedSchema, err := mgr.ResolveSnapshotSchema(conn, opts.Schema)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:       "schema_resolve",
			Status:     statusError,
			Message:    err.Error(),
			DurationMS: elapsed(schemaStart),
		})
		return report, nil
	}
	report.Connection.Schema = resolvedSchema
	report.Checks = append(report.Checks, Check{
		Name:       "schema_resolve",
		Status:     statusOK,
		Schema:     resolvedSchema,
		DurationMS: elapsed(schemaStart),
	})

	schemasStart := time.Now()
	schemas, err := mgr.GetSchemas()
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:       "schemas",
			Status:     statusWarn,
			Message:    err.Error(),
			DurationMS: elapsed(schemasStart),
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:       "schemas",
			Status:     statusOK,
			Count:      len(schemas),
			DurationMS: elapsed(schemasStart),
		})
	}

	tablesStart := time.Now()
	tables, err := mgr.GetStorageUnits(resolvedSchema)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:       "storage_units",
			Status:     statusError,
			Message:    err.Error(),
			Schema:     resolvedSchema,
			DurationMS: elapsed(tablesStart),
		})
		return report, nil
	}
	report.Checks = append(report.Checks, Check{
		Name:       "storage_units",
		Status:     statusOK,
		Count:      len(tables),
		Schema:     resolvedSchema,
		DurationMS: elapsed(tablesStart),
	})

	sslStart := time.Now()
	if summary, err := mgr.GetSSLStatusSummary(); err == nil && strings.TrimSpace(summary) != "" {
		report.Checks = append(report.Checks, Check{
			Name:       "ssl_status",
			Status:     statusOK,
			Message:    summary,
			DurationMS: elapsed(sslStart),
		})
	}

	report.SuggestedTools = suggestedTools(len(tables))
	return report, nil
}

func resolveConnection(mgr *dbmgr.Manager, name string) (*dbmgr.Connection, string, error) {
	if strings.TrimSpace(name) != "" {
		return mgr.ResolveConnection(name)
	}

	connections := mgr.ListConnectionsWithSource()
	if len(connections) == 0 {
		return nil, "", fmt.Errorf("no connections available")
	}
	conn := connections[0].Connection
	return new(conn), connections[0].Source, nil
}

func summarizeConnection(conn *dbmgr.Connection, sourceName string) ConnectionSummary {
	if conn == nil {
		return ConnectionSummary{}
	}
	return ConnectionSummary{
		Name:     conn.Name,
		Type:     conn.Type,
		Host:     conn.Host,
		Port:     conn.Port,
		Database: conn.Database,
		Schema:   conn.Schema,
		Source:   sourceName,
	}
}

func suggestedTools(tableCount int) []string {
	tools := []string{"whodb_tables", "whodb_suggestions"}
	if tableCount > 0 {
		tools = append(tools, "whodb_erd", "whodb_audit")
	}
	return tools
}

func elapsed(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
