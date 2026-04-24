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
	"errors"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/core/src/engine"
)

func TestParseRowPayload_RequiresJSONObject(t *testing.T) {
	_, err := parseRowPayload(`["not","an","object"]`)
	if err == nil || !strings.Contains(err.Error(), "JSON object") {
		t.Fatalf("expected JSON object error, got %v", err)
	}
}

func TestBuildRowRecords_RejectsDatabaseManagedColumns(t *testing.T) {
	_, err := buildRowRecords(map[string]any{"id": 1}, []engine.Column{
		{Name: "id", Type: "integer", IsAutoIncrement: true},
	})
	if err == nil || !strings.Contains(err.Error(), "database-managed") {
		t.Fatalf("expected database-managed column error, got %v", err)
	}
}

func TestAddRowFromJSON_ReadOnlyBlocks(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.currentConnection = &Connection{
		Name: "test",
		Type: "postgres",
		Host: "localhost",
	}
	mgr.config.SetReadOnly(true)

	err = mgr.AddRowFromJSON("public", "users", `{"name":"alice"}`)
	if !errors.Is(err, ErrReadOnly) {
		t.Fatalf("expected ErrReadOnly, got %v", err)
	}
}

func TestDeleteRow_ReadOnlyBlocks(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.currentConnection = &Connection{
		Name: "test",
		Type: "postgres",
		Host: "localhost",
	}
	mgr.config.SetReadOnly(true)

	err = mgr.DeleteRow("public", "users", map[string]string{"id": "1"})
	if !errors.Is(err, ErrReadOnly) {
		t.Fatalf("expected ErrReadOnly, got %v", err)
	}
}

func TestAddRowFromJSON_RejectsViewTargets(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.currentConnection = &Connection{
		Name:     "db",
		Type:     "postgres",
		Host:     "localhost",
		Database: "app",
	}
	mgr.cache.SetTables("public", []engine.StorageUnit{
		{Name: "order_summary", Attributes: []engine.Record{{Key: "Type", Value: "VIEW"}}},
	})

	err = mgr.AddRowFromJSON("public", "order_summary", `{"name":"alice"}`)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "view") {
		t.Fatalf("expected view target error, got %v", err)
	}
}

func TestDeleteRow_RejectsViewTargets(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.currentConnection = &Connection{
		Name: "db",
		Type: "postgres",
		Host: "localhost",
	}
	mgr.cache.SetTables("public", []engine.StorageUnit{
		{Name: "order_summary", Attributes: []engine.Record{{Key: "Type", Value: "VIEW"}}},
	})

	err = mgr.DeleteRow("public", "order_summary", map[string]string{"id": "1"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "view") {
		t.Fatalf("expected view target error, got %v", err)
	}
}

func TestUpdateRow_ReadOnlyBlocks(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.currentConnection = &Connection{
		Name: "test",
		Type: "postgres",
		Host: "localhost",
	}
	mgr.config.SetReadOnly(true)

	err = mgr.UpdateRow("public", "users", map[string]string{"id": "1"}, map[string]string{"name": "alice"})
	if !errors.Is(err, ErrReadOnly) {
		t.Fatalf("expected ErrReadOnly, got %v", err)
	}
}

func TestUpdateRow_RejectsViewTargets(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.currentConnection = &Connection{
		Name: "db",
		Type: "postgres",
		Host: "localhost",
	}
	mgr.cache.SetTables("public", []engine.StorageUnit{
		{Name: "order_summary", Attributes: []engine.Record{{Key: "Type", Value: "VIEW"}}},
	})

	err = mgr.UpdateRow("public", "order_summary", map[string]string{"id": "1"}, map[string]string{"name": "alice"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "view") {
		t.Fatalf("expected view target error, got %v", err)
	}
}

func TestUpdateRow_RejectsPrimaryKeyEdits(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.currentConnection = &Connection{
		Name: "db",
		Type: "postgres",
		Host: "localhost",
	}
	mgr.cache.SetTables("public", []engine.StorageUnit{
		{Name: "users", Attributes: []engine.Record{{Key: "Type", Value: "TABLE"}}},
	})
	mgr.cache.SetColumns("public", "users", []engine.Column{
		{Name: "id", Type: "integer", IsPrimary: true},
		{Name: "name", Type: "text"},
	})

	err = mgr.UpdateRow("public", "users", map[string]string{"id": "1", "name": "alice"}, map[string]string{"id": "2", "name": "alice"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "primary key") {
		t.Fatalf("expected primary key error, got %v", err)
	}
}

func TestRowCRUDIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)
	tempDir := t.TempDir()

	dbPath := tempDir + "/crud.db"

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := &config.Connection{
		Name:     "test-sqlite",
		Type:     "Sqlite",
		Host:     dbPath,
		Database: dbPath,
	}

	err = mgr.Connect(conn)
	if err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", err)
	}

	_, err = mgr.ExecuteQuery("CREATE TABLE IF NOT EXISTS test_users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatalf("Create table failed: %v", err)
	}

	err = mgr.AddRowFromJSON("", "test_users", `{"name":"alice","email":"a@b.com"}`)
	if err != nil {
		t.Fatalf("AddRowFromJSON failed: %v", err)
	}

	rows, err := mgr.GetRows("", "test_users", nil, 50, 0)
	if err != nil {
		t.Fatalf("GetRows failed: %v", err)
	}
	if len(rows.Rows) != 1 {
		t.Fatalf("expected 1 row after insert, got %d", len(rows.Rows))
	}

	values := map[string]string{}
	for idx, column := range rows.Columns {
		if idx < len(rows.Rows[0]) {
			values[column.Name] = rows.Rows[0][idx]
		}
	}

	err = mgr.UpdateRow("", "test_users", values, map[string]string{"name": "carol", "email": "c@b.com"})
	if err != nil {
		t.Fatalf("UpdateRow failed: %v", err)
	}

	rows, err = mgr.GetRows("", "test_users", nil, 50, 0)
	if err != nil {
		t.Fatalf("GetRows failed after update: %v", err)
	}
	if len(rows.Rows) != 1 {
		t.Fatalf("expected 1 row after update, got %d", len(rows.Rows))
	}
	if rows.Rows[0][1] != "carol" || rows.Rows[0][2] != "c@b.com" {
		t.Fatalf("expected updated row values, got %+v", rows.Rows[0])
	}

	values = map[string]string{}
	for idx, column := range rows.Columns {
		if idx < len(rows.Rows[0]) {
			values[column.Name] = rows.Rows[0][idx]
		}
	}

	err = mgr.DeleteRow("", "test_users", values)
	if err != nil {
		t.Fatalf("DeleteRow failed: %v", err)
	}

	rows, err = mgr.GetRows("", "test_users", nil, 50, 0)
	if err != nil {
		t.Fatalf("GetRows failed after delete: %v", err)
	}
	if len(rows.Rows) != 0 {
		t.Fatalf("expected 0 rows after delete, got %d", len(rows.Rows))
	}
}
