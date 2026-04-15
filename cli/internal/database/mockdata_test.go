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
	"strings"
	"testing"

	coremockdata "github.com/clidey/whodb/core/src/mockdata"
)

func TestAnalyzeMockDataDependencies_UnsupportedDatabase(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.currentConnection = &Connection{
		Name: "cache",
		Type: "redis",
		Host: "localhost",
	}

	_, err = mgr.AnalyzeMockDataDependencies("", "users", 10, 0)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected unsupported database error, got %v", err)
	}
}

func TestAnalyzeMockDataDependencies_RejectsLargeRowCount(t *testing.T) {
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

	_, err = mgr.AnalyzeMockDataDependencies("public", "users", coremockdata.GetMockDataGenerationMaxRowCount()+1, 0)
	if err == nil || !strings.Contains(err.Error(), "maximum limit") {
		t.Fatalf("expected row limit error, got %v", err)
	}
}

func TestGenerateMockData_UnsupportedDatabase(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.currentConnection = &Connection{
		Name: "cache",
		Type: "redis",
		Host: "localhost",
	}

	_, err = mgr.GenerateMockData("", "users", 10, false, 0)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected unsupported database error, got %v", err)
	}
}
