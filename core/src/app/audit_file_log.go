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

package app

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/clidey/whodb/core/src/audit"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

// fileAuditService appends each audit event as one JSON object per line.
// Telemetry-class events are skipped. Writes are best-effort: a failed write
// is logged and the event is dropped.
type fileAuditService struct {
	mu sync.Mutex
	w  io.WriteCloser
}

func (s *fileAuditService) Record(event audit.AuditEvent) {
	if audit.Classify(event.Action) == audit.ClassTelemetry {
		return
	}
	line, err := json.Marshal(event)
	if err != nil {
		log.WithError(err).Warnf("audit file log: failed to encode %s event", event.Action)
		return
	}
	line = append(line, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.w.Write(line); err != nil {
		log.WithError(err).Warnf("audit file log: failed to write %s event", event.Action)
	}
}

func (s *fileAuditService) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Close()
}

// initializeAuditFileLog registers a file audit service when
// WHODB_AUDIT_LOG_FILE is set ("default" uses env.DefaultAuditLogFile) and
// returns a function that unregisters it and closes the file. When the
// variable is unset it registers nothing. The process exits if a configured
// file cannot be opened, matching the other log files.
func initializeAuditFileLog() func() {
	path := log.ResolveLogPath(env.AuditLogFile, env.DefaultAuditLogFile)
	if path == "" {
		return func() {}
	}
	service := &fileAuditService{w: log.OpenLogFile(path)}
	audit.SetAuditService(service)
	return func() {
		audit.SetAuditService(nil)
		if err := service.close(); err != nil {
			log.WithError(err).Warn("audit file log: failed to close")
		}
	}
}
