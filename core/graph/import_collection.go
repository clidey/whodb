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

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/bson"
)

// collectionImportFileFormat maps the tabular collection formats to the shared
// ImportFileFormat. JSON is not tabular and returns ok=false.
func collectionImportFileFormat(format model.CollectionImportFormat) (model.ImportFileFormat, bool) {
	switch format {
	case model.CollectionImportFormatCSV:
		return model.ImportFileFormatCSV, true
	case model.CollectionImportFormatExcel:
		return model.ImportFileFormatExcel, true
	default:
		return "", false
	}
}

// parseCollectionDocuments parses an uploaded file into documents ready for a
// MongoDB collection. JSON preserves nested structure and types via Extended
// JSON; CSV/Excel rows become flat string-valued documents.
func parseCollectionDocuments(data []byte, format model.CollectionImportFormat, delimiter *string, sheet *string, skipColumns []string, maxRows int, enforceRowCap bool) ([]map[string]any, bool, error) {
	if format == model.CollectionImportFormatJSON {
		return parseJSONDocuments(data, maxRows, enforceRowCap)
	}

	fileFormat, ok := collectionImportFileFormat(format)
	if !ok {
		return nil, false, newImportValidationError(importValidationUnsupportedFormat)
	}

	options := &model.ImportFileOptions{Format: fileFormat, Delimiter: delimiter, Sheet: sheet}
	parsed, err := parseImportFile(data, options, maxRows, enforceRowCap)
	if err != nil {
		return nil, false, err
	}
	return buildDocumentsFromRows(parsed, skipColumns), parsed.truncated, nil
}

// parseJSONDocuments parses a JSON import file into documents. It accepts either
// a top-level JSON array of objects or newline-delimited JSON (one object per
// line), chosen by the first non-whitespace byte. Values use MongoDB Extended
// JSON (relaxed) semantics, matching the JSON document editor.
func parseJSONDocuments(data []byte, maxRows int, enforceRowCap bool) ([]map[string]any, bool, error) {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return nil, false, newImportValidationError(importValidationParseFailed)
	}

	var documents []map[string]any
	if trimmed[0] == '[' {
		if err := bson.UnmarshalExtJSON(trimmed, false, &documents); err != nil {
			return nil, false, newImportValidationError(importValidationParseFailed)
		}
	} else {
		for _, raw := range bytes.Split(trimmed, []byte("\n")) {
			line := bytes.TrimSpace(raw)
			if len(line) == 0 {
				continue
			}
			var doc map[string]any
			if err := bson.UnmarshalExtJSON(line, false, &doc); err != nil {
				return nil, false, newImportValidationError(importValidationParseFailed)
			}
			documents = append(documents, doc)
		}
	}

	if maxRows > 0 && len(documents) > maxRows {
		if enforceRowCap {
			return nil, false, newImportValidationError(importValidationRowLimitExceeded)
		}
		return documents[:maxRows], true, nil
	}
	return documents, false, nil
}

// buildDocumentsFromRows converts parsed tabular rows into documents. Each row
// becomes one document keyed by the header columns; skipped columns are dropped
// and empty cells are omitted so the field is simply absent. Values stay strings
// (no type inference), matching the documented Collection File Import behavior.
func buildDocumentsFromRows(parsed *importParseResult, skipColumns []string) []map[string]any {
	skip := make(map[string]bool, len(skipColumns))
	for _, column := range skipColumns {
		skip[column] = true
	}

	documents := make([]map[string]any, 0, len(parsed.rows))
	for _, row := range parsed.rows {
		document := make(map[string]any, len(parsed.columns))
		for i, column := range parsed.columns {
			if skip[column] || i >= len(row) || row[i] == "" {
				continue
			}
			document[column] = row[i]
		}
		documents = append(documents, document)
	}
	return documents
}

// buildCollectionImportPreview parses a sample of the file for display. It never
// returns a hard error for malformed content; parse problems are reported in the
// preview's ValidationError so the modal can show them inline.
func buildCollectionImportPreview(data []byte, input model.ImportCollectionPreviewInput) *model.CollectionImportPreview {
	preview := &model.CollectionImportPreview{
		Format:    input.Format,
		Sheets:    []string{},
		Columns:   []string{},
		Rows:      [][]string{},
		Documents: []string{},
	}

	if input.Format == model.CollectionImportFormatJSON {
		documents, _, err := parseJSONDocuments(data, maxImportRows, false)
		if err != nil {
			preview.ValidationError = validationKeyPtr(err)
			return preview
		}
		total := len(documents)
		preview.Count = &total
		limit := total
		if limit > importPreviewRowLimit {
			limit = importPreviewRowLimit
			preview.Truncated = true
		}
		rendered := make([]string, 0, limit)
		for _, document := range documents[:limit] {
			text, marshalErr := bson.MarshalExtJSON(document, false, false)
			if marshalErr != nil {
				continue
			}
			rendered = append(rendered, string(text))
		}
		preview.Documents = rendered
		return preview
	}

	fileFormat, ok := collectionImportFileFormat(input.Format)
	if !ok {
		key := importValidationUnsupportedFormat
		preview.ValidationError = &key
		return preview
	}

	options := &model.ImportFileOptions{Format: fileFormat, Delimiter: input.Delimiter, Sheet: input.Sheet}
	parsed, err := parseImportFile(data, options, importPreviewRowLimit, false)
	if err != nil {
		preview.ValidationError = validationKeyPtr(err)
		return preview
	}
	if parsed.sheet != "" {
		preview.Sheet = &parsed.sheet
	}
	preview.Sheets = parsed.sheets
	preview.Columns = parsed.columns
	preview.Rows = parsed.rows
	preview.Truncated = parsed.truncated
	return preview
}

// runCollectionImport executes a parsed import against the collection importer,
// applying the requested mode. Overwrite clears the collection first, then
// inserts; the clear is not transactional and cannot be rolled back.
func runCollectionImport(plugin *engine.Plugin, config *engine.PluginConfig, importer engine.CollectionImporter, input model.ImportCollectionFileInput, documents []map[string]any) (*model.CollectionImportResult, error) {
	if input.Mode == model.ImportModeOverwrite {
		if _, err := plugin.ClearTableData(config, input.Schema, input.Collection); err != nil {
			return collectionImportFailure(importErrorClearFailed, err.Error()), nil
		}
	}

	if input.Mode == model.ImportModeUpsert {
		outcome, failures, err := importer.UpsertDocuments(config, input.Schema, input.Collection, input.UpsertKeys, documents)
		if err != nil {
			return collectionImportFailure(importErrorImportFailed, err.Error()), nil
		}
		return collectionUpsertResult(len(documents), outcome, failures), nil
	}

	inserted, failures, err := importer.InsertDocuments(config, input.Schema, input.Collection, documents)
	if err != nil {
		return collectionImportFailure(importErrorImportFailed, err.Error()), nil
	}
	return collectionInsertResult(inserted, failures), nil
}

// collectionInsertResult builds a successful result for insert-based modes.
func collectionInsertResult(inserted int, failures []engine.DocumentImportFailure) *model.CollectionImportResult {
	return &model.CollectionImportResult{
		Status:        true,
		ImportedCount: inserted,
		SkippedCount:  len(failures),
		Errors:        collectionImportErrors(failures),
	}
}

// collectionUpsertResult builds a successful result for upsert mode, exposing the
// matched/modified/upserted breakdown alongside the processed and skipped counts.
func collectionUpsertResult(total int, outcome engine.DocumentUpsertResult, failures []engine.DocumentImportFailure) *model.CollectionImportResult {
	matched := outcome.Matched
	modified := outcome.Modified
	upserted := outcome.Upserted
	return &model.CollectionImportResult{
		Status:        true,
		ImportedCount: total - len(failures),
		SkippedCount:  len(failures),
		MatchedCount:  &matched,
		ModifiedCount: &modified,
		UpsertedCount: &upserted,
		Errors:        collectionImportErrors(failures),
	}
}

// collectionImportFailure builds a failed result carrying an i18n detail key and
// an optional raw message.
func collectionImportFailure(detail string, message ...string) *model.CollectionImportResult {
	result := &model.CollectionImportResult{
		Status: false,
		Errors: []*model.CollectionImportError{},
		Detail: &detail,
	}
	if len(message) > 0 && message[0] != "" {
		result.Message = &message[0]
	}
	return result
}

// collectionImportErrors converts skipped-document failures into result errors,
// capped at collectionImportErrorLimit to avoid flooding the client.
func collectionImportErrors(failures []engine.DocumentImportFailure) []*model.CollectionImportError {
	limit := len(failures)
	if limit > collectionImportErrorLimit {
		limit = collectionImportErrorLimit
	}
	errors := make([]*model.CollectionImportError, 0, limit)
	for _, failure := range failures[:limit] {
		errors = append(errors, &model.CollectionImportError{Index: failure.Index, Reason: failure.Reason})
	}
	return errors
}

// validationKeyPtr returns a pointer to the i18n validation key for err.
func validationKeyPtr(err error) *string {
	key := validationKeyFromError(err)
	return &key
}
