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

// Package schemadiff compares schema metadata between two connections using the
// same metadata APIs that power the CLI and TUI.
package schemadiff

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/core/src/dbcatalog"
	"github.com/clidey/whodb/core/src/engine"
)

type snapshot struct {
	Connection   string
	Type         string
	Schema       string
	StorageUnits []storageUnitSnapshot
}

type storageUnitSnapshot struct {
	Name          string
	Kind          string
	Columns       []ColumnState
	Relationships []RelationshipState
}

// Result is the full schema diff output returned by CompareConnections.
type Result struct {
	From         SchemaReference     `json:"from"`
	To           SchemaReference     `json:"to"`
	Summary      Summary             `json:"summary"`
	StorageUnits []StorageUnitChange `json:"storageUnits,omitempty"`
}

// SchemaReference identifies one side of a schema comparison.
type SchemaReference struct {
	Connection string `json:"connection"`
	Type       string `json:"type"`
	Schema     string `json:"schema,omitempty"`
}

// Summary contains aggregate counts for the schema diff.
type Summary struct {
	HasDifferences       bool `json:"hasDifferences"`
	AddedStorageUnits    int  `json:"addedStorageUnits"`
	RemovedStorageUnits  int  `json:"removedStorageUnits"`
	ChangedStorageUnits  int  `json:"changedStorageUnits"`
	AddedColumns         int  `json:"addedColumns"`
	RemovedColumns       int  `json:"removedColumns"`
	ChangedColumns       int  `json:"changedColumns"`
	AddedRelationships   int  `json:"addedRelationships"`
	RemovedRelationships int  `json:"removedRelationships"`
	ChangedRelationships int  `json:"changedRelationships"`
}

// StorageUnitChange describes how one table, collection, or similar storage
// unit changed between the two connections.
type StorageUnitChange struct {
	Name          string               `json:"name"`
	Change        string               `json:"change"`
	FromKind      string               `json:"fromKind,omitempty"`
	ToKind        string               `json:"toKind,omitempty"`
	Differences   []FieldChange        `json:"differences,omitempty"`
	Columns       []ColumnChange       `json:"columns,omitempty"`
	Relationships []RelationshipChange `json:"relationships,omitempty"`
}

// ColumnChange describes how one column changed between the two connections.
type ColumnChange struct {
	Name        string        `json:"name"`
	Change      string        `json:"change"`
	From        *ColumnState  `json:"from,omitempty"`
	To          *ColumnState  `json:"to,omitempty"`
	Differences []FieldChange `json:"differences,omitempty"`
}

// ColumnState captures the normalized schema properties for a single column.
type ColumnState struct {
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	IsNullable       bool     `json:"isNullable"`
	IsPrimary        bool     `json:"isPrimary"`
	IsAutoIncrement  bool     `json:"isAutoIncrement"`
	IsComputed       bool     `json:"isComputed"`
	IsForeignKey     bool     `json:"isForeignKey"`
	IsUnique         bool     `json:"isUnique"`
	ReferencedTable  string   `json:"referencedTable,omitempty"`
	ReferencedColumn string   `json:"referencedColumn,omitempty"`
	DefaultValue     string   `json:"defaultValue,omitempty"`
	CheckValues      []string `json:"checkValues,omitempty"`
	Length           *int     `json:"length,omitempty"`
	Precision        *int     `json:"precision,omitempty"`
	Scale            *int     `json:"scale,omitempty"`
}

// RelationshipChange describes how one graph relationship changed between the
// two connections.
type RelationshipChange struct {
	TargetStorageUnit string             `json:"targetStorageUnit"`
	Change            string             `json:"change"`
	From              *RelationshipState `json:"from,omitempty"`
	To                *RelationshipState `json:"to,omitempty"`
	Differences       []FieldChange      `json:"differences,omitempty"`
}

// RelationshipState captures the normalized metadata for one relationship edge.
type RelationshipState struct {
	TargetStorageUnit string `json:"targetStorageUnit"`
	RelationshipType  string `json:"relationshipType,omitempty"`
	SourceColumn      string `json:"sourceColumn,omitempty"`
	TargetColumn      string `json:"targetColumn,omitempty"`
}

// FieldChange records a single property change inside a column, relationship,
// or storage-unit diff.
type FieldChange struct {
	Field string `json:"field"`
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
}

// CompareConnections captures schema metadata for two connections and returns a
// normalized diff result.
func CompareConnections(
	fromConn *database.Connection,
	toConn *database.Connection,
	fromSchemaHint string,
	toSchemaHint string,
) (*Result, error) {
	if fromConn == nil {
		return nil, fmt.Errorf("source connection is required")
	}
	if toConn == nil {
		return nil, fmt.Errorf("target connection is required")
	}

	fromSnapshot, err := captureSnapshot(fromConn, fromSchemaHint)
	if err != nil {
		return nil, fmt.Errorf("capture %s: %w", fromConn.Name, err)
	}

	toSnapshot, err := captureSnapshot(toConn, toSchemaHint)
	if err != nil {
		return nil, fmt.Errorf("capture %s: %w", toConn.Name, err)
	}

	return buildResult(fromSnapshot, toSnapshot), nil
}

// RenderText renders a human-readable schema diff summary.
func RenderText(result *Result) string {
	if result == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString("Schema Diff\n")
	fmt.Fprintf(&b, "  From: %s (%s)", result.From.Connection, result.From.Type)
	if result.From.Schema != "" {
		fmt.Fprintf(&b, " schema=%s", result.From.Schema)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "  To:   %s (%s)", result.To.Connection, result.To.Type)
	if result.To.Schema != "" {
		fmt.Fprintf(&b, " schema=%s", result.To.Schema)
	}
	b.WriteString("\n\n")

	b.WriteString("Summary\n")
	fmt.Fprintf(&b, "  Storage units: +%d -%d ~%d\n",
		result.Summary.AddedStorageUnits,
		result.Summary.RemovedStorageUnits,
		result.Summary.ChangedStorageUnits,
	)
	fmt.Fprintf(&b, "  Columns:       +%d -%d ~%d\n",
		result.Summary.AddedColumns,
		result.Summary.RemovedColumns,
		result.Summary.ChangedColumns,
	)
	fmt.Fprintf(&b, "  Relationships: +%d -%d ~%d\n",
		result.Summary.AddedRelationships,
		result.Summary.RemovedRelationships,
		result.Summary.ChangedRelationships,
	)

	if !result.Summary.HasDifferences {
		b.WriteString("\nNo schema differences found.\n")
		return b.String()
	}

	renderStorageUnitSection(&b, "Added Storage Units", result.StorageUnits, "added")
	renderStorageUnitSection(&b, "Removed Storage Units", result.StorageUnits, "removed")
	renderStorageUnitSection(&b, "Changed Storage Units", result.StorageUnits, "changed")

	return b.String()
}

func captureSnapshot(conn *database.Connection, schemaHint string) (*snapshot, error) {
	mgr, err := database.NewManager()
	if err != nil {
		return nil, fmt.Errorf("cannot initialize database manager: %w", err)
	}

	if err := mgr.Connect(conn); err != nil {
		return nil, fmt.Errorf("cannot connect to database: %w", err)
	}
	defer mgr.Disconnect()

	schemaName, err := resolveSchemaName(mgr, conn, schemaHint)
	if err != nil {
		return nil, err
	}

	storageUnits, err := mgr.GetStorageUnits(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to load storage units: %w", err)
	}
	sort.Slice(storageUnits, func(i, j int) bool {
		return storageUnits[i].Name < storageUnits[j].Name
	})

	graphUnits, err := mgr.GetGraph(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph data: %w", err)
	}
	relationshipsByUnit := relationshipsFromGraph(graphUnits)

	storageUnitNames := make([]string, 0, len(storageUnits))
	for _, storageUnit := range storageUnits {
		storageUnitNames = append(storageUnitNames, storageUnit.Name)
	}

	columnsByUnit, err := mgr.GetColumnsForStorageUnits(schemaName, storageUnitNames)
	if err != nil {
		return nil, fmt.Errorf("failed to load columns: %w", err)
	}

	constraintsByUnit, err := mgr.GetColumnConstraintsForStorageUnits(schemaName, storageUnitNames)
	if err != nil {
		return nil, fmt.Errorf("failed to load column constraints: %w", err)
	}

	result := &snapshot{
		Connection:   conn.Name,
		Type:         conn.Type,
		Schema:       schemaName,
		StorageUnits: make([]storageUnitSnapshot, 0, len(storageUnits)),
	}

	for _, storageUnit := range storageUnits {
		columns := columnsByUnit[storageUnit.Name]
		sort.Slice(columns, func(i, j int) bool {
			return columns[i].Name < columns[j].Name
		})

		columnStates := make([]ColumnState, 0, len(columns))
		for _, column := range columns {
			columnStates = append(columnStates, snapshotColumn(column, constraintsByUnit[storageUnit.Name][column.Name]))
		}

		relationships := slices.Clone(relationshipsByUnit[storageUnit.Name])
		sort.Slice(relationships, func(i, j int) bool {
			return relationshipSortKey(relationships[i]) < relationshipSortKey(relationships[j])
		})

		result.StorageUnits = append(result.StorageUnits, storageUnitSnapshot{
			Name:          storageUnit.Name,
			Kind:          storageUnitKind(storageUnit),
			Columns:       columnStates,
			Relationships: relationships,
		})
	}

	return result, nil
}

func resolveSchemaName(mgr *database.Manager, conn *database.Connection, explicitSchema string) (string, error) {
	if strings.TrimSpace(explicitSchema) != "" {
		return explicitSchema, nil
	}
	if entry, ok := dbcatalog.Find(conn.Type); ok && entry.UsesDatabaseInsteadOfSchema && strings.TrimSpace(conn.Database) != "" {
		return conn.Database, nil
	}
	if strings.TrimSpace(conn.Schema) != "" {
		return conn.Schema, nil
	}

	schemas, err := mgr.GetSchemas()
	if err != nil || len(schemas) == 0 {
		return "", nil
	}

	return schemas[0], nil
}

func snapshotColumn(column engine.Column, constraints map[string]any) ColumnState {
	referencedTable := ""
	if column.ReferencedTable != nil {
		referencedTable = *column.ReferencedTable
	}
	referencedColumn := ""
	if column.ReferencedColumn != nil {
		referencedColumn = *column.ReferencedColumn
	}

	return ColumnState{
		Name:             column.Name,
		Type:             column.Type,
		IsNullable:       column.IsNullable,
		IsPrimary:        column.IsPrimary,
		IsAutoIncrement:  column.IsAutoIncrement,
		IsComputed:       column.IsComputed,
		IsForeignKey:     column.IsForeignKey,
		IsUnique:         normalizeBool(constraints["unique"]) && !column.IsPrimary,
		ReferencedTable:  referencedTable,
		ReferencedColumn: referencedColumn,
		DefaultValue:     normalizeDefaultValue(constraints["default"]),
		CheckValues:      normalizeStringSlice(constraints["check_values"]),
		Length:           column.Length,
		Precision:        column.Precision,
		Scale:            column.Scale,
	}
}

func storageUnitKind(storageUnit engine.StorageUnit) string {
	for _, attribute := range storageUnit.Attributes {
		if strings.EqualFold(attribute.Key, "Type") {
			return attribute.Value
		}
	}
	return ""
}

func relationshipsFromGraph(graphUnits []engine.GraphUnit) map[string][]RelationshipState {
	result := make(map[string][]RelationshipState, len(graphUnits))

	for _, graphUnit := range graphUnits {
		relationships := make([]RelationshipState, 0, len(graphUnit.Relations))
		for _, relation := range graphUnit.Relations {
			relationship := RelationshipState{
				TargetStorageUnit: relation.Name,
				RelationshipType:  string(relation.RelationshipType),
			}
			if relation.SourceColumn != nil {
				relationship.SourceColumn = *relation.SourceColumn
			}
			if relation.TargetColumn != nil {
				relationship.TargetColumn = *relation.TargetColumn
			}
			relationships = append(relationships, relationship)
		}
		result[graphUnit.Unit.Name] = relationships
	}

	return result
}

func buildResult(fromSnapshot, toSnapshot *snapshot) *Result {
	result := &Result{
		From: SchemaReference{
			Connection: fromSnapshot.Connection,
			Type:       fromSnapshot.Type,
			Schema:     fromSnapshot.Schema,
		},
		To: SchemaReference{
			Connection: toSnapshot.Connection,
			Type:       toSnapshot.Type,
			Schema:     toSnapshot.Schema,
		},
	}

	fromUnits := make(map[string]storageUnitSnapshot, len(fromSnapshot.StorageUnits))
	for _, storageUnit := range fromSnapshot.StorageUnits {
		fromUnits[storageUnit.Name] = storageUnit
	}
	toUnits := make(map[string]storageUnitSnapshot, len(toSnapshot.StorageUnits))
	for _, storageUnit := range toSnapshot.StorageUnits {
		toUnits[storageUnit.Name] = storageUnit
	}

	unitNames := mergeSortedNames(fromUnits, toUnits)

	var added []StorageUnitChange
	var removed []StorageUnitChange
	var changed []StorageUnitChange

	for _, name := range unitNames {
		fromUnit, hasFrom := fromUnits[name]
		toUnit, hasTo := toUnits[name]

		switch {
		case !hasFrom && hasTo:
			result.Summary.AddedStorageUnits++
			result.Summary.AddedColumns += len(toUnit.Columns)
			result.Summary.AddedRelationships += len(toUnit.Relationships)
			added = append(added, StorageUnitChange{
				Name:          name,
				Change:        "added",
				ToKind:        toUnit.Kind,
				Columns:       buildAddedColumnChanges(toUnit.Columns),
				Relationships: buildAddedRelationshipChanges(toUnit.Relationships),
			})
		case hasFrom && !hasTo:
			result.Summary.RemovedStorageUnits++
			result.Summary.RemovedColumns += len(fromUnit.Columns)
			result.Summary.RemovedRelationships += len(fromUnit.Relationships)
			removed = append(removed, StorageUnitChange{
				Name:          name,
				Change:        "removed",
				FromKind:      fromUnit.Kind,
				Columns:       buildRemovedColumnChanges(fromUnit.Columns),
				Relationships: buildRemovedRelationshipChanges(fromUnit.Relationships),
			})
		default:
			change := StorageUnitChange{
				Name:     name,
				Change:   "changed",
				FromKind: fromUnit.Kind,
				ToKind:   toUnit.Kind,
			}
			if fromUnit.Kind != toUnit.Kind {
				change.Differences = append(change.Differences, FieldChange{
					Field: "kind",
					From:  fromUnit.Kind,
					To:    toUnit.Kind,
				})
			}

			columnChanges, columnSummary := diffColumns(fromUnit.Columns, toUnit.Columns)
			change.Columns = columnChanges
			result.Summary.AddedColumns += columnSummary.AddedColumns
			result.Summary.RemovedColumns += columnSummary.RemovedColumns
			result.Summary.ChangedColumns += columnSummary.ChangedColumns

			relationshipChanges, relationshipSummary := diffRelationships(fromUnit.Relationships, toUnit.Relationships)
			change.Relationships = relationshipChanges
			result.Summary.AddedRelationships += relationshipSummary.AddedRelationships
			result.Summary.RemovedRelationships += relationshipSummary.RemovedRelationships
			result.Summary.ChangedRelationships += relationshipSummary.ChangedRelationships

			if len(change.Differences) == 0 && len(change.Columns) == 0 && len(change.Relationships) == 0 {
				continue
			}

			result.Summary.ChangedStorageUnits++
			changed = append(changed, change)
		}
	}

	result.StorageUnits = append(result.StorageUnits, added...)
	result.StorageUnits = append(result.StorageUnits, removed...)
	result.StorageUnits = append(result.StorageUnits, changed...)
	result.Summary.HasDifferences = len(result.StorageUnits) > 0

	return result
}

func mergeSortedNames[T any](from map[string]T, to map[string]T) []string {
	names := make([]string, 0, len(from)+len(to))
	seen := make(map[string]struct{}, len(from)+len(to))

	for name := range from {
		names = append(names, name)
		seen[name] = struct{}{}
	}
	for name := range to {
		if _, ok := seen[name]; ok {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func buildAddedColumnChanges(columns []ColumnState) []ColumnChange {
	changes := make([]ColumnChange, 0, len(columns))
	for _, column := range columns {
		columnCopy := column
		changes = append(changes, ColumnChange{
			Name:   column.Name,
			Change: "added",
			To:     &columnCopy,
		})
	}
	return changes
}

func buildRemovedColumnChanges(columns []ColumnState) []ColumnChange {
	changes := make([]ColumnChange, 0, len(columns))
	for _, column := range columns {
		columnCopy := column
		changes = append(changes, ColumnChange{
			Name:   column.Name,
			Change: "removed",
			From:   &columnCopy,
		})
	}
	return changes
}

func diffColumns(fromColumns, toColumns []ColumnState) ([]ColumnChange, Summary) {
	fromMap := make(map[string]ColumnState, len(fromColumns))
	for _, column := range fromColumns {
		fromMap[column.Name] = column
	}
	toMap := make(map[string]ColumnState, len(toColumns))
	for _, column := range toColumns {
		toMap[column.Name] = column
	}

	columnNames := mergeSortedNames(fromMap, toMap)
	changes := make([]ColumnChange, 0)
	summary := Summary{}

	for _, name := range columnNames {
		fromColumn, hasFrom := fromMap[name]
		toColumn, hasTo := toMap[name]

		switch {
		case !hasFrom && hasTo:
			summary.AddedColumns++
			columnCopy := toColumn
			changes = append(changes, ColumnChange{
				Name:   name,
				Change: "added",
				To:     &columnCopy,
			})
		case hasFrom && !hasTo:
			summary.RemovedColumns++
			columnCopy := fromColumn
			changes = append(changes, ColumnChange{
				Name:   name,
				Change: "removed",
				From:   &columnCopy,
			})
		default:
			differences := diffColumnFields(fromColumn, toColumn)
			if len(differences) == 0 {
				continue
			}

			summary.ChangedColumns++
			fromCopy := fromColumn
			toCopy := toColumn
			changes = append(changes, ColumnChange{
				Name:        name,
				Change:      "changed",
				From:        &fromCopy,
				To:          &toCopy,
				Differences: differences,
			})
		}
	}

	return changes, summary
}

func diffColumnFields(fromColumn, toColumn ColumnState) []FieldChange {
	var changes []FieldChange

	appendIfChanged := func(field string, fromValue, toValue string) {
		if fromValue == toValue {
			return
		}
		changes = append(changes, FieldChange{
			Field: field,
			From:  fromValue,
			To:    toValue,
		})
	}

	appendIfChanged("type", fromColumn.Type, toColumn.Type)
	appendIfChanged("nullable", strconv.FormatBool(fromColumn.IsNullable), strconv.FormatBool(toColumn.IsNullable))
	appendIfChanged("primary", strconv.FormatBool(fromColumn.IsPrimary), strconv.FormatBool(toColumn.IsPrimary))
	appendIfChanged("autoIncrement", strconv.FormatBool(fromColumn.IsAutoIncrement), strconv.FormatBool(toColumn.IsAutoIncrement))
	appendIfChanged("computed", strconv.FormatBool(fromColumn.IsComputed), strconv.FormatBool(toColumn.IsComputed))
	appendIfChanged("foreignKey", strconv.FormatBool(fromColumn.IsForeignKey), strconv.FormatBool(toColumn.IsForeignKey))
	appendIfChanged("unique", strconv.FormatBool(fromColumn.IsUnique), strconv.FormatBool(toColumn.IsUnique))
	appendIfChanged("referencedTable", fromColumn.ReferencedTable, toColumn.ReferencedTable)
	appendIfChanged("referencedColumn", fromColumn.ReferencedColumn, toColumn.ReferencedColumn)
	appendIfChanged("defaultValue", fromColumn.DefaultValue, toColumn.DefaultValue)
	appendIfChanged("checkValues", strings.Join(fromColumn.CheckValues, ", "), strings.Join(toColumn.CheckValues, ", "))
	appendIfChanged("length", formatOptionalInt(fromColumn.Length), formatOptionalInt(toColumn.Length))
	appendIfChanged("precision", formatOptionalInt(fromColumn.Precision), formatOptionalInt(toColumn.Precision))
	appendIfChanged("scale", formatOptionalInt(fromColumn.Scale), formatOptionalInt(toColumn.Scale))

	return changes
}

func buildAddedRelationshipChanges(relationships []RelationshipState) []RelationshipChange {
	changes := make([]RelationshipChange, 0, len(relationships))
	for _, relationship := range relationships {
		relationshipCopy := relationship
		changes = append(changes, RelationshipChange{
			TargetStorageUnit: relationship.TargetStorageUnit,
			Change:            "added",
			To:                &relationshipCopy,
		})
	}
	return changes
}

func buildRemovedRelationshipChanges(relationships []RelationshipState) []RelationshipChange {
	changes := make([]RelationshipChange, 0, len(relationships))
	for _, relationship := range relationships {
		relationshipCopy := relationship
		changes = append(changes, RelationshipChange{
			TargetStorageUnit: relationship.TargetStorageUnit,
			Change:            "removed",
			From:              &relationshipCopy,
		})
	}
	return changes
}

func diffRelationships(fromRelationships, toRelationships []RelationshipState) ([]RelationshipChange, Summary) {
	fromMap := make(map[string]RelationshipState, len(fromRelationships))
	for _, relationship := range fromRelationships {
		fromMap[relationshipIdentity(relationship)] = relationship
	}
	toMap := make(map[string]RelationshipState, len(toRelationships))
	for _, relationship := range toRelationships {
		toMap[relationshipIdentity(relationship)] = relationship
	}

	relationshipIDs := mergeSortedNames(fromMap, toMap)
	changes := make([]RelationshipChange, 0)
	summary := Summary{}

	for _, id := range relationshipIDs {
		fromRelationship, hasFrom := fromMap[id]
		toRelationship, hasTo := toMap[id]

		switch {
		case !hasFrom && hasTo:
			summary.AddedRelationships++
			relationshipCopy := toRelationship
			changes = append(changes, RelationshipChange{
				TargetStorageUnit: toRelationship.TargetStorageUnit,
				Change:            "added",
				To:                &relationshipCopy,
			})
		case hasFrom && !hasTo:
			summary.RemovedRelationships++
			relationshipCopy := fromRelationship
			changes = append(changes, RelationshipChange{
				TargetStorageUnit: fromRelationship.TargetStorageUnit,
				Change:            "removed",
				From:              &relationshipCopy,
			})
		default:
			differences := diffRelationshipFields(fromRelationship, toRelationship)
			if len(differences) == 0 {
				continue
			}

			summary.ChangedRelationships++
			fromCopy := fromRelationship
			toCopy := toRelationship
			changes = append(changes, RelationshipChange{
				TargetStorageUnit: toRelationship.TargetStorageUnit,
				Change:            "changed",
				From:              &fromCopy,
				To:                &toCopy,
				Differences:       differences,
			})
		}
	}

	return changes, summary
}

func relationshipIdentity(relationship RelationshipState) string {
	return strings.ToLower(strings.Join([]string{
		relationship.TargetStorageUnit,
		relationship.SourceColumn,
		relationship.TargetColumn,
	}, "\x00"))
}

func relationshipSortKey(relationship RelationshipState) string {
	return strings.ToLower(strings.Join([]string{
		relationship.TargetStorageUnit,
		relationship.RelationshipType,
		relationship.SourceColumn,
		relationship.TargetColumn,
	}, "\x00"))
}

func diffRelationshipFields(fromRelationship, toRelationship RelationshipState) []FieldChange {
	var changes []FieldChange

	appendIfChanged := func(field string, fromValue, toValue string) {
		if fromValue == toValue {
			return
		}
		changes = append(changes, FieldChange{
			Field: field,
			From:  fromValue,
			To:    toValue,
		})
	}

	appendIfChanged("relationshipType", fromRelationship.RelationshipType, toRelationship.RelationshipType)
	appendIfChanged("sourceColumn", fromRelationship.SourceColumn, toRelationship.SourceColumn)
	appendIfChanged("targetColumn", fromRelationship.TargetColumn, toRelationship.TargetColumn)

	return changes
}

func normalizeBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		return err == nil && parsed
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func normalizeDefaultValue(value any) string {
	normalized := normalizeScalarString(value)
	if normalized == "" {
		return ""
	}

	normalized = strings.TrimSpace(normalized)
	normalized = trimWrappedParens(normalized)
	if idx := strings.Index(normalized, "::"); idx >= 0 {
		normalized = strings.TrimSpace(normalized[:idx])
	}
	normalized = trimWrappedQuotes(normalized)

	switch strings.ToLower(normalized) {
	case "current_timestamp", "current_timestamp()", "now()", "localtimestamp", "localtimestamp()":
		return "current_timestamp"
	default:
		return normalized
	}
}

func normalizeStringSlice(value any) []string {
	var items []string

	switch v := value.(type) {
	case []string:
		items = append(items, v...)
	case []any:
		for _, item := range v {
			if text := trimWrappedQuotes(normalizeScalarString(item)); text != "" {
				items = append(items, text)
			}
		}
	case string:
		if text := trimWrappedQuotes(strings.TrimSpace(v)); text != "" {
			items = append(items, text)
		}
	}

	slices.Sort(items)
	return items
}

func normalizeScalarString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func trimWrappedParens(value string) string {
	for {
		trimmed := strings.TrimSpace(value)
		if len(trimmed) < 2 || trimmed[0] != '(' || trimmed[len(trimmed)-1] != ')' {
			return trimmed
		}
		if !hasBalancedOuterParens(trimmed) {
			return trimmed
		}
		value = trimmed[1 : len(trimmed)-1]
	}
}

func hasBalancedOuterParens(value string) bool {
	depth := 0
	for i, ch := range value {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 && i < len(value)-1 {
				return false
			}
		}
	}
	return depth == 0
}

func trimWrappedQuotes(value string) string {
	trimmed := strings.TrimSpace(value)
	for len(trimmed) >= 2 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') || (first == '`' && last == '`') {
			trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			continue
		}
		break
	}
	return trimmed
}

func renderStorageUnitSection(b *strings.Builder, title string, storageUnits []StorageUnitChange, change string) {
	sectionUnits := make([]StorageUnitChange, 0)
	for _, storageUnit := range storageUnits {
		if storageUnit.Change == change {
			sectionUnits = append(sectionUnits, storageUnit)
		}
	}
	if len(sectionUnits) == 0 {
		return
	}

	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n")

	for _, storageUnit := range sectionUnits {
		switch storageUnit.Change {
		case "added":
			fmt.Fprintf(b, "  + %s", storageUnit.Name)
			if storageUnit.ToKind != "" {
				fmt.Fprintf(b, " (%s)", storageUnit.ToKind)
			}
			fmt.Fprintf(b, " [%d columns, %d relationships]\n", len(storageUnit.Columns), len(storageUnit.Relationships))
		case "removed":
			fmt.Fprintf(b, "  - %s", storageUnit.Name)
			if storageUnit.FromKind != "" {
				fmt.Fprintf(b, " (%s)", storageUnit.FromKind)
			}
			fmt.Fprintf(b, " [%d columns, %d relationships]\n", len(storageUnit.Columns), len(storageUnit.Relationships))
		case "changed":
			fmt.Fprintf(b, "  ~ %s\n", storageUnit.Name)
			for _, difference := range storageUnit.Differences {
				fmt.Fprintf(b, "      %s: %s -> %s\n", difference.Field, printableValue(difference.From), printableValue(difference.To))
			}
			for _, relationship := range storageUnit.Relationships {
				renderRelationshipChange(b, relationship)
			}
			for _, column := range storageUnit.Columns {
				renderColumnChange(b, column)
			}
		}
	}
}

func renderColumnChange(b *strings.Builder, column ColumnChange) {
	switch column.Change {
	case "added":
		fmt.Fprintf(b, "      + %s %s\n", column.Name, describeColumnState(column.To))
	case "removed":
		fmt.Fprintf(b, "      - %s %s\n", column.Name, describeColumnState(column.From))
	case "changed":
		fmt.Fprintf(b, "      ~ %s\n", column.Name)
		for _, difference := range column.Differences {
			fmt.Fprintf(b, "          %s: %s -> %s\n", difference.Field, printableValue(difference.From), printableValue(difference.To))
		}
	}
}

func renderRelationshipChange(b *strings.Builder, relationship RelationshipChange) {
	switch relationship.Change {
	case "added":
		fmt.Fprintf(b, "      + relationship %s %s\n", relationship.TargetStorageUnit, describeRelationshipState(relationship.To))
	case "removed":
		fmt.Fprintf(b, "      - relationship %s %s\n", relationship.TargetStorageUnit, describeRelationshipState(relationship.From))
	case "changed":
		fmt.Fprintf(b, "      ~ relationship %s\n", relationship.TargetStorageUnit)
		for _, difference := range relationship.Differences {
			fmt.Fprintf(b, "          %s: %s -> %s\n", difference.Field, printableValue(difference.From), printableValue(difference.To))
		}
	}
}

func describeColumnState(column *ColumnState) string {
	if column == nil {
		return ""
	}

	parts := []string{column.Type}
	if !column.IsNullable {
		parts = append(parts, "not null")
	}
	if column.IsPrimary {
		parts = append(parts, "primary")
	}
	if column.IsUnique {
		parts = append(parts, "unique")
	}
	if column.IsAutoIncrement {
		parts = append(parts, "auto increment")
	}
	if column.IsComputed {
		parts = append(parts, "computed")
	}
	if column.DefaultValue != "" {
		parts = append(parts, "default="+column.DefaultValue)
	}
	if len(column.CheckValues) > 0 {
		parts = append(parts, "check=["+strings.Join(column.CheckValues, ", ")+"]")
	}
	if column.IsForeignKey {
		target := column.ReferencedTable
		if column.ReferencedColumn != "" {
			target += "." + column.ReferencedColumn
		}
		if target != "" {
			parts = append(parts, "fk="+target)
		} else {
			parts = append(parts, "foreign key")
		}
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

func describeRelationshipState(relationship *RelationshipState) string {
	if relationship == nil {
		return ""
	}

	parts := make([]string, 0, 3)
	if relationship.RelationshipType != "" {
		parts = append(parts, relationship.RelationshipType)
	}
	if relationship.SourceColumn != "" && relationship.TargetColumn != "" {
		parts = append(parts, relationship.SourceColumn+" -> "+relationship.TargetColumn)
	} else if relationship.SourceColumn != "" {
		parts = append(parts, "source="+relationship.SourceColumn)
	} else if relationship.TargetColumn != "" {
		parts = append(parts, "target="+relationship.TargetColumn)
	}
	if len(parts) == 0 {
		return ""
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func printableValue(value string) string {
	if value == "" {
		return "<empty>"
	}
	return value
}
