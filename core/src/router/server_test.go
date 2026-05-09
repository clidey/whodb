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

package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	graphapi "github.com/clidey/whodb/core/graph"
	coreaudit "github.com/clidey/whodb/core/src/audit"
	"github.com/clidey/whodb/core/src/env"
	"github.com/go-chi/chi/v5"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestHealthCheckMiddlewareShortCircuitsHandler(t *testing.T) {
	called := false
	handler := healthCheckMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected health middleware to return 200, got %d", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Fatalf("expected health body 'ok', got %q", rr.Body.String())
	}
	if called {
		t.Fatal("expected health middleware to bypass the wrapped handler")
	}
}

func TestSetupMiddlewaresPublicPathsBypassAuth(t *testing.T) {
	router := chi.NewRouter()
	setupMiddlewares(router, nil, []string{"/api/public"})

	router.Get("/api/public", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	router.Get("/api/private", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	publicReq := httptest.NewRequest(http.MethodGet, "/api/public", nil)
	publicRes := httptest.NewRecorder()
	router.ServeHTTP(publicRes, publicReq)
	if publicRes.Code != http.StatusNoContent {
		t.Fatalf("expected public path to bypass auth, got %d", publicRes.Code)
	}

	privateReq := httptest.NewRequest(http.MethodGet, "/api/private", nil)
	privateRes := httptest.NewRecorder()
	router.ServeHTTP(privateRes, privateReq)
	if privateRes.Code != http.StatusUnauthorized {
		t.Fatalf("expected private path to require auth, got %d", privateRes.Code)
	}
}

func TestNewGraphQLServerTogglesIntrospectionByEnvironment(t *testing.T) {
	queryBody, err := json.Marshal(map[string]any{
		"query": `query IntrospectionQuery { __schema { queryType { name } } }`,
	})
	if err != nil {
		t.Fatalf("failed to build request body: %v", err)
	}

	runQuery := func(isDevelopment bool) string {
		originalDev := env.IsDevelopment
		env.IsDevelopment = isDevelopment
		defer func() { env.IsDevelopment = originalDev }()

		server := NewGraphQLServer(graphapi.NewExecutableSchema(graphapi.Config{Resolvers: &graphapi.Resolver{}}))
		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(queryBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected GraphQL HTTP 200, got %d with body %s", rr.Code, rr.Body.String())
		}
		return rr.Body.String()
	}

	devBody := runQuery(true)
	if !strings.Contains(devBody, "__schema") {
		t.Fatalf("expected introspection data in development mode, got %s", devBody)
	}

	prodBody := runQuery(false)
	if !strings.Contains(prodBody, "errors") {
		t.Fatalf("expected introspection to be rejected outside development mode, got %s", prodBody)
	}
}

func TestNewGraphQLServerAuditsGraphQLRootFieldName(t *testing.T) {
	service := &capturingAuditService{}
	coreaudit.SetAuditService(service)
	coreaudit.SetActorProvider(nil)
	t.Cleanup(func() {
		coreaudit.SetAuditService(nil)
		coreaudit.SetActorProvider(nil)
	})

	server := NewGraphQLServer(graphapi.NewExecutableSchema(graphapi.Config{Resolvers: &graphapi.Resolver{}}))

	queryBody, err := json.Marshal(map[string]any{
		"query": `query GetHealth { Health { Server Database } }`,
	})
	if err != nil {
		t.Fatalf("failed to build request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(queryBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected GraphQL HTTP 200, got %d with body %s", rr.Code, rr.Body.String())
	}

	events := service.Events()
	if len(events) != 1 {
		t.Fatalf("expected exactly one root-field audit event, got %d", len(events))
	}

	event := events[0]
	if event.Action != "graphql.query.Health" {
		t.Fatalf("expected action graphql.query.Health, got %s", event.Action)
	}
	if event.Resource.ID != "Health" {
		t.Fatalf("expected resource id Health, got %s", event.Resource.ID)
	}
	if event.Request.OperationName != "GetHealth" {
		t.Fatalf("expected operation name GetHealth, got %s", event.Request.OperationName)
	}
	if path, ok := event.Details["path"].(string); !ok || path != "Health" {
		t.Fatalf("expected detail path Health, got %#v", event.Details["path"])
	}
}

func TestGraphQLAuditScopeFindsNestedProjectAndSourceIDs(t *testing.T) {
	scope := graphQLAuditScope(map[string]any{
		"input": map[string]any{
			"projectId": "project-123",
			"filters": map[string]any{
				"sourceId": "source-456",
			},
		},
	})

	if scope.ProjectID != "project-123" {
		t.Fatalf("expected nested project id, got %q", scope.ProjectID)
	}
	if scope.SourceID != "source-456" {
		t.Fatalf("expected nested source id, got %q", scope.SourceID)
	}
}

func TestGraphQLAuditScopeFindsStructProjectID(t *testing.T) {
	type nested struct {
		ProjectID string
	}

	scope := graphQLAuditScope(map[string]any{
		"input": nested{ProjectID: "project-789"},
	})

	if scope.ProjectID != "project-789" {
		t.Fatalf("expected struct project id, got %q", scope.ProjectID)
	}
}

func TestGraphQLAuditArgumentsUsesRootFieldArgumentsBeforeFieldContextArgsExist(t *testing.T) {
	rootFieldCtx := &graphql.RootFieldContext{
		Object: "ProjectSources",
		Field: graphql.CollectedField{
			Field: &ast.Field{
				Name: "ProjectSources",
				Definition: &ast.FieldDefinition{
					Name: "ProjectSources",
					Arguments: ast.ArgumentDefinitionList{
						&ast.ArgumentDefinition{Name: "projectId"},
						&ast.ArgumentDefinition{Name: "sourceId"},
					},
				},
				Arguments: ast.ArgumentList{
					&ast.Argument{
						Name: "projectId",
						Value: &ast.Value{
							Raw:  "projectId",
							Kind: ast.Variable,
						},
					},
					&ast.Argument{
						Name: "sourceId",
						Value: &ast.Value{
							Raw:  "sourceId",
							Kind: ast.Variable,
						},
					},
				},
			},
		},
	}

	args := graphQLAuditArguments(rootFieldCtx, &graphql.OperationContext{
		Variables: map[string]any{
			"projectId": "project-123",
			"sourceId":  "source-456",
		},
	}, nil)
	scope := graphQLAuditScope(args)

	if got := args["projectId"]; got != "project-123" {
		t.Fatalf("expected projectId arg from root field, got %#v", got)
	}
	if got := scope.ProjectID; got != "project-123" {
		t.Fatalf("expected project id from root field args, got %q", got)
	}
	if got := scope.SourceID; got != "source-456" {
		t.Fatalf("expected source id from root field args, got %q", got)
	}
}

type capturingAuditService struct {
	mu     sync.Mutex
	events []coreaudit.AuditEvent
}

func (s *capturingAuditService) Record(event coreaudit.AuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *capturingAuditService) Events() []coreaudit.AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]coreaudit.AuditEvent(nil), s.events...)
}
