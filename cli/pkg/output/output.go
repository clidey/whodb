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

package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/clidey/whodb/cli/pkg/styles"
	"golang.org/x/term"
)

type Format string

const (
	FormatAuto   Format = "auto"
	FormatTable  Format = "table"
	FormatPlain  Format = "plain"
	FormatJSON   Format = "json"
	FormatNDJSON Format = "ndjson"
	FormatCSV    Format = "csv"
)

type QueryResult struct {
	Columns []Column `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

// StringQueryResult represents query results that are already materialized as
// strings and can be emitted without converting through [][]any.
type StringQueryResult struct {
	Columns []Column   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

// QueryStream writes a query result incrementally after the columns are known.
type QueryStream interface {
	WriteRow(row []string) error
	Close() error
}

type Column struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type Writer struct {
	out    io.Writer
	err    io.Writer
	format Format
	isTTY  bool
	quiet  bool
}

type Option func(*Writer)

func WithFormat(f Format) Option {
	return func(w *Writer) {
		w.format = f
	}
}

func WithQuiet(quiet bool) Option {
	return func(w *Writer) {
		w.quiet = quiet
	}
}

func WithOutput(out io.Writer) Option {
	return func(w *Writer) {
		w.out = out
	}
}

func WithErrorOutput(err io.Writer) Option {
	return func(w *Writer) {
		w.err = err
	}
}

func New(opts ...Option) *Writer {
	w := &Writer{
		out:    os.Stdout,
		err:    os.Stderr,
		format: FormatAuto,
	}

	for _, opt := range opts {
		opt(w)
	}

	w.isTTY = w.detectTTY()

	return w
}

func (w *Writer) detectTTY() bool {
	if f, ok := w.out.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func (w *Writer) IsTTY() bool {
	return w.isTTY
}

func (w *Writer) ColorEnabled() bool {
	return styles.ColorEnabled() && w.isTTY
}

func (w *Writer) resolveFormat() Format {
	if w.format != FormatAuto {
		return w.format
	}

	if w.isTTY {
		return FormatTable
	}
	return FormatPlain
}

func (w *Writer) WriteQueryResult(result *QueryResult) error {
	format := w.resolveFormat()

	switch format {
	case FormatJSON:
		return w.writeJSON(result)
	case FormatNDJSON:
		return w.writeNDJSON(result)
	case FormatCSV:
		return w.writeCSV(result)
	case FormatPlain:
		return w.writePlain(result)
	case FormatTable:
		return w.writeTable(result)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

// WriteStringQueryResult writes a result set whose cells are already strings.
func (w *Writer) WriteStringQueryResult(result *StringQueryResult) error {
	format := w.resolveFormat()

	switch format {
	case FormatJSON:
		return w.writeStringJSON(result)
	case FormatNDJSON:
		return w.writeStringNDJSON(result)
	case FormatCSV:
		return w.writeStringCSV(result)
	case FormatPlain:
		return w.writeStringPlain(result)
	case FormatTable:
		return w.writeStringTable(result)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

// BeginQueryStream starts a streaming query result writer for the configured
// output format.
func (w *Writer) BeginQueryStream(columns []Column) (QueryStream, error) {
	format := w.resolveFormat()

	switch format {
	case FormatJSON:
		return w.beginJSONStream(columns)
	case FormatNDJSON:
		return w.beginNDJSONStream(columns), nil
	case FormatCSV:
		return w.beginCSVStream(columns)
	case FormatPlain:
		return w.beginPlainStream(columns)
	case FormatTable:
		return nil, fmt.Errorf("streaming output is not supported for table format")
	default:
		return nil, fmt.Errorf("unknown output format: %s", format)
	}
}

func (w *Writer) writeJSON(result *QueryResult) error {
	return w.writeJSONRows(len(result.Rows), func(i int) map[string]any {
		return result.recordForRow(result.Rows[i])
	})
}

func (w *Writer) writeStringJSON(result *StringQueryResult) error {
	return w.writeJSONRows(len(result.Rows), func(i int) map[string]any {
		return recordForStringRow(result.Columns, result.Rows[i])
	})
}

func (w *Writer) writeNDJSON(result *QueryResult) error {
	return w.writeNDJSONRows(len(result.Rows), func(i int) map[string]any {
		return result.recordForRow(result.Rows[i])
	})
}

func (w *Writer) writeStringNDJSON(result *StringQueryResult) error {
	return w.writeNDJSONRows(len(result.Rows), func(i int) map[string]any {
		return recordForStringRow(result.Columns, result.Rows[i])
	})
}

func (r *QueryResult) recordForRow(row []any) map[string]any {
	return recordForAnyRow(r.Columns, row)
}

func recordForAnyRow(columns []Column, row []any) map[string]any {
	record := make(map[string]any, len(columns))
	for i, col := range columns {
		if i < len(row) {
			record[col.Name] = row[i]
		}
	}
	return record
}

func recordForStringRow(columns []Column, row []string) map[string]any {
	record := make(map[string]any, len(columns))
	for i, col := range columns {
		if i < len(row) {
			record[col.Name] = typedValueForStringColumn(col, row[i])
		}
	}
	return record
}

func typedValueForStringColumn(column Column, value string) any {
	switch strings.ToLower(strings.TrimSpace(column.Type)) {
	case "bool", "boolean":
		if parsed, err := strconv.ParseBool(strings.TrimSpace(value)); err == nil {
			return parsed
		}
	}
	return value
}

func (w *Writer) writeCSV(result *QueryResult) error {
	return w.writeCSVRows(result.Columns, len(result.Rows), func(i int) []string {
		return stringValuesForAnyRow(result.Rows[i])
	})
}

func (w *Writer) writeStringCSV(result *StringQueryResult) error {
	return w.writeCSVRows(result.Columns, len(result.Rows), func(i int) []string {
		return result.Rows[i]
	})
}

func (w *Writer) writePlain(result *QueryResult) error {
	return w.writePlainRows(result.Columns, len(result.Rows), func(i int) []string {
		return stringValuesForAnyRow(result.Rows[i])
	})
}

func (w *Writer) writeStringPlain(result *StringQueryResult) error {
	return w.writePlainRows(result.Columns, len(result.Rows), func(i int) []string {
		return result.Rows[i]
	})
}

func (w *Writer) writeTable(result *QueryResult) error {
	return w.writeTableRows(result.Columns, len(result.Rows), func(i int) []string {
		return stringValuesForAnyRow(result.Rows[i])
	})
}

func (w *Writer) writeStringTable(result *StringQueryResult) error {
	return w.writeTableRows(result.Columns, len(result.Rows), func(i int) []string {
		return result.Rows[i]
	})
}

func (w *Writer) writeJSONRows(rowCount int, recordForIndex func(int) map[string]any) error {
	output := make([]map[string]any, 0, rowCount)
	for i := 0; i < rowCount; i++ {
		output = append(output, recordForIndex(i))
	}

	encoder := json.NewEncoder(w.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (w *Writer) writeNDJSONRows(rowCount int, recordForIndex func(int) map[string]any) error {
	encoder := json.NewEncoder(w.out)
	for i := 0; i < rowCount; i++ {
		if err := encoder.Encode(recordForIndex(i)); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) writeCSVRows(columns []Column, rowCount int, rowValues func(int) []string) error {
	csvWriter := csv.NewWriter(w.out)
	defer csvWriter.Flush()

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
	}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("writing CSV headers: %w", err)
	}

	for i := 0; i < rowCount; i++ {
		if err := csvWriter.Write(rowValues(i)); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	return csvWriter.Error()
}

func (w *Writer) writePlainRows(columns []Column, rowCount int, rowValues func(int) []string) error {
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
	}
	if _, err := fmt.Fprintln(w.out, strings.Join(headers, "\t")); err != nil {
		return err
	}

	for i := 0; i < rowCount; i++ {
		if _, err := fmt.Fprintln(w.out, strings.Join(rowValues(i), "\t")); err != nil {
			return err
		}
	}

	return nil
}

func (w *Writer) writeTableRows(columns []Column, rowCount int, rowValues func(int) []string) error {
	if len(columns) == 0 {
		return nil
	}

	widths := make([]int, len(columns))
	rows := make([][]string, rowCount)
	headers := make([]string, len(columns))

	for i, col := range columns {
		headers[i] = col.Name
		widths[i] = max(3, lipgloss.Width(col.Name))
	}

	for i := 0; i < rowCount; i++ {
		row := rowValues(i)
		renderedRow := make([]string, len(columns))
		for j := range columns {
			if j < len(row) {
				renderedRow[j] = row[j]
			}
			widths[j] = max(widths[j], lipgloss.Width(renderedRow[j]))
		}
		rows[i] = renderedRow
	}

	if _, err := fmt.Fprintln(w.out, w.renderTableRow(headers, widths, true)); err != nil {
		return err
	}

	separators := make([]string, len(columns))
	for i, width := range widths {
		separators[i] = strings.Repeat("─", width)
	}
	if _, err := fmt.Fprintln(w.out, w.renderTableRow(separators, widths, false)); err != nil {
		return err
	}

	for _, row := range rows {
		if _, err := fmt.Fprintln(w.out, w.renderTableRow(row, widths, false)); err != nil {
			return err
		}
	}

	return nil
}

func stringValuesForAnyRow(row []any) []string {
	record := make([]string, len(row))
	for i, cell := range row {
		record[i] = fmt.Sprintf("%v", cell)
	}
	return record
}

func (w *Writer) renderTableRow(cells []string, widths []int, header bool) string {
	var b strings.Builder

	for i, cell := range cells {
		rendered := padTableCell(cell, widths[i])
		if header && w.ColorEnabled() {
			rendered = "\033[1m" + rendered + "\033[0m"
		}

		b.WriteString(rendered)
		if i < len(cells)-1 {
			b.WriteString("  ")
		}
	}

	return b.String()
}

func padTableCell(text string, width int) string {
	padding := width - lipgloss.Width(text)
	if padding <= 0 {
		return text
	}

	return text + strings.Repeat(" ", padding)
}

func (w *Writer) Info(format string, args ...any) {
	if w.quiet {
		return
	}
	fmt.Fprintf(w.err, format+"\n", args...)
}

func (w *Writer) Error(format string, args ...any) {
	fmt.Fprintf(w.err, "Error: "+format+"\n", args...)
}

func (w *Writer) Success(format string, args ...any) {
	if w.quiet {
		return
	}
	prefix := "✓ "
	if w.ColorEnabled() {
		prefix = "\033[32m✓\033[0m "
	}
	fmt.Fprintf(w.err, prefix+format+"\n", args...)
}

type plainQueryStream struct {
	out io.Writer
}

func (s *plainQueryStream) WriteRow(row []string) error {
	_, err := fmt.Fprintln(s.out, strings.Join(row, "\t"))
	return err
}

func (s *plainQueryStream) Close() error {
	return nil
}

type csvQueryStream struct {
	writer *csv.Writer
}

func (s *csvQueryStream) WriteRow(row []string) error {
	return s.writer.Write(row)
}

func (s *csvQueryStream) Close() error {
	s.writer.Flush()
	return s.writer.Error()
}

type ndjsonQueryStream struct {
	encoder *json.Encoder
	columns []Column
}

func (s *ndjsonQueryStream) WriteRow(row []string) error {
	return s.encoder.Encode(recordForStringRow(s.columns, row))
}

func (s *ndjsonQueryStream) Close() error {
	return nil
}

type jsonQueryStream struct {
	out       io.Writer
	encoder   *json.Encoder
	columns   []Column
	wroteRows bool
	closed    bool
}

func (s *jsonQueryStream) WriteRow(row []string) error {
	if s.closed {
		return fmt.Errorf("json stream is already closed")
	}
	if s.wroteRows {
		if _, err := io.WriteString(s.out, ",\n"); err != nil {
			return err
		}
	}
	if err := s.encoder.Encode(recordForStringRow(s.columns, row)); err != nil {
		return err
	}
	s.wroteRows = true
	return nil
}

func (s *jsonQueryStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.wroteRows {
		_, err := io.WriteString(s.out, "]\n")
		return err
	}
	_, err := io.WriteString(s.out, "]\n")
	return err
}

func (w *Writer) beginPlainStream(columns []Column) (QueryStream, error) {
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
	}
	if _, err := fmt.Fprintln(w.out, strings.Join(headers, "\t")); err != nil {
		return nil, err
	}
	return &plainQueryStream{out: w.out}, nil
}

func (w *Writer) beginCSVStream(columns []Column) (QueryStream, error) {
	csvWriter := csv.NewWriter(w.out)
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
	}
	if err := csvWriter.Write(headers); err != nil {
		return nil, fmt.Errorf("writing CSV headers: %w", err)
	}
	return &csvQueryStream{writer: csvWriter}, nil
}

func (w *Writer) beginNDJSONStream(columns []Column) QueryStream {
	return &ndjsonQueryStream{
		encoder: json.NewEncoder(w.out),
		columns: columns,
	}
}

func (w *Writer) beginJSONStream(columns []Column) (QueryStream, error) {
	if _, err := io.WriteString(w.out, "["); err != nil {
		return nil, err
	}
	return &jsonQueryStream{
		out:     w.out,
		encoder: json.NewEncoder(w.out),
		columns: columns,
	}, nil
}

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "auto", "":
		return FormatAuto, nil
	case "table":
		return FormatTable, nil
	case "plain":
		return FormatPlain, nil
	case "json":
		return FormatJSON, nil
	case "ndjson":
		return FormatNDJSON, nil
	case "csv":
		return FormatCSV, nil
	default:
		return "", fmt.Errorf("unknown format %q (valid: auto, table, plain, json, ndjson, csv)", s)
	}
}
