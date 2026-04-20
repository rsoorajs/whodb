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

package sourcecatalog

import (
	"strings"
	"sync"

	"github.com/clidey/whodb/core/src/azure"
)

var (
	azureADScopeMu    sync.RWMutex
	azureADScopeSpecs = map[string]string{}
)

func init() {
	RegisterAzureADScopeAliases(azure.ScopePostgreSQLMySQL, connectorPostgres, connectorMySQL)
	RegisterAzureADScope(connectorRedis, azure.ScopeRedis)
}

// RegisterAzureADScope registers a source-owned Azure AD token scope for one
// source type or connector id.
func RegisterAzureADScope(id string, scope string) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(scope) == "" {
		return
	}

	azureADScopeMu.Lock()
	defer azureADScopeMu.Unlock()
	azureADScopeSpecs[strings.ToLower(strings.TrimSpace(id))] = strings.TrimSpace(scope)
}

// RegisterAzureADScopeAliases registers the same Azure AD token scope for
// multiple source type or connector ids.
func RegisterAzureADScopeAliases(scope string, ids ...string) {
	for _, id := range ids {
		RegisterAzureADScope(id, scope)
	}
}

// ResolveAzureADScope resolves the first registered Azure AD token scope for
// the provided source type or connector ids.
func ResolveAzureADScope(ids ...string) (string, bool) {
	azureADScopeMu.RLock()
	defer azureADScopeMu.RUnlock()

	for _, id := range ids {
		scope, ok := azureADScopeSpecs[strings.ToLower(strings.TrimSpace(id))]
		if ok {
			return scope, true
		}
	}

	return "", false
}
