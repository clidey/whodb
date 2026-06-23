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

import "strings"

var sourceContentSelectionFields = map[string]string{
	"text":       "text: Text",
	"mimeType":   "mimeType: MIMEType",
	"isBinary":   "isBinary: IsBinary",
	"sizeBytes":  "sizeBytes: SizeBytes",
	"truncated":  "truncated: Truncated",
	"fileName":   "fileName: FileName",
	"modifiedAt": "modifiedAt: ModifiedAt",
}

var functionSelectionFields = map[string]string{
	"id":                  "id",
	"projectId":           "projectId",
	"name":                "name",
	"description":         "description",
	"language":            "language",
	"entryPoint":          "entryPoint",
	"timeoutSeconds":      "timeoutSeconds",
	"memory":              "memory",
	"cpu":                 "cpu",
	"createdBy":           "createdBy",
	"createdAt":           "createdAt",
	"updatedAt":           "updatedAt",
	"files":               "files {\n    id\n    path\n    content\n  }",
	"dependencies":        "dependencies {\n    id\n    name\n    version\n  }",
	"providerIds":         "providerIds",
	"ontologyIds":         "ontologyIds",
	"readOnlyOntologyIds": "readOnlyOntologyIds",
	"providerConfigs":     "providerConfigs {\n    providerId\n    model\n  }",
	"secretBindings":      "secretBindings {\n    name\n    secretId\n    target\n  }",
	"defaultMaxTokens":    "defaultMaxTokens",
	"defaultTemperature":  "defaultTemperature",
	"isDeployed":          "isDeployed",
}

var folderContentsSelectionFields = map[string]string{
	"folders":     "folders {\n" + indentSelection(projectFolderFields, 4) + "\n  }",
	"files":       "files {\n" + indentSelection(projectFileFields, 4) + "\n  }",
	"breadcrumbs": "breadcrumbs {\n" + indentSelection(projectFolderFields, 4) + "\n  }",
	"storageUsed": "storageUsed",
}

var filePreviewSelectionFields = map[string]string{
	"mimeType":  "mimeType",
	"sizeBytes": "sizeBytes",
	"isTabular": "isTabular",
	"tabular": `tabular {
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
  }`,
	"textContent": "textContent",
}

func operationPlatformSourceContentForFields(fields []string) string {
	if len(normalizeSelectionFields(fields)) == 0 {
		return operationPlatformSourceContent
	}
	return `
query CLIPlatformSourceContent($projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!) {
  PlatformSourceContent(projectId: $projectId, sourceId: $sourceId, ref: $ref) {
` + indentSelection(selectGraphQLFields(fields, sourceContentSelectionFields, "fileName"), 4) + `
  }
}
`
}

func operationProjectFunctionsForFields(fields []string) string {
	if len(normalizeSelectionFields(fields)) == 0 {
		return operationProjectFunctions
	}
	return `
query CLIPlatformProjectFunctions($projectId: ID!) {
  ProjectFunctions(projectId: $projectId) {
` + indentSelection(selectGraphQLFields(fields, functionSelectionFields, "id"), 4) + `
  }
}
`
}

func operationFunctionDetailForFields(fields []string) string {
	if len(normalizeSelectionFields(fields)) == 0 {
		return operationFunctionDetail
	}
	return `
query CLIPlatformFunctionDetail($projectId: ID!, $id: ID!) {
  FunctionDetail(projectId: $projectId, id: $id) {
` + indentSelection(selectGraphQLFields(fields, functionSelectionFields, "id"), 4) + `
  }
}
`
}

func operationFolderContentsForFields(fields []string) string {
	if len(normalizeSelectionFields(fields)) == 0 {
		return operationFolderContents
	}
	return `
query CLIPlatformFolderContents($projectId: ID!, $folderId: ID) {
  FolderContents(projectId: $projectId, folderId: $folderId) {
` + indentSelection(selectGraphQLFields(fields, folderContentsSelectionFields, "storageUsed"), 4) + `
  }
}
`
}

func operationFilePreviewForFields(fields []string) string {
	if len(normalizeSelectionFields(fields)) == 0 {
		return operationFilePreview
	}
	return `
query CLIPlatformFilePreview($projectId: ID!, $fileId: ID!, $sheetIndex: Int) {
  FilePreview(projectId: $projectId, fileId: $fileId, sheetIndex: $sheetIndex) {
` + indentSelection(selectGraphQLFields(fields, filePreviewSelectionFields, "mimeType"), 4) + `
  }
}
`
}

func selectGraphQLFields(fields []string, allowed map[string]string, fallback string) string {
	normalized := normalizeSelectionFields(fields)
	selected := make([]string, 0, len(normalized))
	for _, field := range normalized {
		selection, ok := allowed[field]
		if !ok {
			continue
		}
		selected = append(selected, selection)
	}
	if len(selected) == 0 {
		return allowed[fallback]
	}
	return strings.Join(selected, "\n")
}

func normalizeSelectionFields(fields []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		normalized = append(normalized, field)
	}
	return normalized
}

func indentSelection(selection string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.Trim(selection, "\n"), "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}
