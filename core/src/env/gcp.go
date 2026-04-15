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

package env

// GCPProviderEnvConfig represents GCP provider configuration from environment variables.
// Authentication is handled by Application Default Credentials or a service account key file.
// Example:
//
//	WHODB_GCP_PROVIDER='[{
//	  "name": "Production GCP",
//	  "projectId": "my-project-123",
//	  "region": "us-central1",
//	  "serviceAccountKeyPath": "/path/to/key.json"
//	}]'
type GCPProviderEnvConfig struct {
	// Name is a human-readable name for this provider.
	Name string `json:"name"`

	// ProjectID is the GCP project to discover resources in.
	ProjectID string `json:"projectId"`

	// Region is the GCP region to discover resources in.
	Region string `json:"region"`

	// ServiceAccountKeyPath for service account key auth. If set, uses the key file.
	// If empty, uses Application Default Credentials.
	ServiceAccountKeyPath string `json:"serviceAccountKeyPath,omitempty"`

	// DiscoverCloudSQL enables Cloud SQL instance discovery (defaults to true if omitted).
	DiscoverCloudSQL *bool `json:"discoverCloudSQL,omitempty"`

	// DiscoverAlloyDB enables AlloyDB cluster discovery (defaults to true if omitted).
	DiscoverAlloyDB *bool `json:"discoverAlloyDB,omitempty"`

	// DiscoverMemorystore enables Memorystore discovery (defaults to true if omitted).
	DiscoverMemorystore *bool `json:"discoverMemorystore,omitempty"`
}
