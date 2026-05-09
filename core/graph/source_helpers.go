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

package graph

import (
	"context"
	"fmt"
	"slices"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func getSourceSpecForContext(ctx context.Context) (source.TypeSpec, *source.Credentials, error) {
	credentials := auth.GetSourceCredentials(ctx)
	if credentials == nil {
		return source.TypeSpec{}, nil, fmt.Errorf("unauthorized")
	}

	spec, ok := sourcecatalog.Find(credentials.SourceType)
	if !ok {
		return source.TypeSpec{}, nil, fmt.Errorf("unsupported source type: %s", credentials.SourceType)
	}

	return spec, credentials, nil
}

func getSourceSessionForContext(ctx context.Context) (source.TypeSpec, source.SourceSession, error) {
	spec, credentials, err := getSourceSpecForContext(ctx)
	if err != nil {
		return source.TypeSpec{}, nil, err
	}

	session, err := source.Open(ctx, spec, credentials)
	if err != nil {
		return source.TypeSpec{}, nil, err
	}

	return spec, session, nil
}

func sourceAuditScopeFromContext(ctx context.Context, spec source.TypeSpec) source.AuditScope {
	return source.AuditScopeFromCredentials(spec, auth.GetSourceCredentials(ctx))
}

func sourceCredentialsFromInput(input model.SourceLoginInput) *source.Credentials {
	return &source.Credentials{
		ID:          input.ID,
		SourceType:  input.SourceType,
		Values:      recordInputsToMap(input.Values),
		AccessToken: input.AccessToken,
	}
}

func sourceCredentialsFromProfileInput(profile model.SourceProfileLoginInput) *source.Credentials {
	return &source.Credentials{
		ID:     &profile.ID,
		Values: recordInputsToMap(profile.Values),
	}
}

func recordInputsToMap(values []*model.RecordInput) map[string]string {
	mapped := make(map[string]string, len(values))
	for _, value := range values {
		mapped[value.Key] = value.Value
	}
	return mapped
}

func recordInputsToSourceRecords(values []*model.RecordInput) []source.Record {
	records := make([]source.Record, 0, len(values))
	for _, value := range values {
		extra := map[string]string{}
		for _, item := range value.Extra {
			extra[item.Key] = item.Value
		}
		records = append(records, source.Record{
			Key:   value.Key,
			Value: value.Value,
			Extra: extra,
		})
	}
	return records
}

func recordsToModel(values map[string]string) []*model.Record {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	records := make([]*model.Record, 0, len(keys))
	for _, key := range keys {
		records = append(records, &model.Record{
			Key:   key,
			Value: values[key],
		})
	}
	return records
}

func recordSliceToModel(values []source.Record) []*model.Record {
	records := make([]*model.Record, 0, len(values))
	for _, value := range values {
		records = append(records, &model.Record{
			Key:   value.Key,
			Value: value.Value,
		})
	}
	return records
}

func sourceRefFromInput(ref *model.SourceObjectRefInput) *source.ObjectRef {
	if ref == nil {
		return nil
	}

	resolved := source.ObjectRef{
		Kind: source.ObjectKind(ref.Kind),
		Path: slices.Clone(ref.Path),
	}
	if ref.Locator != nil {
		resolved.Locator = *ref.Locator
	}
	normalized := source.NormalizeObjectRef(resolved)
	return &normalized
}

func sourceRefToModel(ref source.ObjectRef) *model.SourceObjectRef {
	normalized := source.NormalizeObjectRef(ref)
	return &model.SourceObjectRef{
		Kind:    model.SourceObjectKind(normalized.Kind),
		Locator: normalized.Locator,
		Path:    slices.Clone(normalized.Path),
	}
}

func sourceObjectToModel(object source.Object) *model.SourceObject {
	actions := make([]model.SourceAction, 0, len(object.Actions))
	for _, action := range object.Actions {
		actions = append(actions, model.SourceAction(action))
	}

	return &model.SourceObject{
		Ref:         sourceRefToModel(object.Ref),
		Kind:        model.SourceObjectKind(object.Kind),
		Name:        object.Name,
		Path:        slices.Clone(object.Path),
		HasChildren: object.HasChildren,
		Actions:     actions,
		Metadata:    recordSliceToModel(object.Metadata),
	}
}

func sourceFileBaseName(ref *source.ObjectRef, fallback string) string {
	if ref == nil || len(ref.Path) == 0 {
		return fallback
	}
	if len(ref.Path) == 1 {
		return ref.Path[0]
	}
	return ref.Path[len(ref.Path)-2] + "_" + ref.Path[len(ref.Path)-1]
}

func sourceTypeToModel(spec source.TypeSpec) *model.SourceType {
	fields := make([]*model.SourceConnectionField, 0, len(spec.ConnectionFields))
	for _, field := range spec.ConnectionFields {
		var placeholder *string
		if field.PlaceholderKey != "" {
			placeholder = &field.PlaceholderKey
		}
		var defaultValue *string
		if field.DefaultValue != "" {
			defaultValue = &field.DefaultValue
		}

		fields = append(fields, &model.SourceConnectionField{
			Key:             field.Key,
			Kind:            model.SourceConnectionFieldKind(field.Kind),
			Section:         model.SourceConnectionFieldSection(field.Section),
			Required:        field.Required,
			LabelKey:        field.LabelKey,
			PlaceholderKey:  placeholder,
			DefaultValue:    defaultValue,
			SupportsOptions: field.SupportsOptions,
		})
	}

	sslModes := make([]*model.SourceSSLMode, 0, len(spec.SSLModes))
	for _, sslMode := range spec.SSLModes {
		sslModes = append(sslModes, &model.SourceSSLMode{
			Value:   sslMode.Value,
			Aliases: slices.Clone(sslMode.Aliases),
		})
	}

	return &model.SourceType{
		ID:               spec.ID,
		Label:            spec.Label,
		Connector:        spec.Connector,
		Category:         model.SourceCategory(spec.Category),
		Traits:           sourceTraitsToModel(spec.Traits),
		ConnectionFields: fields,
		Contract:         sourceContractToModel(spec.Contract),
		DiscoveryPrefill: sourceDiscoveryPrefillToModel(spec.DiscoveryPrefill),
		IsAWSManaged:     spec.IsAWSManaged,
		SSLModes:         sslModes,
	}
}

func sourceDiscoveryPrefillToModel(prefill source.DiscoveryPrefill) *model.SourceDiscoveryPrefill {
	advancedDefaults := make([]*model.SourceDiscoveryAdvancedDefault, 0, len(prefill.AdvancedDefaults))
	for _, item := range prefill.AdvancedDefaults {
		conditions := make([]*model.SourceDiscoveryMetadataCondition, 0, len(item.Conditions))
		for _, condition := range item.Conditions {
			conditions = append(conditions, &model.SourceDiscoveryMetadataCondition{
				Key:   condition.Key,
				Value: condition.Value,
			})
		}

		advancedDefaults = append(advancedDefaults, &model.SourceDiscoveryAdvancedDefault{
			Key:           item.Key,
			Value:         item.Value,
			MetadataKey:   item.MetadataKey,
			DefaultValue:  item.DefaultValue,
			ProviderTypes: slices.Clone(item.ProviderTypes),
			Conditions:    conditions,
		})
	}

	return &model.SourceDiscoveryPrefill{
		AdvancedDefaults: advancedDefaults,
	}
}

func sourceTraitsToModel(traits source.TypeTraits) *model.SourceTraits {
	return &model.SourceTraits{
		Connection: &model.SourceConnectionTraits{
			Transport:               model.SourceConnectionTransport(traits.Connection.Transport),
			HostInputMode:           model.SourceHostInputMode(traits.Connection.HostInputMode),
			HostInputURLParser:      model.SourceHostInputURLParser(traits.Connection.HostInputURLParser),
			SupportsCustomCAContent: traits.Connection.SupportsCustomCAContent,
		},
		Presentation: &model.SourcePresentationTraits{
			ProfileLabelStrategy: model.SourceProfileLabelStrategy(traits.Presentation.ProfileLabelStrategy),
			SchemaFidelity:       model.SourceSchemaFidelity(traits.Presentation.SchemaFidelity),
		},
		Query: &model.SourceQueryTraits{
			SupportsAnalyze: traits.Query.SupportsAnalyze,
			ExplainMode:     model.SourceQueryExplainMode(traits.Query.ExplainMode),
		},
		MockData: &model.SourceMockDataTraits{
			SupportsRelationalDependencies: traits.MockData.SupportsRelationalDependencies,
		},
	}
}

func sourceContractToModel(contract source.Contract) *model.SourceContract {
	surfaces := make([]model.SourceSurface, 0, len(contract.Surfaces))
	for _, surface := range contract.Surfaces {
		surfaces = append(surfaces, model.SourceSurface(surface))
	}

	rootActions := make([]model.SourceAction, 0, len(contract.RootActions))
	for _, action := range contract.RootActions {
		rootActions = append(rootActions, model.SourceAction(action))
	}

	browsePath := make([]model.SourceObjectKind, 0, len(contract.BrowsePath))
	for _, kind := range contract.BrowsePath {
		browsePath = append(browsePath, model.SourceObjectKind(kind))
	}

	objectTypes := make([]*model.SourceObjectType, 0, len(contract.ObjectTypes))
	for _, objectType := range contract.ObjectTypes {
		actions := make([]model.SourceAction, 0, len(objectType.Actions))
		for _, action := range objectType.Actions {
			actions = append(actions, model.SourceAction(action))
		}
		views := make([]model.SourceView, 0, len(objectType.Views))
		for _, view := range objectType.Views {
			views = append(views, model.SourceView(view))
		}
		objectTypes = append(objectTypes, &model.SourceObjectType{
			Kind:          model.SourceObjectKind(objectType.Kind),
			DataShape:     model.DataShape(objectType.DataShape),
			Actions:       actions,
			Views:         views,
			SingularLabel: objectType.SingularLabel,
			PluralLabel:   objectType.PluralLabel,
		})
	}

	var graphScopeKind *model.SourceObjectKind
	if contract.GraphScopeKind != nil {
		kind := model.SourceObjectKind(*contract.GraphScopeKind)
		graphScopeKind = &kind
	}

	return &model.SourceContract{
		Model:             model.SourceModel(contract.Model),
		Surfaces:          surfaces,
		RootActions:       rootActions,
		BrowsePath:        browsePath,
		DefaultObjectKind: model.SourceObjectKind(contract.DefaultObjectKind),
		GraphScopeKind:    graphScopeKind,
		ObjectTypes:       objectTypes,
	}
}

func sourceSessionMetadataToModel(metadata *source.SessionMetadata) *model.SourceSessionMetadata {
	if metadata == nil {
		return nil
	}

	typeDefinitions := make([]*model.TypeDefinition, 0, len(metadata.TypeDefinitions))
	for _, definition := range metadata.TypeDefinitions {
		typeDefinitions = append(typeDefinitions, &model.TypeDefinition{
			ID:               definition.ID,
			Label:            definition.Label,
			HasLength:        definition.HasLength,
			HasPrecision:     definition.HasPrecision,
			DefaultLength:    definition.DefaultLength,
			DefaultPrecision: definition.DefaultPrecision,
			Category:         model.TypeCategory(definition.Category),
			InsertFunc:       stringPtr(definition.InsertFunc),
			TableModel:       stringPtr(definition.TableModel),
		})
	}

	return &model.SourceSessionMetadata{
		SourceType:      metadata.SourceType,
		QueryLanguages:  slices.Clone(metadata.QueryLanguages),
		TypeDefinitions: typeDefinitions,
		Operators:       slices.Clone(metadata.Operators),
		AliasMap:        recordsToModel(metadata.AliasMap),
	}
}

func sourceContentToModel(content *source.ContentResult) *model.SourceContent {
	if content == nil {
		return nil
	}

	return &model.SourceContent{
		Text:       content.Text,
		MIMEType:   content.MIMEType,
		IsBinary:   content.IsBinary,
		SizeBytes:  fmt.Sprintf("%d", content.SizeBytes),
		Truncated:  content.Truncated,
		FileName:   content.FileName,
		ModifiedAt: content.ModifiedAt,
	}
}

func rowsResultToModel(rowsResult *source.RowsResult) *model.RowsResult {
	if rowsResult == nil {
		return nil
	}
	return &model.RowsResult{
		Columns:       MapColumnsToModel(rowsResult.Columns),
		Rows:          rowsResult.Rows,
		DisableUpdate: rowsResult.DisableUpdate,
		TotalCount:    int(rowsResult.TotalCount),
	}
}

func sourceObjectColumnsToModel(results []source.ObjectColumns) []*model.SourceObjectColumns {
	columns := make([]*model.SourceObjectColumns, 0, len(results))
	for _, result := range results {
		columns = append(columns, &model.SourceObjectColumns{
			Ref:     sourceRefToModel(result.Ref),
			Columns: MapColumnsToModel(result.Columns),
		})
	}
	return columns
}

func graphUnitsToModel(graphUnits []source.GraphUnit, parent *source.ObjectRef, defaultKind source.ObjectKind) []*model.GraphUnit {
	mapped := make([]*model.GraphUnit, 0, len(graphUnits))
	for _, graphUnit := range graphUnits {
		relations := make([]*model.GraphUnitRelationship, 0, len(graphUnit.Relations))
		for _, relation := range graphUnit.Relations {
			relations = append(relations, &model.GraphUnitRelationship{
				Name:         relation.Name,
				Relationship: model.GraphUnitRelationshipType(relation.RelationshipType),
				SourceColumn: relation.SourceColumn,
				TargetColumn: relation.TargetColumn,
			})
		}

		object := source.Object{
			Ref:      source.NewObjectRef(defaultKind, appendGraphPath(parent, graphUnit.Unit.Name)),
			Kind:     defaultKind,
			Name:     graphUnit.Unit.Name,
			Path:     appendGraphPath(parent, graphUnit.Unit.Name),
			Metadata: graphUnit.Unit.Attributes,
		}

		mapped = append(mapped, &model.GraphUnit{
			Unit:      sourceObjectToModel(object),
			Relations: relations,
		})
	}
	return mapped
}

func sourceProfilesToModel(profiles []source.Profile) []*model.SourceProfile {
	mapped := make([]*model.SourceProfile, 0, len(profiles))
	for _, profile := range profiles {
		mapped = append(mapped, &model.SourceProfile{
			ID:                   profile.ID,
			DisplayName:          profile.DisplayName,
			SourceType:           profile.SourceType,
			Values:               recordsToModel(profile.Values),
			IsEnvironmentDefined: profile.IsEnvironmentDefined,
			Source:               profile.Source,
			SSLConfigured:        profile.SSLConfigured,
		})
	}
	return mapped
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func appendGraphPath(parent *source.ObjectRef, name string) []string {
	if parent == nil {
		return []string{name}
	}
	path := slices.Clone(parent.Path)
	path = append(path, name)
	return path
}

func namespaceAndObjectNameForRef(spec source.TypeSpec, ref source.ObjectRef) (string, string) {
	defaultIndex := slices.Index(spec.Contract.BrowsePath, spec.Contract.DefaultObjectKind)
	namespace := ""
	if defaultIndex > 0 && defaultIndex-1 < len(ref.Path) {
		namespace = ref.Path[defaultIndex-1]
	}

	name := ""
	if len(ref.Path) > 0 {
		name = ref.Path[len(ref.Path)-1]
	}
	return namespace, name
}

func scopeValueForKind(spec source.TypeSpec, ref source.ObjectRef, kind source.ObjectKind) string {
	index := slices.Index(spec.Contract.BrowsePath, kind)
	if index < 0 || index >= len(ref.Path) {
		return ""
	}
	return ref.Path[index]
}

func mergeCredentialValues(base map[string]string, overrides map[string]string) map[string]string {
	merged := map[string]string{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = value
	}
	return merged
}

func cloneStringRows(rows [][]string) [][]string {
	cloned := make([][]string, 0, len(rows))
	for _, row := range rows {
		cloned = append(cloned, slices.Clone(row))
	}
	return cloned
}

func sourceImportColumnMappings(mappings []*model.ImportColumnMapping) []source.ImportColumnMapping {
	if len(mappings) == 0 {
		return nil
	}

	converted := make([]source.ImportColumnMapping, 0, len(mappings))
	for _, mapping := range mappings {
		if mapping == nil {
			continue
		}
		converted = append(converted, source.ImportColumnMapping{
			SourceColumn: mapping.SourceColumn,
			TargetColumn: mapping.TargetColumn,
			Skip:         mapping.Skip,
		})
	}
	return converted
}
