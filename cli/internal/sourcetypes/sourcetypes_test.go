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

package sourcetypes

import (
	"testing"

	"github.com/clidey/whodb/core/src/source"
)

func TestExplainMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  source.QueryExplainMode
		ok    bool
	}{
		{input: "Postgres", want: source.QueryExplainModeExplainAnalyze, ok: true},
		{input: "ClickHouse", want: source.QueryExplainModeExplainPipeline, ok: true},
		{input: "MongoDB", want: source.QueryExplainModeNone, ok: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, ok := ExplainMode(tt.input)
			if ok != tt.ok {
				t.Fatalf("expected ExplainMode(%q) ok=%t, got %t", tt.input, tt.ok, ok)
			}
			if got != tt.want {
				t.Fatalf("expected ExplainMode(%q)=%q, got %q", tt.input, tt.want, got)
			}
		})
	}
}
