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

package cloud

import (
	"context"
	"os"
	"strings"
	"testing"

	commonconfig "github.com/clidey/whodb/core/src/common/config"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/providers"
	"github.com/clidey/whodb/core/src/settings"
)

func setupCloudTestEnv(t *testing.T) func() {
	t.Helper()

	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	originalXDG := os.Getenv("XDG_DATA_HOME")
	originalAppData := os.Getenv("APPDATA")
	originalAWSProvider := os.Getenv("WHODB_AWS_PROVIDER")
	originalAzureProvider := os.Getenv("WHODB_AZURE_PROVIDER")
	originalGCPProvider := os.Getenv("WHODB_GCP_PROVIDER")
	originalEnterprise := env.IsEnterpriseEdition
	originalAWS := env.IsAWSProviderEnabled
	originalAzure := env.IsAzureProviderEnabled
	originalGCP := env.IsGCPProviderEnabled

	testHome := t.TempDir()
	for _, envVar := range []struct {
		key   string
		value string
	}{
		{key: "HOME", value: testHome},
		{key: "USERPROFILE", value: testHome},
		{key: "XDG_DATA_HOME", value: testHome},
		{key: "APPDATA", value: testHome},
	} {
		if err := os.Setenv(envVar.key, envVar.value); err != nil {
			t.Fatalf("Failed to set %s: %v", envVar.key, err)
		}
	}

	for _, key := range []string{"WHODB_AWS_PROVIDER", "WHODB_AZURE_PROVIDER", "WHODB_GCP_PROVIDER"} {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Failed to unset %s: %v", key, err)
		}
	}

	commonconfig.ResetConfigPath()
	resetCloudProviders(t)
	env.IsEnterpriseEdition = false
	env.IsAWSProviderEnabled = true
	env.IsAzureProviderEnabled = true
	env.IsGCPProviderEnabled = true

	return func() {
		resetCloudProviders(t)
		commonconfig.ResetConfigPath()
		restoreEnv(t, "HOME", originalHome)
		restoreEnv(t, "USERPROFILE", originalUserProfile)
		restoreEnv(t, "XDG_DATA_HOME", originalXDG)
		restoreEnv(t, "APPDATA", originalAppData)
		restoreEnv(t, "WHODB_AWS_PROVIDER", originalAWSProvider)
		restoreEnv(t, "WHODB_AZURE_PROVIDER", originalAzureProvider)
		restoreEnv(t, "WHODB_GCP_PROVIDER", originalGCPProvider)
		env.IsEnterpriseEdition = originalEnterprise
		env.IsAWSProviderEnabled = originalAWS
		env.IsAzureProviderEnabled = originalAzure
		env.IsGCPProviderEnabled = originalGCP
	}
}

func restoreEnv(t *testing.T, key, value string) {
	t.Helper()
	if value == "" {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Failed to unset %s: %v", key, err)
		}
		return
	}
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to restore %s: %v", key, err)
	}
}

func resetCloudProviders(t *testing.T) {
	t.Helper()

	for _, state := range settings.GetAWSProviders() {
		if err := settings.RemoveAWSProvider(state.Config.ID); err != nil {
			t.Fatalf("RemoveAWSProvider(%q) failed: %v", state.Config.ID, err)
		}
	}
	for _, state := range settings.GetAzureProviders() {
		if err := settings.RemoveAzureProvider(state.Config.ID); err != nil {
			t.Fatalf("RemoveAzureProvider(%q) failed: %v", state.Config.ID, err)
		}
	}
	for _, state := range settings.GetGCPProviders() {
		if err := settings.RemoveGCPProvider(state.Config.ID); err != nil {
			t.Fatalf("RemoveGCPProvider(%q) failed: %v", state.Config.ID, err)
		}
	}

	_ = providers.GetDefaultRegistry().Close(context.Background())
}

func TestListProviders_IncludesConfiguredProviders(t *testing.T) {
	cleanup := setupCloudTestEnv(t)
	defer cleanup()

	if _, err := settings.AddAWSProvider(&settings.AWSProviderConfig{
		ID:                  "aws-prod-us-west-2",
		Name:                "AWS Prod",
		Region:              "us-west-2",
		AuthMethod:          "profile",
		ProfileName:         "prod",
		DiscoverRDS:         true,
		DiscoverElastiCache: true,
	}); err != nil {
		t.Fatalf("AddAWSProvider failed: %v", err)
	}
	if _, err := settings.AddAzureProvider(&settings.AzureProviderConfig{
		ID:                 "azure-prod-sub-123",
		Name:               "Azure Prod",
		SubscriptionID:     "sub-123",
		AuthMethod:         "default",
		DiscoverPostgreSQL: true,
		DiscoverMySQL:      true,
	}, ""); err != nil {
		t.Fatalf("AddAzureProvider failed: %v", err)
	}
	if _, err := settings.AddGCPProvider(&settings.GCPProviderConfig{
		ID:               "gcp-prod-us-central1",
		Name:             "GCP Prod",
		ProjectID:        "project-123",
		Region:           "us-central1",
		AuthMethod:       "default",
		DiscoverCloudSQL: true,
	}); err != nil {
		t.Fatalf("AddGCPProvider failed: %v", err)
	}

	providers, err := ListProviders()
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers, got %d (%#v)", len(providers), providers)
	}

	if providers[0].ProviderType != "aws" || providers[0].Scope != "prod" {
		t.Fatalf("unexpected AWS provider summary: %#v", providers[0])
	}
	if providers[1].ProviderType != "azure" || providers[1].Scope != "sub-123" {
		t.Fatalf("unexpected Azure provider summary: %#v", providers[1])
	}
	if providers[2].ProviderType != "gcp" || providers[2].Scope != "project-123" {
		t.Fatalf("unexpected GCP provider summary: %#v", providers[2])
	}
}

func TestListConnections_NoProvidersReturnsEmpty(t *testing.T) {
	cleanup := setupCloudTestEnv(t)
	defer cleanup()

	connections, err := ListConnections(context.Background(), "")
	if err != nil {
		t.Fatalf("ListConnections returned unexpected error: %v", err)
	}
	if len(connections) != 0 {
		t.Fatalf("expected no discovered connections, got %#v", connections)
	}
}

func TestListProviders_DisabledProviderSupport(t *testing.T) {
	cleanup := setupCloudTestEnv(t)
	defer cleanup()

	env.IsAWSProviderEnabled = false
	env.IsAzureProviderEnabled = false
	env.IsGCPProviderEnabled = false

	_, err := ListProviders()
	if err == nil || !strings.Contains(err.Error(), "cloud provider support is disabled") {
		t.Fatalf("expected disabled cloud provider error, got %v", err)
	}
}

func TestTestProvider_MissingProvider(t *testing.T) {
	cleanup := setupCloudTestEnv(t)
	defer cleanup()

	_, err := TestProvider("missing-provider")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing provider error, got %v", err)
	}
}

func TestBuildPrefillConnection_UsesMetadataAndPrefillRules(t *testing.T) {
	conn, err := BuildPrefillConnection(ConnectionSummary{
		ID:         "aws-prod/prod-db",
		Name:       "prod-db",
		SourceType: "Postgres",
		Metadata: map[string]string{
			"endpoint": "prod-db.example.com",
			"port":     "5432",
		},
	})
	if err != nil {
		t.Fatalf("BuildPrefillConnection failed: %v", err)
	}

	if conn.Host != "prod-db.example.com" {
		t.Fatalf("expected endpoint host, got %#v", conn)
	}
	if conn.Port != 5432 {
		t.Fatalf("expected port 5432, got %#v", conn)
	}
	if conn.Advanced["SSL Mode"] != "require" {
		t.Fatalf("expected SSL prefill, got %#v", conn.Advanced)
	}
}

func TestBuildPrefillConnection_RejectsUnsupportedType(t *testing.T) {
	_, err := BuildPrefillConnection(ConnectionSummary{
		ID:         "aws-prod/unknown",
		Name:       "unknown",
		SourceType: "Neptune",
		Metadata:   map[string]string{},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported database type") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

func TestResolveConnection_FindsRegisteredProviderDiscovery(t *testing.T) {
	cleanup := setupCloudTestEnv(t)
	defer cleanup()

	registry := providers.GetDefaultRegistry()
	if err := registry.Register(&testProvider{
		id:   "aws-test",
		name: "AWS Test",
		connections: []providers.DiscoveredConnection{{
			ID:           "aws-test/prod-db",
			ProviderType: providers.ProviderTypeAWS,
			ProviderID:   "aws-test",
			Name:         "prod-db",
			DatabaseType: engine.DatabaseType_Postgres,
			Status:       providers.ConnectionStatusAvailable,
			Metadata: map[string]string{
				"endpoint": "prod-db.example.com",
				"port":     "5432",
			},
		}},
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	summary, err := ResolveConnection(context.Background(), "aws-test/prod-db")
	if err != nil {
		t.Fatalf("ResolveConnection failed: %v", err)
	}

	if summary.ProviderID != "aws-test" || summary.Name != "prod-db" {
		t.Fatalf("unexpected resolved summary: %#v", summary)
	}
}

type testProvider struct {
	id          string
	name        string
	connections []providers.DiscoveredConnection
}

func (p *testProvider) Type() providers.ProviderType { return providers.ProviderTypeAWS }

func (p *testProvider) ID() string { return p.id }

func (p *testProvider) Name() string { return p.name }

func (p *testProvider) DiscoverConnections(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	return append([]providers.DiscoveredConnection(nil), p.connections...), nil
}

func (p *testProvider) TestConnection(ctx context.Context) error { return nil }

func (p *testProvider) RefreshConnection(ctx context.Context, connectionID string) (bool, error) {
	return false, nil
}

func (p *testProvider) Close(ctx context.Context) error { return nil }
