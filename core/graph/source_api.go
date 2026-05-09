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
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/analytics"
	"github.com/clidey/whodb/core/src/audit"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/importer"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/mockdata"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func performSourceLogin(ctx context.Context, credentials *source.Credentials, profileSource string) (*model.StatusResponse, error) {
	start := time.Now()
	hasProfileID := credentials.ID != nil && strings.TrimSpace(*credentials.ID) != ""
	resource := audit.SourceResource(credentials.SourceType, credentials.ID)
	recordAudit := func(err error) {
		details := map[string]any{
			"source_type":        credentials.SourceType,
			"profile_id_present": hasProfileID,
			"profile_source":     profileSource,
		}
		severity := audit.SeverityInfo
		if err != nil {
			severity = audit.SeverityWarn
			details["error"] = err.Error()
		}

		audit.RecordWithContext(ctx, audit.AuditEvent{
			Timestamp: start,
			Action:    "login.source",
			Severity:  severity,
			Resource:  resource,
			Details:   details,
			Duration:  time.Since(start),
		})
	}

	if env.DisableCredentialForm {
		log.WithField("sourceType", credentials.SourceType).Error("Login with credentials is disabled; use preconfigured connections")
		err := errors.New("login with credentials is disabled; use preconfigured connections")
		recordAudit(err)
		return nil, err
	}

	spec, ok := sourcecatalog.Find(credentials.SourceType)
	if !ok {
		err := errors.New("unauthorized")
		recordAudit(err)
		return nil, err
	}
	session, err := source.Open(ctx, spec, credentials)
	if err != nil {
		recordAudit(err)
		return nil, err
	}
	availability, ok := source.AsAvailabilityChecker(source.AuditScopeFromCredentials(spec, credentials), session)
	if !ok {
		err := errors.New("unauthorized")
		recordAudit(err)
		return nil, err
	}

	identity := strings.TrimSpace(analytics.MetadataFromContext(ctx).DistinctID)
	hasIdentity := identity != "" && identity != "disabled"

	if hasIdentity {
		properties := map[string]any{
			"source_type":        credentials.SourceType,
			"profile_id_present": hasProfileID,
			"connector":          spec.Connector,
			"profile_source":     profileSource,
			"is_saved_profile":   profileSource != "",
		}
		analytics.CaptureWithDistinctID(ctx, identity, "login.attempt", properties)
	}

	if !availability.IsAvailable(ctx) {
		if hasIdentity {
			analytics.CaptureWithDistinctID(ctx, identity, "login.denied", map[string]any{
				"source_type":        credentials.SourceType,
				"profile_id_present": hasProfileID,
				"connector":          spec.Connector,
				"profile_source":     profileSource,
			})
		}
		err := errors.New("unauthorized")
		recordAudit(err)
		return nil, err
	}

	resp, err := auth.LoginSource(ctx, credentials)
	if err != nil {
		if hasIdentity {
			analytics.CaptureError(ctx, "login.execute", err, map[string]any{
				"source_type":        credentials.SourceType,
				"profile_id_present": hasProfileID,
				"connector":          spec.Connector,
				"profile_source":     profileSource,
			})
		}
		recordAudit(err)
		return nil, err
	}

	if hasIdentity {
		traits := map[string]any{
			"profile_id_present": hasProfileID,
			"source_type":        credentials.SourceType,
			"connector":          spec.Connector,
		}
		if profileSource != "" {
			traits["profile_source"] = profileSource
			traits["saved_profile"] = true
		}
		if hashedHost := analytics.HashIdentifier(credentials.CloneValues()["Hostname"]); hashedHost != "" {
			traits["hostname_hash"] = hashedHost
		}
		if hashedDatabase := analytics.HashIdentifier(credentials.CloneValues()["Database"]); hashedDatabase != "" {
			traits["database_hash"] = hashedDatabase
		}

		analytics.IdentifyWithDistinctID(ctx, identity, traits)
		analytics.CaptureWithDistinctID(ctx, identity, "login.success", map[string]any{
			"source_type":        credentials.SourceType,
			"profile_id_present": hasProfileID,
			"connector":          spec.Connector,
			"profile_source":     profileSource,
		})
	}

	recordAudit(nil)
	return resp, nil
}

func sourceObjectModels(_ source.TypeSpec, objects []source.Object) []*model.SourceObject {
	mapped := make([]*model.SourceObject, 0, len(objects))
	for _, object := range objects {
		mapped = append(mapped, sourceObjectToModel(object))
	}
	return mapped
}

func sourceContainerScope(spec source.TypeSpec, ref *source.ObjectRef) string {
	if ref == nil {
		return ""
	}

	defaultIndex := slices.Index(spec.Contract.BrowsePath, spec.Contract.DefaultObjectKind)
	if defaultIndex < 0 {
		return ""
	}

	if ref.Kind == spec.Contract.DefaultObjectKind {
		if defaultIndex == 0 {
			return ""
		}
		if defaultIndex-1 < len(ref.Path) {
			return ref.Path[defaultIndex-1]
		}
		return ""
	}

	if len(ref.Path) == defaultIndex {
		return ref.Path[len(ref.Path)-1]
	}

	return ""
}

func sourceScopeForChat(spec source.TypeSpec, ref *source.ObjectRef) string {
	if ref == nil {
		return ""
	}
	if spec.Contract.GraphScopeKind != nil {
		if scope := scopeValueForKind(spec, *ref, *spec.Contract.GraphScopeKind); scope != "" {
			return scope
		}
	}
	return sourceContainerScope(spec, ref)
}

func mergeRowsColumns(rowsResult *source.RowsResult, columns []source.Column) *model.RowsResult {
	if rowsResult == nil {
		return nil
	}

	if len(columns) == 0 {
		return rowsResultToModel(rowsResult)
	}

	columnInfo := make(map[string]source.Column, len(columns))
	for _, column := range columns {
		columnInfo[column.Name] = column
	}

	mappedColumns := make([]*model.Column, 0, len(rowsResult.Columns))
	for _, column := range rowsResult.Columns {
		info := columnInfo[column.Name]
		mappedColumns = append(mappedColumns, &model.Column{
			Type:             column.Type,
			Name:             column.Name,
			IsPrimary:        info.IsPrimary,
			IsForeignKey:     info.IsForeignKey,
			ReferencedTable:  info.ReferencedTable,
			ReferencedColumn: info.ReferencedColumn,
			Length:           column.Length,
			Precision:        column.Precision,
			Scale:            column.Scale,
		})
	}

	return &model.RowsResult{
		Columns:       mappedColumns,
		Rows:          rowsResult.Rows,
		DisableUpdate: rowsResult.DisableUpdate,
		TotalCount:    int(rowsResult.TotalCount),
	}
}

func importPreviewForRef(ctx context.Context, file graphql.Upload, options model.ImportFileOptions, ref *model.SourceObjectRefInput, useHeaderMapping *bool) (*model.ImportPreview, error) {
	data, err := readUploadBytes(file, maxImportFileSizeBytes)
	if err != nil {
		return nil, err
	}

	result, err := parseImportFile(data, &options, importPreviewRowLimit, false)
	if err != nil {
		return nil, err
	}

	preview := &model.ImportPreview{
		Sheet:                      stringPtr(result.sheet),
		Columns:                    result.columns,
		Rows:                       result.rows,
		Truncated:                  result.truncated,
		RequiresAllowAutoGenerated: false,
		AutoGeneratedColumns:       []string{},
	}

	if ref == nil || useHeaderMapping == nil {
		return preview, nil
	}

	spec, session, err := getSourceSessionForContext(ctx)
	if err != nil {
		return nil, err
	}
	scope := sourceAuditScopeFromContext(ctx, spec)
	resolvedRef := sourceRefFromInput(ref)
	if resolvedRef == nil {
		return preview, nil
	}

	reader, ok := source.AsTabularReader(scope, session)
	if !ok {
		return nil, errors.New("source columns are not supported")
	}

	columns, err := reader.Columns(ctx, *resolvedRef)
	if err != nil {
		return nil, err
	}

	mappings, autoGeneratedColumns, err := buildImportMappingInputs(preview.Columns, columns, *useHeaderMapping, *useHeaderMapping)
	if err != nil {
		key := validationKeyFromError(err)
		if key != "" {
			preview.ValidationError = &key
		}
		if len(autoGeneratedColumns) > 0 {
			preview.RequiresAllowAutoGenerated = true
			preview.AutoGeneratedColumns = autoGeneratedColumns
			preview.Mapping = importColumnMappingPreviewModel(mappings)
		}
		return preview, nil
	}

	if len(autoGeneratedColumns) > 0 {
		preview.RequiresAllowAutoGenerated = true
		preview.AutoGeneratedColumns = autoGeneratedColumns
	}
	preview.Mapping = importColumnMappingPreviewModel(mappings)
	return preview, nil
}

func importColumnMappingPreviewModel(mappings []*model.ImportColumnMapping) []*model.ImportColumnMappingPreview {
	if len(mappings) == 0 {
		return nil
	}

	preview := make([]*model.ImportColumnMappingPreview, 0, len(mappings))
	for _, mapping := range mappings {
		if mapping == nil || mapping.TargetColumn == nil {
			continue
		}
		preview = append(preview, &model.ImportColumnMappingPreview{
			SourceColumn: mapping.SourceColumn,
			TargetColumn: *mapping.TargetColumn,
		})
	}
	return preview
}

func importSourceObjectFile(ctx context.Context, input model.ImportFileInput) (*model.ImportResult, error) {
	spec, session, err := getSourceSessionForContext(ctx)
	if err != nil {
		return nil, err
	}
	scope := sourceAuditScopeFromContext(ctx, spec)
	resolvedRef := sourceRefFromInput(input.Ref)
	if resolvedRef == nil {
		return importResult(false, importValidationInvalidOptions), nil
	}

	dataImporter, ok := source.AsDataImporter(scope, session)
	if !ok {
		return nil, errors.New("source import is not supported")
	}

	data, err := readUploadBytes(input.File, maxImportFileSizeBytes)
	if err != nil {
		return importResult(false, validationKeyFromError(err)), nil
	}

	parsed, err := parseImportFile(data, input.Options, maxImportRows, true)
	if err != nil {
		return importResult(false, validationKeyFromError(err)), nil
	}

	allowAutoGenerated := false
	if input.AllowAutoGenerated != nil {
		allowAutoGenerated = *input.AllowAutoGenerated
	}

	_, err = dataImporter.ImportData(ctx, *resolvedRef, source.ImportRequest{
		Mode:               source.ImportMode(input.Mode),
		Parsed:             source.ParsedImportFile{Columns: slices.Clone(parsed.columns), Rows: cloneStringRows(parsed.rows), Truncated: parsed.truncated, Sheet: parsed.sheet},
		Mapping:            sourceImportColumnMappings(input.Mapping),
		AllowAutoGenerated: allowAutoGenerated,
		BatchSize:          importBatchSize,
	})
	if err != nil {
		if key := importer.ErrorKeyFromError(err); key != "" {
			return importResult(false, key), nil
		}
		log.WithError(err).Error("Import failed")
		return importResult(false, importErrorImportFailed), nil
	}

	return importResult(true, ""), nil
}

func generateMockDataForRef(ctx context.Context, input model.MockDataGenerationInput) (*model.MockDataGenerationStatus, error) {
	if input.Ref == nil {
		return nil, errors.New("source object reference is required")
	}

	maxRowLimit := mockdata.GetMockDataGenerationMaxRowCount()
	if input.RowCount > maxRowLimit {
		return nil, fmt.Errorf("row count exceeds maximum limit of %d", maxRowLimit)
	}

	spec, session, err := getSourceSessionForContext(ctx)
	if err != nil {
		return nil, err
	}
	scope := sourceAuditScopeFromContext(ctx, spec)
	resolvedRef := sourceRefFromInput(input.Ref)
	_, objectName := namespaceAndObjectNameForRef(spec, *resolvedRef)
	if !mockdata.IsMockDataGenerationAllowed(objectName) {
		return nil, errors.New("mock data generation is not allowed for this table")
	}

	manager, ok := source.AsMockDataManager(scope, session)
	if !ok {
		return nil, errors.New("mock data generation is not supported")
	}

	fkRatio := 0
	if input.FkDensityRatio != nil {
		fkRatio = *input.FkDensityRatio
	}
	result, err := manager.GenerateMockData(ctx, *resolvedRef, input.RowCount, fkRatio, input.OverwriteExisting)
	if err != nil {
		return nil, fmt.Errorf("mock data generation failed: %w", err)
	}

	details := make([]*model.MockDataTableDetail, 0, len(result.Details))
	for _, detail := range result.Details {
		details = append(details, &model.MockDataTableDetail{
			Table:            detail.Table,
			RowsGenerated:    detail.RowsGenerated,
			UsedExistingData: detail.UsedExistingData,
		})
	}

	return &model.MockDataGenerationStatus{
		AmountGenerated: result.TotalGenerated,
		Details:         details,
	}, nil
}

func analyzeMockDataDependenciesForRef(ctx context.Context, ref model.SourceObjectRefInput, rowCount int, fkDensityRatio *int) (*model.MockDataDependencyAnalysis, error) {
	maxRowLimit := mockdata.GetMockDataGenerationMaxRowCount()
	if rowCount > maxRowLimit {
		errMsg := fmt.Sprintf("row count exceeds maximum limit of %d", maxRowLimit)
		return &model.MockDataDependencyAnalysis{Error: &errMsg}, nil
	}

	spec, session, err := getSourceSessionForContext(ctx)
	if err != nil {
		return nil, err
	}
	scope := sourceAuditScopeFromContext(ctx, spec)
	resolvedRef := sourceRefFromInput(&ref)
	_, objectName := namespaceAndObjectNameForRef(spec, *resolvedRef)
	if !mockdata.IsMockDataGenerationAllowed(objectName) {
		errMsg := "mock data generation is not allowed for this table"
		return &model.MockDataDependencyAnalysis{Error: &errMsg}, nil
	}

	manager, ok := source.AsMockDataManager(scope, session)
	if !ok {
		errMsg := "mock data dependency analysis is not supported"
		return &model.MockDataDependencyAnalysis{Error: &errMsg}, nil
	}

	fkRatio := 0
	if fkDensityRatio != nil {
		fkRatio = *fkDensityRatio
	}
	analysis, err := manager.AnalyzeMockDataDependencies(ctx, *resolvedRef, rowCount, fkRatio)
	if err != nil {
		errMsg := err.Error()
		return &model.MockDataDependencyAnalysis{Error: &errMsg}, nil
	}

	tables := make([]*model.MockDataTableInfo, 0, len(analysis.Tables))
	for _, table := range analysis.Tables {
		tables = append(tables, &model.MockDataTableInfo{
			Table:            table.Table,
			RowsToGenerate:   table.RowCount,
			IsBlocked:        table.IsBlocked,
			UsesExistingData: table.UsesExistingData,
		})
	}

	var errorPtr *string
	if analysis.Error != "" {
		errorPtr = &analysis.Error
	}

	return &model.MockDataDependencyAnalysis{
		GenerationOrder: analysis.GenerationOrder,
		Tables:          tables,
		TotalRows:       analysis.TotalRows,
		Warnings:        analysis.Warnings,
		Error:           errorPtr,
	}, nil
}

func sourceQuerySuggestionsForRef(ctx context.Context, ref *model.SourceObjectRefInput) ([]*model.SourceQuerySuggestion, error) {
	spec, session, err := getSourceSessionForContext(ctx)
	if err != nil {
		return nil, err
	}
	suggester, ok := source.AsQuerySuggester(sourceAuditScopeFromContext(ctx, spec), session)
	if !ok {
		return nil, errors.New("source query suggestions are not supported")
	}

	var resolvedRef *source.ObjectRef
	if ref != nil {
		resolvedRef = sourceRefFromInput(ref)
	}
	suggestions, err := suggester.QuerySuggestions(ctx, resolvedRef)
	if err != nil {
		return nil, err
	}

	response := make([]*model.SourceQuerySuggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		response = append(response, &model.SourceQuerySuggestion{
			Description: suggestion.Description,
			Category:    suggestion.Category,
		})
	}
	return response, nil
}
