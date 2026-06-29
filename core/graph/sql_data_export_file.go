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
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
)

const sqlDataExportContentType = "application/sql; charset=utf-8"

var sqlDataExportFiles = newSQLDataExportFileStore(30 * time.Minute)

type sqlDataExportFile struct {
	ID          string
	Path        string
	Filename    string
	ContentType string
	OwnerKey    string
	Size        int64
	CreatedAt   time.Time
}

type sqlDataExportFileStore struct {
	mu    sync.Mutex
	ttl   time.Duration
	files map[string]*sqlDataExportFile
}

func newSQLDataExportFileStore(ttl time.Duration) *sqlDataExportFileStore {
	return &sqlDataExportFileStore{
		ttl:   ttl,
		files: make(map[string]*sqlDataExportFile),
	}
}

func (s *sqlDataExportFileStore) put(file *sqlDataExportFile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())
	s.files[file.ID] = file
}

func (s *sqlDataExportFileStore) get(id string) (*sqlDataExportFile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, ok := s.files[id]
	if !ok {
		return nil, false
	}
	if time.Since(file.CreatedAt) > s.ttl {
		delete(s.files, id)
		_ = os.Remove(file.Path)
		return nil, false
	}
	copy := *file
	return &copy, true
}

func (s *sqlDataExportFileStore) pruneLocked(now time.Time) {
	for id, file := range s.files {
		if now.Sub(file.CreatedAt) <= s.ttl {
			continue
		}
		delete(s.files, id)
		_ = os.Remove(file.Path)
	}
}

func createSQLDataExportFile(plugin *engine.Plugin, config *engine.PluginConfig, req *engine.SQLDataExportRequest) (*sqlDataExportFile, error) {
	if plugin == nil {
		return nil, fmt.Errorf("database plugin not found")
	}
	if config == nil || config.Credentials == nil {
		return nil, fmt.Errorf("database credentials not found")
	}

	file, err := os.CreateTemp("", "whodb-sql-data-export-*.sql")
	if err != nil {
		return nil, err
	}
	path := file.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(path)
		}
	}()

	if err := plugin.ExportSQLData(config, req, file); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	exportFile := &sqlDataExportFile{
		ID:          uuid.NewString(),
		Path:        path,
		Filename:    sqlDataExportFilename(req),
		ContentType: sqlDataExportContentType,
		OwnerKey:    sqlDataExportOwnerKey(config.Credentials),
		Size:        stat.Size(),
		CreatedAt:   time.Now(),
	}
	sqlDataExportFiles.put(exportFile)
	success = true
	return exportFile, nil
}

func handleSQLDataExportDownload(w http.ResponseWriter, r *http.Request) {
	credentials := auth.GetCredentials(r.Context())
	if credentials == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	file, ok := sqlDataExportFiles.get(id)
	if !ok {
		http.Error(w, "Export not found", http.StatusNotFound)
		return
	}
	if file.OwnerKey != sqlDataExportOwnerKey(credentials) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	content, err := os.Open(file.Path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Export not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to open export", http.StatusInternalServerError)
		return
	}
	defer content.Close()

	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", escapeHeaderFilename(file.Filename)))
	w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	http.ServeContent(w, r, file.Filename, file.CreatedAt, content)
}

func validateSQLDataExportTable(plugin *engine.Plugin, config *engine.PluginConfig, schema string, storageUnit string) error {
	units, err := plugin.GetStorageUnits(config, schema)
	if err != nil {
		return err
	}
	for _, unit := range units {
		if unit.Name != storageUnit {
			continue
		}
		if storageUnitIsView(unit) {
			return fmt.Errorf("SQL Data Export supports SQL Tables, not SQL Views")
		}
		return nil
	}
	return fmt.Errorf("storage unit %s not found", storageUnit)
}

func storageUnitIsView(unit engine.StorageUnit) bool {
	for _, attribute := range unit.Attributes {
		if !strings.EqualFold(attribute.Key, "Type") {
			continue
		}
		return strings.Contains(strings.ToUpper(attribute.Value), "VIEW")
	}
	return false
}

func sqlDataExportFilename(req *engine.SQLDataExportRequest) string {
	mode := strings.ToLower(string(req.Mode))
	if mode == "" {
		mode = "export"
	}
	return sanitizeExportFilename(req.StorageUnit) + "_" + mode + ".sql"
}

func sanitizeExportFilename(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.', r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}
	name := strings.Trim(builder.String(), "._-")
	if name == "" {
		return "export"
	}
	return name
}

func escapeHeaderFilename(value string) string {
	return strings.ReplaceAll(value, `"`, "'")
}

func sqlDataExportOwnerKey(credentials *engine.Credentials) string {
	id := ""
	if credentials.Id != nil {
		id = *credentials.Id
	}
	return strings.Join([]string{
		id,
		credentials.Type,
		credentials.Hostname,
		credentials.Username,
		credentials.Database,
	}, "\x00")
}
