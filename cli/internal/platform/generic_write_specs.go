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

// GenericWriteMode describes how generic hosted platform write variables are shaped.
type GenericWriteMode string

const (
	// GenericWriteModeInput wraps payload fields in an input variable.
	GenericWriteModeInput GenericWriteMode = "input"
	// GenericWriteModeProjectID passes projectId and id as top-level variables.
	GenericWriteModeProjectID GenericWriteMode = "project_id"
	// GenericWriteModeID passes id as the only top-level identity variable.
	GenericWriteModeID GenericWriteMode = "id"
	// GenericWriteModeProjectIDName passes projectId, id, and name as top-level variables.
	GenericWriteModeProjectIDName GenericWriteMode = "project_id_name"
	// GenericWriteModeDirect passes the payload as direct GraphQL variables.
	GenericWriteModeDirect GenericWriteMode = "direct"
	// GenericWriteModeFileUpload uses the hosted multipart file upload path.
	GenericWriteModeFileUpload GenericWriteMode = "file_upload"
)

// GenericWriteSpec describes one hosted platform resource write supported by CLI and MCP.
type GenericWriteSpec struct {
	Resource        string
	Action          string
	Mutation        string
	Mode            GenericWriteMode
	NeedsID         bool
	InjectProjectID bool
}

// PayloadShape describes the payload_json object expected for one generic write.
type PayloadShape struct {
	Key         string
	Resource    string
	Action      string
	Description string
	Fields      []PayloadField
	Examples    []string
}

// PayloadField describes one payload_json field for a generic write.
type PayloadField struct {
	Name        string
	Type        string
	Required    bool
	Secret      bool
	Description string
}

// GenericWriteSpecs maps operation/resource tokens to hosted platform mutations.
var GenericWriteSpecs = map[string]GenericWriteSpec{
	"create:secret":                  {Resource: "secret", Action: "create", Mutation: "CreateSecret", Mode: GenericWriteModeInput, InjectProjectID: true},
	"update:secret":                  {Resource: "secret", Action: "update", Mutation: "UpdateSecret", Mode: GenericWriteModeInput, NeedsID: true, InjectProjectID: true},
	"delete:secret":                  {Resource: "secret", Action: "delete", Mutation: "DeleteSecret", Mode: GenericWriteModeProjectID, NeedsID: true},
	"create:ai_provider":             {Resource: "ai_provider", Action: "create", Mutation: "CreateAIProvider", Mode: GenericWriteModeInput, InjectProjectID: true},
	"update:ai_provider":             {Resource: "ai_provider", Action: "update", Mutation: "UpdateAIProvider", Mode: GenericWriteModeInput, NeedsID: true},
	"delete:ai_provider":             {Resource: "ai_provider", Action: "delete", Mutation: "DeleteAIProvider", Mode: GenericWriteModeID, NeedsID: true},
	"create:ontology":                {Resource: "ontology", Action: "create", Mutation: "CreateOntology", Mode: GenericWriteModeInput, InjectProjectID: true},
	"update:ontology":                {Resource: "ontology", Action: "update", Mutation: "UpdateOntology", Mode: GenericWriteModeInput, NeedsID: true, InjectProjectID: true},
	"delete:ontology":                {Resource: "ontology", Action: "delete", Mutation: "DeleteOntology", Mode: GenericWriteModeProjectID, NeedsID: true},
	"create:ontology_fast_lookup":    {Resource: "ontology_fast_lookup", Action: "create", Mutation: "CreateOntologyFastLookup", Mode: GenericWriteModeInput, InjectProjectID: true},
	"delete:ontology_fast_lookup":    {Resource: "ontology_fast_lookup", Action: "delete", Mutation: "RemoveOntologyFastLookup", Mode: GenericWriteModeProjectID, NeedsID: true},
	"create:dataset":                 {Resource: "dataset", Action: "create", Mutation: "CreateDataset", Mode: GenericWriteModeInput, InjectProjectID: true},
	"update:dataset":                 {Resource: "dataset", Action: "update", Mutation: "UpdateDataset", Mode: GenericWriteModeInput, NeedsID: true, InjectProjectID: true},
	"delete:dataset":                 {Resource: "dataset", Action: "delete", Mutation: "DeleteDataset", Mode: GenericWriteModeProjectID, NeedsID: true},
	"create:transform":               {Resource: "transform", Action: "create", Mutation: "SaveTransform", Mode: GenericWriteModeInput, InjectProjectID: true},
	"update:transform":               {Resource: "transform", Action: "update", Mutation: "SaveTransform", Mode: GenericWriteModeInput, NeedsID: true, InjectProjectID: true},
	"delete:transform":               {Resource: "transform", Action: "delete", Mutation: "DeleteTransform", Mode: GenericWriteModeProjectID, NeedsID: true},
	"create:folder":                  {Resource: "folder", Action: "create", Mutation: "CreateProjectFolder", Mode: GenericWriteModeInput, InjectProjectID: true},
	"delete:folder":                  {Resource: "folder", Action: "delete", Mutation: "DeleteProjectFolder", Mode: GenericWriteModeProjectID, NeedsID: true},
	"delete:file":                    {Resource: "file", Action: "delete", Mutation: "DeleteProjectFile", Mode: GenericWriteModeProjectID, NeedsID: true},
	"create:function":                {Resource: "function", Action: "create", Mutation: "CreateFunction", Mode: GenericWriteModeInput, InjectProjectID: true},
	"update:function":                {Resource: "function", Action: "update", Mutation: "UpdateFunction", Mode: GenericWriteModeInput, NeedsID: true, InjectProjectID: true},
	"delete:function":                {Resource: "function", Action: "delete", Mutation: "DeleteFunction", Mode: GenericWriteModeProjectID, NeedsID: true},
	"create:source_object":           {Resource: "source_object", Action: "create", Mutation: "CreatePlatformSourceObject", Mode: GenericWriteModeDirect, InjectProjectID: true},
	"update:source_object":           {Resource: "source_object", Action: "update", Mutation: "UpdatePlatformSourceObject", Mode: GenericWriteModeDirect, InjectProjectID: true},
	"delete:source_object":           {Resource: "source_object", Action: "delete", Mutation: "DeletePlatformSourceObject", Mode: GenericWriteModeDirect, InjectProjectID: true},
	"action:run:transform":           {Resource: "transform", Action: "run", Mutation: "RunTransform", Mode: GenericWriteModeProjectID, NeedsID: true},
	"action:upload:file":             {Resource: "file", Action: "upload", Mutation: "UploadProjectFile", Mode: GenericWriteModeFileUpload, InjectProjectID: true},
	"action:rename:file":             {Resource: "file", Action: "rename", Mutation: "RenameProjectFile", Mode: GenericWriteModeProjectIDName, NeedsID: true},
	"action:rename:folder":           {Resource: "folder", Action: "rename", Mutation: "RenameProjectFolder", Mode: GenericWriteModeProjectIDName, NeedsID: true},
	"action:move:file":               {Resource: "file", Action: "move", Mutation: "MoveProjectFile", Mode: GenericWriteModeInput, NeedsID: true, InjectProjectID: true},
	"action:move:folder":             {Resource: "folder", Action: "move", Mutation: "MoveProjectFolder", Mode: GenericWriteModeInput, NeedsID: true, InjectProjectID: true},
	"action:promote_to_dataset:file": {Resource: "file", Action: "promote_to_dataset", Mutation: "PromoteFileToDataset", Mode: GenericWriteModeInput, NeedsID: true, InjectProjectID: true},
	"action:deploy:function":         {Resource: "function", Action: "deploy", Mutation: "DeployFunction", Mode: GenericWriteModeProjectID, NeedsID: true},
	"action:redeploy:function":       {Resource: "function", Action: "redeploy", Mutation: "RedeployFunction", Mode: GenericWriteModeProjectID, NeedsID: true},
}

// PayloadShapes describes payload_json shapes for common hosted platform writes.
var PayloadShapes = map[string]PayloadShape{
	"create:secret": {
		Key: "create:secret", Resource: "secret", Action: "create", Description: "Create secret metadata and value. projectId is injected from the selected workspace.",
		Fields: []PayloadField{
			{Name: "name", Type: "string", Required: true, Description: "Secret name"},
			{Name: "description", Type: "string", Description: "Optional secret description"},
			{Name: "value", Type: "string", Required: true, Secret: true, Description: "Secret value"},
		},
		Examples: []string{`{"name":"OPENAI_API_KEY","description":"OpenAI key","value":"sk-..."}`},
	},
	"update:secret": {
		Key: "update:secret", Resource: "secret", Action: "update", Description: "Update secret metadata or rotate its value. id and projectId are injected.",
		Fields: []PayloadField{
			{Name: "name", Type: "string", Description: "New secret name"},
			{Name: "description", Type: "string", Description: "New secret description"},
			{Name: "value", Type: "string", Secret: true, Description: "New secret value"},
		},
		Examples: []string{`{"description":"rotated July 2026","value":"new-secret"}`},
	},
	"create:ai_provider": {
		Key: "create:ai_provider", Resource: "ai_provider", Action: "create", Description: "Create an AI provider. projectId is injected from the selected workspace.",
		Fields: []PayloadField{
			{Name: "name", Type: "string", Required: true, Description: "Provider display name"},
			{Name: "providerType", Type: "string", Required: true, Description: "Provider type such as openai or anthropic"},
			{Name: "endpoint", Type: "string", Required: true, Description: "Provider API base URL"},
			{Name: "apiKey", Type: "string", Required: true, Secret: true, Description: "Provider API key"},
			{Name: "models", Type: "[string]", Description: "Optional allow-list of model names"},
		},
		Examples: []string{`{"name":"OpenAI","providerType":"openai","endpoint":"https://api.openai.com/v1","apiKey":"sk-...","models":["gpt-4.1"]}`},
	},
	"update:ai_provider": {
		Key: "update:ai_provider", Resource: "ai_provider", Action: "update", Description: "Update an AI provider. id is injected.",
		Fields: []PayloadField{
			{Name: "name", Type: "string", Description: "New provider display name"},
			{Name: "endpoint", Type: "string", Description: "New provider API base URL"},
			{Name: "apiKey", Type: "string", Secret: true, Description: "New provider API key"},
			{Name: "models", Type: "[string]", Description: "Replacement allow-list of model names"},
		},
		Examples: []string{`{"models":["gpt-4.1","gpt-4.1-mini"]}`},
	},
	"create:dataset": {
		Key: "create:dataset", Resource: "dataset", Action: "create", Description: "Create a dataset. projectId is injected from the selected workspace.",
		Fields: []PayloadField{
			{Name: "name", Type: "string", Required: true, Description: "Dataset name"},
			{Name: "description", Type: "string", Description: "Dataset description"},
			{Name: "sourceId", Type: "string", Description: "Optional source id for source-backed datasets"},
			{Name: "sourceObjectRef", Type: "SourceObjectRefInput", Description: "Optional source object reference"},
			{Name: "columns", Type: "[ColumnDefInput]", Description: "Manual schema columns"},
			{Name: "schemaMode", Type: "string", Description: "Schema mode such as manual"},
		},
		Examples: []string{`{"name":"Customers","schemaMode":"manual","columns":[{"name":"id","type":"text","isNullable":false,"isPrimary":true}]}`},
	},
	"update:dataset": {
		Key: "update:dataset", Resource: "dataset", Action: "update", Description: "Update a dataset. id and projectId are injected.",
		Fields: []PayloadField{
			{Name: "name", Type: "string", Description: "New dataset name"},
			{Name: "description", Type: "string", Description: "New dataset description"},
			{Name: "columns", Type: "[ColumnDefInput]", Description: "Replacement manual schema columns"},
			{Name: "schemaMode", Type: "string", Description: "New schema mode"},
		},
		Examples: []string{`{"description":"Customer import","schemaMode":"manual"}`},
	},
	"action:run:transform": {
		Key: "action:run:transform", Resource: "transform", Action: "run", Description: "Run an existing transform. id and projectId are injected.",
	},
	"action:upload:file": {
		Key: "action:upload:file", Resource: "file", Action: "upload", Description: "Upload a local file. projectId is injected.",
		Fields: []PayloadField{
			{Name: "file_path", Type: "string", Required: true, Description: "Local file path"},
			{Name: "folderId", Type: "string", Description: "Destination folder id"},
		},
		Examples: []string{`{"file_path":"./customers.csv","folderId":"folder_123"}`},
	},
	"action:rename:file": {
		Key: "action:rename:file", Resource: "file", Action: "rename", Description: "Rename a file. id and projectId are injected.",
		Fields:   []PayloadField{{Name: "name", Type: "string", Required: true, Description: "New file name"}},
		Examples: []string{`{"name":"customers-2026.csv"}`},
	},
	"action:move:file": {
		Key: "action:move:file", Resource: "file", Action: "move", Description: "Move a file. id and projectId are injected.",
		Fields:   []PayloadField{{Name: "newFolderId", Type: "string|null", Description: "Destination folder id; null or empty means project root"}},
		Examples: []string{`{"newFolderId":"folder_123"}`},
	},
	"action:deploy:function": {
		Key: "action:deploy:function", Resource: "function", Action: "deploy", Description: "Deploy an existing function. id and projectId are injected.",
	},
	"action:redeploy:function": {
		Key: "action:redeploy:function", Resource: "function", Action: "redeploy", Description: "Redeploy an existing function. id and projectId are injected.",
	},
}
