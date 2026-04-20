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
)

// QuestDBPlugin extends PostgresPlugin with QuestDB-specific catalog behavior.
// QuestDB uses the PostgreSQL wire protocol in our product, but its table
// metadata path is schema-less in practice and does not support the PostgreSQL
// relation size functions used by the base plugin.
type QuestDBPlugin struct {
	PostgresPlugin
}

// GetTableInfoQuery returns a QuestDB-compatible table info query.
func (p *QuestDBPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			t.table_name,
			t.table_type
		FROM
			information_schema.tables t
		WHERE
			($1 = '' OR t.table_schema = $1)
			AND t.table_schema NOT IN ('information_schema', 'pg_catalog');
	`
}

// GetTableNameAndAttributes parses QuestDB table info rows.
func (p *QuestDBPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.WithError(err).Error("Failed to scan QuestDB table info row data")
		return "", nil
	}

	return tableName, []engine.Record{
		{Key: "Type", Value: tableType},
	}
}

// GetStorageUnitExistsQuery returns a QuestDB-compatible table existence check.
func (p *QuestDBPlugin) GetStorageUnitExistsQuery() string {
	return `
		SELECT EXISTS(
			SELECT 1
			FROM information_schema.tables
			WHERE ($1 = '' OR table_schema = $1)
				AND table_name = $2
				AND table_schema NOT IN ('information_schema', 'pg_catalog')
		)
	`
}

// GetPrimaryKeyColQuery returns a primary key query that tolerates schema-less
// QuestDB source references.
func (p *QuestDBPlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE ($1 = '' OR n.nspname = $1) AND c.relname = $2 AND i.indisprimary;
	`
}

// GetForeignKeyRelationships returns an empty relationship set because the
// QuestDB fixtures and source model treat QuestDB tables as lacking foreign-key
// graph metadata.
func (p *QuestDBPlugin) GetForeignKeyRelationships(_ *engine.PluginConfig, _, _ string) (map[string]*engine.ForeignKeyRelationship, error) {
	return map[string]*engine.ForeignKeyRelationship{}, nil
}

// GetSSLStatus derives QuestDB SSL status from connection configuration.
// QuestDB speaks the PostgreSQL wire protocol but does not expose pg_stat_ssl,
// so the generic PostgreSQL runtime query fails.
func (p *QuestDBPlugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
	if cached := plugins.GetCachedSSLStatus(config); cached != nil {
		return cached, nil
	}

	sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_QuestDB, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)

	var status *engine.SSLStatus
	if sslConfig == nil || !sslConfig.IsEnabled() {
		status = &engine.SSLStatus{
			IsEnabled: false,
			Mode:      string(ssl.SSLModeDisabled),
		}
	} else {
		status = &engine.SSLStatus{
			IsEnabled: true,
			Mode:      string(sslConfig.Mode),
		}
	}

	plugins.SetCachedSSLStatus(config, status)
	return status, nil
}

// NewQuestDBPlugin creates a QuestDB plugin that reuses the PostgreSQL runtime
// while overriding the incompatible catalog and metadata paths.
func NewQuestDBPlugin() *engine.Plugin {
	questDBPlugin := &QuestDBPlugin{}
	questDBPlugin.Type = engine.DatabaseType_QuestDB
	questDBPlugin.PluginFunctions = questDBPlugin
	questDBPlugin.GormPluginFunctions = questDBPlugin
	return &questDBPlugin.Plugin
}

func init() {
	ssl.RegisterDatabaseSSLModes(engine.DatabaseType_QuestDB, []ssl.SSLModeInfo{
		ssl.ModeInfoDisabled,
		ssl.ModeInfoRequired,
		ssl.ModeInfoVerifyCA,
		ssl.ModeInfoVerifyIdentity,
	})
	ssl.RegisterSSLModeAliases(engine.DatabaseType_QuestDB, map[string]ssl.SSLMode{
		"disable":     ssl.SSLModeDisabled,
		"require":     ssl.SSLModeRequired,
		"verify-full": ssl.SSLModeVerifyIdentity,
	})
	engine.RegisterPlugin(NewQuestDBPlugin())
}
