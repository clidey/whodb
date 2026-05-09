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
	"context"
	"embed"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler/extension"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/clidey/whodb/core/graph"
	coreaudit "github.com/clidey/whodb/core/src/audit"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type OAuthLoginUrl struct {
	Url string `json:"url"`
}

func NewGraphQLServer(es graphql.ExecutableSchema) *handler.Server {
	srv := handler.New(es)

	srv.AddTransport(&transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(&transport.Options{})
	srv.AddTransport(&transport.GET{})
	srv.AddTransport(&transport.POST{})
	srv.AddTransport(&transport.MultipartForm{
		MaxUploadSize: 250 * 1024 * 1024, // 250 MB
	})

	srv.Use(extension.FixedComplexityLimit(100))

	if env.IsDevelopment {
		srv.Use(extension.Introspection{})
	}

	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		opCtx := graphql.GetOperationContext(ctx)
		if opCtx != nil && opCtx.Operation != nil {
			ctx = coreaudit.WithRequest(ctx, coreaudit.Request{
				OperationName: graphQLOperationName(opCtx),
				OperationType: strings.ToLower(string(opCtx.Operation.Operation)),
			})
		}
		return next(ctx)
	})

	srv.AroundRootFields(func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
		start := time.Now()
		rootFieldCtx := graphql.GetRootFieldContext(ctx)
		fc := graphql.GetFieldContext(ctx)
		opCtx := graphql.GetOperationContext(ctx)
		rootArgs := graphQLAuditArguments(rootFieldCtx, opCtx, fc)

		fieldName := "unknown"
		objectName := ""
		details := map[string]any{}
		if rootFieldCtx != nil && rootFieldCtx.Field.Field != nil {
			fieldName = strings.TrimSpace(rootFieldCtx.Field.Name)
			if alias := strings.TrimSpace(rootFieldCtx.Field.Alias); alias != "" {
				details["path"] = alias
				if alias != fieldName {
					details["field_alias"] = alias
				}
			}
		}
		if fc != nil {
			if fieldName == "unknown" && fc.Field.Field != nil {
				fieldName = strings.TrimSpace(fc.Field.Name)
			}
			objectName = strings.TrimSpace(fc.Object)
			if _, ok := details["path"]; !ok {
				details["path"] = fmt.Sprintf("%v", fc.Path())
			}
		}
		details["arg_keys"] = sortedGraphQLArgKeys(rootArgs)
		ctx = coreaudit.WithIsolatedScope(ctx, graphQLAuditScope(rootArgs))
		if opCtx != nil && opCtx.Operation != nil {
			details["operation_name"] = graphQLOperationName(opCtx)
			details["operation_type"] = strings.ToLower(string(opCtx.Operation.Operation))
			if objectName == "" {
				objectName = strings.TrimSpace(string(opCtx.Operation.Operation))
			}
		}

		marshaler := next(ctx)

		outcome := coreaudit.OutcomeSuccess
		severity := coreaudit.SeverityInfo
		errorMessage := ""
		if fc != nil && graphql.HasFieldError(ctx, fc) {
			outcome = coreaudit.OutcomeFailure
			severity = coreaudit.SeverityWarn
			if errs := graphql.GetFieldErrors(ctx, fc); len(errs) > 0 {
				errorMessage = errs.Error()
			}
		}

		actionPrefix := "graphql.operation"
		if opCtx != nil && opCtx.Operation != nil {
			actionPrefix = "graphql." + strings.ToLower(string(opCtx.Operation.Operation))
		}

		coreaudit.RecordWithContext(ctx, coreaudit.AuditEvent{
			Timestamp: start,
			Action:    actionPrefix + "." + fieldName,
			Outcome:   outcome,
			Severity:  severity,
			Resource: coreaudit.Resource{
				ID:   fieldName,
				Type: "graphql_field",
				Name: objectName,
			},
			Details:  details,
			Error:    errorMessage,
			Duration: time.Since(start),
		})

		return marshaler
	})

	return srv
}

func graphQLAuditArguments(rootFieldCtx *graphql.RootFieldContext, opCtx *graphql.OperationContext, fc *graphql.FieldContext) map[string]any {
	if fc != nil && len(fc.Args) > 0 {
		return fc.Args
	}
	if rootFieldCtx == nil || rootFieldCtx.Field.Field == nil {
		return nil
	}

	var variables map[string]any
	if opCtx != nil {
		variables = opCtx.Variables
	}

	return rootFieldCtx.Field.ArgumentMap(variables)
}

func setupServer(router *chi.Mux, schema graphql.ExecutableSchema, httpHandlers map[string]http.Handler, staticFiles embed.FS) {
	if !env.IsAPIGatewayEnabled {
		fileServer(router, staticFiles)
	}

	server := NewGraphQLServer(schema)
	server.AddTransport(&transport.Websocket{})
	graph.SetupHTTPServer(router)
	setupPlaygroundHandler(router, server)

	// Register additional HTTP handlers
	for path, h := range httpHandlers {
		router.Handle(path, h)
	}
}

func graphQLAuditScope(args map[string]any) coreaudit.Scope {
	return coreaudit.Scope{
		OrgID:     firstGraphQLArgumentValue(args, "orgid"),
		ProjectID: firstGraphQLArgumentValue(args, "projectid"),
		SourceID:  firstGraphQLArgumentValue(args, "sourceid"),
	}
}

func firstGraphQLArgumentValue(value any, wanted string) string {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if normalizeGraphQLArgumentName(key) == wanted {
				if stringValue := strings.TrimSpace(argumentStringValue(child)); stringValue != "" {
					return stringValue
				}
			}
			if nested := firstGraphQLArgumentValue(child, wanted); nested != "" {
				return nested
			}
		}
		return ""
	case []any:
		for _, child := range typed {
			if nested := firstGraphQLArgumentValue(child, wanted); nested != "" {
				return nested
			}
		}
		return ""
	}

	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return ""
	}
	for reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface {
		if reflected.IsNil() {
			return ""
		}
		reflected = reflected.Elem()
	}

	switch reflected.Kind() {
	case reflect.Map:
		for _, key := range reflected.MapKeys() {
			if key.Kind() != reflect.String {
				continue
			}
			keyValue := key.String()
			child := reflected.MapIndex(key)
			if normalizeGraphQLArgumentName(keyValue) == wanted {
				if stringValue := strings.TrimSpace(argumentStringValue(child.Interface())); stringValue != "" {
					return stringValue
				}
			}
			if nested := firstGraphQLArgumentValue(child.Interface(), wanted); nested != "" {
				return nested
			}
		}
	case reflect.Slice, reflect.Array:
		for index := 0; index < reflected.Len(); index++ {
			if nested := firstGraphQLArgumentValue(reflected.Index(index).Interface(), wanted); nested != "" {
				return nested
			}
		}
	case reflect.Struct:
		reflectedType := reflected.Type()
		for index := 0; index < reflected.NumField(); index++ {
			field := reflectedType.Field(index)
			if !field.IsExported() {
				continue
			}
			fieldValue := reflected.Field(index).Interface()
			if normalizeGraphQLArgumentName(field.Name) == wanted {
				if stringValue := strings.TrimSpace(argumentStringValue(fieldValue)); stringValue != "" {
					return stringValue
				}
			}
			if nested := firstGraphQLArgumentValue(fieldValue, wanted); nested != "" {
				return nested
			}
		}
	}

	return ""
}

func argumentStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case *string:
		if typed == nil {
			return ""
		}
		return *typed
	}
	return ""
}

func normalizeGraphQLArgumentName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "_", ""))
}

// statusResponseWriter wraps http.ResponseWriter to capture the status code.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush forwards to the underlying ResponseWriter if it supports flushing (required for SSE streaming).
func (w *statusResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// healthCheckMiddleware responds to GET /health without requiring authentication.
// Used by E2E setup scripts to verify the server is ready to handle requests.
func healthCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// accessLogMiddleware logs HTTP requests with method, path, status, duration, host, and remote address.
func accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.LogAccess(r.Method, r.URL.Path, sw.statusCode, time.Since(start), r.Host, r.RemoteAddr)
	})
}

func setupMiddlewares(router *chi.Mux, additionalMiddlewares []func(http.Handler) http.Handler, publicPaths []string) {
	allowedOrigins := env.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = append(allowedOrigins, "https://*", "http://*")
	}

	middlewares := []func(http.Handler) http.Handler{
		accessLogMiddleware,
		healthCheckMiddleware,
		middleware.ThrottleBacklog(100, 50, time.Second*5),
		middleware.RequestID,
		middleware.RealIP,
		middleware.RedirectSlashes,
		middleware.Recoverer,
		middleware.Timeout(90 * time.Second), // Increased for LLM inference time
		cors.Handler(cors.Options{
			AllowedOrigins:   allowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{},
			AllowCredentials: true,
			MaxAge:           300,
		}),
		contextMiddleware,
	}
	// Additional middlewares run before CE credential auth so that
	// a bypass registered via auth.RegisterAuthBypass can see context they set.
	middlewares = append(middlewares, additionalMiddlewares...)
	middlewares = append(middlewares, auditHTTPMiddleware)
	if len(publicPaths) > 0 {
		bypassSet := make(map[string]struct{}, len(publicPaths))
		for _, p := range publicPaths {
			bypassSet[p] = struct{}{}
		}
		middlewares = append(middlewares, func(next http.Handler) http.Handler {
			authNext := auth.AuthMiddleware(next)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, ok := bypassSet[r.URL.Path]; ok {
					next.ServeHTTP(w, r)
					return
				}
				authNext.ServeHTTP(w, r)
			})
		})
	} else {
		middlewares = append(middlewares, auth.AuthMiddleware)
	}

	router.Use(middlewares...)
}

func sortedGraphQLArgKeys(args map[string]any) []string {
	if len(args) == 0 {
		return []string{}
	}

	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func graphQLOperationName(opCtx *graphql.OperationContext) string {
	if opCtx == nil {
		return ""
	}
	if name := strings.TrimSpace(opCtx.OperationName); name != "" {
		return name
	}
	if opCtx.Operation != nil {
		return strings.TrimSpace(opCtx.Operation.Name)
	}
	return ""
}

func wrapWithBasePath(handler http.Handler, basePath string) *chi.Mux {
	router := chi.NewRouter()
	redirectToBasePath := func(w http.ResponseWriter, r *http.Request) {
		target := basePath + "/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	}

	router.Get(basePath, redirectToBasePath)
	router.Head(basePath, redirectToBasePath)
	router.Handle("/health", handler)
	router.Handle(basePath+"/*", http.StripPrefix(basePath, handler))

	return router
}

// InitializeRouter creates the chi router with all middleware, GraphQL server, and additional HTTP handlers.
func InitializeRouter(schema graphql.ExecutableSchema, httpHandlers map[string]http.Handler, additionalMiddlewares []func(http.Handler) http.Handler, publicPaths []string, staticFiles embed.FS) *chi.Mux {
	router := chi.NewRouter()

	setupMiddlewares(router, additionalMiddlewares, publicPaths)
	setupServer(router, schema, httpHandlers, staticFiles)

	if env.BasePath == "" {
		return router
	}

	if env.IsAPIGatewayEnabled || !hasEmbeddedFrontend(staticFiles) {
		log.Warnf("Ignoring WHODB_BASE_PATH=%s because bundled frontend assets are not being served", env.BasePath)
		return router
	}

	return wrapWithBasePath(router, env.BasePath)
}
