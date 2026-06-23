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

package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const statusResponseFields = `
  Status
`

const aiProviderFields = `
  id
  projectId
  name
  providerType
  endpoint
  createdBy
  createdAt
  updatedAt
`

const ontologyFastLookupFields = `
  id
  entityId
  displayName
  fields
  status
  reason
  createdAt
  updatedAt
`

const transformRunFields = `
  id
  transformId
  status
  errorMessage
  triggeredBy
  startedAt
  completedAt
`

// PlatformMutationResult contains the raw JSON payload returned by one hosted platform mutation.
type PlatformMutationResult struct {
	Operation string
	Data      json.RawMessage
}

type platformMutationSpec struct {
	Operation string
	Query     string
}

var platformMutationSpecs = map[string]platformMutationSpec{
	"CreateSecret":               mutationSpecWithInput("CreateSecret", "CreateSecretInput", projectSecretFields),
	"UpdateSecret":               mutationSpecWithInput("UpdateSecret", "UpdateSecretInput", projectSecretFields),
	"DeleteSecret":               mutationSpecWithProjectID("DeleteSecret", statusResponseFields),
	"CreateAIProvider":           mutationSpecWithInput("CreateAIProvider", "CreateAIProviderInput", aiProviderFields),
	"UpdateAIProvider":           mutationSpecWithInput("UpdateAIProvider", "UpdateAIProviderInput", aiProviderFields),
	"DeleteAIProvider":           mutationSpecWithID("DeleteAIProvider", statusResponseFields),
	"CreateOntology":             mutationSpecWithInput("CreateOntology", "CreateOntologyInput", ontologyFields),
	"UpdateOntology":             mutationSpecWithInput("UpdateOntology", "UpdateOntologyEntityInput", ontologyFields),
	"DeleteOntology":             mutationSpecWithProjectID("DeleteOntology", statusResponseFields),
	"CreateOntologyFastLookup":   mutationSpecWithInput("CreateOntologyFastLookup", "CreateOntologyFastLookupInput", ontologyFastLookupFields),
	"RemoveOntologyFastLookup":   mutationSpecWithProjectID("RemoveOntologyFastLookup", statusResponseFields),
	"CreateDataset":              mutationSpecWithInput("CreateDataset", "CreateDatasetInput", datasetFields),
	"UpdateDataset":              mutationSpecWithInput("UpdateDataset", "UpdateDatasetInput", datasetFields),
	"DeleteDataset":              mutationSpecWithProjectID("DeleteDataset", statusResponseFields),
	"SaveTransform":              mutationSpecWithInput("SaveTransform", "SaveTransformInput", transformFields),
	"DeleteTransform":            mutationSpecWithProjectID("DeleteTransform", statusResponseFields),
	"RunTransform":               mutationSpecWithProjectID("RunTransform", transformRunFields),
	"CreateProjectFolder":        mutationSpecWithInput("CreateProjectFolder", "CreateFolderInput", projectFolderFields),
	"RenameProjectFile":          mutationSpecWithProjectIDAndName("RenameProjectFile", projectFileFields),
	"RenameProjectFolder":        mutationSpecWithProjectIDAndName("RenameProjectFolder", projectFolderFields),
	"MoveProjectFile":            mutationSpecWithInput("MoveProjectFile", "MoveFileInput", projectFileFields),
	"MoveProjectFolder":          mutationSpecWithInput("MoveProjectFolder", "MoveFolderInput", projectFolderFields),
	"DeleteProjectFile":          mutationSpecWithProjectID("DeleteProjectFile", statusResponseFields),
	"DeleteProjectFolder":        mutationSpecWithProjectID("DeleteProjectFolder", statusResponseFields),
	"PromoteFileToDataset":       mutationSpecWithInput("PromoteFileToDataset", "PromoteFileInput", datasetFields),
	"CreateFunction":             mutationSpecWithInput("CreateFunction", "CreateFunctionInput", functionWriteFields),
	"UpdateFunction":             mutationSpecWithInput("UpdateFunction", "UpdateFunctionInput", functionWriteFields),
	"DeleteFunction":             mutationSpecWithProjectID("DeleteFunction", statusResponseFields),
	"DeployFunction":             mutationSpecWithProjectID("DeployFunction", statusResponseFields),
	"RedeployFunction":           mutationSpecWithProjectID("RedeployFunction", statusResponseFields),
	"CreatePlatformSourceObject": mutationSpecWithDirect("CreatePlatformSourceObject", "$projectId: ID!, $sourceId: ID!, $parent: SourceObjectRefInput, $name: String!, $fields: [RecordInput!]!", "projectId: $projectId, sourceId: $sourceId, parent: $parent, name: $name, fields: $fields", statusResponseFields),
	"UpdatePlatformSourceObject": mutationSpecWithDirect("UpdatePlatformSourceObject", "$projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!, $values: [RecordInput!]!, $updatedColumns: [String!]!", "projectId: $projectId, sourceId: $sourceId, ref: $ref, values: $values, updatedColumns: $updatedColumns", statusResponseFields),
	"DeletePlatformSourceObject": mutationSpecWithDirect("DeletePlatformSourceObject", "$projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!, $values: [RecordInput!]!", "projectId: $projectId, sourceId: $sourceId, ref: $ref, values: $values", statusResponseFields),
}

const projectSecretFields = `
  id
  projectId
  name
  description
  createdBy
  updatedBy
  createdAt
  updatedAt
  lastUsedAt
  usedBy {
    consumerType
    consumerId
    consumerName
    bindingName
    mode
  }
`

const functionWriteFields = `
  id
  projectId
  name
  description
  language
  entryPoint
  timeoutSeconds
  memory
  cpu
  createdBy
  createdAt
  updatedAt
  providerIds
  ontologyIds
  readOnlyOntologyIds
  providerConfigs {
    providerId
    model
  }
  secretBindings {
    name
    secretId
    target
  }
  defaultMaxTokens
  defaultTemperature
  isDeployed
`

func mutationSpecWithInput(operation, inputType, fields string) platformMutationSpec {
	return platformMutationSpec{
		Operation: operation,
		Query: fmt.Sprintf(`
mutation CLIPlatform%s($input: %s!) {
  %s(input: $input) {
%s
  }
}
`, operation, inputType, operation, fields),
	}
}

func mutationSpecWithProjectID(operation, fields string) platformMutationSpec {
	return platformMutationSpec{
		Operation: operation,
		Query: fmt.Sprintf(`
mutation CLIPlatform%s($projectId: ID!, $id: ID!) {
  %s(projectId: $projectId, id: $id) {
%s
  }
}
`, operation, operation, fields),
	}
}

func mutationSpecWithID(operation, fields string) platformMutationSpec {
	return platformMutationSpec{
		Operation: operation,
		Query: fmt.Sprintf(`
mutation CLIPlatform%s($id: ID!) {
  %s(id: $id) {
%s
  }
}
`, operation, operation, fields),
	}
}

func mutationSpecWithProjectIDAndName(operation, fields string) platformMutationSpec {
	return platformMutationSpec{
		Operation: operation,
		Query: fmt.Sprintf(`
mutation CLIPlatform%s($projectId: ID!, $id: ID!, $name: String!) {
  %s(projectId: $projectId, id: $id, name: $name) {
%s
  }
}
`, operation, operation, fields),
	}
}

func mutationSpecWithDirect(operation, variables, args, fields string) platformMutationSpec {
	return platformMutationSpec{
		Operation: operation,
		Query: fmt.Sprintf(`
mutation CLIPlatform%s(%s) {
  %s(%s) {
%s
  }
}
`, operation, variables, operation, args, fields),
	}
}

// PlatformMutation executes one whitelisted hosted platform mutation.
func (c *Client) PlatformMutation(ctx context.Context, operation string, variables map[string]any) (*PlatformMutationResult, error) {
	spec, ok := platformMutationSpecs[operation]
	if !ok {
		return nil, fmt.Errorf("unsupported platform mutation %q", operation)
	}
	if err := c.RequireOperation("Mutation", operation, "platform write"); err != nil {
		return nil, err
	}
	var resp map[string]json.RawMessage
	if err := c.graphQL(ctx, spec.Query, variables, &resp); err != nil {
		return nil, err
	}
	raw, ok := resp[operation]
	if !ok {
		return nil, fmt.Errorf("platform returned no %s result", operation)
	}
	return &PlatformMutationResult{Operation: operation, Data: raw}, nil
}

// UploadProjectFile uploads one local file to hosted project storage.
func (c *Client) UploadProjectFile(ctx context.Context, projectID string, folderID *string, filePath string) (*ProjectFile, error) {
	if err := c.RequireOperation("Mutation", "UploadProjectFile", "file upload"); err != nil {
		return nil, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	operations, err := json.Marshal(map[string]any{
		"query": `
mutation CLIPlatformUploadProjectFile($projectId: ID!, $folderId: ID, $file: Upload!) {
  UploadProjectFile(projectId: $projectId, folderId: $folderId, file: $file) {
` + projectFileFields + `
  }
}
`,
		"variables": map[string]any{"projectId": projectID, "folderId": folderID, "file": nil},
	})
	if err != nil {
		return nil, err
	}
	if err := writer.WriteField("operations", string(operations)); err != nil {
		return nil, err
	}
	if err := writer.WriteField("map", `{"0":["variables.file"]}`); err != nil {
		return nil, err
	}
	part, err := writer.CreateFormFile("0", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+defaultPath, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	if c.workspaceOrgID != "" {
		req.Header.Set(workspaceOrgHeader, c.workspaceOrgID)
	}
	if c.workspaceProjectID != "" {
		req.Header.Set(workspaceProjectHeader, c.workspaceProjectID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("platform request failed: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var envelope struct {
		Data struct {
			UploadProjectFile *ProjectFile `json:"UploadProjectFile"`
		} `json:"data"`
		Errors []struct {
			Message    string `json:"message"`
			Extensions struct {
				Code string `json:"code"`
			} `json:"extensions"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		return nil, GraphQLError{Message: first.Message, Code: first.Extensions.Code}
	}
	if envelope.Data.UploadProjectFile == nil {
		return nil, fmt.Errorf("platform returned no uploaded file")
	}
	return envelope.Data.UploadProjectFile, nil
}
