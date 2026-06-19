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
	"fmt"
	"strings"
)

const operationMe = `
query CLIPlatformMe {
  Me {
    %s
  }
}
`

const operationPlatformManifest = `
query CLIPlatformManifest {
  PlatformManifest {
    platformVersion
    manifestProtocolVersion
    generatedAt
    operations {
      name
      kind
      returns
      args {
        name
        type
        required
        list
      }
    }
    types {
      name
      fields {
        name
        type
        required
        list
      }
    }
  }
}
`

const operationPlatformVersion = `
query CLIPlatformVersion {
  Version
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

const operationProjectSources = `
query CLIPlatformProjectSources($orgId: ID, $projectId: ID!) {
  ProjectSources(orgId: $orgId, projectId: $projectId) {
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

const operationSourceConfig = `
query CLIPlatformSourceConfig($orgId: ID, $projectId: ID!, $sourceId: ID!) {
  SourceConfig(orgId: $orgId, projectId: $projectId, sourceId: $sourceId) {
    hostname
    port
    username
    password
    database
    advanced {
      key: Key
      value: Value
    }
  }
}
`

const operationUpdateSource = `
mutation CLIPlatformUpdateSource($input: UpdateSourceInput!) {
  UpdateSource(input: $input) {
    id
    projectId
    name
    databaseType
    createdBy
    createdAt
  }
}
`

const operationTestSourceConnection = `
mutation CLIPlatformTestSourceConnection($credentials: SourceLoginInput!) {
  TestSourceConnection(credentials: $credentials) {
    Status
  }
}
`

const operationDeleteSource = `
mutation CLIPlatformDeleteSource($orgId: ID, $projectId: ID!, $id: ID!) {
  DeleteSource(orgId: $orgId, projectId: $projectId, id: $id) {
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
query CLIPlatformSourceObjects($orgId: ID, $projectId: ID!, $sourceId: ID!, $parent: SourceObjectRefInput, $kinds: [SourceObjectKind!], $pageSize: Int, $pageOffset: Int) {
  PlatformSourceObjects(orgId: $orgId, projectId: $projectId, sourceId: $sourceId, parent: $parent, kinds: $kinds, pageSize: $pageSize, pageOffset: $pageOffset) {
` + sourceObjectFields + `
  }
}
`

const operationPlatformSourceColumns = `
query CLIPlatformSourceColumns($orgId: ID, $projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!) {
  PlatformSourceColumns(orgId: $orgId, projectId: $projectId, sourceId: $sourceId, ref: $ref) {
` + columnFields + `
  }
}
`

const operationPlatformSourceRows = `
query CLIPlatformSourceRows($orgId: ID, $projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!, $pageSize: Int!, $pageOffset: Int!) {
  PlatformSourceRows(orgId: $orgId, projectId: $projectId, sourceId: $sourceId, ref: $ref, pageSize: $pageSize, pageOffset: $pageOffset) {
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
	"me":                operationMeForFields([]string{"id", "email", "displayName"}),
	"platform_manifest": operationPlatformManifest,
	"platform_version":  operationPlatformVersion,
	"organizations":     operationOrganizations,
	"projects":          operationProjects,
	"project_sources":   operationProjectSources,
	"source_types":      operationSourceTypes,
	"create_source":     operationCreateSource,
	"source_config":     operationSourceConfig,
	"update_source":     operationUpdateSource,
	"test_source":       operationTestSourceConnection,
	"delete_source":     operationDeleteSource,
	"source_objects":    operationPlatformSourceObjects,
	"source_columns":    operationPlatformSourceColumns,
	"source_rows":       operationPlatformSourceRows,
}

func operationMeForFields(fields []string) string {
	return fmt.Sprintf(operationMe, strings.Join(fields, "\n    "))
}
