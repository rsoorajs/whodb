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

package gcp

import (
	"strings"
	"testing"

	gcpinfra "github.com/clidey/whodb/core/src/gcp"
	"github.com/clidey/whodb/core/src/providers"
)

func TestNew_ValidConfig(t *testing.T) {
	config := &Config{
		ID:        "gcp-proj-1",
		Name:      "Test GCP",
		ProjectID: "my-project-123",
		Region:    "us-central1",
	}

	p, err := New(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Type() != providers.ProviderTypeGCP {
		t.Errorf("expected type %s, got %s", providers.ProviderTypeGCP, p.Type())
	}
	if p.ID() != "gcp-proj-1" {
		t.Errorf("expected ID gcp-proj-1, got %s", p.ID())
	}
	if p.Name() != "Test GCP" {
		t.Errorf("expected Name Test GCP, got %s", p.Name())
	}
}

func TestNew_NilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestNew_MissingID(t *testing.T) {
	config := &Config{
		Name:      "Test GCP",
		ProjectID: "my-project-123",
		Region:    "us-central1",
	}

	_, err := New(config)
	if err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestNew_MissingProjectID(t *testing.T) {
	config := &Config{
		ID:     "gcp-proj-1",
		Name:   "Test GCP",
		Region: "us-central1",
	}

	_, err := New(config)
	if err == nil {
		t.Error("expected error for missing project ID")
	}
}

func TestNew_MissingRegion(t *testing.T) {
	config := &Config{
		ID:        "gcp-proj-1",
		Name:      "Test GCP",
		ProjectID: "my-project-123",
	}

	_, err := New(config)
	if err == nil {
		t.Error("expected error for missing region")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("gcp-proj-1", "Test GCP", "my-project-123", "us-central1")

	if config.ID != "gcp-proj-1" {
		t.Errorf("expected ID gcp-proj-1, got %s", config.ID)
	}
	if config.Name != "Test GCP" {
		t.Errorf("expected Name Test GCP, got %s", config.Name)
	}
	if config.ProjectID != "my-project-123" {
		t.Errorf("expected ProjectID my-project-123, got %s", config.ProjectID)
	}
	if config.Region != "us-central1" {
		t.Errorf("expected Region us-central1, got %s", config.Region)
	}
	if config.AuthMethod != gcpinfra.AuthMethodDefault {
		t.Errorf("expected AuthMethod default, got %s", config.AuthMethod)
	}
	if !config.DiscoverCloudSQL {
		t.Error("expected DiscoverCloudSQL to be true")
	}
	if !config.DiscoverAlloyDB {
		t.Error("expected DiscoverAlloyDB to be true")
	}
	if !config.DiscoverMemorystore {
		t.Error("expected DiscoverMemorystore to be true")
	}
}

func TestProvider_ConnectionID(t *testing.T) {
	p, _ := New(&Config{
		ID:        "gcp-proj-1",
		Name:      "Test GCP",
		ProjectID: "my-project-123",
		Region:    "us-central1",
	})

	connID := p.connectionID("cloudsql-my-instance")
	expected := "gcp-proj-1/cloudsql-my-instance"
	if connID != expected {
		t.Errorf("expected %s, got %s", expected, connID)
	}
}

func TestConfig_DiscoveryFlags(t *testing.T) {
	config := &Config{
		ID:                  "test",
		Name:                "Test",
		ProjectID:           "proj-1",
		Region:              "us-central1",
		DiscoverCloudSQL:    true,
		DiscoverAlloyDB:     false,
		DiscoverMemorystore: true,
	}

	if !config.DiscoverCloudSQL {
		t.Error("expected DiscoverCloudSQL to be true")
	}
	if config.DiscoverAlloyDB {
		t.Error("expected DiscoverAlloyDB to be false")
	}
	if !config.DiscoverMemorystore {
		t.Error("expected DiscoverMemorystore to be true")
	}
}

func TestConfig_String_ExcludesSecrets(t *testing.T) {
	config := &Config{
		ID:                    "gcp-proj-1",
		Name:                  "Test GCP",
		ProjectID:             "proj-123",
		Region:                "us-central1",
		AuthMethod:            gcpinfra.AuthMethodServiceAccountKey,
		ServiceAccountKeyPath: "/secret/path/to/key.json",
	}

	s := config.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
	if strings.Contains(s, "/secret/path/to/key.json") {
		t.Error("String() should not include service account key path")
	}
}

func TestProvider_GetConfig(t *testing.T) {
	config := &Config{
		ID:        "gcp-proj-1",
		Name:      "Test GCP",
		ProjectID: "my-project-123",
		Region:    "us-central1",
	}

	p, _ := New(config)
	if p.GetConfig() != config {
		t.Error("expected GetConfig to return the same config")
	}
}
