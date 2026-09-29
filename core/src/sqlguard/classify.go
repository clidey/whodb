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

// Package sqlguard classifies queries for read-only and approval gates. SQL
// outside the supported read grammar is reported as mutating, including syntax
// whose effects cannot be established. Classification must be combined with
// connector enforcement and trusted database schema objects.
package sqlguard

import (
	"strings"
)

// StatementType is the kind of statement, derived from its leading keyword.
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
	StatementMerge    StatementType = "MERGE"
	StatementReplace  StatementType = "REPLACE"
	StatementUpsert   StatementType = "UPSERT"
	StatementGrant    StatementType = "GRANT"
	StatementRevoke   StatementType = "REVOKE"
	StatementSet      StatementType = "SET"
	StatementCopy     StatementType = "COPY"
	StatementCall     StatementType = "CALL"
	StatementDo       StatementType = "DO"
	StatementLock     StatementType = "LOCK"
	StatementVacuum   StatementType = "VACUUM"
	StatementAnalyze  StatementType = "ANALYZE"
	StatementPragma   StatementType = "PRAGMA"
	StatementUse      StatementType = "USE"
	StatementCommand  StatementType = "COMMAND"
	StatementUnknown  StatementType = "UNKNOWN"
)

// Classification is the verdict for one statement or command.
type Classification struct {
	// Type is the statement kind, or StatementUnknown when unrecognized.
	Type StatementType
	// Mutating reports whether the statement requires approval because it may
	// change state or its effects are unknown. It does not prove a write occurs.
	Mutating bool
	// MultiStatement reports whether more than one statement was submitted. It is
	// tracked separately from Mutating because execution paths differ: a single
	// write still runs as a normal query, while a batch needs script execution.
	MultiStatement bool
	// Reason explains the verdict in one clause, for audit and error messages.
	Reason string
}

// mutatingKeywords reserves write and administrative words in the read grammar.
// This is not the security boundary: the complete grammar must also match.
var mutatingKeywords = map[string]bool{
	"INSERT": true, "UPDATE": true, "DELETE": true, "MERGE": true,
	"DROP": true, "TRUNCATE": true, "ALTER": true, "CREATE": true,
	"GRANT": true, "REVOKE": true, "REPLACE": true, "UPSERT": true,
	"CALL": true, "EXEC": true, "EXECUTE": true, "DO": true,
	"COPY": true, "LOCK": true, "VACUUM": true, "REINDEX": true,
	"RENAME": true, "COMMENT": true, "REFRESH": true, "IMPORT": true,
	"LOAD": true, "UNLOAD": true, "ATTACH": true, "DETACH": true,
	"OPTIMIZE": true, "CLUSTER": true, "PREPARE": true, "DEALLOCATE": true,
	"START": true, "COMMIT": true, "ROLLBACK": true, "SAVEPOINT": true,
	"KILL": true, "SHUTDOWN": true, "RESET": true,
}

// readStatementTypes identifies read-shaped prefixes for presentation only.
// Classify must inspect the complete query before execution.
var readStatementTypes = map[StatementType]bool{
	StatementSelect:   true,
	StatementShow:     true,
	StatementDescribe: true,
	StatementExplain:  true,
	StatementWith:     true,
}

// leadingKeywords maps a statement's first token to its type.
var leadingKeywords = map[string]StatementType{
	"SELECT": StatementSelect, "TABLE": StatementSelect, "VALUES": StatementSelect,
	"INSERT": StatementInsert, "UPDATE": StatementUpdate, "DELETE": StatementDelete,
	"DROP": StatementDrop, "CREATE": StatementCreate, "ALTER": StatementAlter,
	"TRUNCATE": StatementTruncate, "SHOW": StatementShow,
	"DESCRIBE": StatementDescribe, "DESC": StatementDescribe,
	"EXPLAIN": StatementExplain, "WITH": StatementWith,
	"MERGE": StatementMerge, "REPLACE": StatementReplace, "UPSERT": StatementUpsert,
	"GRANT": StatementGrant, "REVOKE": StatementRevoke, "SET": StatementSet,
	"COPY": StatementCopy, "CALL": StatementCall, "EXEC": StatementCall,
	"EXECUTE": StatementCall, "DO": StatementDo, "LOCK": StatementLock,
	"VACUUM": StatementVacuum, "ANALYZE": StatementAnalyze,
	"PRAGMA": StatementPragma, "USE": StatementUse,
}

// Classify reports whether SQL requires approval. Only the bounded read grammar
// is automatic; unsupported syntax and ambiguous dialect semantics fail closed.
func Classify(query string) Classification {
	stripped := stripNoise(query)
	tokens := tokenize(stripped)
	cls := Classification{Type: StatementUnknown, MultiStatement: isMultiStatement(stripped)}
	if len(tokens) > 0 {
		cls.Type = statementType(tokens)
	}
	if !understoodRead(query) {
		cls.Mutating = true
		cls.Reason = "statement is not a supported read-only query"
	}
	return cls
}

// IsReadOnly reports whether SQL fits the supported automatic-read grammar.
func IsReadOnly(query string) bool {
	return !Classify(query).Mutating
}

// OperationName returns the lowercase verb for a statement, for event payloads
// and telemetry. Reads report "get" so callers can key display off one value.
func OperationName(cls Classification) string {
	if !cls.Mutating {
		return "get"
	}
	if cls.Type == StatementUnknown {
		return "execute"
	}
	return strings.ToLower(string(cls.Type))
}

// IsReadOnlyStatement reports whether a statement type is read-only on its own.
// Prefer Classify, which also inspects the rest of the statement.
func IsReadOnlyStatement(stmtType StatementType) bool {
	return readStatementTypes[stmtType]
}

// IsWriteStatement reports whether a statement type modifies data or schema.
func IsWriteStatement(stmtType StatementType) bool {
	switch stmtType {
	case StatementInsert, StatementUpdate, StatementDelete, StatementDrop,
		StatementCreate, StatementAlter, StatementTruncate, StatementMerge,
		StatementReplace, StatementUpsert, StatementCopy, StatementCall,
		StatementDo, StatementGrant, StatementRevoke, StatementLock,
		StatementVacuum:
		return true
	}
	return false
}

// statementType returns the type for the statement's first significant token.
func statementType(tokens []string) StatementType {
	if stmtType, ok := leadingKeywords[tokens[0]]; ok {
		return stmtType
	}
	return StatementUnknown
}

// ContainsKeyword reports whether a query contains any of the given keywords as
// whole words, ignoring comments and string literals.
func ContainsKeyword(query string, keywords ...string) bool {
	set := make(map[string]bool, len(keywords))
	for _, k := range keywords {
		set[strings.ToUpper(k)] = true
	}
	for _, tok := range tokenize(stripNoise(query)) {
		if set[tok] {
			return true
		}
	}
	return false
}

// isMultiStatement reports whether more than one statement was submitted. It
// runs on the noise-stripped query, so a semicolon inside a string literal, a
// comment, or a dollar-quoted block does not count.
func isMultiStatement(stripped string) bool {
	trimmed := strings.TrimSpace(stripped)
	for strings.HasSuffix(trimmed, ";") {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	}
	return strings.Contains(trimmed, ";")
}

// tokenize returns uppercased SQL words, treating punctuation as separators.
// Splitting on punctuation as well as whitespace prevents constructs such as
// AS(DELETE or EXPLAIN (ANALYZE,BUFFERS) from hiding security-significant words.
func tokenize(query string) []string {
	return strings.FieldsFunc(strings.ToUpper(query), func(r rune) bool {
		return (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_'
	})
}

// stripNoise replaces comments, string literals, quoted identifiers, and
// dollar-quoted bodies with whitespace. Keyword scanning then sees only the
// statement's own structure: a keyword inside a comment or literal cannot
// trigger a verdict, and a keyword hidden behind one cannot escape it.
func stripNoise(query string) string {
	var b strings.Builder
	b.Grow(len(query))
	for i := 0; i < len(query); {
		switch c := query[i]; {
		case c == '-' && i+1 < len(query) && query[i+1] == '-',
			c == '#':
			for i < len(query) && query[i] != '\n' {
				i++
			}
			b.WriteByte(' ')
		case c == '/' && i+1 < len(query) && query[i+1] == '*':
			i += 2
			for i+1 < len(query) && (query[i] != '*' || query[i+1] != '/') {
				i++
			}
			if i+1 < len(query) {
				i += 2
			} else {
				i = len(query)
			}
			b.WriteByte(' ')
		case c == '\'' || c == '"' || c == '`':
			i = skipQuoted(query, i, c)
			b.WriteString(" ? ")
		case c == '$':
			if tagEnd := dollarTagEnd(query, i); tagEnd > i {
				tag := query[i:tagEnd]
				rest := query[tagEnd:]
				if idx := strings.Index(rest, tag); idx >= 0 {
					i = tagEnd + idx + len(tag)
				} else {
					i = len(query)
				}
				b.WriteString(" ? ")
				continue
			}
			b.WriteByte(c)
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// skipQuoted returns the index just past a quoted run that starts at i,
// honoring backslash escapes and doubled quote characters.
func skipQuoted(query string, i int, quote byte) int {
	for i++; i < len(query); i++ {
		if query[i] == '\\' && quote == '\'' {
			i++
			continue
		}
		if query[i] == quote {
			if i+1 < len(query) && query[i+1] == quote {
				i++
				continue
			}
			return i + 1
		}
	}
	return len(query)
}

// dollarTagEnd returns the index just past a `$tag$` opener at i, or i when the
// text is an ordinary `$` (a placeholder such as `$1`).
func dollarTagEnd(query string, i int) int {
	for j := i + 1; j < len(query); j++ {
		c := query[j]
		switch {
		case c == '$':
			return j + 1
		case c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9' && j > i+1):
			continue
		default:
			return i
		}
	}
	return i
}
