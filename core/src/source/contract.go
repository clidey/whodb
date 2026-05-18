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
	"fmt"
	"slices"
	"strings"
)

// NormalizeContract fills derived contract actions and returns an independent
// copy of slices owned by the contract.
func NormalizeContract(contract Contract) Contract {
	contract.Surfaces = slices.Clone(contract.Surfaces)
	contract.RootActions = slices.Clone(contract.RootActions)
	contract.BrowsePath = slices.Clone(contract.BrowsePath)
	contract.ObjectTypes = cloneContractObjectTypes(contract.ObjectTypes)
	if !contract.SupportsSurface(SurfaceGraph) {
		return contract
	}
	if contract.GraphScopeKind == nil {
		if !slices.Contains(contract.RootActions, ActionViewGraph) {
			contract.RootActions = append(contract.RootActions, ActionViewGraph)
		}
		return contract
	}

	for i := range contract.ObjectTypes {
		if contract.ObjectTypes[i].Kind != *contract.GraphScopeKind {
			continue
		}
		if !slices.Contains(contract.ObjectTypes[i].Actions, ActionViewGraph) {
			contract.ObjectTypes[i].Actions = append(contract.ObjectTypes[i].Actions, ActionViewGraph)
		}
		if !slices.Contains(contract.ObjectTypes[i].Views, ViewGraph) {
			contract.ObjectTypes[i].Views = append(contract.ObjectTypes[i].Views, ViewGraph)
		}
	}
	return contract
}

// ValidateContract reports source contract inconsistencies that would make the
// frontend and backend disagree about source behavior.
func ValidateContract(spec TypeSpec) error {
	contract := NormalizeContract(spec.Contract)
	if contract.SupportsSurface(SurfaceBrowser) && len(contract.BrowsePath) == 0 {
		return fmt.Errorf("%s browser surface requires a browse path", sourceLabel(spec))
	}
	if len(contract.BrowsePath) > 0 || contract.SupportsSurface(SurfaceBrowser) || contract.SupportsSurface(SurfaceGraph) {
		if _, ok := contract.ObjectTypeForKind(contract.DefaultObjectKind); !ok {
			return fmt.Errorf("%s default object kind %q is not declared in object types", sourceLabel(spec), contract.DefaultObjectKind)
		}
	}

	for _, kind := range contract.BrowsePath {
		if _, ok := contract.ObjectTypeForKind(kind); !ok {
			return fmt.Errorf("%s browse kind %q is not declared in object types", sourceLabel(spec), kind)
		}
	}
	for i, kind := range contract.BrowsePath {
		if i == len(contract.BrowsePath)-1 {
			break
		}
		if !contract.ObjectKindSupportsAction(kind, ActionBrowse) {
			return fmt.Errorf("%s browse parent kind %q does not support browsing", sourceLabel(spec), kind)
		}
	}
	if contract.SupportsSurface(SurfaceBrowser) && !contract.SupportsRootAction(ActionBrowse) {
		return fmt.Errorf("%s browser surface requires root browse support", sourceLabel(spec))
	}
	if !contract.SupportsSurface(SurfaceGraph) {
		return nil
	}
	if contract.GraphScopeKind == nil {
		if !contract.SupportsRootAction(ActionViewGraph) {
			return fmt.Errorf("%s graph surface requires root graph support", sourceLabel(spec))
		}
		return nil
	}
	if _, ok := contract.ObjectTypeForKind(*contract.GraphScopeKind); !ok {
		return fmt.Errorf("%s graph scope kind %q is not declared in object types", sourceLabel(spec), *contract.GraphScopeKind)
	}
	if !contract.ObjectKindSupportsAction(*contract.GraphScopeKind, ActionViewGraph) {
		return fmt.Errorf("%s graph scope kind %q does not support graph views", sourceLabel(spec), *contract.GraphScopeKind)
	}
	return nil
}

// ValidateConnectionContract reports source connection-field inconsistencies
// that would make generated forms, discovery prefills, and stored credentials
// disagree about the source's connection shape.
func ValidateConnectionContract(spec TypeSpec) error {
	seenFields := map[string]struct{}{}
	for _, field := range spec.ConnectionFields {
		fieldKey := strings.TrimSpace(field.Key)
		if fieldKey == "" {
			return fmt.Errorf("%s connection field has an empty key", sourceLabel(spec))
		}

		normalizedKey := strings.ToLower(fieldKey)
		if _, ok := seenFields[normalizedKey]; ok {
			return fmt.Errorf("%s connection field %q is declared more than once", sourceLabel(spec), field.Key)
		}
		seenFields[normalizedKey] = struct{}{}

		if !isValidConnectionFieldKind(field.Kind) {
			return fmt.Errorf("%s connection field %q has unsupported kind %q", sourceLabel(spec), field.Key, field.Kind)
		}
		if !isValidConnectionFieldSection(field.Section) {
			return fmt.Errorf("%s connection field %q has unsupported section %q", sourceLabel(spec), field.Key, field.Section)
		}
		if strings.TrimSpace(field.LabelKey) == "" {
			return fmt.Errorf("%s connection field %q has an empty label key", sourceLabel(spec), field.Key)
		}
		if !isValidCredentialField(field.CredentialField) {
			return fmt.Errorf("%s connection field %q has unsupported credential field %q", sourceLabel(spec), field.Key, field.CredentialField)
		}
		if field.CredentialField == CredentialFieldAdvanced && strings.TrimSpace(field.AdvancedKey) == "" {
			return fmt.Errorf("%s advanced connection field %q has an empty advanced key", sourceLabel(spec), field.Key)
		}
	}

	for _, item := range spec.DiscoveryPrefill.AdvancedDefaults {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			return fmt.Errorf("%s discovery prefill has an empty field key", sourceLabel(spec))
		}
		if isReservedSSLConnectionKey(key) && len(spec.SSLModes) == 0 {
			return fmt.Errorf("%s discovery prefill references SSL field %q without declaring SSL modes", sourceLabel(spec), item.Key)
		}
		if _, ok := seenFields[strings.ToLower(key)]; !ok && !isReservedSSLConnectionKey(key) {
			return fmt.Errorf("%s discovery prefill references undeclared connection field %q", sourceLabel(spec), item.Key)
		}
	}

	seenModes := map[string]struct{}{}
	for _, mode := range spec.SSLModes {
		value := strings.TrimSpace(mode.Value)
		if value == "" {
			return fmt.Errorf("%s SSL mode has an empty value", sourceLabel(spec))
		}
		normalizedValue := strings.ToLower(value)
		if _, ok := seenModes[normalizedValue]; ok {
			return fmt.Errorf("%s SSL mode %q is declared more than once", sourceLabel(spec), mode.Value)
		}
		seenModes[normalizedValue] = struct{}{}

		seenAliases := map[string]struct{}{}
		for _, alias := range mode.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				return fmt.Errorf("%s SSL mode %q has an empty alias", sourceLabel(spec), mode.Value)
			}
			normalizedAlias := strings.ToLower(alias)
			if _, ok := seenAliases[normalizedAlias]; ok {
				return fmt.Errorf("%s SSL mode %q alias %q is declared more than once", sourceLabel(spec), mode.Value, alias)
			}
			seenAliases[normalizedAlias] = struct{}{}
		}
	}

	return nil
}

// ValidateSurfaceSupported returns an error when the source contract does not
// expose the requested surface.
func ValidateSurfaceSupported(spec TypeSpec, surface Surface) error {
	if spec.Contract.SupportsSurface(surface) {
		return nil
	}

	return fmt.Errorf("%s is not supported for %s", SurfaceDescription(surface), sourceLabel(spec))
}

// ValidateObjectActionSupported returns an error when an object kind cannot
// perform the requested action according to the source contract.
func ValidateObjectActionSupported(spec TypeSpec, kind ObjectKind, action Action) error {
	objectType, ok := spec.Contract.ObjectTypeForKind(kind)
	if !ok {
		return fmt.Errorf("%s objects are not supported for %s", kind, sourceLabel(spec))
	}
	if objectType.SupportsAction(action) {
		return nil
	}

	return fmt.Errorf("%s is not supported for %s objects in %s", ActionDescription(action), kind, sourceLabel(spec))
}

// ValidateRootActionSupported returns an error when the source root cannot
// perform the requested action according to the source contract.
func ValidateRootActionSupported(spec TypeSpec, action Action) error {
	if spec.Contract.SupportsRootAction(action) {
		return nil
	}

	return fmt.Errorf("%s is not supported at the source root for %s", ActionDescription(action), sourceLabel(spec))
}

// SurfaceDescription returns a user-facing description for a source surface.
func SurfaceDescription(surface Surface) string {
	switch surface {
	case SurfaceQuery:
		return "querying"
	case SurfaceGraph:
		return "graph views"
	case SurfaceChat:
		return "chat"
	case SurfaceBrowser:
		return "browsing"
	default:
		return strings.ToLower(string(surface))
	}
}

// ActionDescription returns a user-facing description for a source action.
func ActionDescription(action Action) string {
	switch action {
	case ActionBrowse:
		return "browsing"
	case ActionInspect:
		return "inspecting objects"
	case ActionViewRows:
		return "viewing rows"
	case ActionViewContent:
		return "viewing content"
	case ActionViewDefinition:
		return "viewing definitions"
	case ActionCreateChild:
		return "creating child objects"
	case ActionDelete:
		return "deleting objects"
	case ActionInsertData:
		return "inserting data"
	case ActionUpdateData:
		return "updating data"
	case ActionDeleteData:
		return "deleting data"
	case ActionImportData:
		return "importing data"
	case ActionGenerateMockData:
		return "generating mock data"
	case ActionExecute:
		return "executing actions"
	case ActionViewGraph:
		return "viewing graphs"
	default:
		return strings.ToLower(string(action))
	}
}

func sourceLabel(spec TypeSpec) string {
	if strings.TrimSpace(spec.Label) != "" {
		return spec.Label
	}
	if strings.TrimSpace(spec.ID) != "" {
		return spec.ID
	}
	return "source"
}

func isValidConnectionFieldKind(kind ConnectionFieldKind) bool {
	switch kind {
	case ConnectionFieldKindText, ConnectionFieldKindPassword, ConnectionFieldKindBoolean, ConnectionFieldKindFilePath:
		return true
	default:
		return false
	}
}

func isValidConnectionFieldSection(section ConnectionFieldSection) bool {
	switch section {
	case ConnectionFieldSectionPrimary, ConnectionFieldSectionAdvanced:
		return true
	default:
		return false
	}
}

func isValidCredentialField(field CredentialField) bool {
	switch field {
	case "", CredentialFieldHostname, CredentialFieldUsername, CredentialFieldPassword, CredentialFieldDatabase, CredentialFieldAdvanced:
		return true
	default:
		return false
	}
}

func isReservedSSLConnectionKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "ssl mode", "ssl ca content", "ssl client cert content", "ssl client key content", "ssl server name":
		return true
	default:
		return false
	}
}

func cloneContractObjectTypes(objectTypes []ObjectType) []ObjectType {
	cloned := make([]ObjectType, 0, len(objectTypes))
	for _, objectType := range objectTypes {
		cloned = append(cloned, ObjectType{
			Kind:          objectType.Kind,
			DataShape:     objectType.DataShape,
			Actions:       slices.Clone(objectType.Actions),
			Views:         slices.Clone(objectType.Views),
			SingularLabel: objectType.SingularLabel,
			PluralLabel:   objectType.PluralLabel,
		})
	}
	return cloned
}
