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

package postgres

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestCockroachDBSchemaQueryExcludesSystemSchemas(t *testing.T) {
	plugin := NewCockroachDBPlugin().PluginFunctions.(*CockroachDBPlugin)
	query := plugin.GetAllSchemasQuery()

	for _, systemSchema := range []string{"information_schema", "pg_catalog", "crdb_internal", "pg_extension"} {
		if !strings.Contains(query, systemSchema) {
			t.Fatalf("expected CockroachDB schema query to exclude %q, got:\n%s", systemSchema, query)
		}
	}
}

func TestCockroachDBBuildColumnFromInformationSchema(t *testing.T) {
	plugin := NewCockroachDBPlugin().PluginFunctions.(*CockroachDBPlugin)

	column := plugin.buildCockroachDBColumn(cockroachDBColumnInfo{
		columnName:             "username",
		dataType:               "character varying",
		isNullable:             "NO",
		characterMaximumLength: sql.NullInt64{Int64: 50, Valid: true},
	})

	if column.Name != "username" {
		t.Fatalf("expected username column, got %q", column.Name)
	}
	if column.Type != "CHARACTER VARYING(50)" {
		t.Fatalf("expected CHARACTER VARYING(50), got %q", column.Type)
	}
	if column.IsNullable {
		t.Fatalf("expected username to be non-nullable")
	}
	if column.Length == nil || *column.Length != 50 {
		t.Fatalf("expected length 50, got %#v", column.Length)
	}
}

func TestCockroachDBBuildColumnMarksPrimaryForeignAndAutoIncrement(t *testing.T) {
	plugin := NewCockroachDBPlugin().PluginFunctions.(*CockroachDBPlugin)

	column := plugin.buildCockroachDBColumn(cockroachDBColumnInfo{
		columnName:       "user_id",
		dataType:         "bigint",
		columnDefault:    sql.NullString{String: "nextval('test_schema.users_id_seq'::REGCLASS)", Valid: true},
		isNullable:       "NO",
		numericPrecision: sql.NullInt64{Int64: 64, Valid: true},
		numericScale:     sql.NullInt64{Int64: 0, Valid: true},
		isPrimary:        true,
		referencedTable:  sql.NullString{String: "users", Valid: true},
		referencedColumn: sql.NullString{String: "id", Valid: true},
	})

	if column.Type != "BIGINT" {
		t.Fatalf("expected BIGINT, got %q", column.Type)
	}
	if !column.IsPrimary {
		t.Fatalf("expected primary column")
	}
	if !column.IsAutoIncrement {
		t.Fatalf("expected auto-increment column")
	}
	if !column.IsForeignKey {
		t.Fatalf("expected foreign key column")
	}
	if column.ReferencedTable == nil || *column.ReferencedTable != "users" {
		t.Fatalf("expected referenced table users, got %#v", column.ReferencedTable)
	}
	if column.ReferencedColumn == nil || *column.ReferencedColumn != "id" {
		t.Fatalf("expected referenced column id, got %#v", column.ReferencedColumn)
	}
}

func TestCockroachDBBuildColumnMarksComputed(t *testing.T) {
	plugin := NewCockroachDBPlugin().PluginFunctions.(*CockroachDBPlugin)

	column := plugin.buildCockroachDBColumn(cockroachDBColumnInfo{
		columnName:  "full_name",
		dataType:    "text",
		isNullable:  "YES",
		isGenerated: "ALWAYS",
	})

	if !column.IsComputed {
		t.Fatalf("expected computed column")
	}
}

func TestCockroachDBMarkGeneratedColumnsIsNoOp(t *testing.T) {
	plugin := NewCockroachDBPlugin().PluginFunctions.(*CockroachDBPlugin)
	columns := []engine.Column{{Name: "id"}}

	if err := plugin.MarkGeneratedColumns(nil, "test_schema", "users", columns); err != nil {
		t.Fatalf("MarkGeneratedColumns returned error: %v", err)
	}
}
