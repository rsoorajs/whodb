package mongodb

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func mongoAtomicWhere(key, operator, value string) *model.WhereCondition {
	return &model.WhereCondition{
		Type: model.WhereConditionTypeAtomic,
		Atomic: &model.AtomicWhereCondition{
			Key:      key,
			Operator: operator,
			Value:    value,
		},
	}
}

func TestConvertWhereConditionToMongoDB(t *testing.T) {
	id := bson.NewObjectID()

	t.Run("coerces object ids for equality", func(t *testing.T) {
		filter, err := convertWhereConditionToMongoDB(mongoAtomicWhere("_id", "eq", id.Hex()))
		if err != nil {
			t.Fatalf("expected _id equality conversion to succeed, got %v", err)
		}
		gotID, ok := filter["_id"].(bson.M)["$eq"].(bson.ObjectID)
		if !ok || gotID != id {
			t.Fatalf("expected _id value to be converted to ObjectID, got %#v", filter)
		}
	})

	t.Run("supports csv list operators", func(t *testing.T) {
		filter, err := convertWhereConditionToMongoDB(mongoAtomicWhere("status", "in", "paid, pending"))
		if err != nil {
			t.Fatalf("expected IN conversion to succeed, got %v", err)
		}
		if !reflect.DeepEqual(filter, bson.M{
			"status": bson.M{"$in": []any{"paid", "pending"}},
		}) {
			t.Fatalf("unexpected IN filter: %#v", filter)
		}
	})

	t.Run("supports exists and expr operators", func(t *testing.T) {
		filter, err := convertWhereConditionToMongoDB(mongoAtomicWhere("nickname", "exists", "true"))
		if err != nil {
			t.Fatalf("expected exists conversion to succeed, got %v", err)
		}
		if !reflect.DeepEqual(filter, bson.M{"nickname": bson.M{"$exists": true}}) {
			t.Fatalf("unexpected exists filter: %#v", filter)
		}

		filter, err = convertWhereConditionToMongoDB(mongoAtomicWhere("ignored", "expr", `{"$gt":["$qty", 0]}`))
		if err != nil {
			t.Fatalf("expected expr conversion to succeed, got %v", err)
		}
		if _, ok := filter["$expr"]; !ok {
			t.Fatalf("expected expr filter payload, got %#v", filter)
		}
	})

	t.Run("supports nested AND trees", func(t *testing.T) {
		filter, err := convertWhereConditionToMongoDB(&model.WhereCondition{
			Type: model.WhereConditionTypeAnd,
			And: &model.OperationWhereCondition{
				Children: []*model.WhereCondition{
					mongoAtomicWhere("qty", "gte", "10"),
					mongoAtomicWhere("status", "eq", "paid"),
				},
			},
		})
		if err != nil {
			t.Fatalf("expected nested AND conversion to succeed, got %v", err)
		}
		andClauses, ok := filter["$and"].([]bson.M)
		if !ok || len(andClauses) != 2 {
			t.Fatalf("expected two AND clauses, got %#v", filter)
		}
	})

	t.Run("returns helpful validation errors", func(t *testing.T) {
		if _, err := convertWhereConditionToMongoDB(mongoAtomicWhere("flag", "exists", "not-bool")); err == nil {
			t.Fatal("expected invalid exists payload to fail")
		}
		if _, err := convertWhereConditionToMongoDB(mongoAtomicWhere("value", "mod", "4")); err == nil {
			t.Fatal("expected invalid mod payload to fail")
		}
	})
}

func TestMongoDBHelpers(t *testing.T) {
	if got := inferMongoDBType(bson.NewObjectID()); got != "ObjectId" {
		t.Fatalf("expected ObjectId type inference, got %q", got)
	}
	if got := inferMongoDBType(bson.DateTime(123)); got != "date" {
		t.Fatalf("expected date type inference, got %q", got)
	}
	if got := mergeMongoTypes("string", "int"); got != "mixed" {
		t.Fatalf("expected conflicting mongo types to become mixed, got %q", got)
	}

	dupErr := mongo.WriteException{
		WriteErrors: mongo.WriteErrors{
			{Code: 11000, Message: "duplicate key"},
		},
	}
	if got := handleMongoError(dupErr); got == nil || !strings.Contains(got.Error(), "duplicate key") {
		t.Fatalf("expected duplicate key errors to be normalized, got %v", got)
	}

	commandErr := mongo.CommandError{Code: 121, Message: "schema mismatch"}
	if got := handleMongoError(commandErr); got == nil || !strings.Contains(got.Error(), "document validation failed") {
		t.Fatalf("expected command errors to be normalized, got %v", got)
	}

	if got := handleMongoError(errors.New("line one\nline two")); got == nil || strings.Contains(got.Error(), "\n") {
		t.Fatalf("expected generic mongo errors to be sanitized, got %v", got)
	}
}
