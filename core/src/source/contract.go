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
	if err := ValidateExecutionContract(spec); err != nil {
		return err
	}
	if err := ValidateMutationContract(spec); err != nil {
		return err
	}
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

// ValidateExecutionContract reports execution-trait inconsistencies that would
// make query runners, script runners, and streaming runners disagree with the
// source contract.
func ValidateExecutionContract(spec TypeSpec) error {
	contract := NormalizeContract(spec.Contract)
	if err := validateQueryTraits(spec); err != nil {
		return err
	}
	if spec.Traits.Query.SupportsScripts && !contract.SupportsSurface(SurfaceQuery) {
		return fmt.Errorf("%s script execution requires the query surface", sourceLabel(spec))
	}
	if spec.Traits.Query.SupportsStreaming && !contract.SupportsSurface(SurfaceQuery) {
		return fmt.Errorf("%s streaming execution requires the query surface", sourceLabel(spec))
	}
	if spec.Traits.Query.SupportsMultiStatement && !spec.Traits.Query.SupportsScripts {
		return fmt.Errorf("%s multi-statement execution requires script execution", sourceLabel(spec))
	}
	if spec.Traits.Query.SupportsScripts && !contract.SupportsRootAction(ActionExecute) {
		return fmt.Errorf("%s script execution requires root execute support", sourceLabel(spec))
	}
	return nil
}

// ValidateScriptExecutionSupported returns an error when the source contract
// does not allow script execution through the query surface.
func ValidateScriptExecutionSupported(spec TypeSpec) error {
	if err := ValidateSurfaceSupported(spec, SurfaceQuery); err != nil {
		return err
	}
	if err := ValidateRootActionSupported(spec, ActionExecute); err != nil {
		return err
	}
	if !spec.Traits.Query.SupportsScripts {
		return fmt.Errorf("script execution is not supported for %s", sourceLabel(spec))
	}
	return nil
}

// ValidateMutationContract reports source mutation inconsistencies that would
// make backend write enforcement and frontend controls disagree.
func ValidateMutationContract(spec TypeSpec) error {
	contract := NormalizeContract(spec.Contract)
	for _, objectType := range contract.ObjectTypes {
		if err := validateObjectMutationActions(spec, objectType); err != nil {
			return err
		}
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

// ValidateSessionMetadataContract reports source session metadata
// inconsistencies that would make query-builder/editor behavior diverge from
// the source contract.
func ValidateSessionMetadataContract(spec TypeSpec, metadata *TypeSessionMetadata, ok bool) error {
	if !spec.Contract.SupportsSurface(SurfaceQuery) && !ok {
		return nil
	}
	if err := validateQueryTraits(spec); err != nil {
		return err
	}
	if !ok || metadata == nil {
		return fmt.Errorf("%s query surface requires session metadata", sourceLabel(spec))
	}
	if len(metadata.Operators) == 0 && spec.Contract.SupportsSurface(SurfaceQuery) {
		return fmt.Errorf("%s query surface requires operator metadata", sourceLabel(spec))
	}
	if err := validateTypeDefinitions(spec, "session metadata", metadata.TypeDefinitions); err != nil {
		return err
	}
	if err := validateOperators(spec, metadata.Operators); err != nil {
		return err
	}
	return validateAliasMap(spec, metadata)
}

// ValidateObjectCreationMetadataContract reports create-object metadata
// inconsistencies that would make the create-object UI diverge from the source
// contract.
func ValidateObjectCreationMetadataContract(spec TypeSpec, metadata ObjectCreationMetadata, ok bool) error {
	requiresCreationMetadata := spec.Contract.SupportsRootAction(ActionCreateChild) || spec.Contract.SupportsAction(ActionCreateChild)
	if !requiresCreationMetadata && !ok {
		return nil
	}
	if !ok {
		return fmt.Errorf("%s create-object contract requires object creation metadata", sourceLabel(spec))
	}
	if !requiresCreationMetadata && !metadata.Supported {
		return nil
	}
	if !metadata.Supported {
		return fmt.Errorf("%s create-object contract requires supported object creation metadata", sourceLabel(spec))
	}
	if _, exists := spec.Contract.ObjectTypeForKind(metadata.ObjectKind); !exists {
		return fmt.Errorf("%s object creation kind %q is not declared in object types", sourceLabel(spec), metadata.ObjectKind)
	}
	if metadata.RequiresColumns && !metadata.ColumnCapabilities.Types {
		return fmt.Errorf("%s column-based object creation requires type capability metadata", sourceLabel(spec))
	}
	if metadata.ColumnCapabilities.Types && len(metadata.TypeDefinitions) == 0 {
		return fmt.Errorf("%s object creation type capability requires type definitions", sourceLabel(spec))
	}
	if err := validateTypeDefinitions(spec, "object creation metadata", metadata.TypeDefinitions); err != nil {
		return err
	}
	return validateCreationOptions(spec, metadata.TableOptions)
}

// ValidateObjectMetadataContract reports metadata trait inconsistencies that
// would make object columns, constraints, graph data, or internal-object rules
// diverge from the source contract.
func ValidateObjectMetadataContract(spec TypeSpec) error {
	contract := NormalizeContract(spec.Contract)
	metadata := NormalizeMetadataTraits(spec.Traits.Metadata)
	if err := validateMetadataFidelity(spec, "column metadata", metadata.Columns); err != nil {
		return err
	}
	if err := validateMetadataFidelity(spec, "constraint metadata", metadata.Constraints); err != nil {
		return err
	}
	if err := validateMetadataFidelity(spec, "graph metadata", metadata.Graph); err != nil {
		return err
	}
	if err := validateMetadataFidelity(spec, "system object filtering metadata", metadata.SystemObjectFiltering); err != nil {
		return err
	}

	if metadata.Columns != MetadataFidelityUnsupported && !contractSupportsObjectMetadata(contract) {
		return fmt.Errorf("%s declares column metadata without inspectable or row-viewable objects", sourceLabel(spec))
	}
	if metadata.Constraints != MetadataFidelityUnsupported && !contract.SupportsAction(ActionInspect) {
		return fmt.Errorf("%s declares constraint metadata without inspectable objects", sourceLabel(spec))
	}
	if metadata.Graph != MetadataFidelityUnsupported && !contract.SupportsSurface(SurfaceGraph) {
		return fmt.Errorf("%s declares graph metadata without the graph surface", sourceLabel(spec))
	}
	if contract.SupportsSurface(SurfaceGraph) && metadata.Graph == MetadataFidelityUnsupported {
		return fmt.Errorf("%s graph surface requires graph metadata fidelity", sourceLabel(spec))
	}
	return validateHiddenObjectRules(spec, contract, metadata)
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

func validateObjectMutationActions(spec TypeSpec, objectType ObjectType) error {
	for _, action := range []Action{ActionInsertData, ActionUpdateData, ActionDeleteData, ActionImportData, ActionGenerateMockData} {
		if !objectType.SupportsAction(action) {
			continue
		}
		if !objectTypeSupportsDataMutation(objectType, action) {
			return fmt.Errorf("%s declares %s for %s objects without a compatible data view", sourceLabel(spec), ActionDescription(action), objectType.Kind)
		}
	}
	return nil
}

func objectTypeSupportsDataMutation(objectType ObjectType, action Action) bool {
	switch action {
	case ActionImportData, ActionGenerateMockData:
		return objectType.SupportsAction(ActionViewRows)
	case ActionInsertData:
		return objectType.SupportsAction(ActionViewRows) || objectType.DataShape == DataShapeDocument
	case ActionUpdateData, ActionDeleteData:
		return objectType.SupportsAction(ActionViewRows) || objectType.SupportsAction(ActionViewContent)
	default:
		return true
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

func validateQueryTraits(spec TypeSpec) error {
	switch spec.Traits.Query.ExplainMode {
	case QueryExplainModeNone, QueryExplainModeExplain, QueryExplainModeExplainAnalyze, QueryExplainModeExplainPipeline:
	default:
		return fmt.Errorf("%s has unsupported query explain mode %q", sourceLabel(spec), spec.Traits.Query.ExplainMode)
	}
	if spec.Traits.Query.SupportsAnalyze && spec.Traits.Query.ExplainMode != QueryExplainModeExplainAnalyze {
		return fmt.Errorf("%s analyze support requires explain-analyze mode", sourceLabel(spec))
	}
	if spec.Traits.Query.ExplainMode == QueryExplainModeExplainAnalyze && !spec.Traits.Query.SupportsAnalyze {
		return fmt.Errorf("%s explain-analyze mode requires analyze support", sourceLabel(spec))
	}
	return nil
}

func validateMetadataFidelity(spec TypeSpec, name string, fidelity MetadataFidelity) error {
	if isValidMetadataFidelity(fidelity) {
		return nil
	}
	return fmt.Errorf("%s has unsupported %s fidelity %q", sourceLabel(spec), name, fidelity)
}

func validateHiddenObjectRules(spec TypeSpec, contract Contract, metadata MetadataTraits) error {
	hasRules := len(metadata.HiddenObjectNames) > 0 || len(metadata.HiddenObjectPrefixes) > 0
	if hasRules && metadata.SystemObjectFiltering == MetadataFidelityUnsupported {
		return fmt.Errorf("%s declares hidden object rules while system object filtering is unsupported", sourceLabel(spec))
	}

	for kind, names := range metadata.HiddenObjectNames {
		if _, ok := contract.ObjectTypeForKind(kind); !ok {
			return fmt.Errorf("%s hidden object names reference undeclared kind %q", sourceLabel(spec), kind)
		}
		if err := validateHiddenObjectRuleValues(spec, kind, "name", names); err != nil {
			return err
		}
	}
	for kind, prefixes := range metadata.HiddenObjectPrefixes {
		if _, ok := contract.ObjectTypeForKind(kind); !ok {
			return fmt.Errorf("%s hidden object prefixes reference undeclared kind %q", sourceLabel(spec), kind)
		}
		if err := validateHiddenObjectRuleValues(spec, kind, "prefix", prefixes); err != nil {
			return err
		}
	}
	return nil
}

func validateHiddenObjectRuleValues(spec TypeSpec, kind ObjectKind, valueKind string, values []string) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("%s hidden object %s rule for kind %q is empty", sourceLabel(spec), valueKind, kind)
		}
		normalized := strings.ToLower(trimmed)
		if _, exists := seen[normalized]; exists {
			return fmt.Errorf("%s hidden object %s rule %q for kind %q is declared more than once", sourceLabel(spec), valueKind, value, kind)
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

func contractSupportsObjectMetadata(contract Contract) bool {
	for _, objectType := range contract.ObjectTypes {
		if objectType.SupportsAction(ActionInspect) || objectType.SupportsAction(ActionViewRows) {
			return true
		}
		switch objectType.DataShape {
		case DataShapeTabular, DataShapeDocument, DataShapeContent:
			return true
		}
	}
	return false
}

func validateTypeDefinitions(spec TypeSpec, owner string, definitions []TypeDefinition) error {
	seen := map[string]struct{}{}
	for _, definition := range definitions {
		id := strings.TrimSpace(definition.ID)
		if id == "" {
			return fmt.Errorf("%s %s contains an empty type id", sourceLabel(spec), owner)
		}
		normalizedID := strings.ToUpper(id)
		if _, exists := seen[normalizedID]; exists {
			return fmt.Errorf("%s %s type %q is declared more than once", sourceLabel(spec), owner, definition.ID)
		}
		seen[normalizedID] = struct{}{}
		if strings.TrimSpace(definition.Label) == "" {
			return fmt.Errorf("%s %s type %q has an empty label", sourceLabel(spec), owner, definition.ID)
		}
		if !isValidTypeCategory(definition.Category) {
			return fmt.Errorf("%s %s type %q has unsupported category %q", sourceLabel(spec), owner, definition.ID, definition.Category)
		}
	}
	return nil
}

func validateOperators(spec TypeSpec, operators []string) error {
	seen := map[string]struct{}{}
	for _, operator := range operators {
		trimmed := strings.TrimSpace(operator)
		if trimmed == "" {
			return fmt.Errorf("%s session metadata contains an empty operator", sourceLabel(spec))
		}
		normalized := strings.ToLower(trimmed)
		if _, exists := seen[normalized]; exists {
			return fmt.Errorf("%s session metadata operator %q is declared more than once", sourceLabel(spec), operator)
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

func validateAliasMap(spec TypeSpec, metadata *TypeSessionMetadata) error {
	if len(metadata.AliasMap) == 0 {
		return nil
	}
	typeIDs := map[string]struct{}{}
	for _, definition := range metadata.TypeDefinitions {
		typeIDs[strings.ToUpper(strings.TrimSpace(definition.ID))] = struct{}{}
	}
	if len(typeIDs) == 0 {
		return fmt.Errorf("%s session metadata aliases require type definitions", sourceLabel(spec))
	}

	for alias, target := range metadata.AliasMap {
		alias = strings.TrimSpace(alias)
		target = strings.TrimSpace(target)
		if alias == "" {
			return fmt.Errorf("%s session metadata contains an empty type alias", sourceLabel(spec))
		}
		if target == "" {
			return fmt.Errorf("%s session metadata alias %q has an empty target type", sourceLabel(spec), alias)
		}
		if _, exists := typeIDs[strings.ToUpper(target)]; !exists {
			return fmt.Errorf("%s session metadata alias %q references unknown type %q", sourceLabel(spec), alias, target)
		}
	}
	return nil
}

func validateCreationOptions(spec TypeSpec, options []CreationOptionDefinition) error {
	seenOptions := map[string]struct{}{}
	for _, option := range options {
		key := strings.TrimSpace(option.Key)
		if key == "" {
			return fmt.Errorf("%s object creation option has an empty key", sourceLabel(spec))
		}
		normalizedKey := strings.ToLower(key)
		if _, exists := seenOptions[normalizedKey]; exists {
			return fmt.Errorf("%s object creation option %q is declared more than once", sourceLabel(spec), option.Key)
		}
		seenOptions[normalizedKey] = struct{}{}
		if strings.TrimSpace(option.Label) == "" {
			return fmt.Errorf("%s object creation option %q has an empty label", sourceLabel(spec), option.Key)
		}

		seenValues := map[string]struct{}{}
		for _, value := range option.Values {
			value = strings.TrimSpace(value)
			if value == "" {
				return fmt.Errorf("%s object creation option %q contains an empty value", sourceLabel(spec), option.Key)
			}
			normalizedValue := strings.ToLower(value)
			if _, exists := seenValues[normalizedValue]; exists {
				return fmt.Errorf("%s object creation option %q value %q is declared more than once", sourceLabel(spec), option.Key, value)
			}
			seenValues[normalizedValue] = struct{}{}
		}
	}
	return nil
}

func isValidTypeCategory(category TypeCategory) bool {
	switch category {
	case TypeCategoryNumeric, TypeCategoryText, TypeCategoryBinary, TypeCategoryDatetime, TypeCategoryBoolean, TypeCategoryJSON, TypeCategoryOther:
		return true
	default:
		return false
	}
}

func isValidMetadataFidelity(fidelity MetadataFidelity) bool {
	switch fidelity {
	case MetadataFidelityExact, MetadataFidelityDriver, MetadataFidelitySampled, MetadataFidelityInferred, MetadataFidelitySynthetic, MetadataFidelityUnsupported, MetadataFidelityUnknown:
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
