/*
 * Copyright 2025 Clidey, Inc.
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

package router

import (
	"context"
	"net/http"
	"time"

	"github.com/clidey/whodb/core/src/analytics"
	coreaudit "github.com/clidey/whodb/core/src/audit"
	"github.com/clidey/whodb/core/src/common"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func contextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), common.RouterKey_ResponseWriter, w)
		ctx = propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(r.Header))

		metadata := analytics.BuildMetadata(r)
		if metadata.RequestID == "" {
			if requestID := middleware.GetReqID(ctx); requestID != "" {
				metadata.RequestID = requestID
			}
		}

		request := coreaudit.Request{
			ID:        metadata.RequestID,
			Host:      r.Host,
			Method:    r.Method,
			Path:      r.URL.Path,
			RemoteIP:  r.RemoteAddr,
			UserAgent: metadata.UserAgent,
			Protocol:  r.Proto,
		}
		spanContext := trace.SpanContextFromContext(ctx)
		if spanContext.IsValid() {
			request.TraceID = spanContext.TraceID().String()
			request.SpanID = spanContext.SpanID().String()
		}

		ctx = analytics.WithMetadata(ctx, metadata)
		ctx = coreaudit.WithRequest(ctx, request)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func auditHTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		ctx := coreaudit.WithIsolatedScope(r.Context(), coreaudit.Scope{})
		next.ServeHTTP(sw, r.WithContext(ctx))

		route := ""
		if routeCtx := chi.RouteContext(ctx); routeCtx != nil {
			route = routeCtx.RoutePattern()
		}

		ctx = coreaudit.WithRequest(ctx, coreaudit.Request{Route: route})
		outcome := coreaudit.OutcomeSuccess
		severity := coreaudit.SeverityInfo
		switch {
		case sw.statusCode >= http.StatusInternalServerError:
			outcome = coreaudit.OutcomeFailure
			severity = coreaudit.SeverityWarn
		case sw.statusCode >= http.StatusBadRequest:
			outcome = coreaudit.OutcomeDenied
			severity = coreaudit.SeverityWarn
		}

		resourceID := route
		if resourceID == "" {
			resourceID = r.URL.Path
		}

		coreaudit.RecordWithContext(ctx, coreaudit.AuditEvent{
			Timestamp: start,
			Action:    "http.request",
			Outcome:   outcome,
			Severity:  severity,
			Resource: coreaudit.Resource{
				ID:   resourceID,
				Type: "http_route",
				Name: route,
			},
			Details: map[string]any{
				"status_code": sw.statusCode,
			},
			Duration: time.Since(start),
		})
	})
}
