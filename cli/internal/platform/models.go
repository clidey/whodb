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

import "sort"

// User is the platform identity returned by WhoDB.
type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	OrgID       string `json:"orgId"`
}

// Organization is a WhoDB platform organization visible to the user.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Project is a WhoDB platform project visible to the user.
type Project struct {
	ID          string `json:"id"`
	OrgID       string `json:"orgId"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// Source is a hosted WhoDB project source visible to the authenticated user.
type Source struct {
	ID           string `json:"id"`
	ProjectID    string `json:"projectId"`
	Name         string `json:"name"`
	DatabaseType string `json:"databaseType"`
	CreatedBy    string `json:"createdBy"`
	CreatedAt    string `json:"createdAt"`
}

// SourceObjectKind identifies a source hierarchy object kind.
type SourceObjectKind string

// Record is a key-value metadata entry returned by WhoDB.
type Record struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SourceObjectRef identifies one object inside a source hierarchy.
type SourceObjectRef struct {
	Kind    SourceObjectKind `json:"kind"`
	Locator string           `json:"locator"`
	Path    []string         `json:"path"`
}

// SourceObjectRefInput identifies one source object in GraphQL variables.
type SourceObjectRefInput struct {
	Kind    SourceObjectKind
	Locator string
	Path    []string
}

// SourceObject is a browsable object returned by a hosted source.
type SourceObject struct {
	Ref         SourceObjectRef  `json:"ref"`
	Kind        SourceObjectKind `json:"kind"`
	Name        string           `json:"name"`
	Path        []string         `json:"path"`
	HasChildren bool             `json:"hasChildren"`
	Actions     []string         `json:"actions"`
	Metadata    []Record         `json:"metadata"`
}

// Column describes a column returned by a hosted source.
type Column struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	MetadataFidelity string `json:"metadataFidelity"`
	IsPrimary        bool   `json:"isPrimary"`
	IsForeignKey     bool   `json:"isForeignKey"`
	ReferencedTable  string `json:"referencedTable,omitempty"`
	ReferencedColumn string `json:"referencedColumn,omitempty"`
	Length           *int   `json:"length,omitempty"`
	Precision        *int   `json:"precision,omitempty"`
	Scale            *int   `json:"scale,omitempty"`
}

// RowsResult contains tabular rows returned by a hosted source.
type RowsResult struct {
	Columns       []Column   `json:"columns"`
	Rows          [][]string `json:"rows"`
	DisableUpdate bool       `json:"disableUpdate"`
	TotalCount    int        `json:"totalCount"`
}

// CreateSourceInput describes a hosted WhoDB source to create in one project.
type CreateSourceInput struct {
	ProjectID    string
	Name         string
	DatabaseType string
	Hostname     string
	Port         string
	Username     string
	Password     string
	Database     string
	Advanced     map[string]string
}

type recordInput struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func (input SourceObjectRefInput) graphQLInput() map[string]any {
	return map[string]any{
		"Kind":    input.Kind,
		"Locator": input.Locator,
		"Path":    input.Path,
	}
}

func (input CreateSourceInput) graphQLInput() map[string]any {
	advanced := make([]recordInput, 0, len(input.Advanced))
	keys := make([]string, 0, len(input.Advanced))
	for key := range input.Advanced {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := input.Advanced[key]
		advanced = append(advanced, recordInput{Key: key, Value: value})
	}
	return map[string]any{
		"projectId":    input.ProjectID,
		"name":         input.Name,
		"databaseType": input.DatabaseType,
		"hostname":     input.Hostname,
		"port":         input.Port,
		"username":     input.Username,
		"password":     input.Password,
		"database":     input.Database,
		"advanced":     advanced,
	}
}
