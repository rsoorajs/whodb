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
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
)

func TestBuildSchemaDiffOutput_TracksStructuralChanges(t *testing.T) {
	fromSnapshot := &schemaSnapshot{
		Connection: "from",
		Type:       "Postgres",
		Schema:     "public",
		StorageUnits: []storageUnitSnapshot{
			{
				Name: "legacy",
				Kind: "BASE TABLE",
				Columns: []columnSnapshot{
					{Name: "id", Type: "integer", IsPrimary: true},
					{Name: "note", Type: "text", IsNullable: true},
				},
			},
			{
				Name: "users",
				Kind: "BASE TABLE",
				Columns: []columnSnapshot{
					{Name: "id", Type: "integer", IsPrimary: true},
					{Name: "email", Type: "text", IsNullable: true},
					{Name: "name", Type: "text", IsNullable: true},
				},
			},
		},
	}
	toSnapshot := &schemaSnapshot{
		Connection: "to",
		Type:       "Postgres",
		Schema:     "public",
		StorageUnits: []storageUnitSnapshot{
			{
				Name: "audit_log",
				Kind: "VIEW",
				Columns: []columnSnapshot{
					{Name: "id", Type: "integer", IsPrimary: true},
				},
			},
			{
				Name: "users",
				Kind: "VIEW",
				Columns: []columnSnapshot{
					{Name: "id", Type: "integer", IsPrimary: true},
					{Name: "email", Type: "text", IsNullable: false},
					{Name: "status", Type: "text", IsNullable: false},
				},
			},
		},
	}

	result := buildSchemaDiffOutput(fromSnapshot, toSnapshot)

	if !result.Summary.HasDifferences {
		t.Fatal("Expected differences to be detected")
	}
	if result.Summary.AddedStorageUnits != 1 {
		t.Errorf("Expected 1 added storage unit, got %d", result.Summary.AddedStorageUnits)
	}
	if result.Summary.RemovedStorageUnits != 1 {
		t.Errorf("Expected 1 removed storage unit, got %d", result.Summary.RemovedStorageUnits)
	}
	if result.Summary.ChangedStorageUnits != 1 {
		t.Errorf("Expected 1 changed storage unit, got %d", result.Summary.ChangedStorageUnits)
	}
	if result.Summary.AddedColumns != 2 {
		t.Errorf("Expected 2 added columns, got %d", result.Summary.AddedColumns)
	}
	if result.Summary.RemovedColumns != 3 {
		t.Errorf("Expected 3 removed columns, got %d", result.Summary.RemovedColumns)
	}
	if result.Summary.ChangedColumns != 1 {
		t.Errorf("Expected 1 changed column, got %d", result.Summary.ChangedColumns)
	}
	if len(result.StorageUnits) != 3 {
		t.Fatalf("Expected 3 storage unit diffs, got %d", len(result.StorageUnits))
	}

	changed := result.StorageUnits[2]
	if changed.Name != "users" {
		t.Fatalf("Expected users diff, got %s", changed.Name)
	}
	if len(changed.Differences) != 1 || changed.Differences[0].Field != "kind" {
		t.Fatalf("Expected kind change for users, got %#v", changed.Differences)
	}
	if len(changed.Columns) != 3 {
		t.Fatalf("Expected 3 column diffs for users, got %d", len(changed.Columns))
	}
	if changed.Columns[0].Name != "email" || changed.Columns[0].Change != "changed" {
		t.Fatalf("Expected changed email column diff, got %#v", changed.Columns[0])
	}
	if len(changed.Columns[0].Differences) != 1 || changed.Columns[0].Differences[0].Field != "nullable" {
		t.Fatalf("Expected nullable change for email, got %#v", changed.Columns[0].Differences)
	}
}

func TestResolveSnapshotSchema_UsesConnectionDatabaseForDatabaseScopedTypes(t *testing.T) {
	schema, err := resolveSnapshotSchema(nil, &config.Connection{
		Type:     "MySQL",
		Database: "test_db",
	}, "")
	if err != nil {
		t.Fatalf("resolveSnapshotSchema returned error: %v", err)
	}
	if schema != "test_db" {
		t.Fatalf("expected test_db, got %q", schema)
	}
}

func TestResolveSnapshotSchema_ExplicitSchemaWins(t *testing.T) {
	schema, err := resolveSnapshotSchema(nil, &config.Connection{
		Type:     "MySQL",
		Database: "test_db",
	}, "reporting")
	if err != nil {
		t.Fatalf("resolveSnapshotSchema returned error: %v", err)
	}
	if schema != "reporting" {
		t.Fatalf("expected reporting, got %q", schema)
	}
}
