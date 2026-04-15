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
	"strings"

	"github.com/clidey/whodb/core/src/dbcatalog"
	"github.com/clidey/whodb/core/src/engine"
)

var dbTypeSynonyms = map[string]string{
	"postgresql": "Postgres",
	"sqlite":     "Sqlite3",
	"cockroach":  "CockroachDB",
	"yugabyte":   "YugabyteDB",
	"quest":      "QuestDB",
	"elastic":    "ElasticSearch",
}

func lookupDatabaseType(input string) (dbcatalog.ConnectableDatabase, bool) {
	if strings.TrimSpace(input) == "" {
		return dbcatalog.ConnectableDatabase{}, false
	}

	normalizedInput := normalizeDBTypeKey(input)
	if alias, ok := dbTypeSynonyms[normalizedInput]; ok {
		return dbcatalog.Find(alias)
	}

	for _, id := range dbcatalog.IDs() {
		if normalizeDBTypeKey(id) == normalizedInput {
			return dbcatalog.Find(id)
		}
	}

	return dbcatalog.ConnectableDatabase{}, false
}

func normalizeDBType(dbType string) string {
	entry, ok := lookupDatabaseType(dbType)
	if !ok {
		return strings.TrimSpace(dbType)
	}
	return string(entry.ID)
}

func getDefaultPort(dbType string) int {
	port, ok := dbcatalog.DefaultPort(normalizeDBType(dbType))
	if !ok {
		return 5432
	}
	return port
}

func isFileBasedDatabaseType(dbType string) bool {
	entry, ok := lookupDatabaseType(dbType)
	if !ok {
		return false
	}

	switch entry.PluginType {
	case engine.DatabaseType_Sqlite3, engine.DatabaseType_DuckDB:
		return true
	default:
		return false
	}
}

func normalizeDBTypeKey(value string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}
