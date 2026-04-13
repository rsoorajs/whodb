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

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func TestMapRedisStatus(t *testing.T) {
	testCases := []struct {
		state    armredis.ProvisioningState
		expected providers.ConnectionStatus
	}{
		{armredis.ProvisioningStateSucceeded, providers.ConnectionStatusAvailable},
		{armredis.ProvisioningStateCreating, providers.ConnectionStatusStarting},
		{armredis.ProvisioningStateProvisioning, providers.ConnectionStatusStarting},
		{armredis.ProvisioningStateLinking, providers.ConnectionStatusStarting},
		{armredis.ProvisioningStateScaling, providers.ConnectionStatusStarting},
		{armredis.ProvisioningStateUpdating, providers.ConnectionStatusStarting},
		{armredis.ProvisioningStateRecoveringScaleFailure, providers.ConnectionStatusStarting},
		{armredis.ProvisioningStateDisabled, providers.ConnectionStatusStopped},
		{armredis.ProvisioningStateUnlinking, providers.ConnectionStatusStopped},
		{armredis.ProvisioningStateUnprovisioning, providers.ConnectionStatusStopped},
		{armredis.ProvisioningStateDeleting, providers.ConnectionStatusDeleting},
		{armredis.ProvisioningStateFailed, providers.ConnectionStatusFailed},
	}

	for _, tc := range testCases {
		state := tc.state
		result := mapRedisStatus(&state)
		if result != tc.expected {
			t.Errorf("mapRedisStatus(%s): expected %s, got %s", tc.state, tc.expected, result)
		}
	}
}

func TestMapRedisStatus_Nil(t *testing.T) {
	result := mapRedisStatus(nil)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapRedisStatus(nil): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func TestMapRedisStatus_UnknownValue(t *testing.T) {
	unknown := armredis.ProvisioningState("SomeUnknownState")
	result := mapRedisStatus(&unknown)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapRedisStatus(SomeUnknownState): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func newTestRedisProvider() *Provider {
	p, _ := New(&Config{
		ID:             "test-redis",
		Name:           "Test Redis",
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
		DiscoverRedis:  true,
	})
	return p
}

func TestRedisCacheToConnection_HappyPath(t *testing.T) {
	p := newTestRedisProvider()
	name := "my-redis"
	location := "eastus"
	hostname := "my-redis.redis.cache.windows.net"
	sslPort := int32(6380)
	nonSslPort := int32(6379)
	enableNonSsl := true
	provState := armredis.ProvisioningStateSucceeded
	resourceID := "/subscriptions/00000000/resourceGroups/my-rg/providers/Microsoft.Cache/Redis/my-redis"

	cache := &armredis.ResourceInfo{
		Name:     &name,
		Location: &location,
		ID:       &resourceID,
		Properties: &armredis.Properties{
			HostName:          &hostname,
			SSLPort:           &sslPort,
			Port:              &nonSslPort,
			EnableNonSSLPort:  &enableNonSsl,
			ProvisioningState: &provState,
		},
	}

	conn := p.redisCacheToConnection(cache)

	if conn.DatabaseType != engine.DatabaseType_Redis {
		t.Errorf("expected Redis, got %s", conn.DatabaseType)
	}
	if conn.Name != "my-redis" {
		t.Errorf("expected name my-redis, got %s", conn.Name)
	}
	if conn.Region != "eastus" {
		t.Errorf("expected region eastus, got %s", conn.Region)
	}
	if conn.Status != providers.ConnectionStatusAvailable {
		t.Errorf("expected Available status, got %s", conn.Status)
	}
	if conn.Metadata["endpoint"] != "my-redis.redis.cache.windows.net" {
		t.Errorf("unexpected endpoint: %s", conn.Metadata["endpoint"])
	}
	if conn.Metadata["port"] != "6380" {
		t.Errorf("unexpected port: %s", conn.Metadata["port"])
	}
	if conn.Metadata["enableNonSslPort"] != "true" {
		t.Errorf("unexpected enableNonSslPort: %s", conn.Metadata["enableNonSslPort"])
	}
	if conn.Metadata["nonSslPort"] != "6379" {
		t.Errorf("unexpected nonSslPort: %s", conn.Metadata["nonSslPort"])
	}
	if conn.Metadata["resourceGroup"] != "my-rg" {
		t.Errorf("unexpected resourceGroup: %s", conn.Metadata["resourceGroup"])
	}
	if conn.ProviderType != providers.ProviderTypeAzure {
		t.Errorf("expected ProviderType Azure, got %s", conn.ProviderType)
	}
	if conn.ProviderID != "test-redis" {
		t.Errorf("expected ProviderID test-redis, got %s", conn.ProviderID)
	}
}

func TestRedisCacheToConnection_NilProperties(t *testing.T) {
	p := newTestRedisProvider()
	name := "my-redis"
	location := "eastus"

	cache := &armredis.ResourceInfo{
		Name:       &name,
		Location:   &location,
		Properties: nil,
	}

	conn := p.redisCacheToConnection(cache)

	if conn.Status != providers.ConnectionStatusUnknown {
		t.Errorf("expected Unknown status for nil properties, got %s", conn.Status)
	}
}

func TestRedisCacheToConnection_SSLOnly(t *testing.T) {
	p := newTestRedisProvider()
	name := "my-redis"
	sslPort := int32(6380)
	enableNonSsl := false
	provState := armredis.ProvisioningStateSucceeded

	cache := &armredis.ResourceInfo{
		Name: &name,
		Properties: &armredis.Properties{
			SSLPort:           &sslPort,
			EnableNonSSLPort:  &enableNonSsl,
			ProvisioningState: &provState,
		},
	}

	conn := p.redisCacheToConnection(cache)

	if conn.Metadata["port"] != "6380" {
		t.Errorf("expected SSL port 6380, got %s", conn.Metadata["port"])
	}
	if conn.Metadata["enableNonSslPort"] != "false" {
		t.Errorf("expected enableNonSslPort=false, got %s", conn.Metadata["enableNonSslPort"])
	}
	if _, ok := conn.Metadata["nonSslPort"]; ok {
		t.Error("expected no nonSslPort metadata when non-SSL is disabled")
	}
}
