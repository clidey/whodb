/*
 * Copyright 2025 Clidey, Inc.
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
	"strings"
	"text/tabwriter"

	"github.com/clidey/whodb/cli/pkg/styles"
	"golang.org/x/term"
)

type Format string

const (
	FormatAuto  Format = "auto"
	FormatTable Format = "table"
	FormatPlain Format = "plain"
	FormatJSON  Format = "json"
	FormatCSV   Format = "csv"
)

type QueryResult struct {
	Columns []Column `json:"columns"`
	Rows    [][]any  `json:"rows"`
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

func (w *Writer) writeJSON(result *QueryResult) error {
	output := make([]map[string]any, 0, len(result.Rows))

	for _, row := range result.Rows {
		record := make(map[string]any)
		for i, col := range result.Columns {
			if i < len(row) {
				record[col.Name] = row[i]
			}
		}
		output = append(output, record)
	}

	encoder := json.NewEncoder(w.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (w *Writer) writeCSV(result *QueryResult) error {
	csvWriter := csv.NewWriter(w.out)
	defer csvWriter.Flush()

	headers := make([]string, len(result.Columns))
	for i, col := range result.Columns {
		headers[i] = col.Name
	}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("writing CSV headers: %w", err)
	}

	for _, row := range result.Rows {
		record := make([]string, len(row))
		for i, cell := range row {
			record[i] = fmt.Sprintf("%v", cell)
		}
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	return csvWriter.Error()
}

func (w *Writer) writePlain(result *QueryResult) error {
	headers := make([]string, len(result.Columns))
	for i, col := range result.Columns {
		headers[i] = col.Name
	}
	fmt.Fprintln(w.out, strings.Join(headers, "\t"))

	for _, row := range result.Rows {
		values := make([]string, len(row))
		for i, cell := range row {
			values[i] = fmt.Sprintf("%v", cell)
		}
		fmt.Fprintln(w.out, strings.Join(values, "\t"))
	}

	return nil
}

func (w *Writer) writeTable(result *QueryResult) error {
	tw := tabwriter.NewWriter(w.out, 0, 0, 2, ' ', 0)

	for i, col := range result.Columns {
		if w.ColorEnabled() {
			fmt.Fprint(tw, "\033[1m"+col.Name+"\033[0m")
		} else {
			fmt.Fprint(tw, col.Name)
		}
		if i < len(result.Columns)-1 {
			fmt.Fprint(tw, "\t")
		}
	}
	fmt.Fprintln(tw)

	for i := range result.Columns {
		fmt.Fprint(tw, "───")
		if i < len(result.Columns)-1 {
			fmt.Fprint(tw, "\t")
		}
	}
	fmt.Fprintln(tw)

	for _, row := range result.Rows {
		for i, cell := range row {
			fmt.Fprintf(tw, "%v", cell)
			if i < len(row)-1 {
				fmt.Fprint(tw, "\t")
			}
		}
		fmt.Fprintln(tw)
	}

	return tw.Flush()
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
	case "csv":
		return FormatCSV, nil
	default:
		return "", fmt.Errorf("unknown format %q (valid: auto, table, plain, json, csv)", s)
	}
}
