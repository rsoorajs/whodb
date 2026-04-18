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

// Package adapters exposes source connectors backed by the existing database
// plugin layer.
package adapters

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
)

// DatabaseConnector opens source sessions backed by the existing database plugin layer.
type DatabaseConnector struct{}

// EngineCredentials converts source-first credentials into the legacy engine
// credential shape required by the current database plugins.
func EngineCredentials(spec source.TypeSpec, credentials *source.Credentials) *engine.Credentials {
	session := &DatabaseSession{
		spec:        spec,
		credentials: credentials,
	}
	return session.engineCredentials(nil)
}

// Open creates a new database-backed source session.
func (c *DatabaseConnector) Open(_ context.Context, spec source.TypeSpec, credentials *source.Credentials) (source.SourceSession, error) {
	plugin := src.MainEngine.Choose(engine.DatabaseType(spec.Connector))
	if plugin == nil {
		return nil, fmt.Errorf("unsupported source connector: %s", spec.Connector)
	}

	return &DatabaseSession{
		spec:        spec,
		plugin:      plugin,
		credentials: credentials,
	}, nil
}

// DatabaseSession adapts one database plugin instance to the source session interfaces.
type DatabaseSession struct {
	spec        source.TypeSpec
	plugin      *engine.Plugin
	credentials *source.Credentials
}

// Metadata returns source session metadata derived from plugin metadata.
func (s *DatabaseSession) Metadata(_ context.Context) (*source.SessionMetadata, error) {
	metadata := s.plugin.GetDatabaseMetadata()
	if metadata == nil {
		return &source.SessionMetadata{
			SourceType:     s.spec.ID,
			QueryLanguages: queryLanguagesForSpec(s.spec),
			AliasMap:       map[string]string{},
		}, nil
	}

	aliasMap := map[string]string{}
	for key, value := range metadata.AliasMap {
		aliasMap[key] = value
	}

	return &source.SessionMetadata{
		SourceType:      s.spec.ID,
		QueryLanguages:  queryLanguagesForSpec(s.spec),
		TypeDefinitions: slices.Clone(metadata.TypeDefinitions),
		Operators:       slices.Clone(metadata.Operators),
		AliasMap:        aliasMap,
	}, nil
}

// ConnectionFieldOptions returns dynamic options for a connection field.
func (s *DatabaseSession) ConnectionFieldOptions(_ context.Context, fieldKey string, values map[string]string) ([]string, error) {
	if !strings.EqualFold(fieldKey, "Database") {
		return []string{}, nil
	}

	config := engine.NewPluginConfig(s.engineCredentials(values))
	return s.plugin.GetDatabases(config)
}

// ListObjects lists child objects beneath the provided parent.
func (s *DatabaseSession) ListObjects(_ context.Context, parent *source.ObjectRef, kinds []source.ObjectKind) ([]source.Object, error) {
	nextKind, ok := s.nextKind(parent)
	if !ok {
		return []source.Object{}, nil
	}
	if len(kinds) > 0 && !slices.Contains(kinds, nextKind) {
		return []source.Object{}, nil
	}

	config := engine.NewPluginConfig(s.credentialsForRef(parent))

	switch nextKind {
	case source.ObjectKindDatabase:
		names, err := s.plugin.GetDatabases(config)
		if err != nil {
			return nil, err
		}
		objects := make([]source.Object, 0, len(names))
		for _, name := range names {
			objects = append(objects, s.makeContainerObject(parent, nextKind, name, nil))
		}
		return objects, nil
	case source.ObjectKindSchema:
		names, err := s.plugin.GetAllSchemas(config)
		if err != nil {
			return nil, err
		}
		objects := make([]source.Object, 0, len(names))
		for _, name := range names {
			objects = append(objects, s.makeContainerObject(parent, nextKind, name, nil))
		}
		return objects, nil
	default:
		namespace := s.namespaceForRef(parent)
		units, err := s.plugin.GetStorageUnits(config, namespace)
		if err != nil {
			return nil, err
		}
		objects := make([]source.Object, 0, len(units))
		for _, unit := range units {
			kind := s.kindForUnit(nextKind, unit)
			objectType, _ := s.spec.Contract.ObjectTypeForKind(kind)
			objects = append(objects, source.Object{
				Ref: source.ObjectRef{
					Kind: kind,
					Path: appendPath(parent, unit.Name),
				},
				Kind:        kind,
				Name:        unit.Name,
				Path:        appendPath(parent, unit.Name),
				HasChildren: s.hasChildren(kind),
				Actions:     slices.Clone(objectType.Actions),
				Metadata:    slices.Clone(unit.Attributes),
			})
		}
		return objects, nil
	}
}

// GetObject loads one object by reference.
func (s *DatabaseSession) GetObject(ctx context.Context, ref source.ObjectRef) (*source.Object, error) {
	parent := parentForRef(ref)
	objects, err := s.ListObjects(ctx, parent, []source.ObjectKind{ref.Kind})
	if err != nil {
		return nil, err
	}

	for _, object := range objects {
		if object.Kind == ref.Kind && slices.Equal(object.Path, ref.Path) {
			objectCopy := object
			return &objectCopy, nil
		}
	}

	return nil, fmt.Errorf("source object not found")
}

// ReadRows returns rows for a tabular source object.
func (s *DatabaseSession) ReadRows(_ context.Context, ref source.ObjectRef, where *model.WhereCondition, sort []*model.SortCondition, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	config := engine.NewPluginConfig(s.credentialsForRef(&ref))
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return nil, err
	}

	return s.plugin.GetRows(config, &engine.GetRowsRequest{
		Schema:      namespace,
		StorageUnit: name,
		Where:       where,
		Sort:        sort,
		PageSize:    pageSize,
		PageOffset:  pageOffset,
	})
}

// Columns returns columns for one source object.
func (s *DatabaseSession) Columns(_ context.Context, ref source.ObjectRef) ([]engine.Column, error) {
	config := engine.NewPluginConfig(s.credentialsForRef(&ref))
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return nil, err
	}
	return s.plugin.GetColumnsForTable(config, namespace, name)
}

// ColumnsBatch returns columns for multiple source objects.
func (s *DatabaseSession) ColumnsBatch(ctx context.Context, refs []source.ObjectRef) ([]source.ObjectColumns, error) {
	results := make([]source.ObjectColumns, 0, len(refs))
	for _, ref := range refs {
		columns, err := s.Columns(ctx, ref)
		if err != nil {
			continue
		}
		results = append(results, source.ObjectColumns{
			Ref:     ref,
			Columns: columns,
		})
	}
	return results, nil
}

// RunQuery executes a query against the source session.
func (s *DatabaseSession) RunQuery(_ context.Context, query string, params ...any) (*engine.GetRowsResult, error) {
	config := engine.NewPluginConfig(s.engineCredentials(nil))
	return s.plugin.RawExecute(config, query, params...)
}

// ReadGraph returns graph data for a source scope.
func (s *DatabaseSession) ReadGraph(_ context.Context, ref *source.ObjectRef) ([]engine.GraphUnit, error) {
	config := engine.NewPluginConfig(s.credentialsForRef(ref))
	scope := ""
	if ref != nil {
		scope = s.graphScopeForRef(*ref)
	}
	return s.plugin.GetGraph(config, scope)
}

// Reply runs AI chat against the source session.
func (s *DatabaseSession) Reply(_ context.Context, ref *source.ObjectRef, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	config := engine.NewPluginConfig(s.credentialsForRef(ref))
	scope := ""
	if ref != nil {
		scope = s.graphScopeForRef(*ref)
	}
	return s.plugin.Chat(config, scope, previousConversation, query)
}

// CreateObject creates a new source object.
func (s *DatabaseSession) CreateObject(_ context.Context, parent *source.ObjectRef, name string, fields []engine.Record) (bool, error) {
	config := engine.NewPluginConfig(s.credentialsForRef(parent))
	namespace := s.namespaceForRef(parent)
	return s.plugin.AddStorageUnit(config, namespace, name, fields)
}

// UpdateObject updates an existing source object.
func (s *DatabaseSession) UpdateObject(_ context.Context, ref source.ObjectRef, values map[string]string, updatedColumns []string) (bool, error) {
	config := engine.NewPluginConfig(s.credentialsForRef(&ref))
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return false, err
	}
	return s.plugin.UpdateStorageUnit(config, namespace, name, values, updatedColumns)
}

// AddRow inserts a row into a source object.
func (s *DatabaseSession) AddRow(_ context.Context, ref source.ObjectRef, values []engine.Record) (bool, error) {
	config := engine.NewPluginConfig(s.credentialsForRef(&ref))
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return false, err
	}
	return s.plugin.AddRow(config, namespace, name, values)
}

// DeleteRow deletes a row from a source object.
func (s *DatabaseSession) DeleteRow(_ context.Context, ref source.ObjectRef, values map[string]string) (bool, error) {
	config := engine.NewPluginConfig(s.credentialsForRef(&ref))
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return false, err
	}
	return s.plugin.DeleteRow(config, namespace, name, values)
}

func (s *DatabaseSession) engineCredentials(values map[string]string) *engine.Credentials {
	mergedValues := s.credentials.CloneValues()
	for _, field := range s.spec.ConnectionFields {
		if field.DefaultValue == "" {
			continue
		}
		if _, ok := mergedValues[field.Key]; !ok {
			mergedValues[field.Key] = field.DefaultValue
		}
	}
	for key, value := range values {
		mergedValues[key] = value
	}

	engineCredentials := &engine.Credentials{
		Id:          s.credentials.ID,
		Type:        s.spec.ID,
		AccessToken: s.credentials.AccessToken,
		IsProfile:   s.credentials.IsProfile,
	}

	knownFields := map[string]bool{}
	for _, field := range s.spec.ConnectionFields {
		value := mergedValues[field.Key]
		if value == "" {
			continue
		}
		knownFields[field.Key] = true

		switch field.CredentialField {
		case source.CredentialFieldHostname:
			engineCredentials.Hostname = value
		case source.CredentialFieldUsername:
			engineCredentials.Username = value
		case source.CredentialFieldPassword:
			engineCredentials.Password = value
		case source.CredentialFieldDatabase:
			engineCredentials.Database = value
		case source.CredentialFieldAdvanced:
			advancedKey := field.AdvancedKey
			if advancedKey == "" {
				advancedKey = field.Key
			}
			engineCredentials.Advanced = append(engineCredentials.Advanced, engine.Record{
				Key:   advancedKey,
				Value: value,
			})
		}
	}

	for key, value := range mergedValues {
		if value == "" || knownFields[key] {
			continue
		}
		engineCredentials.Advanced = append(engineCredentials.Advanced, engine.Record{
			Key:   key,
			Value: value,
		})
	}

	return engineCredentials
}

func (s *DatabaseSession) credentialsForRef(ref *source.ObjectRef) *engine.Credentials {
	engineCredentials := s.engineCredentials(nil)
	if ref == nil {
		return engineCredentials
	}

	if databaseName := s.valueForKind(ref.Path, source.ObjectKindDatabase); databaseName != "" {
		engineCredentials.Database = databaseName
	}

	return engineCredentials
}

func (s *DatabaseSession) nextKind(parent *source.ObjectRef) (source.ObjectKind, bool) {
	depth := 0
	if parent != nil {
		depth = len(parent.Path)
	}
	if depth >= len(s.spec.Contract.BrowsePath) {
		return "", false
	}
	return s.spec.Contract.BrowsePath[depth], true
}

func (s *DatabaseSession) namespaceForRef(ref *source.ObjectRef) string {
	if ref == nil {
		return ""
	}

	defaultIndex := slices.Index(s.spec.Contract.BrowsePath, s.spec.Contract.DefaultObjectKind)
	if defaultIndex <= 0 || defaultIndex-1 >= len(ref.Path) {
		return ""
	}
	return ref.Path[defaultIndex-1]
}

func (s *DatabaseSession) graphScopeForRef(ref source.ObjectRef) string {
	if s.spec.Contract.GraphScopeKind == nil {
		return ""
	}
	return s.valueForKind(ref.Path, *s.spec.Contract.GraphScopeKind)
}

func (s *DatabaseSession) valueForKind(path []string, kind source.ObjectKind) string {
	index := slices.Index(s.spec.Contract.BrowsePath, kind)
	if index < 0 || index >= len(path) {
		return ""
	}
	return path[index]
}

func (s *DatabaseSession) makeContainerObject(parent *source.ObjectRef, kind source.ObjectKind, name string, metadata []engine.Record) source.Object {
	objectType, _ := s.spec.Contract.ObjectTypeForKind(kind)
	path := appendPath(parent, name)
	return source.Object{
		Ref: source.ObjectRef{
			Kind: kind,
			Path: path,
		},
		Kind:        kind,
		Name:        name,
		Path:        path,
		HasChildren: s.hasChildren(kind),
		Actions:     slices.Clone(objectType.Actions),
		Metadata:    slices.Clone(metadata),
	}
}

func (s *DatabaseSession) hasChildren(kind source.ObjectKind) bool {
	index := slices.Index(s.spec.Contract.BrowsePath, kind)
	return index >= 0 && index < len(s.spec.Contract.BrowsePath)-1
}

func (s *DatabaseSession) kindForUnit(defaultKind source.ObjectKind, unit engine.StorageUnit) source.ObjectKind {
	for _, attribute := range unit.Attributes {
		if !strings.EqualFold(attribute.Key, "Type") {
			continue
		}

		switch strings.ToUpper(strings.TrimSpace(attribute.Value)) {
		case "TABLE":
			return source.ObjectKindTable
		case "VIEW":
			return source.ObjectKindView
		case "COLLECTION":
			return source.ObjectKindCollection
		case "INDEX":
			return source.ObjectKindIndex
		case "KEY":
			return source.ObjectKindKey
		case "ITEM":
			return source.ObjectKindItem
		}
	}
	return defaultKind
}

func (s *DatabaseSession) validateObject(config *engine.PluginConfig, namespace string, name string) error {
	exists, err := s.plugin.StorageUnitExists(config, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to validate source object: %w", err)
	}
	if !exists {
		return fmt.Errorf("source object %q not found", name)
	}
	return nil
}

func queryLanguagesForSpec(spec source.TypeSpec) []string {
	if spec.Contract.SupportsSurface(source.SurfaceQuery) {
		return []string{"sql"}
	}
	return []string{}
}

func appendPath(parent *source.ObjectRef, name string) []string {
	if parent == nil {
		return []string{name}
	}
	path := slices.Clone(parent.Path)
	path = append(path, name)
	return path
}

func parentForRef(ref source.ObjectRef) *source.ObjectRef {
	if len(ref.Path) == 0 {
		return nil
	}

	if len(ref.Path) == 1 {
		return nil
	}

	return &source.ObjectRef{
		Kind: ref.Kind,
		Path: slices.Clone(ref.Path[:len(ref.Path)-1]),
	}
}

func objectName(ref source.ObjectRef) string {
	if len(ref.Path) == 0 {
		return ""
	}
	return ref.Path[len(ref.Path)-1]
}
