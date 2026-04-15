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

// AzureProviderEnvConfig represents Azure provider configuration from environment variables.
// Authentication is handled by the Azure SDK default credential chain or service principal.
// Example:
//
//	WHODB_AZURE_PROVIDER='[{
//	  "name": "Production Azure",
//	  "subscriptionId": "12345678-1234-1234-1234-123456789012",
//	  "authMethod": "default"
//	}]'
type AzureProviderEnvConfig struct {
	// Name is a human-readable name for this provider.
	Name string `json:"name"`

	// SubscriptionID is the Azure subscription to discover resources in.
	SubscriptionID string `json:"subscriptionId"`

	// TenantID for service principal auth.
	TenantID string `json:"tenantId,omitempty"`

	// ClientID for service principal auth.
	ClientID string `json:"clientId,omitempty"`

	// ClientSecret for service principal auth. Only accepted via environment variable.
	ClientSecret string `json:"clientSecret,omitempty"`

	// AuthMethod determines how to authenticate ("default" or "service-principal").
	AuthMethod string `json:"authMethod,omitempty"`

	// ResourceGroup optionally limits discovery to a single resource group.
	ResourceGroup string `json:"resourceGroup,omitempty"`

	// DiscoverPostgreSQL enables Azure Database for PostgreSQL discovery (defaults to true if omitted).
	DiscoverPostgreSQL *bool `json:"discoverPostgreSQL,omitempty"`

	// DiscoverMySQL enables Azure Database for MySQL discovery (defaults to true if omitted).
	DiscoverMySQL *bool `json:"discoverMySQL,omitempty"`

	// DiscoverRedis enables Azure Cache for Redis discovery (defaults to true if omitted).
	DiscoverRedis *bool `json:"discoverRedis,omitempty"`

	// DiscoverCosmosDB enables Azure Cosmos DB for MongoDB discovery (defaults to true if omitted).
	DiscoverCosmosDB *bool `json:"discoverCosmosDB,omitempty"`
}
