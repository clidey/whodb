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

package platform

import "strings"

// FileColumnInspection describes one inferred tabular file column.
type FileColumnInspection struct {
	SourceColumn     string `json:"sourceColumn"`
	DatasetColumn    string `json:"datasetColumn"`
	DataType         string `json:"dataType"`
	IsNullable       bool   `json:"isNullable"`
	IsPrimary        bool   `json:"isPrimary"`
	ColumnMapFlag    string `json:"columnMapFlag"`
	ColumnMapSnippet string `json:"columnMapSnippet"`
}

// FileInspection describes a project file and promotion-ready column mappings.
type FileInspection struct {
	FileID           string                 `json:"fileId"`
	MIMEType         string                 `json:"mimeType"`
	SizeBytes        int                    `json:"sizeBytes"`
	IsTabular        bool                   `json:"isTabular"`
	SheetName        *string                `json:"sheetName,omitempty"`
	SheetIndex       *int                   `json:"sheetIndex,omitempty"`
	SheetCount       *int                   `json:"sheetCount,omitempty"`
	SheetNames       []string               `json:"sheetNames,omitempty"`
	Columns          []FileColumnInspection `json:"columns"`
	Rows             [][]string             `json:"rows,omitempty"`
	Total            int                    `json:"total"`
	ColumnMapFlags   []string               `json:"columnMapFlags"`
	ColumnMapExample string                 `json:"columnMapExample"`
}

// InspectFilePreview builds a stable inspection payload from a file preview.
func InspectFilePreview(fileID string, preview *FilePreviewResult) *FileInspection {
	inspection := &FileInspection{FileID: strings.TrimSpace(fileID)}
	if preview == nil {
		return inspection
	}
	inspection.MIMEType = preview.MIMEType
	inspection.SizeBytes = preview.SizeBytes
	inspection.IsTabular = preview.IsTabular
	if preview.Tabular == nil {
		return inspection
	}
	inspection.SheetName = preview.Tabular.SheetName
	inspection.SheetIndex = preview.Tabular.SheetIndex
	inspection.SheetCount = preview.Tabular.SheetCount
	inspection.SheetNames = preview.Tabular.SheetNames
	inspection.Rows = preview.Tabular.Rows
	inspection.Total = preview.Tabular.Total
	inspection.Columns = make([]FileColumnInspection, 0, len(preview.Tabular.Columns))
	inspection.ColumnMapFlags = make([]string, 0, len(preview.Tabular.Columns))
	for _, column := range preview.Tabular.Columns {
		inspected := inspectFileColumn(column)
		inspection.Columns = append(inspection.Columns, inspected)
		inspection.ColumnMapFlags = append(inspection.ColumnMapFlags, "--column-map "+inspected.ColumnMapSnippet)
	}
	inspection.ColumnMapExample = strings.Join(inspection.ColumnMapFlags, " ")
	return inspection
}

func inspectFileColumn(column FilePreviewColumn) FileColumnInspection {
	source := strings.TrimSpace(column.Name)
	if source == "" {
		source = "column"
	}
	dataset := normalizeDatasetColumnName(source)
	dataType := normalizeFilePreviewDataType(column.Type)
	isPrimary := isLikelyPrimaryColumn(source)
	isNullable := !isPrimary
	options := []string{}
	if isNullable {
		options = append(options, "nullable")
	}
	if isPrimary {
		options = append(options, "primary")
	}
	parts := []string{source, dataset, dataType}
	parts = append(parts, options...)
	snippet := strings.Join(parts, ":")
	return FileColumnInspection{
		SourceColumn:     source,
		DatasetColumn:    dataset,
		DataType:         dataType,
		IsNullable:       isNullable,
		IsPrimary:        isPrimary,
		ColumnMapFlag:    "--column-map " + snippet,
		ColumnMapSnippet: snippet,
	}
}

func normalizeDatasetColumnName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		keep := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if keep {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && builder.Len() > 0 {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return "column"
	}
	return out
}

func normalizeFilePreviewDataType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "int", "integer", "bigint", "smallint", "number":
		return "integer"
	case "float", "double", "decimal", "numeric", "real":
		return "float"
	case "bool", "boolean":
		return "boolean"
	case "timestamp", "datetime":
		return "timestamp"
	case "date":
		return "date"
	case "json", "jsonb", "object", "array":
		return "json"
	default:
		return "text"
	}
}

func isLikelyPrimaryColumn(value string) bool {
	normalized := normalizeDatasetColumnName(value)
	return normalized == "id" || strings.HasSuffix(normalized, "_id")
}
