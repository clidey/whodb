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

package graph

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/xuri/excelize/v2"
)

type importParseResult struct {
	columns   []string
	rows      [][]string
	truncated bool
	sheet     string
}

type importMapping struct {
	sourceIndex int
	targetName  string
	targetType  string
}

type importValidationError struct {
	key string
}

func (err importValidationError) Error() string {
	return err.key
}

func newImportValidationError(key string) error {
	return importValidationError{key: key}
}

func isImportValidationKey(value string) bool {
	return strings.HasPrefix(value, "import.validation.")
}

func validationKeyFromError(err error) string {
	if err == nil {
		return ""
	}
	var validationErr importValidationError
	if errors.As(err, &validationErr) {
		return validationErr.key
	}
	if isImportValidationKey(err.Error()) {
		return err.Error()
	}
	return importValidationGeneric
}

func readUploadBytes(upload graphql.Upload, maxBytes int64) ([]byte, error) {
	if upload.File == nil {
		return nil, newImportValidationError(importValidationMissingFile)
	}

	if closer, ok := upload.File.(io.Closer); ok {
		defer closer.Close()
	}

	if upload.Size > maxBytes {
		return nil, newImportValidationError(importValidationFileTooLarge)
	}

	limited := io.LimitReader(upload.File, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, newImportValidationError(importValidationFileTooLarge)
	}

	return data, nil
}

func parseImportFile(data []byte, options *model.ImportFileOptions, maxRows int, enforceRowCap bool) (*importParseResult, error) {
	if options == nil {
		return nil, newImportValidationError(importValidationInvalidOptions)
	}

	switch options.Format {
	case model.ImportFileFormatCSV:
		return parseCSVImport(data, options, maxRows, enforceRowCap)
	case model.ImportFileFormatExcel:
		return parseExcelImport(data, options, maxRows, enforceRowCap)
	default:
		return nil, newImportValidationError(importValidationUnsupportedFormat)
	}
}

func parseCSVImport(data []byte, options *model.ImportFileOptions, maxRows int, enforceRowCap bool) (*importParseResult, error) {
	delimiter := ""
	if options.Delimiter != nil {
		delimiter = *options.Delimiter
	}
	if delimiter == "" {
		detected, err := detectCSVDelimiter(data)
		if err != nil {
			return nil, err
		}
		delimiter = detected
	}
	if err := validateDelimiter(delimiter); err != nil {
		return nil, newImportValidationError(importValidationInvalidDelimiter)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = rune(delimiter[0])
	reader.FieldsPerRecord = -1

	result := &importParseResult{}
	firstRow := true

	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, newImportValidationError(importValidationParseFailed)
		}

		if firstRow {
			firstRow = false
			if err := validateHeaderColumns(row); err != nil {
				return nil, err
			}
			result.columns = row
			continue
		}

		normalized, err := normalizeRow(row, len(result.columns))
		if err != nil {
			return nil, err
		}

		result.rows = append(result.rows, normalized)
		if enforceRowCap && len(result.rows) > maxImportRows {
			return nil, newImportValidationError(importValidationRowLimitExceeded)
		}

		if maxRows > 0 && len(result.rows) >= maxRows {
			result.truncated = true
			break
		}
	}

	if len(result.columns) == 0 {
		return nil, newImportValidationError(importValidationNoColumns)
	}

	return result, nil
}

func parseExcelImport(data []byte, options *model.ImportFileOptions, maxRows int, enforceRowCap bool) (*importParseResult, error) {
	file, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit:    maxImportFileSizeBytes,
		UnzipXMLSizeLimit: maxImportFileSizeBytes,
	})
	if err != nil {
		return nil, newImportValidationError(importValidationParseFailed)
	}
	defer file.Close()

	sheetName := ""
	if options.Sheet != nil {
		sheetName = *options.Sheet
	}
	if sheetName == "" {
		sheets := file.GetSheetList()
		if len(sheets) == 0 {
			return nil, newImportValidationError(importValidationNoColumns)
		}
		sheetName = sheets[0]
	}

	rows, err := file.Rows(sheetName)
	if err != nil {
		return nil, newImportValidationError(importValidationParseFailed)
	}
	defer rows.Close()

	result := &importParseResult{sheet: sheetName}
	firstRow := true

	for rows.Next() {
		row, err := rows.Columns()
		if err != nil {
			return nil, newImportValidationError(importValidationParseFailed)
		}

		if firstRow {
			firstRow = false
			if err := validateHeaderColumns(row); err != nil {
				return nil, err
			}
			result.columns = row
			continue
		}

		normalized, err := normalizeRow(row, len(result.columns))
		if err != nil {
			return nil, err
		}

		result.rows = append(result.rows, normalized)
		if enforceRowCap && len(result.rows) > maxImportRows {
			return nil, newImportValidationError(importValidationRowLimitExceeded)
		}

		if maxRows > 0 && len(result.rows) >= maxRows {
			result.truncated = true
			break
		}
	}

	if len(result.columns) == 0 {
		return nil, newImportValidationError(importValidationNoColumns)
	}

	return result, nil
}

func normalizeRow(row []string, columnCount int) ([]string, error) {
	if len(row) > columnCount {
		return nil, newImportValidationError(importValidationRowTooManyColumns)
	}

	if len(row) == columnCount {
		return row, nil
	}

	normalized := make([]string, columnCount)
	copy(normalized, row)
	for i := len(row); i < columnCount; i++ {
		normalized[i] = ""
	}
	return normalized, nil
}

func validateHeaderColumns(columns []string) error {
	seen := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		if column == "" {
			return newImportValidationError(importValidationEmptyHeader)
		}
		if _, exists := seen[column]; exists {
			return newImportValidationError(importValidationDuplicateHeader)
		}
		seen[column] = struct{}{}
	}
	return nil
}

func detectCSVDelimiter(data []byte) (string, error) {
	candidates := []rune{',', ';', '|'}
	valid := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		reader := csv.NewReader(bytes.NewReader(data))
		reader.Comma = candidate
		reader.FieldsPerRecord = -1

		records := 0
		expectedFields := -1
		consistent := true
		for records < 5 {
			row, err := reader.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				consistent = false
				break
			}

			if len(row) == 0 || (len(row) == 1 && row[0] == "") {
				continue
			}

			fieldCount := len(row)
			if expectedFields == -1 {
				expectedFields = fieldCount
			} else if expectedFields != fieldCount {
				consistent = false
				break
			}
			records++
		}

		if !consistent || records == 0 {
			continue
		}
		if expectedFields <= 1 {
			continue
		}
		valid = append(valid, string(candidate))
	}

	if len(valid) == 0 {
		return ",", nil
	}

	if len(valid) > 1 {
		return "", newImportValidationError(importValidationAmbiguousDelimiter)
	}

	return valid[0], nil
}

func resolveImportMappings(
	sourceColumns []string,
	mappings []*model.ImportColumnMapping,
	targetColumns []engine.Column,
	allowAutoGenerated bool,
) ([]importMapping, error) {
	if len(mappings) == 0 {
		return nil, newImportValidationError(importValidationMappingInvalid)
	}
	if len(mappings) != len(sourceColumns) {
		return nil, newImportValidationError(importValidationMappingInvalid)
	}

	sourceIndex := make(map[string]int, len(sourceColumns))
	for idx, col := range sourceColumns {
		if _, exists := sourceIndex[col]; exists {
			return nil, newImportValidationError(importValidationMappingInvalid)
		}
		sourceIndex[col] = idx
	}

	targetTypes := make(map[string]engine.Column, len(targetColumns))
	for _, column := range targetColumns {
		targetTypes[column.Name] = column
	}

	seenSources := make(map[string]struct{}, len(sourceColumns))
	seenTargets := make(map[string]struct{}, len(sourceColumns))
	resolved := make([]importMapping, 0, len(sourceColumns))
	mappedCount := 0

	for _, mapping := range mappings {
		if mapping == nil {
			return nil, newImportValidationError(importValidationMappingInvalid)
		}
		if _, exists := sourceIndex[mapping.SourceColumn]; !exists {
			return nil, newImportValidationError(importValidationMappingInvalid)
		}
		if _, exists := seenSources[mapping.SourceColumn]; exists {
			return nil, newImportValidationError(importValidationMappingInvalid)
		}
		seenSources[mapping.SourceColumn] = struct{}{}

		targetName := ""
		if mapping.TargetColumn != nil {
			targetName = *mapping.TargetColumn
		}

		if mapping.Skip {
			return nil, newImportValidationError(importValidationMappingInvalid)
		}

		if targetName == "" {
			return nil, newImportValidationError(importValidationMappingInvalid)
		}

		if _, exists := seenTargets[targetName]; exists {
			return nil, newImportValidationError(importValidationMappingInvalid)
		}
		seenTargets[targetName] = struct{}{}

		column, exists := targetTypes[targetName]
		if !exists {
			return nil, newImportValidationError(importValidationUnknownColumn)
		}
		if column.IsComputed {
			return nil, newImportValidationError(importValidationGeneratedColumn)
		}
		if column.IsAutoIncrement && !allowAutoGenerated {
			return nil, newImportValidationError(importValidationAutoGeneratedColumn)
		}

		resolved = append(resolved, importMapping{
			sourceIndex: sourceIndex[mapping.SourceColumn],
			targetName:  targetName,
			targetType:  column.Type,
		})
		mappedCount++
	}

	if len(seenSources) != len(sourceColumns) {
		return nil, newImportValidationError(importValidationMappingInvalid)
	}
	if mappedCount == 0 {
		return nil, newImportValidationError(importValidationMappingInvalid)
	}

	return resolved, nil
}

func buildImportMappingInputs(
	sourceColumns []string,
	targetColumns []engine.Column,
	useHeaderMapping bool,
	allowAutoGenerated bool,
) ([]*model.ImportColumnMapping, []string, error) {
	if len(sourceColumns) == 0 {
		return nil, nil, newImportValidationError(importValidationNoColumns)
	}

	if useHeaderMapping {
		targetByName := make(map[string]engine.Column, len(targetColumns))
		for _, column := range targetColumns {
			targetByName[column.Name] = column
		}

		mappings := make([]*model.ImportColumnMapping, 0, len(sourceColumns))
		autoGeneratedColumns := make([]string, 0)
		for _, source := range sourceColumns {
			column, ok := targetByName[source]
			if !ok {
				return nil, nil, newImportValidationError(importValidationUnknownColumn)
			}
			if column.IsComputed {
				return nil, nil, newImportValidationError(importValidationGeneratedColumn)
			}
			if column.IsAutoIncrement {
				autoGeneratedColumns = append(autoGeneratedColumns, column.Name)
			}
			targetCopy := column.Name
			mappings = append(mappings, &model.ImportColumnMapping{
				SourceColumn: source,
				TargetColumn: &targetCopy,
				Skip:         false,
			})
		}

		if !allowAutoGenerated && len(autoGeneratedColumns) > 0 {
			return mappings, autoGeneratedColumns, newImportValidationError(importValidationAutoGeneratedToggle)
		}

		if _, err := resolveImportMappings(sourceColumns, mappings, targetColumns, allowAutoGenerated); err != nil {
			return nil, autoGeneratedColumns, err
		}
		return mappings, autoGeneratedColumns, nil
	}

	insertable := make([]engine.Column, 0, len(targetColumns))
	autoGeneratedColumns := make([]string, 0)
	for _, column := range targetColumns {
		if column.IsComputed {
			continue
		}
		if column.IsAutoIncrement {
			autoGeneratedColumns = append(autoGeneratedColumns, column.Name)
			continue
		}
		insertable = append(insertable, column)
	}

	if len(sourceColumns) != len(insertable) {
		if len(autoGeneratedColumns) > 0 {
			sort.Strings(autoGeneratedColumns)
			return nil, autoGeneratedColumns, newImportValidationError(importValidationColumnCountInsertable)
		}
		return nil, nil, newImportValidationError(importValidationColumnCountMismatch)
	}

	mappings := make([]*model.ImportColumnMapping, 0, len(sourceColumns))
	for idx, source := range sourceColumns {
		targetCopy := insertable[idx].Name
		mappings = append(mappings, &model.ImportColumnMapping{
			SourceColumn: source,
			TargetColumn: &targetCopy,
			Skip:         false,
		})
	}

	if _, err := resolveImportMappings(sourceColumns, mappings, targetColumns, false); err != nil {
		return nil, nil, err
	}
	return mappings, nil, nil
}

func importResult(success bool, detail string) *model.ImportResult {
	result := &model.ImportResult{Status: success}
	if detail != "" {
		result.Detail = &detail
	}
	return result
}
