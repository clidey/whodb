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
	"sort"
)

// User is the platform identity returned by WhoDB.
type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

// PlatformManifest describes the hosted CLI contract published by WhoDB.
type PlatformManifest struct {
	PlatformVersion         string                      `json:"platformVersion"`
	ManifestProtocolVersion string                      `json:"manifestProtocolVersion"`
	GeneratedAt             string                      `json:"generatedAt"`
	Operations              []PlatformManifestOperation `json:"operations"`
	Types                   []PlatformManifestType      `json:"types"`
}

// PlatformManifestOperation describes one hosted operation available to the CLI.
type PlatformManifestOperation struct {
	Name    string                  `json:"name"`
	Kind    string                  `json:"kind"`
	Args    []PlatformManifestField `json:"args"`
	Returns string                  `json:"returns"`
}

// PlatformManifestType describes one hosted object type available to the CLI.
type PlatformManifestType struct {
	Name   string                  `json:"name"`
	Fields []PlatformManifestField `json:"fields"`
}

// PlatformManifestField describes one field or argument in the hosted CLI contract.
type PlatformManifestField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	List     bool   `json:"list"`
}

// HasOperation reports whether the hosted platform published an operation.
func (m *PlatformManifest) HasOperation(kind, name string) bool {
	if m == nil {
		return false
	}
	for _, operation := range m.Operations {
		if operation.Kind == kind && operation.Name == name {
			return true
		}
	}
	return false
}

// RequireOperation returns an error if the hosted platform did not publish an operation.
func (m *PlatformManifest) RequireOperation(kind, name, feature string) error {
	if m == nil {
		return nil
	}
	if m.HasOperation(kind, name) {
		return nil
	}
	return UnsupportedFeatureError{
		Feature:   feature,
		Operation: kind + "." + name,
	}
}

// SelectFields returns desired fields that are present in the hosted type.
func (m *PlatformManifest) SelectFields(typeName string, desired []string) []string {
	if m == nil {
		return append([]string(nil), desired...)
	}
	available := map[string]struct{}{}
	for _, typ := range m.Types {
		if typ.Name != typeName {
			continue
		}
		for _, field := range typ.Fields {
			available[field.Name] = struct{}{}
		}
		break
	}
	if len(available) == 0 {
		return append([]string(nil), desired...)
	}
	selected := make([]string, 0, len(desired))
	for _, field := range desired {
		if _, ok := available[field]; ok {
			selected = append(selected, field)
		}
	}
	return selected
}

// UnsupportedFeatureError reports that the host does not publish a CLI feature.
type UnsupportedFeatureError struct {
	Feature   string
	Operation string
}

func (e UnsupportedFeatureError) Error() string {
	if e.Feature == "" {
		return fmt.Sprintf("this WhoDB host does not support %s yet", e.Operation)
	}
	return fmt.Sprintf("this WhoDB host does not support %s yet", e.Feature)
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

// SourceType describes one hosted WhoDB source type available for creation.
type SourceType struct {
	ID               string                  `json:"id"`
	Label            string                  `json:"label"`
	Connector        string                  `json:"connector"`
	Category         string                  `json:"category"`
	ConnectionFields []SourceConnectionField `json:"connectionFields"`
}

// SourceConnectionField describes one source credential/configuration field.
type SourceConnectionField struct {
	Key             string  `json:"key"`
	Kind            string  `json:"kind"`
	Section         string  `json:"section"`
	Required        bool    `json:"required"`
	LabelKey        string  `json:"labelKey"`
	PlaceholderKey  *string `json:"placeholderKey,omitempty"`
	DefaultValue    *string `json:"defaultValue,omitempty"`
	SupportsOptions bool    `json:"supportsOptions"`
}

// SourceConfig is a hosted source's connection configuration.
type SourceConfig struct {
	Hostname string            `json:"hostname"`
	Port     string            `json:"port"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Database string            `json:"database"`
	Advanced map[string]string `json:"advanced"`
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

// UpdateSourceInput describes source metadata/config fields to update.
type UpdateSourceInput struct {
	ID     string
	Name   *string
	Config *SourceConfig
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
	return map[string]any{
		"projectId":    input.ProjectID,
		"name":         input.Name,
		"databaseType": input.DatabaseType,
		"hostname":     input.Hostname,
		"port":         input.Port,
		"username":     input.Username,
		"password":     input.Password,
		"database":     input.Database,
		"advanced":     advancedGraphQLInput(input.Advanced),
	}
}

func (input CreateSourceInput) sourceLoginInput() map[string]any {
	values := map[string]string{}
	for key, value := range input.Advanced {
		values[key] = value
	}
	for key, value := range map[string]string{
		"Hostname": input.Hostname,
		"Port":     input.Port,
		"Username": input.Username,
		"Password": input.Password,
		"Database": input.Database,
	} {
		if value != "" {
			values[key] = value
		}
	}

	return map[string]any{
		"SourceType": input.DatabaseType,
		"Values":     advancedGraphQLInput(values),
	}
}

func (input UpdateSourceInput) graphQLInput() map[string]any {
	result := map[string]any{
		"id": input.ID,
	}
	if input.Name != nil {
		result["name"] = *input.Name
	}
	if input.Config != nil {
		result["config"] = input.Config.graphQLInput()
	}
	return result
}

func (config SourceConfig) graphQLInput() map[string]any {
	return map[string]any{
		"hostname": config.Hostname,
		"port":     config.Port,
		"username": config.Username,
		"password": config.Password,
		"database": config.Database,
		"advanced": advancedGraphQLInput(config.Advanced),
	}
}

func advancedGraphQLInput(advanced map[string]string) []recordInput {
	records := make([]recordInput, 0, len(advanced))
	keys := make([]string, 0, len(advanced))
	for key := range advanced {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		records = append(records, recordInput{Key: key, Value: advanced[key]})
	}
	return records
}
