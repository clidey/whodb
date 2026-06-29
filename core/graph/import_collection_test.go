package graph

import (
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestParseJSONDocumentsArray(t *testing.T) {
	docs, truncated, err := parseJSONDocuments([]byte(`[{"a":"x"},{"a":"y"}]`), 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Fatalf("did not expect truncation")
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}
	if docs[0]["a"] != "x" || docs[1]["a"] != "y" {
		t.Fatalf("unexpected documents: %+v", docs)
	}
}

func TestParseJSONDocumentsNDJSON(t *testing.T) {
	docs, _, err := parseJSONDocuments([]byte("{\"a\":\"1\"}\n\n{\"a\":\"2\"}\n"), 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents (blank line skipped), got %d", len(docs))
	}
}

func TestParseJSONDocumentsExtendedJSONTypes(t *testing.T) {
	data := []byte(`[{"_id":{"$oid":"5d505646cf6d4fe581014ab2"},"n":{"$numberInt":"5"},"big":{"$numberLong":"9000000000"}}]`)
	docs, _, err := parseJSONDocuments(data, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := docs[0]["_id"].(primitive.ObjectID); !ok {
		t.Fatalf("expected _id to be ObjectID, got %T", docs[0]["_id"])
	}
	if v, ok := docs[0]["n"].(int32); !ok || v != 5 {
		t.Fatalf("expected n to be int32(5), got %T %v", docs[0]["n"], docs[0]["n"])
	}
	if v, ok := docs[0]["big"].(int64); !ok || v != 9000000000 {
		t.Fatalf("expected big to be int64, got %T %v", docs[0]["big"], docs[0]["big"])
	}
}

func TestParseJSONDocumentsEmptyAndMalformed(t *testing.T) {
	if _, _, err := parseJSONDocuments([]byte("   \n  "), 0, false); err == nil {
		t.Fatalf("expected error for empty input")
	}
	if _, _, err := parseJSONDocuments([]byte(`[{"a":}]`), 0, false); err == nil {
		t.Fatalf("expected error for malformed array")
	}
	if _, _, err := parseJSONDocuments([]byte("{\"a\":1}\n{bad}"), 0, false); err == nil {
		t.Fatalf("expected error for malformed NDJSON line")
	}
}

func TestParseJSONDocumentsRowCap(t *testing.T) {
	data := []byte(`[{"a":"1"},{"a":"2"},{"a":"3"}]`)

	if _, _, err := parseJSONDocuments(data, 2, true); err == nil || validationKeyFromError(err) != importValidationRowLimitExceeded {
		t.Fatalf("expected row limit error, got %v", err)
	}

	docs, truncated, err := parseJSONDocuments(data, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !truncated || len(docs) != 2 {
		t.Fatalf("expected truncation to 2 docs, got %d truncated=%v", len(docs), truncated)
	}
}

func TestBuildDocumentsFromRows(t *testing.T) {
	parsed := &importParseResult{
		columns: []string{"name", "age", "note"},
		rows: [][]string{
			{"Alice", "30", "hi"},
			{"", "25", ""},
		},
	}

	docs := buildDocumentsFromRows(parsed, []string{"note"})
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}
	// note column is skipped everywhere
	if _, ok := docs[0]["note"]; ok {
		t.Fatalf("expected note column to be skipped")
	}
	// empty cells omitted
	if _, ok := docs[1]["name"]; ok {
		t.Fatalf("expected empty name cell to be omitted")
	}
	if docs[0]["name"] != "Alice" || docs[0]["age"] != "30" {
		t.Fatalf("unexpected first document: %+v", docs[0])
	}
	// values stay strings (no inference)
	if _, ok := docs[0]["age"].(string); !ok {
		t.Fatalf("expected age to remain a string, got %T", docs[0]["age"])
	}
}

func TestCollectionImportFileFormat(t *testing.T) {
	if f, ok := collectionImportFileFormat(model.CollectionImportFormatCSV); !ok || f != model.ImportFileFormatCSV {
		t.Fatalf("expected CSV mapping, got %v ok=%v", f, ok)
	}
	if f, ok := collectionImportFileFormat(model.CollectionImportFormatExcel); !ok || f != model.ImportFileFormatExcel {
		t.Fatalf("expected Excel mapping, got %v ok=%v", f, ok)
	}
	if _, ok := collectionImportFileFormat(model.CollectionImportFormatJSON); ok {
		t.Fatalf("expected JSON to not map to a tabular format")
	}
}

func TestParseCollectionDocumentsCSV(t *testing.T) {
	data := []byte("name,age\nAlice,30\nBob,40\n")
	docs, _, err := parseCollectionDocuments(data, model.CollectionImportFormatCSV, nil, nil, []string{"age"}, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}
	if _, ok := docs[0]["age"]; ok {
		t.Fatalf("expected age column to be skipped")
	}
	if docs[0]["name"] != "Alice" {
		t.Fatalf("unexpected document: %+v", docs[0])
	}
}

func TestBuildCollectionImportPreviewJSON(t *testing.T) {
	data := []byte(`[{"a":"1"},{"a":"2"}]`)
	preview := buildCollectionImportPreview(data, model.ImportCollectionPreviewInput{Format: model.CollectionImportFormatJSON})
	if preview.ValidationError != nil {
		t.Fatalf("unexpected validation error: %v", *preview.ValidationError)
	}
	if preview.Count == nil || *preview.Count != 2 {
		t.Fatalf("expected count 2, got %v", preview.Count)
	}
	if len(preview.Documents) != 2 {
		t.Fatalf("expected 2 rendered documents, got %d", len(preview.Documents))
	}
}

func TestBuildCollectionImportPreviewInvalidJSON(t *testing.T) {
	preview := buildCollectionImportPreview([]byte("not json"), model.ImportCollectionPreviewInput{Format: model.CollectionImportFormatJSON})
	if preview.ValidationError == nil {
		t.Fatalf("expected a validation error for invalid JSON")
	}
}

func TestBuildCollectionImportPreviewCSV(t *testing.T) {
	data := []byte("name,age\nAlice,30\n")
	preview := buildCollectionImportPreview(data, model.ImportCollectionPreviewInput{Format: model.CollectionImportFormatCSV})
	if preview.ValidationError != nil {
		t.Fatalf("unexpected validation error: %v", *preview.ValidationError)
	}
	if len(preview.Columns) != 2 || preview.Columns[0] != "name" {
		t.Fatalf("unexpected columns: %+v", preview.Columns)
	}
	if len(preview.Rows) != 1 {
		t.Fatalf("expected 1 preview row, got %d", len(preview.Rows))
	}
}

func TestBuildCollectionImportPreviewExcel(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	secondSheet := "Second"
	_, _ = f.NewSheet(secondSheet)
	_ = f.SetSheetRow(sheet, "A1", &[]any{"name", "age"})
	_ = f.SetSheetRow(sheet, "A2", &[]any{"Alice", "30"})
	_ = f.SetSheetRow(secondSheet, "A1", &[]any{"title"})
	_ = f.SetSheetRow(secondSheet, "A2", &[]any{"Report"})
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("failed to build excel: %v", err)
	}
	preview := buildCollectionImportPreview(buf.Bytes(), model.ImportCollectionPreviewInput{Format: model.CollectionImportFormatExcel})
	if preview.ValidationError != nil {
		t.Fatalf("unexpected validation error: %v", *preview.ValidationError)
	}
	if len(preview.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %+v", preview.Columns)
	}
	if preview.Sheet == nil || *preview.Sheet != sheet {
		t.Fatalf("expected default sheet %q, got %v", sheet, preview.Sheet)
	}
	if len(preview.Sheets) != 2 || preview.Sheets[0] != sheet || preview.Sheets[1] != secondSheet {
		t.Fatalf("unexpected sheets: %+v", preview.Sheets)
	}

	preview = buildCollectionImportPreview(buf.Bytes(), model.ImportCollectionPreviewInput{
		Format: model.CollectionImportFormatExcel,
		Sheet:  &secondSheet,
	})
	if preview.ValidationError != nil {
		t.Fatalf("unexpected selected sheet validation error: %v", *preview.ValidationError)
	}
	if preview.Sheet == nil || *preview.Sheet != secondSheet {
		t.Fatalf("expected selected sheet %q, got %v", secondSheet, preview.Sheet)
	}
	if len(preview.Columns) != 1 || preview.Columns[0] != "title" {
		t.Fatalf("unexpected selected sheet columns: %+v", preview.Columns)
	}
}

func TestCollectionImportErrorsCap(t *testing.T) {
	failures := make([]engine.DocumentImportFailure, collectionImportErrorLimit+10)
	for i := range failures {
		failures[i] = engine.DocumentImportFailure{Index: i, Reason: "dup"}
	}
	errs := collectionImportErrors(failures)
	if len(errs) != collectionImportErrorLimit {
		t.Fatalf("expected errors capped at %d, got %d", collectionImportErrorLimit, len(errs))
	}
}

func TestCollectionResultBuilders(t *testing.T) {
	failures := []engine.DocumentImportFailure{{Index: 1, Reason: "dup"}}

	insert := collectionInsertResult(9, failures)
	if !insert.Status || insert.ImportedCount != 9 || insert.SkippedCount != 1 {
		t.Fatalf("unexpected insert result: %+v", insert)
	}

	upsert := collectionUpsertResult(10, engine.DocumentUpsertResult{Matched: 4, Modified: 3, Upserted: 5}, failures)
	if upsert.ImportedCount != 9 || upsert.SkippedCount != 1 {
		t.Fatalf("unexpected upsert counts: %+v", upsert)
	}
	if upsert.MatchedCount == nil || *upsert.MatchedCount != 4 {
		t.Fatalf("expected matched 4, got %v", upsert.MatchedCount)
	}
	if upsert.UpsertedCount == nil || *upsert.UpsertedCount != 5 {
		t.Fatalf("expected upserted 5, got %v", upsert.UpsertedCount)
	}
}

// ensure failure builder marshals the detail key without panicking on no message
func TestCollectionImportFailure(t *testing.T) {
	res := collectionImportFailure(importErrorCollectionUnsupported)
	if res.Status {
		t.Fatalf("expected failure result")
	}
	if res.Detail == nil || *res.Detail != importErrorCollectionUnsupported {
		t.Fatalf("expected detail key, got %v", res.Detail)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("expected no per-document errors")
	}

	withMsg := collectionImportFailure(importErrorImportFailed, "boom")
	if withMsg.Message == nil || *withMsg.Message != "boom" {
		t.Fatalf("expected message, got %v", withMsg.Message)
	}
}
