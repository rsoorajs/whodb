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
	"maps"
	"slices"
	"testing"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/dbcatalog"
	"github.com/clidey/whodb/core/src/source"
)

func TestBuildTypeSpecCoversSharedDatabaseCatalog(t *testing.T) {
	t.Parallel()

	for _, entry := range dbcatalog.All() {
		entry := entry
		t.Run(string(entry.ID), func(t *testing.T) {
			t.Parallel()

			spec, ok := BuildTypeSpec(DatabaseEntry{
				ID:             string(entry.ID),
				Label:          entry.Label,
				Connector:      string(entry.PluginType),
				Extra:          maps.Clone(entry.Extra),
				Fields:         FieldVisibility(entry.Fields),
				RequiredFields: FieldRequirements(entry.RequiredFields),
				IsAWSManaged:   entry.IsAWSManaged,
				SSLModes:       sourceSSLModes(entry.SSLModes),
			})
			if !ok {
				t.Fatalf("expected %q to map into the source catalog", entry.ID)
			}
			if spec.ID != string(entry.ID) {
				t.Fatalf("expected source id %q, got %q", entry.ID, spec.ID)
			}
		})
	}
}

func TestBuildTypeSpecExposesMutableDataActions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id      string
		kind    source.ObjectKind
		actions []source.Action
	}{
		{
			id:   "Postgres",
			kind: source.ObjectKindTable,
			actions: []source.Action{
				source.ActionUpdateData,
				source.ActionDeleteData,
			},
		},
		{
			id:   "MongoDB",
			kind: source.ObjectKindCollection,
			actions: []source.Action{
				source.ActionUpdateData,
				source.ActionDeleteData,
			},
		},
		{
			id:   "Redis",
			kind: source.ObjectKindKey,
			actions: []source.Action{
				source.ActionUpdateData,
				source.ActionDeleteData,
			},
		},
		{
			id:   "Memcached",
			kind: source.ObjectKindItem,
			actions: []source.Action{
				source.ActionUpdateData,
				source.ActionDeleteData,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()

			entry, ok := dbcatalog.Find(tt.id)
			if !ok {
				t.Fatalf("expected database catalog entry for %q", tt.id)
			}

			spec, ok := BuildTypeSpec(DatabaseEntry{
				ID:             string(entry.ID),
				Label:          entry.Label,
				Connector:      string(entry.PluginType),
				Extra:          maps.Clone(entry.Extra),
				Fields:         FieldVisibility(entry.Fields),
				RequiredFields: FieldRequirements(entry.RequiredFields),
				IsAWSManaged:   entry.IsAWSManaged,
				SSLModes:       sourceSSLModes(entry.SSLModes),
			})
			if !ok {
				t.Fatalf("expected %q to map into the source catalog", tt.id)
			}

			objectType, ok := spec.Contract.ObjectTypeForKind(tt.kind)
			if !ok {
				t.Fatalf("expected object kind %q for %q", tt.kind, tt.id)
			}

			for _, action := range tt.actions {
				if !slices.Contains(objectType.Actions, action) {
					t.Fatalf("expected %q to expose action %q, got %v", tt.id, action, objectType.Actions)
				}
			}
		})
	}
}

func TestBuildTypeSpecExposesSourceTraits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   string
		want func(t *testing.T, spec source.TypeSpec)
	}{
		{
			id: "Sqlite3",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Connection.Transport != source.ConnectionTransportFile {
					t.Fatalf("expected Sqlite3 transport %q, got %q", source.ConnectionTransportFile, spec.Traits.Connection.Transport)
				}
				if spec.Traits.Connection.HostInputMode != source.HostInputModeNone {
					t.Fatalf("expected Sqlite3 host input mode %q, got %q", source.HostInputModeNone, spec.Traits.Connection.HostInputMode)
				}
				if spec.Traits.Presentation.ProfileLabelStrategy != source.ProfileLabelStrategyDatabase {
					t.Fatalf("expected Sqlite3 profile label strategy %q, got %q", source.ProfileLabelStrategyDatabase, spec.Traits.Presentation.ProfileLabelStrategy)
				}
				databaseField, ok := spec.ConnectionFieldByKey("Database")
				if !ok {
					t.Fatalf("expected Sqlite3 database field")
				}
				if databaseField.Kind != source.ConnectionFieldKindFilePath {
					t.Fatalf("expected Sqlite3 database field kind %q, got %q", source.ConnectionFieldKindFilePath, databaseField.Kind)
				}
				if !databaseField.SupportsOptions {
					t.Fatalf("expected Sqlite3 database field options support")
				}
			},
		},
		{
			id: "Postgres",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Connection.HostInputMode != source.HostInputModeHostnameOrURL {
					t.Fatalf("expected Postgres host input mode %q, got %q", source.HostInputModeHostnameOrURL, spec.Traits.Connection.HostInputMode)
				}
				if spec.Traits.Connection.HostInputURLParser != source.HostInputURLParserPostgres {
					t.Fatalf("expected Postgres URL parser %q, got %q", source.HostInputURLParserPostgres, spec.Traits.Connection.HostInputURLParser)
				}
				if !spec.Traits.Query.SupportsAnalyze {
					t.Fatalf("expected Postgres analyze support")
				}
			},
		},
		{
			id: "YugabyteDB",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Connection.HostInputMode != source.HostInputModeHostnameOrURL {
					t.Fatalf("expected YugabyteDB host input mode %q, got %q", source.HostInputModeHostnameOrURL, spec.Traits.Connection.HostInputMode)
				}
				if spec.Traits.Query.SupportsAnalyze {
					t.Fatalf("expected YugabyteDB analyze support to remain disabled")
				}
			},
		},
		{
			id: "MongoDB",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Connection.HostInputURLParser != source.HostInputURLParserMongoSRV {
					t.Fatalf("expected MongoDB URL parser %q, got %q", source.HostInputURLParserMongoSRV, spec.Traits.Connection.HostInputURLParser)
				}
				if spec.Traits.Presentation.SchemaFidelity != source.SchemaFidelitySampled {
					t.Fatalf("expected MongoDB schema fidelity %q, got %q", source.SchemaFidelitySampled, spec.Traits.Presentation.SchemaFidelity)
				}
			},
		},
		{
			id: "Valkey",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Presentation.ProfileLabelStrategy != source.ProfileLabelStrategyHostname {
					t.Fatalf("expected Valkey profile label strategy %q, got %q", source.ProfileLabelStrategyHostname, spec.Traits.Presentation.ProfileLabelStrategy)
				}
			},
		},
		{
			id: "OpenSearch",
			want: func(t *testing.T, spec source.TypeSpec) {
				t.Helper()
				if spec.Traits.Presentation.SchemaFidelity != source.SchemaFidelitySampled {
					t.Fatalf("expected OpenSearch schema fidelity %q, got %q", source.SchemaFidelitySampled, spec.Traits.Presentation.SchemaFidelity)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()

			entry, ok := dbcatalog.Find(tt.id)
			if !ok {
				t.Fatalf("expected database catalog entry for %q", tt.id)
			}

			spec, ok := BuildTypeSpec(DatabaseEntry{
				ID:             string(entry.ID),
				Label:          entry.Label,
				Connector:      string(entry.PluginType),
				Extra:          maps.Clone(entry.Extra),
				Fields:         FieldVisibility(entry.Fields),
				RequiredFields: FieldRequirements(entry.RequiredFields),
				IsAWSManaged:   entry.IsAWSManaged,
				SSLModes:       sourceSSLModes(entry.SSLModes),
			})
			if !ok {
				t.Fatalf("expected %q to map into the source catalog", tt.id)
			}

			tt.want(t, spec)
		})
	}
}

func sourceSSLModes(modes []ssl.SSLModeInfo) []source.SSLModeInfo {
	cloned := make([]source.SSLModeInfo, 0, len(modes))
	for _, mode := range modes {
		cloned = append(cloned, source.SSLModeInfo{
			Value:       string(mode.Value),
			Label:       mode.Label,
			Description: mode.Description,
		})
	}
	return cloned
}
