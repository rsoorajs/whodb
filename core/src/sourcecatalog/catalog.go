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
	"sync"

	"github.com/clidey/whodb/core/src/source"
)

// FamilySpec describes how one connector/plugin type should be projected into
// the public source-first catalog.
type FamilySpec struct {
	Category       source.Category
	Model          source.Model
	RootActions    []source.Action
	BrowsePath     []source.ObjectKind
	DefaultObject  source.ObjectKind
	GraphScopeKind *source.ObjectKind
	GraphSupported bool
	ObjectTypes    []source.ObjectType
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

const (
	connectorPostgres      = "Postgres"
	connectorCockroachDB   = "CockroachDB"
	connectorMySQL         = "MySQL"
	connectorMariaDB       = "MariaDB"
	connectorClickHouse    = "ClickHouse"
	connectorSqlite3       = "Sqlite3"
	connectorDuckDB        = "DuckDB"
	connectorMongoDB       = "MongoDB"
	connectorRedis         = "Redis"
	connectorMemcached     = "Memcached"
	connectorElasticSearch = "ElasticSearch"
)

var (
	customFamilySpecsMu sync.RWMutex
	customFamilySpecs   = map[string]FamilySpec{}
)

var familySpecs = map[string]FamilySpec{
	connectorPostgres: {
		Category:       source.CategoryDatabase,
		Model:          source.ModelRelational,
		BrowsePath:     []source.ObjectKind{objectKindDatabase, objectKindSchema, objectKindTable},
		DefaultObject:  objectKindTable,
		GraphScopeKind: ptr(objectKindSchema),
		GraphSupported: true,
		ObjectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			metadataObjectType(objectKindSchema, "Schema", "Schemas", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
			tabularReadOnlyObjectType(objectKindView, "View", "Views"),
		},
	},
	connectorCockroachDB: {
		Category:       source.CategoryDatabase,
		Model:          source.ModelRelational,
		BrowsePath:     []source.ObjectKind{objectKindDatabase, objectKindSchema, objectKindTable},
		DefaultObject:  objectKindTable,
		GraphScopeKind: ptr(objectKindSchema),
		GraphSupported: true,
		ObjectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			metadataObjectType(objectKindSchema, "Schema", "Schemas", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
			tabularReadOnlyObjectType(objectKindView, "View", "Views"),
		},
	},
	connectorMySQL: {
		Category:       source.CategoryDatabase,
		Model:          source.ModelRelational,
		BrowsePath:     []source.ObjectKind{objectKindDatabase, objectKindTable},
		DefaultObject:  objectKindTable,
		GraphScopeKind: ptr(objectKindDatabase),
		GraphSupported: true,
		ObjectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
			tabularReadOnlyObjectType(objectKindView, "View", "Views"),
		},
	},
	connectorMariaDB: {
		Category:       source.CategoryDatabase,
		Model:          source.ModelRelational,
		BrowsePath:     []source.ObjectKind{objectKindDatabase, objectKindTable},
		DefaultObject:  objectKindTable,
		GraphScopeKind: ptr(objectKindDatabase),
		GraphSupported: true,
		ObjectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
			tabularReadOnlyObjectType(objectKindView, "View", "Views"),
		},
	},
	connectorClickHouse: {
		Category:       source.CategoryDatabase,
		Model:          source.ModelRelational,
		BrowsePath:     []source.ObjectKind{objectKindDatabase, objectKindTable},
		DefaultObject:  objectKindTable,
		GraphScopeKind: ptr(objectKindDatabase),
		GraphSupported: true,
		ObjectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			tabularObjectType(objectKindTable, "Table", "Tables"),
		},
	},
	connectorSqlite3: {
		Category:      source.CategoryDatabase,
		Model:         source.ModelRelational,
		RootActions:   []source.Action{source.ActionBrowse, source.ActionCreateChild},
		BrowsePath:    []source.ObjectKind{objectKindTable},
		DefaultObject: objectKindTable,
		ObjectTypes: []source.ObjectType{
			tabularObjectType(objectKindTable, "Table", "Tables"),
		},
	},
	connectorDuckDB: {
		Category:      source.CategoryDatabase,
		Model:         source.ModelRelational,
		RootActions:   []source.Action{source.ActionBrowse, source.ActionCreateChild},
		BrowsePath:    []source.ObjectKind{objectKindTable},
		DefaultObject: objectKindTable,
		ObjectTypes: []source.ObjectType{
			tabularObjectType(objectKindTable, "Table", "Tables"),
		},
	},
	connectorMongoDB: {
		Category:       source.CategoryDatabase,
		Model:          source.ModelDocument,
		BrowsePath:     []source.ObjectKind{objectKindDatabase, objectKindColl},
		DefaultObject:  objectKindColl,
		GraphScopeKind: ptr(objectKindDatabase),
		GraphSupported: true,
		ObjectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			documentObjectType(objectKindColl, "Collection", "Collections"),
			metadataObjectType(objectKindIndex, "Index", "Indexes", false),
		},
	},
	connectorRedis: {
		Category:      source.CategoryCache,
		Model:         source.ModelKeyValue,
		BrowsePath:    []source.ObjectKind{objectKindDatabase, objectKindKey},
		DefaultObject: objectKindKey,
		ObjectTypes: []source.ObjectType{
			metadataObjectType(objectKindDatabase, "Database", "Databases", true),
			keyValueObjectType(objectKindKey, "Key", "Keys"),
		},
	},
	connectorMemcached: {
		Category:      source.CategoryCache,
		Model:         source.ModelKeyValue,
		BrowsePath:    []source.ObjectKind{objectKindItem},
		DefaultObject: objectKindItem,
		ObjectTypes: []source.ObjectType{
			keyValueReadOnlyObjectType(objectKindItem, "Item", "Items"),
		},
	},
	connectorElasticSearch: {
		Category:       source.CategorySearch,
		Model:          source.ModelSearch,
		BrowsePath:     []source.ObjectKind{objectKindIndex},
		DefaultObject:  objectKindIndex,
		GraphScopeKind: ptr(objectKindIndex),
		GraphSupported: true,
		ObjectTypes: []source.ObjectType{
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
	return source.RegisteredTypes()
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
	return source.FindType(id)
}

// FieldVisibility declares which standard connection fields are shown for one
// source-backed database type.
type FieldVisibility struct {
	Hostname   bool
	Username   bool
	Password   bool
	Database   bool
	SearchPath bool
}

// FieldRequirements declares which standard connection fields are required for
// one source-backed database type.
type FieldRequirements struct {
	Hostname bool
	Username bool
	Password bool
	Database bool
}

// DatabaseEntry contains the database-specific metadata needed to expose a
// database family member through the source catalog.
type DatabaseEntry struct {
	ID                 string
	Label              string
	Connector          string
	Extra              map[string]string
	Fields             FieldVisibility
	RequiredFields     FieldRequirements
	SupportsScratchpad bool
	IsAWSManaged       bool
	SSLModes           []source.SSLModeInfo
}

// RegisterFamilySpec registers a source-family mapping for a connector/plugin
// type so extension modules can expose additional source types through the
// shared source-first catalog.
func RegisterFamilySpec(connector string, spec FamilySpec) {
	customFamilySpecsMu.Lock()
	defer customFamilySpecsMu.Unlock()
	customFamilySpecs[connector] = cloneFamilySpec(spec)
}

// BuildTypeSpec converts one database-backed source registration into a public
// source type specification.
func BuildTypeSpec(entry DatabaseEntry) (source.TypeSpec, bool) {
	family, ok := familySpecFor(entry.Connector)
	if !ok {
		return source.TypeSpec{}, false
	}

	contract := source.Contract{
		Model:             family.Model,
		Surfaces:          buildSurfaces(entry, family),
		RootActions:       buildRootActions(family),
		BrowsePath:        slices.Clone(family.BrowsePath),
		DefaultObjectKind: family.DefaultObject,
		GraphScopeKind:    family.GraphScopeKind,
		ObjectTypes:       cloneObjectTypes(family.ObjectTypes),
	}

	return source.TypeSpec{
		ID:               entry.ID,
		Label:            entry.Label,
		DriverID:         "database",
		Connector:        entry.Connector,
		Category:         family.Category,
		ConnectionFields: buildConnectionFields(entry),
		Contract:         contract,
		IsAWSManaged:     entry.IsAWSManaged,
		SSLModes:         cloneSourceSSLModes(entry.SSLModes),
	}, true
}

func buildRootActions(family FamilySpec) []source.Action {
	if len(family.RootActions) > 0 {
		return slices.Clone(family.RootActions)
	}
	return []source.Action{source.ActionBrowse}
}

func familySpecFor(connector string) (FamilySpec, bool) {
	customFamilySpecsMu.RLock()
	spec, ok := customFamilySpecs[connector]
	customFamilySpecsMu.RUnlock()
	if ok {
		return cloneFamilySpec(spec), true
	}

	spec, ok = familySpecs[connector]
	if !ok {
		return FamilySpec{}, false
	}
	return cloneFamilySpec(spec), true
}

func buildSurfaces(entry DatabaseEntry, family FamilySpec) []source.Surface {
	surfaces := []source.Surface{source.SurfaceBrowser}
	if entry.SupportsScratchpad {
		surfaces = append(surfaces, source.SurfaceQuery, source.SurfaceChat)
	}
	if family.GraphSupported {
		surfaces = append(surfaces, source.SurfaceGraph)
	}
	return surfaces
}

func buildConnectionFields(entry DatabaseEntry) []source.ConnectionField {
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

func cloneSourceSSLModes(modes []source.SSLModeInfo) []source.SSLModeInfo {
	cloned := make([]source.SSLModeInfo, 0, len(modes))
	for _, mode := range modes {
		cloned = append(cloned, mode)
	}
	return cloned
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

func cloneFamilySpec(spec FamilySpec) FamilySpec {
	return FamilySpec{
		Category:       spec.Category,
		Model:          spec.Model,
		RootActions:    slices.Clone(spec.RootActions),
		BrowsePath:     slices.Clone(spec.BrowsePath),
		DefaultObject:  spec.DefaultObject,
		GraphScopeKind: spec.GraphScopeKind,
		GraphSupported: spec.GraphSupported,
		ObjectTypes:    cloneObjectTypes(spec.ObjectTypes),
	}
}

func ptr(kind source.ObjectKind) *source.ObjectKind {
	return &kind
}
