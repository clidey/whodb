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

	"github.com/clidey/whodb/core/src/sqlguard"
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

// StatementType re-exports the shared classifier's statement type so existing
// MCP callers and analytics payloads keep the same type.
type StatementType = sqlguard.StatementType

// Statement type constants re-exported from the shared classifier.
const (
	StatementSelect   = sqlguard.StatementSelect
	StatementInsert   = sqlguard.StatementInsert
	StatementUpdate   = sqlguard.StatementUpdate
	StatementDelete   = sqlguard.StatementDelete
	StatementDrop     = sqlguard.StatementDrop
	StatementCreate   = sqlguard.StatementCreate
	StatementAlter    = sqlguard.StatementAlter
	StatementTruncate = sqlguard.StatementTruncate
	StatementShow     = sqlguard.StatementShow
	StatementDescribe = sqlguard.StatementDescribe
	StatementExplain  = sqlguard.StatementExplain
	StatementWith     = sqlguard.StatementWith
	StatementUnknown  = sqlguard.StatementUnknown
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

// DetectStatementType returns the type of SQL statement based on its first
// significant token.
func DetectStatementType(query string) StatementType {
	return sqlguard.Classify(query).Type
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
		if sqlguard.ContainsKeyword(query, "DROP") {
			return fmt.Errorf("%w: DROP detected (use --confirm-writes or --allow-drop to enable)", ErrDestructiveOperation)
		}
		if sqlguard.ContainsKeyword(query, "TRUNCATE") {
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

	cls := sqlguard.Classify(query)
	stmtType := cls.Type

	// Allow writes - only apply minimal restrictions
	if allowWrite {
		if securityLevel == SecurityLevelMinimal {
			// DROP/TRUNCATE already handled by allowDestructive check above
			// Only block DELETE without WHERE in minimal mode
			if stmtType == StatementDelete && !sqlguard.ContainsKeyword(query, "WHERE") {
				return fmt.Errorf("%w: DELETE without WHERE clause", ErrDestructiveOperation)
			}
		}
		return nil
	}

	// Read-only mode: only allow statements that cannot modify data. This rejects
	// writable CTEs, EXPLAIN ANALYZE over writes, and unrecognized statement types.
	if !cls.Mutating {
		return nil
	}
	return fmt.Errorf("%w: %s (%s)", ErrWriteNotAllowed, stmtType, cls.Reason)
}

// IsReadOnlyStatement returns true if the statement type is read-only.
func IsReadOnlyStatement(stmtType StatementType) bool {
	return sqlguard.IsReadOnlyStatement(stmtType)
}

// IsWriteStatement returns true if the statement type modifies data.
func IsWriteStatement(stmtType StatementType) bool {
	return sqlguard.IsWriteStatement(stmtType)
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
