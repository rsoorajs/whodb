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

// Package cloud exposes the CLI's cloud-provider discovery surface on top of
// the shared provider and settings runtime.
package cloud

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/providers"
	"github.com/clidey/whodb/core/src/settings"
)

var ErrCloudProvidersDisabled = errors.New("cloud provider support is disabled; set WHODB_ENABLE_AWS_PROVIDER=true, WHODB_ENABLE_AZURE_PROVIDER=true, or WHODB_ENABLE_GCP_PROVIDER=true")

// ProviderSummary is the CLI's normalized view of a configured cloud
// provider.
type ProviderSummary struct {
	ID                    string `json:"id"`
	ProviderType          string `json:"providerType"`
	Name                  string `json:"name"`
	Scope                 string `json:"scope,omitempty"`
	Region                string `json:"region"`
	Status                string `json:"status"`
	LastDiscoveryAt       string `json:"lastDiscoveryAt,omitempty"`
	DiscoveredCount       int    `json:"discoveredCount"`
	Error                 string `json:"error,omitempty"`
	AuthMethod            string `json:"authMethod,omitempty"`
	ProfileName           string `json:"profileName,omitempty"`
	SubscriptionID        string `json:"subscriptionId,omitempty"`
	TenantID              string `json:"tenantId,omitempty"`
	ResourceGroup         string `json:"resourceGroup,omitempty"`
	ProjectID             string `json:"projectId,omitempty"`
	ServiceAccountKeyPath string `json:"serviceAccountKeyPath,omitempty"`
	DiscoverRDS           bool   `json:"discoverRDS,omitempty"`
	DiscoverElastiCache   bool   `json:"discoverElastiCache,omitempty"`
	DiscoverDocumentDB    bool   `json:"discoverDocumentDB,omitempty"`
	DiscoverPostgreSQL    bool   `json:"discoverPostgreSQL,omitempty"`
	DiscoverMySQL         bool   `json:"discoverMySQL,omitempty"`
	DiscoverRedis         bool   `json:"discoverRedis,omitempty"`
	DiscoverCosmosDB      bool   `json:"discoverCosmosDB,omitempty"`
	DiscoverCloudSQL      bool   `json:"discoverCloudSQL,omitempty"`
	DiscoverAlloyDB       bool   `json:"discoverAlloyDB,omitempty"`
	DiscoverMemorystore   bool   `json:"discoverMemorystore,omitempty"`
}

// ConnectionSummary is the CLI's normalized view of a discovered cloud
// database or cache resource.
type ConnectionSummary struct {
	ID           string            `json:"id"`
	ProviderType string            `json:"providerType"`
	ProviderID   string            `json:"providerId"`
	Name         string            `json:"name"`
	SourceType   string            `json:"sourceType"`
	Region       string            `json:"region,omitempty"`
	Status       string            `json:"status"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ListProviders returns the configured cloud providers available to the CLI.
func ListProviders() ([]ProviderSummary, error) {
	if err := prepareRuntime(); err != nil {
		return nil, err
	}

	result := make([]ProviderSummary, 0, len(settings.GetAWSProviders())+len(settings.GetAzureProviders())+len(settings.GetGCPProviders()))
	for _, state := range settings.GetAWSProviders() {
		result = append(result, providerSummaryFromAWS(state))
	}
	for _, state := range settings.GetAzureProviders() {
		result = append(result, providerSummaryFromAzure(state))
	}
	for _, state := range settings.GetGCPProviders() {
		result = append(result, providerSummaryFromGCP(state))
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].ProviderType != result[j].ProviderType {
			return result[i].ProviderType < result[j].ProviderType
		}
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].ID < result[j].ID
	})

	return result, nil
}

// ListConnections discovers resources from the configured providers. If
// providerID is non-empty, discovery is limited to that provider.
func ListConnections(ctx context.Context, providerID string) ([]ConnectionSummary, error) {
	if err := prepareRuntime(); err != nil {
		return nil, err
	}

	registry := providers.GetDefaultRegistry()
	var (
		discovered []providers.DiscoveredConnection
		err        error
	)

	if providerID != "" {
		discovered, err = registry.FilterByProvider(ctx, providerID)
	} else {
		discovered, err = registry.DiscoverAll(ctx)
	}

	result := make([]ConnectionSummary, 0, len(discovered))
	for _, conn := range discovered {
		result = append(result, ConnectionSummary{
			ID:           conn.ID,
			ProviderType: string(conn.ProviderType),
			ProviderID:   conn.ProviderID,
			Name:         conn.Name,
			SourceType:   string(conn.DatabaseType),
			Region:       conn.Region,
			Status:       string(conn.Status),
			Metadata:     cloneMetadata(conn.Metadata),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].ProviderType != result[j].ProviderType {
			return result[i].ProviderType < result[j].ProviderType
		}
		if result[i].ProviderID != result[j].ProviderID {
			return result[i].ProviderID < result[j].ProviderID
		}
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].ID < result[j].ID
	})

	return result, err
}

// TestProvider runs the provider connectivity check for the given provider ID
// and returns the updated provider state.
func TestProvider(id string) (ProviderSummary, error) {
	if err := prepareRuntime(); err != nil {
		return ProviderSummary{}, err
	}

	switch providerKind(id) {
	case providers.ProviderTypeAWS:
		_, err := settings.TestAWSProvider(id)
		if err != nil {
			state, stateErr := settings.GetAWSProvider(id)
			if stateErr != nil {
				return ProviderSummary{}, err
			}
			return providerSummaryFromAWS(state), err
		}
		state, err := settings.GetAWSProvider(id)
		if err != nil {
			return ProviderSummary{}, err
		}
		return providerSummaryFromAWS(state), nil
	case providers.ProviderTypeAzure:
		_, err := settings.TestAzureProvider(id)
		if err != nil {
			state, stateErr := settings.GetAzureProvider(id)
			if stateErr != nil {
				return ProviderSummary{}, err
			}
			return providerSummaryFromAzure(state), err
		}
		state, err := settings.GetAzureProvider(id)
		if err != nil {
			return ProviderSummary{}, err
		}
		return providerSummaryFromAzure(state), nil
	case providers.ProviderTypeGCP:
		_, err := settings.TestGCPProvider(id)
		if err != nil {
			state, stateErr := settings.GetGCPProvider(id)
			if stateErr != nil {
				return ProviderSummary{}, err
			}
			return providerSummaryFromGCP(state), err
		}
		state, err := settings.GetGCPProvider(id)
		if err != nil {
			return ProviderSummary{}, err
		}
		return providerSummaryFromGCP(state), nil
	default:
		return ProviderSummary{}, fmt.Errorf("cloud provider %q not found", id)
	}
}

// RefreshProvider re-discovers resources for the given provider ID and returns
// the updated provider state.
func RefreshProvider(id string) (ProviderSummary, error) {
	if err := prepareRuntime(); err != nil {
		return ProviderSummary{}, err
	}

	switch providerKind(id) {
	case providers.ProviderTypeAWS:
		state, err := settings.RefreshAWSProvider(id)
		if state == nil {
			return ProviderSummary{}, err
		}
		return providerSummaryFromAWS(state), err
	case providers.ProviderTypeAzure:
		state, err := settings.RefreshAzureProvider(id)
		if state == nil {
			return ProviderSummary{}, err
		}
		return providerSummaryFromAzure(state), err
	case providers.ProviderTypeGCP:
		state, err := settings.RefreshGCPProvider(id)
		if state == nil {
			return ProviderSummary{}, err
		}
		return providerSummaryFromGCP(state), err
	default:
		return ProviderSummary{}, fmt.Errorf("cloud provider %q not found", id)
	}
}

// RefreshAllProviders re-discovers resources for every configured provider and
// returns the updated provider states.
func RefreshAllProviders() ([]ProviderSummary, error) {
	providersList, err := ListProviders()
	if err != nil {
		return nil, err
	}

	result := make([]ProviderSummary, 0, len(providersList))
	var errs []error
	for _, provider := range providersList {
		updated, refreshErr := RefreshProvider(provider.ID)
		if refreshErr != nil {
			errs = append(errs, refreshErr)
		}
		if updated.ID != "" {
			result = append(result, updated)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].ProviderType != result[j].ProviderType {
			return result[i].ProviderType < result[j].ProviderType
		}
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].ID < result[j].ID
	})

	return result, errors.Join(errs...)
}

func prepareRuntime() error {
	if !cloudProvidersEnabled() {
		return ErrCloudProvidersDisabled
	}

	var errs []error

	if err := settings.LoadProvidersFromFile(); err != nil {
		errs = append(errs, fmt.Errorf("load AWS providers: %w", err))
	}
	if err := settings.InitAWSProvidersFromEnv(); err != nil {
		errs = append(errs, fmt.Errorf("initialize AWS providers from environment: %w", err))
	}
	if err := settings.LoadAzureProvidersFromFile(); err != nil {
		errs = append(errs, fmt.Errorf("load Azure providers: %w", err))
	}
	if err := settings.InitAzureProvidersFromEnv(); err != nil {
		errs = append(errs, fmt.Errorf("initialize Azure providers from environment: %w", err))
	}
	if err := settings.LoadGCPProvidersFromFile(); err != nil {
		errs = append(errs, fmt.Errorf("load GCP providers: %w", err))
	}
	if err := settings.InitGCPProvidersFromEnv(); err != nil {
		errs = append(errs, fmt.Errorf("initialize GCP providers from environment: %w", err))
	}

	return errors.Join(errs...)
}

func cloudProvidersEnabled() bool {
	return env.IsAWSProviderEnabled || env.IsAzureProviderEnabled || env.IsGCPProviderEnabled
}

func providerKind(id string) providers.ProviderType {
	if _, err := settings.GetAWSProvider(id); err == nil {
		return providers.ProviderTypeAWS
	}
	if _, err := settings.GetAzureProvider(id); err == nil {
		return providers.ProviderTypeAzure
	}
	if _, err := settings.GetGCPProvider(id); err == nil {
		return providers.ProviderTypeGCP
	}
	return ""
}

func providerSummaryFromAWS(state *settings.AWSProviderState) ProviderSummary {
	if state == nil || state.Config == nil {
		return ProviderSummary{}
	}

	summary := ProviderSummary{
		ID:                  state.Config.ID,
		ProviderType:        string(providers.ProviderTypeAWS),
		Name:                state.Config.Name,
		Scope:               state.Config.ProfileName,
		Region:              state.Config.Region,
		Status:              state.Status,
		DiscoveredCount:     state.DiscoveredCount,
		Error:               state.Error,
		AuthMethod:          state.Config.AuthMethod,
		ProfileName:         state.Config.ProfileName,
		DiscoverRDS:         state.Config.DiscoverRDS,
		DiscoverElastiCache: state.Config.DiscoverElastiCache,
		DiscoverDocumentDB:  state.Config.DiscoverDocumentDB,
	}
	if summary.Scope == "" {
		summary.Scope = state.Config.AuthMethod
	}
	if state.LastDiscoveryAt != nil {
		summary.LastDiscoveryAt = state.LastDiscoveryAt.UTC().Format(time.RFC3339)
	}
	return summary
}

func providerSummaryFromAzure(state *settings.AzureProviderState) ProviderSummary {
	if state == nil || state.Config == nil {
		return ProviderSummary{}
	}

	summary := ProviderSummary{
		ID:                 state.Config.ID,
		ProviderType:       string(providers.ProviderTypeAzure),
		Name:               state.Config.Name,
		Scope:              state.Config.SubscriptionID,
		Region:             state.Config.SubscriptionID,
		Status:             state.Status,
		DiscoveredCount:    state.DiscoveredCount,
		Error:              state.Error,
		AuthMethod:         state.Config.AuthMethod,
		SubscriptionID:     state.Config.SubscriptionID,
		TenantID:           state.Config.TenantID,
		ResourceGroup:      state.Config.ResourceGroup,
		DiscoverPostgreSQL: state.Config.DiscoverPostgreSQL,
		DiscoverMySQL:      state.Config.DiscoverMySQL,
		DiscoverRedis:      state.Config.DiscoverRedis,
		DiscoverCosmosDB:   state.Config.DiscoverCosmosDB,
	}
	if state.LastDiscoveryAt != nil {
		summary.LastDiscoveryAt = state.LastDiscoveryAt.UTC().Format(time.RFC3339)
	}
	return summary
}

func providerSummaryFromGCP(state *settings.GCPProviderState) ProviderSummary {
	if state == nil || state.Config == nil {
		return ProviderSummary{}
	}

	summary := ProviderSummary{
		ID:                    state.Config.ID,
		ProviderType:          string(providers.ProviderTypeGCP),
		Name:                  state.Config.Name,
		Scope:                 state.Config.ProjectID,
		Region:                state.Config.Region,
		Status:                state.Status,
		DiscoveredCount:       state.DiscoveredCount,
		Error:                 state.Error,
		AuthMethod:            state.Config.AuthMethod,
		ProjectID:             state.Config.ProjectID,
		ServiceAccountKeyPath: state.Config.ServiceAccountKeyPath,
		DiscoverCloudSQL:      state.Config.DiscoverCloudSQL,
		DiscoverAlloyDB:       state.Config.DiscoverAlloyDB,
		DiscoverMemorystore:   state.Config.DiscoverMemorystore,
	}
	if state.LastDiscoveryAt != nil {
		summary.LastDiscoveryAt = state.LastDiscoveryAt.UTC().Format(time.RFC3339)
	}
	return summary
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}
