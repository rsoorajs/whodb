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

package graph

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/source"
)

const (
	testContentDriverID = "test-content-driver"
	testContentSourceID = "TestContentSource"
)

var registerTestContentSourceOnce sync.Once

func TestQuerySourceContentReadsRegisteredContentSource(t *testing.T) {
	registerTestContentSource()

	ctx := context.WithValue(context.Background(), auth.AuthKey_Source, &source.Credentials{
		SourceType: testContentSourceID,
		Values: map[string]string{
			"Content": "hello from resolver",
		},
	})

	content, err := (&Resolver{}).Query().SourceContent(ctx, model.SourceObjectRefInput{
		Kind: model.SourceObjectKindItem,
		Path: []string{"notes.txt"},
	})
	if err != nil {
		t.Fatalf("expected SourceContent query to succeed, got %v", err)
	}
	if content == nil || content.Text == nil || *content.Text != "hello from resolver" {
		t.Fatalf("expected SourceContent text payload, got %#v", content)
	}
	if content.IsBinary {
		t.Fatalf("expected text file to be treated as text, got %#v", content)
	}
}

func registerTestContentSource() {
	registerTestContentSourceOnce.Do(func() {
		source.RegisterDriver(testContentDriverID, testContentConnector{})
		source.RegisterType(source.TypeSpec{
			ID:        testContentSourceID,
			Label:     "Test Content Source",
			DriverID:  testContentDriverID,
			Connector: testContentSourceID,
			Category:  source.CategoryObjectStore,
			Contract: source.Contract{
				Model:             source.ModelObject,
				Surfaces:          []source.Surface{source.SurfaceBrowser},
				RootActions:       []source.Action{source.ActionBrowse},
				BrowsePath:        []source.ObjectKind{source.ObjectKindItem},
				DefaultObjectKind: source.ObjectKindItem,
				ObjectTypes: []source.ObjectType{
					{
						Kind:          source.ObjectKindItem,
						DataShape:     source.DataShapeContent,
						Actions:       []source.Action{source.ActionViewContent},
						Views:         []source.View{source.ViewText},
						SingularLabel: "Item",
						PluralLabel:   "Items",
					},
				},
			},
		})
	})
}

type testContentConnector struct{}

func (testContentConnector) Open(_ context.Context, spec source.TypeSpec, credentials *source.Credentials) (source.SourceSession, error) {
	return &testContentSession{
		spec:    spec,
		content: credentials.CloneValues()["Content"],
	}, nil
}

type testContentSession struct {
	spec    source.TypeSpec
	content string
}

func (s *testContentSession) Metadata(_ context.Context) (*source.SessionMetadata, error) {
	return &source.SessionMetadata{
		SourceType:     s.spec.ID,
		QueryLanguages: []string{},
		AliasMap:       map[string]string{},
	}, nil
}

func (s *testContentSession) ReadContent(_ context.Context, ref source.ObjectRef) (*source.ContentResult, error) {
	fileName := "content.txt"
	if len(ref.Path) > 0 {
		fileName = ref.Path[len(ref.Path)-1]
	}

	text := s.content
	return &source.ContentResult{
		Text:      &text,
		MIMEType:  "text/plain; charset=utf-8",
		IsBinary:  false,
		SizeBytes: int64(len(text)),
		FileName:  fileName,
	}, nil
}

func (s *testContentSession) IsAvailable(_ context.Context) bool {
	return strings.TrimSpace(s.content) != ""
}
