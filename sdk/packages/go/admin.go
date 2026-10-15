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

package whodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/sdk/packages/go/gen"
)

// ServiceAccountsHandle is the client.ServiceAccounts() facade: API access
// principals for the organization the client is scoped to.
type ServiceAccountsHandle struct {
	client *Client
}

// List lists service accounts with key metadata; raw keys are never returned.
func (h *ServiceAccountsHandle) List(ctx context.Context) ([]map[string]any, error) {
	orgID, err := h.client.orgID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ServiceAccounts")
	result, err := h.client.execute(ctx, gen.NewServiceAccountsRequest(map[string]any{"orgId": orgID}))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Create creates a service account. Issue a key with CreateKey.
func (h *ServiceAccountsHandle) Create(ctx context.Context, name, description string) (map[string]any, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	orgID, err := h.client.orgID(ctx)
	if err != nil {
		return nil, err
	}
	variables := map[string]any{"orgId": orgID, "name": strings.TrimSpace(name), "description": nil}
	if description != "" {
		variables["description"] = description
	}
	warnIfFlagged("CreateServiceAccount")
	result, err := h.client.execute(ctx, gen.NewCreateServiceAccountRequest(variables))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// CreateKey issues an API key. The returned rawKey is shown exactly once;
// store it securely and never log it. expiresAt is RFC 3339 or empty.
func (h *ServiceAccountsHandle) CreateKey(ctx context.Context, serviceAccountID, expiresAt string) (map[string]any, error) {
	if serviceAccountID == "" {
		return nil, fmt.Errorf("%w: serviceAccountID is required", ErrValidation)
	}
	orgID, err := h.client.orgID(ctx)
	if err != nil {
		return nil, err
	}
	variables := map[string]any{"orgId": orgID, "serviceAccountId": serviceAccountID, "expiresAt": nil}
	if expiresAt != "" {
		variables["expiresAt"] = expiresAt
	}
	warnIfFlagged("CreateServiceAccountKey")
	result, err := h.client.execute(ctx, gen.NewCreateServiceAccountKeyRequest(variables))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// RevokeKey revokes one key.
func (h *ServiceAccountsHandle) RevokeKey(ctx context.Context, keyID string) error {
	if keyID == "" {
		return fmt.Errorf("%w: keyID is required", ErrValidation)
	}
	orgID, err := h.client.orgID(ctx)
	if err != nil {
		return err
	}
	warnIfFlagged("RevokeServiceAccountKey")
	_, err = h.client.execute(ctx, gen.NewRevokeServiceAccountKeyRequest(map[string]any{"orgId": orgID, "keyId": keyID}))
	return err
}

// Delete deletes a service account and revokes all of its keys.
func (h *ServiceAccountsHandle) Delete(ctx context.Context, serviceAccountID string) error {
	if serviceAccountID == "" {
		return fmt.Errorf("%w: serviceAccountID is required", ErrValidation)
	}
	orgID, err := h.client.orgID(ctx)
	if err != nil {
		return err
	}
	warnIfFlagged("DeleteServiceAccount")
	_, err = h.client.execute(ctx, gen.NewDeleteServiceAccountRequest(map[string]any{"orgId": orgID, "id": serviceAccountID}))
	return err
}

// LineageProvidersHandle is the client.LineageProviders() facade: external
// OpenLineage providers feeding the project's lineage graph.
type LineageProvidersHandle struct {
	client *Client
}

// List lists the project's providers; API keys are never returned.
func (h *LineageProvidersHandle) List(ctx context.Context) ([]map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ProjectLineageProviders")
	result, err := h.client.execute(ctx, gen.NewProjectLineageProvidersRequest(map[string]any{"projectId": projectID}))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Save creates a provider, or updates the one whose id is in the definition.
func (h *LineageProvidersHandle) Save(ctx context.Context, definition map[string]any) (map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("SaveLineageProvider")
	result, err := h.client.execute(ctx, gen.NewSaveLineageProviderRequest(map[string]any{
		"input": mergeInput(definition, map[string]any{"projectId": projectID}),
	}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// Delete deletes a provider.
func (h *LineageProvidersHandle) Delete(ctx context.Context, providerID string) error {
	if providerID == "" {
		return fmt.Errorf("%w: providerID is required", ErrValidation)
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return err
	}
	warnIfFlagged("DeleteLineageProvider")
	_, err = h.client.execute(ctx, gen.NewDeleteLineageProviderRequest(map[string]any{"projectId": projectID, "providerId": providerID}))
	return err
}

// SharingHandle is the client.Sharing() facade: share ontologies with other
// projects in the organization and reference shared ontologies locally.
type SharingHandle struct {
	client *Client
}

// Shared lists ontologies this project shares, with their consumers.
func (h *SharingHandle) Shared(ctx context.Context) ([]map[string]any, error) {
	return h.projectList(ctx, "SharedOntologies", "projectId", gen.NewSharedOntologiesRequest)
}

// Discover lists ontologies other projects share that this project may reference.
func (h *SharingHandle) Discover(ctx context.Context) ([]map[string]any, error) {
	return h.projectList(ctx, "DiscoverSharedOntologies", "consumerProjectId", gen.NewDiscoverSharedOntologiesRequest)
}

// References lists shared ontologies this project references.
func (h *SharingHandle) References(ctx context.Context) ([]map[string]any, error) {
	return h.projectList(ctx, "OntologyReferences", "projectId", gen.NewOntologyReferencesRequest)
}

func (h *SharingHandle) projectList(ctx context.Context, operation, variable string, build func(map[string]any) gen.Request) ([]map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged(operation)
	result, err := h.client.execute(ctx, build(map[string]any{variable: projectID}))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// ReferenceImpact counts dependents of a reference before removing it.
func (h *SharingHandle) ReferenceImpact(ctx context.Context, referenceID string) (map[string]any, error) {
	if referenceID == "" {
		return nil, fmt.Errorf("%w: referenceID is required", ErrValidation)
	}
	warnIfFlagged("OntologyReferenceImpact")
	result, err := h.client.execute(ctx, gen.NewOntologyReferenceImpactRequest(map[string]any{"referenceId": referenceID}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// Share shares one of this project's ontologies. visibility is "organization"
// or "allowlist"; allowedProjectIDs applies to allowlist sharing.
func (h *SharingHandle) Share(ctx context.Context, ontologyID, visibility string, allowedProjectIDs []string) (map[string]any, error) {
	if ontologyID == "" {
		return nil, fmt.Errorf("%w: ontologyID is required", ErrValidation)
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	input := map[string]any{"projectId": projectID, "ontologyId": ontologyID, "visibility": visibility, "allowedProjectIds": nil}
	if allowedProjectIDs != nil {
		input["allowedProjectIds"] = allowedProjectIDs
	}
	warnIfFlagged("ShareOntology")
	result, err := h.client.execute(ctx, gen.NewShareOntologyRequest(map[string]any{"input": input}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// Unshare stops sharing an ontology; existing references become broken.
func (h *SharingHandle) Unshare(ctx context.Context, ontologyID string) error {
	if ontologyID == "" {
		return fmt.Errorf("%w: ontologyID is required", ErrValidation)
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return err
	}
	warnIfFlagged("UnshareOntology")
	_, err = h.client.execute(ctx, gen.NewUnshareOntologyRequest(map[string]any{"projectId": projectID, "ontologyId": ontologyID}))
	return err
}

// CreateReference references a shared ontology from this project under a local alias.
func (h *SharingHandle) CreateReference(ctx context.Context, ownerProjectID, ontologyID, alias string) (map[string]any, error) {
	if ownerProjectID == "" || ontologyID == "" || alias == "" {
		return nil, fmt.Errorf("%w: ownerProjectID, ontologyID and alias are required", ErrValidation)
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("CreateOntologyReference")
	result, err := h.client.execute(ctx, gen.NewCreateOntologyReferenceRequest(map[string]any{
		"input": map[string]any{"consumerProjectId": projectID, "ownerProjectId": ownerProjectID, "ontologyId": ontologyID, "alias": alias},
	}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// RemoveReference removes a reference. Check ReferenceImpact first.
func (h *SharingHandle) RemoveReference(ctx context.Context, referenceID string) error {
	if referenceID == "" {
		return fmt.Errorf("%w: referenceID is required", ErrValidation)
	}
	warnIfFlagged("RemoveOntologyReference")
	_, err := h.client.execute(ctx, gen.NewRemoveOntologyReferenceRequest(map[string]any{"referenceId": referenceID}))
	return err
}
