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

package source

import (
	"fmt"
	"sort"
	"strings"
)

// ExportWriter receives one exported row, header row first. text marks the
// cells that hold free text, which CSV output guards against spreadsheet
// formulas; a nil text slice marks every cell.
type ExportWriter func(row []string, text []bool) error

// ExportSelection is a set of grid rows to export instead of a whole object.
type ExportSelection struct {
	// Columns lists the grid's columns in display order, with the types the
	// grid showed for them.
	Columns []Column
	Rows    []map[string]any
}

// HasRows reports whether the selection names any rows; a nil selection has none.
func (s *ExportSelection) HasRows() bool {
	return s != nil && len(s.Rows) > 0
}

// ColumnNames returns the selection's columns in display order. Without
// declared columns it returns every key used by the rows, sorted, because the
// rows themselves carry no order.
func (s *ExportSelection) ColumnNames() []string {
	if len(s.Columns) > 0 {
		names := make([]string, len(s.Columns))
		for i, column := range s.Columns {
			names[i] = column.Name
		}
		return names
	}

	seen := map[string]bool{}
	names := []string{}
	for _, row := range s.Rows {
		for key := range row {
			if !seen[key] {
				seen[key] = true
				names = append(names, key)
			}
		}
	}
	sort.Strings(names)
	return names
}

// FreeTextColumns reports, per column, whether its values are free text. A
// column is not free text only when its type is declared numeric, date/time or
// boolean: those values are produced by the source and cannot carry a formula.
// Undeclared types count as text so an unknown column is never left unguarded.
func FreeTextColumns(metadata *TypeSessionMetadata, columns []Column) []bool {
	text := make([]bool, len(columns))
	for i, column := range columns {
		text[i] = isFreeTextType(metadata, column.Type)
	}
	return text
}

func isFreeTextType(metadata *TypeSessionMetadata, typeName string) bool {
	if metadata == nil {
		return true
	}

	base, parameters, parameterised := strings.Cut(typeName, "(")
	base = strings.ToUpper(strings.TrimSpace(base))
	if canonical, ok := metadata.AliasMap[base]; ok {
		base = strings.ToUpper(canonical)
	}

	for i := range metadata.TypeDefinitions {
		if strings.ToUpper(metadata.TypeDefinitions[i].ID) != base {
			continue
		}
		switch metadata.TypeDefinitions[i].Category {
		case TypeCategoryNumeric, TypeCategoryDatetime, TypeCategoryBoolean:
			return false
		}
		return true
	}

	// An undeclared type that wraps one other type, such as Nullable(Int32),
	// holds only what the wrapped type can hold.
	if parameterised && strings.HasSuffix(parameters, ")") {
		return isFreeTextType(metadata, strings.TrimSuffix(parameters, ")"))
	}
	return true
}

// Write emits the selection as a header row followed by its rows, with values
// exactly as the grid supplied them. text follows ExportWriter's contract.
func (s *ExportSelection) Write(writer ExportWriter, text []bool) error {
	names := s.ColumnNames()
	if err := writer(names, nil); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	for i, selected := range s.Rows {
		row := make([]string, len(names))
		for j, name := range names {
			switch value := selected[name].(type) {
			case nil:
			case string:
				row[j] = value
			default:
				row[j] = fmt.Sprint(value)
			}
		}
		if err := writer(row, text); err != nil {
			return fmt.Errorf("failed to write row %d: %w", i+1, err)
		}
	}
	return nil
}

// FreeTextColumns reports which of the selection's declared columns hold free
// text, or nil, marking every cell, when the selection declares no columns.
func (s *ExportSelection) FreeTextColumns(metadata *TypeSessionMetadata) []bool {
	if len(s.Columns) == 0 {
		return nil
	}
	return FreeTextColumns(metadata, s.Columns)
}
