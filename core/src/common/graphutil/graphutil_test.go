package graphutil

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestInferForeignKeys(t *testing.T) {
	got := InferForeignKeys(
		"orders",
		[]string{"user_id", "accountid", "payments.id", "_id", "notes"},
		[]string{"users", "accounts", "payments", "orders"},
	)

	want := map[string]string{
		"users":    "user_id",
		"accounts": "accountid",
		"payments": "payments.id",
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d inferred keys, got %d: %#v", len(want), len(got), got)
	}
	for key, expected := range want {
		if got[key] != expected {
			t.Fatalf("expected %s -> %q, got %q", key, expected, got[key])
		}
	}
	if _, exists := got["_id"]; exists {
		t.Fatal("did not expect MongoDB _id to be inferred as a foreign key")
	}
}

func TestBuildGraphUnits(t *testing.T) {
	units := []engine.StorageUnit{
		{Name: "orders"},
		{Name: "users"},
	}
	relations := []Relation{{
		Table1:       "orders",
		Table2:       "users",
		Relation:     "ManyToOne",
		SourceColumn: "user_id",
		TargetColumn: "id",
	}}

	graphUnits := BuildGraphUnits(relations, units)

	if len(graphUnits) != 2 {
		t.Fatalf("expected 2 graph units, got %d", len(graphUnits))
	}

	if len(graphUnits[0].Relations) != 1 {
		t.Fatalf("expected first unit to have one relation, got %d", len(graphUnits[0].Relations))
	}
	relation := graphUnits[0].Relations[0]
	if relation.Name != "users" {
		t.Fatalf("expected relation target users, got %q", relation.Name)
	}
	if relation.RelationshipType != "ManyToOne" {
		t.Fatalf("expected ManyToOne relation, got %q", relation.RelationshipType)
	}
	if relation.SourceColumn == nil || *relation.SourceColumn != "user_id" {
		t.Fatalf("expected source column user_id, got %#v", relation.SourceColumn)
	}
	if relation.TargetColumn == nil || *relation.TargetColumn != "id" {
		t.Fatalf("expected target column id, got %#v", relation.TargetColumn)
	}

	if len(graphUnits[1].Relations) != 0 {
		t.Fatalf("expected unrelated unit to have no relations, got %#v", graphUnits[1].Relations)
	}
}
