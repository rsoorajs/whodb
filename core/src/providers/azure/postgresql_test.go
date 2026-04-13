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

package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func TestMapPostgresStatus(t *testing.T) {
	testCases := []struct {
		state    armpostgresqlflexibleservers.ServerState
		expected providers.ConnectionStatus
	}{
		{armpostgresqlflexibleservers.ServerStateReady, providers.ConnectionStatusAvailable},
		{armpostgresqlflexibleservers.ServerStateStarting, providers.ConnectionStatusStarting},
		{armpostgresqlflexibleservers.ServerStateUpdating, providers.ConnectionStatusStarting},
		{armpostgresqlflexibleservers.ServerStateStopped, providers.ConnectionStatusStopped},
		{armpostgresqlflexibleservers.ServerStateStopping, providers.ConnectionStatusStopped},
		{armpostgresqlflexibleservers.ServerStateDisabled, providers.ConnectionStatusStopped},
		{armpostgresqlflexibleservers.ServerStateDropping, providers.ConnectionStatusDeleting},
	}

	for _, tc := range testCases {
		state := tc.state
		result := mapPostgresStatus(&state)
		if result != tc.expected {
			t.Errorf("mapPostgresStatus(%s): expected %s, got %s", tc.state, tc.expected, result)
		}
	}
}

func TestMapPostgresStatus_Nil(t *testing.T) {
	result := mapPostgresStatus(nil)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapPostgresStatus(nil): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func TestMapPostgresStatus_UnknownValue(t *testing.T) {
	unknown := armpostgresqlflexibleservers.ServerState("SomeUnknownState")
	result := mapPostgresStatus(&unknown)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapPostgresStatus(SomeUnknownState): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func newTestPostgresProvider() *Provider {
	p, _ := New(&Config{
		ID:                 "test-pg",
		Name:               "Test PostgreSQL",
		SubscriptionID:     "00000000-0000-0000-0000-000000000000",
		DiscoverPostgreSQL: true,
	})
	return p
}

func TestPostgresServerToConnection_HappyPath(t *testing.T) {
	p := newTestPostgresProvider()
	name := "my-postgres"
	location := "eastus"
	fqdn := "my-postgres.postgres.database.azure.com"
	version := armpostgresqlflexibleservers.ServerVersion("14")
	state := armpostgresqlflexibleservers.ServerStateReady
	sku := "Standard_D2s_v3"
	resourceID := "/subscriptions/00000000/resourceGroups/my-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/my-postgres"

	server := &armpostgresqlflexibleservers.Server{
		Name:     &name,
		Location: &location,
		ID:       &resourceID,
		Properties: &armpostgresqlflexibleservers.ServerProperties{
			FullyQualifiedDomainName: &fqdn,
			Version:                  &version,
			State:                    &state,
		},
		SKU: &armpostgresqlflexibleservers.SKU{
			Name: &sku,
		},
	}

	conn := p.postgresServerToConnection(server)

	if conn.DatabaseType != engine.DatabaseType_Postgres {
		t.Errorf("expected Postgres, got %s", conn.DatabaseType)
	}
	if conn.Name != "my-postgres" {
		t.Errorf("expected name my-postgres, got %s", conn.Name)
	}
	if conn.Region != "eastus" {
		t.Errorf("expected region eastus, got %s", conn.Region)
	}
	if conn.Status != providers.ConnectionStatusAvailable {
		t.Errorf("expected Available status, got %s", conn.Status)
	}
	if conn.Metadata["endpoint"] != "my-postgres.postgres.database.azure.com" {
		t.Errorf("unexpected endpoint: %s", conn.Metadata["endpoint"])
	}
	if conn.Metadata["port"] != "5432" {
		t.Errorf("unexpected port: %s", conn.Metadata["port"])
	}
	if conn.Metadata["version"] != "14" {
		t.Errorf("unexpected version: %s", conn.Metadata["version"])
	}
	if conn.Metadata["resourceGroup"] != "my-rg" {
		t.Errorf("unexpected resourceGroup: %s", conn.Metadata["resourceGroup"])
	}
	if conn.Metadata["sku"] != "Standard_D2s_v3" {
		t.Errorf("unexpected sku: %s", conn.Metadata["sku"])
	}
	if conn.ProviderType != providers.ProviderTypeAzure {
		t.Errorf("expected ProviderType Azure, got %s", conn.ProviderType)
	}
	if conn.ProviderID != "test-pg" {
		t.Errorf("expected ProviderID test-pg, got %s", conn.ProviderID)
	}
}

func TestPostgresServerToConnection_NilProperties(t *testing.T) {
	p := newTestPostgresProvider()
	name := "my-postgres"
	location := "eastus"

	server := &armpostgresqlflexibleservers.Server{
		Name:       &name,
		Location:   &location,
		Properties: nil,
	}

	conn := p.postgresServerToConnection(server)

	if conn.Status != providers.ConnectionStatusUnknown {
		t.Errorf("expected Unknown status for nil properties, got %s", conn.Status)
	}
	if conn.Metadata["port"] != "5432" {
		t.Errorf("expected port 5432, got %s", conn.Metadata["port"])
	}
}

func TestPostgresServerToConnection_NilSKU(t *testing.T) {
	p := newTestPostgresProvider()
	name := "my-postgres"
	state := armpostgresqlflexibleservers.ServerStateReady

	server := &armpostgresqlflexibleservers.Server{
		Name: &name,
		Properties: &armpostgresqlflexibleservers.ServerProperties{
			State: &state,
		},
		SKU: nil,
	}

	conn := p.postgresServerToConnection(server)

	if _, ok := conn.Metadata["sku"]; ok {
		t.Error("expected no sku metadata when SKU is nil")
	}
}
