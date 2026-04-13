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

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// CockroachDBPlugin extends PostgresPlugin with CockroachDB-specific overrides.
// CockroachDB is PostgreSQL wire-compatible but lacks some pg_catalog functions.
type CockroachDBPlugin struct {
	PostgresPlugin
}

// GetTableInfoQuery returns a CockroachDB-compatible table info query.
// CockroachDB does not support pg_size_pretty() or pg_total_relation_size(),
// so we query only the table name and type from information_schema.
func (p *CockroachDBPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			t.table_name,
			t.table_type
		FROM
			information_schema.tables t
		WHERE
			t.table_schema = ?;
	`
}

// GetTableNameAndAttributes parses CockroachDB table info rows.
func (p *CockroachDBPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.WithError(err).Error("Failed to scan CockroachDB table info row data")
		return "", nil
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
	}

	return tableName, attributes
}

// GetStorageUnitExistsQuery returns a CockroachDB-compatible table existence check.
// CockroachDB does not support to_regclass().
func (p *CockroachDBPlugin) GetStorageUnitExistsQuery() string {
	return `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)`
}

// IsGeometryType returns false for CockroachDB since it has limited geometry support
// compared to PostGIS and does not use the same binary encoding.
func (p *CockroachDBPlugin) IsGeometryType(columnType string) bool {
	return false
}

// GetSSLStatus determines SSL status for CockroachDB connections.
// CockroachDB does not have pg_stat_ssl, so we query the session's ssl variable
// via SHOW ssl (returns "on"/"off" as a string).
func (p *CockroachDBPlugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
	if cached := plugins.GetCachedSSLStatus(config); cached != nil {
		return cached, nil
	}

	status, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.SSLStatus, error) {
		var result struct {
			SSL string `gorm:"column:ssl"`
		}

		query := db.Raw("SHOW ssl").Scan(&result)
		if query.Error != nil {
			return nil, query.Error
		}

		if result.SSL != "on" {
			return &engine.SSLStatus{
				IsEnabled: false,
				Mode:      string(ssl.SSLModeDisabled),
			}, nil
		}

		sslConfig := ssl.ParseSSLConfig(engine.DatabaseType(p.Type), config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)
		mode := "enabled"
		if sslConfig != nil {
			mode = string(sslConfig.Mode)
		}

		return &engine.SSLStatus{
			IsEnabled: true,
			Mode:      mode,
		}, nil
	})

	if err == nil && status != nil {
		plugins.SetCachedSSLStatus(config, status)
	}
	return status, err
}

// CockroachDB-supported type definitions (excludes MONEY, XML, HSTORE, geometric types,
// CIDR, MACADDR, TIMETZ which CockroachDB does not support).
var cockroachDBTypeDefinitions = []engine.TypeDefinition{
	{ID: "SMALLINT", Label: "smallint", Category: engine.TypeCategoryNumeric},
	{ID: "INTEGER", Label: "integer", Category: engine.TypeCategoryNumeric},
	{ID: "BIGINT", Label: "bigint", Category: engine.TypeCategoryNumeric},
	{ID: "SMALLSERIAL", Label: "smallserial", Category: engine.TypeCategoryNumeric},
	{ID: "SERIAL", Label: "serial", Category: engine.TypeCategoryNumeric},
	{ID: "BIGSERIAL", Label: "bigserial", Category: engine.TypeCategoryNumeric},
	{ID: "DECIMAL", Label: "decimal", HasPrecision: true, DefaultPrecision: engine.IntPtr(10), Category: engine.TypeCategoryNumeric},
	{ID: "NUMERIC", Label: "numeric", HasPrecision: true, DefaultPrecision: engine.IntPtr(10), Category: engine.TypeCategoryNumeric},
	{ID: "REAL", Label: "real", Category: engine.TypeCategoryNumeric},
	{ID: "DOUBLE PRECISION", Label: "double precision", Category: engine.TypeCategoryNumeric},
	{ID: "CHARACTER VARYING", Label: "varchar", HasLength: true, DefaultLength: engine.IntPtr(255), Category: engine.TypeCategoryText},
	{ID: "CHARACTER", Label: "char", HasLength: true, DefaultLength: engine.IntPtr(1), Category: engine.TypeCategoryText},
	{ID: "TEXT", Label: "text", Category: engine.TypeCategoryText},
	{ID: "BYTEA", Label: "bytea", Category: engine.TypeCategoryBinary},
	{ID: "TIMESTAMP", Label: "timestamp", Category: engine.TypeCategoryDatetime},
	{ID: "TIMESTAMP WITH TIME ZONE", Label: "timestamptz", Category: engine.TypeCategoryDatetime},
	{ID: "DATE", Label: "date", Category: engine.TypeCategoryDatetime},
	{ID: "TIME", Label: "time", Category: engine.TypeCategoryDatetime},
	{ID: "BOOLEAN", Label: "boolean", Category: engine.TypeCategoryBoolean},
	{ID: "JSON", Label: "json", Category: engine.TypeCategoryJSON},
	{ID: "JSONB", Label: "jsonb", Category: engine.TypeCategoryJSON},
	{ID: "UUID", Label: "uuid", Category: engine.TypeCategoryOther},
	{ID: "INET", Label: "inet", Category: engine.TypeCategoryOther},
	{ID: "ARRAY", Label: "array", Category: engine.TypeCategoryOther},
}

// GetDatabaseMetadata returns CockroachDB metadata with only supported types.
func (p *CockroachDBPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	operators := make([]string, 0, len(supportedOperators))
	for op := range supportedOperators {
		operators = append(operators, op)
	}
	return &engine.DatabaseMetadata{
		DatabaseType:    engine.DatabaseType(p.Type),
		TypeDefinitions: cockroachDBTypeDefinitions,
		Operators:       operators,
		AliasMap:        AliasMap,
		Capabilities: engine.Capabilities{
			SupportsScratchpad:     true,
			SupportsChat:           true,
			SupportsGraph:          true,
			SupportsSchema:         true,
			SupportsDatabaseSwitch: true,
			SupportsModifiers:      true,
		},
	}
}

// NewCockroachDBPlugin creates a CockroachDB plugin with PostgreSQL compatibility
// and CockroachDB-specific overrides for unsupported catalog functions.
func NewCockroachDBPlugin() *engine.Plugin {
	crdbPlugin := &CockroachDBPlugin{}
	crdbPlugin.Type = engine.DatabaseType_CockroachDB
	crdbPlugin.PluginFunctions = crdbPlugin
	crdbPlugin.GormPluginFunctions = crdbPlugin
	return &crdbPlugin.Plugin
}
