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

package mcp

import (
	"errors"
	"strings"
	"testing"
	"time"

	coreaudit "github.com/clidey/whodb/core/src/audit"
)

type capturingAuditService struct{ events []coreaudit.AuditEvent }

func (s *capturingAuditService) Record(event coreaudit.AuditEvent) {
	s.events = append(s.events, event)
}

func TestRecordQueryAuditNeverCarriesSQL(t *testing.T) {
	capture := &capturingAuditService{}
	coreaudit.SetAuditService(capture)
	t.Cleanup(func() { coreaudit.SetAuditService(nil) })

	start := time.Now().Add(-time.Second)
	recordQueryAudit(start, "prod-pg", "Postgres", StatementType("SELECT"), 12, false, nil)
	recordQueryAudit(start, "prod-pg", "Postgres", StatementType("DELETE"), 0, true, errors.New("relation \"secret_table\" does not exist"))

	if len(capture.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(capture.events))
	}
	read, write := capture.events[0], capture.events[1]
	if read.Action != "mcp.query.execute" || read.Outcome != coreaudit.OutcomeSuccess || read.Resource.ID != "prod-pg" || read.Details["row_count"] != 12 || read.Details["confirmed"] != false {
		t.Fatalf("unexpected read event: %+v", read)
	}
	if write.Outcome != coreaudit.OutcomeFailure || write.Details["confirmed"] != true || write.Details["statement_type"] != "DELETE" || write.Error == "" {
		t.Fatalf("unexpected write event: %+v", write)
	}
	for key := range read.Details {
		if strings.Contains(strings.ToLower(key), "query") || strings.Contains(strings.ToLower(key), "sql") {
			t.Fatalf("details must not carry statement text, found key %q", key)
		}
	}
	if coreaudit.Classify(read.Action) != coreaudit.ClassAccess {
		t.Fatalf("mcp.query.execute must be an access-class event, got %s", coreaudit.Classify(read.Action))
	}
}
