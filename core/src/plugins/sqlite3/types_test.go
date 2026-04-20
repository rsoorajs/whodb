package sqlite3

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func TestNormalizeType(t *testing.T) {
	plugin := NewSqlite3Plugin().PluginFunctions.(*Sqlite3Plugin)

	if got := plugin.NormalizeType("int"); got != "INTEGER" {
		t.Fatalf("expected INTEGER, got %q", got)
	}

	if got := plugin.NormalizeType("varchar(100)"); got != "TEXT(100)" {
		t.Fatalf("expected TEXT(100), got %q", got)
	}
}

func TestSourceSessionMetadataIncludesAliasMap(t *testing.T) {
	meta, ok := sourcecatalog.ResolveSessionMetadata(string(engine.DatabaseType_Sqlite3))
	if !ok || meta == nil {
		t.Fatalf("expected metadata, got nil")
	}
	if meta.AliasMap["INT"] != "INTEGER" {
		t.Fatalf("expected INT alias to be INTEGER, got %q", meta.AliasMap["INT"])
	}
}
