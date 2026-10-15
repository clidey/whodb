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

package mcp

import (
	"time"

	coreaudit "github.com/clidey/whodb/core/src/audit"
)

// recordQueryAudit records one access-class audit event for a SQL statement
// the MCP server ran against a configured connection. It carries the
// connection name, database type, statement type and row count, never the SQL
// text or parameters. With no audit service registered, the default for a
// local CLI, it is a no-op; `whodb mcp serve` registers the NDJSON file sink
// when WHODB_AUDIT_LOG_FILE is set.
func recordQueryAudit(start time.Time, connection, dbType string, statementType StatementType, rowCount int, confirmed bool, err error) {
	event := coreaudit.AuditEvent{
		Timestamp: start,
		Action:    "mcp.query.execute",
		Outcome:   coreaudit.OutcomeSuccess,
		Severity:  coreaudit.SeverityInfo,
		Resource:  coreaudit.Resource{ID: connection, Type: "database_connection", Name: connection},
		Details: map[string]any{
			"db_type":        dbType,
			"statement_type": string(statementType),
			"row_count":      rowCount,
			"confirmed":      confirmed,
		},
		Duration: time.Since(start),
	}
	if err != nil {
		event.Outcome = coreaudit.OutcomeFailure
		event.Severity = coreaudit.SeverityWarn
		event.Error = err.Error()
	}
	coreaudit.Record(event)
}
