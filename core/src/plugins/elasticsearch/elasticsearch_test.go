package elasticsearch

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func TestConvertAtomicConditionToES(t *testing.T) {
	tests := []struct {
		name   string
		atomic *model.AtomicWhereCondition
		want   map[string]any
	}{
		{
			name: "id equality uses ids query",
			atomic: &model.AtomicWhereCondition{
				Key: "_id", Operator: "=", Value: "doc-1",
			},
			want: map[string]any{
				"ids": map[string]any{
					"values": []any{"doc-1"},
				},
			},
		},
		{
			name: "contains uses wildcard",
			atomic: &model.AtomicWhereCondition{
				Key: "email", Operator: "CONTAINS", Value: "example.com",
			},
			want: map[string]any{
				"wildcard": map[string]any{
					"email": map[string]any{"value": "*example.com*"},
				},
			},
		},
		{
			name: "terms parses csv values",
			atomic: &model.AtomicWhereCondition{
				Key: "status", Operator: "TERMS", Value: "paid, pending",
			},
			want: map[string]any{
				"terms": map[string]any{
					"status": []any{"paid", "pending"},
				},
			},
		},
		{
			name: "range accepts open upper bound",
			atomic: &model.AtomicWhereCondition{
				Key: "price", Operator: "RANGE", Value: "10,",
			},
			want: map[string]any{
				"range": map[string]any{
					"price": map[string]any{"gte": "10"},
				},
			},
		},
		{
			name: "unknown operators fall back to match",
			atomic: &model.AtomicWhereCondition{
				Key: "notes", Operator: "UNSUPPORTED", Value: "needle",
			},
			want: map[string]any{
				"match": map[string]any{
					"notes": "needle",
				},
			},
		},
	}

	for _, tt := range tests {
		got, err := convertAtomicConditionToES(tt.atomic)
		if err != nil {
			t.Fatalf("%s: expected conversion to succeed, got %v", tt.name, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("%s: unexpected clause\n got: %#v\nwant: %#v", tt.name, got, tt.want)
		}
	}
}

func TestConvertWhereConditionToES(t *testing.T) {
	where := &model.WhereCondition{
		Type: model.WhereConditionTypeAnd,
		And: &model.OperationWhereCondition{
			Children: []*model.WhereCondition{
				{
					Type: model.WhereConditionTypeAtomic,
					Atomic: &model.AtomicWhereCondition{
						Key:      "status",
						Operator: "=",
						Value:    "paid",
					},
				},
				{
					Type: model.WhereConditionTypeOr,
					Or: &model.OperationWhereCondition{
						Children: []*model.WhereCondition{
							{
								Type: model.WhereConditionTypeAtomic,
								Atomic: &model.AtomicWhereCondition{
									Key:      "priority",
									Operator: "=",
									Value:    "high",
								},
							},
							{
								Type: model.WhereConditionTypeAtomic,
								Atomic: &model.AtomicWhereCondition{
									Key:      "priority",
									Operator: "=",
									Value:    "urgent",
								},
							},
						},
					},
				},
			},
		},
	}

	got, err := convertWhereConditionToES(where)
	if err != nil {
		t.Fatalf("expected nested ES condition conversion to succeed, got %v", err)
	}
	mustClauses, ok := got["must"].([]map[string]any)
	if !ok || len(mustClauses) != 2 {
		t.Fatalf("expected two must clauses, got %#v", got)
	}
	nestedBool, ok := mustClauses[1]["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected OR child to be wrapped in bool query, got %#v", mustClauses[1])
	}
	if nestedBool["minimum_should_match"] != 1 {
		t.Fatalf("expected minimum_should_match=1, got %#v", nestedBool)
	}

	if _, err := convertWhereConditionToES(&model.WhereCondition{Type: model.WhereConditionTypeAtomic}); err == nil {
		t.Fatal("expected invalid atomic condition to fail")
	}
}

func TestElasticSearchHelpers(t *testing.T) {
	if got := parseCSVToSlice("paid, pending"); !reflect.DeepEqual(got, []any{"paid", "pending"}) {
		t.Fatalf("expected CSV values to be trimmed, got %#v", got)
	}
	min, max := parseRangeBounds("10, 20")
	if min != "10" || max != "20" {
		t.Fatalf("expected parsed range bounds, got %q %q", min, max)
	}

	if got := inferElasticSearchType(map[string]any{"id": 1}); got != "object" {
		t.Fatalf("expected object type inference, got %q", got)
	}
	if got := inferElasticSearchType([]any{"a"}); got != "array" {
		t.Fatalf("expected array type inference, got %q", got)
	}
	if got := mergeElasticTypes("text", "keyword"); got != "mixed" {
		t.Fatalf("expected mixed type merge, got %q", got)
	}

	mappings := buildElasticMappings([]engine.Record{
		{Key: "title", Value: "text"},
		{Key: "price", Value: "decimal"},
		{Key: "", Value: "text"},
	})
	if len(mappings) != 2 {
		t.Fatalf("expected empty field names to be filtered, got %#v", mappings)
	}
	if mappings["title"].(map[string]any)["type"] != "text" || mappings["price"].(map[string]any)["type"] != "double" {
		t.Fatalf("unexpected mappings: %#v", mappings)
	}

	res := &esapi.Response{
		StatusCode: 400,
		Body:       io.NopCloser(strings.NewReader(`{"error":"bad query"}`)),
	}
	if got := formatElasticError(res); got != `{"error":"bad query"}` {
		t.Fatalf("expected formatted body error, got %q", got)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected response body to remain readable, got %v", err)
	}
	if string(body) != `{"error":"bad query"}` {
		t.Fatalf("expected response body to be restored, got %q", string(body))
	}
}
