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

package duckdb

import "testing"

func TestDuckDBParseCheckConstraint(t *testing.T) {
	plugin := &DuckDBPlugin{}
	constraints := map[string]map[string]any{}

	plugin.parseCheckConstraint("status", "CHECK((status IN ('pending', 'completed', 'canceled')))", constraints)

	values, ok := constraints["status"]["check_values"].([]string)
	if !ok {
		t.Fatalf("expected status check_values, got %#v", constraints["status"]["check_values"])
	}
	if len(values) != 3 || values[0] != "pending" || values[1] != "completed" || values[2] != "canceled" {
		t.Fatalf("unexpected status check_values: %#v", values)
	}
}
