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
	"errors"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// AuthMethod determines how to authenticate with Azure.
type AuthMethod string

const (
	// AuthMethodDefault uses the Azure SDK's default credential chain.
	// This automatically handles: environment variables, managed identity, Azure CLI, etc.
	AuthMethodDefault AuthMethod = "default"

	// AuthMethodServicePrincipal uses explicit Tenant ID, Client ID, and Client Secret.
	AuthMethodServicePrincipal AuthMethod = "service-principal"
)

const (
	AdvancedKeyAuthMethod     = "Auth Method"
	AdvancedKeyTenantID       = "Tenant ID"
	AdvancedKeyClientID       = "Client ID"
	AdvancedKeyClientSecret   = "Client Secret"
	AdvancedKeyResourceGroup  = "Resource Group"
	AdvancedKeySubscriptionID = "Subscription ID"
)

// AzureCredentialConfig holds parsed Azure configuration extracted from WhoDB credentials.
type AzureCredentialConfig struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
	AuthMethod     AuthMethod
	ResourceGroup  string
}

// ParseFromWhoDB extracts Azure configuration from WhoDB credentials.
// Returns an error if required fields are missing or invalid.
func ParseFromWhoDB(creds *engine.Credentials) (*AzureCredentialConfig, error) {
	if creds == nil {
		return nil, errors.New("credentials cannot be nil")
	}

	subscriptionID := strings.TrimSpace(creds.Hostname)
	if subscriptionID == "" {
		subscriptionID = common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeySubscriptionID, "")
	}

	authMethodStr := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyAuthMethod, string(AuthMethodDefault))
	tenantID := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyTenantID, "")
	clientID := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyClientID, "")
	clientSecret := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyClientSecret, "")
	resourceGroup := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyResourceGroup, "")

	config := &AzureCredentialConfig{
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		AuthMethod:     AuthMethod(strings.ToLower(strings.TrimSpace(authMethodStr))),
		ResourceGroup:  resourceGroup,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that the configuration is valid for the selected auth method.
func (c *AzureCredentialConfig) Validate() error {
	if c.SubscriptionID == "" {
		return ErrSubscriptionRequired
	}

	switch c.AuthMethod {
	case AuthMethodServicePrincipal:
		if c.TenantID == "" {
			return ErrTenantIDRequired
		}
		if c.ClientID == "" {
			return ErrClientIDRequired
		}
		if c.ClientSecret == "" {
			return ErrClientSecretRequired
		}
	case AuthMethodDefault:
	default:
		return ErrInvalidAuthMethod
	}

	return nil
}

// IsServicePrincipalAuth returns true if using explicit service principal credentials.
func (c *AzureCredentialConfig) IsServicePrincipalAuth() bool {
	return c.AuthMethod == AuthMethodServicePrincipal
}
