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

const operationMe = `
query CLIPlatformMe {
  Me {
    id
    email
    displayName
    orgId
  }
}
`

const operationOrganizations = `
query CLIPlatformOrganizations {
  MyOrganizations {
    id
    name
    slug
  }
}
`

const operationProjects = `
query CLIPlatformProjects($orgId: ID!) {
  Projects(orgId: $orgId) {
    id
    orgId
    name
    slug
    description
  }
}
`

const operationSwitchOrganization = `
mutation CLIPlatformSwitchOrganization($orgId: ID!) {
  SwitchOrganization(orgId: $orgId) {
    id
    name
    slug
  }
}
`

const operationProjectSources = `
query CLIPlatformProjectSources($projectId: ID!) {
  ProjectSources(projectId: $projectId) {
    id
    projectId
    name
    databaseType
    createdBy
    createdAt
  }
}
`

const operationSourceTypes = `
query CLIPlatformSourceTypes {
  SourceTypes {
    id: Id
    label: Label
    connector: Connector
    category: Category
    connectionFields: ConnectionFields {
      key: Key
      kind: Kind
      section: Section
      required: Required
      labelKey: LabelKey
      placeholderKey: PlaceholderKey
      defaultValue: DefaultValue
      supportsOptions: SupportsOptions
    }
  }
}
`

const operationCreateSource = `
mutation CLIPlatformCreateSource($input: CreateSourceInput!) {
  CreateSource(input: $input) {
    id
    projectId
    name
    databaseType
    createdBy
    createdAt
  }
}
`

const operationDeleteSource = `
mutation CLIPlatformDeleteSource($projectId: ID!, $id: ID!) {
  DeleteSource(projectId: $projectId, id: $id) {
    Status
  }
}
`

const sourceObjectFields = `
  ref: Ref {
    kind: Kind
    locator: Locator
    path: Path
  }
  kind: Kind
  name: Name
  path: Path
  hasChildren: HasChildren
  actions: Actions
  metadata: Metadata {
    key: Key
    value: Value
  }
`

const columnFields = `
  type: Type
  name: Name
  metadataFidelity: MetadataFidelity
  isPrimary: IsPrimary
  isForeignKey: IsForeignKey
  referencedTable: ReferencedTable
  referencedColumn: ReferencedColumn
  length: Length
  precision: Precision
  scale: Scale
`

const operationPlatformSourceObjects = `
query CLIPlatformSourceObjects($projectId: ID!, $sourceId: ID!, $parent: SourceObjectRefInput, $kinds: [SourceObjectKind!], $pageSize: Int, $pageOffset: Int) {
  PlatformSourceObjects(projectId: $projectId, sourceId: $sourceId, parent: $parent, kinds: $kinds, pageSize: $pageSize, pageOffset: $pageOffset) {
` + sourceObjectFields + `
  }
}
`

const operationPlatformSourceColumns = `
query CLIPlatformSourceColumns($projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!) {
  PlatformSourceColumns(projectId: $projectId, sourceId: $sourceId, ref: $ref) {
` + columnFields + `
  }
}
`

const operationPlatformSourceRows = `
query CLIPlatformSourceRows($projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!, $pageSize: Int!, $pageOffset: Int!) {
  PlatformSourceRows(projectId: $projectId, sourceId: $sourceId, ref: $ref, pageSize: $pageSize, pageOffset: $pageOffset) {
    columns: Columns {
` + columnFields + `
    }
    rows: Rows
    disableUpdate: DisableUpdate
    totalCount: TotalCount
  }
}
`

var platformOperations = map[string]string{
	"me":                  operationMe,
	"organizations":       operationOrganizations,
	"projects":            operationProjects,
	"switch_organization": operationSwitchOrganization,
	"project_sources":     operationProjectSources,
	"source_types":        operationSourceTypes,
	"create_source":       operationCreateSource,
	"delete_source":       operationDeleteSource,
	"source_objects":      operationPlatformSourceObjects,
	"source_columns":      operationPlatformSourceColumns,
	"source_rows":         operationPlatformSourceRows,
}
