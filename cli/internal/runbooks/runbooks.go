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

// Package runbooks provides built-in database workflows for the WhoDB CLI.
package runbooks

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/doctor"
	"github.com/clidey/whodb/cli/internal/schemadiff"
)

const (
	stepStatusOK      = "ok"
	stepStatusPlanned = "planned"
	stepStatusError   = "error"
)

// Argument describes one runbook argument.
type Argument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
}

// Step describes one planned runbook step.
type Step struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

// Definition describes a built-in runbook.
type Definition struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Arguments   []Argument `json:"arguments,omitempty"`
	Steps       []Step     `json:"steps"`
}

// Options configures a runbook execution.
type Options struct {
	Connection string
	Schema     string
	From       string
	To         string
	FromSchema string
	ToSchema   string
	DryRun     bool
}

// Result is the output produced by a runbook execution.
type Result struct {
	Name   string       `json:"name"`
	DryRun bool         `json:"dry_run,omitempty"`
	Steps  []StepResult `json:"steps"`
	Data   any          `json:"data,omitempty"`
}

// StepResult reports the status of one runbook step.
type StepResult struct {
	Name       string `json:"name"`
	Command    string `json:"command"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	DurationMS int64  `json:"duration_ms,omitempty"`
}

// SchemaAuditData is the data payload for the schema-audit runbook.
type SchemaAuditData struct {
	Connection   string                 `json:"connection"`
	Schema       string                 `json:"schema"`
	StorageUnits int                    `json:"storage_units"`
	Audit        []*dbmgr.TableAudit    `json:"audit"`
	Summary      SchemaAuditSummaryData `json:"summary"`
}

// SchemaAuditSummaryData summarizes schema-audit results.
type SchemaAuditSummaryData struct {
	TablesScanned int `json:"tables_scanned"`
	IssuesFound   int `json:"issues_found"`
}

// List returns the built-in runbook definitions.
func List() []Definition {
	definitions := []Definition{
		connectionDoctorDefinition(),
		schemaAuditDefinition(),
		schemaDiffDefinition(),
	}
	slices.SortFunc(definitions, func(a, b Definition) int {
		return strings.Compare(a.Name, b.Name)
	})
	return definitions
}

// Describe returns one built-in runbook definition by name.
func Describe(name string) (Definition, bool) {
	normalized := strings.TrimSpace(name)
	for _, definition := range List() {
		if definition.Name == normalized {
			return definition, true
		}
	}
	return Definition{}, false
}

// Run executes one built-in runbook.
func Run(ctx context.Context, name string, opts Options) (Result, error) {
	definition, ok := Describe(name)
	if !ok {
		return Result{}, fmt.Errorf("runbook %q not found", name)
	}

	if opts.DryRun {
		return plannedResult(definition), nil
	}

	switch definition.Name {
	case "connection-doctor":
		return runConnectionDoctor(ctx, opts)
	case "schema-audit":
		return runSchemaAudit(ctx, opts)
	case "schema-diff":
		return runSchemaDiff(opts)
	default:
		return Result{}, fmt.Errorf("runbook %q not implemented", definition.Name)
	}
}

func connectionDoctorDefinition() Definition {
	return Definition{
		Name:        "connection-doctor",
		Description: "Resolve and test one connection, then inspect basic schema and storage-unit metadata.",
		Arguments: []Argument{
			{Name: "connection", Description: "Connection name. Uses the first available connection when omitted."},
			{Name: "schema", Description: "Schema override for metadata checks."},
		},
		Steps: []Step{
			{Name: "doctor", Command: "doctor --connection {{connection}}", Description: "Run connection diagnostics and metadata checks."},
		},
	}
}

func schemaAuditDefinition() Definition {
	return Definition{
		Name:        "schema-audit",
		Description: "Inspect storage units and run data-quality checks for one schema.",
		Arguments: []Argument{
			{Name: "connection", Description: "Connection name.", Required: true},
			{Name: "schema", Description: "Schema override for metadata checks."},
		},
		Steps: []Step{
			{Name: "connect", Command: "connections test {{connection}}", Description: "Resolve and connect to the database."},
			{Name: "storage-units", Command: "tables --connection {{connection}} --schema {{schema}}", Description: "Load tables or storage units for the resolved schema."},
			{Name: "audit", Command: "audit --connection {{connection}} --schema {{schema}}", Description: "Run data-quality checks."},
		},
	}
}

func schemaDiffDefinition() Definition {
	return Definition{
		Name:        "schema-diff",
		Description: "Compare schema metadata between two connections.",
		Arguments: []Argument{
			{Name: "from", Description: "Source connection name.", Required: true},
			{Name: "to", Description: "Target connection name.", Required: true},
			{Name: "from_schema", Description: "Source schema override."},
			{Name: "to_schema", Description: "Target schema override."},
		},
		Steps: []Step{
			{Name: "diff", Command: "diff --from {{from}} --to {{to}}", Description: "Compare schema metadata."},
		},
	}
}

func plannedResult(definition Definition) Result {
	steps := make([]StepResult, len(definition.Steps))
	for i, step := range definition.Steps {
		steps[i] = StepResult{
			Name:    step.Name,
			Command: step.Command,
			Status:  stepStatusPlanned,
			Message: step.Description,
		}
	}
	return Result{Name: definition.Name, DryRun: true, Steps: steps}
}

func runConnectionDoctor(ctx context.Context, opts Options) (Result, error) {
	start := time.Now()
	report, err := doctor.Run(ctx, doctor.Options{
		Connection: opts.Connection,
		Schema:     opts.Schema,
	})
	step := StepResult{
		Name:       "doctor",
		Command:    "doctor",
		Status:     stepStatusOK,
		DurationMS: elapsed(start),
	}
	if err != nil {
		step.Status = stepStatusError
		step.Message = err.Error()
	}
	return Result{Name: "connection-doctor", Steps: []StepResult{step}, Data: report}, err
}

func runSchemaAudit(ctx context.Context, opts Options) (Result, error) {
	_ = ctx
	if strings.TrimSpace(opts.Connection) == "" {
		return Result{Name: "schema-audit"}, fmt.Errorf("--connection is required for schema-audit")
	}

	result := Result{Name: "schema-audit"}
	mgr, err := dbmgr.NewManager()
	if err != nil {
		return result, err
	}

	connectStart := time.Now()
	conn, _, err := mgr.ResolveConnection(opts.Connection)
	if err != nil {
		result.Steps = append(result.Steps, failedStep("connect", "connections test", connectStart, err))
		return result, err
	}
	if err := mgr.Connect(conn); err != nil {
		result.Steps = append(result.Steps, failedStep("connect", "connections test", connectStart, err))
		return result, err
	}
	result.Steps = append(result.Steps, okStep("connect", "connections test", connectStart))
	defer mgr.Disconnect()

	schemaStart := time.Now()
	schema, err := mgr.ResolveSnapshotSchema(conn, opts.Schema)
	if err != nil {
		result.Steps = append(result.Steps, failedStep("storage-units", "tables", schemaStart, err))
		return result, err
	}
	tables, err := mgr.GetStorageUnits(schema)
	if err != nil {
		result.Steps = append(result.Steps, failedStep("storage-units", "tables", schemaStart, err))
		return result, err
	}
	result.Steps = append(result.Steps, okStep("storage-units", "tables", schemaStart))

	auditStart := time.Now()
	auditResults, err := mgr.AuditSchema(schema, dbmgr.DefaultAuditConfig())
	if err != nil {
		result.Steps = append(result.Steps, failedStep("audit", "audit", auditStart, err))
		return result, err
	}
	result.Steps = append(result.Steps, okStep("audit", "audit", auditStart))
	result.Data = SchemaAuditData{
		Connection:   opts.Connection,
		Schema:       schema,
		StorageUnits: len(tables),
		Audit:        auditResults,
		Summary:      summarizeAudit(auditResults),
	}
	return result, nil
}

func runSchemaDiff(opts Options) (Result, error) {
	if strings.TrimSpace(opts.From) == "" {
		return Result{Name: "schema-diff"}, fmt.Errorf("--from is required for schema-diff")
	}
	if strings.TrimSpace(opts.To) == "" {
		return Result{Name: "schema-diff"}, fmt.Errorf("--to is required for schema-diff")
	}

	mgr, err := dbmgr.NewManager()
	if err != nil {
		return Result{Name: "schema-diff"}, err
	}
	fromConn, _, err := mgr.ResolveConnection(opts.From)
	if err != nil {
		return Result{Name: "schema-diff"}, err
	}
	toConn, _, err := mgr.ResolveConnection(opts.To)
	if err != nil {
		return Result{Name: "schema-diff"}, err
	}

	start := time.Now()
	diff, err := schemadiff.CompareConnections(fromConn, toConn, opts.FromSchema, opts.ToSchema)
	step := okStep("diff", "diff", start)
	if err != nil {
		step = failedStep("diff", "diff", start, err)
	}
	return Result{
		Name:  "schema-diff",
		Steps: []StepResult{step},
		Data:  diff,
	}, err
}

func summarizeAudit(results []*dbmgr.TableAudit) SchemaAuditSummaryData {
	issues := 0
	for _, result := range results {
		issues += len(result.Issues)
	}
	return SchemaAuditSummaryData{
		TablesScanned: len(results),
		IssuesFound:   issues,
	}
}

func okStep(name, command string, start time.Time) StepResult {
	return StepResult{
		Name:       name,
		Command:    command,
		Status:     stepStatusOK,
		DurationMS: elapsed(start),
	}
}

func failedStep(name, command string, start time.Time, err error) StepResult {
	return StepResult{
		Name:       name,
		Command:    command,
		Status:     stepStatusError,
		Message:    err.Error(),
		DurationMS: elapsed(start),
	}
}

func elapsed(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
