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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/clidey/whodb/core/graph"
	coreaudit "github.com/clidey/whodb/core/src/audit"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
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
		for index := range reflected.Len() {
			if nested := firstGraphQLArgumentValue(reflected.Index(index).Interface(), wanted); nested != "" {
				return nested
			}
		}
	case reflect.Struct:
		reflectedType := reflected.Type()
		for index := range reflected.NumField() {
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
	statusCode  int
	wroteHeader bool
}

func (w *statusResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Flush forwards to the underlying ResponseWriter if it supports flushing (required for SSE streaming).
func (w *statusResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// sseAwareTimeout applies the given timeout to all requests except SSE streaming endpoints,
// which are long-lived connections that manage their own timeouts via the LLM HTTP client.
func sseAwareTimeout(dt time.Duration) func(http.Handler) http.Handler {
	timeoutMiddleware := middleware.Timeout(dt)
	return func(next http.Handler) http.Handler {
		timedHandler := timeoutMiddleware(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/stream") {
				next.ServeHTTP(w, r)
				return
			}
			timedHandler.ServeHTTP(w, r)
		})
	}
}

// sseAwareThrottle applies concurrency throttling to all requests except SSE streaming
// endpoints. SSE connections are long-lived and would permanently consume slots meant
// for transactional traffic.
func sseAwareThrottle(limit, backlog int, timeout time.Duration) func(http.Handler) http.Handler {
	throttleMiddleware := middleware.ThrottleBacklog(limit, backlog, timeout)
	return func(next http.Handler) http.Handler {
		throttledHandler := throttleMiddleware(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/stream") {
				next.ServeHTTP(w, r)
				return
			}
			throttledHandler.ServeHTTP(w, r)
		})
	}
}

// sseAwareCompress applies gzip compression to all responses except SSE streaming endpoints.
// SSE payloads are tiny text fragments where compression saves nothing meaningful, and
// buffering inside the compressor can prevent heartbeat bytes from reaching the wire —
// causing load balancers to close idle connections.
func sseAwareCompress(level int) func(http.Handler) http.Handler {
	compressMiddleware := middleware.Compress(level)
	return func(next http.Handler) http.Handler {
		compressedHandler := compressMiddleware(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/stream") {
				next.ServeHTTP(w, r)
				return
			}
			compressedHandler.ServeHTTP(w, r)
		})
	}
}

// healthCheckMiddleware responds to GET /health without requiring authentication.
// Used by E2E setup scripts to verify the server is ready to handle requests.
func healthCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
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

// cspReportOnlyPolicy is the full Content-Security-Policy we want to enforce.
// It is shipped in Report-Only mode first: browsers report violations (visible
// in DevTools / a report endpoint) without blocking anything, so we can confirm
// it does not break the SPA before switching to enforcement. Notes on each
// directive's necessity (verified against the frontend):
//   - script-src 'self': the app loads no inline/eval scripts (no eval/new
//     Function/wasm found); Vite emits module scripts from same-origin.
//   - style-src 'unsafe-inline': Tailwind v4 + @emotion inject runtime inline
//     styles, so this cannot be dropped without nonces they don't support.
//   - connect-src 'self': the browser only talks to the same-origin backend
//     (which proxies AI/SSE); external provider hosts are server-side only.
//   - img-src 'self' data: https:: DB cell content and provider icons can be
//     arbitrary https/data URLs.
const cspReportOnlyPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: https:; " +
	"font-src 'self' data:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'"

// securityHeadersMiddleware adds defense-in-depth response headers:
//   - X-Frame-Options/frame-ancestors: block clickjacking by disallowing framing
//   - X-Content-Type-Options: stop MIME sniffing
//   - Referrer-Policy: avoid leaking full URLs cross-origin
//   - Strict-Transport-Security: enforce HTTPS once seen over TLS
//   - Content-Security-Policy: enforced frame-ancestors (safe) + a full policy
//     in Report-Only mode (observe-before-enforce)
//
// HSTS is only emitted when the request arrived over TLS so local HTTP dev is
// unaffected.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Enforce only the anti-clickjacking directive (cannot break resource
		// loading). The full resource policy ships Report-Only until validated.
		h.Set("Content-Security-Policy", "frame-ancestors 'none'")
		h.Set("Content-Security-Policy-Report-Only", cspReportOnlyPolicy)
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// originsContainWildcard reports whether any configured CORS origin is the "*"
// literal or a wildcard pattern (e.g. "https://*"). go-chi/cors reflects the
// caller's Origin for wildcard patterns, which must never be paired with
// Access-Control-Allow-Credentials.
func originsContainWildcard(origins []string) bool {
	for _, o := range origins {
		if strings.Contains(o, "*") {
			return true
		}
	}
	return false
}

func setupMiddlewares(router *chi.Mux, additionalMiddlewares []func(http.Handler) http.Handler, publicPaths []string) {
	// CORS must fail closed. Reflecting an arbitrary Origin together with
	// Access-Control-Allow-Credentials:true lets any website issue credentialed
	// cross-origin requests and read the responses, which would expose the
	// session/credential cookies. We therefore:
	//   - never default to a wildcard origin (an unset allowlist disables CORS);
	//   - only send Allow-Credentials when the configured origins are explicit
	//     (no "*" / wildcard pattern), since credentials + wildcard is unsafe.
	allowedOrigins := env.AllowedOrigins
	allowCredentials := len(allowedOrigins) > 0 && !originsContainWildcard(allowedOrigins)
	if len(allowedOrigins) == 0 {
		log.Warnf("WHODB_ALLOWED_ORIGINS is unset; cross-origin requests are disabled. Set it to your app origin (e.g. https://app.example.com) to enable CORS.")
	} else if !allowCredentials {
		log.Warnf("WHODB_ALLOWED_ORIGINS contains a wildcard (%v); Access-Control-Allow-Credentials is disabled because credentialed wildcard CORS is unsafe. Use explicit origins to allow credentials.", allowedOrigins)
	}

	middlewares := []func(http.Handler) http.Handler{
		accessLogMiddleware,
		securityHeadersMiddleware,
		healthCheckMiddleware,
		sseAwareThrottle(100, 50, time.Second*5),
		middleware.RequestID,
		middleware.ClientIPFromRemoteAddr,
		middleware.RedirectSlashes,
		middleware.Recoverer,
		sseAwareCompress(5),
		sseAwareTimeout(90 * time.Second),
		cors.Handler(cors.Options{
			AllowedOrigins:   allowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{},
			AllowCredentials: allowCredentials,
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

func wrapWithBasePath(h http.Handler, basePath string) *chi.Mux {
	router := chi.NewRouter()
	redirectToBasePath := func(w http.ResponseWriter, r *http.Request) {
		target := basePath + "/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently) //nolint:gosec
	}

	router.Get(basePath, redirectToBasePath)
	router.Head(basePath, redirectToBasePath)
	router.Handle("/health", h)
	router.Handle(basePath+"/*", http.StripPrefix(basePath, h))

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
