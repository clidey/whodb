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

const sourceFieldConstraintsFields = `
  name: Name
  type: Type
  metadataFidelity: MetadataFidelity
  nullable: Nullable
  primary: Primary
  unique: Unique
  identity: Identity
  defaultValue: DefaultValue
  allowedValues: AllowedValues
  checkMin: CheckMin
  checkMax: CheckMax
  foreignKey: ForeignKey {
    table: Table
    column: Column
  }
  length: Length
  precision: Precision
  scale: Scale
`

const sourceContentFields = `
  text: Text
  mimeType: MIMEType
  isBinary: IsBinary
  sizeBytes: SizeBytes
  truncated: Truncated
  fileName: FileName
  modifiedAt: ModifiedAt
`

const ontologyFields = `
  id
  projectId
  apiName
  displayName
  pluralDisplayName
  description
  primaryKey
  sourceId
  tableName
  schemaName
  status
  icon
  color
  createdAt
  updatedAt
  properties {
    id
    apiName
    displayName
    description
    columnName
    dataType
    arrayElementType
    isRequired
    visibility
    isSearchable
    isSortable
    isEditOnly
    sortOrder
  }
  links {
    id
    apiName
    targetOntologyApiName
    cardinality
    foreignKeyProperty
    targetForeignKeyProperty
    joinTable
    sourceColumnInJoinTable
    targetColumnInJoinTable
    displayName
    pluralDisplayName
    reverseDisplayName
  }
`

const datasetFields = `
  id
  projectId
  sourceId
  name
  description
  schema {
    name
    type
    isNullable
    isPrimary
  }
  schemaMode
  ownerId
  rowCount
  sizeBytes
  createdAt
  updatedAt
`

const datasetQueryResultFields = `
  columns
  rows
  total
`

const lineageGraphFields = `
  nodes {
    id
    nodeType
    name
  }
  edges {
    sourceId
    sourceType
    targetId
    targetType
    createdAt
  }
`

const transformFields = `
  id
  projectId
  name
  description
  graphJson
  scheduleCron
  triggerMode
  createdAt
  updatedAt
`

const functionFields = `
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
  files {
    id
    path
    content
  }
  dependencies {
    id
    name
    version
  }
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

const projectFileFields = `
  id
  projectId
  folderId
  name
  mimeType
  sizeBytes
  isTabular
  rowCount
  columnCount
  datasetId
  uploadedBy
  createdAt
  updatedAt
`

const projectFolderFields = `
  id
  projectId
  parentId
  name
  createdBy
  createdAt
`

const operationProjectSecrets = `
query CLIPlatformProjectSecrets($projectId: ID!) {
  ProjectSecrets(projectId: $projectId) {
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
  }
}
`

const operationPlatformSourceFieldConstraints = `
query CLIPlatformSourceFieldConstraints($projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!) {
  PlatformSourceFieldConstraints(projectId: $projectId, sourceId: $sourceId, ref: $ref) {
` + sourceFieldConstraintsFields + `
  }
}
`

const operationPlatformSourceContent = `
query CLIPlatformSourceContent($projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!) {
  PlatformSourceContent(projectId: $projectId, sourceId: $sourceId, ref: $ref) {
` + sourceContentFields + `
  }
}
`

const operationPlatformAIProviders = `
query CLIPlatformAIProviders($projectId: ID!) {
  PlatformAIProviders(projectId: $projectId) {
    id
    projectId
    name
    providerType
    endpoint
    createdBy
    createdAt
    updatedAt
  }
}
`

const operationAIProviderModels = `
query CLIPlatformAIProviderModels($projectId: ID!, $providerId: ID!) {
  AIProviderModels(projectId: $projectId, providerId: $providerId)
}
`

const operationOntologyEntities = `
query CLIPlatformOntologyEntities($projectId: ID!) {
  OntologyEntities(projectId: $projectId) {
` + ontologyFields + `
  }
}
`

const operationOntologyEntity = `
query CLIPlatformOntologyEntity($projectId: ID!, $id: ID!) {
  OntologyEntity(projectId: $projectId, id: $id) {
` + ontologyFields + `
  }
}
`

const operationOntologyFastLookups = `
query CLIPlatformOntologyFastLookups($projectId: ID!, $entityId: ID!) {
  OntologyFastLookups(projectId: $projectId, entityId: $entityId) {
    id
    entityId
    displayName
    fields
    status
    reason
    createdAt
    updatedAt
  }
}
`

const operationOntologyFastLookupSuggestions = `
query CLIPlatformOntologyFastLookupSuggestions($projectId: ID!, $entityId: ID!) {
  OntologyFastLookupSuggestions(projectId: $projectId, entityId: $entityId) {
    entityId
    displayName
    fields
    reason
    canCreate
  }
}
`

const operationOntologyRows = `
query CLIPlatformOntologyRows($projectId: ID!, $id: ID!, $pageSize: Int!, $pageOffset: Int!) {
  OntologyRows(projectId: $projectId, id: $id, pageSize: $pageSize, pageOffset: $pageOffset) {
` + datasetQueryResultFields + `
  }
}
`

const operationOntologyFollowLink = `
query CLIPlatformOntologyFollowLink($projectId: ID!, $entityId: ID!, $pk: String!, $linkApiName: String!, $pageSize: Int!, $pageOffset: Int!) {
  OntologyFollowLink(projectId: $projectId, entityId: $entityId, pk: $pk, linkApiName: $linkApiName, pageSize: $pageSize, pageOffset: $pageOffset) {
` + datasetQueryResultFields + `
  }
}
`

const operationProjectDatasets = `
query CLIPlatformProjectDatasets($projectId: ID!) {
  ProjectDatasets(projectId: $projectId) {
` + datasetFields + `
  }
}
`

const operationDatasetDetail = `
query CLIPlatformDatasetDetail($projectId: ID!, $id: ID!) {
  DatasetDetail(projectId: $projectId, id: $id) {
` + datasetFields + `
  }
}
`

const operationQueryDataset = `
query CLIPlatformQueryDataset($input: QueryDatasetInput!) {
  QueryDataset(input: $input) {
` + datasetQueryResultFields + `
  }
}
`

const operationLineageGraph = `
query CLIPlatformLineageGraph($projectId: ID!, $rootId: ID!, $rootType: String!, $direction: String, $maxDepth: Int) {
  LineageGraph(projectId: $projectId, rootId: $rootId, rootType: $rootType, direction: $direction, maxDepth: $maxDepth) {
` + lineageGraphFields + `
  }
}
`

const operationLineageNeighbors = `
query CLIPlatformLineageNeighbors($projectId: ID!, $nodeId: ID!, $nodeType: String!) {
  LineageNeighbors(projectId: $projectId, nodeId: $nodeId, nodeType: $nodeType) {
` + lineageGraphFields + `
  }
}
`

const operationProjectLineage = `
query CLIPlatformProjectLineage($projectId: ID!) {
  ProjectLineage(projectId: $projectId) {
` + lineageGraphFields + `
  }
}
`

const operationProjectTransforms = `
query CLIPlatformProjectTransforms($projectId: ID!) {
  ProjectTransforms(projectId: $projectId) {
` + transformFields + `
  }
}
`

const operationTransformRuns = `
query CLIPlatformTransformRuns($projectId: ID!, $transformId: ID!, $limit: Int) {
  TransformRuns(projectId: $projectId, transformId: $transformId, limit: $limit) {
    id
    transformId
    status
    errorMessage
    triggeredBy
    startedAt
    completedAt
  }
}
`

const operationProjectFunctions = `
query CLIPlatformProjectFunctions($projectId: ID!) {
  ProjectFunctions(projectId: $projectId) {
` + functionFields + `
  }
}
`

const operationFunctionDetail = `
query CLIPlatformFunctionDetail($projectId: ID!, $id: ID!) {
  FunctionDetail(projectId: $projectId, id: $id) {
` + functionFields + `
  }
}
`

const operationFolderContents = `
query CLIPlatformFolderContents($projectId: ID!, $folderId: ID) {
  FolderContents(projectId: $projectId, folderId: $folderId) {
    folders {
` + projectFolderFields + `
    }
    files {
` + projectFileFields + `
    }
    breadcrumbs {
` + projectFolderFields + `
    }
    storageUsed
  }
}
`

const operationFilePreview = `
query CLIPlatformFilePreview($projectId: ID!, $fileId: ID!, $sheetIndex: Int) {
  FilePreview(projectId: $projectId, fileId: $fileId, sheetIndex: $sheetIndex) {
    mimeType
    sizeBytes
    isTabular
    tabular {
      columns {
        name
        type
      }
      rows
      total
      sheetName
      sheetIndex
      sheetCount
      sheetNames
    }
    textContent
  }
}
`

const operationSearchProjectFiles = `
query CLIPlatformSearchProjectFiles($projectId: ID!, $query: String!) {
  SearchProjectFiles(projectId: $projectId, query: $query) {
` + projectFileFields + `
  }
}
`

const operationProjectTabularFiles = `
query CLIPlatformProjectTabularFiles($projectId: ID!) {
  ProjectTabularFiles(projectId: $projectId) {
` + projectFileFields + `
  }
}
`

const operationProjectStorageUsage = `
query CLIPlatformProjectStorageUsage($projectId: ID!) {
  ProjectStorageUsage(projectId: $projectId)
}
`

const operationExecuteFunction = `
mutation CLIPlatformExecuteFunction($projectId: ID!, $functionId: ID!, $input: String!, $inputFileIds: [ID!]) {
  ExecuteFunction(projectId: $projectId, functionId: $functionId, input: $input, inputFileIds: $inputFileIds) {
    output
    logs
    durationMs
    success
    error
  }
}
`

func init() {
	platformOperations["project_secrets"] = operationProjectSecrets
	platformOperations["source_constraints"] = operationPlatformSourceFieldConstraints
	platformOperations["source_content"] = operationPlatformSourceContent
	platformOperations["ai_providers"] = operationPlatformAIProviders
	platformOperations["ai_provider_models"] = operationAIProviderModels
	platformOperations["ontology_entities"] = operationOntologyEntities
	platformOperations["ontology_entity"] = operationOntologyEntity
	platformOperations["ontology_fast_lookups"] = operationOntologyFastLookups
	platformOperations["ontology_fast_lookup_suggestions"] = operationOntologyFastLookupSuggestions
	platformOperations["ontology_rows"] = operationOntologyRows
	platformOperations["ontology_follow_link"] = operationOntologyFollowLink
	platformOperations["datasets"] = operationProjectDatasets
	platformOperations["dataset"] = operationDatasetDetail
	platformOperations["dataset_rows"] = operationQueryDataset
	platformOperations["lineage"] = operationLineageGraph
	platformOperations["lineage_neighbors"] = operationLineageNeighbors
	platformOperations["project_lineage"] = operationProjectLineage
	platformOperations["transforms"] = operationProjectTransforms
	platformOperations["transform_runs"] = operationTransformRuns
	platformOperations["functions"] = operationProjectFunctions
	platformOperations["function"] = operationFunctionDetail
	platformOperations["folder_contents"] = operationFolderContents
	platformOperations["file_preview"] = operationFilePreview
	platformOperations["file_search"] = operationSearchProjectFiles
	platformOperations["tabular_files"] = operationProjectTabularFiles
	platformOperations["storage_usage"] = operationProjectStorageUsage
	platformOperations["function_execute"] = operationExecuteFunction
}
