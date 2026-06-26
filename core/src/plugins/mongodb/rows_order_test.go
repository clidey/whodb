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

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMarshalMongoDocumentJSONPreservesFieldOrder(t *testing.T) {
	doc := bson.D{
		{Key: "z", Value: int32(1)},
		{Key: "nested", Value: bson.D{
			{Key: "b", Value: int32(2)},
			{Key: "a", Value: int32(1)},
		}},
		{Key: "arr", Value: bson.A{
			bson.D{
				{Key: "y", Value: int32(1)},
				{Key: "x", Value: int32(2)},
			},
		}},
		{Key: "_id", Value: "1"},
		{Key: "a", Value: int32(3)},
	}

	gotBytes, err := marshalMongoDocumentJSON(doc)
	if err != nil {
		t.Fatalf("marshalMongoDocumentJSON returned error: %v", err)
	}

	want := `{"z":1,"nested":{"b":2,"a":1},"arr":[{"y":1,"x":2}],"_id":"1","a":3}`
	if string(gotBytes) != want {
		t.Fatalf("expected %s, got %s", want, string(gotBytes))
	}
}
