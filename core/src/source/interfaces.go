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

package source

import (
	"context"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
)

// SourceSession is the common session interface exposed by a source connector.
type SourceSession interface {
	// Metadata returns session-scoped metadata for editors and query builders.
	Metadata(ctx context.Context) (*SessionMetadata, error)
}

// SourceConnector opens sessions for one source type.
type SourceConnector interface {
	// Open creates a new session for the provided source type and credentials.
	Open(ctx context.Context, spec TypeSpec, credentials *Credentials) (SourceSession, error)
}

// SourceBrowser lists and resolves browseable objects.
type SourceBrowser interface {
	// ListObjects lists objects beneath the provided parent.
	ListObjects(ctx context.Context, parent *ObjectRef, kinds []ObjectKind) ([]Object, error)
	// GetObject loads one object by reference.
	GetObject(ctx context.Context, ref ObjectRef) (*Object, error)
}

// TabularReader reads row/column data from a source object.
type TabularReader interface {
	// ReadRows returns tabular rows for the provided object reference.
	ReadRows(ctx context.Context, ref ObjectRef, where *model.WhereCondition, sort []*model.SortCondition, pageSize int, pageOffset int) (*engine.GetRowsResult, error)
	// Columns returns columns for one object.
	Columns(ctx context.Context, ref ObjectRef) ([]engine.Column, error)
	// ColumnsBatch returns columns for multiple objects.
	ColumnsBatch(ctx context.Context, refs []ObjectRef) ([]ObjectColumns, error)
}

// ContentReader reads blob/text content from a source object.
type ContentReader interface {
	// ReadContent returns a content payload for the provided object reference.
	ReadContent(ctx context.Context, ref ObjectRef) (string, error)
}

// QueryRunner executes source-native queries.
type QueryRunner interface {
	// RunQuery executes a query against the active source session.
	RunQuery(ctx context.Context, query string, params ...any) (*engine.GetRowsResult, error)
}

// GraphReader reads graph data for a source scope.
type GraphReader interface {
	// ReadGraph returns graph units for the provided scope reference, or the
	// default source graph when ref is nil.
	ReadGraph(ctx context.Context, ref *ObjectRef) ([]engine.GraphUnit, error)
}

// SourceAssistant runs AI chat against a source scope.
type SourceAssistant interface {
	// Reply runs the source assistant against the provided scope.
	Reply(ctx context.Context, ref *ObjectRef, previousConversation string, query string) ([]*engine.ChatMessage, error)
}

// ObjectManager mutates source objects and row data.
type ObjectManager interface {
	// CreateObject creates a new object beneath the provided parent.
	CreateObject(ctx context.Context, parent *ObjectRef, name string, fields []engine.Record) (bool, error)
	// UpdateObject updates an existing object.
	UpdateObject(ctx context.Context, ref ObjectRef, values map[string]string, updatedColumns []string) (bool, error)
	// AddRow inserts a row/document into an object.
	AddRow(ctx context.Context, ref ObjectRef, values []engine.Record) (bool, error)
	// DeleteRow deletes a row/document from an object.
	DeleteRow(ctx context.Context, ref ObjectRef, values map[string]string) (bool, error)
}

// ConnectionFieldOptionsReader loads dynamic options for a connection field.
type ConnectionFieldOptionsReader interface {
	// ConnectionFieldOptions returns selectable values for a connection field.
	ConnectionFieldOptions(ctx context.Context, fieldKey string, values map[string]string) ([]string, error)
}
