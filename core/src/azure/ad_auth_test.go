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

package azure

import (
	"testing"
)

func TestScopeConstants(t *testing.T) {
	if ScopePostgreSQLMySQL != "https://ossrdbms-aad.database.windows.net/.default" {
		t.Errorf("ScopePostgreSQLMySQL = %q, want ossrdbms-aad scope", ScopePostgreSQLMySQL)
	}
	if ScopeRedis != "https://redis.azure.com/.default" {
		t.Errorf("ScopeRedis = %q, want redis.azure.com scope", ScopeRedis)
	}
}

func TestScopeForDatabaseType(t *testing.T) {
	tests := []struct {
		dbType    string
		wantScope string
		wantErr   bool
	}{
		{"Postgres", ScopePostgreSQLMySQL, false},
		{"PostgreSQL", ScopePostgreSQLMySQL, false},
		{"MySQL", ScopePostgreSQLMySQL, false},
		{"Redis", ScopeRedis, false},
		{"MongoDB", "", true},
		{"DynamoDB", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			scope, err := ScopeForDatabaseType(tt.dbType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScopeForDatabaseType(%q) error = %v, wantErr %v", tt.dbType, err, tt.wantErr)
				return
			}
			if scope != tt.wantScope {
				t.Errorf("ScopeForDatabaseType(%q) = %q, want %q", tt.dbType, scope, tt.wantScope)
			}
		})
	}
}
