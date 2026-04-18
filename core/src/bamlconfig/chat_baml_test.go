//go:build !arm && !riscv64

package bamlconfig

import (
	"errors"
	"testing"

	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/src/engine"
)

type rawExecuteStub struct {
	queries []string
	result  *engine.GetRowsResult
	err     error
}

func (s *rawExecuteStub) RawExecute(_ *engine.PluginConfig, query string, _ ...any) (*engine.GetRowsResult, error) {
	s.queries = append(s.queries, query)
	return s.result, s.err
}

func TestProcessBAMLResponse(t *testing.T) {
	t.Run("non sql messages pass through unchanged", func(t *testing.T) {
		resp := &types.ChatResponse{
			Type: types.ChatMessageTypeMESSAGE,
			Text: "hello",
		}

		message := ProcessBAMLResponse(resp, &engine.PluginConfig{}, &rawExecuteStub{})

		if message.Type != "message" {
			t.Fatalf("expected message type, got %q", message.Type)
		}
		if message.Text != "hello" {
			t.Fatalf("expected text to be preserved, got %q", message.Text)
		}
		if message.RequiresConfirmation {
			t.Fatal("did not expect non-SQL message to require confirmation")
		}
	})

	t.Run("mutations require confirmation without executing sql", func(t *testing.T) {
		op := types.OperationTypeINSERT
		stub := &rawExecuteStub{}
		resp := &types.ChatResponse{
			Type:      types.ChatMessageTypeSQL,
			Operation: &op,
			Text:      "INSERT INTO users VALUES (1)",
		}

		message := ProcessBAMLResponse(resp, &engine.PluginConfig{}, stub)

		if message.Type != "sql:insert" {
			t.Fatalf("expected sql:insert type, got %q", message.Type)
		}
		if !message.RequiresConfirmation {
			t.Fatal("expected mutation to require confirmation")
		}
		if len(stub.queries) != 0 {
			t.Fatalf("expected mutation to skip execution, got %#v", stub.queries)
		}
	})

	t.Run("read queries execute and attach results", func(t *testing.T) {
		op := types.OperationTypeGET
		expected := &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "id", Type: "int"}},
			Rows:    [][]string{{"1"}},
		}
		stub := &rawExecuteStub{result: expected}
		resp := &types.ChatResponse{
			Type:      types.ChatMessageTypeSQL,
			Operation: &op,
			Text:      "SELECT id FROM users",
		}

		message := ProcessBAMLResponse(resp, &engine.PluginConfig{}, stub)

		if message.Type != "sql:get" {
			t.Fatalf("expected sql:get type, got %q", message.Type)
		}
		if message.Result != expected {
			t.Fatal("expected query result to be attached to message")
		}
		if len(stub.queries) != 1 || stub.queries[0] != "SELECT id FROM users" {
			t.Fatalf("expected query to be executed once, got %#v", stub.queries)
		}
	})

	t.Run("read query errors become error messages", func(t *testing.T) {
		op := types.OperationTypeGET
		stub := &rawExecuteStub{err: errors.New("query failed")}
		resp := &types.ChatResponse{
			Type:      types.ChatMessageTypeSQL,
			Operation: &op,
			Text:      "SELECT id FROM users",
		}

		message := ProcessBAMLResponse(resp, &engine.PluginConfig{}, stub)

		if message.Type != "error" {
			t.Fatalf("expected error type, got %q", message.Type)
		}
		if message.Text != "query failed" {
			t.Fatalf("expected execution error text, got %q", message.Text)
		}
	})
}

func TestConvertBAMLTypeToWhoDB(t *testing.T) {
	testCases := map[types.ChatMessageType]string{
		types.ChatMessageTypeSQL:     "sql",
		types.ChatMessageTypeMESSAGE: "message",
		types.ChatMessageTypeERROR:   "error",
		types.ChatMessageType("X"):   "message",
	}

	for input, expected := range testCases {
		if got := ConvertBAMLTypeToWhoDB(input); got != expected {
			t.Fatalf("ConvertBAMLTypeToWhoDB(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestOperationHelpers(t *testing.T) {
	if got := OperationToString(types.OperationTypeUPDATE); got != "update" {
		t.Fatalf("expected update, got %q", got)
	}
	if got := OperationToString(types.OperationType("CUSTOM")); got != "CUSTOM" {
		t.Fatalf("expected unknown operation to pass through, got %q", got)
	}
	if got := ConvertOperationType(types.OperationTypeDELETE); got != "sql:delete" {
		t.Fatalf("expected prefixed delete op, got %q", got)
	}
}

func TestGetBAMLProviderAndOptions(t *testing.T) {
	originalResolver := bamlConfigResolver
	bamlConfigResolver = nil
	t.Cleanup(func() {
		bamlConfigResolver = originalResolver
	})

	model := &engine.ExternalModel{
		Type:     "Custom",
		Token:    "secret",
		Model:    "test-model",
		Endpoint: "https://gateway.example.com/v1",
	}

	RegisterBAMLConfigResolver(func(providerType, apiKey, endpoint, modelName string) (string, map[string]any, error) {
		if providerType != "Custom" || apiKey != "secret" || endpoint != "https://gateway.example.com/v1" || modelName != "test-model" {
			t.Fatalf("resolver received unexpected inputs: %q %q %q %q", providerType, apiKey, endpoint, modelName)
		}
		return "anthropic", map[string]any{"model": modelName, "api_key": apiKey}, nil
	})

	provider, opts := getBAMLProviderAndOptions(model)
	if provider != "anthropic" {
		t.Fatalf("expected resolver provider, got %q", provider)
	}
	if opts["model"] != "test-model" || opts["api_key"] != "secret" {
		t.Fatalf("expected resolver options to be returned, got %#v", opts)
	}

	RegisterBAMLConfigResolver(func(_, _, _, _ string) (string, map[string]any, error) {
		return "", nil, errors.New("boom")
	})

	provider, opts = getBAMLProviderAndOptions(model)
	if provider != "openai-generic" {
		t.Fatalf("expected fallback provider, got %q", provider)
	}
	if opts["model"] != "test-model" || opts["api_key"] != "secret" || opts["base_url"] != "https://gateway.example.com/v1" {
		t.Fatalf("expected fallback options to preserve model config, got %#v", opts)
	}
}

func TestSetupAIClientAndCreateDynamicBAMLClient(t *testing.T) {
	originalResolver := bamlConfigResolver
	bamlConfigResolver = nil
	t.Cleanup(func() {
		bamlConfigResolver = originalResolver
	})

	if got := SetupAIClient(nil); len(got) != 0 {
		t.Fatalf("expected no call options for nil model, got %d", len(got))
	}

	if got := SetupAIClient(&engine.ExternalModel{Type: "OpenAI"}); len(got) != 0 {
		t.Fatalf("expected no call options for model without model id, got %d", len(got))
	}

	RegisterBAMLConfigResolver(func(providerType, apiKey, endpoint, modelName string) (string, map[string]any, error) {
		return "openai-generic", map[string]any{
			"model":    modelName,
			"api_key":  apiKey,
			"base_url": endpoint,
		}, nil
	})

	model := &engine.ExternalModel{
		Type:     "OpenAI",
		Token:    "secret",
		Model:    "gpt-test",
		Endpoint: "https://gateway.example.com/v1",
	}

	if registry := CreateDynamicBAMLClient(model); registry == nil {
		t.Fatal("expected dynamic BAML client registry to be created")
	}

	if got := SetupAIClient(model); len(got) != 1 {
		t.Fatalf("expected one call option when model is configured, got %d", len(got))
	}
}
