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
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	_ "github.com/clidey/whodb/core/src/sources/database"
)

func TestNewQuestDBPlugin(t *testing.T) {
	pluginDef := NewQuestDBPlugin()
	if pluginDef.Type != engine.DatabaseType_QuestDB {
		t.Fatalf("expected type %q, got %q", engine.DatabaseType_QuestDB, pluginDef.Type)
	}

	plugin, ok := pluginDef.PluginFunctions.(*QuestDBPlugin)
	if !ok {
		t.Fatalf("unexpected QuestDB plugin type %T", pluginDef.PluginFunctions)
	}
	if plugin.GormPluginFunctions != plugin {
		t.Fatal("expected QuestDB Gorm plugin hooks to point at the QuestDB wrapper")
	}
}

func TestQuestDBOverridesPostgresCatalogQueries(t *testing.T) {
	plugin := NewQuestDBPlugin().PluginFunctions.(*QuestDBPlugin)

	tableInfoQuery := plugin.GetTableInfoQuery()
	if strings.Contains(tableInfoQuery, "pg_total_relation_size") {
		t.Fatalf("expected QuestDB table info query to avoid pg_total_relation_size, got:\n%s", tableInfoQuery)
	}
	if !strings.Contains(tableInfoQuery, "($1 = '' OR t.table_schema = $1)") {
		t.Fatalf("expected QuestDB table info query to tolerate empty schema, got:\n%s", tableInfoQuery)
	}

	existsQuery := plugin.GetStorageUnitExistsQuery()
	if !strings.Contains(existsQuery, "($1 = '' OR table_schema = $1)") {
		t.Fatalf("expected QuestDB storage-unit exists query to tolerate empty schema, got:\n%s", existsQuery)
	}
	if strings.Contains(strings.ToUpper(existsQuery), "EXISTS(") {
		t.Fatalf("expected QuestDB storage-unit exists query to avoid EXISTS(), got:\n%s", existsQuery)
	}
	if strings.Contains(existsQuery, "COUNT(*)") {
		t.Fatalf("expected QuestDB storage-unit exists query to avoid COUNT(*), got:\n%s", existsQuery)
	}
	if !strings.Contains(existsQuery, "COUNT(1)") {
		t.Fatalf("expected QuestDB storage-unit exists query to use COUNT(1), got:\n%s", existsQuery)
	}

	columnQuery := plugin.getColumnsQuery()
	if !strings.Contains(columnQuery, "information_schema.columns") {
		t.Fatalf("expected QuestDB columns query to use information_schema.columns, got:\n%s", columnQuery)
	}

	pkQuery := plugin.GetPrimaryKeyColQuery()
	if !strings.Contains(pkQuery, "($1 = '' OR n.nspname = $1)") {
		t.Fatalf("expected QuestDB primary-key query to tolerate empty schema, got:\n%s", pkQuery)
	}
}

func TestQuestDBNormalizesColumnMetadata(t *testing.T) {
	plugin := NewQuestDBPlugin().PluginFunctions.(*QuestDBPlugin)

	column := plugin.normalizeQuestDBColumnMetadata("created_at", "timestamp without time zone", "YES")
	if column.name != "created_at" {
		t.Fatalf("expected column name created_at, got %q", column.name)
	}
	if column.dataType != "TIMESTAMP" {
		t.Fatalf("expected normalized type TIMESTAMP, got %q", column.dataType)
	}
	if !column.isNullable {
		t.Fatal("expected YES nullable metadata to map to true")
	}
}

func TestQuestDBReturnsNoForeignKeyRelationships(t *testing.T) {
	plugin := NewQuestDBPlugin().PluginFunctions.(*QuestDBPlugin)

	relationships, err := plugin.GetForeignKeyRelationships(nil, "", "users")
	if err != nil {
		t.Fatalf("GetForeignKeyRelationships returned error: %v", err)
	}
	if len(relationships) != 0 {
		t.Fatalf("expected no QuestDB foreign-key relationships, got %#v", relationships)
	}
}

func TestQuestDBRegistersPostgresStyleSSLModes(t *testing.T) {
	modes := ssl.GetSSLModes(engine.DatabaseType_QuestDB)
	if len(modes) != 4 {
		t.Fatalf("expected four QuestDB SSL modes, got %#v", modes)
	}
	if ssl.NormalizeSSLMode(engine.DatabaseType_QuestDB, "verify-full") != ssl.SSLModeVerifyIdentity {
		t.Fatal("expected QuestDB to reuse PostgreSQL SSL mode aliases")
	}
}

func TestQuestDBGetSSLStatusUsesConfiguredMode(t *testing.T) {
	plugin := NewQuestDBPlugin().PluginFunctions.(*QuestDBPlugin)

	disabledStatus, err := plugin.GetSSLStatus(&engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type: string(engine.DatabaseType_QuestDB),
		},
	})
	if err != nil {
		t.Fatalf("GetSSLStatus returned error for disabled config: %v", err)
	}
	if disabledStatus == nil || disabledStatus.IsEnabled || disabledStatus.Mode != string(ssl.SSLModeDisabled) {
		t.Fatalf("expected disabled QuestDB SSL status, got %#v", disabledStatus)
	}

	enabledStatus, err := plugin.GetSSLStatus(&engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     string(engine.DatabaseType_QuestDB),
			Hostname: "questdb.local",
			Advanced: []engine.Record{{Key: ssl.KeySSLMode, Value: "require"}},
		},
	})
	if err != nil {
		t.Fatalf("GetSSLStatus returned error for enabled config: %v", err)
	}
	if enabledStatus == nil || !enabledStatus.IsEnabled || enabledStatus.Mode != string(ssl.SSLModeRequired) {
		t.Fatalf("expected enabled QuestDB SSL status, got %#v", enabledStatus)
	}
}
