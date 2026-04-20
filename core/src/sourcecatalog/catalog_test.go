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
