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

package dbcatalog

import (
	"slices"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
)

func sslModesFor(dbType engine.DatabaseType) []source.SSLModeInfo {
	switch dbType {
	case engine.DatabaseType_Postgres, engine.DatabaseType_CockroachDB, engine.DatabaseType_QuestDB:
		return cloneSSLModes(sslModesStandard("disable", "require", "verify-full"))
	case engine.DatabaseType_MySQL, engine.DatabaseType_MariaDB, engine.DatabaseType_TiDB:
		return cloneSSLModes(sslModesWithPreferred("DISABLED", "PREFERRED", "REQUIRED", "VERIFY_CA", "VERIFY_IDENTITY"))
	case engine.DatabaseType_ClickHouse, engine.DatabaseType_MongoDB, engine.DatabaseType_Redis, engine.DatabaseType_Memcached, engine.DatabaseType_ElasticSearch:
		return cloneSSLModes(sslModesSimple())
	default:
		return nil
	}
}

func sslModesStandard(disabledAlias string, requiredAlias string, verifyIdentityAlias string) []source.SSLModeInfo {
	return []source.SSLModeInfo{
		sourceSSLMode(ssl.ModeInfoDisabled, disabledAlias),
		sourceSSLMode(ssl.ModeInfoRequired, requiredAlias),
		sourceSSLMode(ssl.ModeInfoVerifyCA),
		sourceSSLMode(ssl.ModeInfoVerifyIdentity, verifyIdentityAlias),
	}
}

func sslModesWithPreferred(disabledAlias string, preferredAlias string, requiredAlias string, verifyCAAlias string, verifyIdentityAlias string) []source.SSLModeInfo {
	return []source.SSLModeInfo{
		sourceSSLMode(ssl.ModeInfoDisabled, disabledAlias),
		sourceSSLMode(ssl.ModeInfoPreferred, preferredAlias),
		sourceSSLMode(ssl.ModeInfoRequired, requiredAlias),
		sourceSSLMode(ssl.ModeInfoVerifyCA, verifyCAAlias),
		sourceSSLMode(ssl.ModeInfoVerifyIdentity, verifyIdentityAlias),
	}
}

func sslModesSimple() []source.SSLModeInfo {
	return []source.SSLModeInfo{
		sourceSSLMode(ssl.ModeInfoDisabled),
		sourceSSLMode(ssl.ModeInfoEnabled),
		sourceSSLMode(ssl.ModeInfoInsecure),
	}
}

func sourceSSLMode(info ssl.SSLModeInfo, aliases ...string) source.SSLModeInfo {
	mode := source.SSLModeInfo{
		Value:       string(info.Value),
		Label:       info.Label,
		Description: info.Description,
	}
	if len(aliases) > 0 {
		mode.Aliases = slices.Clone(aliases)
	}
	return mode
}

func cloneSSLModes(modes []source.SSLModeInfo) []source.SSLModeInfo {
	if len(modes) == 0 {
		return nil
	}
	cloned := make([]source.SSLModeInfo, 0, len(modes))
	for _, mode := range modes {
		cloned = append(cloned, source.SSLModeInfo{
			Value:       mode.Value,
			Label:       mode.Label,
			Description: mode.Description,
			Aliases:     slices.Clone(mode.Aliases),
		})
	}
	return cloned
}
