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

// Package dbcatalog exposes the shared connectable database catalog used by the
// frontend, desktop, and CLI.
package dbcatalog

import (
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
)

// FieldVisibility declares which standard connection form fields are shown for
// a database type.
type FieldVisibility struct {
	Hostname   bool
	Username   bool
	Password   bool
	Database   bool
	SearchPath bool
}

// FieldRequirements declares which standard connection form fields are
// required for a database type.
type FieldRequirements struct {
	Hostname bool
	Username bool
	Password bool
	Database bool
}

// ConnectableDatabase describes a database type that WhoDB can connect to from
// the shared login and connection flows.
type ConnectableDatabase struct {
	ID                          engine.DatabaseType
	Label                       string
	PluginType                  engine.DatabaseType
	Extra                       map[string]string
	Fields                      FieldVisibility
	RequiredFields              FieldRequirements
	SupportsModifiers           bool
	SupportsScratchpad          bool
	SupportsSchema              bool
	SupportsDatabaseSwitching   bool
	UsesSchemaForGraph          bool
	UsesDatabaseInsteadOfSchema bool
	SupportsMockData            bool
	IsAWSManaged                bool
	SSLModes                    []ssl.SSLModeInfo
}

var catalog = []ConnectableDatabase{
	{
		ID:         engine.DatabaseType_Postgres,
		Label:      "Postgres",
		PluginType: engine.DatabaseType_Postgres,
		Extra:      map[string]string{"Port": "5432"},
		Fields: FieldVisibility{
			Hostname:   true,
			Username:   true,
			Password:   true,
			Database:   true,
			SearchPath: true,
		},
		RequiredFields:            FieldRequirements{Hostname: true, Username: true, Password: true, Database: true},
		SupportsModifiers:         true,
		SupportsScratchpad:        true,
		SupportsSchema:            true,
		SupportsDatabaseSwitching: true,
		UsesSchemaForGraph:        true,
		SupportsMockData:          true,
		SSLModes:                  sslModesFor(engine.DatabaseType_Postgres),
	},
	{
		ID:         engine.DatabaseType_MySQL,
		Label:      "MySQL",
		PluginType: engine.DatabaseType_MySQL,
		Extra: map[string]string{
			"Port":                       "3306",
			"Parse Time":                 "True",
			"Loc":                        "UTC",
			"Allow clear text passwords": "0",
		},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
			Database: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true, Username: true, Password: true, Database: true},
		SupportsModifiers:           true,
		SupportsScratchpad:          true,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            true,
		SSLModes:                    sslModesFor(engine.DatabaseType_MySQL),
	},
	{
		ID:         engine.DatabaseType_MariaDB,
		Label:      "MariaDB",
		PluginType: engine.DatabaseType_MariaDB,
		Extra: map[string]string{
			"Port":                       "3306",
			"Parse Time":                 "True",
			"Loc":                        "UTC",
			"Allow clear text passwords": "0",
		},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
			Database: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true, Username: true, Password: true, Database: true},
		SupportsModifiers:           true,
		SupportsScratchpad:          true,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            true,
		SSLModes:                    sslModesFor(engine.DatabaseType_MariaDB),
	},
	{
		ID:         engine.DatabaseType_CockroachDB,
		Label:      "CockroachDB",
		PluginType: engine.DatabaseType_CockroachDB,
		Extra:      map[string]string{"Port": "26257"},
		Fields: FieldVisibility{
			Hostname:   true,
			Username:   true,
			Password:   true,
			Database:   true,
			SearchPath: true,
		},
		RequiredFields:            FieldRequirements{Hostname: true, Username: true, Password: true, Database: true},
		SupportsModifiers:         true,
		SupportsScratchpad:        true,
		SupportsSchema:            true,
		SupportsDatabaseSwitching: true,
		UsesSchemaForGraph:        true,
		SupportsMockData:          true,
		SSLModes:                  sslModesFor(engine.DatabaseType_CockroachDB),
	},
	{
		ID:         engine.DatabaseType_Sqlite3,
		Label:      "Sqlite3",
		PluginType: engine.DatabaseType_Sqlite3,
		Extra:      map[string]string{},
		Fields: FieldVisibility{
			Database: true,
		},
		RequiredFields:            FieldRequirements{Database: true},
		SupportsModifiers:         true,
		SupportsScratchpad:        true,
		SupportsSchema:            false,
		SupportsDatabaseSwitching: false,
		UsesSchemaForGraph:        true,
		SupportsMockData:          true,
	},
	{
		ID:         engine.DatabaseType_MongoDB,
		Label:      "MongoDB",
		PluginType: engine.DatabaseType_MongoDB,
		Extra: map[string]string{
			"Port":        "27017",
			"URL Params":  "?",
			"DNS Enabled": "false",
		},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
			Database: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true},
		SupportsModifiers:           false,
		SupportsScratchpad:          false,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            true,
		SSLModes:                    sslModesFor(engine.DatabaseType_MongoDB),
	},
	{
		ID:         engine.DatabaseType_Redis,
		Label:      "Redis",
		PluginType: engine.DatabaseType_Redis,
		Extra:      map[string]string{"Port": "6379"},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true},
		SupportsModifiers:           false,
		SupportsScratchpad:          false,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            false,
		SSLModes:                    sslModesFor(engine.DatabaseType_Redis),
	},
	{
		ID:         engine.DatabaseType_ElasticSearch,
		Label:      "ElasticSearch",
		PluginType: engine.DatabaseType_ElasticSearch,
		Extra:      map[string]string{"Port": "9200"},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
		},
		RequiredFields:            FieldRequirements{Hostname: true},
		SupportsScratchpad:        false,
		SupportsSchema:            false,
		SupportsDatabaseSwitching: false,
		UsesSchemaForGraph:        false,
		SupportsMockData:          false,
		SSLModes:                  sslModesFor(engine.DatabaseType_ElasticSearch),
	},
	{
		ID:         engine.DatabaseType_ClickHouse,
		Label:      "ClickHouse",
		PluginType: engine.DatabaseType_ClickHouse,
		Extra: map[string]string{
			"Port":          "9000",
			"HTTP Protocol": "disable",
			"Readonly":      "disable",
			"Debug":         "disable",
		},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
			Database: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true, Username: true, Password: true, Database: true},
		SupportsModifiers:           true,
		SupportsScratchpad:          true,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            true,
		SSLModes:                    sslModesFor(engine.DatabaseType_ClickHouse),
	},
	{
		ID:         engine.DatabaseType_DuckDB,
		Label:      "DuckDB",
		PluginType: engine.DatabaseType_DuckDB,
		Extra:      map[string]string{},
		Fields: FieldVisibility{
			Database: true,
		},
		RequiredFields:            FieldRequirements{Database: true},
		SupportsModifiers:         true,
		SupportsScratchpad:        true,
		SupportsSchema:            true,
		SupportsDatabaseSwitching: false,
		UsesSchemaForGraph:        true,
		SupportsMockData:          true,
	},
	{
		ID:         engine.DatabaseType_Memcached,
		Label:      "Memcached",
		PluginType: engine.DatabaseType_Memcached,
		Extra:      map[string]string{"Port": "11211"},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
		},
		RequiredFields:            FieldRequirements{Hostname: true},
		SupportsScratchpad:        false,
		SupportsSchema:            false,
		SupportsDatabaseSwitching: false,
		UsesSchemaForGraph:        false,
		SupportsMockData:          false,
		SSLModes:                  sslModesFor(engine.DatabaseType_Memcached),
	},
	{
		ID:         engine.DatabaseType_TiDB,
		Label:      "TiDB",
		PluginType: engine.DatabaseType_TiDB,
		Extra: map[string]string{
			"Port":                       "4000",
			"Parse Time":                 "True",
			"Loc":                        "UTC",
			"Allow clear text passwords": "0",
		},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
			Database: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true, Username: true, Password: true, Database: true},
		SupportsModifiers:           true,
		SupportsScratchpad:          true,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            true,
		SSLModes:                    sslModesFor(engine.DatabaseType_TiDB),
	},
	{
		ID:         engine.DatabaseType_Valkey,
		Label:      "Valkey",
		PluginType: engine.DatabaseType_Redis,
		Extra:      map[string]string{"Port": "6379"},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true},
		SupportsScratchpad:          false,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            false,
		SSLModes:                    sslModesFor(engine.DatabaseType_Redis),
	},
	{
		ID:         engine.DatabaseType_Dragonfly,
		Label:      "Dragonfly",
		PluginType: engine.DatabaseType_Redis,
		Extra:      map[string]string{"Port": "6379"},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true},
		SupportsScratchpad:          false,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            false,
		SSLModes:                    sslModesFor(engine.DatabaseType_Redis),
	},
	{
		ID:         engine.DatabaseType_OpenSearch,
		Label:      "OpenSearch",
		PluginType: engine.DatabaseType_ElasticSearch,
		Extra:      map[string]string{"Port": "9200"},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
		},
		RequiredFields:            FieldRequirements{Hostname: true},
		SupportsScratchpad:        false,
		SupportsSchema:            false,
		SupportsDatabaseSwitching: false,
		UsesSchemaForGraph:        false,
		SupportsMockData:          false,
		SSLModes:                  sslModesFor(engine.DatabaseType_ElasticSearch),
	},
	{
		ID:         engine.DatabaseType_YugabyteDB,
		Label:      "YugabyteDB",
		PluginType: engine.DatabaseType_Postgres,
		Extra:      map[string]string{"Port": "5433"},
		Fields: FieldVisibility{
			Hostname:   true,
			Username:   true,
			Password:   true,
			Database:   true,
			SearchPath: true,
		},
		RequiredFields:            FieldRequirements{Hostname: true, Username: true, Password: true, Database: true},
		SupportsModifiers:         true,
		SupportsScratchpad:        true,
		SupportsSchema:            true,
		SupportsDatabaseSwitching: true,
		UsesSchemaForGraph:        true,
		SupportsMockData:          true,
		SSLModes:                  sslModesFor(engine.DatabaseType_Postgres),
	},
	{
		ID:         engine.DatabaseType_QuestDB,
		Label:      "QuestDB",
		PluginType: engine.DatabaseType_Postgres,
		Extra:      map[string]string{"Port": "8812"},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
			Database: true,
		},
		RequiredFields:            FieldRequirements{Hostname: true, Username: true, Password: true, Database: true},
		SupportsModifiers:         true,
		SupportsScratchpad:        true,
		SupportsSchema:            false,
		SupportsDatabaseSwitching: false,
		UsesSchemaForGraph:        false,
		SupportsMockData:          true,
		SSLModes:                  sslModesFor(engine.DatabaseType_Postgres),
	},
	{
		ID:         engine.DatabaseType_FerretDB,
		Label:      "FerretDB",
		PluginType: engine.DatabaseType_MongoDB,
		Extra: map[string]string{
			"Port":        "27017",
			"URL Params":  "?",
			"DNS Enabled": "false",
		},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
			Database: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true},
		SupportsScratchpad:          false,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            true,
		SSLModes:                    sslModesFor(engine.DatabaseType_MongoDB),
	},
	{
		ID:         engine.DatabaseType_ElastiCache,
		Label:      "ElastiCache",
		PluginType: engine.DatabaseType_Redis,
		Extra: map[string]string{
			"Port": "6379",
			"TLS":  "true",
		},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true},
		SupportsScratchpad:          false,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            false,
		IsAWSManaged:                true,
		SSLModes:                    sslModesFor(engine.DatabaseType_Redis),
	},
	{
		ID:         engine.DatabaseType_DocumentDB,
		Label:      "DocumentDB",
		PluginType: engine.DatabaseType_MongoDB,
		Extra:      map[string]string{"Port": "27017"},
		Fields: FieldVisibility{
			Hostname: true,
			Username: true,
			Password: true,
			Database: true,
		},
		RequiredFields:              FieldRequirements{Hostname: true},
		SupportsScratchpad:          false,
		SupportsSchema:              false,
		SupportsDatabaseSwitching:   true,
		UsesSchemaForGraph:          false,
		UsesDatabaseInsteadOfSchema: true,
		SupportsMockData:            true,
		IsAWSManaged:                true,
		SSLModes:                    sslModesFor(engine.DatabaseType_MongoDB),
	},
}

var registeredCatalog []ConnectableDatabase

func init() {
	for _, entry := range catalog {
		registerPluginAlias(entry)
	}
}

// Register appends additional catalog entries after the core catalog.
// Extension packages use this to register edition-specific database types.
func Register(entries ...ConnectableDatabase) {
	for _, entry := range entries {
		registeredCatalog = append(registeredCatalog, cloneEntry(entry))
		registerPluginAlias(entry)
	}
}

// All returns the full catalog in UI order.
func All() []ConnectableDatabase {
	entries := make([]ConnectableDatabase, 0, len(catalog)+len(registeredCatalog))
	for _, entry := range catalog {
		entries = append(entries, cloneEntry(entry))
	}
	for _, entry := range registeredCatalog {
		entries = append(entries, cloneEntry(entry))
	}
	return entries
}

// IDs returns the catalog database identifiers in UI order.
func IDs() []string {
	ids := make([]string, 0, len(catalog)+len(registeredCatalog))
	for _, entry := range catalog {
		ids = append(ids, string(entry.ID))
	}
	for _, entry := range registeredCatalog {
		ids = append(ids, string(entry.ID))
	}
	return ids
}

// Find looks up a catalog entry by database ID using a case-insensitive match.
func Find(id string) (ConnectableDatabase, bool) {
	for _, entry := range catalog {
		if strings.EqualFold(string(entry.ID), id) {
			return cloneEntry(entry), true
		}
	}
	for _, entry := range registeredCatalog {
		if strings.EqualFold(string(entry.ID), id) {
			return cloneEntry(entry), true
		}
	}
	return ConnectableDatabase{}, false
}

// DefaultPort returns the effective default port for the database type.
func DefaultPort(id string) (int, bool) {
	entry, ok := Find(id)
	if !ok {
		return 0, false
	}

	if port, ok := parsePort(entry.Extra["Port"]); ok {
		return port, true
	}

	defaultPort, ok := plugins.GetDefaultPort(entry.PluginType)
	if !ok {
		return 0, false
	}

	return parsePort(defaultPort)
}

// IsNetworkDatabase reports whether the database connects via a hostname.
func IsNetworkDatabase(id string) bool {
	entry, ok := Find(id)
	return ok && entry.Fields.Hostname
}

func cloneEntry(entry ConnectableDatabase) ConnectableDatabase {
	cloned := entry
	if entry.Extra != nil {
		cloned.Extra = make(map[string]string, len(entry.Extra))
		for key, value := range entry.Extra {
			cloned.Extra[key] = value
		}
	}
	if entry.SSLModes != nil {
		cloned.SSLModes = append([]ssl.SSLModeInfo(nil), entry.SSLModes...)
	}
	return cloned
}

func sslModesFor(dbType engine.DatabaseType) []ssl.SSLModeInfo {
	modes := ssl.GetSSLModes(dbType)
	if len(modes) == 0 {
		return nil
	}
	return append([]ssl.SSLModeInfo(nil), modes...)
}

func parsePort(raw string) (int, bool) {
	if raw == "" {
		return 0, false
	}

	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}

	return port, true
}

func registerPluginAlias(entry ConnectableDatabase) {
	if entry.ID == entry.PluginType {
		return
	}
	engine.RegisterPluginTypeAlias(entry.ID, entry.PluginType)
}
