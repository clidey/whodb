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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/bson"
)

func convertWhereConditionToMongoDB(where *model.WhereCondition) (bson.M, error) {
	if where == nil {
		return bson.M{}, nil
	}

	// Normalize operator to lower for comparisons
	getOp := func(op string) string { return strings.ToLower(op) }

	switch where.Type {
	case model.WhereConditionTypeAtomic:
		if where.Atomic == nil {
			return nil, fmt.Errorf("atomic condition must have an atomicwherecondition")
		}

		operator := getOp(where.Atomic.Operator)

		switch operator {
		case "eq", "ne", "gt", "gte", "lt", "lte":
			mongoOperator := "$" + operator
			value := convertMongoValue(where.Atomic.Key, where.Atomic.Value)
			return bson.M{where.Atomic.Key: bson.M{mongoOperator: value}}, nil

		case "in", "nin":
			values := parseCommaSeparatedValues(where.Atomic.Key, where.Atomic.Value)
			mongoOperator := "$" + operator
			return bson.M{where.Atomic.Key: bson.M{mongoOperator: values}}, nil

		case "regex":
			return bson.M{where.Atomic.Key: bson.M{"$regex": where.Atomic.Value}}, nil

		case "exists":
			exists, err := strconv.ParseBool(where.Atomic.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid exists value: %s", where.Atomic.Value)
			}
			return bson.M{where.Atomic.Key: bson.M{"$exists": exists}}, nil

		case "type":
			return bson.M{where.Atomic.Key: bson.M{"$type": where.Atomic.Value}}, nil

		case "expr":
			// Expect a JSON expression; try to decode
			var expr any
			if err := json.Unmarshal([]byte(where.Atomic.Value), &expr); err != nil {
				return nil, fmt.Errorf("invalid expr payload: %w", err)
			}
			return bson.M{"$expr": expr}, nil

		case "mod":
			parts := strings.Split(where.Atomic.Value, ",")
			if len(parts) != 2 {
				return nil, fmt.Errorf("mod expects 'divisor,remainder'")
			}
			div, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
			rem, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("mod operands must be integers")
			}
			return bson.M{where.Atomic.Key: bson.M{"$mod": []int64{div, rem}}}, nil

		case "all":
			values := parseCommaSeparatedValues(where.Atomic.Key, where.Atomic.Value)
			return bson.M{where.Atomic.Key: bson.M{"$all": values}}, nil

		case "elemmatch":
			var elem bson.M
			if err := json.Unmarshal([]byte(where.Atomic.Value), &elem); err != nil {
				return nil, fmt.Errorf("elemMatch expects JSON object: %w", err)
			}
			return bson.M{where.Atomic.Key: bson.M{"$elemMatch": elem}}, nil

		case "size":
			size, err := strconv.ParseInt(where.Atomic.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("size expects integer: %w", err)
			}
			return bson.M{where.Atomic.Key: bson.M{"$size": size}}, nil

		case "bitsallclear", "bitsallset", "bitsanyclear", "bitsanyset":
			mask, err := strconv.ParseInt(where.Atomic.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s expects integer bitmask", operator)
			}
			return bson.M{where.Atomic.Key: bson.M{"$" + operator: mask}}, nil

		case "geointersects", "geowithin", "near", "nearsphere":
			var payload any
			if err := json.Unmarshal([]byte(where.Atomic.Value), &payload); err != nil {
				return nil, fmt.Errorf("%s expects JSON payload: %w", operator, err)
			}
			return bson.M{where.Atomic.Key: bson.M{"$" + operator: payload}}, nil

		default:
			return nil, fmt.Errorf("unsupported operator: %s", where.Atomic.Operator)
		}

	case model.WhereConditionTypeAnd:
		if where.And == nil || len(where.And.Children) == 0 {
			return bson.M{}, nil
		}

		andConditions := []bson.M{}
		for _, child := range where.And.Children {
			childCondition, err := convertWhereConditionToMongoDB(child)
			if err != nil {
				log.WithError(err).Error("Failed to convert child AND condition to MongoDB filter")
				return nil, err
			}
			andConditions = append(andConditions, childCondition)
		}

		return bson.M{"$and": andConditions}, nil

	case model.WhereConditionTypeOr:
		if where.Or == nil || len(where.Or.Children) == 0 {
			return bson.M{}, nil
		}

		orConditions := []bson.M{}
		for _, child := range where.Or.Children {
			childCondition, err := convertWhereConditionToMongoDB(child)
			if err != nil {
				log.WithError(err).Error("Failed to convert child OR condition to MongoDB filter")
				return nil, err
			}
			orConditions = append(orConditions, childCondition)
		}

		return bson.M{"$or": orConditions}, nil

	default:
		return nil, fmt.Errorf("unknown whereconditiontype: %v", where.Type)
	}
}

// convertMongoValue handles ObjectID conversion for _id and basic numeric/bool coercion.
// Used for query building where type hints are not available.
func convertMongoValue(key string, raw string) any {
	if key == "_id" {
		id, err := normalizeMongoID(raw)
		if err == nil {
			return id
		}
		return raw
	}
	return coerceMongoValue(key, raw, "") // No type hint for queries
}

func parseCommaSeparatedValues(key string, raw string) []any {
	if strings.TrimSpace(raw) == "" {
		return []any{}
	}
	parts := strings.Split(raw, ",")
	values := make([]any, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		values = append(values, convertMongoValue(key, v))
	}
	return values
}
