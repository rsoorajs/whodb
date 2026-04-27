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

	"github.com/clidey/whodb/core/src/engine"
	_ "github.com/clidey/whodb/core/src/sources/database"
)

func TestNewYugabyteDBPlugin(t *testing.T) {
	pluginDef := NewYugabyteDBPlugin()
	if pluginDef.Type != engine.DatabaseType_YugabyteDB {
		t.Fatalf("expected type %q, got %q", engine.DatabaseType_YugabyteDB, pluginDef.Type)
	}

	plugin, ok := pluginDef.PluginFunctions.(*YugabyteDBPlugin)
	if !ok {
		t.Fatalf("unexpected YugabyteDB plugin type %T", pluginDef.PluginFunctions)
	}
	if plugin.GormPluginFunctions != plugin {
		t.Fatal("expected YugabyteDB Gorm plugin hooks to point at the YugabyteDB wrapper")
	}
}

func TestYugabyteDBOverridesPostgresCatalogQueries(t *testing.T) {
	plugin := NewYugabyteDBPlugin().PluginFunctions.(*YugabyteDBPlugin)

	tableInfoQuery := plugin.GetTableInfoQuery()
	if strings.Contains(tableInfoQuery, "pg_total_relation_size") {
		t.Fatalf("expected YugabyteDB table info query to avoid pg_total_relation_size, got:\n%s", tableInfoQuery)
	}
	if !strings.Contains(tableInfoQuery, "information_schema.tables") {
		t.Fatalf("expected YugabyteDB table info query to use information_schema.tables, got:\n%s", tableInfoQuery)
	}

	pkQuery := plugin.GetPrimaryKeyColQuery()
	if strings.Contains(pkQuery, "ANY(") {
		t.Fatalf("expected YugabyteDB primary-key query to avoid ANY(), got:\n%s", pkQuery)
	}
	if !strings.Contains(pkQuery, "information_schema.key_column_usage") {
		t.Fatalf("expected YugabyteDB primary-key query to use information_schema.key_column_usage, got:\n%s", pkQuery)
	}
}
