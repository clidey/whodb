/*
 * // Copyright 2025 Clidey, Inc.
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package mongodb

import (
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// normalizeMongoID converts supported ID formats to a MongoDB-friendly value.
// Returns the original value if no conversion is possible.
func normalizeMongoID(value any) (any, error) {
	switch v := value.(type) {
	case primitive.ObjectID:
		return v, nil
	case string:
		oid, err := primitive.ObjectIDFromHex(v)
		if err != nil {
			// Not an ObjectID, use the raw string as-is
			return v, nil
		}
		return oid, nil
	default:
		return value, nil
	}
}

// coerceMongoValue attempts to convert a string to a sensible BSON value.
// Numbers and booleans are converted; everything else remains a string.
func coerceMongoValue(key string, raw string) any {
	if key == "_id" {
		id, err := normalizeMongoID(raw)
		if err == nil {
			return id
		}
	}

	if b, err := strconv.ParseBool(raw); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return f
	}
	return raw
}

// Helper functions for logging
func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getDocumentFieldNames(doc bson.M) []string {
	fields := make([]string, 0, len(doc))
	for k := range doc {
		fields = append(fields, k)
	}
	return fields
}
