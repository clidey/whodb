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
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/clidey/whodb/core/src/audit"
	"github.com/clidey/whodb/core/src/env"
)

func TestAuditFileLogWritesNonTelemetryEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit", "whodb.audit.log")
	original := env.AuditLogFile
	env.AuditLogFile = path
	t.Cleanup(func() { env.AuditLogFile = original })

	closeLog := initializeAuditFileLog()
	audit.Record(audit.AuditEvent{Action: "http.request"})
	audit.Record(audit.AuditEvent{Action: "login.source"})
	closeLog()

	// Recorded after close: must not reach the file.
	audit.Record(audit.AuditEvent{Action: "logout.source"})

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open audit log: %v", err)
	}
	defer file.Close()

	var events []audit.AuditEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event audit.AuditEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode line %q: %v", scanner.Text(), err)
		}
		events = append(events, event)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 audit line, got %d", len(events))
	}
	if events[0].Action != "login.source" || events[0].ID == "" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 audit log, got %v (err %v)", info.Mode().Perm(), err)
	}
}
