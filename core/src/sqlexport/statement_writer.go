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

package sqlexport

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/engine"
)

// DefaultInsertBatchSize is the backend-controlled number of rows per INSERT statement.
const DefaultInsertBatchSize = 500

var numericLiteralPattern = regexp.MustCompile(`^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?$`)

// Table identifies the SQL Table being exported.
type Table struct {
	Schema string
	Name   string
}

// Row stores raw row values keyed by column name.
type Row map[string]any

// Dialect provides the SQL syntax hooks needed by the statement writer.
type Dialect interface {
	QuoteIdentifier(identifier string) string
	FormatLiteral(value any, column engine.Column) (string, error)
}

// GenericDialect implements conservative SQL literal formatting for common SQL dialects.
type GenericDialect struct {
	QuoteIdentifierFunc func(identifier string) string
}

// QuoteIdentifier quotes an identifier using the configured callback.
func (d GenericDialect) QuoteIdentifier(identifier string) string {
	if d.QuoteIdentifierFunc == nil {
		return identifier
	}
	return d.QuoteIdentifierFunc(identifier)
}

// FormatLiteral converts a raw database value into a SQL literal for one column.
func (d GenericDialect) FormatLiteral(value any, column engine.Column) (string, error) {
	return FormatLiteral(value, column)
}

// FormatLiteral converts a raw database value into a conservative SQL literal.
func FormatLiteral(value any, column engine.Column) (string, error) {
	if value == nil {
		return "NULL", nil
	}

	if valuer, ok := value.(driver.Valuer); ok {
		driverValue, err := valuer.Value()
		if err != nil {
			return "", err
		}
		return FormatLiteral(driverValue, column)
	}

	switch v := value.(type) {
	case []byte:
		return formatBytesLiteral(v, column)
	case string:
		return formatTextLiteral(v, column)
	case bool:
		if v {
			return "TRUE", nil
		}
		return "FALSE", nil
	case time.Time:
		return quoteString(formatTime(v)), nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return formatFloat(float64(v), 32)
	case float64:
		return formatFloat(v, 64)
	case fmt.Stringer:
		return formatTextLiteral(v.String(), column)
	default:
		return quoteString(fmt.Sprintf("%v", v)), nil
	}
}

// WriteInsert writes one multi-row INSERT statement for the supplied rows.
func WriteInsert(w io.Writer, dialect Dialect, table Table, columns []engine.Column, rows []Row) error {
	if len(rows) == 0 {
		return nil
	}
	if len(columns) == 0 {
		return fmt.Errorf("cannot export INSERT without writable columns")
	}

	if _, err := fmt.Fprintf(w, "INSERT INTO %s (%s) VALUES\n", quoteTable(dialect, table), quoteColumns(dialect, columns)); err != nil {
		return err
	}

	for rowIndex, row := range rows {
		literals := make([]string, len(columns))
		for columnIndex, column := range columns {
			literal, err := dialect.FormatLiteral(row[column.Name], column)
			if err != nil {
				return fmt.Errorf("format column %s: %w", column.Name, err)
			}
			literals[columnIndex] = literal
		}

		suffix := ","
		if rowIndex == len(rows)-1 {
			suffix = ";"
		}
		if _, err := fmt.Fprintf(w, "  (%s)%s\n", strings.Join(literals, ", "), suffix); err != nil {
			return err
		}
	}

	return nil
}

// WriteUpdate writes one UPDATE statement for the supplied row.
func WriteUpdate(w io.Writer, dialect Dialect, table Table, setColumns []engine.Column, primaryKeyColumns []engine.Column, row Row) error {
	if len(primaryKeyColumns) == 0 {
		return fmt.Errorf("cannot export UPDATE without primary key columns")
	}
	if len(setColumns) == 0 {
		return fmt.Errorf("cannot export UPDATE without writable non-primary columns")
	}

	assignments := make([]string, len(setColumns))
	for i, column := range setColumns {
		literal, err := dialect.FormatLiteral(row[column.Name], column)
		if err != nil {
			return fmt.Errorf("format SET column %s: %w", column.Name, err)
		}
		assignments[i] = fmt.Sprintf("%s = %s", dialect.QuoteIdentifier(column.Name), literal)
	}

	predicates := make([]string, len(primaryKeyColumns))
	for i, column := range primaryKeyColumns {
		literal, err := dialect.FormatLiteral(row[column.Name], column)
		if err != nil {
			return fmt.Errorf("format primary key column %s: %w", column.Name, err)
		}
		if literal == "NULL" {
			return fmt.Errorf("primary key column %s is NULL", column.Name)
		}
		predicates[i] = fmt.Sprintf("%s = %s", dialect.QuoteIdentifier(column.Name), literal)
	}

	_, err := fmt.Fprintf(
		w,
		"UPDATE %s SET %s WHERE %s;\n",
		quoteTable(dialect, table),
		strings.Join(assignments, ", "),
		strings.Join(predicates, " AND "),
	)
	return err
}

func quoteTable(dialect Dialect, table Table) string {
	if table.Schema == "" {
		return dialect.QuoteIdentifier(table.Name)
	}
	return dialect.QuoteIdentifier(table.Schema) + "." + dialect.QuoteIdentifier(table.Name)
}

func quoteColumns(dialect Dialect, columns []engine.Column) string {
	quoted := make([]string, len(columns))
	for i, column := range columns {
		quoted[i] = dialect.QuoteIdentifier(column.Name)
	}
	return strings.Join(quoted, ", ")
}

func quoteString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func formatFloat(value float64, bitSize int) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", fmt.Errorf("non-finite float values cannot be exported as SQL literals")
	}
	return strconv.FormatFloat(value, 'g', -1, bitSize), nil
}

func formatTime(value time.Time) string {
	if value.Nanosecond() == 0 {
		return value.Format("2006-01-02 15:04:05")
	}
	return value.Format("2006-01-02 15:04:05.999999999")
}

func formatTextLiteral(value string, column engine.Column) (string, error) {
	trimmed := strings.TrimSpace(value)
	if isNumericColumn(column.Type) && numericLiteralPattern.MatchString(trimmed) {
		return trimmed, nil
	}
	if isBooleanColumn(column.Type) {
		switch strings.ToLower(trimmed) {
		case "true", "t", "1":
			return "TRUE", nil
		case "false", "f", "0":
			return "FALSE", nil
		}
	}
	return quoteString(value), nil
}

func formatBytesLiteral(value []byte, column engine.Column) (string, error) {
	if !isBinaryColumn(column.Type) {
		return formatTextLiteral(string(value), column)
	}

	encoded := strings.ToUpper(hex.EncodeToString(value))
	if strings.Contains(strings.ToLower(column.Type), "bytea") {
		return quoteString("\\x" + encoded), nil
	}
	return "X'" + encoded + "'", nil
}

func isBinaryColumn(columnType string) bool {
	normalized := strings.ToLower(columnType)
	return strings.Contains(normalized, "binary") ||
		strings.Contains(normalized, "blob") ||
		strings.Contains(normalized, "bytea")
}

func isBooleanColumn(columnType string) bool {
	normalized := strings.ToLower(columnType)
	return strings.Contains(normalized, "bool")
}

func isNumericColumn(columnType string) bool {
	normalized := strings.ToLower(columnType)
	return strings.Contains(normalized, "int") ||
		strings.Contains(normalized, "serial") ||
		strings.Contains(normalized, "decimal") ||
		strings.Contains(normalized, "numeric") ||
		strings.Contains(normalized, "number") ||
		strings.Contains(normalized, "float") ||
		strings.Contains(normalized, "double") ||
		strings.Contains(normalized, "real")
}
