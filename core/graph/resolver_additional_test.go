package graph

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/dbcatalog"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/types"
)

func TestQueryProfilesMapsAliasCustomIDAndSSL(t *testing.T) {
	originalEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	t.Cleanup(func() {
		src.MainEngine = originalEngine
	})

	src.MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Type:     "Postgres",
		Alias:    "Reporting",
		Hostname: "db.internal",
		Database: "analytics",
		Source:   "env",
		Advanced: map[string]string{
			ssl.KeySSLMode: string(ssl.SSLModeRequired),
		},
	})
	src.MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Type:     "MySQL",
		CustomId: "mysql-profile",
		Hostname: "mysql.internal",
		Database: "app",
		Source:   "env",
	})

	profiles, err := (&Resolver{}).Query().Profiles(context.Background())
	if err != nil {
		t.Fatalf("expected profiles query to succeed, got %v", err)
	}

	var reportingProfile *model.LoginProfile
	var mysqlProfile *model.LoginProfile
	for _, profile := range profiles {
		if profile.Alias != nil && *profile.Alias == "Reporting" {
			reportingProfile = profile
		}
		if profile.ID == "mysql-profile" {
			mysqlProfile = profile
		}
	}

	if reportingProfile == nil {
		t.Fatalf("expected reporting profile to be present, got %#v", profiles)
	}
	if !reportingProfile.SSLConfigured {
		t.Fatal("expected SSLConfigured to be true when SSL mode is enabled")
	}
	if mysqlProfile == nil {
		t.Fatalf("expected custom ID profile to be present, got %#v", profiles)
	}
}

func TestQueryDatabaseUsesMinimalConfigWithoutSessionCredentials(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetDatabasesFunc = func(config *engine.PluginConfig) ([]string, error) {
		if config == nil || config.Credentials == nil {
			t.Fatal("expected credentials to be present")
		}
		if config.Credentials.Type != "Test" {
			t.Fatalf("expected type Test, got %q", config.Credentials.Type)
		}
		if config.Credentials.Hostname != "" {
			t.Fatalf("expected no session credentials to be used, got hostname %q", config.Credentials.Hostname)
		}
		return []string{"db_a", "db_b"}, nil
	}
	setEngineMock(t, mock)

	databases, err := (&Resolver{}).Query().Database(context.Background(), "Test")
	if err != nil {
		t.Fatalf("expected database query to succeed, got %v", err)
	}
	if len(databases) != 2 || databases[0] != "db_a" || databases[1] != "db_b" {
		t.Fatalf("unexpected databases result: %#v", databases)
	}
}

func TestQueryRowValidatesPaginationAndEnrichesColumns(t *testing.T) {
	t.Run("rejects invalid page size before hitting plugins", func(t *testing.T) {
		originalMaxPageSize := env.MaxPageSize
		env.MaxPageSize = 10
		t.Cleanup(func() {
			env.MaxPageSize = originalMaxPageSize
		})

		_, err := (&Resolver{}).Query().Row(context.Background(), "public", "orders", nil, nil, 11, 0)
		if err == nil || !strings.Contains(err.Error(), "pageSize must not exceed 10") {
			t.Fatalf("expected max page size validation error, got %v", err)
		}
	})

	t.Run("maps row and column metadata", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
		mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
		mock.GetRowsFunc = func(*engine.PluginConfig, *engine.GetRowsRequest) (*engine.GetRowsResult, error) {
			return &engine.GetRowsResult{
				Columns: []engine.Column{
					{Name: "id", Type: "INTEGER"},
					{Name: "user_id", Type: "INTEGER"},
				},
				Rows:       [][]string{{"1", "42"}},
				TotalCount: 17,
			}, nil
		}
		refTable := "users"
		refColumn := "id"
		mock.GetColumnsForTableFunc = func(*engine.PluginConfig, string, string) ([]engine.Column, error) {
			return []engine.Column{
				{Name: "id", Type: "INTEGER", IsPrimary: true},
				{Name: "user_id", Type: "INTEGER", IsForeignKey: true, ReferencedTable: &refTable, ReferencedColumn: &refColumn},
			}, nil
		}
		setEngineMock(t, mock)

		ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
		result, err := (&Resolver{}).Query().Row(ctx, "public", "orders", nil, nil, 5, 0)
		if err != nil {
			t.Fatalf("expected row query to succeed, got %v", err)
		}
		if result.TotalCount != 17 || len(result.Rows) != 1 {
			t.Fatalf("unexpected row result: %#v", result)
		}
		if len(result.Columns) != 2 || !result.Columns[0].IsPrimary {
			t.Fatalf("expected primary key metadata to be attached, got %#v", result.Columns)
		}
		if !result.Columns[1].IsForeignKey || result.Columns[1].ReferencedTable == nil || *result.Columns[1].ReferencedTable != "users" {
			t.Fatalf("expected foreign key metadata to be attached, got %#v", result.Columns[1])
		}
	})
}

func TestQueryColumnsBatchSkipsFailedTables(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.GetColumnsForTableFunc = func(_ *engine.PluginConfig, _ string, storageUnit string) ([]engine.Column, error) {
		if storageUnit == "broken" {
			return nil, errors.New("boom")
		}
		return []engine.Column{{Name: "id", Type: "INTEGER"}}, nil
	}
	setEngineMock(t, mock)

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	result, err := (&Resolver{}).Query().ColumnsBatch(ctx, "public", []string{"users", "broken"})
	if err != nil {
		t.Fatalf("expected columns batch to succeed, got %v", err)
	}
	if len(result) != 1 || result[0].StorageUnit != "users" {
		t.Fatalf("expected only successful tables to be returned, got %#v", result)
	}
}

func TestQueryConnectableDatabasesMapsCatalogEntries(t *testing.T) {
	result, err := (&Resolver{}).Query().ConnectableDatabases(context.Background())
	if err != nil {
		t.Fatalf("expected connectable databases query to succeed, got %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected connectable databases to be returned")
	}

	var postgres *model.ConnectableDatabase
	for _, entry := range result {
		if entry.ID == string(engine.DatabaseType_Postgres) {
			postgres = entry
			break
		}
	}
	if postgres == nil {
		t.Fatalf("expected %q entry to be present in connectable databases", engine.DatabaseType_Postgres)
	}
	if postgres.Fields == nil || !postgres.Fields.Hostname || !postgres.RequiredFields.Database {
		t.Fatalf("expected postgres field metadata to be mapped, got %#v / %#v", postgres.Fields, postgres.RequiredFields)
	}
	if len(postgres.SslModes) == 0 {
		t.Fatal("expected postgres SSL modes to be exposed")
	}
	portFound := false
	for _, record := range postgres.Extra {
		if record.Key == "Port" && record.Value == "5432" {
			portFound = true
			break
		}
	}
	if !portFound {
		t.Fatalf("expected postgres default port to be included, got %#v", postgres.Extra)
	}
}

func TestQueryDatabaseQuerySuggestionsCapsAtThree(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetStorageUnitsFunc = func(*engine.PluginConfig, string) ([]engine.StorageUnit, error) {
		return []engine.StorageUnit{
			{Name: "users"},
			{Name: "orders"},
			{Name: "payments"},
			{Name: "ignored"},
		}, nil
	}
	setEngineMock(t, mock)

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	suggestions, err := (&Resolver{}).Query().DatabaseQuerySuggestions(ctx, "public")
	if err != nil {
		t.Fatalf("expected suggestions query to succeed, got %v", err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("expected suggestions to be capped at 3, got %#v", suggestions)
	}
	if !strings.Contains(suggestions[0].Description, "users") || suggestions[1].Category != "AGGREGATE" {
		t.Fatalf("expected deterministic suggestion text/categories, got %#v", suggestions)
	}
	for _, suggestion := range suggestions {
		if strings.Contains(suggestion.Description, "ignored") {
			t.Fatalf("did not expect truncated table to appear in suggestions, got %#v", suggestions)
		}
	}
}

func TestQueryHealthReportsDatabaseStatus(t *testing.T) {
	t.Run("healthy plugin reports healthy database", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
		mock.IsAvailableFunc = func(context.Context, *engine.PluginConfig) bool { return true }
		setEngineMock(t, mock)

		ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
		status, err := (&Resolver{}).Query().Health(ctx)
		if err != nil {
			t.Fatalf("expected health query to succeed, got %v", err)
		}
		if status.Server != "healthy" || status.Database != "healthy" {
			t.Fatalf("expected healthy server/database, got %#v", status)
		}
	})

	t.Run("failed availability reports database error", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
		mock.IsAvailableFunc = func(context.Context, *engine.PluginConfig) bool { return false }
		setEngineMock(t, mock)

		ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
		status, err := (&Resolver{}).Query().Health(ctx)
		if err != nil {
			t.Fatalf("expected health query to succeed, got %v", err)
		}
		if status.Database != "error" {
			t.Fatalf("expected database error status, got %#v", status)
		}
	})
}

func TestMutationExecuteConfirmedSQLMapsResultsAndErrors(t *testing.T) {
	t.Run("successful execution returns query result", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
		mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
			return &engine.GetRowsResult{
				Columns: []engine.Column{{Name: "id", Type: "INTEGER"}},
				Rows:    [][]string{{"1"}},
			}, nil
		}
		setEngineMock(t, mock)

		ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
		message, err := (&Resolver{}).Mutation().ExecuteConfirmedSQL(ctx, "SELECT 1", "sql:get")
		if err != nil {
			t.Fatalf("expected confirmed SQL execution to succeed, got %v", err)
		}
		if message.Type != "sql:get" || message.Result == nil || len(message.Result.Rows) != 1 {
			t.Fatalf("unexpected confirmed SQL message: %#v", message)
		}
	})

	t.Run("execution errors become error messages", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
		mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
			return nil, errors.New("query failed")
		}
		setEngineMock(t, mock)

		ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
		message, err := (&Resolver{}).Mutation().ExecuteConfirmedSQL(ctx, "DELETE FROM orders", "sql:delete")
		if err != nil {
			t.Fatalf("expected confirmed SQL execution to return a mapped message, got %v", err)
		}
		if message.Type != "error" || message.Text != "query failed" {
			t.Fatalf("expected error message, got %#v", message)
		}
	})
}

func TestMutationImportSQLValidatesSourcesAndExecutesScripts(t *testing.T) {
	mutation := (&Resolver{}).Mutation()
	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})

	t.Run("rejects missing or conflicting SQL sources", func(t *testing.T) {
		setEngineMock(t, testutil.NewPluginMock(engine.DatabaseType("Test")))

		result, err := mutation.ImportSQL(ctx, model.ImportSQLInput{})
		if err != nil {
			t.Fatalf("expected validation error to be returned as result, got %v", err)
		}
		if result.Status || result.Detail == nil || *result.Detail != importErrorSQLSourceMissing {
			t.Fatalf("expected missing source validation result, got %#v", result)
		}

		script := "SELECT 1"
		upload := graphql.Upload{Filename: "query.sql"}
		result, err = mutation.ImportSQL(ctx, model.ImportSQLInput{
			Script: &script,
			File:   &upload,
		})
		if err != nil {
			t.Fatalf("expected conflicting source validation result, got %v", err)
		}
		if result.Status || result.Detail == nil || *result.Detail != importErrorSQLSourceBoth {
			t.Fatalf("expected conflicting source validation result, got %#v", result)
		}
	})

	t.Run("executes script with multistatement enabled", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
		mock.RawExecuteFunc = func(config *engine.PluginConfig, query string, _ ...any) (*engine.GetRowsResult, error) {
			if !config.MultiStatement {
				t.Fatal("expected SQL import to enable multistatement mode")
			}
			if query != "CREATE TABLE demo(id INT);" {
				t.Fatalf("unexpected SQL script: %q", query)
			}
			return &engine.GetRowsResult{}, nil
		}
		setEngineMock(t, mock)

		script := "CREATE TABLE demo(id INT);"
		result, err := mutation.ImportSQL(ctx, model.ImportSQLInput{Script: &script})
		if err != nil {
			t.Fatalf("expected SQL import to succeed, got %v", err)
		}
		if !result.Status || result.Detail != nil {
			t.Fatalf("expected successful import result, got %#v", result)
		}
	})

	t.Run("maps unsupported multistatement errors to validation keys", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
		mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
			return nil, engine.ErrMultiStatementUnsupported
		}
		setEngineMock(t, mock)

		script := "DROP TABLE demo;"
		result, err := mutation.ImportSQL(ctx, model.ImportSQLInput{Script: &script})
		if err != nil {
			t.Fatalf("expected unsupported error to be returned as result, got %v", err)
		}
		if result.Status || result.Detail == nil || *result.Detail != importErrorSQLMultiStatementUnsupported {
			t.Fatalf("expected unsupported multistatement result, got %#v", result)
		}
	})
}

func TestCatalogHasStableDefaultPortForPostgres(t *testing.T) {
	port, ok := dbcatalog.DefaultPort(string(engine.DatabaseType_Postgres))
	if !ok || port != 5432 {
		t.Fatalf("expected postgres default port 5432, got %d (ok=%t)", port, ok)
	}
}
