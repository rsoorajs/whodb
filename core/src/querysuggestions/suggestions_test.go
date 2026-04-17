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

package querysuggestions

import (
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestFromStorageUnitsCapsAtThree(t *testing.T) {
	suggestions := FromStorageUnits([]engine.StorageUnit{
		{Name: "users"},
		{Name: "orders"},
		{Name: "payments"},
		{Name: "ignored"},
	})

	if len(suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %#v", suggestions)
	}
	if !strings.Contains(suggestions[0].Description, "users") {
		t.Fatalf("expected first suggestion to mention users, got %#v", suggestions[0])
	}
	if suggestions[1].Category != "AGGREGATE" {
		t.Fatalf("expected second suggestion category AGGREGATE, got %#v", suggestions[1])
	}
	for _, suggestion := range suggestions {
		if strings.Contains(suggestion.Description, "ignored") {
			t.Fatalf("did not expect ignored table to appear, got %#v", suggestions)
		}
	}
}

func TestFromStorageUnitsEmpty(t *testing.T) {
	suggestions := FromStorageUnits(nil)
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions, got %#v", suggestions)
	}
}
