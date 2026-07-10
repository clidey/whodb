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
	"errors"
	"fmt"
	"strings"
)

// SecurityLevel defines the strictness of SQL validation for MCP access control.
type SecurityLevel string

const (
	SecurityLevelStrict   SecurityLevel = "strict"   // Blocks writes + dangerous functions
	SecurityLevelStandard SecurityLevel = "standard" // Blocks writes
	SecurityLevelMinimal  SecurityLevel = "minimal"  // Only blocks DROP/TRUNCATE/DELETE without WHERE
)

// SQL validation errors
var (
	ErrWriteNotAllowed      = errors.New("write operations are not allowed in read-only mode")
	ErrMultipleStatements   = errors.New("multiple SQL statements are not allowed")
	ErrDangerousFunction    = errors.New("dangerous database function detected")
	ErrDestructiveOperation = errors.New("destructive operation detected")
)

// StatementType represents the type of SQL statement.
type StatementType string

const (
	StatementSelect   StatementType = "SELECT"
	StatementInsert   StatementType = "INSERT"
	StatementUpdate   StatementType = "UPDATE"
	StatementDelete   StatementType = "DELETE"
	StatementDrop     StatementType = "DROP"
	StatementCreate   StatementType = "CREATE"
	StatementAlter    StatementType = "ALTER"
	StatementTruncate StatementType = "TRUNCATE"
	StatementShow     StatementType = "SHOW"
	StatementDescribe StatementType = "DESCRIBE"
	StatementExplain  StatementType = "EXPLAIN"
	StatementWith     StatementType = "WITH"
	StatementUnknown  StatementType = "UNKNOWN"
)

// dangerousFunctions that are blocked in strict mode.
var dangerousFunctions = []string{
	"PG_TERMINATE_BACKEND", "PG_READ_FILE", "PG_WRITE_FILE", "LO_IMPORT", "LO_EXPORT",
	"LOAD_FILE", "INTO OUTFILE", "INTO DUMPFILE",
	"XP_CMDSHELL", "OPENROWSET", "OPENDATASOURCE",
	"LOAD_EXTENSION", "WRITEFILE", "READFILE",
	"URL(", "FILE(", "REMOTE(",
	"UTL_FILE", "UTL_HTTP",
	"COPY",
}

// dataModifyingKeywords are verbs that can write data, execute code, or exfiltrate
// data. They gate the read-only and confirmation checks so that statements which
// execute writes (including inside CTEs or EXPLAIN ANALYZE) cannot slip through.
var dataModifyingKeywords = []string{
	"INSERT", "UPDATE", "DELETE", "MERGE", "DROP", "TRUNCATE", "ALTER",
	"CREATE", "GRANT", "REVOKE", "REPLACE", "UPSERT", "CALL", "DO", "COPY",
}

// sqlTokens splits a query into uppercased, punctuation-trimmed tokens. It splits
// on any whitespace, so a tab or newline separator cannot hide a keyword the way a
// literal-space prefix check would.
func sqlTokens(query string) []string {
	fields := strings.Fields(strings.ToUpper(query))
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.Trim(f, "(),;"); f != "" {
			tokens = append(tokens, f)
		}
	}
	return tokens
}

// containsSQLKeyword reports whether any whole-word token equals one of the given
// keywords. Whole-word matching avoids false positives on identifiers that merely
// contain a keyword as a substring (e.g. "backdrop" containing "DROP").
func containsSQLKeyword(query string, keywords ...string) bool {
	set := make(map[string]bool, len(keywords))
	for _, k := range keywords {
		set[k] = true
	}
	for _, tok := range sqlTokens(query) {
		if set[tok] {
			return true
		}
	}
	return false
}

// isSafeReadOnly reports whether a query cannot modify data. WITH is safe only when
// it contains no data-modifying statement, because Postgres data-modifying CTEs
// (WITH ... AS (DELETE ...)) execute writes. EXPLAIN is safe unless it uses ANALYZE
// over a data-modifying statement, because EXPLAIN ANALYZE executes the statement.
func isSafeReadOnly(query string) bool {
	switch DetectStatementType(query) {
	case StatementSelect, StatementShow, StatementDescribe:
		return true
	case StatementWith:
		return !containsSQLKeyword(query, dataModifyingKeywords...)
	case StatementExplain:
		if containsSQLKeyword(query, "ANALYZE") {
			return !containsSQLKeyword(query, dataModifyingKeywords...)
		}
		return true // plain EXPLAIN does not execute the statement
	}
	return false
}

// DetectStatementType returns the type of SQL statement based on its first
// significant token. Tokenizing (rather than prefix matching) makes detection
// robust to tab/newline separators between the verb and the rest of the statement.
func DetectStatementType(query string) StatementType {
	tokens := sqlTokens(query)
	if len(tokens) == 0 {
		return StatementUnknown
	}
	switch tokens[0] {
	case "SELECT":
		return StatementSelect
	case "INSERT":
		return StatementInsert
	case "UPDATE":
		return StatementUpdate
	case "DELETE":
		return StatementDelete
	case "DROP":
		return StatementDrop
	case "CREATE":
		return StatementCreate
	case "ALTER":
		return StatementAlter
	case "TRUNCATE":
		return StatementTruncate
	case "SHOW":
		return StatementShow
	case "DESCRIBE", "DESC":
		return StatementDescribe
	case "EXPLAIN":
		return StatementExplain
	case "WITH":
		return StatementWith
	default:
		return StatementUnknown
	}
}

// ValidateSQLStatement validates a SQL statement against security rules.
// Parameters:
//   - allowWrite: permits INSERT/UPDATE/DELETE/CREATE/ALTER
//   - allowDestructive: permits DROP/TRUNCATE (requires explicit opt-in via --allow-drop or --confirm-writes)
func ValidateSQLStatement(query string, allowWrite bool, securityLevel SecurityLevel, allowMultiStatement bool, allowDestructive bool) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return errors.New("empty query")
	}

	// block DROP/TRUNCATE unless explicitly allowed
	if !allowDestructive {
		if containsSQLKeyword(query, "DROP") {
			return fmt.Errorf("%w: DROP detected (use --confirm-writes or --allow-drop to enable)", ErrDestructiveOperation)
		}
		if containsSQLKeyword(query, "TRUNCATE") {
			return fmt.Errorf("%w: TRUNCATE detected (use --confirm-writes or --allow-drop to enable)", ErrDestructiveOperation)
		}
	}

	// Block SQL comments
	if strings.Contains(query, "--") || strings.Contains(query, "/*") {
		return errors.New("SQL comments are not allowed")
	}

	// Block multiple statements: no semicolons except optional trailing one
	if !allowMultiStatement {
		trimmed := strings.TrimSuffix(strings.TrimSpace(query), ";")
		if strings.Contains(trimmed, ";") {
			return ErrMultipleStatements
		}
	}

	// Block dangerous functions in strict mode
	if securityLevel == SecurityLevelStrict {
		upper := strings.ToUpper(query)
		for _, fn := range dangerousFunctions {
			if strings.Contains(upper, fn) {
				return fmt.Errorf("%w: %s", ErrDangerousFunction, fn)
			}
		}
	}

	stmtType := DetectStatementType(query)

	// Allow writes - only apply minimal restrictions
	if allowWrite {
		if securityLevel == SecurityLevelMinimal {
			// DROP/TRUNCATE already handled by allowDestructive check above
			// Only block DELETE without WHERE in minimal mode
			if stmtType == StatementDelete && !strings.Contains(strings.ToUpper(query), " WHERE ") {
				return fmt.Errorf("%w: DELETE without WHERE clause", ErrDestructiveOperation)
			}
		}
		return nil
	}

	// Read-only mode: only allow statements that cannot modify data. This rejects
	// writable CTEs, EXPLAIN ANALYZE over writes, and unrecognized statement types.
	if isSafeReadOnly(query) {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrWriteNotAllowed, stmtType)
}

// IsReadOnlyStatement returns true if the statement type is read-only.
func IsReadOnlyStatement(stmtType StatementType) bool {
	switch stmtType {
	case StatementSelect, StatementShow, StatementDescribe, StatementExplain, StatementWith:
		return true
	}
	return false
}

// IsWriteStatement returns true if the statement type modifies data.
func IsWriteStatement(stmtType StatementType) bool {
	switch stmtType {
	case StatementInsert, StatementUpdate, StatementDelete, StatementDrop,
		StatementCreate, StatementAlter, StatementTruncate:
		return true
	}
	return false
}

// Input validation errors - designed to be helpful for AI assistants
var (
	ErrQueryRequired      = errors.New("query is required and cannot be empty")
	ErrTableRequired      = errors.New("table is required - specify which table to describe")
	ErrTokenRequired      = errors.New("token is required - use the confirmation_token from the previous query response")
	ErrTokenInvalid       = errors.New("token is not a valid confirmation token format")
	ErrConnectionRequired = errors.New("connection is required when multiple connections are configured (use whodb_connections to list available connections)")
)

// ValidateQueryInput validates the input for whodb_query tool.
func ValidateQueryInput(input *QueryInput, connectionCount int) error {
	if strings.TrimSpace(input.Query) == "" {
		return ErrQueryRequired
	}
	if connectionCount > 1 && strings.TrimSpace(input.Connection) == "" {
		return ErrConnectionRequired
	}
	return nil
}

// ValidateSchemasInput validates the input for whodb_schemas tool.
func ValidateSchemasInput(input *SchemasInput, connectionCount int) error {
	if connectionCount > 1 && strings.TrimSpace(input.Connection) == "" {
		return ErrConnectionRequired
	}
	return nil
}

// ValidateTablesInput validates the input for whodb_tables tool.
func ValidateTablesInput(input *TablesInput, connectionCount int) error {
	if connectionCount > 1 && strings.TrimSpace(input.Connection) == "" {
		return ErrConnectionRequired
	}
	return nil
}

// ValidateColumnsInput validates the input for whodb_columns tool.
func ValidateColumnsInput(input *ColumnsInput, connectionCount int) error {
	if strings.TrimSpace(input.Table) == "" {
		return ErrTableRequired
	}
	if connectionCount > 1 && strings.TrimSpace(input.Connection) == "" {
		return ErrConnectionRequired
	}
	return nil
}

// ValidateConfirmInput validates the input for whodb_confirm tool.
func ValidateConfirmInput(input *ConfirmInput) error {
	token := strings.TrimSpace(input.Token)
	if token == "" {
		return ErrTokenRequired
	}
	// Basic UUID format check (8-4-4-4-12 pattern)
	if len(token) != 36 || token[8] != '-' || token[13] != '-' || token[18] != '-' || token[23] != '-' {
		return ErrTokenInvalid
	}
	return nil
}
