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
	"errors"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// AuthMethod defines how to authenticate with Google Cloud.
type AuthMethod string

const (
	// AuthMethodDefault uses Application Default Credentials (ADC).
	// This automatically handles: GOOGLE_APPLICATION_CREDENTIALS env var,
	// gcloud CLI credentials, GCE/GKE metadata server, etc.
	AuthMethodDefault AuthMethod = "default"

	// AuthMethodServiceAccountKey uses an explicit service account JSON key file.
	AuthMethodServiceAccountKey AuthMethod = "service-account-key"
)

const (
	AdvancedKeyAuthMethod            = "Auth Method"
	AdvancedKeyServiceAccountKeyPath = "Service Account Key Path"
	AdvancedKeyProjectID             = "Project ID"
)

// GCPCredentialConfig holds parsed GCP configuration extracted from WhoDB credentials.
type GCPCredentialConfig struct {
	ProjectID             string
	Region                string
	AuthMethod            AuthMethod
	ServiceAccountKeyPath string
}

// ParseFromWhoDB extracts GCP configuration from WhoDB credentials.
// Returns an error if required fields are missing or invalid.
func ParseFromWhoDB(creds *engine.Credentials) (*GCPCredentialConfig, error) {
	if creds == nil {
		return nil, errors.New("credentials cannot be nil")
	}

	region := strings.TrimSpace(creds.Hostname)
	if region == "" {
		region = common.GetRecordValueOrDefault(creds.Advanced, "Region", "")
	}

	projectID := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyProjectID, "")
	authMethodStr := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyAuthMethod, string(AuthMethodDefault))
	serviceAccountKeyPath := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyServiceAccountKeyPath, "")

	config := &GCPCredentialConfig{
		ProjectID:             projectID,
		Region:                region,
		AuthMethod:            AuthMethod(strings.ToLower(strings.TrimSpace(authMethodStr))),
		ServiceAccountKeyPath: serviceAccountKeyPath,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that the configuration is valid for the selected auth method.
func (c *GCPCredentialConfig) Validate() error {
	if c.Region == "" {
		return ErrRegionRequired
	}
	if c.ProjectID == "" {
		return ErrProjectIDRequired
	}

	switch c.AuthMethod {
	case AuthMethodServiceAccountKey:
		if c.ServiceAccountKeyPath == "" {
			return ErrServiceAccountKeyPathRequired
		}
	case AuthMethodDefault:
	default:
		return ErrInvalidAuthMethod
	}

	return nil
}

// IsServiceAccountKeyAuth returns true if using explicit service account key authentication.
func (c *GCPCredentialConfig) IsServiceAccountKeyAuth() bool {
	return c.AuthMethod == AuthMethodServiceAccountKey
}
