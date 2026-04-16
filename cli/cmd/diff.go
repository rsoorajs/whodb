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

package cmd

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/clidey/whodb/core/src/dbcatalog"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/spf13/cobra"
)

var (
	diffFromConnection string
	diffToConnection   string
	diffSchema         string
	diffFromSchema     string
	diffToSchema       string
	diffFormat         string
	diffQuiet          bool
)

var diffCmd = &cobra.Command{
	Use:           "diff",
	Short:         "Compare schema metadata between two connections",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Compare schema metadata between two connections.

The first pass focuses on structural differences:
  - Added, removed, and changed storage units
  - Added, removed, and changed columns
  - Column properties such as type, nullability, primary key, foreign key, and generated flags

By default, the CLI compares each connection's configured schema or the first
available schema. Use --schema to compare the same schema name on both sides,
or use --from-schema and --to-schema when the schema names differ.

For database-scoped connections such as MySQL and MariaDB, the diff command
uses the connection's configured database by default when no schema flag is
provided.`,
	Example: `  # Compare two saved connections using their default schemas
  whodb-cli diff --from staging --to prod

  # Compare the same schema name on both connections
  whodb-cli diff --from staging --to prod --schema public

  # Compare Postgres to MySQL by using each connection's configured namespace
  whodb-cli diff --from dev-e2e_postgres-1 --to dev-e2e_mysql-1

  # Compare two different schema names
  whodb-cli diff --from dev --to prod --from-schema app_dev --to-schema app_prod

  # Emit machine-readable JSON
  whodb-cli diff --from staging --to prod --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(diffFromConnection) == "" {
			return fmt.Errorf("--from is required")
		}
		if strings.TrimSpace(diffToConnection) == "" {
			return fmt.Errorf("--to is required")
		}

		format, err := resolveSchemaDiffFormat(diffFormat)
		if err != nil {
			return err
		}
		quiet := diffQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		fromConn, _, err := mgr.ResolveConnection(diffFromConnection)
		if err != nil {
			return err
		}
		toConn, _, err := mgr.ResolveConnection(diffToConnection)
		if err != nil {
			return err
		}

		fromSchemaName := diffFromSchema
		if fromSchemaName == "" {
			fromSchemaName = diffSchema
		}
		toSchemaName := diffToSchema
		if toSchemaName == "" {
			toSchemaName = diffSchema
		}

		fromSnapshot, err := captureSchemaSnapshot(fromConn, fromSchemaName, quiet, out)
		if err != nil {
			return fmt.Errorf("capture %s: %w", fromConn.Name, err)
		}
		toSnapshot, err := captureSchemaSnapshot(toConn, toSchemaName, quiet, out)
		if err != nil {
			return fmt.Errorf("capture %s: %w", toConn.Name, err)
		}

		diffResult := buildSchemaDiffOutput(fromSnapshot, toSnapshot)
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "diff", diffResult)
		}

		printSchemaDiff(cmd.OutOrStdout(), diffResult)
		return nil
	},
}

type schemaSnapshot struct {
	Connection   string                `json:"connection"`
	Type         string                `json:"type"`
	Schema       string                `json:"schema,omitempty"`
	StorageUnits []storageUnitSnapshot `json:"storageUnits"`
}

type storageUnitSnapshot struct {
	Name    string           `json:"name"`
	Kind    string           `json:"kind,omitempty"`
	Columns []columnSnapshot `json:"columns"`
}

type columnSnapshot struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	IsNullable       bool   `json:"isNullable"`
	IsPrimary        bool   `json:"isPrimary"`
	IsAutoIncrement  bool   `json:"isAutoIncrement"`
	IsComputed       bool   `json:"isComputed"`
	IsForeignKey     bool   `json:"isForeignKey"`
	ReferencedTable  string `json:"referencedTable,omitempty"`
	ReferencedColumn string `json:"referencedColumn,omitempty"`
	Length           *int   `json:"length,omitempty"`
	Precision        *int   `json:"precision,omitempty"`
	Scale            *int   `json:"scale,omitempty"`
}

type schemaDiffOutput struct {
	From         schemaReference   `json:"from"`
	To           schemaReference   `json:"to"`
	Summary      schemaDiffSummary `json:"summary"`
	StorageUnits []storageUnitDiff `json:"storageUnits,omitempty"`
}

type schemaReference struct {
	Connection string `json:"connection"`
	Type       string `json:"type"`
	Schema     string `json:"schema,omitempty"`
}

type schemaDiffSummary struct {
	HasDifferences      bool `json:"hasDifferences"`
	AddedStorageUnits   int  `json:"addedStorageUnits"`
	RemovedStorageUnits int  `json:"removedStorageUnits"`
	ChangedStorageUnits int  `json:"changedStorageUnits"`
	AddedColumns        int  `json:"addedColumns"`
	RemovedColumns      int  `json:"removedColumns"`
	ChangedColumns      int  `json:"changedColumns"`
}

type storageUnitDiff struct {
	Name        string       `json:"name"`
	Change      string       `json:"change"`
	FromKind    string       `json:"fromKind,omitempty"`
	ToKind      string       `json:"toKind,omitempty"`
	Differences []fieldDiff  `json:"differences,omitempty"`
	Columns     []columnDiff `json:"columns,omitempty"`
}

type columnDiff struct {
	Name        string          `json:"name"`
	Change      string          `json:"change"`
	From        *columnSnapshot `json:"from,omitempty"`
	To          *columnSnapshot `json:"to,omitempty"`
	Differences []fieldDiff     `json:"differences,omitempty"`
}

type fieldDiff struct {
	Field string `json:"field"`
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
}

func init() {
	rootCmd.AddCommand(diffCmd)

	diffCmd.Flags().StringVar(&diffFromConnection, "from", "", "source connection name (required)")
	diffCmd.Flags().StringVar(&diffToConnection, "to", "", "target connection name (required)")
	diffCmd.Flags().StringVar(&diffSchema, "schema", "", "schema name to compare on both sides")
	diffCmd.Flags().StringVar(&diffFromSchema, "from-schema", "", "source schema name")
	diffCmd.Flags().StringVar(&diffToSchema, "to-schema", "", "target schema name")
	diffCmd.Flags().StringVarP(&diffFormat, "format", "f", "table", "output format: table or json")
	diffCmd.Flags().BoolVarP(&diffQuiet, "quiet", "q", false, "suppress informational messages")

	diffCmd.RegisterFlagCompletionFunc("from", completeConnectionNames)
	diffCmd.RegisterFlagCompletionFunc("to", completeConnectionNames)
	diffCmd.RegisterFlagCompletionFunc("format", completeAuditFormats)
}

func resolveSchemaDiffFormat(value string) (output.Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "table":
		return output.FormatTable, nil
	case "json":
		return output.FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid --format %q (expected table or json)", value)
	}
}

func captureSchemaSnapshot(
	conn *dbmgr.Connection,
	schemaHint string,
	quiet bool,
	out *output.Writer,
) (*schemaSnapshot, error) {
	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, fmt.Errorf("cannot initialize database manager: %w", err)
	}

	var spinner *output.Spinner
	if !quiet {
		spinner = output.NewSpinner(fmt.Sprintf("Loading schema metadata from %s...", conn.Name))
		spinner.Start()
	}

	if err := mgr.Connect(conn); err != nil {
		if spinner != nil {
			spinner.StopWithError("Connection failed")
		}
		return nil, fmt.Errorf("cannot connect to database: %w", err)
	}
	defer mgr.Disconnect()

	schemaName, err := resolveSnapshotSchema(mgr, conn, schemaHint)
	if err != nil {
		if spinner != nil {
			spinner.StopWithError("Schema resolution failed")
		}
		return nil, err
	}

	storageUnits, err := mgr.GetStorageUnits(schemaName)
	if err != nil {
		if spinner != nil {
			spinner.StopWithError("Failed to load storage units")
		}
		return nil, fmt.Errorf("failed to load storage units: %w", err)
	}

	sort.Slice(storageUnits, func(i, j int) bool {
		return storageUnits[i].Name < storageUnits[j].Name
	})

	snapshot := &schemaSnapshot{
		Connection:   conn.Name,
		Type:         conn.Type,
		Schema:       schemaName,
		StorageUnits: make([]storageUnitSnapshot, 0, len(storageUnits)),
	}

	for _, storageUnit := range storageUnits {
		columns, err := mgr.GetColumns(schemaName, storageUnit.Name)
		if err != nil {
			if spinner != nil {
				spinner.StopWithError("Failed to load columns")
			}
			return nil, fmt.Errorf("failed to load columns for %s: %w", storageUnit.Name, err)
		}

		sort.Slice(columns, func(i, j int) bool {
			return columns[i].Name < columns[j].Name
		})

		columnSnapshots := make([]columnSnapshot, 0, len(columns))
		for _, column := range columns {
			columnSnapshots = append(columnSnapshots, snapshotColumn(column))
		}

		snapshot.StorageUnits = append(snapshot.StorageUnits, storageUnitSnapshot{
			Name:    storageUnit.Name,
			Kind:    storageUnitKind(storageUnit),
			Columns: columnSnapshots,
		})
	}

	if spinner != nil {
		spinner.StopWithSuccess("Schema metadata loaded")
	}
	if !quiet && snapshot.Schema != "" {
		out.Info("%s schema: %s", conn.Name, snapshot.Schema)
	}

	return snapshot, nil
}

func resolveSnapshotSchema(mgr *dbmgr.Manager, conn *dbmgr.Connection, explicitSchema string) (string, error) {
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

func snapshotColumn(column engine.Column) columnSnapshot {
	referencedTable := ""
	if column.ReferencedTable != nil {
		referencedTable = *column.ReferencedTable
	}
	referencedColumn := ""
	if column.ReferencedColumn != nil {
		referencedColumn = *column.ReferencedColumn
	}

	return columnSnapshot{
		Name:             column.Name,
		Type:             column.Type,
		IsNullable:       column.IsNullable,
		IsPrimary:        column.IsPrimary,
		IsAutoIncrement:  column.IsAutoIncrement,
		IsComputed:       column.IsComputed,
		IsForeignKey:     column.IsForeignKey,
		ReferencedTable:  referencedTable,
		ReferencedColumn: referencedColumn,
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

func buildSchemaDiffOutput(fromSnapshot, toSnapshot *schemaSnapshot) *schemaDiffOutput {
	result := &schemaDiffOutput{
		From: schemaReference{
			Connection: fromSnapshot.Connection,
			Type:       fromSnapshot.Type,
			Schema:     fromSnapshot.Schema,
		},
		To: schemaReference{
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

	var added []storageUnitDiff
	var removed []storageUnitDiff
	var changed []storageUnitDiff

	unitNames := make([]string, 0, len(fromUnits)+len(toUnits))
	seen := make(map[string]struct{}, len(fromUnits)+len(toUnits))
	for name := range fromUnits {
		unitNames = append(unitNames, name)
		seen[name] = struct{}{}
	}
	for name := range toUnits {
		if _, ok := seen[name]; ok {
			continue
		}
		unitNames = append(unitNames, name)
	}
	sort.Strings(unitNames)

	for _, name := range unitNames {
		fromUnit, hasFrom := fromUnits[name]
		toUnit, hasTo := toUnits[name]

		switch {
		case !hasFrom && hasTo:
			result.Summary.AddedStorageUnits++
			result.Summary.AddedColumns += len(toUnit.Columns)
			added = append(added, storageUnitDiff{
				Name:    name,
				Change:  "added",
				ToKind:  toUnit.Kind,
				Columns: buildAddedColumnDiffs(toUnit.Columns),
			})
		case hasFrom && !hasTo:
			result.Summary.RemovedStorageUnits++
			result.Summary.RemovedColumns += len(fromUnit.Columns)
			removed = append(removed, storageUnitDiff{
				Name:     name,
				Change:   "removed",
				FromKind: fromUnit.Kind,
				Columns:  buildRemovedColumnDiffs(fromUnit.Columns),
			})
		default:
			unitDiff := storageUnitDiff{
				Name:     name,
				Change:   "changed",
				FromKind: fromUnit.Kind,
				ToKind:   toUnit.Kind,
			}
			if fromUnit.Kind != toUnit.Kind {
				unitDiff.Differences = append(unitDiff.Differences, fieldDiff{
					Field: "kind",
					From:  fromUnit.Kind,
					To:    toUnit.Kind,
				})
			}

			columnDiffs, summary := diffColumns(fromUnit.Columns, toUnit.Columns)
			unitDiff.Columns = columnDiffs
			result.Summary.AddedColumns += summary.AddedColumns
			result.Summary.RemovedColumns += summary.RemovedColumns
			result.Summary.ChangedColumns += summary.ChangedColumns

			if len(unitDiff.Differences) == 0 && len(unitDiff.Columns) == 0 {
				continue
			}

			result.Summary.ChangedStorageUnits++
			changed = append(changed, unitDiff)
		}
	}

	result.StorageUnits = append(result.StorageUnits, added...)
	result.StorageUnits = append(result.StorageUnits, removed...)
	result.StorageUnits = append(result.StorageUnits, changed...)
	result.Summary.HasDifferences = len(result.StorageUnits) > 0

	return result
}

func buildAddedColumnDiffs(columns []columnSnapshot) []columnDiff {
	diffs := make([]columnDiff, 0, len(columns))
	for _, column := range columns {
		columnCopy := column
		diffs = append(diffs, columnDiff{
			Name:   column.Name,
			Change: "added",
			To:     &columnCopy,
		})
	}
	return diffs
}

func buildRemovedColumnDiffs(columns []columnSnapshot) []columnDiff {
	diffs := make([]columnDiff, 0, len(columns))
	for _, column := range columns {
		columnCopy := column
		diffs = append(diffs, columnDiff{
			Name:   column.Name,
			Change: "removed",
			From:   &columnCopy,
		})
	}
	return diffs
}

func diffColumns(fromColumns, toColumns []columnSnapshot) ([]columnDiff, schemaDiffSummary) {
	fromMap := make(map[string]columnSnapshot, len(fromColumns))
	for _, column := range fromColumns {
		fromMap[column.Name] = column
	}
	toMap := make(map[string]columnSnapshot, len(toColumns))
	for _, column := range toColumns {
		toMap[column.Name] = column
	}

	columnNames := make([]string, 0, len(fromMap)+len(toMap))
	seen := make(map[string]struct{}, len(fromMap)+len(toMap))
	for name := range fromMap {
		columnNames = append(columnNames, name)
		seen[name] = struct{}{}
	}
	for name := range toMap {
		if _, ok := seen[name]; ok {
			continue
		}
		columnNames = append(columnNames, name)
	}
	sort.Strings(columnNames)

	result := make([]columnDiff, 0)
	summary := schemaDiffSummary{}

	for _, name := range columnNames {
		fromColumn, hasFrom := fromMap[name]
		toColumn, hasTo := toMap[name]

		switch {
		case !hasFrom && hasTo:
			summary.AddedColumns++
			columnCopy := toColumn
			result = append(result, columnDiff{
				Name:   name,
				Change: "added",
				To:     &columnCopy,
			})
		case hasFrom && !hasTo:
			summary.RemovedColumns++
			columnCopy := fromColumn
			result = append(result, columnDiff{
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
			result = append(result, columnDiff{
				Name:        name,
				Change:      "changed",
				From:        &fromCopy,
				To:          &toCopy,
				Differences: differences,
			})
		}
	}

	return result, summary
}

func diffColumnFields(fromColumn, toColumn columnSnapshot) []fieldDiff {
	var diffs []fieldDiff

	appendIfChanged := func(field string, fromValue, toValue string) {
		if fromValue == toValue {
			return
		}
		diffs = append(diffs, fieldDiff{
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
	appendIfChanged("referencedTable", fromColumn.ReferencedTable, toColumn.ReferencedTable)
	appendIfChanged("referencedColumn", fromColumn.ReferencedColumn, toColumn.ReferencedColumn)
	appendIfChanged("length", formatOptionalInt(fromColumn.Length), formatOptionalInt(toColumn.Length))
	appendIfChanged("precision", formatOptionalInt(fromColumn.Precision), formatOptionalInt(toColumn.Precision))
	appendIfChanged("scale", formatOptionalInt(fromColumn.Scale), formatOptionalInt(toColumn.Scale))

	return diffs
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func printSchemaDiff(out io.Writer, result *schemaDiffOutput) {
	fmt.Fprintln(out, "Schema Diff")
	fmt.Fprintf(out, "  From: %s (%s)", result.From.Connection, result.From.Type)
	if result.From.Schema != "" {
		fmt.Fprintf(out, " schema=%s", result.From.Schema)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  To:   %s (%s)", result.To.Connection, result.To.Type)
	if result.To.Schema != "" {
		fmt.Fprintf(out, " schema=%s", result.To.Schema)
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Summary")
	fmt.Fprintf(out, "  Storage units: +%d -%d ~%d\n",
		result.Summary.AddedStorageUnits,
		result.Summary.RemovedStorageUnits,
		result.Summary.ChangedStorageUnits,
	)
	fmt.Fprintf(out, "  Columns:       +%d -%d ~%d\n",
		result.Summary.AddedColumns,
		result.Summary.RemovedColumns,
		result.Summary.ChangedColumns,
	)

	if !result.Summary.HasDifferences {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "No schema differences found.")
		return
	}

	printStorageUnitSection(out, "Added Storage Units", result.StorageUnits, "added")
	printStorageUnitSection(out, "Removed Storage Units", result.StorageUnits, "removed")
	printStorageUnitSection(out, "Changed Storage Units", result.StorageUnits, "changed")
}

func printStorageUnitSection(out io.Writer, title string, storageUnits []storageUnitDiff, change string) {
	sectionUnits := make([]storageUnitDiff, 0)
	for _, storageUnit := range storageUnits {
		if storageUnit.Change == change {
			sectionUnits = append(sectionUnits, storageUnit)
		}
	}
	if len(sectionUnits) == 0 {
		return
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, title)
	for _, storageUnit := range sectionUnits {
		switch storageUnit.Change {
		case "added":
			fmt.Fprintf(out, "  + %s", storageUnit.Name)
			if storageUnit.ToKind != "" {
				fmt.Fprintf(out, " (%s)", storageUnit.ToKind)
			}
			fmt.Fprintf(out, " [%d columns]\n", len(storageUnit.Columns))
		case "removed":
			fmt.Fprintf(out, "  - %s", storageUnit.Name)
			if storageUnit.FromKind != "" {
				fmt.Fprintf(out, " (%s)", storageUnit.FromKind)
			}
			fmt.Fprintf(out, " [%d columns]\n", len(storageUnit.Columns))
		case "changed":
			fmt.Fprintf(out, "  ~ %s\n", storageUnit.Name)
			for _, difference := range storageUnit.Differences {
				fmt.Fprintf(out, "      %s: %s -> %s\n", difference.Field, printableValue(difference.From), printableValue(difference.To))
			}
			for _, column := range storageUnit.Columns {
				printColumnDiff(out, column)
			}
		}
	}
}

func printColumnDiff(out io.Writer, column columnDiff) {
	switch column.Change {
	case "added":
		fmt.Fprintf(out, "      + %s %s\n", column.Name, describeColumnSnapshot(column.To))
	case "removed":
		fmt.Fprintf(out, "      - %s %s\n", column.Name, describeColumnSnapshot(column.From))
	case "changed":
		fmt.Fprintf(out, "      ~ %s\n", column.Name)
		for _, difference := range column.Differences {
			fmt.Fprintf(out, "          %s: %s -> %s\n", difference.Field, printableValue(difference.From), printableValue(difference.To))
		}
	}
}

func describeColumnSnapshot(column *columnSnapshot) string {
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
	if column.IsAutoIncrement {
		parts = append(parts, "auto increment")
	}
	if column.IsComputed {
		parts = append(parts, "computed")
	}
	if column.IsForeignKey {
		target := column.ReferencedTable
		if column.ReferencedColumn != "" {
			target = target + "." + column.ReferencedColumn
		}
		if target != "" {
			parts = append(parts, "fk="+target)
		} else {
			parts = append(parts, "foreign key")
		}
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

func printableValue(value string) string {
	if value == "" {
		return "<empty>"
	}
	return value
}
