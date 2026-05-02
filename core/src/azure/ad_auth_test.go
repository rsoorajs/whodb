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

package azure_test

import (
	"testing"

	"github.com/clidey/whodb/core/src/azure"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func TestScopeConstants(t *testing.T) {
	if azure.ScopePostgreSQLMySQL != "https://ossrdbms-aad.database.windows.net/.default" {
		t.Errorf("ScopePostgreSQLMySQL = %q, want ossrdbms-aad scope", azure.ScopePostgreSQLMySQL)
	}
	if azure.ScopeRedis != "https://redis.azure.com/.default" {
		t.Errorf("ScopeRedis = %q, want redis.azure.com scope", azure.ScopeRedis)
	}
}

func TestSourceCatalogAzureADScopes(t *testing.T) {
	tests := []struct {
		ids       []string
		wantScope string
		wantFound bool
	}{
		{[]string{"Postgres"}, azure.ScopePostgreSQLMySQL, true},
		{[]string{"Aurora MySQL", "MySQL"}, azure.ScopePostgreSQLMySQL, true},
		{[]string{"Azure Managed Redis", "Redis"}, azure.ScopeRedis, true},
		{[]string{"MongoDB"}, "", false},
		{[]string{"DynamoDB"}, "", false},
		{[]string{""}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.ids[0], func(t *testing.T) {
			scope, ok := sourcecatalog.ResolveAzureADScope(tt.ids...)
			if ok != tt.wantFound {
				t.Errorf("ResolveAzureADScope(%q) found = %v, want %v", tt.ids, ok, tt.wantFound)
				return
			}
			if scope != tt.wantScope {
				t.Errorf("ResolveAzureADScope(%q) = %q, want %q", tt.ids, scope, tt.wantScope)
			}
		})
	}
}
