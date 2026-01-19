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

// DetectStatementType returns the type of SQL statement.
func DetectStatementType(query string) StatementType {
	q := strings.ToUpper(strings.TrimSpace(query))

	keywords := []struct {
		kw  string
		typ StatementType
	}{
		{"SELECT ", StatementSelect}, {"INSERT ", StatementInsert},
		{"UPDATE ", StatementUpdate}, {"DELETE ", StatementDelete},
		{"DROP ", StatementDrop}, {"CREATE ", StatementCreate},
		{"ALTER ", StatementAlter}, {"TRUNCATE ", StatementTruncate},
		{"SHOW ", StatementShow}, {"DESCRIBE ", StatementDescribe},
		{"DESC ", StatementDescribe}, {"EXPLAIN ", StatementExplain},
		{"WITH ", StatementWith},
	}

	for _, k := range keywords {
		if strings.HasPrefix(q, k.kw) {
			return k.typ
		}
	}
	return StatementUnknown
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
		upper := strings.ToUpper(query)
		if strings.Contains(upper, "DROP ") {
			return fmt.Errorf("%w: DROP detected (use --confirm-writes or --allow-drop to enable)", ErrDestructiveOperation)
		}
		if strings.Contains(upper, "TRUNCATE ") {
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

	// Read-only mode: only allow safe statements
	switch stmtType {
	case StatementSelect, StatementShow, StatementDescribe, StatementExplain, StatementWith:
		return nil
	case StatementInsert, StatementUpdate, StatementDelete, StatementDrop,
		StatementCreate, StatementAlter, StatementTruncate:
		return fmt.Errorf("%w: %s", ErrWriteNotAllowed, stmtType)
	default:
		if securityLevel != SecurityLevelMinimal {
			return fmt.Errorf("%w: unknown statement type", ErrWriteNotAllowed)
		}
		return nil
	}
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
