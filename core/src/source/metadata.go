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

package source

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// NormalizeMetadataTraits fills undeclared metadata traits with explicit values.
func NormalizeMetadataTraits(traits MetadataTraits) MetadataTraits {
	if traits.Columns == "" {
		traits.Columns = MetadataFidelityUnknown
	}
	if traits.Constraints == "" {
		traits.Constraints = MetadataFidelityUnknown
	}
	if traits.Graph == "" {
		traits.Graph = MetadataFidelityUnsupported
	}
	if traits.SystemObjectFiltering == "" {
		if len(traits.HiddenObjectNames) > 0 || len(traits.HiddenObjectPrefixes) > 0 {
			traits.SystemObjectFiltering = MetadataFidelityExact
		} else {
			traits.SystemObjectFiltering = MetadataFidelityUnsupported
		}
	}
	traits.HiddenObjectNames = cloneObjectRules(traits.HiddenObjectNames)
	traits.HiddenObjectPrefixes = cloneObjectRules(traits.HiddenObjectPrefixes)
	return traits
}

// DefaultMetadataTraitsForSpec fills undeclared metadata traits from a source contract.
func DefaultMetadataTraitsForSpec(spec TypeSpec) MetadataTraits {
	metadata := CloneMetadataTraits(spec.Traits.Metadata)
	switch spec.Contract.Model {
	case ModelRelational:
		metadata = metadataWithDefaults(metadata, MetadataFidelityExact, MetadataFidelityExact, graphFidelityForSpec(spec, MetadataFidelityExact))
	case ModelDocument, ModelSearch:
		metadata = metadataWithDefaults(metadata, MetadataFidelitySampled, MetadataFidelitySampled, graphFidelityForSpec(spec, MetadataFidelityInferred))
	case ModelKeyValue:
		metadata = metadataWithDefaults(metadata, MetadataFidelitySynthetic, MetadataFidelityUnsupported, MetadataFidelityUnsupported)
	case ModelGraph:
		metadata = metadataWithDefaults(metadata, MetadataFidelityUnsupported, MetadataFidelityUnsupported, graphFidelityForSpec(spec, MetadataFidelityExact))
	default:
		metadata = metadataWithDefaults(metadata, MetadataFidelityUnsupported, MetadataFidelityUnsupported, MetadataFidelityUnsupported)
	}
	if !spec.Contract.SupportsSurface(SurfaceGraph) {
		metadata.Graph = MetadataFidelityUnsupported
	}
	return NormalizeMetadataTraits(metadata)
}

// CloneMetadataTraits returns a deep copy of source metadata traits.
func CloneMetadataTraits(traits MetadataTraits) MetadataTraits {
	traits.HiddenObjectNames = cloneObjectRules(traits.HiddenObjectNames)
	traits.HiddenObjectPrefixes = cloneObjectRules(traits.HiddenObjectPrefixes)
	return traits
}

// MetadataFidelityOrUnknown returns a non-empty metadata fidelity value.
func MetadataFidelityOrUnknown(fidelity MetadataFidelity) MetadataFidelity {
	if fidelity == "" {
		return MetadataFidelityUnknown
	}
	return fidelity
}

// ApplyColumnMetadataFidelity annotates columns missing per-column fidelity.
func ApplyColumnMetadataFidelity(columns []Column, fidelity MetadataFidelity) []Column {
	fidelity = MetadataFidelityOrUnknown(fidelity)
	for i := range columns {
		if columns[i].MetadataFidelity == "" {
			columns[i].MetadataFidelity = fidelity
		}
	}
	return columns
}

// ApplyFieldConstraintMetadataFidelity annotates constraints missing per-field fidelity.
func ApplyFieldConstraintMetadataFidelity(fields []FieldConstraints, fidelity MetadataFidelity) []FieldConstraints {
	fidelity = MetadataFidelityOrUnknown(fidelity)
	for i := range fields {
		if fields[i].MetadataFidelity == "" {
			fields[i].MetadataFidelity = fidelity
		}
	}
	return fields
}

// ApplyGraphMetadataFidelity annotates graph relationships missing relationship fidelity.
func ApplyGraphMetadataFidelity(units []GraphUnit, fidelity MetadataFidelity) []GraphUnit {
	return NormalizeGraphMetadata(units, fidelity)
}

// NormalizeGraphMetadata fills missing graph relationship metadata with the
// source-declared fidelity and a stable relationship type.
func NormalizeGraphMetadata(units []GraphUnit, fidelity MetadataFidelity) []GraphUnit {
	fidelity = MetadataFidelityOrUnknown(fidelity)
	for i := range units {
		for j := range units[i].Relations {
			if units[i].Relations[j].MetadataFidelity == "" {
				units[i].Relations[j].MetadataFidelity = fidelity
			}
			if units[i].Relations[j].RelationshipType == "" {
				units[i].Relations[j].RelationshipType = GraphRelationshipTypeUnknown
			}
		}
	}
	return units
}

// ValidateColumns reports malformed source column metadata after source-level
// normalization has been applied.
func ValidateColumns(columns []Column) error {
	seen := map[string]struct{}{}
	for _, column := range columns {
		name := strings.TrimSpace(column.Name)
		if name == "" {
			return errors.New("source column has an empty name")
		}
		normalizedName := strings.ToLower(name)
		if _, exists := seen[normalizedName]; exists {
			return fmt.Errorf("source column %q is declared more than once", column.Name)
		}
		seen[normalizedName] = struct{}{}
		if !isValidMetadataFidelity(MetadataFidelityOrUnknown(column.MetadataFidelity)) {
			return fmt.Errorf("source column %q has unsupported metadata fidelity %q", column.Name, column.MetadataFidelity)
		}
		if err := validateColumnShape(column); err != nil {
			return err
		}
	}
	return nil
}

// ValidateFieldConstraints reports malformed normalized source field
// constraints.
func ValidateFieldConstraints(fields []FieldConstraints) error {
	seen := map[string]struct{}{}
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			return errors.New("source field constraint has an empty name")
		}
		normalizedName := strings.ToLower(name)
		if _, exists := seen[normalizedName]; exists {
			return fmt.Errorf("source field constraint %q is declared more than once", field.Name)
		}
		seen[normalizedName] = struct{}{}
		if !isValidMetadataFidelity(MetadataFidelityOrUnknown(field.MetadataFidelity)) {
			return fmt.Errorf("source field constraint %q has unsupported metadata fidelity %q", field.Name, field.MetadataFidelity)
		}
		if err := validateFieldConstraintShape(field); err != nil {
			return err
		}
	}
	return nil
}

// ValidateGraphUnits reports malformed source graph metadata after source-level
// normalization has been applied.
func ValidateGraphUnits(units []GraphUnit) error {
	seen := map[string]struct{}{}
	for _, unit := range units {
		name := strings.TrimSpace(unit.Unit.Name)
		if name == "" {
			return errors.New("source graph unit has an empty name")
		}
		normalizedName := strings.ToLower(name)
		if _, exists := seen[normalizedName]; exists {
			return fmt.Errorf("source graph unit %q is declared more than once", unit.Unit.Name)
		}
		seen[normalizedName] = struct{}{}
		for _, relationship := range unit.Relations {
			if strings.TrimSpace(relationship.Name) == "" {
				return fmt.Errorf("source graph unit %q has a relationship with an empty name", unit.Unit.Name)
			}
			if !isValidGraphRelationshipType(relationship.RelationshipType) {
				return fmt.Errorf("source graph unit %q has unsupported relationship type %q", unit.Unit.Name, relationship.RelationshipType)
			}
			if !isValidMetadataFidelity(MetadataFidelityOrUnknown(relationship.MetadataFidelity)) {
				return fmt.Errorf("source graph unit %q has unsupported relationship fidelity %q", unit.Unit.Name, relationship.MetadataFidelity)
			}
			if relationship.SourceColumn != nil && strings.TrimSpace(*relationship.SourceColumn) == "" {
				return fmt.Errorf("source graph unit %q has an empty source column", unit.Unit.Name)
			}
			if relationship.TargetColumn != nil && strings.TrimSpace(*relationship.TargetColumn) == "" {
				return fmt.Errorf("source graph unit %q has an empty target column", unit.Unit.Name)
			}
		}
	}
	return nil
}

// FilterInternalObjects removes catalog-declared internal objects from a list.
func FilterInternalObjects(spec TypeSpec, objects []Object) []Object {
	if len(objects) == 0 {
		return objects
	}

	filtered := objects[:0]
	for _, object := range objects {
		if ShouldHideObject(spec, object.Kind, object.Name) {
			continue
		}
		filtered = append(filtered, object)
	}
	return filtered
}

// FilterInternalGraphUnits removes catalog-declared internal graph units.
func FilterInternalGraphUnits(spec TypeSpec, units []GraphUnit, kind ObjectKind) []GraphUnit {
	if len(units) == 0 {
		return units
	}

	hidden := make(map[string]struct{})
	filtered := units[:0]
	for _, unit := range units {
		if ShouldHideObject(spec, kind, unit.Unit.Name) {
			hidden[strings.ToLower(unit.Unit.Name)] = struct{}{}
			continue
		}
		filtered = append(filtered, unit)
	}
	if len(hidden) == 0 {
		return filtered
	}

	for i := range filtered {
		relations := filtered[i].Relations[:0]
		for _, relation := range filtered[i].Relations {
			if _, ok := hidden[strings.ToLower(relation.Name)]; ok {
				continue
			}
			relations = append(relations, relation)
		}
		filtered[i].Relations = relations
	}
	return filtered
}

// ShouldHideObject reports whether a source object matches catalog-declared
// internal object rules.
func ShouldHideObject(spec TypeSpec, kind ObjectKind, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}

	for _, hiddenName := range spec.Traits.Metadata.HiddenObjectNames[kind] {
		if name == strings.ToLower(strings.TrimSpace(hiddenName)) {
			return true
		}
	}
	for _, prefix := range spec.Traits.Metadata.HiddenObjectPrefixes[kind] {
		prefix = strings.ToLower(strings.TrimSpace(prefix))
		if prefix != "" && strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func cloneObjectRules(rules map[ObjectKind][]string) map[ObjectKind][]string {
	if len(rules) == 0 {
		return nil
	}

	cloned := make(map[ObjectKind][]string, len(rules))
	for kind, values := range rules {
		cloned[kind] = slices.Clone(values)
	}
	return cloned
}

func validateColumnShape(column Column) error {
	if (column.ReferencedTable == nil) != (column.ReferencedColumn == nil) {
		return fmt.Errorf("source column %q has incomplete foreign-key metadata", column.Name)
	}
	if column.ReferencedTable != nil && strings.TrimSpace(*column.ReferencedTable) == "" {
		return fmt.Errorf("source column %q has an empty referenced table", column.Name)
	}
	if column.ReferencedColumn != nil && strings.TrimSpace(*column.ReferencedColumn) == "" {
		return fmt.Errorf("source column %q has an empty referenced column", column.Name)
	}
	if column.Length != nil && *column.Length < 0 {
		return fmt.Errorf("source column %q has negative length %d", column.Name, *column.Length)
	}
	if column.Precision != nil && *column.Precision < 0 {
		return fmt.Errorf("source column %q has negative precision %d", column.Name, *column.Precision)
	}
	if column.Scale != nil && *column.Scale < 0 {
		return fmt.Errorf("source column %q has negative scale %d", column.Name, *column.Scale)
	}
	if column.Precision != nil && column.Scale != nil && *column.Scale > *column.Precision {
		return fmt.Errorf("source column %q has scale greater than precision", column.Name)
	}
	return nil
}

func validateFieldConstraintShape(field FieldConstraints) error {
	if field.ForeignKey != nil {
		if strings.TrimSpace(field.ForeignKey.Table) == "" {
			return fmt.Errorf("source field constraint %q has an empty foreign-key table", field.Name)
		}
		if strings.TrimSpace(field.ForeignKey.Column) == "" {
			return fmt.Errorf("source field constraint %q has an empty foreign-key column", field.Name)
		}
	}
	if field.Length != nil && *field.Length < 0 {
		return fmt.Errorf("source field constraint %q has negative length %d", field.Name, *field.Length)
	}
	if field.Precision != nil && *field.Precision < 0 {
		return fmt.Errorf("source field constraint %q has negative precision %d", field.Name, *field.Precision)
	}
	if field.Scale != nil && *field.Scale < 0 {
		return fmt.Errorf("source field constraint %q has negative scale %d", field.Name, *field.Scale)
	}
	if field.Precision != nil && field.Scale != nil && *field.Scale > *field.Precision {
		return fmt.Errorf("source field constraint %q has scale greater than precision", field.Name)
	}
	if field.CheckMin != nil && field.CheckMax != nil && *field.CheckMin > *field.CheckMax {
		return fmt.Errorf("source field constraint %q has minimum greater than maximum", field.Name)
	}
	return nil
}

func isValidGraphRelationshipType(relationshipType GraphRelationshipType) bool {
	switch relationshipType {
	case GraphRelationshipTypeOneToOne, GraphRelationshipTypeOneToMany, GraphRelationshipTypeManyToOne, GraphRelationshipTypeManyToMany, GraphRelationshipTypeUnknown:
		return true
	default:
		return false
	}
}

func metadataWithDefaults(metadata MetadataTraits, columns MetadataFidelity, constraints MetadataFidelity, graph MetadataFidelity) MetadataTraits {
	if metadata.Columns == "" {
		metadata.Columns = columns
	}
	if metadata.Constraints == "" {
		metadata.Constraints = constraints
	}
	if metadata.Graph == "" {
		metadata.Graph = graph
	}
	return metadata
}

func graphFidelityForSpec(spec TypeSpec, fidelity MetadataFidelity) MetadataFidelity {
	if spec.Contract.SupportsSurface(SurfaceGraph) {
		return fidelity
	}
	return MetadataFidelityUnsupported
}
