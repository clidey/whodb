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
