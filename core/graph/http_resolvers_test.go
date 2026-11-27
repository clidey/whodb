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
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"

	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
)

func TestRESTExportSuccessAndFailure(t *testing.T) {
	successMock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	successMock.ExportDataFunc = func(*engine.PluginConfig, string, string, engine.ExportWriter, *engine.ExportSelection) error {
		return nil
	}
	setEngineMock(t, successMock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	payload := `{"ref":{"Kind":"Table","Path":["app","public","users"]},"selectedRows":[{"id":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/export", bytes.NewBufferString(payload))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected export to succeed, got %d", rec.Code)
	}

	unsupportedPayload := `{"ref":{"Kind":"Schema","Path":["app","public"]},"selectedRows":[{"id":1}]}`
	req = httptest.NewRequest(http.MethodPost, "/api/export", bytes.NewBufferString(unsupportedPayload))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "viewing rows is not supported") {
		t.Fatalf("expected export action rejection, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Failure case
	failMock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	failMock.ExportDataFunc = func(*engine.PluginConfig, string, string, engine.ExportWriter, *engine.ExportSelection) error {
		return errors.New("export failed")
	}
	setEngineMock(t, failMock)

	req = httptest.NewRequest(http.MethodPost, "/api/export", bytes.NewBufferString(payload))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected export failure to return 500, got %d", rec.Code)
	}
}

func exportTestRouter(t *testing.T, export func(writer engine.ExportWriter, selection *engine.ExportSelection) error) http.Handler {
	t.Helper()

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.ExportDataFunc = func(_ *engine.PluginConfig, _ string, _ string, writer engine.ExportWriter, selection *engine.ExportSelection) error {
		return export(writer, selection)
	}
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), auth.AuthKey_Source, auth.GetSourceCredentials(testSourceContext("Postgres", map[string]string{"Database": "app"})))
		router.ServeHTTP(w, r.WithContext(ctx))
	})
}

func postExport(t *testing.T, handler http.Handler, payload string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/export", bytes.NewBufferString(payload)))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected export to succeed, got %d (%s)", rec.Code, rec.Body.String())
	}
	return rec
}

func TestRESTExportCSVGuardsOnlyFreeTextCells(t *testing.T) {
	handler := exportTestRouter(t, func(writer engine.ExportWriter, _ *engine.ExportSelection) error {
		if err := writer([]string{"amount", "note", "=header"}, nil); err != nil {
			return err
		}
		return writer([]string{"-1200.50", "=1+1", "-1 days"}, []bool{false, true, false})
	})

	rec := postExport(t, handler, `{"ref":{"Kind":"Table","Path":["app","public","orders"]}}`)
	if got, want := rec.Body.String(), "amount,note,'=header\n-1200.50,'=1+1,-1 days\n"; got != want {
		t.Fatalf("unexpected CSV:\n got %q\nwant %q", got, want)
	}
}

func TestRESTExportExcelWritesValuesUnchanged(t *testing.T) {
	handler := exportTestRouter(t, func(writer engine.ExportWriter, _ *engine.ExportSelection) error {
		if err := writer([]string{"amount", "note"}, nil); err != nil {
			return err
		}
		return writer([]string{"-1200.50", "=1+1"}, []bool{false, true})
	})

	rec := postExport(t, handler, `{"ref":{"Kind":"Table","Path":["app","public","orders"]},"format":"excel"}`)
	workbook, err := excelize.OpenReader(rec.Body)
	if err != nil {
		t.Fatalf("failed to open exported workbook: %v", err)
	}
	defer func() { _ = workbook.Close() }()

	rows, err := workbook.GetRows("Data")
	if err != nil {
		t.Fatalf("failed to read exported sheet: %v", err)
	}
	if len(rows) != 2 || rows[1][0] != "-1200.50" || rows[1][1] != "=1+1" {
		t.Fatalf("expected values exactly as written, got %#v", rows)
	}
	if formula, _ := workbook.GetCellFormula("Data", "B2"); formula != "" {
		t.Fatalf("expected a text cell, got formula %q", formula)
	}
}

func TestRESTExportSelectedRowsKeepGridColumns(t *testing.T) {
	var received *engine.ExportSelection
	handler := exportTestRouter(t, func(_ engine.ExportWriter, selection *engine.ExportSelection) error {
		received = selection
		return nil
	})
	columns := `"columns":[{"name":"total","type":"NUMERIC(12,2)"},{"name":"customer","type":"TEXT"},{"name":"age","type":"INT4"}]`
	rows := `"selectedRows":[{"customer":"=cmd","age":"-3","total":"-9.50"},{"customer":"+15551234567","age":null,"total":"1.00"}]`

	// Without an object the rows are written by the handler itself.
	rec := postExport(t, handler, `{"fileBaseName":"query",`+columns+`,`+rows+`}`)
	if got, want := rec.Body.String(), "total,customer,age\n-9.50,'=cmd,-3\n1.00,'+15551234567,\n"; got != want {
		t.Fatalf("unexpected CSV:\n got %q\nwant %q", got, want)
	}

	// With an object the plugin receives the same ordered, typed columns.
	postExport(t, handler, `{"ref":{"Kind":"Table","Path":["app","public","orders"]},`+columns+`,`+rows+`}`)
	if received == nil || len(received.Rows) != 2 || len(received.Columns) != 3 ||
		received.Columns[0].Name != "total" || received.Columns[0].Type != "NUMERIC(12,2)" || received.Columns[2].Name != "age" {
		t.Fatalf("expected the plugin to receive the grid's columns, got %#v", received)
	}
}

func TestRESTExportCSVFailureIsNotAShortFile(t *testing.T) {
	handler := exportTestRouter(t, func(writer engine.ExportWriter, _ *engine.ExportSelection) error {
		for range 250 {
			if err := writer([]string{"value"}, nil); err != nil {
				return err
			}
		}
		return errors.New("connection lost")
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := http.Post(server.URL+"/api/export", "application/json", bytes.NewBufferString(`{"ref":{"Kind":"Table","Path":["app","public","orders"]}}`))
	if err != nil {
		return
	}
	defer func() { _ = response.Body.Close() }()
	if _, err := io.ReadAll(response.Body); err == nil {
		t.Fatal("expected a failed export to break the download instead of ending it cleanly")
	}
}
