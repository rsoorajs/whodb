//go:build !arm && !riscv64

/*
 * Copyright 2025 Clidey, Inc.
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

package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/src/bamlconfig"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/envconfig"
)

// SendAIChatStream starts a streaming AI chat and returns a channel of StreamChunks.
// Each chunk contains the accumulated text so far. The final chunk has IsFinal=true
// with the complete ChatMessage responses.
func (m *Manager) SendAIChatStream(ctx context.Context, providerID, modelType, token, schema, model, previousConversation, query string) (<-chan StreamChunk, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)
	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)
	config := engine.NewPluginConfig(credentials)

	// Resolve provider credentials
	if providerID != "" {
		providers := envconfig.GetConfiguredChatProviders()
		for _, provider := range providers {
			if provider.ProviderId == providerID {
				config.ExternalModel = &engine.ExternalModel{
					Type:  modelType,
					Token: provider.APIKey,
					Model: model,
				}
				break
			}
		}
	} else {
		config.ExternalModel = &engine.ExternalModel{
			Type:  modelType,
			Model: model,
		}
		if token != "" {
			config.ExternalModel.Token = token
		}
	}

	// Build table details (same as GormPlugin.Chat does)
	tableDetails, err := buildTableDetails(plugin, config, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}

	// Build BAML context
	dbContext := types.DatabaseContext{
		Database_type:         string(dbType),
		Schema:                schema,
		Tables_and_fields:     tableDetails,
		Previous_conversation: previousConversation,
	}

	// Setup BAML client
	callOpts := bamlconfig.SetupAIClient(config.ExternalModel)

	// Start BAML stream
	bamlStream, err := baml_client.Stream.GenerateSQLQuery(ctx, dbContext, query, callOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start stream: %w", err)
	}

	// Read from BAML stream and convert to StreamChunks
	out := make(chan StreamChunk, 1)
	go func() {
		defer close(out)

		var lastText string

		for chunk := range bamlStream {
			if chunk.IsError {
				out <- StreamChunk{Err: chunk.Error}
				return
			}

			if chunk.IsFinal {
				final := chunk.Final()
				if final == nil {
					out <- StreamChunk{IsFinal: true, Final: []*ChatMessage{}}
					return
				}
				messages := convertFinalResponses(*final, plugin, config)
				out <- StreamChunk{IsFinal: true, Final: messages}
				return
			}

			// Streaming chunk — accumulate text
			if stream := chunk.Stream(); stream != nil {
				for _, resp := range *stream {
					if resp.Text != nil {
						lastText = *resp.Text
					}
				}
				if lastText != "" {
					out <- StreamChunk{Text: lastText}
				}
			}
		}

		// Stream ended without an explicit IsFinal chunk — synthesize a final message
		// from whatever text we accumulated
		if lastText != "" {
			out <- StreamChunk{
				IsFinal: true,
				Final: []*ChatMessage{{
					Type: "message",
					Text: lastText,
				}},
			}
		} else {
			out <- StreamChunk{IsFinal: true, Final: []*ChatMessage{}}
		}
	}()

	return out, nil
}

// buildTableDetails fetches table and column info for the schema.
func buildTableDetails(plugin *engine.Plugin, config *engine.PluginConfig, schema string) (string, error) {
	units, err := plugin.GetStorageUnits(config, schema)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, unit := range units {
		b.WriteString(fmt.Sprintf("table: %s\n", unit.Name))
		columns, err := plugin.GetColumnsForTable(config, schema, unit.Name)
		if err != nil {
			continue
		}
		for _, col := range columns {
			b.WriteString(fmt.Sprintf("- %s (%s)\n", col.Name, col.Type))
		}
	}
	return b.String(), nil
}

// convertFinalResponses converts BAML final responses to ChatMessages,
// executing SELECT queries and marking mutations for confirmation.
func convertFinalResponses(responses []types.ChatResponse, plugin *engine.Plugin, config *engine.PluginConfig) []*ChatMessage {
	var messages []*ChatMessage
	for _, resp := range responses {
		msg := &ChatMessage{
			Type: convertBAMLType(resp.Type),
			Text: resp.Text,
		}

		if resp.Type == types.ChatMessageTypeSQL && resp.Operation != nil {
			// Strip trailing semicolons — some DB drivers reject them
			queryText := strings.TrimRight(strings.TrimSpace(resp.Text), ";")

			isMutation := *resp.Operation == types.OperationTypeINSERT ||
				*resp.Operation == types.OperationTypeUPDATE ||
				*resp.Operation == types.OperationTypeDELETE ||
				*resp.Operation == types.OperationTypeCREATE ||
				*resp.Operation == types.OperationTypeALTER ||
				*resp.Operation == types.OperationTypeDROP

			if isMutation {
				msg.Type = convertOperationType(*resp.Operation)
				msg.RequiresConfirmation = true
			} else {
				result, err := plugin.RawExecute(config, queryText)
				if err != nil {
					msg.Type = "error"
					msg.Text = err.Error()
				} else {
					msg.Type = convertOperationType(*resp.Operation)
					msg.Result = result
				}
			}
		}

		messages = append(messages, msg)
	}
	return messages
}

func convertBAMLType(t types.ChatMessageType) string {
	switch t {
	case types.ChatMessageTypeSQL:
		return "sql"
	case types.ChatMessageTypeMESSAGE:
		return "message"
	case types.ChatMessageTypeERROR:
		return "error"
	default:
		return "message"
	}
}

func convertOperationType(op types.OperationType) string {
	switch op {
	case types.OperationTypeGET:
		return "sql:get"
	case types.OperationTypeINSERT:
		return "sql:insert"
	case types.OperationTypeUPDATE:
		return "sql:update"
	case types.OperationTypeDELETE:
		return "sql:delete"
	case types.OperationTypeCREATE:
		return "sql:create"
	case types.OperationTypeALTER:
		return "sql:alter"
	case types.OperationTypeDROP:
		return "sql:drop"
	case types.OperationTypeTEXT:
		return "text"
	default:
		return "sql"
	}
}
