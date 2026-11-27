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
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestFormatBSONValueUnwrapsByTypeOnly(t *testing.T) {
	objectID, err := bson.ObjectIDFromHex("65a0f0f0f0f0f0f0f0f0f0f0")
	if err != nil {
		t.Fatalf("failed to build ObjectID: %v", err)
	}
	when := bson.NewDateTimeFromTime(time.Date(2024, 1, 5, 13, 14, 15, 123000000, time.UTC))

	cases := []struct {
		name  string
		value any
		want  string
		text  bool
	}{
		{"string", "=1+1", "=1+1", true},
		{"negative number", int32(-5), "-5", false},
		{"object id", objectID, "65a0f0f0f0f0f0f0f0f0f0f0", false},
		{"date", when, "2024-01-05T13:14:15.123Z", false},
		// A stored sub-document shaped like an ObjectID stays JSON, so its
		// text can never reach a cell bare.
		{"lookalike document", bson.D{{Key: "$oid", Value: "=cmd()"}}, `{"$oid":"=cmd()"}`, false},
		{"null", nil, "", false},
	}
	for _, tc := range cases {
		got, text := formatBSONValue(tc.value)
		if got != tc.want || text != tc.text {
			t.Errorf("%s: formatBSONValue = %q free text %v, want %q free text %v", tc.name, got, text, tc.want, tc.text)
		}
	}
}
