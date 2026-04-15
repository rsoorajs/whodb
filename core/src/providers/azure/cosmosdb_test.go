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

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func TestMapCosmosStatus(t *testing.T) {
	testCases := []struct {
		state    string
		expected providers.ConnectionStatus
	}{
		{"Succeeded", providers.ConnectionStatusAvailable},
		{"succeeded", providers.ConnectionStatusAvailable},
		{"SUCCEEDED", providers.ConnectionStatusAvailable},
		{"Creating", providers.ConnectionStatusStarting},
		{"creating", providers.ConnectionStatusStarting},
		{"Updating", providers.ConnectionStatusStarting},
		{"updating", providers.ConnectionStatusStarting},
		{"Deleting", providers.ConnectionStatusDeleting},
		{"deleting", providers.ConnectionStatusDeleting},
		{"Failed", providers.ConnectionStatusFailed},
		{"failed", providers.ConnectionStatusFailed},
		{"unknown-state", providers.ConnectionStatusUnknown},
		{"", providers.ConnectionStatusUnknown},
	}

	for _, tc := range testCases {
		state := tc.state
		result := mapCosmosStatus(&state)
		if result != tc.expected {
			t.Errorf("mapCosmosStatus(%s): expected %s, got %s", tc.state, tc.expected, result)
		}
	}
}

func TestMapCosmosStatus_Nil(t *testing.T) {
	result := mapCosmosStatus(nil)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapCosmosStatus(nil): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func TestIsMongoDBAccount_KindMongoDB(t *testing.T) {
	kind := armcosmos.DatabaseAccountKindMongoDB
	account := &armcosmos.DatabaseAccountGetResults{
		Kind: &kind,
	}

	if !isMongoDBAccount(account) {
		t.Error("expected true for Kind=MongoDB")
	}
}

func TestIsMongoDBAccount_EnableMongoCapability(t *testing.T) {
	kind := armcosmos.DatabaseAccountKindGlobalDocumentDB
	capName := "EnableMongo"
	account := &armcosmos.DatabaseAccountGetResults{
		Kind: &kind,
		Properties: &armcosmos.DatabaseAccountGetProperties{
			Capabilities: []*armcosmos.Capability{
				{Name: &capName},
			},
		},
	}

	if !isMongoDBAccount(account) {
		t.Error("expected true for EnableMongo capability")
	}
}

func TestIsMongoDBAccount_EnableMongoCapability_CaseInsensitive(t *testing.T) {
	capName := "enablemongo"
	account := &armcosmos.DatabaseAccountGetResults{
		Properties: &armcosmos.DatabaseAccountGetProperties{
			Capabilities: []*armcosmos.Capability{
				{Name: &capName},
			},
		},
	}

	if !isMongoDBAccount(account) {
		t.Error("expected true for enablemongo capability (case-insensitive)")
	}
}

func TestIsMongoDBAccount_NotMongo(t *testing.T) {
	kind := armcosmos.DatabaseAccountKindGlobalDocumentDB
	account := &armcosmos.DatabaseAccountGetResults{
		Kind: &kind,
	}

	if isMongoDBAccount(account) {
		t.Error("expected false for GlobalDocumentDB without EnableMongo")
	}
}

func TestIsMongoDBAccount_NilKindNilProperties(t *testing.T) {
	account := &armcosmos.DatabaseAccountGetResults{}

	if isMongoDBAccount(account) {
		t.Error("expected false for nil Kind and nil Properties")
	}
}

func TestIsMongoDBAccount_OtherCapabilities(t *testing.T) {
	kind := armcosmos.DatabaseAccountKindGlobalDocumentDB
	cap1 := "EnableCassandra"
	cap2 := "EnableTable"
	account := &armcosmos.DatabaseAccountGetResults{
		Kind: &kind,
		Properties: &armcosmos.DatabaseAccountGetProperties{
			Capabilities: []*armcosmos.Capability{
				{Name: &cap1},
				{Name: &cap2},
			},
		},
	}

	if isMongoDBAccount(account) {
		t.Error("expected false for non-Mongo capabilities")
	}
}

func newTestCosmosProvider() *Provider {
	p, _ := New(&Config{
		ID:               "test-cosmos",
		Name:             "Test CosmosDB",
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		DiscoverCosmosDB: true,
	})
	return p
}

func TestCosmosAccountToConnection_HappyPath(t *testing.T) {
	p := newTestCosmosProvider()
	name := "my-cosmos"
	location := "westeurope"
	kind := armcosmos.DatabaseAccountKindMongoDB
	docEndpoint := "https://my-cosmos.mongo.cosmos.azure.com:443/"
	provState := "Succeeded"
	resourceID := "/subscriptions/00000000/resourceGroups/my-rg/providers/Microsoft.DocumentDB/databaseAccounts/my-cosmos"

	account := &armcosmos.DatabaseAccountGetResults{
		Name:     &name,
		Location: &location,
		ID:       &resourceID,
		Kind:     &kind,
		Properties: &armcosmos.DatabaseAccountGetProperties{
			DocumentEndpoint:  &docEndpoint,
			ProvisioningState: &provState,
		},
	}

	conn := p.cosmosAccountToConnection(account)

	if conn.DatabaseType != engine.DatabaseType_MongoDB {
		t.Errorf("expected MongoDB, got %s", conn.DatabaseType)
	}
	if conn.Name != "my-cosmos" {
		t.Errorf("expected name my-cosmos, got %s", conn.Name)
	}
	if conn.Region != "westeurope" {
		t.Errorf("expected region westeurope, got %s", conn.Region)
	}
	if conn.Status != providers.ConnectionStatusAvailable {
		t.Errorf("expected Available status, got %s", conn.Status)
	}
	if conn.Metadata["endpoint"] != "my-cosmos.mongo.cosmos.azure.com" {
		t.Errorf("unexpected endpoint: %s", conn.Metadata["endpoint"])
	}
	if conn.Metadata["port"] != "10255" {
		t.Errorf("unexpected port: %s", conn.Metadata["port"])
	}
	if conn.Metadata["kind"] != string(armcosmos.DatabaseAccountKindMongoDB) {
		t.Errorf("unexpected kind: %s", conn.Metadata["kind"])
	}
	if conn.Metadata["resourceGroup"] != "my-rg" {
		t.Errorf("unexpected resourceGroup: %s", conn.Metadata["resourceGroup"])
	}
	if conn.ProviderType != providers.ProviderTypeAzure {
		t.Errorf("expected ProviderType Azure, got %s", conn.ProviderType)
	}
	if conn.ProviderID != "test-cosmos" {
		t.Errorf("expected ProviderID test-cosmos, got %s", conn.ProviderID)
	}
}

func TestCosmosAccountToConnection_NilProperties(t *testing.T) {
	p := newTestCosmosProvider()
	name := "my-cosmos"
	location := "westeurope"

	account := &armcosmos.DatabaseAccountGetResults{
		Name:       &name,
		Location:   &location,
		Properties: nil,
	}

	conn := p.cosmosAccountToConnection(account)

	if conn.Status != providers.ConnectionStatusUnknown {
		t.Errorf("expected Unknown status for nil properties, got %s", conn.Status)
	}
	if conn.Metadata["port"] != "10255" {
		t.Errorf("expected port 10255, got %s", conn.Metadata["port"])
	}
}

func TestCosmosAccountToConnection_EndpointParsing(t *testing.T) {
	p := newTestCosmosProvider()
	name := "my-cosmos"
	testCases := []struct {
		docEndpoint string
		expected    string
	}{
		{"https://my-cosmos.mongo.cosmos.azure.com:443/", "my-cosmos.mongo.cosmos.azure.com"},
		{"https://my-cosmos.mongo.cosmos.azure.com/", "my-cosmos.mongo.cosmos.azure.com"},
		{"http://my-cosmos.mongo.cosmos.azure.com/", "my-cosmos.mongo.cosmos.azure.com"},
	}

	for _, tc := range testCases {
		endpoint := tc.docEndpoint
		account := &armcosmos.DatabaseAccountGetResults{
			Name: &name,
			Properties: &armcosmos.DatabaseAccountGetProperties{
				DocumentEndpoint: &endpoint,
			},
		}

		conn := p.cosmosAccountToConnection(account)
		if conn.Metadata["endpoint"] != tc.expected {
			t.Errorf("endpoint for %q: expected %q, got %q", tc.docEndpoint, tc.expected, conn.Metadata["endpoint"])
		}
	}
}
