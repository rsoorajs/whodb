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

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func TestMapMySQLStatus(t *testing.T) {
	testCases := []struct {
		state    armmysqlflexibleservers.ServerState
		expected providers.ConnectionStatus
	}{
		{armmysqlflexibleservers.ServerStateReady, providers.ConnectionStatusAvailable},
		{armmysqlflexibleservers.ServerStateStarting, providers.ConnectionStatusStarting},
		{armmysqlflexibleservers.ServerStateUpdating, providers.ConnectionStatusStarting},
		{armmysqlflexibleservers.ServerStateStopped, providers.ConnectionStatusStopped},
		{armmysqlflexibleservers.ServerStateStopping, providers.ConnectionStatusStopped},
		{armmysqlflexibleservers.ServerStateDisabled, providers.ConnectionStatusStopped},
		{armmysqlflexibleservers.ServerStateDropping, providers.ConnectionStatusDeleting},
	}

	for _, tc := range testCases {
		state := tc.state
		result := mapMySQLStatus(&state)
		if result != tc.expected {
			t.Errorf("mapMySQLStatus(%s): expected %s, got %s", tc.state, tc.expected, result)
		}
	}
}

func TestMapMySQLStatus_Nil(t *testing.T) {
	result := mapMySQLStatus(nil)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapMySQLStatus(nil): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func TestMapMySQLStatus_UnknownValue(t *testing.T) {
	unknown := armmysqlflexibleservers.ServerState("SomeUnknownState")
	result := mapMySQLStatus(&unknown)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapMySQLStatus(SomeUnknownState): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func newTestMySQLProvider() *Provider {
	p, _ := New(&Config{
		ID:             "test-mysql",
		Name:           "Test MySQL",
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
		DiscoverMySQL:  true,
	})
	return p
}

func TestMySQLServerToConnection_HappyPath(t *testing.T) {
	p := newTestMySQLProvider()
	name := "my-mysql"
	location := "westus2"
	fqdn := "my-mysql.mysql.database.azure.com"
	version := armmysqlflexibleservers.ServerVersion("8.0.21")
	state := armmysqlflexibleservers.ServerStateReady
	sku := "Standard_B1ms"
	resourceID := "/subscriptions/00000000/resourceGroups/my-rg/providers/Microsoft.DBforMySQL/flexibleServers/my-mysql"

	server := &armmysqlflexibleservers.Server{
		Name:     &name,
		Location: &location,
		ID:       &resourceID,
		Properties: &armmysqlflexibleservers.ServerProperties{
			FullyQualifiedDomainName: &fqdn,
			Version:                  &version,
			State:                    &state,
		},
		SKU: &armmysqlflexibleservers.SKU{
			Name: &sku,
		},
	}

	conn := p.mysqlServerToConnection(server)

	if conn.DatabaseType != engine.DatabaseType_MySQL {
		t.Errorf("expected MySQL, got %s", conn.DatabaseType)
	}
	if conn.Name != "my-mysql" {
		t.Errorf("expected name my-mysql, got %s", conn.Name)
	}
	if conn.Region != "westus2" {
		t.Errorf("expected region westus2, got %s", conn.Region)
	}
	if conn.Status != providers.ConnectionStatusAvailable {
		t.Errorf("expected Available status, got %s", conn.Status)
	}
	if conn.Metadata["endpoint"] != "my-mysql.mysql.database.azure.com" {
		t.Errorf("unexpected endpoint: %s", conn.Metadata["endpoint"])
	}
	if conn.Metadata["port"] != "3306" {
		t.Errorf("unexpected port: %s", conn.Metadata["port"])
	}
	if conn.Metadata["version"] != "8.0.21" {
		t.Errorf("unexpected version: %s", conn.Metadata["version"])
	}
	if conn.Metadata["resourceGroup"] != "my-rg" {
		t.Errorf("unexpected resourceGroup: %s", conn.Metadata["resourceGroup"])
	}
	if conn.Metadata["sku"] != "Standard_B1ms" {
		t.Errorf("unexpected sku: %s", conn.Metadata["sku"])
	}
	if conn.ProviderType != providers.ProviderTypeAzure {
		t.Errorf("expected ProviderType Azure, got %s", conn.ProviderType)
	}
	if conn.ProviderID != "test-mysql" {
		t.Errorf("expected ProviderID test-mysql, got %s", conn.ProviderID)
	}
}

func TestMySQLServerToConnection_NilProperties(t *testing.T) {
	p := newTestMySQLProvider()
	name := "my-mysql"
	location := "westus2"

	server := &armmysqlflexibleservers.Server{
		Name:       &name,
		Location:   &location,
		Properties: nil,
	}

	conn := p.mysqlServerToConnection(server)

	if conn.Status != providers.ConnectionStatusUnknown {
		t.Errorf("expected Unknown status for nil properties, got %s", conn.Status)
	}
	if conn.Metadata["port"] != "3306" {
		t.Errorf("expected port 3306, got %s", conn.Metadata["port"])
	}
}

func TestMySQLServerToConnection_NilSKU(t *testing.T) {
	p := newTestMySQLProvider()
	name := "my-mysql"
	state := armmysqlflexibleservers.ServerStateReady

	server := &armmysqlflexibleservers.Server{
		Name: &name,
		Properties: &armmysqlflexibleservers.ServerProperties{
			State: &state,
		},
		SKU: nil,
	}

	conn := p.mysqlServerToConnection(server)

	if _, ok := conn.Metadata["sku"]; ok {
		t.Error("expected no sku metadata when SKU is nil")
	}
}
