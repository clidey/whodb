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

package mcp

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// PlatformOutputScope identifies the hosted workspace used for a platform MCP read.
type PlatformOutputScope struct {
	Host        string `json:"host,omitempty"`
	OrgID       string `json:"org_id,omitempty"`
	OrgName     string `json:"org_name,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
}

func platformScope(session *platformToolSession) *PlatformOutputScope {
	if session == nil {
		return nil
	}
	return &PlatformOutputScope{
		Host:        session.Host.URL,
		OrgID:       session.Host.DefaultOrgID,
		OrgName:     session.Host.DefaultOrgName,
		ProjectID:   session.Host.DefaultProjectID,
		ProjectName: session.Host.DefaultProjectName,
	}
}

func platformReadOutput(session *platformToolSession, toolName string, data any, count int, truncated bool, requestID string, fields []string) PlatformReadOutput {
	projected, warnings := projectOutputFields(data, fields)
	if count == 0 && isListLike(data) {
		warnings = append(warnings, emptyPlatformReadWarning(toolName, platformScope(session)))
	}
	return PlatformReadOutput{
		Data:      projected,
		Items:     listItems(data, projected),
		Count:     count,
		Scope:     platformScope(session),
		Fields:    normalizeOutputFields(fields),
		Warnings:  warnings,
		Truncated: truncated,
		RequestID: requestID,
	}
}

func projectOutputFields(data any, fields []string) (any, []string) {
	fields = normalizeOutputFields(fields)
	if len(fields) == 0 || data == nil {
		return data, nil
	}

	var decoded any
	raw, err := json.Marshal(data)
	if err != nil {
		return data, []string{fmt.Sprintf("could not apply fields projection: %v", err)}
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return data, []string{fmt.Sprintf("could not apply fields projection: %v", err)}
	}

	projected, missing := projectJSONValue(decoded, fields)
	warnings := make([]string, 0, 1)
	if len(missing) > 0 {
		sort.Strings(missing)
		warnings = append(warnings, "ignored unknown fields: "+strings.Join(missing, ", "))
	}
	return projected, warnings
}

func normalizeOutputFields(fields []string) []string {
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

func projectJSONValue(value any, fields []string) (any, []string) {
	switch typed := value.(type) {
	case []any:
		projected := make([]any, 0, len(typed))
		missingSet := map[string]struct{}{}
		for _, item := range typed {
			projectedItem, missing := projectJSONObject(item, fields)
			projected = append(projected, projectedItem)
			for _, field := range missing {
				missingSet[field] = struct{}{}
			}
		}
		return projected, sortedKeys(missingSet)
	default:
		return projectJSONObject(typed, fields)
	}
}

func projectJSONObject(value any, fields []string) (any, []string) {
	object, ok := value.(map[string]any)
	if !ok {
		return value, []string{"fields projection only applies to object or list outputs"}
	}
	projected := make(map[string]any, len(fields))
	missing := make([]string, 0)
	for _, field := range fields {
		if value, ok := object[field]; ok {
			projected[field] = value
			continue
		}
		missing = append(missing, field)
	}
	return projected, missing
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isListLike(value any) bool {
	if value == nil {
		return false
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
}

func listItems(original, projected any) []map[string]any {
	if !isListLike(original) {
		return nil
	}
	raw, err := json.Marshal(projected)
	if err != nil {
		return nil
	}
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	return items
}

func emptyPlatformReadWarning(toolName string, scope *PlatformOutputScope) string {
	location := "the selected workspace"
	if scope != nil {
		switch {
		case scope.OrgName != "" && scope.ProjectName != "":
			location = scope.OrgName + " / " + scope.ProjectName
		case scope.ProjectName != "":
			location = scope.ProjectName
		case scope.OrgName != "":
			location = scope.OrgName
		}
	}
	return fmt.Sprintf("no %s results found in %s", strings.TrimPrefix(toolName, "platform_"), location)
}
