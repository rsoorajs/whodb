package postgres

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func TestNormalizeType(t *testing.T) {
	if got := NormalizeType("int"); got != "INTEGER" {
		t.Fatalf("expected INTEGER, got %q", got)
	}

	if got := NormalizeType("varchar(25)"); got != "CHARACTER VARYING(25)" {
		t.Fatalf("expected CHARACTER VARYING(25), got %q", got)
	}
}

func TestSourceSessionMetadataIncludesAliasMap(t *testing.T) {
	meta, ok := sourcecatalog.ResolveSessionMetadata(string(engine.DatabaseType_Postgres))
	if !ok || meta == nil {
		t.Fatalf("expected metadata, got nil")
	}
	if meta.AliasMap["INT"] != "INTEGER" {
		t.Fatalf("expected INT alias to be INTEGER, got %q", meta.AliasMap["INT"])
	}
}
