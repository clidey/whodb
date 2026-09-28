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

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

var publicQueryFields = map[string]struct{}{
	"AWSRegions":            {},
	"AzureProvider":         {},
	"AzureProviders":        {},
	"AzureRegions":          {},
	"AzureSubscriptions":    {},
	"CloudProvider":         {},
	"CloudProviders":        {},
	"DiscoveredConnections": {},
	"GCPProvider":           {},
	"GCPProviders":          {},
	"GCPRegions":            {},
	"Health":                {},
	"LocalAWSProfiles":      {},
	"LocalGCPProjects":      {},
	"ProviderConnections":   {},
	"SettingsConfig":        {},
	"SourceProfiles":        {},
	"SourceTypes":           {},
	"Version":               {},
}

var publicMutationFields = map[string]struct{}{
	"AddAWSProvider":               {},
	"AddAzureProvider":             {},
	"AddGCPProvider":               {},
	"GenerateAzureADToken":         {},
	"GenerateCloudSQLIAMAuthToken": {},
	"GenerateRDSAuthToken":         {},
	"LoginSource":                  {},
	"LoginWithSourceProfile":       {},
	"RefreshAzureProvider":         {},
	"RefreshCloudProvider":         {},
	"RefreshGCPProvider":           {},
	"RemoveCloudProvider":          {},
	"TestAWSCredentials":           {},
	"TestAzureCredentials":         {},
	"TestCloudProvider":            {},
	"TestGCPCredentials":           {},
	"TestSourceConnection":         {},
	"UpdateAWSProvider":            {},
	"UpdateAzureProvider":          {},
	"UpdateGCPProvider":            {},
}

// GraphQLAuthorizationMiddleware authorizes the selected GraphQL operation
// using the root fields collected by gqlgen. Operation names never affect the
// decision, and the entire operation is rejected before resolver execution if
// any selected root field requires authentication.
func GraphQLAuthorizationMiddleware(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	if err := authorizeGraphQLOperation(ctx); err != nil {
		return graphql.OneShot(&graphql.Response{Errors: gqlerror.List{err}})
	}
	return next(ctx)
}

func authorizeGraphQLOperation(ctx context.Context) *gqlerror.Error {
	opCtx := graphql.GetOperationContext(ctx)
	if opCtx.Operation == nil {
		return newGraphQLAuthorizationError("FORBIDDEN", "unsupported GraphQL operation")
	}

	rootType := graphQLRootType(opCtx.Operation.Operation)
	if rootType == "" {
		return newGraphQLAuthorizationError("FORBIDDEN", "unsupported GraphQL operation")
	}

	fields := graphql.CollectFields(opCtx, opCtx.Operation.SelectionSet, []string{rootType})
	if len(fields) == 0 {
		return newGraphQLAuthorizationError("FORBIDDEN", "GraphQL operation has no executable fields")
	}

	allPublic := true
	for _, field := range fields {
		if !isPublicGraphQLField(opCtx, field) {
			allPublic = false
			break
		}
	}
	if allPublic {
		return nil
	}
	if unavailable, _ := ctx.Value(authenticationUnavailableKey{}).(bool); unavailable {
		return newGraphQLAuthorizationError("SERVICE_UNAVAILABLE", "authentication service unavailable")
	}
	if GetAuthenticatedSourceCredentials(ctx) == nil {
		return newGraphQLAuthorizationError("UNAUTHENTICATED", "authentication required")
	}
	if failed, _ := ctx.Value(csrfFailedKey{}).(bool); failed {
		return newGraphQLAuthorizationError("FORBIDDEN", "csrf token required")
	}
	return nil
}

func graphQLRootType(operation ast.Operation) string {
	switch operation {
	case ast.Query:
		return "Query"
	case ast.Mutation:
		return "Mutation"
	case ast.Subscription:
		return "Subscription"
	default:
		return ""
	}
}

func isPublicGraphQLField(opCtx *graphql.OperationContext, field graphql.CollectedField) bool {
	if field.Name == "__typename" {
		return true
	}
	if env.IsDevelopment && (field.Name == "__schema" || field.Name == "__type") {
		return true
	}

	switch opCtx.Operation.Operation {
	case ast.Query:
		if field.Name == "SourceFieldOptions" {
			return sourceFieldOptionsArePublic(field.ArgumentMap(opCtx.Variables))
		}
		_, ok := publicQueryFields[field.Name]
		return ok
	case ast.Mutation:
		_, ok := publicMutationFields[field.Name]
		return ok
	default:
		return false
	}
}

func sourceFieldOptionsArePublic(args map[string]any) bool {
	sourceType, _ := args["sourceType"].(string)
	fieldKey, _ := args["fieldKey"].(string)
	spec, ok := sourcecatalog.Find(sourceType)
	if !ok {
		return false
	}
	field, ok := spec.ConnectionFieldByKey(fieldKey)
	return ok && field.SupportsOptions
}

func newGraphQLAuthorizationError(code, message string) *gqlerror.Error {
	err := gqlerror.Errorf("%s", message)
	err.Extensions = map[string]any{"code": code}
	return err
}
