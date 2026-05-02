package envconfig

import "testing"

func TestGetCloudProvidersFromEnvApplyDefaults(t *testing.T) {
	t.Setenv("WHODB_AWS_PROVIDER", `[{"name":"AWS One","region":"us-west-2","authMethod":"default"}]`)
	awsConfigs, err := GetAWSProvidersFromEnv()
	if err != nil {
		t.Fatalf("expected AWS provider env config to parse, got %v", err)
	}
	if len(awsConfigs) != 1 || awsConfigs[0].DiscoverRDS == nil || !*awsConfigs[0].DiscoverRDS || awsConfigs[0].DiscoverElastiCache == nil || !*awsConfigs[0].DiscoverElastiCache || awsConfigs[0].DiscoverDocumentDB == nil || !*awsConfigs[0].DiscoverDocumentDB {
		t.Fatalf("expected AWS provider discovery flags to default to true, got %#v", awsConfigs)
	}

	t.Setenv("WHODB_AZURE_PROVIDER", `[{"name":"Azure One","subscriptionId":"sub-123","authMethod":"default"}]`)
	azureConfigs, err := GetAzureProvidersFromEnv()
	if err != nil {
		t.Fatalf("expected Azure provider env config to parse, got %v", err)
	}
	if len(azureConfigs) != 1 || azureConfigs[0].DiscoverPostgreSQL == nil || !*azureConfigs[0].DiscoverPostgreSQL || azureConfigs[0].DiscoverMySQL == nil || !*azureConfigs[0].DiscoverMySQL || azureConfigs[0].DiscoverRedis == nil || !*azureConfigs[0].DiscoverRedis || azureConfigs[0].DiscoverCosmosDB == nil || !*azureConfigs[0].DiscoverCosmosDB {
		t.Fatalf("expected Azure provider discovery flags to default to true, got %#v", azureConfigs)
	}

	t.Setenv("WHODB_GCP_PROVIDER", `[{"name":"GCP One","projectId":"project-123","region":"us-central1"}]`)
	gcpConfigs, err := GetGCPProvidersFromEnv()
	if err != nil {
		t.Fatalf("expected GCP provider env config to parse, got %v", err)
	}
	if len(gcpConfigs) != 1 || gcpConfigs[0].DiscoverCloudSQL == nil || !*gcpConfigs[0].DiscoverCloudSQL || gcpConfigs[0].DiscoverAlloyDB == nil || !*gcpConfigs[0].DiscoverAlloyDB || gcpConfigs[0].DiscoverMemorystore == nil || !*gcpConfigs[0].DiscoverMemorystore {
		t.Fatalf("expected GCP provider discovery flags to default to true, got %#v", gcpConfigs)
	}
}

func TestGetCloudProvidersFromEnvRejectInvalidJSON(t *testing.T) {
	t.Setenv("WHODB_AWS_PROVIDER", `{`)
	if _, err := GetAWSProvidersFromEnv(); err == nil {
		t.Fatal("expected invalid AWS provider JSON to return an error")
	}

	t.Setenv("WHODB_AZURE_PROVIDER", `{`)
	if _, err := GetAzureProvidersFromEnv(); err == nil {
		t.Fatal("expected invalid Azure provider JSON to return an error")
	}

	t.Setenv("WHODB_GCP_PROVIDER", `{`)
	if _, err := GetGCPProvidersFromEnv(); err == nil {
		t.Fatal("expected invalid GCP provider JSON to return an error")
	}
}

func TestParseGenericProvidersDiscoversAndSkipsConfigs(t *testing.T) {
	t.Setenv("WHODB_AI_GENERIC_MISTRAL_NAME", "Mistral Hosted")
	t.Setenv("WHODB_AI_GENERIC_MISTRAL_BASE_URL", "https://mistral.example/v1")
	t.Setenv("WHODB_AI_GENERIC_MISTRAL_API_KEY", "mistral-key")
	t.Setenv("WHODB_AI_GENERIC_MISTRAL_MODELS", "small,medium")

	t.Setenv("WHODB_AI_GENERIC_ANTHROPIC_STYLE_NAME", "Anthropic Style")
	t.Setenv("WHODB_AI_GENERIC_ANTHROPIC_STYLE_TYPE", "anthropic")
	t.Setenv("WHODB_AI_GENERIC_ANTHROPIC_STYLE_BASE_URL", "https://anthropic.example")
	t.Setenv("WHODB_AI_GENERIC_ANTHROPIC_STYLE_MODELS", "claude-sonnet")

	t.Setenv("WHODB_AI_GENERIC_INCOMPLETE_BASE_URL", "https://broken.example")
	t.Setenv("WHODB_AI_GENERIC_EMPTY_MODELS_BASE_URL", "https://empty.example")
	t.Setenv("WHODB_AI_GENERIC_EMPTY_MODELS_MODELS", " , ")

	providers := ParseGenericProviders()
	if len(providers) != 2 {
		t.Fatalf("expected two valid generic providers, got %#v", providers)
	}

	if providers[0].ProviderId != "mistral" || providers[0].Name != "Mistral Hosted" || providers[0].ClientType != "openai-generic" {
		t.Fatalf("expected first provider defaults to be applied, got %#v", providers[0])
	}
	if len(providers[0].Models) != 2 || providers[0].Models[0] != "small" || providers[0].Models[1] != "medium" {
		t.Fatalf("expected first provider models to be parsed, got %#v", providers[0].Models)
	}

	if providers[1].ProviderId != "anthropic_style" || providers[1].ClientType != "anthropic" {
		t.Fatalf("expected second provider to preserve explicit type and normalized ID, got %#v", providers[1])
	}
}

func TestGetDefaultDatabaseCredentialsHandlesInvalidEnvPayloads(t *testing.T) {
	t.Setenv("WHODB_POSTGRES", `{`)
	if creds := GetDefaultDatabaseCredentials("postgres"); creds != nil {
		t.Fatalf("expected invalid JSON payload to return nil credentials, got %#v", creds)
	}

	t.Setenv("WHODB_MYSQL", "")
	t.Setenv("WHODB_MYSQL_1", `{"host":"mysql.local","user":"alice","database":"app"}`)
	t.Setenv("WHODB_MYSQL_2", `{`)

	creds := GetDefaultDatabaseCredentials("mysql")
	if len(creds) != 1 || creds[0].Hostname != "mysql.local" {
		t.Fatalf("expected numbered fallback parsing to stop after invalid payload, got %#v", creds)
	}
}
