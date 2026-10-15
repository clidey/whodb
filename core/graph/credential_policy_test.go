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

package graph

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/types"
)

func TestDisabledCredentialFormRejectsConnectionTestBeforePlugin(t *testing.T) {
	original := env.DisableCredentialForm
	env.DisableCredentialForm = true
	t.Cleanup(func() { env.DisableCredentialForm = original })

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	connectionAttempts := 0
	mock.IsAvailableFunc = func(_ context.Context, _ *engine.PluginConfig) bool {
		connectionAttempts++
		return false
	}
	setEngineMock(t, mock)

	graphQLServer := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	graphQLServer.AroundOperations(auth.GraphQLAuthorizationMiddleware)
	srv := auth.AuthMiddleware(graphQLServer)
	body := `{"query":"mutation { TestSourceConnection(credentials:{SourceType:\"Postgres\", Values:[{Key:\"Hostname\", Value:\"attacker.example\"}]}) { Status } }"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "login with credentials is disabled") {
		t.Fatalf("expected policy rejection, got status=%d body=%s", w.Code, w.Body.String())
	}
	if connectionAttempts != 0 {
		t.Fatalf("connection test reached plugin %d times", connectionAttempts)
	}
}

func TestDisabledCredentialFormRestrictsFieldOptionsAndSecretExport(t *testing.T) {
	original := env.DisableCredentialForm
	env.DisableCredentialForm = true
	t.Cleanup(func() { env.DisableCredentialForm = original })

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	connectionAttempts := 0
	var connectedHost, connectedDatabase string
	mock.GetDatabasesFunc = func(config *engine.PluginConfig) ([]string, error) {
		connectionAttempts++
		connectedHost = config.Credentials.Hostname
		connectedDatabase = config.Credentials.Database
		return []string{"app", "reports"}, nil
	}
	setEngineMock(t, mock)
	src.MainEngine.AddLoginProfile(types.DatabaseCredentials{
		CustomId: "production", Type: "Postgres", Source: "environment",
		Hostname: "db.internal", Username: "reader", Password: "server-owned-secret", Database: "app",
		Advanced: map[string]string{"API Key": "synthetic-private-api-key"},
	})

	graphQLServer := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	graphQLServer.AroundOperations(auth.GraphQLAuthorizationMiddleware)
	srv := auth.AuthMiddleware(graphQLServer)

	profileID := "production"
	profileReference, err := json.Marshal(source.Credentials{ID: &profileID})
	if err != nil {
		t.Fatal(err)
	}
	bearer := base64.StdEncoding.EncodeToString(profileReference)
	send := func(query string, authenticated bool) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(map[string]string{"query": query})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if authenticated {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		return w
	}

	w := send(`query { SourceProfiles { Id Values { Key Value } } }`, false)
	if !strings.Contains(w.Body.String(), "production") || strings.Contains(w.Body.String(), "synthetic-private-api-key") {
		t.Fatalf("public profile listing exposed a managed secret: status=%d body=%s", w.Code, w.Body.String())
	}

	optionsQuery := `query { SourceFieldOptions(sourceType:"Postgres", fieldKey:"Database", values:[{Key:"Hostname", Value:"attacker.example"}]) }`
	w = send(optionsQuery, false)
	if !strings.Contains(w.Body.String(), "authentication required") || connectionAttempts != 0 {
		t.Fatalf("anonymous field options reached connector: status=%d body=%s attempts=%d", w.Code, w.Body.String(), connectionAttempts)
	}

	w = send(optionsQuery, true)
	if !strings.Contains(w.Body.String(), "source profile connection fields cannot be overridden") || connectionAttempts != 0 {
		t.Fatalf("profile override reached connector: status=%d body=%s attempts=%d", w.Code, w.Body.String(), connectionAttempts)
	}

	w = send(`query { SourceFieldOptions(sourceType:"Postgres", fieldKey:"Hostname") }`, true)
	if !strings.Contains(w.Body.String(), `"SourceFieldOptions":[]`) || connectionAttempts != 0 {
		t.Fatalf("unsupported options field reached connector: status=%d body=%s attempts=%d", w.Code, w.Body.String(), connectionAttempts)
	}

	w = send(`query { SourceFieldOptions(sourceType:"Postgres", fieldKey:"Database", values:[{Key:"Database", Value:"reports"}]) }`, true)
	if !strings.Contains(w.Body.String(), `"reports"`) || connectionAttempts != 1 || connectedHost != "db.internal" || connectedDatabase != "reports" {
		t.Fatalf("configured profile database selection failed: status=%d body=%s attempts=%d host=%q database=%q", w.Code, w.Body.String(), connectionAttempts, connectedHost, connectedDatabase)
	}

	w = send(`mutation { ExportSourceConnection(id:"production", includeSecrets:true) { Key Value } }`, true)
	if !strings.Contains(w.Body.String(), "exporting source connections is disabled") || strings.Contains(w.Body.String(), "server-owned-secret") {
		t.Fatalf("managed profile secret export was allowed: status=%d body=%s", w.Code, w.Body.String())
	}
	w = send(`mutation { ExportSourceConnection(id:"production", includeSecrets:false) { Key Value } }`, true)
	if !strings.Contains(w.Body.String(), "exporting source connections is disabled") || strings.Contains(w.Body.String(), "synthetic-private-api-key") {
		t.Fatalf("managed profile export exposed an unknown secret: status=%d body=%s", w.Code, w.Body.String())
	}
}
