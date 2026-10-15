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

package clickhouse

import (
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
)

// enumValuePattern matches 'value' = N inside ClickHouse Enum definitions.
// Captures the quoted value (group 1).
var enumValuePattern = regexp.MustCompile(`'([^']*)'`)

// GetColumnConstraints overrides the base implementation to extract ClickHouse-specific
// constraints from system.columns, including Enum values and Decimal scale/precision.
func (p *ClickHousePlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	// Start with base GORM constraints
	constraints, err := p.GormPlugin.GetColumnConstraints(config, schema, storageUnit)
	if err != nil {
		constraints = make(map[string]map[string]any)
	}

	// Query system.columns for original type strings (case-preserved)
	_, err = plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		var columns []struct {
			Name              string `gorm:"column:name"`
			Type              string `gorm:"column:type"`
			DefaultExpression string `gorm:"column:default_expression"`
		}

		if err := db.Table("system.columns").
			Select("name, type, default_expression").
			Where("database = ? AND table = ?", schema, storageUnit).
			Scan(&columns).Error; err != nil {
			return false, err
		}

		for _, col := range columns {
			enrichConstraintsFromType(constraints, col.Name, col.Type)
			// GORM's migrator does not surface ClickHouse defaults, so take them from
			// system.columns; otherwise columns like `created_at DateTime DEFAULT now()`
			// are reported as required.
			if col.DefaultExpression != "" {
				colConstraints := gorm_plugin.EnsureConstraintEntry(constraints, col.Name)
				colConstraints["default"] = col.DefaultExpression
			} else if !strings.Contains(strings.ToUpper(col.Type), "NULLABLE(") {
				// ClickHouse fills omitted non-Nullable columns with the type's zero value
				// on INSERT, so they are never required even without an explicit DEFAULT.
				if implicit, ok := implicitClickHouseDefault(unwrapClickHouseModifiers(col.Type)); ok {
					colConstraints := gorm_plugin.EnsureConstraintEntry(constraints, col.Name)
					colConstraints["default"] = implicit
				}
			}
		}

		return true, nil
	})

	if err != nil {
		// Return base constraints if system.columns query fails
		return constraints, nil //nolint:nilerr
	}

	return constraints, nil
}

// enrichConstraintsFromType parses a ClickHouse type string and injects relevant
// constraints (check_values for enums, scale/precision for decimals).
func enrichConstraintsFromType(constraints map[string]map[string]any, columnName string, fullType string) {
	// ClickHouse's system.columns always wraps nullable types in Nullable(...),
	// so its absence is authoritative. Without this, complex types (Tuple, Array, Map)
	// default to nullable and mock data generation may produce nil for non-nullable columns.
	upperFull := strings.ToUpper(fullType)
	colConstraints := gorm_plugin.EnsureConstraintEntry(constraints, columnName)
	colConstraints["nullable"] = strings.Contains(upperFull, "NULLABLE(")

	// Unwrap Nullable/LowCardinality to find the inner type
	innerType := unwrapClickHouseModifiers(fullType)
	upperInner := strings.ToUpper(innerType)

	// Parse Enum8/Enum16 values
	if strings.HasPrefix(upperInner, "ENUM8(") || strings.HasPrefix(upperInner, "ENUM16(") {
		values := parseClickHouseEnumValues(innerType)
		if len(values) > 0 {
			colConstraints["check_values"] = values
		}
		return
	}

	// Parse Decimal scale/precision
	// Decimal32(S), Decimal64(S), Decimal128(S), Decimal256(S) — single param is scale
	// Decimal(P, S) — two params
	if strings.HasPrefix(upperInner, "DECIMAL") {
		scale, precision := parseClickHouseDecimalParams(innerType)
		if scale >= 0 || precision > 0 {
			if scale >= 0 {
				colConstraints["scale"] = scale
			}
			if precision > 0 {
				colConstraints["precision"] = precision
			}
		}
	}
}

// unwrapClickHouseModifiers strips Nullable(...) and LowCardinality(...) wrappers,
// preserving the original case of the inner type.
func unwrapClickHouseModifiers(typeName string) string {
	for {
		lower := strings.ToLower(strings.TrimSpace(typeName))
		unwrapped := false
		for _, prefix := range []string{"nullable(", "lowcardinality("} {
			if strings.HasPrefix(lower, prefix) && strings.HasSuffix(lower, ")") {
				typeName = strings.TrimSpace(typeName[len(prefix) : len(typeName)-1])
				unwrapped = true
				break
			}
		}
		if !unwrapped {
			return typeName
		}
	}
}

// parseClickHouseEnumValues extracts the string values from ClickHouse Enum definitions.
// Input: "Enum8('active' = 1, 'inactive' = 2)" -> ["active", "inactive"]
func parseClickHouseEnumValues(enumType string) []string {
	// Find content between the outer parentheses
	start := strings.Index(enumType, "(")
	end := strings.LastIndex(enumType, ")")
	if start == -1 || end == -1 || end <= start {
		return nil
	}

	content := enumType[start+1 : end]
	matches := enumValuePattern.FindAllStringSubmatch(content, -1)

	var values []string
	for _, match := range matches {
		if len(match) > 1 {
			if sanitized, ok := common.SanitizeConstraintValue(match[1]); ok {
				values = append(values, sanitized)
			}
		}
	}
	return values
}

// parseClickHouseDecimalParams extracts scale and precision from ClickHouse Decimal types.
// Decimal32(S) -> scale=S, precision=9
// Decimal64(S) -> scale=S, precision=18
// Decimal128(S) -> scale=S, precision=38
// Decimal256(S) -> scale=S, precision=76
// Decimal(P, S) -> scale=S, precision=P
func parseClickHouseDecimalParams(decimalType string) (scale int, precision int) {
	scale = -1 // -1 means not found

	start := strings.Index(decimalType, "(")
	end := strings.LastIndex(decimalType, ")")
	if start == -1 || end == -1 || end <= start {
		return scale, precision
	}

	baseName := strings.ToUpper(strings.TrimSpace(decimalType[:start]))
	content := decimalType[start+1 : end]
	parts := strings.Split(content, ",")

	if len(parts) == 1 {
		// Single param: Decimal32(S) — the param is scale
		if s, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
			scale = s
		}
		// Set max precision based on type width
		switch baseName {
		case "DECIMAL32":
			precision = 9
		case "DECIMAL64":
			precision = 18
		case "DECIMAL128":
			precision = 38
		case "DECIMAL256":
			precision = 76
		}
	} else if len(parts) == 2 {
		// Two params: Decimal(P, S)
		if p, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
			precision = p
		}
		if s, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			scale = s
		}
	}

	return scale, precision
}

// implicitClickHouseDefault returns the value ClickHouse inserts for an omitted
// non-Nullable column of the given (unwrapped) type, rendered as a SQL literal.
// It reports false for types whose implicit default is not known here.
func implicitClickHouseDefault(innerType string) (string, bool) {
	upper := strings.ToUpper(strings.TrimSpace(innerType))
	switch {
	case strings.HasPrefix(upper, "INT"), strings.HasPrefix(upper, "UINT"),
		strings.HasPrefix(upper, "FLOAT"), strings.HasPrefix(upper, "DECIMAL"),
		strings.HasPrefix(upper, "BFLOAT"):
		return "0", true
	case upper == "BOOL", upper == "BOOLEAN":
		return "false", true
	case upper == "STRING", strings.HasPrefix(upper, "FIXEDSTRING("):
		return "''", true
	case upper == "UUID":
		return "'00000000-0000-0000-0000-000000000000'", true
	case upper == "DATE", upper == "DATE32":
		return "'1970-01-01'", true
	case strings.HasPrefix(upper, "DATETIME"):
		return "'1970-01-01 00:00:00'", true
	case strings.HasPrefix(upper, "ARRAY("):
		return "[]", true
	case strings.HasPrefix(upper, "MAP("):
		return "{}", true
	case strings.HasPrefix(upper, "TUPLE("):
		return "()", true
	case strings.HasPrefix(upper, "ENUM8("), strings.HasPrefix(upper, "ENUM16("):
		values := parseClickHouseEnumValues(innerType)
		if len(values) == 0 {
			return "", false
		}
		return "'" + values[0] + "'", true
	case strings.HasPrefix(upper, "IPV4"):
		return "'0.0.0.0'", true
	case strings.HasPrefix(upper, "IPV6"):
		return "'::'", true
	}
	return "", false
}
