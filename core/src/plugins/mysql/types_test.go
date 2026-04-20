package mysql

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func TestNormalizeType(t *testing.T) {
	if got := NormalizeType("integer"); got != "INT" {
		t.Fatalf("expected INT, got %q", got)
	}

	if got := NormalizeType("character varying(50)"); got != "VARCHAR(50)" {
		t.Fatalf("expected VARCHAR(50), got %q", got)
	}
}

func TestSourceSessionMetadataUsesSourceType(t *testing.T) {
	meta, ok := sourcecatalog.ResolveSessionMetadata(string(engine.DatabaseType_MySQL))
	if !ok || meta == nil {
		t.Fatalf("expected metadata, got nil")
	}
	if meta.AliasMap["INTEGER"] != "INT" {
		t.Fatalf("expected INTEGER alias to be INT, got %q", meta.AliasMap["INTEGER"])
	}
}
