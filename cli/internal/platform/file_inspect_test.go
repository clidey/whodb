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

import "testing"

func TestInspectFilePreviewInfersPromotionColumnMappings(t *testing.T) {
	preview := &FilePreviewResult{
		MIMEType:  "text/csv",
		SizeBytes: 128,
		IsTabular: true,
		Tabular: &TabularPreviewData{
			Columns: []FilePreviewColumn{
				{Name: "ID", Type: "integer"},
				{Name: "Customer Name", Type: "string"},
				{Name: "Signed Up At", Type: "datetime"},
			},
			Rows:  [][]string{{"1", "Ada", "2026-01-01T00:00:00Z"}},
			Total: 1,
		},
	}

	inspection := InspectFilePreview("file-1", preview)

	if inspection.FileID != "file-1" || !inspection.IsTabular || len(inspection.Columns) != 3 {
		t.Fatalf("inspection = %#v, want tabular file with three columns", inspection)
	}
	if inspection.Columns[0].ColumnMapSnippet != "ID:id:integer:primary" {
		t.Fatalf("first column map = %q, want primary id", inspection.Columns[0].ColumnMapSnippet)
	}
	if inspection.Columns[1].ColumnMapSnippet != "Customer Name:customer_name:text:nullable" {
		t.Fatalf("second column map = %q, want normalized nullable text", inspection.Columns[1].ColumnMapSnippet)
	}
	if inspection.Columns[2].DataType != "timestamp" {
		t.Fatalf("third data type = %q, want timestamp", inspection.Columns[2].DataType)
	}
	if inspection.ColumnMapExample == "" || len(inspection.ColumnMapFlags) != 3 {
		t.Fatalf("column map example = %q flags=%#v, want command-ready mappings", inspection.ColumnMapExample, inspection.ColumnMapFlags)
	}
}
