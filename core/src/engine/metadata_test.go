package engine

import (
	"errors"
	"testing"

	"github.com/clidey/whodb/core/src/source"
)

func TestValidateColumnTypeAllowsWhenMetadataMissing(t *testing.T) {
	if err := ValidateColumnType("NOT_A_REAL_TYPE", "TestDB", nil); err != nil {
		t.Fatalf("expected nil error when metadata is nil, got %v", err)
	}

	if err := ValidateColumnType("NOT_A_REAL_TYPE", "TestDB", &source.TypeSessionMetadata{}); err != nil {
		t.Fatalf("expected nil error when metadata has no type definitions, got %v", err)
	}
}

func TestValidateColumnTypeRejectsUnsupported(t *testing.T) {
	meta := &source.TypeSessionMetadata{
		TypeDefinitions: []TypeDefinition{
			{ID: "INTEGER"},
		},
		AliasMap: map[string]string{},
	}

	err := ValidateColumnType("NOT_A_REAL_TYPE", "TestDB", meta)
	var unsupported *UnsupportedTypeError
	if err == nil || !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedTypeError, got %v", err)
	}
	if unsupported.TypeName != "NOT_A_REAL_TYPE" {
		t.Fatalf("expected TypeName to be preserved, got %q", unsupported.TypeName)
	}
	if unsupported.DatabaseType != "TestDB" {
		t.Fatalf("expected DatabaseType to be preserved, got %q", unsupported.DatabaseType)
	}
}

func TestValidateColumnTypeResolvesAliases(t *testing.T) {
	meta := &source.TypeSessionMetadata{
		TypeDefinitions: []TypeDefinition{
			{ID: "INTEGER"},
			{ID: "VARCHAR", HasLength: true},
		},
		AliasMap: map[string]string{
			"INT": "INTEGER",
		},
	}

	if err := ValidateColumnType("int", "TestDB", meta); err != nil {
		t.Fatalf("expected alias INT -> INTEGER to validate, got %v", err)
	}

	if err := ValidateColumnType("VARCHAR(255)", "TestDB", meta); err != nil {
		t.Fatalf("expected parametrized type to validate, got %v", err)
	}
}
