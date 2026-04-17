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

package schemadiff

import (
	"strings"
	"testing"
)

func TestBuildResult_TracksStructuralConstraintAndRelationshipChanges(t *testing.T) {
	fromSnapshot := &snapshot{
		Connection: "from",
		Type:       "Postgres",
		Schema:     "public",
		StorageUnits: []storageUnitSnapshot{
			{
				Name: "legacy",
				Kind: "BASE TABLE",
				Columns: []ColumnState{
					{Name: "id", Type: "integer", IsPrimary: true},
					{Name: "note", Type: "text", IsNullable: true},
				},
			},
			{
				Name: "users",
				Kind: "BASE TABLE",
				Columns: []ColumnState{
					{Name: "id", Type: "integer", IsPrimary: true},
					{Name: "email", Type: "text", IsNullable: true},
					{Name: "name", Type: "text", IsNullable: true},
				},
				Relationships: []RelationshipState{
					{TargetStorageUnit: "profiles", RelationshipType: "OneToMany", SourceColumn: "user_id", TargetColumn: "id"},
					{TargetStorageUnit: "sessions", RelationshipType: "OneToMany", SourceColumn: "user_id", TargetColumn: "id"},
				},
			},
		},
	}

	toSnapshot := &snapshot{
		Connection: "to",
		Type:       "Postgres",
		Schema:     "public",
		StorageUnits: []storageUnitSnapshot{
			{
				Name: "audit_log",
				Kind: "VIEW",
				Columns: []ColumnState{
					{Name: "id", Type: "integer", IsPrimary: true},
				},
			},
			{
				Name: "users",
				Kind: "VIEW",
				Columns: []ColumnState{
					{Name: "id", Type: "integer", IsPrimary: true},
					{Name: "email", Type: "text", IsNullable: false, IsUnique: true, DefaultValue: "none@example.com"},
					{Name: "status", Type: "text", IsNullable: false, CheckValues: []string{"active", "pending"}},
				},
				Relationships: []RelationshipState{
					{TargetStorageUnit: "profiles", RelationshipType: "ManyToOne", SourceColumn: "user_id", TargetColumn: "id"},
					{TargetStorageUnit: "orders", RelationshipType: "OneToMany", SourceColumn: "user_id", TargetColumn: "id"},
				},
			},
		},
	}

	result := buildResult(fromSnapshot, toSnapshot)

	if !result.Summary.HasDifferences {
		t.Fatal("expected differences to be detected")
	}
	if result.Summary.AddedStorageUnits != 1 {
		t.Fatalf("expected 1 added storage unit, got %d", result.Summary.AddedStorageUnits)
	}
	if result.Summary.RemovedStorageUnits != 1 {
		t.Fatalf("expected 1 removed storage unit, got %d", result.Summary.RemovedStorageUnits)
	}
	if result.Summary.ChangedStorageUnits != 1 {
		t.Fatalf("expected 1 changed storage unit, got %d", result.Summary.ChangedStorageUnits)
	}
	if result.Summary.AddedColumns != 2 {
		t.Fatalf("expected 2 added columns, got %d", result.Summary.AddedColumns)
	}
	if result.Summary.RemovedColumns != 3 {
		t.Fatalf("expected 3 removed columns, got %d", result.Summary.RemovedColumns)
	}
	if result.Summary.ChangedColumns != 1 {
		t.Fatalf("expected 1 changed column, got %d", result.Summary.ChangedColumns)
	}
	if result.Summary.AddedRelationships != 1 {
		t.Fatalf("expected 1 added relationship, got %d", result.Summary.AddedRelationships)
	}
	if result.Summary.RemovedRelationships != 1 {
		t.Fatalf("expected 1 removed relationship, got %d", result.Summary.RemovedRelationships)
	}
	if result.Summary.ChangedRelationships != 1 {
		t.Fatalf("expected 1 changed relationship, got %d", result.Summary.ChangedRelationships)
	}

	if len(result.StorageUnits) != 3 {
		t.Fatalf("expected 3 storage unit changes, got %d", len(result.StorageUnits))
	}

	changed := result.StorageUnits[2]
	if changed.Name != "users" {
		t.Fatalf("expected users change, got %s", changed.Name)
	}
	if len(changed.Differences) != 1 || changed.Differences[0].Field != "kind" {
		t.Fatalf("expected kind change for users, got %#v", changed.Differences)
	}
	if len(changed.Columns) != 3 {
		t.Fatalf("expected 3 column changes for users, got %d", len(changed.Columns))
	}
	if changed.Columns[0].Name != "email" || changed.Columns[0].Change != "changed" {
		t.Fatalf("expected changed email column, got %#v", changed.Columns[0])
	}
	if len(changed.Columns[0].Differences) != 3 {
		t.Fatalf("expected 3 email field changes, got %#v", changed.Columns[0].Differences)
	}
	if changed.Columns[0].Differences[0].Field != "nullable" {
		t.Fatalf("expected nullable diff first, got %#v", changed.Columns[0].Differences)
	}

	if len(changed.Relationships) != 3 {
		t.Fatalf("expected 3 relationship changes for users, got %d", len(changed.Relationships))
	}
	if changed.Relationships[0].TargetStorageUnit != "orders" || changed.Relationships[0].Change != "added" {
		t.Fatalf("expected orders relationship to be added, got %#v", changed.Relationships[0])
	}
	if changed.Relationships[1].TargetStorageUnit != "profiles" || changed.Relationships[1].Change != "changed" {
		t.Fatalf("expected profiles relationship to be changed, got %#v", changed.Relationships[1])
	}
	if len(changed.Relationships[1].Differences) != 1 || changed.Relationships[1].Differences[0].Field != "relationshipType" {
		t.Fatalf("expected relationshipType change, got %#v", changed.Relationships[1].Differences)
	}
	if changed.Relationships[2].TargetStorageUnit != "sessions" || changed.Relationships[2].Change != "removed" {
		t.Fatalf("expected sessions relationship to be removed, got %#v", changed.Relationships[2])
	}
}

func TestRenderText_NoDifferences(t *testing.T) {
	text := RenderText(&Result{
		From:    SchemaReference{Connection: "a", Type: "Sqlite3"},
		To:      SchemaReference{Connection: "b", Type: "Sqlite3"},
		Summary: Summary{},
	})

	if text == "" {
		t.Fatal("expected rendered text")
	}
	if !containsAll(text, "Schema Diff", "Summary", "No schema differences found.") {
		t.Fatalf("unexpected rendered text: %q", text)
	}
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
