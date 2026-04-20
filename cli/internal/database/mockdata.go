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

package database

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	coremockdata "github.com/clidey/whodb/core/src/mockdata"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

// AnalyzeMockDataDependencies analyzes table dependencies for mock data generation.
func (m *Manager) AnalyzeMockDataDependencies(
	schema string,
	table string,
	rowCount int,
	fkDensityRatio int,
) (*coremockdata.DependencyAnalysis, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}
	if rowCount <= 0 {
		return nil, fmt.Errorf("row count must be greater than 0")
	}
	if err := m.ensureMockDataSupported(); err != nil {
		return nil, err
	}

	maxRowCount := coremockdata.GetMockDataGenerationMaxRowCount()
	if rowCount > maxRowCount {
		return nil, fmt.Errorf("row count exceeds maximum limit of %d", maxRowCount)
	}
	if !coremockdata.IsMockDataGenerationAllowed(table) {
		return nil, fmt.Errorf("mock data generation is not allowed for this table")
	}
	if err := m.ensureWritableStorageUnit(schema, table); err != nil {
		return nil, err
	}

	plugin, config, err := m.currentPlugin()
	if err != nil {
		return nil, err
	}

	generator := src.NewMockDataGenerator(fkDensityRatio)
	analysis, err := generator.AnalyzeDependencies(plugin, config, schema, table, rowCount)
	if err != nil {
		return nil, fmt.Errorf("dependency analysis failed: %w", err)
	}
	if analysis.Error != "" {
		return analysis, fmt.Errorf("%s", analysis.Error)
	}

	return analysis, nil
}

// GenerateMockData generates mock data for the target table and its dependencies.
func (m *Manager) GenerateMockData(
	schema string,
	table string,
	rowCount int,
	overwrite bool,
	fkDensityRatio int,
) (*coremockdata.GenerationResult, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}
	if m.config != nil && m.config.GetReadOnly() {
		return nil, ErrReadOnly
	}
	if rowCount <= 0 {
		return nil, fmt.Errorf("row count must be greater than 0")
	}
	if err := m.ensureMockDataSupported(); err != nil {
		return nil, err
	}

	maxRowCount := coremockdata.GetMockDataGenerationMaxRowCount()
	if rowCount > maxRowCount {
		return nil, fmt.Errorf("row count exceeds maximum limit of %d", maxRowCount)
	}
	if !coremockdata.IsMockDataGenerationAllowed(table) {
		return nil, fmt.Errorf("mock data generation is not allowed for this table")
	}
	if err := m.ensureWritableStorageUnit(schema, table); err != nil {
		return nil, err
	}

	plugin, config, err := m.currentPlugin()
	if err != nil {
		return nil, err
	}

	generator := src.NewMockDataGenerator(fkDensityRatio)
	result, err := generator.Generate(plugin, config, schema, table, rowCount, overwrite)
	if err != nil {
		return nil, fmt.Errorf("mock data generation failed: %w", err)
	}

	if m.cache != nil {
		m.cache.Clear()
	}

	return result, nil
}

func (m *Manager) ensureMockDataSupported() error {
	if m.currentConnection == nil {
		return fmt.Errorf("not connected to any database")
	}

	spec, ok := sourcecatalog.Find(m.currentConnection.Type)
	if ok && !spec.Contract.SupportsAction(source.ActionGenerateMockData) {
		return fmt.Errorf("mock data generation is not supported for %s", spec.Label)
	}

	return nil
}

func (m *Manager) ensureWritableStorageUnit(schema, table string) error {
	storageUnits, err := m.GetStorageUnits(schema)
	if err != nil {
		return err
	}

	for _, unit := range storageUnits {
		if unit.Name != table {
			continue
		}
		if storageUnitTypeLooksLikeView(unit) {
			unitType := storageUnitType(unit)
			if unitType == "" {
				unitType = "view"
			}
			return fmt.Errorf("mock data generation requires a writable table or collection; %s is a %s", table, unitType)
		}
		return nil
	}

	return nil
}

func storageUnitTypeLooksLikeView(unit engine.StorageUnit) bool {
	return strings.Contains(strings.ToLower(storageUnitType(unit)), "view")
}

func storageUnitType(unit engine.StorageUnit) string {
	for _, attr := range unit.Attributes {
		if strings.EqualFold(attr.Key, "Type") {
			return attr.Value
		}
	}
	return ""
}
