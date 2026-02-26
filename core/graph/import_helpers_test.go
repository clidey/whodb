package graph

import (
	"bytes"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/xuri/excelize/v2"
)

func TestValidationKeyFromError(t *testing.T) {
	if got := validationKeyFromError(nil); got != "" {
		t.Fatalf("expected empty key for nil error, got %q", got)
	}

	if got := validationKeyFromError(newImportValidationError(importValidationMissingFile)); got != importValidationMissingFile {
		t.Fatalf("expected %q, got %q", importValidationMissingFile, got)
	}

	// Should accept raw key strings as errors
	if got := validationKeyFromError(errors.New(importValidationInvalidDelimiter)); got != importValidationInvalidDelimiter {
		t.Fatalf("expected %q, got %q", importValidationInvalidDelimiter, got)
	}

	if got := validationKeyFromError(errors.New("some other error")); got != importValidationGeneric {
		t.Fatalf("expected generic key, got %q", got)
	}
}

func TestReadUploadBytesValidations(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := readUploadBytes(graphql.Upload{}, 10)
		if err == nil || validationKeyFromError(err) != importValidationMissingFile {
			t.Fatalf("expected %q error, got %v", importValidationMissingFile, err)
		}
	})

	t.Run("too large by advertised size", func(t *testing.T) {
		up := graphql.Upload{
			File: bytes.NewReader([]byte("123")),
			Size: 100,
		}
		_, err := readUploadBytes(up, 10)
		if err == nil || validationKeyFromError(err) != importValidationFileTooLarge {
			t.Fatalf("expected %q error, got %v", importValidationFileTooLarge, err)
		}
	})

	t.Run("too large by actual read", func(t *testing.T) {
		up := graphql.Upload{
			File: bytes.NewReader([]byte("0123456789ABCDEF")),
			Size: 1,
		}
		_, err := readUploadBytes(up, 10)
		if err == nil || validationKeyFromError(err) != importValidationFileTooLarge {
			t.Fatalf("expected %q error, got %v", importValidationFileTooLarge, err)
		}
	})

	t.Run("success", func(t *testing.T) {
		data := []byte("hello")
		up := graphql.Upload{
			File: bytes.NewReader(data),
			Size: int64(len(data)),
		}
		got, err := readUploadBytes(up, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("expected %q, got %q", string(data), string(got))
		}
	})
}

func TestDetectCSVDelimiter(t *testing.T) {
	t.Run("comma", func(t *testing.T) {
		delim, err := detectCSVDelimiter([]byte("a,b\n1,2\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if delim != "," {
			t.Fatalf("expected ',', got %q", delim)
		}
	})

	t.Run("semicolon", func(t *testing.T) {
		delim, err := detectCSVDelimiter([]byte("a;b\n1;2\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if delim != ";" {
			t.Fatalf("expected ';', got %q", delim)
		}
	})

	t.Run("pipe", func(t *testing.T) {
		delim, err := detectCSVDelimiter([]byte("a|b\n1|2\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if delim != "|" {
			t.Fatalf("expected '|', got %q", delim)
		}
	})

	t.Run("ambiguous", func(t *testing.T) {
		_, err := detectCSVDelimiter([]byte("a,b;c\n1,2;3\n4,5;6\n"))
		if err == nil || validationKeyFromError(err) != importValidationAmbiguousDelimiter {
			t.Fatalf("expected %q error, got %v", importValidationAmbiguousDelimiter, err)
		}
	})

	t.Run("defaults to comma when no candidates", func(t *testing.T) {
		delim, err := detectCSVDelimiter([]byte("only\none\ncol\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if delim != "," {
			t.Fatalf("expected ',', got %q", delim)
		}
	})
}

func TestParseCSVImport(t *testing.T) {
	opts := &model.ImportFileOptions{
		Format: model.ImportFileFormatCSV,
	}

	t.Run("invalid options", func(t *testing.T) {
		_, err := parseImportFile([]byte("a,b\n1,2\n"), nil, 0, false)
		if err == nil || validationKeyFromError(err) != importValidationInvalidOptions {
			t.Fatalf("expected %q error, got %v", importValidationInvalidOptions, err)
		}
	})

	t.Run("header validation", func(t *testing.T) {
		_, err := parseImportFile([]byte(",b\n1,2\n"), opts, 0, false)
		if err == nil || validationKeyFromError(err) != importValidationEmptyHeader {
			t.Fatalf("expected %q error, got %v", importValidationEmptyHeader, err)
		}
	})

	t.Run("duplicate header", func(t *testing.T) {
		_, err := parseImportFile([]byte("a,a\n1,2\n"), opts, 0, false)
		if err == nil || validationKeyFromError(err) != importValidationDuplicateHeader {
			t.Fatalf("expected %q error, got %v", importValidationDuplicateHeader, err)
		}
	})

	t.Run("row normalization and truncation", func(t *testing.T) {
		res, err := parseImportFile([]byte("a,b\n1\n2,3\n"), opts, 1, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.columns) != 2 || res.columns[0] != "a" {
			t.Fatalf("unexpected columns: %#v", res.columns)
		}
		if len(res.rows) != 1 || len(res.rows[0]) != 2 || res.rows[0][0] != "1" || res.rows[0][1] != "" {
			t.Fatalf("unexpected rows: %#v", res.rows)
		}
		if !res.truncated {
			t.Fatalf("expected truncated=true")
		}
	})

	t.Run("row too many columns", func(t *testing.T) {
		_, err := parseImportFile([]byte("a,b\n1,2,3\n"), opts, 0, false)
		if err == nil || validationKeyFromError(err) != importValidationRowTooManyColumns {
			t.Fatalf("expected %q error, got %v", importValidationRowTooManyColumns, err)
		}
	})

	t.Run("explicit invalid delimiter", func(t *testing.T) {
		delim := "="
		optsBad := &model.ImportFileOptions{
			Format:    model.ImportFileFormatCSV,
			Delimiter: &delim,
		}
		_, err := parseImportFile([]byte("a=b\n1=2\n"), optsBad, 0, false)
		if err == nil || validationKeyFromError(err) != importValidationInvalidDelimiter {
			t.Fatalf("expected %q error, got %v", importValidationInvalidDelimiter, err)
		}
	})
}

func TestParseExcelImport(t *testing.T) {
	file := excelize.NewFile()
	sheet := file.GetSheetName(file.GetActiveSheetIndex())
	if err := file.SetCellValue(sheet, "A1", "a"); err != nil {
		t.Fatalf("set cell failed: %v", err)
	}
	if err := file.SetCellValue(sheet, "B1", "b"); err != nil {
		t.Fatalf("set cell failed: %v", err)
	}
	if err := file.SetCellValue(sheet, "A2", "1"); err != nil {
		t.Fatalf("set cell failed: %v", err)
	}
	if err := file.SetCellValue(sheet, "B2", "2"); err != nil {
		t.Fatalf("set cell failed: %v", err)
	}

	buf, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("failed to serialize excel file: %v", err)
	}

	opts := &model.ImportFileOptions{Format: model.ImportFileFormatExcel}
	res, err := parseImportFile(buf.Bytes(), opts, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.sheet == "" {
		t.Fatalf("expected detected sheet name")
	}
	if len(res.columns) != 2 || res.columns[0] != "a" || res.columns[1] != "b" {
		t.Fatalf("unexpected columns: %#v", res.columns)
	}
	if len(res.rows) != 1 || len(res.rows[0]) != 2 || res.rows[0][0] != "1" || res.rows[0][1] != "2" {
		t.Fatalf("unexpected rows: %#v", res.rows)
	}
}

func TestResolveImportMappings(t *testing.T) {
	source := []string{"a", "b"}
	targetCols := []engine.Column{
		{Name: "col1", Type: "TEXT"},
		{Name: "col2", Type: "INT"},
	}

	col1 := "col1"
	col2 := "col2"
	mappings := []*model.ImportColumnMapping{
		{SourceColumn: "a", TargetColumn: &col1},
		{SourceColumn: "b", TargetColumn: &col2},
	}

	resolved, err := resolveImportMappings(source, mappings, targetCols, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 2 || resolved[0].sourceIndex != 0 || resolved[1].sourceIndex != 1 {
		t.Fatalf("unexpected mapping result: %#v", resolved)
	}

	t.Run("unknown target", func(t *testing.T) {
		unknown := "nope"
		_, err := resolveImportMappings(source, []*model.ImportColumnMapping{
			{SourceColumn: "a", TargetColumn: &unknown},
			{SourceColumn: "b", TargetColumn: &col2},
		}, targetCols, false)
		if err == nil || validationKeyFromError(err) != importValidationUnknownColumn {
			t.Fatalf("expected %q error, got %v", importValidationUnknownColumn, err)
		}
	})

	t.Run("computed target", func(t *testing.T) {
		computedCols := []engine.Column{
			{Name: "col1", Type: "TEXT", IsComputed: true},
			{Name: "col2", Type: "INT"},
		}
		_, err := resolveImportMappings(source, mappings, computedCols, false)
		if err == nil || validationKeyFromError(err) != importValidationGeneratedColumn {
			t.Fatalf("expected %q error, got %v", importValidationGeneratedColumn, err)
		}
	})

	t.Run("auto increment requires toggle", func(t *testing.T) {
		autoCols := []engine.Column{
			{Name: "col1", Type: "TEXT", IsAutoIncrement: true},
			{Name: "col2", Type: "INT"},
		}
		_, err := resolveImportMappings(source, mappings, autoCols, false)
		if err == nil || validationKeyFromError(err) != importValidationAutoGeneratedColumn {
			t.Fatalf("expected %q error, got %v", importValidationAutoGeneratedColumn, err)
		}
	})
}

func TestBuildImportMappingInputs(t *testing.T) {
	source := []string{"col1", "col2"}
	targetCols := []engine.Column{
		{Name: "col1", Type: "TEXT"},
		{Name: "col2", Type: "INT"},
	}

	t.Run("header mapping ok", func(t *testing.T) {
		mappings, auto, err := buildImportMappingInputs(source, targetCols, true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mappings) != 2 {
			t.Fatalf("expected 2 mappings, got %d", len(mappings))
		}
		if len(auto) != 0 {
			t.Fatalf("expected no auto columns, got %v", auto)
		}
	})

	t.Run("header mapping requires toggle", func(t *testing.T) {
		targetAuto := []engine.Column{
			{Name: "col1", Type: "TEXT", IsAutoIncrement: true},
			{Name: "col2", Type: "INT"},
		}
		mappings, auto, err := buildImportMappingInputs(source, targetAuto, true, false)
		if err == nil || validationKeyFromError(err) != importValidationAutoGeneratedToggle {
			t.Fatalf("expected %q error, got %v", importValidationAutoGeneratedToggle, err)
		}
		if len(mappings) != 2 || len(auto) != 1 || auto[0] != "col1" {
			t.Fatalf("unexpected return values: mappings=%v auto=%v", mappings, auto)
		}
	})

	t.Run("positional mapping count mismatch", func(t *testing.T) {
		_, _, err := buildImportMappingInputs([]string{"a"}, targetCols, false, false)
		if err == nil || validationKeyFromError(err) != importValidationColumnCountMismatch {
			t.Fatalf("expected %q error, got %v", importValidationColumnCountMismatch, err)
		}
	})

	t.Run("positional mapping insertable mismatch surfaces auto columns", func(t *testing.T) {
		targetAuto := []engine.Column{
			{Name: "id", Type: "INT", IsAutoIncrement: true},
			{Name: "col1", Type: "TEXT"},
			{Name: "col2", Type: "INT"},
		}

		// Source includes the auto-generated id column; positional mapping cannot
		// safely map this without enabling the auto-generated toggle.
		sourceWithID := []string{"id", "col1", "col2"}
		_, auto, err := buildImportMappingInputs(sourceWithID, targetAuto, false, false)
		if err == nil || validationKeyFromError(err) != importValidationColumnCountInsertable {
			t.Fatalf("expected %q error, got %v", importValidationColumnCountInsertable, err)
		}
		if len(auto) != 1 || auto[0] != "id" {
			t.Fatalf("expected auto columns to be returned, got %v", auto)
		}
	})
}
