/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mongodb

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/clidey/whodb/core/src/query"
)

func mongoAtomicWhere(key, operator, value string) *query.WhereCondition {
	return &query.WhereCondition{
		Type: query.WhereConditionTypeAtomic,
		Atomic: &query.AtomicWhereCondition{
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
		filter, err := convertWhereConditionToMongoDB(&query.WhereCondition{
			Type: query.WhereConditionTypeAnd,
			And: &query.OperationWhereCondition{
				Children: []*query.WhereCondition{
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
