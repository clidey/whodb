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
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestParseMongoShellCommand(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantCollection string
		wantMethod     string
		wantArgs       string
	}{
		{
			name:           "collection method",
			input:          `db.users.find({ "age": { "$gt": 25 } })`,
			wantCollection: "users",
			wantMethod:     "find",
			wantArgs:       `{ "age": { "$gt": 25 } }`,
		},
		{
			name:           "getCollection method",
			input:          `db.getCollection("audit-log").find({})`,
			wantCollection: "audit-log",
			wantMethod:     "find",
			wantArgs:       `{}`,
		},
		{
			name:       "database method",
			input:      `db.dropDatabase()`,
			wantMethod: "dropDatabase",
			wantArgs:   "",
		},
		{
			name:           "comments",
			input:          "// list users\n db.users.find({ active: true })",
			wantCollection: "users",
			wantMethod:     "find",
			wantArgs:       `{ active: true }`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseMongoShellCommand(test.input)
			if err != nil {
				t.Fatalf("expected command to parse, got %v", err)
			}
			if got.Collection != test.wantCollection || got.Method != test.wantMethod || got.RawArgs != test.wantArgs {
				t.Fatalf("unexpected command: %#v", got)
			}
		})
	}
}

func TestParseMongoShellCommandRejectsJavaScript(t *testing.T) {
	_, err := parseMongoShellCommand(`const users = db.users.find({}).toArray()`)
	if err == nil || !strings.Contains(err.Error(), "db.collection.method") {
		t.Fatalf("expected bounded shell error, got %v", err)
	}
}

func TestParseMongoShellArgsSupportsCommonShellLiterals(t *testing.T) {
	args, err := parseMongoShellArgs(`{_id: ObjectId("507f1f77bcf86cd799439011"), name: 'Ada', createdAt: ISODate("2026-01-02T03:04:05Z"), count: NumberInt(2), nested: {flag: true,},}`)
	if err != nil {
		t.Fatalf("expected shell args to parse, got %v", err)
	}
	if len(args) != 1 {
		t.Fatalf("expected one argument, got %#v", args)
	}
	doc, ok := args[0].(bson.D)
	if !ok {
		t.Fatalf("expected document argument, got %T", args[0])
	}

	values := map[string]any{}
	for _, elem := range doc {
		values[elem.Key] = elem.Value
	}
	if _, ok := values["_id"].(bson.ObjectID); !ok {
		t.Fatalf("expected ObjectId to parse, got %#v", values["_id"])
	}
	if values["name"] != "Ada" {
		t.Fatalf("expected single quoted string to parse, got %#v", values["name"])
	}
	if values["count"] != int32(2) {
		t.Fatalf("expected NumberInt to parse as int32, got %#v", values["count"])
	}
	if _, ok := values["nested"].(bson.D); !ok {
		t.Fatalf("expected nested document to parse, got %#v", values["nested"])
	}
}
