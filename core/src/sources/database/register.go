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

// Package database registers the built-in database-backed source types into the
// source registry.
package database

import (
	"maps"
	"slices"

	"github.com/clidey/whodb/core/src/dbcatalog"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func init() {
	Register()
}

// Register adds every database catalog entry that can be projected into the
// shared source-first catalog. It is safe to call multiple times.
func Register() {
	for _, entry := range dbcatalog.All() {
		spec, ok := sourcecatalog.BuildTypeSpec(sourcecatalog.DatabaseEntry{
			ID:             string(entry.ID),
			Label:          entry.Label,
			Connector:      string(entry.PluginType),
			Extra:          maps.Clone(entry.Extra),
			Fields:         sourcecatalog.FieldVisibility(entry.Fields),
			RequiredFields: sourcecatalog.FieldRequirements(entry.RequiredFields),
			IsAWSManaged:   entry.IsAWSManaged,
			SSLModes:       slices.Clone(entry.SSLModes),
		})
		if !ok {
			continue
		}
		source.RegisterType(spec)
	}
}
