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

var platformOperations = map[string]string{
	"me":                  operationMe,
	"organizations":       operationOrganizations,
	"projects":            operationProjects,
	"switch_organization": operationSwitchOrganization,
	"project_sources":     operationProjectSources,
	"create_source":       operationCreateSource,
	"delete_source":       operationDeleteSource,
}
