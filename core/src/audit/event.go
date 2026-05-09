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
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

func prepareEvent(ctx context.Context, event AuditEvent, actorProvider ActorProvider, eventEnricher EventEnricher) AuditEvent {
	if strings.TrimSpace(event.ID) == "" {
		event.ID = uuid.NewString()
	}
	if event.SchemaVersion == 0 {
		event.SchemaVersion = SchemaVersion
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	event.Request = mergeRequest(RequestFromContext(ctx), event.Request)
	if event.Request.TraceID == "" || event.Request.SpanID == "" {
		spanContext := trace.SpanContextFromContext(ctx)
		if spanContext.IsValid() {
			if event.Request.TraceID == "" {
				event.Request.TraceID = spanContext.TraceID().String()
			}
			if event.Request.SpanID == "" {
				event.Request.SpanID = spanContext.SpanID().String()
			}
		}
	}

	if event.Actor == (Actor{}) {
		event.Actor = actorProvider(ctx)
	}
	if event.Actor.IP == "" {
		event.Actor.IP = event.Request.RemoteIP
	}

	scope := ScopeFromContext(ctx)
	if event.OrgID == "" {
		event.OrgID = strings.TrimSpace(scope.OrgID)
	}
	if event.ProjectID == "" {
		event.ProjectID = strings.TrimSpace(scope.ProjectID)
	}

	event = eventEnricher(ctx, event)
	event.OrgID = strings.TrimSpace(event.OrgID)
	event.ProjectID = strings.TrimSpace(event.ProjectID)
	if event.OrgID != "" || event.ProjectID != "" {
		WithScope(ctx, Scope{
			OrgID:     event.OrgID,
			ProjectID: event.ProjectID,
		})
	}

	event.Error = strings.TrimSpace(event.Error)
	if event.Error == "" {
		if rawError, ok := event.Details["error"].(string); ok {
			event.Error = strings.TrimSpace(rawError)
		}
	}

	if event.Outcome == "" {
		switch {
		case event.Severity == SeverityCritical:
			event.Outcome = OutcomeFailure
		case event.Error != "":
			event.Outcome = OutcomeFailure
		default:
			event.Outcome = OutcomeSuccess
		}
	}

	if event.Severity == "" {
		switch event.Outcome {
		case OutcomeFailure, OutcomeDenied:
			event.Severity = SeverityWarn
		default:
			event.Severity = SeverityInfo
		}
	}

	return event
}
