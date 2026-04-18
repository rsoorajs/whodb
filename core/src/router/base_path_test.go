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

package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWrapWithBasePathRoutesRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.URL.Path))
	})

	router := wrapWithBasePath(handler, "/whodb")

	t.Run("redirects exact base path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/whodb", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusMovedPermanently {
			t.Fatalf("expected redirect status, got %d", rr.Code)
		}
		if location := rr.Header().Get("Location"); location != "/whodb/" {
			t.Fatalf("expected redirect location /whodb/, got %s", location)
		}
	})

	t.Run("strips prefix for nested routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/whodb/api/query", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected wrapped handler to succeed, got %d", rr.Code)
		}
		if body := rr.Body.String(); body != "/api/query" {
			t.Fatalf("expected stripped path /api/query, got %s", body)
		}
	})

	t.Run("preserves root health checks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected root health request to succeed, got %d", rr.Code)
		}
		if body := rr.Body.String(); body != "/health" {
			t.Fatalf("expected health request to reach inner handler unchanged, got %s", body)
		}
	})
}

func TestRenderIndexHTMLInjectsBaseHref(t *testing.T) {
	staticFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><head><base href="__WHODB_BASE_HREF__"></head><body>ok</body></html>`),
		},
	}

	rendered, err := renderIndexHTML(staticFS, "/whodb/")
	if err != nil {
		t.Fatalf("expected index.html to render, got error: %v", err)
	}

	if got := string(rendered); !strings.Contains(got, `<base href="/whodb/">`) {
		t.Fatalf("expected base href to be injected, got %s", got)
	}
}

func TestResolveStaticFSFindsEmbeddedBuildDirectory(t *testing.T) {
	staticFS := fstest.MapFS{
		"build/index.html":    &fstest.MapFile{Data: []byte("ok")},
		"build/assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	}

	resolved, found := resolveStaticFS(staticFS)
	if !found {
		t.Fatal("expected embedded frontend assets to be discovered")
	}

	file, err := resolved.Open("index.html")
	if err != nil {
		t.Fatalf("expected resolved fs to expose index.html, got error: %v", err)
	}
	_ = file.Close()
}
