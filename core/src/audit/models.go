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

package audit

import (
	"time"
)

// SchemaVersion is the current audit event schema version.
const SchemaVersion = 3

// Severity represents the importance of an audit event.
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarn     Severity = "WARN"
	SeverityCritical Severity = "CRITICAL"
)

// Outcome represents the final result of an audited action.
type Outcome string

const (
	OutcomeSuccess Outcome = "SUCCESS"
	OutcomeFailure Outcome = "FAILURE"
	OutcomeDenied  Outcome = "DENIED"
)

// AuditEvent captures a single auditable action within the system.
type AuditEvent struct {
	ID            string         `json:"id"`
	SchemaVersion int            `json:"schema_version"`
	Timestamp     time.Time      `json:"timestamp"`
	Actor         Actor          `json:"actor"`
	Request       Request        `json:"request"`
	OrgID         string         `json:"org_id,omitempty"`
	ProjectID     string         `json:"project_id,omitempty"`
	Action        string         `json:"action"`
	Outcome       Outcome        `json:"outcome"`
	Resource      Resource       `json:"resource"`
	Details       map[string]any `json:"details,omitempty"`
	Error         string         `json:"error,omitempty"`
	Duration      time.Duration  `json:"duration,omitempty"`
	Severity      Severity       `json:"severity"`
}

// Actor describes who or what performed the action.
type Actor struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // "user", "system", "api_key"
	Email     string `json:"email,omitempty"`
	IP        string `json:"ip,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// Request describes the request envelope for the audited action.
type Request struct {
	ID            string `json:"id,omitempty"`
	Host          string `json:"host,omitempty"`
	Method        string `json:"method,omitempty"`
	Path          string `json:"path,omitempty"`
	Route         string `json:"route,omitempty"`
	RemoteIP      string `json:"remote_ip,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	OperationName string `json:"operation_name,omitempty"`
	OperationType string `json:"operation_type,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
	SpanID        string `json:"span_id,omitempty"`
}

// Resource describes what the action was performed on.
type Resource struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "source", "role", "user", "config"
	Name string `json:"name,omitempty"`
}
