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

package engine

import "github.com/clidey/whodb/core/src/source"

// StorageUnit aliases the source-owned storage unit type for compatibility.
type StorageUnit = source.StorageUnit

// Record aliases the source-owned record type for compatibility.
type Record = source.Record

// ExternalModel aliases the source-owned external AI model configuration for
// compatibility.
type ExternalModel = source.ExternalModel

// Column aliases the source-owned column type for compatibility.
type Column = source.Column

// FieldConstraints aliases the source-owned normalized field constraints type.
type FieldConstraints = source.FieldConstraints

// ObjectFieldConstraints aliases source-owned object field constraints.
type ObjectFieldConstraints = source.ObjectFieldConstraints

// GetRowsResult aliases the source-owned row result type for compatibility.
type GetRowsResult = source.RowsResult

// GraphUnitRelationshipType aliases the source-owned graph relationship type.
type GraphUnitRelationshipType = source.GraphRelationshipType

// GraphUnitRelationship aliases the source-owned graph relationship type.
type GraphUnitRelationship = source.GraphRelationship

// GraphUnit aliases the source-owned graph unit type for compatibility.
type GraphUnit = source.GraphUnit

// ChatMessage aliases the source-owned chat message type for compatibility.
type ChatMessage = source.ChatMessage

// ForeignKeyRelationship aliases the source-owned FK relationship type.
type ForeignKeyRelationship = source.ForeignKeyRelationship

// SSLStatus aliases the source-owned SSL status type for compatibility.
type SSLStatus = source.SSLStatus

// TypeCategory aliases the source-owned type category.
type TypeCategory = source.TypeCategory

// TypeDefinition aliases the source-owned type definition.
type TypeDefinition = source.TypeDefinition

// ObjectCreationMetadata aliases the source-owned object creation metadata.
type ObjectCreationMetadata = source.ObjectCreationMetadata

// ColumnCreationCapabilities aliases source-owned column creation capabilities.
type ColumnCreationCapabilities = source.ColumnCreationCapabilities

// TableCreationCapabilities aliases source-owned table creation capabilities.
type TableCreationCapabilities = source.TableCreationCapabilities

// CreationOptionDefinition aliases a source-owned create option definition.
type CreationOptionDefinition = source.CreationOptionDefinition

// ObjectDefinition aliases the source-owned object definition.
type ObjectDefinition = source.ObjectDefinition

// ColumnDefinition aliases the source-owned column definition.
type ColumnDefinition = source.ColumnDefinition

// ForeignKeyDefinition aliases the source-owned foreign key definition.
type ForeignKeyDefinition = source.ForeignKeyDefinition

// RecordsToObjectDefinition normalizes legacy records into an object definition.
func RecordsToObjectDefinition(name string, fields []Record) ObjectDefinition {
	return source.RecordsToObjectDefinition(name, fields)
}

// ObjectDefinitionToRecords converts a typed object definition to records.
func ObjectDefinitionToRecords(definition ObjectDefinition) []Record {
	return source.ObjectDefinitionToRecords(definition)
}

// ColumnDefinitionToRecord converts one typed column definition to a record.
func ColumnDefinitionToRecord(column ColumnDefinition) Record {
	return source.ColumnDefinitionToRecord(column)
}

// NormalizeCreationExtra maps legacy and plugin-specific constraint names to
// canonical create-object keys.
func NormalizeCreationExtra(extra map[string]string) map[string]string {
	return source.NormalizeCreationExtra(extra)
}

// NormalizeFieldConstraints converts legacy constraint maps into typed field
// constraints.
func NormalizeFieldConstraints(constraints map[string]map[string]any) []FieldConstraints {
	return source.NormalizeFieldConstraints(constraints)
}

// CreationListSeparator returns the separator used for list-valued extras.
func CreationListSeparator() string {
	return source.CreationListSeparator()
}

const (
	// TypeCategoryNumeric groups numeric types.
	TypeCategoryNumeric = source.TypeCategoryNumeric
	// TypeCategoryText groups text types.
	TypeCategoryText = source.TypeCategoryText
	// TypeCategoryBinary groups binary types.
	TypeCategoryBinary = source.TypeCategoryBinary
	// TypeCategoryDatetime groups date/time types.
	TypeCategoryDatetime = source.TypeCategoryDatetime
	// TypeCategoryBoolean groups boolean types.
	TypeCategoryBoolean = source.TypeCategoryBoolean
	// TypeCategoryJSON groups JSON/document types.
	TypeCategoryJSON = source.TypeCategoryJSON
	// TypeCategoryOther groups uncategorised types.
	TypeCategoryOther = source.TypeCategoryOther
)

const (
	// GraphUnitRelationshipTypeOneToOne identifies a one-to-one relationship.
	GraphUnitRelationshipTypeOneToOne = source.GraphRelationshipTypeOneToOne
	// GraphUnitRelationshipTypeOneToMany identifies a one-to-many relationship.
	GraphUnitRelationshipTypeOneToMany = source.GraphRelationshipTypeOneToMany
	// GraphUnitRelationshipTypeManyToOne identifies a many-to-one relationship.
	GraphUnitRelationshipTypeManyToOne = source.GraphRelationshipTypeManyToOne
	// GraphUnitRelationshipTypeManyToMany identifies a many-to-many relationship.
	GraphUnitRelationshipTypeManyToMany = source.GraphRelationshipTypeManyToMany
	// GraphUnitRelationshipTypeUnknown identifies an unknown relationship.
	GraphUnitRelationshipTypeUnknown = source.GraphRelationshipTypeUnknown
)
