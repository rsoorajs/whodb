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

	"github.com/99designs/gqlgen/graphql"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/analytics"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/importer"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/mockdata"
	"github.com/clidey/whodb/core/src/querysuggestions"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/source/adapters"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func performSourceLogin(ctx context.Context, credentials *source.Credentials, profileSource string) (*model.StatusResponse, error) {
	if env.DisableCredentialForm {
		log.WithField("sourceType", credentials.SourceType).Error("Login with credentials is disabled; use preconfigured connections")
		return nil, errors.New("login with credentials is disabled; use preconfigured connections")
	}

	spec, plugin, config, err := resolveSourcePluginConfig(ctx, credentials)
	if err != nil {
		return nil, err
	}

	identity := strings.TrimSpace(analytics.MetadataFromContext(ctx).DistinctID)
	hasIdentity := identity != "" && identity != "disabled"
	hasProfileID := credentials.ID != nil && strings.TrimSpace(*credentials.ID) != ""

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

	if !plugin.IsAvailable(ctx, config) {
		if hasIdentity {
			analytics.CaptureWithDistinctID(ctx, identity, "login.denied", map[string]any{
				"source_type":        credentials.SourceType,
				"profile_id_present": hasProfileID,
				"connector":          spec.Connector,
				"profile_source":     profileSource,
			})
		}
		return nil, errors.New("unauthorized")
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

	return resp, nil
}

func resolveSourcePluginConfig(ctx context.Context, credentials *source.Credentials) (source.TypeSpec, *engine.Plugin, *engine.PluginConfig, error) {
	spec, ok := sourcecatalog.Find(credentials.SourceType)
	if !ok {
		return source.TypeSpec{}, nil, nil, errors.New("unauthorized")
	}

	plugin := src.MainEngine.Choose(engine.DatabaseType(spec.Connector))
	if plugin == nil {
		return source.TypeSpec{}, nil, nil, errors.New("unauthorized")
	}

	config := engine.NewPluginConfig(adapters.EngineCredentials(spec, credentials))
	return spec, plugin, config, nil
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

func mergeRowsColumns(rowsResult *engine.GetRowsResult, columns []engine.Column) *model.RowsResult {
	if rowsResult == nil {
		return nil
	}

	if len(columns) == 0 {
		return rowsResultToModel(rowsResult)
	}

	columnInfo := make(map[string]engine.Column, len(columns))
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

	spec, sourceCredentials, err := getSourceSpecForContext(ctx)
	if err != nil {
		return nil, err
	}
	resolvedRef := sourceRefFromInput(ref)
	if resolvedRef == nil {
		return preview, nil
	}

	namespace, objectName := namespaceAndObjectNameForRef(spec, *resolvedRef)
	plugin, config, err := pluginAndConfigForSource(spec, sourceCredentials)
	if err != nil {
		return nil, err
	}

	columns, err := plugin.GetColumnsForTable(config, namespace, objectName)
	if err != nil {
		return nil, err
	}
	if err := plugin.MarkGeneratedColumns(config, namespace, objectName, columns); err != nil {
		log.WithError(err).Warn("Failed to mark generated columns for import preview")
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
	spec, sourceCredentials, err := getSourceSpecForContext(ctx)
	if err != nil {
		return nil, err
	}
	resolvedRef := sourceRefFromInput(input.Ref)
	if resolvedRef == nil {
		return importResult(false, importValidationInvalidOptions), nil
	}

	plugin, config, err := pluginAndConfigForSource(spec, sourceCredentials)
	if err != nil {
		return nil, err
	}

	data, err := readUploadBytes(input.File, maxImportFileSizeBytes)
	if err != nil {
		return importResult(false, validationKeyFromError(err)), nil
	}

	parsed, err := parseImportFile(data, input.Options, maxImportRows, true)
	if err != nil {
		return importResult(false, validationKeyFromError(err)), nil
	}

	namespace, objectName := namespaceAndObjectNameForRef(spec, *resolvedRef)
	columns, err := plugin.GetColumnsForTable(config, namespace, objectName)
	if err != nil {
		return importResult(false, importErrorTableColumns), nil
	}
	if err := plugin.MarkGeneratedColumns(config, namespace, objectName, columns); err != nil {
		log.WithError(err).Warn("Failed to mark generated columns for import")
	}

	allowAutoGenerated := false
	if input.AllowAutoGenerated != nil {
		allowAutoGenerated = *input.AllowAutoGenerated
	}

	_, err = importer.Execute(plugin, config, &importer.ExecuteRequest{
		Schema:             namespace,
		StorageUnit:        objectName,
		Mode:               importer.Mode(input.Mode),
		Parsed:             &importer.ParsedFile{Columns: parsed.columns, Rows: parsed.rows, Truncated: parsed.truncated, Sheet: parsed.sheet},
		Mapping:            importerColumnMappings(input.Mapping),
		AllowAutoGenerated: allowAutoGenerated,
		BatchSize:          importBatchSize,
		TargetColumns:      columns,
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

	spec, sourceCredentials, err := getSourceSpecForContext(ctx)
	if err != nil {
		return nil, err
	}
	resolvedRef := sourceRefFromInput(input.Ref)
	namespace, objectName := namespaceAndObjectNameForRef(spec, *resolvedRef)
	if !mockdata.IsMockDataGenerationAllowed(objectName) {
		return nil, errors.New("mock data generation is not allowed for this table")
	}

	plugin, config, err := pluginAndConfigForSource(spec, sourceCredentials)
	if err != nil {
		return nil, err
	}

	fkRatio := 0
	if input.FkDensityRatio != nil {
		fkRatio = *input.FkDensityRatio
	}
	generator := src.NewMockDataGenerator(fkRatio)
	result, err := generator.Generate(plugin, config, namespace, objectName, input.RowCount, input.OverwriteExisting)
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

	spec, sourceCredentials, err := getSourceSpecForContext(ctx)
	if err != nil {
		return nil, err
	}
	resolvedRef := sourceRefFromInput(&ref)
	namespace, objectName := namespaceAndObjectNameForRef(spec, *resolvedRef)
	if !mockdata.IsMockDataGenerationAllowed(objectName) {
		errMsg := "mock data generation is not allowed for this table"
		return &model.MockDataDependencyAnalysis{Error: &errMsg}, nil
	}

	plugin, config, err := pluginAndConfigForSource(spec, sourceCredentials)
	if err != nil {
		return nil, err
	}

	fkRatio := 0
	if fkDensityRatio != nil {
		fkRatio = *fkDensityRatio
	}
	generator := src.NewMockDataGenerator(fkRatio)
	analysis, err := generator.AnalyzeDependencies(plugin, config, namespace, objectName, rowCount)
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
	spec, sourceCredentials, err := getSourceSpecForContext(ctx)
	if err != nil {
		return nil, err
	}
	plugin, config, err := pluginAndConfigForSource(spec, sourceCredentials)
	if err != nil {
		return nil, err
	}

	var resolvedRef *source.ObjectRef
	if ref != nil {
		resolvedRef = sourceRefFromInput(ref)
	}
	scope := sourceContainerScope(spec, resolvedRef)
	suggestions, err := querysuggestions.FromPlugin(plugin, config, scope)
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

func pluginAndConfigForSource(spec source.TypeSpec, credentials *source.Credentials) (*engine.Plugin, *engine.PluginConfig, error) {
	plugin := src.MainEngine.Choose(engine.DatabaseType(spec.Connector))
	if plugin == nil {
		return nil, nil, fmt.Errorf("unsupported source connector: %s", spec.Connector)
	}

	return plugin, engine.NewPluginConfig(adapters.EngineCredentials(spec, credentials)), nil
}
