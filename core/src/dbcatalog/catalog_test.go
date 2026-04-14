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
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestFindReturnsAliasPluginType(t *testing.T) {
	entry, ok := Find("FerretDB")
	if !ok {
		t.Fatal("expected FerretDB catalog entry")
	}

	if entry.PluginType != engine.DatabaseType_MongoDB {
		t.Fatalf("expected FerretDB to resolve to MongoDB plugin, got %q", entry.PluginType)
	}
}

func TestDefaultPortUsesCatalogOverrides(t *testing.T) {
	port, ok := DefaultPort("QuestDB")
	if !ok {
		t.Fatal("expected QuestDB default port")
	}

	if port != 8812 {
		t.Fatalf("expected QuestDB port 8812, got %d", port)
	}
}

func TestManagedServiceEntryRetainsFlags(t *testing.T) {
	entry, ok := Find("ElastiCache")
	if !ok {
		t.Fatal("expected ElastiCache catalog entry")
	}

	if !entry.IsAWSManaged {
		t.Fatal("expected ElastiCache to be marked as AWS managed")
	}

	if entry.Extra["TLS"] != "true" {
		t.Fatalf("expected ElastiCache TLS default, got %q", entry.Extra["TLS"])
	}
}
