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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	graphapi "github.com/clidey/whodb/core/graph"
	"github.com/clidey/whodb/core/src/env"
	"github.com/go-chi/chi/v5"
)

func TestHealthCheckMiddlewareShortCircuitsHandler(t *testing.T) {
	called := false
	handler := healthCheckMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected health middleware to return 200, got %d", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Fatalf("expected health body 'ok', got %q", rr.Body.String())
	}
	if called {
		t.Fatal("expected health middleware to bypass the wrapped handler")
	}
}

func TestSetupMiddlewaresPublicPathsBypassAuth(t *testing.T) {
	router := chi.NewRouter()
	setupMiddlewares(router, nil, []string{"/api/public"})

	router.Get("/api/public", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	router.Get("/api/private", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	publicReq := httptest.NewRequest(http.MethodGet, "/api/public", nil)
	publicRes := httptest.NewRecorder()
	router.ServeHTTP(publicRes, publicReq)
	if publicRes.Code != http.StatusNoContent {
		t.Fatalf("expected public path to bypass auth, got %d", publicRes.Code)
	}

	privateReq := httptest.NewRequest(http.MethodGet, "/api/private", nil)
	privateRes := httptest.NewRecorder()
	router.ServeHTTP(privateRes, privateReq)
	if privateRes.Code != http.StatusUnauthorized {
		t.Fatalf("expected private path to require auth, got %d", privateRes.Code)
	}
}

func TestNewGraphQLServerTogglesIntrospectionByEnvironment(t *testing.T) {
	queryBody, err := json.Marshal(map[string]any{
		"query": `query IntrospectionQuery { __schema { queryType { name } } }`,
	})
	if err != nil {
		t.Fatalf("failed to build request body: %v", err)
	}

	runQuery := func(isDevelopment bool) string {
		originalDev := env.IsDevelopment
		env.IsDevelopment = isDevelopment
		defer func() { env.IsDevelopment = originalDev }()

		server := NewGraphQLServer(graphapi.NewExecutableSchema(graphapi.Config{Resolvers: &graphapi.Resolver{}}))
		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(queryBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected GraphQL HTTP 200, got %d with body %s", rr.Code, rr.Body.String())
		}
		return rr.Body.String()
	}

	devBody := runQuery(true)
	if !strings.Contains(devBody, "__schema") {
		t.Fatalf("expected introspection data in development mode, got %s", devBody)
	}

	prodBody := runQuery(false)
	if !strings.Contains(prodBody, "errors") {
		t.Fatalf("expected introspection to be rejected outside development mode, got %s", prodBody)
	}
}
