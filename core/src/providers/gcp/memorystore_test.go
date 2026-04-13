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
	"testing"

	redispb "cloud.google.com/go/redis/apiv1/redispb"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func TestMapMemorystoreStatus(t *testing.T) {
	testCases := []struct {
		state    redispb.Instance_State
		expected providers.ConnectionStatus
	}{
		{redispb.Instance_READY, providers.ConnectionStatusAvailable},
		{redispb.Instance_CREATING, providers.ConnectionStatusStarting},
		{redispb.Instance_UPDATING, providers.ConnectionStatusStarting},
		{redispb.Instance_IMPORTING, providers.ConnectionStatusStarting},
		{redispb.Instance_FAILING_OVER, providers.ConnectionStatusStarting},
		{redispb.Instance_MAINTENANCE, providers.ConnectionStatusStarting},
		{redispb.Instance_DELETING, providers.ConnectionStatusDeleting},
	}

	for _, tc := range testCases {
		result := mapMemorystoreStatus(tc.state)
		if result != tc.expected {
			t.Errorf("mapMemorystoreStatus(%s): expected %s, got %s", tc.state, tc.expected, result)
		}
	}
}

func TestMapMemorystoreStatus_UnknownValue(t *testing.T) {
	result := mapMemorystoreStatus(redispb.Instance_STATE_UNSPECIFIED)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapMemorystoreStatus(STATE_UNSPECIFIED): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func newTestMemorystoreProvider() *Provider {
	p, _ := New(&Config{
		ID:                  "test-memorystore",
		Name:                "Test Memorystore",
		ProjectID:           "my-project-123",
		Region:              "us-central1",
		DiscoverMemorystore: true,
	})
	return p
}

func TestMemorystoreInstanceToConnection_HappyPath(t *testing.T) {
	p := newTestMemorystoreProvider()

	instance := &redispb.Instance{
		Name:                  "projects/my-project-123/locations/us-central1/instances/my-redis",
		Host:                  "10.0.0.1",
		Port:                  6379,
		Tier:                  redispb.Instance_BASIC,
		RedisVersion:          "REDIS_7_0",
		State:                 redispb.Instance_READY,
		TransitEncryptionMode: redispb.Instance_DISABLED,
		AuthEnabled:           false,
	}

	conn := p.memorystoreInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.DatabaseType != engine.DatabaseType_Redis {
		t.Errorf("expected Redis, got %s", conn.DatabaseType)
	}
	if conn.Name != "my-redis" {
		t.Errorf("expected name my-redis, got %s", conn.Name)
	}
	if conn.Region != "us-central1" {
		t.Errorf("expected region us-central1, got %s", conn.Region)
	}
	if conn.Status != providers.ConnectionStatusAvailable {
		t.Errorf("expected Available status, got %s", conn.Status)
	}
	if conn.Metadata["endpoint"] != "10.0.0.1" {
		t.Errorf("unexpected endpoint: %s", conn.Metadata["endpoint"])
	}
	if conn.Metadata["port"] != "6379" {
		t.Errorf("unexpected port: %s", conn.Metadata["port"])
	}
	if conn.Metadata["tier"] != "BASIC" {
		t.Errorf("unexpected tier: %s", conn.Metadata["tier"])
	}
	if conn.Metadata["redisVersion"] != "REDIS_7_0" {
		t.Errorf("unexpected redisVersion: %s", conn.Metadata["redisVersion"])
	}
	if conn.Metadata["transitEncryption"] != "false" {
		t.Errorf("unexpected transitEncryption: %s", conn.Metadata["transitEncryption"])
	}
	if conn.Metadata["projectId"] != "my-project-123" {
		t.Errorf("unexpected projectId: %s", conn.Metadata["projectId"])
	}
	if _, ok := conn.Metadata["authTokenEnabled"]; ok {
		t.Error("expected no authTokenEnabled metadata when auth is disabled")
	}
	if conn.ProviderType != providers.ProviderTypeGCP {
		t.Errorf("expected ProviderType GCP, got %s", conn.ProviderType)
	}
	if conn.ProviderID != "test-memorystore" {
		t.Errorf("expected ProviderID test-memorystore, got %s", conn.ProviderID)
	}
}

func TestMemorystoreInstanceToConnection_NoHost(t *testing.T) {
	p := newTestMemorystoreProvider()

	instance := &redispb.Instance{
		Name:  "projects/my-project-123/locations/us-central1/instances/my-redis",
		Host:  "",
		Port:  6379,
		State: redispb.Instance_READY,
	}

	conn := p.memorystoreInstanceToConnection(instance)

	if conn != nil {
		t.Error("expected nil connection when no host")
	}
}

func TestMemorystoreInstanceToConnection_TransitEncryption(t *testing.T) {
	p := newTestMemorystoreProvider()

	instance := &redispb.Instance{
		Name:                  "projects/my-project-123/locations/us-central1/instances/my-redis",
		Host:                  "10.0.0.1",
		Port:                  6379,
		State:                 redispb.Instance_READY,
		TransitEncryptionMode: redispb.Instance_SERVER_AUTHENTICATION,
	}

	conn := p.memorystoreInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Metadata["transitEncryption"] != "true" {
		t.Errorf("expected transitEncryption=true, got %s", conn.Metadata["transitEncryption"])
	}
}

func TestMemorystoreInstanceToConnection_AuthEnabled(t *testing.T) {
	p := newTestMemorystoreProvider()

	instance := &redispb.Instance{
		Name:        "projects/my-project-123/locations/us-central1/instances/my-redis",
		Host:        "10.0.0.1",
		Port:        6379,
		State:       redispb.Instance_READY,
		AuthEnabled: true,
	}

	conn := p.memorystoreInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Metadata["authTokenEnabled"] != "true" {
		t.Errorf("expected authTokenEnabled=true, got %s", conn.Metadata["authTokenEnabled"])
	}
}

func TestMemorystoreInstanceToConnection_StandardHATier(t *testing.T) {
	p := newTestMemorystoreProvider()

	instance := &redispb.Instance{
		Name:  "projects/my-project-123/locations/us-central1/instances/my-redis",
		Host:  "10.0.0.1",
		Port:  6379,
		Tier:  redispb.Instance_STANDARD_HA,
		State: redispb.Instance_READY,
	}

	conn := p.memorystoreInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Metadata["tier"] != "STANDARD_HA" {
		t.Errorf("unexpected tier: %s", conn.Metadata["tier"])
	}
}
