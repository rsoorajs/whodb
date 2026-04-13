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

	alloydbpb "cloud.google.com/go/alloydb/apiv1/alloydbpb"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func TestMapAlloyDBStatus(t *testing.T) {
	testCases := []struct {
		state    alloydbpb.Instance_State
		expected providers.ConnectionStatus
	}{
		{alloydbpb.Instance_READY, providers.ConnectionStatusAvailable},
		{alloydbpb.Instance_CREATING, providers.ConnectionStatusStarting},
		{alloydbpb.Instance_PROMOTING, providers.ConnectionStatusStarting},
		{alloydbpb.Instance_BOOTSTRAPPING, providers.ConnectionStatusStarting},
		{alloydbpb.Instance_STOPPED, providers.ConnectionStatusStopped},
		{alloydbpb.Instance_DELETING, providers.ConnectionStatusDeleting},
		{alloydbpb.Instance_FAILED, providers.ConnectionStatusFailed},
		{alloydbpb.Instance_MAINTENANCE, providers.ConnectionStatusFailed},
	}

	for _, tc := range testCases {
		result := mapAlloyDBStatus(tc.state)
		if result != tc.expected {
			t.Errorf("mapAlloyDBStatus(%s): expected %s, got %s", tc.state, tc.expected, result)
		}
	}
}

func TestMapAlloyDBStatus_UnknownValue(t *testing.T) {
	result := mapAlloyDBStatus(alloydbpb.Instance_STATE_UNSPECIFIED)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapAlloyDBStatus(STATE_UNSPECIFIED): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func TestExtractResourceName(t *testing.T) {
	testCases := []struct {
		fullName string
		expected string
	}{
		{"projects/myproject/locations/us-central1/clusters/mycluster", "mycluster"},
		{"projects/myproject/locations/us-central1/clusters/mycluster/instances/myinstance", "myinstance"},
		{"simple-name", "simple-name"},
		{"a/b/c", "c"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := extractResourceName(tc.fullName)
		if result != tc.expected {
			t.Errorf("extractResourceName(%s): expected %q, got %q", tc.fullName, tc.expected, result)
		}
	}
}

func newTestAlloyDBProvider() *Provider {
	p, _ := New(&Config{
		ID:              "test-alloydb",
		Name:            "Test AlloyDB",
		ProjectID:       "my-project-123",
		Region:          "us-central1",
		DiscoverAlloyDB: true,
	})
	return p
}

func TestAlloyDBInstanceToConnection_Writer(t *testing.T) {
	p := newTestAlloyDBProvider()

	instance := &alloydbpb.Instance{
		Name:            "projects/my-project-123/locations/us-central1/clusters/mycluster/instances/writer-0",
		State:           alloydbpb.Instance_READY,
		InstanceType:    alloydbpb.Instance_PRIMARY,
		IpAddress:       "10.0.0.1",
		PublicIpAddress: "34.1.2.3",
	}

	conn := p.alloyDBInstanceToConnection(instance, "mycluster")

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.DatabaseType != engine.DatabaseType_Postgres {
		t.Errorf("expected Postgres, got %s", conn.DatabaseType)
	}
	if conn.Name != "mycluster/writer-0 (writer)" {
		t.Errorf("expected name mycluster/writer-0 (writer), got %s", conn.Name)
	}
	if conn.Region != "us-central1" {
		t.Errorf("expected region us-central1, got %s", conn.Region)
	}
	if conn.Status != providers.ConnectionStatusAvailable {
		t.Errorf("expected Available status, got %s", conn.Status)
	}
	if conn.Metadata["endpoint"] != "34.1.2.3" {
		t.Errorf("unexpected endpoint: %s (expected public IP)", conn.Metadata["endpoint"])
	}
	if conn.Metadata["port"] != "5432" {
		t.Errorf("unexpected port: %s", conn.Metadata["port"])
	}
	if conn.Metadata["endpointType"] != "writer" {
		t.Errorf("unexpected endpointType: %s", conn.Metadata["endpointType"])
	}
	if conn.Metadata["clusterName"] != "mycluster" {
		t.Errorf("unexpected clusterName: %s", conn.Metadata["clusterName"])
	}
	if conn.Metadata["projectId"] != "my-project-123" {
		t.Errorf("unexpected projectId: %s", conn.Metadata["projectId"])
	}
	if conn.ProviderType != providers.ProviderTypeGCP {
		t.Errorf("expected ProviderType GCP, got %s", conn.ProviderType)
	}
	if conn.ProviderID != "test-alloydb" {
		t.Errorf("expected ProviderID test-alloydb, got %s", conn.ProviderID)
	}
}

func TestAlloyDBInstanceToConnection_ReadPool(t *testing.T) {
	p := newTestAlloyDBProvider()

	instance := &alloydbpb.Instance{
		Name:         "projects/my-project-123/locations/us-central1/clusters/mycluster/instances/reader-0",
		State:        alloydbpb.Instance_READY,
		InstanceType: alloydbpb.Instance_READ_POOL,
		IpAddress:    "10.0.0.2",
	}

	conn := p.alloyDBInstanceToConnection(instance, "mycluster")

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Name != "mycluster/reader-0 (reader)" {
		t.Errorf("expected name with (reader) suffix, got %s", conn.Name)
	}
	if conn.Metadata["endpointType"] != "reader" {
		t.Errorf("expected endpointType reader, got %s", conn.Metadata["endpointType"])
	}
}

func TestAlloyDBInstanceToConnection_Secondary(t *testing.T) {
	p := newTestAlloyDBProvider()

	instance := &alloydbpb.Instance{
		Name:         "projects/my-project-123/locations/us-central1/clusters/mycluster/instances/secondary-0",
		State:        alloydbpb.Instance_READY,
		InstanceType: alloydbpb.Instance_SECONDARY,
		IpAddress:    "10.0.0.3",
	}

	conn := p.alloyDBInstanceToConnection(instance, "mycluster")

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Name != "mycluster/secondary-0 (cross-region reader)" {
		t.Errorf("expected name with (cross-region reader) suffix, got %s", conn.Name)
	}
	if conn.Metadata["endpointType"] != "reader" {
		t.Errorf("expected endpointType reader for secondary, got %s", conn.Metadata["endpointType"])
	}
}

func TestAlloyDBInstanceToConnection_NoIPAddress(t *testing.T) {
	p := newTestAlloyDBProvider()

	instance := &alloydbpb.Instance{
		Name:         "projects/my-project-123/locations/us-central1/clusters/mycluster/instances/writer-0",
		State:        alloydbpb.Instance_READY,
		InstanceType: alloydbpb.Instance_PRIMARY,
		IpAddress:    "",
	}

	conn := p.alloyDBInstanceToConnection(instance, "mycluster")

	if conn != nil {
		t.Error("expected nil connection when no IP address")
	}
}

func TestAlloyDBInstanceToConnection_PrivateIPFallback(t *testing.T) {
	p := newTestAlloyDBProvider()

	instance := &alloydbpb.Instance{
		Name:            "projects/my-project-123/locations/us-central1/clusters/mycluster/instances/writer-0",
		State:           alloydbpb.Instance_READY,
		InstanceType:    alloydbpb.Instance_PRIMARY,
		IpAddress:       "10.0.0.1",
		PublicIpAddress: "",
	}

	conn := p.alloyDBInstanceToConnection(instance, "mycluster")

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Metadata["endpoint"] != "10.0.0.1" {
		t.Errorf("expected private IP fallback 10.0.0.1, got %s", conn.Metadata["endpoint"])
	}
}
