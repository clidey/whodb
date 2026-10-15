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

package platform

import (
	"fmt"
	"sort"
	"strings"
)

// WhereFilterOperators lists the operators accepted in the simple filter
// grammar shared by every ontology read tool.
var WhereFilterOperators = []string{"eq", "neq", "gt", "gte", "lt", "lte", "contains", "icontains", "startswith", "in", "nin", "between", "isnull"}

// NormalizeWhereCondition accepts either a platform WhereCondition object
// (detected by its Type key) or the simple filter grammar used by
// OntologyQuery: {prop: value}, {prop: {op: value}}, {_and: [...]}, {_or: [...]}.
// The simple grammar is translated into an equivalent WhereCondition so agents
// can use one filter syntax for query, aggregate, stats and similarity reads.
func NormalizeWhereCondition(where map[string]any) (map[string]any, error) {
	if len(where) == 0 {
		return nil, nil
	}
	if _, isCondition := where["Type"]; isCondition {
		return where, nil
	}
	return whereFilterToCondition(where)
}

func whereFilterToCondition(filter map[string]any) (map[string]any, error) {
	children := make([]map[string]any, 0, len(filter))
	keys := make([]string, 0, len(filter))
	for key := range filter {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := filter[key]
		switch key {
		case "_and", "_or":
			list, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("where.%s must be a list of filters", key)
			}
			group := make([]map[string]any, 0, len(list))
			for index, item := range list {
				sub, ok := item.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("where.%s[%d] must be an object", key, index)
				}
				condition, err := whereFilterToCondition(sub)
				if err != nil {
					return nil, err
				}
				if condition != nil {
					group = append(group, condition)
				}
			}
			if len(group) == 0 {
				continue
			}
			node := "And"
			if key == "_or" {
				node = "Or"
			}
			children = append(children, map[string]any{"Type": node, node: map[string]any{"Children": group}})
		default:
			atomics, err := whereFieldConditions(key, value)
			if err != nil {
				return nil, err
			}
			children = append(children, atomics...)
		}
	}
	switch len(children) {
	case 0:
		return nil, nil
	case 1:
		return children[0], nil
	default:
		return map[string]any{"Type": "And", "And": map[string]any{"Children": children}}, nil
	}
}

func whereFieldConditions(field string, value any) ([]map[string]any, error) {
	ops, isOps := value.(map[string]any)
	if !isOps {
		return []map[string]any{whereAtomic(field, "=", whereScalar(value))}, nil
	}
	opNames := make([]string, 0, len(ops))
	for op := range ops {
		opNames = append(opNames, op)
	}
	sort.Strings(opNames)
	out := make([]map[string]any, 0, len(opNames))
	for _, op := range opNames {
		operand := ops[op]
		switch strings.ToLower(op) {
		case "eq":
			out = append(out, whereAtomic(field, "=", whereScalar(operand)))
		case "neq", "ne":
			out = append(out, whereAtomic(field, "!=", whereScalar(operand)))
		case "gt":
			out = append(out, whereAtomic(field, ">", whereScalar(operand)))
		case "gte":
			out = append(out, whereAtomic(field, ">=", whereScalar(operand)))
		case "lt":
			out = append(out, whereAtomic(field, "<", whereScalar(operand)))
		case "lte":
			out = append(out, whereAtomic(field, "<=", whereScalar(operand)))
		case "contains":
			out = append(out, whereAtomic(field, "LIKE", "%"+whereScalar(operand)+"%"))
		case "icontains":
			out = append(out, whereAtomic(field, "ILIKE", "%"+whereScalar(operand)+"%"))
		case "startswith":
			out = append(out, whereAtomic(field, "LIKE", whereScalar(operand)+"%"))
		case "in", "nin":
			list, err := whereList(field, op, operand)
			if err != nil {
				return nil, err
			}
			operator := "IN"
			if strings.ToLower(op) == "nin" {
				operator = "NOT IN"
			}
			out = append(out, whereAtomic(field, operator, strings.Join(list, ",")))
		case "between":
			list, err := whereList(field, op, operand)
			if err != nil {
				return nil, err
			}
			if len(list) != 2 {
				return nil, fmt.Errorf("where.%s.between needs exactly two values", field)
			}
			out = append(out, whereAtomic(field, "BETWEEN", strings.Join(list, ",")))
		case "isnull":
			isNull, ok := operand.(bool)
			if !ok {
				return nil, fmt.Errorf("where.%s.isnull must be true or false", field)
			}
			operator := "IS NULL"
			if !isNull {
				operator = "IS NOT NULL"
			}
			out = append(out, whereAtomic(field, operator, ""))
		default:
			return nil, fmt.Errorf("unsupported where operator %q for %s; use one of %s", op, field, strings.Join(WhereFilterOperators, ", "))
		}
	}
	return out, nil
}

func whereList(field, op string, operand any) ([]string, error) {
	switch typed := operand.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, whereScalar(item))
		}
		return out, nil
	case []string:
		return typed, nil
	default:
		return nil, fmt.Errorf("where.%s.%s must be a list", field, op)
	}
}

func whereAtomic(field, operator, value string) map[string]any {
	// ColumnType is required by the schema but the ontology resolvers resolve
	// the column from the property name, so a stable placeholder is sufficient.
	return map[string]any{"Type": "Atomic", "Atomic": map[string]any{"Key": field, "Operator": operator, "Value": value, "ColumnType": "String"}}
}

func whereScalar(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%v", typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}
