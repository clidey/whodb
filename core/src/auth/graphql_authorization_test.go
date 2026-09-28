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

package auth

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"

	"github.com/clidey/whodb/core/src/source"
)

func operationContext(t *testing.T, query, operationName string, variables map[string]any) context.Context {
	t.Helper()
	doc, err := parser.ParseQuery(&ast.Source{Name: "test.graphql", Input: query})
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	operation := doc.Operations.ForName(operationName)
	if operation == nil {
		t.Fatalf("operation %q not found", operationName)
	}
	return graphql.WithOperationContext(context.Background(), &graphql.OperationContext{
		Doc:           doc,
		Operation:     operation,
		OperationName: operationName,
		Variables:     variables,
	})
}

func authenticatedOperationContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, AuthKey_Source, &source.Credentials{SourceType: "Postgres"})
	return context.WithValue(ctx, authenticatedSourceKey{}, true)
}

func TestGraphQLAuthorizationUsesSelectedFields(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		operationName string
		wantCode      string
	}{
		{
			name:          "public query",
			query:         `query ArbitraryName { Version }`,
			operationName: "ArbitraryName",
		},
		{
			name:          "public query through fragment and alias",
			query:         `query PublicData { ...Fields } fragment Fields on Query { build: Version SettingsConfig { MaxPageSize } }`,
			operationName: "PublicData",
		},
		{
			name:          "private mutation with spoofed public name",
			query:         `mutation Version { UpdateSettings(newSettings: { MetricsEnabled: true }) { Status } }`,
			operationName: "Version",
			wantCode:      "UNAUTHENTICATED",
		},
		{
			name:          "mixed public and private query",
			query:         `query Version { Version SourceObjects { Name } }`,
			operationName: "Version",
			wantCode:      "UNAUTHENTICATED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := operationContext(t, tt.query, tt.operationName, nil)
			err := authorizeGraphQLOperation(ctx)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("authorization failed: %v", err)
				}
				return
			}
			if err == nil || err.Extensions["code"] != tt.wantCode {
				t.Fatalf("authorization error = %#v, want code %s", err, tt.wantCode)
			}
		})
	}
}

func TestGraphQLAuthorizationRequiresCSRFForAuthenticatedPrivateField(t *testing.T) {
	ctx := operationContext(t, `mutation Version { UpdateSettings(newSettings: { MetricsEnabled: true }) { Status } }`, "Version", nil)
	ctx = authenticatedOperationContext(ctx)
	ctx = context.WithValue(ctx, csrfFailedKey{}, true)

	err := authorizeGraphQLOperation(ctx)
	if err == nil || err.Extensions["code"] != "FORBIDDEN" {
		t.Fatalf("authorization error = %#v, want FORBIDDEN", err)
	}
}

func TestGraphQLAuthorizationAllowsAuthenticatedPrivateField(t *testing.T) {
	ctx := operationContext(t, `query PrivateData { SourceObjects { Name } }`, "PrivateData", nil)
	ctx = authenticatedOperationContext(ctx)

	if err := authorizeGraphQLOperation(ctx); err != nil {
		t.Fatalf("authorization failed: %v", err)
	}
}

func TestGraphQLAuthorizationPreservesTransientAuthenticationFailure(t *testing.T) {
	ctx := operationContext(t, `query PrivateData { SourceObjects { Name } }`, "PrivateData", nil)
	ctx = context.WithValue(ctx, authenticationUnavailableKey{}, true)

	err := authorizeGraphQLOperation(ctx)
	if err == nil || err.Extensions["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("authorization error = %#v, want SERVICE_UNAVAILABLE", err)
	}
}

func TestSourceFieldOptionsFailClosedForUnknownSource(t *testing.T) {
	ctx := operationContext(t,
		`query Options($sourceType: String!, $fieldKey: String!) { SourceFieldOptions(sourceType: $sourceType, fieldKey: $fieldKey) }`,
		"Options",
		map[string]any{"sourceType": "Unknown", "fieldKey": "Database"},
	)

	err := authorizeGraphQLOperation(ctx)
	if err == nil || err.Extensions["code"] != "UNAUTHENTICATED" {
		t.Fatalf("authorization error = %#v, want UNAUTHENTICATED", err)
	}
}
