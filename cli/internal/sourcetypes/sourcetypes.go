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
	"slices"
	"strconv"
	"strings"

	"github.com/clidey/whodb/cli/internal/bootstrap"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

var dbTypeSynonyms = map[string]string{
	"postgresql": "Postgres",
	"sqlite":     "Sqlite3",
	"cockroach":  "CockroachDB",
	"yugabyte":   "YugabyteDB",
	"quest":      "QuestDB",
	"elastic":    "ElasticSearch",
}

// IDs returns source type identifiers in CLI display order.
func IDs() []string {
	bootstrap.Ensure()
	return slices.Clone(sourcecatalog.IDs())
}

// Synonyms returns additional CLI-only aliases accepted for source types.
func Synonyms() []string {
	keys := make([]string, 0, len(dbTypeSynonyms))
	for key := range dbTypeSynonyms {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// Find resolves one source type using CLI aliases and case-insensitive matching.
func Find(input string) (source.TypeSpec, bool) {
	bootstrap.Ensure()

	if strings.TrimSpace(input) == "" {
		return source.TypeSpec{}, false
	}

	normalizedInput := normalizeKey(input)
	if alias, ok := dbTypeSynonyms[normalizedInput]; ok {
		return sourcecatalog.Find(alias)
	}

	for _, id := range sourcecatalog.IDs() {
		if normalizeKey(id) == normalizedInput {
			return sourcecatalog.Find(id)
		}
	}

	return source.TypeSpec{}, false
}

// Normalize returns the canonical source type ID for a CLI input value.
func Normalize(input string) string {
	spec, ok := Find(input)
	if !ok {
		return strings.TrimSpace(input)
	}
	return spec.ID
}

// ConnectionField returns one connection field definition by key.
func ConnectionField(input string, key string) (source.ConnectionField, bool) {
	spec, ok := Find(input)
	if !ok {
		return source.ConnectionField{}, false
	}
	return spec.ConnectionFieldByKey(key)
}

// ConnectionFieldRequired reports whether a connection field is required.
func ConnectionFieldRequired(input string, key string) bool {
	field, ok := ConnectionField(input, key)
	return ok && field.Required
}

// DefaultPort returns the declared default port for a source type when present.
func DefaultPort(input string) (int, bool) {
	field, ok := ConnectionField(input, "Port")
	if !ok {
		if IsFileTransport(input) {
			return 0, true
		}
		return 0, false
	}

	port, err := strconv.Atoi(strings.TrimSpace(field.DefaultValue))
	if err != nil {
		return 0, false
	}
	return port, true
}

// IsFileTransport reports whether a source uses file-backed transport.
func IsFileTransport(input string) bool {
	spec, ok := Find(input)
	return ok && spec.Traits.Connection.Transport == source.ConnectionTransportFile
}

// IsNetworkTransport reports whether a source uses direct network transport.
func IsNetworkTransport(input string) bool {
	spec, ok := Find(input)
	return ok && spec.Traits.Connection.Transport == source.ConnectionTransportNetwork
}

// SSLModes returns the declared SSL modes for a source type.
func SSLModes(input string) []source.SSLModeInfo {
	spec, ok := Find(input)
	if !ok {
		return nil
	}
	return slices.Clone(spec.SSLModes)
}

func normalizeKey(value string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}
