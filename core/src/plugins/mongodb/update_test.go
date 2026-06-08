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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestParseMongoReplacementDocumentPreservesAuthoredOrder(t *testing.T) {
	document, id, err := parseMongoReplacementDocument(`{"z":1,"_id":"507f1f77bcf86cd799439011","nested":{"b":2,"a":1},"a":2}`)
	if err != nil {
		t.Fatalf("expected replacement document to parse, got %v", err)
	}

	objectID, ok := id.(primitive.ObjectID)
	if !ok {
		t.Fatalf("expected ObjectID identity, got %T", id)
	}
	if objectID.Hex() != "507f1f77bcf86cd799439011" {
		t.Fatalf("unexpected object id %s", objectID.Hex())
	}

	keys := make([]string, len(document))
	for index, element := range document {
		keys[index] = element.Key
	}
	expectedKeys := []string{"_id", "z", "nested", "a"}
	for index, expected := range expectedKeys {
		if keys[index] != expected {
			t.Fatalf("expected key %d to be %q, got keys %#v", index, expected, keys)
		}
	}

	if document[0].Value != objectID {
		t.Fatalf("expected replacement _id to be normalized ObjectID, got %#v", document[0].Value)
	}

	nested, ok := document[2].Value.(bson.D)
	if !ok {
		t.Fatalf("expected nested document to preserve order as bson.D, got %T", document[2].Value)
	}
	if nested[0].Key != "b" || nested[1].Key != "a" {
		t.Fatalf("expected nested key order b,a, got %#v", nested)
	}
}

func TestParseMongoReplacementDocumentRequiresID(t *testing.T) {
	if _, _, err := parseMongoReplacementDocument(`{"name":"alice"}`); err == nil {
		t.Fatalf("expected missing _id to fail")
	}
}
