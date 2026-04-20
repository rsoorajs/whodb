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

package adapters

import (
	"context"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
)

func TestDatabaseSessionRunQueryRejectsUnsupportedSurface(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("MongoDB"))
	mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
		t.Fatalf("expected query execution to be blocked by the source contract")
		return nil, nil
	}

	session := newTestDatabaseSession(testTypeSpec("MongoDB", []source.Surface{source.SurfaceBrowser}), mock)

	_, err := session.RunQuery(context.Background(), "SELECT 1")
	if err == nil || !strings.Contains(err.Error(), "querying") {
		t.Fatalf("expected querying error, got %v", err)
	}
}

func TestDatabaseSessionReadGraphRejectsUnsupportedSurface(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("MySQL"))
	mock.GetGraphFunc = func(*engine.PluginConfig, string) ([]engine.GraphUnit, error) {
		t.Fatalf("expected graph reads to be blocked by the source contract")
		return nil, nil
	}

	session := newTestDatabaseSession(testTypeSpec("MySQL", []source.Surface{source.SurfaceBrowser, source.SurfaceQuery}), mock)

	_, err := session.ReadGraph(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "graph") {
		t.Fatalf("expected graph error, got %v", err)
	}
}

func TestDatabaseSessionReplyRejectsUnsupportedSurface(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Redis"))
	mock.ChatFunc = func(*engine.PluginConfig, string, string, string) ([]*engine.ChatMessage, error) {
		t.Fatalf("expected chat to be blocked by the source contract")
		return nil, nil
	}

	session := newTestDatabaseSession(testTypeSpec("Redis", []source.Surface{source.SurfaceBrowser}), mock)

	_, err := session.Reply(context.Background(), nil, "", "hello")
	if err == nil || !strings.Contains(err.Error(), "chat") {
		t.Fatalf("expected chat error, got %v", err)
	}
}

func TestDatabaseSessionReadRowsRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Oracle"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		t.Fatalf("expected row reads to be blocked before object validation")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Oracle",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindTable: {source.ActionInspect},
		},
	), mock)

	_, err := session.ReadRows(context.Background(), testTableRef(), nil, nil, 10, 0)
	if err == nil || !strings.Contains(err.Error(), "viewing rows") {
		t.Fatalf("expected row-view error, got %v", err)
	}
}

func TestDatabaseSessionAddRowRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Memcached"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		t.Fatalf("expected inserts to be blocked before object validation")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Memcached",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindItem: {source.ActionInspect, source.ActionViewRows},
		},
	), mock)

	_, err := session.AddRow(context.Background(), source.NewObjectRef(source.ObjectKindItem, []string{"item-1"}), nil)
	if err == nil || !strings.Contains(err.Error(), "inserting data") {
		t.Fatalf("expected insert error, got %v", err)
	}
}

func TestDatabaseSessionImportDataRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("ElasticSearch"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		t.Fatalf("expected imports to be blocked before object validation")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"ElasticSearch",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindIndex: {source.ActionInspect, source.ActionViewRows, source.ActionInsertData, source.ActionUpdateData},
		},
	), mock)

	err := session.ImportData(context.Background(), source.NewObjectRef(source.ObjectKindIndex, []string{"events"}), source.ImportRequest{})
	if err == nil || !strings.Contains(err.Error(), "importing data") {
		t.Fatalf("expected import error, got %v", err)
	}
}

func TestDatabaseSessionGenerateMockDataRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Redis"))
	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Redis",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindKey: {source.ActionInspect, source.ActionViewRows, source.ActionInsertData, source.ActionUpdateData},
		},
	), mock)

	_, err := session.GenerateMockData(context.Background(), source.NewObjectRef(source.ObjectKindKey, []string{"0", "users"}), 10, 0, false)
	if err == nil || !strings.Contains(err.Error(), "generating mock data") {
		t.Fatalf("expected mock-data error, got %v", err)
	}
}

func TestDatabaseSessionCreateObjectRejectsUnsupportedParentAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Oracle"))
	mock.AddStorageUnitFunc = func(*engine.PluginConfig, string, string, []engine.Record) (bool, error) {
		t.Fatalf("expected object creation to be blocked by the source contract")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Oracle",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindSchema: {source.ActionBrowse},
			source.ObjectKindTable:  {source.ActionInspect, source.ActionViewRows},
		},
	), mock)

	_, err := session.CreateObject(context.Background(), testSchemaRef(), "users", nil)
	if err == nil || !strings.Contains(err.Error(), "creating child objects") {
		t.Fatalf("expected create-child error, got %v", err)
	}
}

func TestDatabaseSessionAddRowAllowsSupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		return true, nil
	}
	addCalled := false
	mock.AddRowFunc = func(*engine.PluginConfig, string, string, []engine.Record) (bool, error) {
		addCalled = true
		return true, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Postgres",
		[]source.Surface{source.SurfaceBrowser, source.SurfaceQuery, source.SurfaceChat, source.SurfaceGraph},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindSchema: {source.ActionBrowse, source.ActionCreateChild},
			source.ObjectKindTable: {
				source.ActionInspect,
				source.ActionViewRows,
				source.ActionInsertData,
				source.ActionUpdateData,
				source.ActionImportData,
				source.ActionGenerateMockData,
			},
		},
	), mock)

	status, err := session.AddRow(context.Background(), testTableRef(), []source.Record{{Key: "name", Value: "alice"}})
	if err != nil {
		t.Fatalf("expected insert to succeed, got %v", err)
	}
	if !status || !addCalled {
		t.Fatalf("expected plugin insert to run, got status=%t addCalled=%t", status, addCalled)
	}
}

func newTestDatabaseSession(spec source.TypeSpec, mock *testutil.PluginMock) *DatabaseSession {
	return &DatabaseSession{
		spec:   spec,
		plugin: mock.AsPlugin(),
		credentials: &source.Credentials{
			SourceType: spec.ID,
			Values: map[string]string{
				"Database": "app",
			},
		},
	}
}

func testTypeSpec(label string, surfaces []source.Surface) source.TypeSpec {
	return testTypeWithObjectActions(label, surfaces, []source.Action{source.ActionBrowse}, map[source.ObjectKind][]source.Action{})
}

func testTypeWithObjectActions(label string, surfaces []source.Surface, rootActions []source.Action, objectActions map[source.ObjectKind][]source.Action) source.TypeSpec {
	objectTypes := make([]source.ObjectType, 0, len(objectActions))
	for kind, actions := range objectActions {
		objectTypes = append(objectTypes, source.ObjectType{
			Kind:      kind,
			DataShape: source.DataShapeTabular,
			Actions:   actions,
			Views:     []source.View{source.ViewGrid, source.ViewMetadata},
		})
	}

	return source.TypeSpec{
		ID:        label,
		Label:     label,
		Connector: label,
		Contract: source.Contract{
			Surfaces:          surfaces,
			RootActions:       rootActions,
			BrowsePath:        []source.ObjectKind{source.ObjectKindDatabase, source.ObjectKindSchema, source.ObjectKindTable},
			DefaultObjectKind: source.ObjectKindTable,
			ObjectTypes:       objectTypes,
		},
	}
}

func testTableRef() source.ObjectRef {
	return source.NewObjectRef(source.ObjectKindTable, []string{"app", "public", "users"})
}

func testSchemaRef() *source.ObjectRef {
	ref := source.NewObjectRef(source.ObjectKindSchema, []string{"app", "public"})
	return &ref
}
