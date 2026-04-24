// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package src

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	coreproviders "github.com/clidey/whodb/core/src/llm/providers"
	"github.com/clidey/whodb/core/src/types"
)

func TestInitializeEngineCollectsRegisteredPlugins(t *testing.T) {
	// Register test plugins via the global registry (simulates what init() does in each plugin package)
	engine.RegisterPlugin(&engine.Plugin{Type: "TestDB1"})
	engine.RegisterPlugin(&engine.Plugin{Type: "TestDB2"})

	t.Cleanup(func() {
		MainEngine = nil
	})

	eng := InitializeEngine()

	if eng.Choose("TestDB1") == nil {
		t.Fatalf("expected TestDB1 to be registered via global registry")
	}
	if eng.Choose("TestDB2") == nil {
		t.Fatalf("expected TestDB2 to be registered via global registry")
	}
}

func TestInitializeEngineRegistersGenericProvidersAndSampleProfile(t *testing.T) {
	originalProviders := env.GenericProviders
	env.GenericProviders = nil
	t.Cleanup(func() {
		env.GenericProviders = originalProviders
		MainEngine = nil
	})

	t.Setenv("WHODB_AI_GENERIC_SRC_INIT_PROVIDER_NAME", "Source Init Provider")
	t.Setenv("WHODB_AI_GENERIC_SRC_INIT_PROVIDER_TYPE", "openai-generic")
	t.Setenv("WHODB_AI_GENERIC_SRC_INIT_PROVIDER_BASE_URL", "https://llm.example.com/v1")
	t.Setenv("WHODB_AI_GENERIC_SRC_INIT_PROVIDER_API_KEY", "test-key")
	t.Setenv("WHODB_AI_GENERIC_SRC_INIT_PROVIDER_MODELS", "model-a,model-b")

	eng := InitializeEngine()

	if !coreproviders.HasProvider(coreproviders.LLMType("src_init_provider")) {
		t.Fatal("expected generic provider from environment to be registered with the LLM provider registry")
	}

	if len(env.GenericProviders) != 1 {
		t.Fatalf("expected exactly one generic provider to be published to env.GenericProviders, got %d", len(env.GenericProviders))
	}

	genericProvider := env.GenericProviders[0]
	if genericProvider.ProviderId != "src_init_provider" {
		t.Fatalf("expected provider id src_init_provider, got %q", genericProvider.ProviderId)
	}
	if genericProvider.Name != "Source Init Provider" {
		t.Fatalf("expected provider name to be preserved, got %q", genericProvider.Name)
	}
	if len(genericProvider.Models) != 2 || genericProvider.Models[0] != "model-a" || genericProvider.Models[1] != "model-b" {
		t.Fatalf("expected provider models to be parsed from env, got %#v", genericProvider.Models)
	}

	for _, profile := range eng.LoginProfiles {
		if profile.Alias == "Sample SQLite Database" &&
			profile.Type == string(engine.DatabaseType_Sqlite3) &&
			profile.Source == "builtin" &&
			profile.IsProfile {
			return
		}
	}

	t.Fatal("expected InitializeEngine to add the built-in SQLite sample profile")
}

func TestGetLoginProfilesMergesSources(t *testing.T) {
	t.Cleanup(func() {
		MainEngine = nil
	})

	MainEngine = &engine.Engine{}
	MainEngine.RegistryPlugin(&engine.Plugin{Type: "Test"})

	MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Alias:     "saved-profile",
		Hostname:  "host1",
		Username:  "alice",
		Password:  "pw",
		Database:  "db1",
		IsProfile: true,
		Type:      "Test",
	})

	MainEngine.RegisterProfileRetriever(func() ([]types.DatabaseCredentials, error) {
		return []types.DatabaseCredentials{{
			Hostname: "host2",
			Username: "retrieved",
			Database: "db2",
			Type:     "Test",
		}}, nil
	})

	envCreds := []types.DatabaseCredentials{{
		Hostname: "env-host",
		Username: "env-user",
		Database: "env-db",
		Password: "env-pw",
	}}
	envValue, err := json.Marshal(envCreds)
	if err != nil {
		t.Fatalf("failed to marshal env credentials: %v", err)
	}
	t.Setenv("WHODB_POSTGRES", string(envValue))

	profiles := GetLoginProfiles()
	foundEnvProfile := false
	foundSavedProfile := false
	foundRetrievedProfile := false
	for _, profile := range profiles {
		if profile.Hostname == "env-host" && profile.IsProfile {
			foundEnvProfile = true
		}
		if profile.Alias == "saved-profile" {
			foundSavedProfile = true
		}
		if profile.Hostname == "host2" && profile.Username == "retrieved" {
			foundRetrievedProfile = true
		}
	}
	if !foundEnvProfile {
		t.Fatalf("expected env-provided profile to be marked as profile and returned")
	}
	if !foundSavedProfile {
		t.Fatalf("expected stored profile to be returned")
	}
	if !foundRetrievedProfile {
		t.Fatalf("expected retriever profile to be returned")
	}
}

func TestGetLoginProfilesSkipsFailingRetrieversAndUsesNumberedEnvFallback(t *testing.T) {
	t.Cleanup(func() {
		MainEngine = nil
	})

	MainEngine = &engine.Engine{}
	MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Alias:    "stored-profile",
		Hostname: "stored-host",
		Type:     "StoredDB",
	})

	MainEngine.RegisterProfileRetriever(func() ([]types.DatabaseCredentials, error) {
		return nil, errors.New("boom")
	})
	MainEngine.RegisterProfileRetriever(func() ([]types.DatabaseCredentials, error) {
		return []types.DatabaseCredentials{{
			Alias:    "retrieved-profile",
			Hostname: "retrieved-host",
			Type:     "RetrievedDB",
		}}, nil
	})

	t.Setenv("WHODB_MYSQL", "")
	t.Setenv("WHODB_MYSQL_1", `{"host":"mysql-one","user":"alice","database":"northwind"}`)
	t.Setenv("WHODB_MYSQL_2", `{"host":"mysql-two","user":"bob","database":"warehouse"}`)

	profiles := GetLoginProfiles()
	if len(profiles) < 4 {
		t.Fatalf("expected at least stored, retrieved, and two numbered env profiles, got %d: %#v", len(profiles), profiles)
	}

	if profiles[0].Alias != "stored-profile" || profiles[0].Hostname != "stored-host" {
		t.Fatalf("expected stored profile to remain first, got %#v", profiles[0])
	}

	if profiles[1].Alias != "retrieved-profile" || profiles[1].Hostname != "retrieved-host" {
		t.Fatalf("expected successful retriever profile after stored profiles, got %#v", profiles[1])
	}

	expectedHosts := map[string]bool{"mysql-one": false, "mysql-two": false}
	for _, profile := range profiles {
		if _, ok := expectedHosts[profile.Hostname]; !ok {
			continue
		}
		if profile.Type != string(engine.DatabaseType_MySQL) {
			t.Fatalf("expected env profile for %q to be typed as MySQL, got %q", profile.Hostname, profile.Type)
		}
		if !profile.IsProfile {
			t.Fatalf("expected env profile for %q to be marked as a profile", profile.Hostname)
		}
		expectedHosts[profile.Hostname] = true
	}

	for host, found := range expectedHosts {
		if !found {
			t.Fatalf("expected numbered env profile for host %q to be returned, got %#v", host, profiles)
		}
	}
}

func TestGetLoginProfilesIncludesCatalogAliasesFromEnv(t *testing.T) {
	t.Cleanup(func() {
		MainEngine = nil
	})

	MainEngine = &engine.Engine{}

	envCreds := []types.DatabaseCredentials{{
		Hostname: "ferret-host",
		Username: "ferret-user",
		Database: "ferret-db",
	}}
	envValue, err := json.Marshal(envCreds)
	if err != nil {
		t.Fatalf("failed to marshal env credentials: %v", err)
	}
	t.Setenv("WHODB_FERRETDB", string(envValue))

	profiles := GetLoginProfiles()
	for _, profile := range profiles {
		if profile.Type == string(engine.DatabaseType_FerretDB) &&
			profile.Hostname == "ferret-host" &&
			profile.IsProfile {
			return
		}
	}

	t.Fatal("expected FerretDB env profile to be returned from the shared catalog")
}

func TestGetLoginProfileIdPrioritizesFields(t *testing.T) {
	profile := types.DatabaseCredentials{
		CustomId: "custom-id",
		Alias:    "alias-id",
		Username: "user",
		Hostname: "host",
		Database: "db",
	}
	if got := GetLoginProfileId(0, profile); got != "custom-id" {
		t.Fatalf("expected custom id to take priority, got %s", got)
	}

	profile.CustomId = ""
	if got := GetLoginProfileId(1, profile); got != "alias-id" {
		t.Fatalf("expected alias to be used when custom id is empty, got %s", got)
	}

	profile.Alias = ""
	if got := GetLoginProfileId(2, profile); got == "" {
		t.Fatalf("expected fallback id to be generated when no custom id or alias is present")
	}
}

func TestGetLoginCredentialsIncludesPortAndAdvancedOptions(t *testing.T) {
	profile := types.DatabaseCredentials{
		Type:      "Postgres",
		Hostname:  "db.local",
		Username:  "alice",
		Password:  "secret",
		Database:  "app",
		Port:      "5432",
		IsProfile: true,
		Advanced: map[string]string{
			"SSL Mode":         "verify-ca",
			"Application Name": "whodb-tests",
		},
	}

	credentials := GetLoginCredentials(profile)
	if credentials.Type != profile.Type || credentials.Hostname != profile.Hostname || credentials.Username != profile.Username {
		t.Fatalf("expected credentials fields to be copied from profile, got %#v", credentials)
	}
	if !credentials.IsProfile {
		t.Fatal("expected IsProfile to be preserved")
	}
	if len(credentials.Advanced) != 3 {
		t.Fatalf("expected port plus two advanced entries, got %#v", credentials.Advanced)
	}
	if credentials.Advanced[0].Key != "Port" || credentials.Advanced[0].Value != "5432" {
		t.Fatalf("expected port to be the first advanced credential, got %#v", credentials.Advanced[0])
	}

	advanced := map[string]string{}
	for _, record := range credentials.Advanced[1:] {
		advanced[record.Key] = record.Value
	}
	if advanced["SSL Mode"] != "verify-ca" {
		t.Fatalf("expected SSL Mode advanced option to be preserved, got %#v", advanced)
	}
	if advanced["Application Name"] != "whodb-tests" {
		t.Fatalf("expected Application Name advanced option to be preserved, got %#v", advanced)
	}
}

func TestGetLoginCredentialsUsesSourceDefaultPortWhenProfilePortMissing(t *testing.T) {
	profile := types.DatabaseCredentials{
		Type:     "Postgres",
		Hostname: "db.local",
		Username: "alice",
		Password: "secret",
		Database: "app",
	}

	credentials := GetLoginCredentials(profile)
	if len(credentials.Advanced) != 1 {
		t.Fatalf("expected one advanced port entry, got %#v", credentials.Advanced)
	}
	if credentials.Advanced[0].Key != "Port" || credentials.Advanced[0].Value != "5432" {
		t.Fatalf("expected source default port 5432, got %#v", credentials.Advanced[0])
	}
}
