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

// Package sourcecatalog exposes the source-first catalog consumed by the public
// GraphQL API and frontend.
package sourcecatalog

import (
	"slices"
	"sort"
	"strings"

	"github.com/clidey/whodb/core/src/dbcatalog"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
)

type familySpec struct {
	category       source.Category
	model          source.Model
	browsePath     []source.ObjectKind
	defaultObject  source.ObjectKind
	graphScopeKind *source.ObjectKind
	graphSupported bool
	objectTypes    []source.ObjectType
}

var (
	objectKindDatabase = source.ObjectKindDatabase
	objectKindSchema   = source.ObjectKindSchema
	objectKindTable    = source.ObjectKindTable
	objectKindView     = source.ObjectKindView
	objectKindIndex    = source.ObjectKindIndex
	objectKindItem     = source.ObjectKindItem
	objectKindKey      = source.ObjectKindKey
	objectKindColl     = source.ObjectKindCollection
)

var familySpecs = map[string]familySpec{
	string(engine.DatabaseType_Postgres): {
		category:       source.CategoryDatabase,
		model:          source.ModelRelational,
		browsePath:     []source.ObjectKind{objectKindDatabase, objectKindSchema, objectKindTable},
		defaultObject:  objectKindTable,
		graphScopeKind: ptr(objectKindSchema),
		graphSupported: true,
		objectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			metadataObjectType(objectKindSchema, "Schema", "Schemas", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
			tabularReadOnlyObjectType(objectKindView, "View", "Views"),
		},
	},
	string(engine.DatabaseType_CockroachDB): {
		category:       source.CategoryDatabase,
		model:          source.ModelRelational,
		browsePath:     []source.ObjectKind{objectKindDatabase, objectKindSchema, objectKindTable},
		defaultObject:  objectKindTable,
		graphScopeKind: ptr(objectKindSchema),
		graphSupported: true,
		objectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			metadataObjectType(objectKindSchema, "Schema", "Schemas", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
			tabularReadOnlyObjectType(objectKindView, "View", "Views"),
		},
	},
	string(engine.DatabaseType_MySQL): {
		category:       source.CategoryDatabase,
		model:          source.ModelRelational,
		browsePath:     []source.ObjectKind{objectKindDatabase, objectKindTable},
		defaultObject:  objectKindTable,
		graphScopeKind: ptr(objectKindDatabase),
		graphSupported: true,
		objectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
			tabularReadOnlyObjectType(objectKindView, "View", "Views"),
		},
	},
	string(engine.DatabaseType_MariaDB): {
		category:       source.CategoryDatabase,
		model:          source.ModelRelational,
		browsePath:     []source.ObjectKind{objectKindDatabase, objectKindTable},
		defaultObject:  objectKindTable,
		graphScopeKind: ptr(objectKindDatabase),
		graphSupported: true,
		objectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
			tabularReadOnlyObjectType(objectKindView, "View", "Views"),
		},
	},
	string(engine.DatabaseType_ClickHouse): {
		category:       source.CategoryDatabase,
		model:          source.ModelRelational,
		browsePath:     []source.ObjectKind{objectKindDatabase, objectKindTable},
		defaultObject:  objectKindTable,
		graphScopeKind: ptr(objectKindDatabase),
		graphSupported: true,
		objectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
		},
	},
	string(engine.DatabaseType_Sqlite3): {
		category:      source.CategoryDatabase,
		model:         source.ModelRelational,
		browsePath:    []source.ObjectKind{objectKindTable},
		defaultObject: objectKindTable,
		objectTypes: []source.ObjectType{
			tabularObjectType(objectKindTable, "Table", "Tables"),
		},
	},
	string(engine.DatabaseType_DuckDB): {
		category:      source.CategoryDatabase,
		model:         source.ModelRelational,
		browsePath:    []source.ObjectKind{objectKindTable},
		defaultObject: objectKindTable,
		objectTypes: []source.ObjectType{
			tabularObjectType(objectKindTable, "Table", "Tables"),
		},
	},
	string(engine.DatabaseType_MongoDB): {
		category:       source.CategoryDatabase,
		model:          source.ModelDocument,
		browsePath:     []source.ObjectKind{objectKindDatabase, objectKindColl},
		defaultObject:  objectKindColl,
		graphScopeKind: ptr(objectKindDatabase),
		graphSupported: true,
		objectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			documentObjectType(objectKindColl, "Collection", "Collections"),
			metadataObjectType(objectKindIndex, "Index", "Indexes", false),
		},
	},
	string(engine.DatabaseType_Redis): {
		category:      source.CategoryCache,
		model:         source.ModelKeyValue,
		browsePath:    []source.ObjectKind{objectKindDatabase, objectKindKey},
		defaultObject: objectKindKey,
		objectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			keyValueObjectType(objectKindKey, "Key", "Keys"),
		},
	},
	string(engine.DatabaseType_Memcached): {
		category:      source.CategoryCache,
		model:         source.ModelKeyValue,
		browsePath:    []source.ObjectKind{objectKindItem},
		defaultObject: objectKindItem,
		objectTypes: []source.ObjectType{
			keyValueReadOnlyObjectType(objectKindItem, "Item", "Items"),
		},
	},
	string(engine.DatabaseType_ElasticSearch): {
		category:       source.CategorySearch,
		model:          source.ModelSearch,
		browsePath:     []source.ObjectKind{objectKindIndex},
		defaultObject:  objectKindIndex,
		graphScopeKind: ptr(objectKindIndex),
		graphSupported: true,
		objectTypes: []source.ObjectType{
			documentObjectType(objectKindIndex, "Index", "Indices"),
		},
	},
}

var extraFieldOrder = []string{
	"Port",
	"Parse Time",
	"Loc",
	"Allow clear text passwords",
	"Search Path",
	"HTTP Protocol",
	"Readonly",
	"Debug",
	"URL Params",
	"DNS Enabled",
}

// All returns the full source catalog in UI order.
func All() []source.TypeSpec {
	entries := dbcatalog.All()
	specs := make([]source.TypeSpec, 0, len(entries))
	for _, entry := range entries {
		spec, ok := mapEntry(entry)
		if !ok {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

// IDs returns source type identifiers in UI order.
func IDs() []string {
	specs := All()
	ids := make([]string, 0, len(specs))
	for _, spec := range specs {
		ids = append(ids, spec.ID)
	}
	return ids
}

// Find looks up a source type by id using a case-insensitive match.
func Find(id string) (source.TypeSpec, bool) {
	for _, spec := range All() {
		if strings.EqualFold(spec.ID, id) {
			return cloneSpec(spec), true
		}
	}
	return source.TypeSpec{}, false
}

func mapEntry(entry dbcatalog.ConnectableDatabase) (source.TypeSpec, bool) {
	family, ok := familySpecs[string(entry.PluginType)]
	if !ok {
		return source.TypeSpec{}, false
	}

	contract := source.Contract{
		Model:             family.model,
		Surfaces:          buildSurfaces(entry, family),
		BrowsePath:        slices.Clone(family.browsePath),
		DefaultObjectKind: family.defaultObject,
		GraphScopeKind:    family.graphScopeKind,
		ObjectTypes:       cloneObjectTypes(family.objectTypes),
	}

	return source.TypeSpec{
		ID:               string(entry.ID),
		Label:            entry.Label,
		Connector:        string(entry.PluginType),
		Category:         family.category,
		ConnectionFields: buildConnectionFields(entry),
		Contract:         contract,
		IsAWSManaged:     entry.IsAWSManaged,
		SSLModes:         slices.Clone(entry.SSLModes),
	}, true
}

func buildSurfaces(entry dbcatalog.ConnectableDatabase, family familySpec) []source.Surface {
	surfaces := []source.Surface{source.SurfaceBrowser}
	if entry.SupportsScratchpad {
		surfaces = append(surfaces, source.SurfaceQuery, source.SurfaceChat)
	}
	if family.graphSupported {
		surfaces = append(surfaces, source.SurfaceGraph)
	}
	return surfaces
}

func buildConnectionFields(entry dbcatalog.ConnectableDatabase) []source.ConnectionField {
	fields := make([]source.ConnectionField, 0, len(entry.Extra)+5)

	if entry.Fields.Hostname {
		fields = append(fields, source.ConnectionField{
			Key:             "Hostname",
			Kind:            source.ConnectionFieldKindText,
			Section:         source.ConnectionFieldSectionPrimary,
			Required:        entry.RequiredFields.Hostname,
			LabelKey:        "hostName",
			PlaceholderKey:  "enterHostName",
			CredentialField: source.CredentialFieldHostname,
		})
	}

	if entry.Fields.Username {
		fields = append(fields, source.ConnectionField{
			Key:             "Username",
			Kind:            source.ConnectionFieldKindText,
			Section:         source.ConnectionFieldSectionPrimary,
			Required:        entry.RequiredFields.Username,
			LabelKey:        "username",
			PlaceholderKey:  "enterUsername",
			CredentialField: source.CredentialFieldUsername,
		})
	}

	if entry.Fields.Password {
		fields = append(fields, source.ConnectionField{
			Key:             "Password",
			Kind:            source.ConnectionFieldKindPassword,
			Section:         source.ConnectionFieldSectionPrimary,
			Required:        entry.RequiredFields.Password,
			LabelKey:        "password",
			PlaceholderKey:  "enterPassword",
			CredentialField: source.CredentialFieldPassword,
		})
	}

	if entry.Fields.Database {
		fileBased := !entry.Fields.Hostname
		kind := source.ConnectionFieldKindText
		placeholderKey := "enterDatabase"
		supportsOptions := false
		if fileBased {
			kind = source.ConnectionFieldKindFilePath
			placeholderKey = "selectOrEnterDatabasePath"
			supportsOptions = true
		}

		fields = append(fields, source.ConnectionField{
			Key:             "Database",
			Kind:            kind,
			Section:         source.ConnectionFieldSectionPrimary,
			Required:        entry.RequiredFields.Database,
			LabelKey:        "databaseType",
			PlaceholderKey:  placeholderKey,
			SupportsOptions: supportsOptions,
			CredentialField: source.CredentialFieldDatabase,
		})
	}

	if entry.Fields.SearchPath {
		fields = append(fields, source.ConnectionField{
			Key:             "Search Path",
			Kind:            source.ConnectionFieldKindText,
			Section:         source.ConnectionFieldSectionPrimary,
			LabelKey:        "advancedFields.searchPath",
			PlaceholderKey:  "enterSearchPath",
			CredentialField: source.CredentialFieldAdvanced,
			AdvancedKey:     "Search Path",
		})
	}

	for _, key := range orderedExtraKeys(entry.Extra) {
		if key == "Search Path" && entry.Fields.SearchPath {
			continue
		}

		fields = append(fields, source.ConnectionField{
			Key:             key,
			Kind:            source.ConnectionFieldKindText,
			Section:         source.ConnectionFieldSectionAdvanced,
			LabelKey:        "advancedFields." + camelCaseKey(key),
			DefaultValue:    entry.Extra[key],
			CredentialField: source.CredentialFieldAdvanced,
			AdvancedKey:     key,
		})
	}

	return fields
}

func orderedExtraKeys(extra map[string]string) []string {
	keys := make([]string, 0, len(extra))
	seen := map[string]bool{}
	for _, key := range extraFieldOrder {
		if _, ok := extra[key]; ok {
			keys = append(keys, key)
			seen[key] = true
		}
	}

	remaining := make([]string, 0, len(extra))
	for key := range extra {
		if seen[key] {
			continue
		}
		remaining = append(remaining, key)
	}
	sort.Strings(remaining)
	return append(keys, remaining...)
}

func camelCaseKey(key string) string {
	parts := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(key))
	if len(parts) == 0 {
		return ""
	}

	for i := range parts {
		lower := strings.ToLower(parts[i])
		if i == 0 {
			parts[i] = lower
			continue
		}
		parts[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}

	return strings.Join(parts, "")
}

func metadataObjectType(kind source.ObjectKind, singular string, plural string, createChild bool) source.ObjectType {
	actions := []source.Action{source.ActionBrowse}
	if createChild {
		actions = append(actions, source.ActionCreateChild)
	}

	return source.ObjectType{
		Kind:          kind,
		DataShape:     source.DataShapeMetadata,
		Actions:       actions,
		Views:         []source.View{source.ViewMetadata},
		SingularLabel: singular,
		PluralLabel:   plural,
	}
}

func tabularObjectType(kind source.ObjectKind, singular string, plural string) source.ObjectType {
	return source.ObjectType{
		Kind:      kind,
		DataShape: source.DataShapeTabular,
		Actions: []source.Action{
			source.ActionInspect,
			source.ActionViewRows,
			source.ActionInsertData,
			source.ActionUpdateData,
			source.ActionImportData,
			source.ActionGenerateMockData,
		},
		Views:         []source.View{source.ViewGrid, source.ViewMetadata},
		SingularLabel: singular,
		PluralLabel:   plural,
	}
}

func tabularReadOnlyObjectType(kind source.ObjectKind, singular string, plural string) source.ObjectType {
	return source.ObjectType{
		Kind:      kind,
		DataShape: source.DataShapeTabular,
		Actions: []source.Action{
			source.ActionInspect,
			source.ActionViewRows,
			source.ActionViewDefinition,
		},
		Views:         []source.View{source.ViewGrid, source.ViewMetadata},
		SingularLabel: singular,
		PluralLabel:   plural,
	}
}

func documentObjectType(kind source.ObjectKind, singular string, plural string) source.ObjectType {
	return source.ObjectType{
		Kind:      kind,
		DataShape: source.DataShapeDocument,
		Actions: []source.Action{
			source.ActionInspect,
			source.ActionViewRows,
			source.ActionInsertData,
			source.ActionUpdateData,
		},
		Views:         []source.View{source.ViewGrid, source.ViewJSON, source.ViewMetadata},
		SingularLabel: singular,
		PluralLabel:   plural,
	}
}

func keyValueObjectType(kind source.ObjectKind, singular string, plural string) source.ObjectType {
	return source.ObjectType{
		Kind:      kind,
		DataShape: source.DataShapeContent,
		Actions: []source.Action{
			source.ActionInspect,
			source.ActionViewRows,
			source.ActionInsertData,
			source.ActionUpdateData,
		},
		Views:         []source.View{source.ViewGrid, source.ViewMetadata},
		SingularLabel: singular,
		PluralLabel:   plural,
	}
}

func keyValueReadOnlyObjectType(kind source.ObjectKind, singular string, plural string) source.ObjectType {
	return source.ObjectType{
		Kind:      kind,
		DataShape: source.DataShapeContent,
		Actions: []source.Action{
			source.ActionInspect,
			source.ActionViewRows,
		},
		Views:         []source.View{source.ViewGrid, source.ViewMetadata},
		SingularLabel: singular,
		PluralLabel:   plural,
	}
}

func cloneSpec(spec source.TypeSpec) source.TypeSpec {
	return source.TypeSpec{
		ID:               spec.ID,
		Label:            spec.Label,
		Connector:        spec.Connector,
		Category:         spec.Category,
		ConnectionFields: slices.Clone(spec.ConnectionFields),
		Contract: source.Contract{
			Model:             spec.Contract.Model,
			Surfaces:          slices.Clone(spec.Contract.Surfaces),
			BrowsePath:        slices.Clone(spec.Contract.BrowsePath),
			DefaultObjectKind: spec.Contract.DefaultObjectKind,
			GraphScopeKind:    spec.Contract.GraphScopeKind,
			ObjectTypes:       cloneObjectTypes(spec.Contract.ObjectTypes),
		},
		IsAWSManaged: spec.IsAWSManaged,
		SSLModes:     slices.Clone(spec.SSLModes),
	}
}

func cloneObjectTypes(objectTypes []source.ObjectType) []source.ObjectType {
	cloned := make([]source.ObjectType, 0, len(objectTypes))
	for _, objectType := range objectTypes {
		cloned = append(cloned, source.ObjectType{
			Kind:          objectType.Kind,
			DataShape:     objectType.DataShape,
			Actions:       slices.Clone(objectType.Actions),
			Views:         slices.Clone(objectType.Views),
			SingularLabel: objectType.SingularLabel,
			PluralLabel:   objectType.PluralLabel,
		})
	}
	return cloned
}

func ptr(kind source.ObjectKind) *source.ObjectKind {
	return &kind
}
