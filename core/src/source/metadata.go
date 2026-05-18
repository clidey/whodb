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
	fidelity = MetadataFidelityOrUnknown(fidelity)
	for i := range units {
		for j := range units[i].Relations {
			if units[i].Relations[j].MetadataFidelity == "" {
				units[i].Relations[j].MetadataFidelity = fidelity
			}
		}
	}
	return units
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
