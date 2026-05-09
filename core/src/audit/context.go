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
	"sync"
)

type requestContextKey struct{}
type scopeContextKey struct{}

type scopeState struct {
	mu    sync.RWMutex
	scope Scope
}

// Scope describes organization, project, and source identifiers associated
// with an audited action.
type Scope struct {
	OrgID     string
	ProjectID string
	SourceID  string
}

// WithRequest stores request metadata inside the supplied context.
func WithRequest(ctx context.Context, request Request) context.Context {
	return context.WithValue(ctx, requestContextKey{}, mergeRequest(RequestFromContext(ctx), request))
}

// RequestFromContext extracts request metadata from the supplied context.
func RequestFromContext(ctx context.Context) Request {
	value := ctx.Value(requestContextKey{})
	request, ok := value.(Request)
	if !ok {
		return Request{}
	}
	return request
}

// WithScope stores audit scope metadata inside the supplied context.
func WithScope(ctx context.Context, scope Scope) context.Context {
	if state, ok := ctx.Value(scopeContextKey{}).(*scopeState); ok && state != nil {
		state.merge(scope)
		return ctx
	}

	return context.WithValue(ctx, scopeContextKey{}, &scopeState{
		scope: mergeScope(ScopeFromContext(ctx), scope),
	})
}

// WithIsolatedScope stores audit scope metadata inside a fresh mutable scope state.
// Child operations can extend this scope without mutating parent contexts.
func WithIsolatedScope(ctx context.Context, scope Scope) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, &scopeState{
		scope: mergeScope(ScopeFromContext(ctx), scope),
	})
}

// ScopeFromContext extracts audit scope metadata from the supplied context.
func ScopeFromContext(ctx context.Context) Scope {
	value := ctx.Value(scopeContextKey{})
	switch typed := value.(type) {
	case Scope:
		return typed
	case *scopeState:
		if typed == nil {
			return Scope{}
		}
		typed.mu.RLock()
		defer typed.mu.RUnlock()
		return typed.scope
	default:
		return Scope{}
	}
}

func mergeRequest(base Request, override Request) Request {
	request := base

	if strings.TrimSpace(override.ID) != "" {
		request.ID = strings.TrimSpace(override.ID)
	}
	if strings.TrimSpace(override.Host) != "" {
		request.Host = strings.TrimSpace(override.Host)
	}
	if strings.TrimSpace(override.Method) != "" {
		request.Method = strings.TrimSpace(override.Method)
	}
	if strings.TrimSpace(override.Path) != "" {
		request.Path = strings.TrimSpace(override.Path)
	}
	if strings.TrimSpace(override.Route) != "" {
		request.Route = strings.TrimSpace(override.Route)
	}
	if strings.TrimSpace(override.RemoteIP) != "" {
		request.RemoteIP = strings.TrimSpace(override.RemoteIP)
	}
	if strings.TrimSpace(override.UserAgent) != "" {
		request.UserAgent = strings.TrimSpace(override.UserAgent)
	}
	if strings.TrimSpace(override.Protocol) != "" {
		request.Protocol = strings.TrimSpace(override.Protocol)
	}
	if strings.TrimSpace(override.OperationName) != "" {
		request.OperationName = strings.TrimSpace(override.OperationName)
	}
	if strings.TrimSpace(override.OperationType) != "" {
		request.OperationType = strings.TrimSpace(override.OperationType)
	}
	if strings.TrimSpace(override.TraceID) != "" {
		request.TraceID = strings.TrimSpace(override.TraceID)
	}
	if strings.TrimSpace(override.SpanID) != "" {
		request.SpanID = strings.TrimSpace(override.SpanID)
	}

	return request
}

func mergeScope(base Scope, override Scope) Scope {
	scope := base

	if strings.TrimSpace(override.OrgID) != "" {
		scope.OrgID = strings.TrimSpace(override.OrgID)
	}
	if strings.TrimSpace(override.ProjectID) != "" {
		scope.ProjectID = strings.TrimSpace(override.ProjectID)
	}
	if strings.TrimSpace(override.SourceID) != "" {
		scope.SourceID = strings.TrimSpace(override.SourceID)
	}

	return scope
}

func (s *scopeState) merge(scope Scope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scope = mergeScope(s.scope, scope)
}
