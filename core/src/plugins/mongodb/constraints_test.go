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
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestParseMongoDBJsonSchema(t *testing.T) {
	// Test schema matching the validated_products collection
	schema := bson.M{
		"bsonType": "object",
		"required": bson.A{"name", "price", "status", "category"},
		"properties": bson.M{
			"name": bson.M{
				"bsonType":  "string",
				"maxLength": float64(100),
			},
			"price": bson.M{
				"bsonType": "double",
				"minimum":  float64(0),
				"maximum":  float64(99999.99),
			},
			"status": bson.M{
				"bsonType": "string",
				"enum":     bson.A{"active", "inactive", "discontinued"},
			},
			"category": bson.M{
				"bsonType": "string",
				"enum":     bson.A{"electronics", "clothing", "food", "other"},
			},
			"stock_quantity": bson.M{
				"bsonType": "int",
				"minimum":  float64(0),
				"maximum":  float64(10000),
			},
			"description": bson.M{
				"bsonType":  "string",
				"maxLength": float64(500),
			},
			"rating": bson.M{
				"bsonType": "double",
				"minimum":  float64(0),
				"maximum":  float64(5),
			},
		},
	}

	constraints := parseMongoDBJsonSchema(schema)

	// Test required fields have nullable: false
	t.Run("required fields are not nullable", func(t *testing.T) {
		requiredFields := []string{"name", "price", "status", "category"}
		for _, field := range requiredFields {
			if constraints[field] == nil {
				t.Errorf("expected constraints for required field %q", field)
				continue
			}
			if nullable, ok := constraints[field]["nullable"].(bool); !ok || nullable {
				t.Errorf("expected field %q to have nullable=false, got %v", field, constraints[field]["nullable"])
			}
		}
	})

	// Test optional fields have nullable: true
	t.Run("optional fields are nullable", func(t *testing.T) {
		optionalFields := []string{"stock_quantity", "description", "rating"}
		for _, field := range optionalFields {
			if constraints[field] == nil {
				t.Errorf("expected constraints for optional field %q", field)
				continue
			}
			if nullable, ok := constraints[field]["nullable"].(bool); !ok || !nullable {
				t.Errorf("expected field %q to have nullable=true, got %v", field, constraints[field]["nullable"])
			}
		}
	})

	// Test enum values are extracted as check_values
	t.Run("enum values extracted as check_values", func(t *testing.T) {
		statusValues, ok := constraints["status"]["check_values"].([]string)
		if !ok {
			t.Fatalf("expected check_values for status field")
		}
		expected := []string{"active", "inactive", "discontinued"}
		if len(statusValues) != len(expected) {
			t.Errorf("expected %d status values, got %d", len(expected), len(statusValues))
		}
		for i, v := range expected {
			if statusValues[i] != v {
				t.Errorf("expected status value %q at index %d, got %q", v, i, statusValues[i])
			}
		}

		categoryValues, ok := constraints["category"]["check_values"].([]string)
		if !ok {
			t.Fatalf("expected check_values for category field")
		}
		expectedCat := []string{"electronics", "clothing", "food", "other"}
		if len(categoryValues) != len(expectedCat) {
			t.Errorf("expected %d category values, got %d", len(expectedCat), len(categoryValues))
		}
	})

	// Test numeric constraints are extracted
	t.Run("numeric min/max extracted as check_min/check_max", func(t *testing.T) {
		// Price constraints
		if min, ok := constraints["price"]["check_min"].(float64); !ok || min != 0 {
			t.Errorf("expected price check_min=0, got %v", constraints["price"]["check_min"])
		}
		if max, ok := constraints["price"]["check_max"].(float64); !ok || max != 99999.99 {
			t.Errorf("expected price check_max=99999.99, got %v", constraints["price"]["check_max"])
		}

		// Rating constraints
		if min, ok := constraints["rating"]["check_min"].(float64); !ok || min != 0 {
			t.Errorf("expected rating check_min=0, got %v", constraints["rating"]["check_min"])
		}
		if max, ok := constraints["rating"]["check_max"].(float64); !ok || max != 5 {
			t.Errorf("expected rating check_max=5, got %v", constraints["rating"]["check_max"])
		}

		// Stock quantity constraints (int32 values)
		if min, ok := constraints["stock_quantity"]["check_min"].(float64); !ok || min != 0 {
			t.Errorf("expected stock_quantity check_min=0, got %v", constraints["stock_quantity"]["check_min"])
		}
		if max, ok := constraints["stock_quantity"]["check_max"].(float64); !ok || max != 10000 {
			t.Errorf("expected stock_quantity check_max=10000, got %v", constraints["stock_quantity"]["check_max"])
		}
	})

	// Test maxLength is extracted as length
	t.Run("maxLength extracted as length", func(t *testing.T) {
		if length, ok := constraints["name"]["length"].(int); !ok || length != 100 {
			t.Errorf("expected name length=100, got %v", constraints["name"]["length"])
		}
		if length, ok := constraints["description"]["length"].(int); !ok || length != 500 {
			t.Errorf("expected description length=500, got %v", constraints["description"]["length"])
		}
	})

	// Test bsonType is extracted as type
	t.Run("bsonType extracted as type", func(t *testing.T) {
		if typ, ok := constraints["name"]["type"].(string); !ok || typ != "string" {
			t.Errorf("expected name type=string, got %v", constraints["name"]["type"])
		}
		if typ, ok := constraints["price"]["type"].(string); !ok || typ != "double" {
			t.Errorf("expected price type=double, got %v", constraints["price"]["type"])
		}
		if typ, ok := constraints["stock_quantity"]["type"].(string); !ok || typ != "int" {
			t.Errorf("expected stock_quantity type=int, got %v", constraints["stock_quantity"]["type"])
		}
	})
}

func TestParseMongoDBJsonSchemaEmptySchema(t *testing.T) {
	constraints := parseMongoDBJsonSchema(bson.M{})
	if len(constraints) != 0 {
		t.Errorf("expected empty constraints for empty schema, got %v", constraints)
	}
}

func TestParseMongoDBJsonSchemaNoProperties(t *testing.T) {
	schema := bson.M{
		"bsonType": "object",
		"required": bson.A{"field1"},
	}
	constraints := parseMongoDBJsonSchema(schema)
	if len(constraints) != 0 {
		t.Errorf("expected empty constraints when no properties defined, got %v", constraints)
	}
}

// TestParseMongoDBJsonSchemaInt32Values verifies that int32 values (as MongoDB
// actually stores integer literals in $jsonSchema) are correctly converted.
func TestParseMongoDBJsonSchemaInt32Values(t *testing.T) {
	schema := bson.M{
		"bsonType": "object",
		"properties": bson.M{
			"rating": bson.M{
				"bsonType": "double",
				"minimum":  int32(0),
				"maximum":  int32(5),
			},
			"name": bson.M{
				"bsonType":  "string",
				"maxLength": int32(100),
			},
		},
	}

	constraints := parseMongoDBJsonSchema(schema)

	t.Run("int32 minimum/maximum converted to float64", func(t *testing.T) {
		if minVal, ok := constraints["rating"]["check_min"].(float64); !ok || minVal != 0 {
			t.Errorf("expected rating check_min=0 (float64), got %v (%T)", constraints["rating"]["check_min"], constraints["rating"]["check_min"])
		}
		if maxVal, ok := constraints["rating"]["check_max"].(float64); !ok || maxVal != 5 {
			t.Errorf("expected rating check_max=5 (float64), got %v (%T)", constraints["rating"]["check_max"], constraints["rating"]["check_max"])
		}
	})

	t.Run("int32 maxLength converted to int", func(t *testing.T) {
		if length, ok := constraints["name"]["length"].(int); !ok || length != 100 {
			t.Errorf("expected name length=100 (int), got %v (%T)", constraints["name"]["length"], constraints["name"]["length"])
		}
	})
}
