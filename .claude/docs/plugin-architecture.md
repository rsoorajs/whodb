# Plugin Architecture Guide

The plugin architecture avoids hardcoded database type checks. All database-specific logic lives in plugins.

## Core Principle: No Switch Statements

```go
// WRONG - Don't do this:
func GetConstraints(dbType string, ...) {
    switch dbType {
    case "Postgres":
        // PostgreSQL logic
    case "MySQL":
        // MySQL logic
    }
}

// CORRECT - Add to PluginFunctions interface:
GetColumnConstraints(config *PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error)

// Then implement in each plugin:
func (p *PostgresPlugin) GetColumnConstraints(...) { /* PostgreSQL-specific */ }
func (p *MySQLPlugin) GetColumnConstraints(...) { /* MySQL-specific */ }
```

## Adding New Functionality

1. Add method to `PluginFunctions` interface in `core/src/engine/plugin.go`
2. Provide default implementation in base plugin (`GormPlugin` in `core/src/plugins/gorm/plugin.go`)
3. Override in specific plugins as needed
4. NoSQL plugins should return appropriate errors for SQL-specific features

## Request Context and Cancellation

Every request-scoped plugin operation must use the context carried by `*engine.PluginConfig`. Do not use `context.Background()` for query execution, metadata fetches, SDK calls, or health checks that are part of a user request.

- GORM-based SQL plugins should inherit cancellation and timeout behavior through `plugins.WithConnection()`, `connection_pool.go`, `connection_cache.go`, and `GormPlugin`
- Direct-driver plugins must use `config.OperationContext()` for request-scoped SDK calls
- Use `config.OperationContextWithTimeout(...)` when the plugin needs an explicit upper bound for a long-running operation
- Reserve `context.Background()` for non-request cleanup work such as cache eviction or best-effort disconnects
- If a plugin talks to a driver directly, add a small local helper so future methods inherit the same context behavior instead of repeating it by hand

```go
ctx, cancel := config.OperationContextWithTimeout(30 * time.Second)
defer cancel()

_, err := client.ListTables(ctx, input)
```

```go
func queryWithContext(session *gocql.Session, config *engine.PluginConfig, stmt string, values ...any) *gocql.Query {
    return session.Query(stmt, values...).WithContext(config.OperationContext())
}
```

## Plugin File Organization

SQL-based plugins follow this structure (see `core/src/plugins/postgres/` as reference):
- `db.go` - Connection creation (implements DB method)
- `postgres.go` (or `mysql.go`, etc.) - Plugin struct, NewXxxPlugin(), database-specific queries
- `types.go` - Type definitions, alias map, and GetDatabaseMetadata() implementation
- `constraints.go` - Column constraint detection (optional override)

GormPlugin base class (`core/src/plugins/gorm/`) provides:
- `plugin.go` - 40+ default method implementations
- `sqlbuilder.go` - SQL query building
- `errors.go` - ErrorHandler for user-friendly error messages
- `add.go`, `update.go`, `delete.go` - CRUD operations

## Adding a New Database

1. Create plugin directory in `core/src/plugins/`
2. Implement `PluginFunctions` interface (extend GormPlugin for SQL databases)
3. Add `init()` function calling `engine.RegisterPlugin(NewYourPlugin())` — the plugin self-registers when imported
4. Add a blank import in the entry point (`core/cmd/whodb/main.go`): `_ "github.com/clidey/whodb/core/src/plugins/yourplugin"`

## Key Methods to Override for SQL Plugins

```go
// Most SQL plugins override these:
GetAllSchemasQuery() string           // information_schema query for schemas
GetSchemaTableQuery() string          // Query for columns in a table
FormTableName(schema, table) string   // Default: "schema.table" (override for different behavior, e.g. SQLite ignores schema)
GetPlaceholder(index int) string      // $1 for Postgres, ? for MySQL
DB(config) (*gorm.DB, error)          // Connection with driver-specific config
GetDatabaseMetadata() *DatabaseMetadata // Operators, types, aliases for frontend
GetLastInsertID(db *gorm.DB) (int64, error) // Default: returns 0 (override for MySQL, Postgres, SQLite)
```

## Database Metadata (types.go)

Each SQL plugin must provide metadata for frontend UI via `GetDatabaseMetadata()`. This is the **single source of truth** for:
- Valid operators (=, >=, LIKE, etc.)
- Type definitions (VARCHAR, INTEGER, etc.) with UI hints (hasLength, hasPrecision)
- Alias maps (INT → INTEGER, BOOL → BOOLEAN)

The frontend fetches this via GraphQL `DatabaseMetadata` query on login. **No fallbacks** - if backend doesn't provide it, the UI will be broken.

### types.go Structure

```go
package postgres

import "github.com/clidey/whodb/core/src/engine"

// AliasMap maps type aliases to canonical names (UPPERCASE keys and values)
var AliasMap = map[string]string{
    "INT":  "INTEGER",
    "BOOL": "BOOLEAN",
}

// TypeDefinitions - canonical types shown in UI type selector
var TypeDefinitions = []engine.TypeDefinition{
    {ID: "INTEGER", Label: "integer", Category: engine.TypeCategoryNumeric},
    {ID: "VARCHAR", Label: "varchar", HasLength: true, DefaultLength: engine.IntPtr(255), Category: engine.TypeCategoryText},
    // ... more types
}

func (p *PostgresPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
    operators := make([]string, 0, len(supportedOperators))
    for op := range supportedOperators {
        operators = append(operators, op)
    }
    return &engine.DatabaseMetadata{
        DatabaseType:    engine.DatabaseType_Postgres,
        TypeDefinitions: TypeDefinitions,
        Operators:       operators,
        AliasMap:        AliasMap,
    }
}
```

### Type Validation

Column type validation uses `engine.ValidateColumnType()` which checks against TypeDefinitions. Types not in TypeDefinitions will be rejected when adding columns.

## Quirks to Know

- SQLite doesn't use schemas - `FormTableName()` returns just table name
- PostgreSQL array types display with underscore prefix (`_text`)
- MySQL `GetDatabases()` returns `ErrUnsupported`
- Redis iterates through 16 database slots to discover databases

- Plugin architecture ensures clean code separation
