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

package cmd

import (
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/env"
)

func TestCloudCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "cloud" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected cloud command to be registered")
	}
}

func TestCloudCmd_HasExpectedSubcommands(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	command := cloudCmd
	foundProviders := false
	foundConnections := false
	for _, child := range command.Commands() {
		switch child.Name() {
		case "providers":
			foundProviders = true
		case "connections":
			foundConnections = true
		}
	}

	if !foundProviders || !foundConnections {
		t.Fatalf("expected providers and connections subcommands, got providers=%v connections=%v", foundProviders, foundConnections)
	}
}

func TestCloudCmd_HelpIncludesAlphaWarning(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	if !strings.Contains(cloudCmd.Long, "ALPHA WARNING:") {
		t.Fatalf("expected cloud help to include alpha warning, got %q", cloudCmd.Long)
	}
}

func TestCloudProvidersList_DisabledSupportReturnsError(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	originalAWS := env.IsAWSProviderEnabled
	originalAzure := env.IsAzureProviderEnabled
	originalGCP := env.IsGCPProviderEnabled
	t.Cleanup(func() {
		env.IsAWSProviderEnabled = originalAWS
		env.IsAzureProviderEnabled = originalAzure
		env.IsGCPProviderEnabled = originalGCP
	})

	env.IsAWSProviderEnabled = false
	env.IsAzureProviderEnabled = false
	env.IsGCPProviderEnabled = false

	command := newCloudProvidersListCommand()
	command.SetArgs([]string{"--format", "json"})

	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "cloud provider support is disabled") {
		t.Fatalf("expected disabled provider support error, got %v", err)
	}
}

func TestCloudProvidersList_JSONNoProviders(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	originalAWS := env.IsAWSProviderEnabled
	originalAzure := env.IsAzureProviderEnabled
	originalGCP := env.IsGCPProviderEnabled
	t.Cleanup(func() {
		env.IsAWSProviderEnabled = originalAWS
		env.IsAzureProviderEnabled = originalAzure
		env.IsGCPProviderEnabled = originalGCP
	})

	env.IsAWSProviderEnabled = true
	env.IsAzureProviderEnabled = false
	env.IsGCPProviderEnabled = false

	command := newCloudProvidersListCommand()
	stdout, stderr := setCommandBuffers(t, command)
	command.SetArgs([]string{"--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if stdout.String() != "[]\n" {
		t.Fatalf("expected empty JSON array, got %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "ALPHA") {
		t.Fatalf("expected JSON output to suppress alpha warning, got %q", stderr.String())
	}
}

func TestCloudConnectionsList_JSONNoProviders(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	originalAWS := env.IsAWSProviderEnabled
	originalAzure := env.IsAzureProviderEnabled
	originalGCP := env.IsGCPProviderEnabled
	t.Cleanup(func() {
		env.IsAWSProviderEnabled = originalAWS
		env.IsAzureProviderEnabled = originalAzure
		env.IsGCPProviderEnabled = originalGCP
	})

	env.IsAWSProviderEnabled = true
	env.IsAzureProviderEnabled = false
	env.IsGCPProviderEnabled = false

	command := newCloudConnectionsListCommand()
	stdout, _ := setCommandBuffers(t, command)
	command.SetArgs([]string{"--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if stdout.String() != "[]\n" {
		t.Fatalf("expected empty JSON array, got %q", stdout.String())
	}
}
