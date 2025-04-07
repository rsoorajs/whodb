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

package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.68

import (
	"context"
	"errors"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/llm"
	"github.com/clidey/whodb/core/src/settings"
)

// Login is the resolver for the Login field.
func (r *mutationResolver) Login(ctx context.Context, credentials model.LoginCredentials) (*model.StatusResponse, error) {
	advanced := []engine.Record{}
	for _, recordInput := range credentials.Advanced {
		advanced = append(advanced, engine.Record{
			Key:   recordInput.Key,
			Value: recordInput.Value,
		})
	}
	if !src.MainEngine.Choose(engine.DatabaseType(credentials.Type)).IsAvailable(&engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     credentials.Type,
			Hostname: credentials.Hostname,
			Username: credentials.Username,
			Password: credentials.Password,
			Database: credentials.Database,
			Advanced: advanced,
		},
	}) {
		return nil, errors.New("unauthorized")
	}
	return auth.Login(ctx, &credentials)
}

// LoginWithProfile is the resolver for the LoginWithProfile field.
func (r *mutationResolver) LoginWithProfile(ctx context.Context, profile model.LoginProfileInput) (*model.StatusResponse, error) {
	profiles := src.GetLoginProfiles()
	for i, loginProfile := range profiles {
		profileId := src.GetLoginProfileId(i, loginProfile)
		if profile.ID == profileId {
			if !src.MainEngine.Choose(engine.DatabaseType(loginProfile.Type)).IsAvailable(&engine.PluginConfig{
				Credentials: src.GetLoginCredentials(loginProfile),
			}) {
				return nil, errors.New("unauthorized")
			}
			credentials := &model.LoginCredentials{
				ID: &profile.ID,
			}
			if profile.Database != nil {
				credentials.Database = *profile.Database
			}
			return auth.Login(ctx, credentials)
		}
	}
	return nil, errors.New("login profile does not exist or is not authorized")
}

// Logout is the resolver for the Logout field.
func (r *mutationResolver) Logout(ctx context.Context) (*model.StatusResponse, error) {
	return auth.Logout(ctx)
}

// UpdateSettings is the resolver for the UpdateSettings field.
func (r *mutationResolver) UpdateSettings(ctx context.Context, newSettings model.SettingsConfigInput) (*model.StatusResponse, error) {
	var fields []settings.ISettingsField

	if newSettings.MetricsEnabled != nil {
		fields = append(fields, settings.MetricsEnabledField(common.StrPtrToBool(newSettings.MetricsEnabled)))
	}

	updated := settings.UpdateSettings(fields...)
	return &model.StatusResponse{
		Status: updated,
	}, nil
}

// AddStorageUnit is the resolver for the AddStorageUnit field.
func (r *mutationResolver) AddStorageUnit(ctx context.Context, schema string, storageUnit string, fields []*model.RecordInput) (*model.StatusResponse, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	fieldsMap := map[string]string{}
	for _, field := range fields {
		fieldsMap[field.Key] = field.Value
	}
	status, err := src.MainEngine.Choose(engine.DatabaseType(typeArg)).AddStorageUnit(config, schema, storageUnit, fieldsMap)
	if err != nil {
		return nil, err
	}
	return &model.StatusResponse{
		Status: status,
	}, nil
}

// UpdateStorageUnit is the resolver for the UpdateStorageUnit field.
func (r *mutationResolver) UpdateStorageUnit(ctx context.Context, schema string, storageUnit string, values []*model.RecordInput, updatedColumns []string) (*model.StatusResponse, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	valuesMap := map[string]string{}
	for _, value := range values {
		valuesMap[value.Key] = value.Value
	}
	status, err := src.MainEngine.Choose(engine.DatabaseType(typeArg)).UpdateStorageUnit(config, schema, storageUnit, valuesMap, updatedColumns)
	if err != nil {
		return nil, err
	}
	return &model.StatusResponse{
		Status: status,
	}, nil
}

// AddRow is the resolver for the AddRow field.
func (r *mutationResolver) AddRow(ctx context.Context, schema string, storageUnit string, values []*model.RecordInput) (*model.StatusResponse, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	valuesRecords := []engine.Record{}
	for _, field := range values {
		extraFields := map[string]string{}
		for _, extraField := range field.Extra {
			extraFields[extraField.Key] = extraField.Value
		}
		valuesRecords = append(valuesRecords, engine.Record{
			Key:   field.Key,
			Value: field.Value,
			Extra: extraFields,
		})
	}
	status, err := src.MainEngine.Choose(engine.DatabaseType(typeArg)).AddRow(config, schema, storageUnit, valuesRecords)

	if err != nil {
		return nil, err
	}
	return &model.StatusResponse{
		Status: status,
	}, nil
}

// DeleteRow is the resolver for the DeleteRow field.
func (r *mutationResolver) DeleteRow(ctx context.Context, schema string, storageUnit string, values []*model.RecordInput) (*model.StatusResponse, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	valuesMap := map[string]string{}
	for _, value := range values {
		valuesMap[value.Key] = value.Value
	}
	status, err := src.MainEngine.Choose(engine.DatabaseType(typeArg)).DeleteRow(config, schema, storageUnit, valuesMap)
	if err != nil {
		return nil, err
	}
	return &model.StatusResponse{
		Status: status,
	}, nil
}

// Version is the resolver for the Version field.
func (r *queryResolver) Version(ctx context.Context) (string, error) {
	return env.GetClideyQuickContainerImage(), nil
}

// Profiles is the resolver for the Profiles field.
func (r *queryResolver) Profiles(ctx context.Context) ([]*model.LoginProfile, error) {
	profiles := []*model.LoginProfile{}
	for i, profile := range src.GetLoginProfiles() {
		profileName := src.GetLoginProfileId(i, profile)
		loginProfile := &model.LoginProfile{
			ID:       profileName,
			Type:     model.DatabaseType(profile.Type),
			Database: &profile.Database,
		}

		if len(profile.Alias) > 0 {
			loginProfile.Alias = &profile.Alias
		}
		profiles = append(profiles, loginProfile)
	}
	return profiles, nil
}

// Database is the resolver for the Database field.
func (r *queryResolver) Database(ctx context.Context, typeArg string) ([]string, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	return src.MainEngine.Choose(engine.DatabaseType(typeArg)).GetDatabases(config)
}

// Schema is the resolver for the Schema field.
func (r *queryResolver) Schema(ctx context.Context) ([]string, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	return src.MainEngine.Choose(engine.DatabaseType(typeArg)).GetAllSchemas(config)
}

// StorageUnit is the resolver for the StorageUnit field.
func (r *queryResolver) StorageUnit(ctx context.Context, schema string) ([]*model.StorageUnit, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	units, err := src.MainEngine.Choose(engine.DatabaseType(typeArg)).GetStorageUnits(config, schema)
	if err != nil {
		return nil, err
	}
	storageUnits := []*model.StorageUnit{}
	for _, unit := range units {
		storageUnits = append(storageUnits, engine.GetStorageUnitModel(unit))
	}
	return storageUnits, nil
}

// Row is the resolver for the Row field.
func (r *queryResolver) Row(ctx context.Context, schema string, storageUnit string, where *model.WhereCondition, pageSize int, pageOffset int) (*model.RowsResult, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	rowsResult, err := src.MainEngine.Choose(engine.DatabaseType(typeArg)).GetRows(config, schema, storageUnit, where, pageSize, pageOffset)
	if err != nil {
		return nil, err
	}
	columns := []*model.Column{}
	for _, column := range rowsResult.Columns {
		columns = append(columns, &model.Column{
			Type: column.Type,
			Name: column.Name,
		})
	}
	return &model.RowsResult{
		Columns:       columns,
		Rows:          rowsResult.Rows,
		DisableUpdate: rowsResult.DisableUpdate,
	}, nil
}

// RawExecute is the resolver for the RawExecute field.
func (r *queryResolver) RawExecute(ctx context.Context, query string) (*model.RowsResult, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	rowsResult, err := src.MainEngine.Choose(engine.DatabaseType(typeArg)).RawExecute(config, query)
	if err != nil {
		return nil, err
	}
	columns := []*model.Column{}
	for _, column := range rowsResult.Columns {
		columns = append(columns, &model.Column{
			Type: column.Type,
			Name: column.Name,
		})
	}
	return &model.RowsResult{
		Columns: columns,
		Rows:    rowsResult.Rows,
	}, nil
}

// Graph is the resolver for the Graph field.
func (r *queryResolver) Graph(ctx context.Context, schema string) ([]*model.GraphUnit, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	graphUnits, err := src.MainEngine.Choose(engine.DatabaseType(typeArg)).GetGraph(config, schema)
	if err != nil {
		return nil, err
	}
	graphUnitsModel := []*model.GraphUnit{}
	for _, graphUnit := range graphUnits {
		relations := []*model.GraphUnitRelationship{}
		for _, relation := range graphUnit.Relations {
			relations = append(relations, &model.GraphUnitRelationship{
				Name:         relation.Name,
				Relationship: model.GraphUnitRelationshipType(relation.RelationshipType),
			})
		}
		graphUnitsModel = append(graphUnitsModel, &model.GraphUnit{
			Unit:      engine.GetStorageUnitModel(graphUnit.Unit),
			Relations: relations,
		})
	}
	return graphUnitsModel, nil
}

// AIModel is the resolver for the AIModel field.
func (r *queryResolver) AIModel(ctx context.Context, modelType string, token *string) ([]string, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	config.ExternalModel = &engine.ExternalModel{
		Type: modelType,
	}
	if token != nil {
		config.ExternalModel.Token = *token
	}
	models, err := llm.Instance(config).GetSupportedModels()
	if err != nil {
		return nil, err
	}
	return models, nil
}

// AIChat is the resolver for the AIChat field.
func (r *queryResolver) AIChat(ctx context.Context, modelType string, token *string, schema string, input model.ChatInput) ([]*model.AIChatMessage, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	typeArg := config.Credentials.Type
	config.ExternalModel = &engine.ExternalModel{
		Type: modelType,
	}
	if token != nil {
		config.ExternalModel.Token = *token
	}
	messages, err := src.MainEngine.Choose(engine.DatabaseType(typeArg)).Chat(config, schema, input.Model, input.PreviousConversation, input.Query)

	if err != nil {
		return nil, err
	}

	chatResponse := []*model.AIChatMessage{}

	for _, message := range messages {
		var result *model.RowsResult
		if strings.HasPrefix(message.Type, "sql") {
			columns := []*model.Column{}
			for _, column := range message.Result.Columns {
				columns = append(columns, &model.Column{
					Type: column.Type,
					Name: column.Name,
				})
			}
			result = &model.RowsResult{
				Columns: columns,
				Rows:    message.Result.Rows,
			}
		}
		chatResponse = append(chatResponse, &model.AIChatMessage{
			Type:   message.Type,
			Result: result,
			Text:   message.Text,
		})
	}

	return chatResponse, nil
}

// SettingsConfig is the resolver for the SettingsConfig field.
func (r *queryResolver) SettingsConfig(ctx context.Context) (*model.SettingsConfig, error) {
	currentSettings := settings.Get()
	return &model.SettingsConfig{MetricsEnabled: &currentSettings.MetricsEnabled}, nil
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
