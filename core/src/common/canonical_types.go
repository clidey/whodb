/*
 * Copyright 2025 Clidey, Inc.
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

package common

import (
	"regexp"
	"strconv"
	"strings"
)

// TypeSpec represents a parsed type specification with its components.
type TypeSpec struct {
	// BaseType is the type name without length/precision (UPPERCASE).
	BaseType string

	// Length is the length parameter for types like VARCHAR(255).
	// Zero if not specified or not applicable. In the example above, the length is 255.
	Length int

	// Precision is the precision for types like DECIMAL(10,2).
	// Zero if not specified or not applicable. In the example above, the precision is 10.
	Precision int

	// Scale is the scale for types like DECIMAL(10,2).
	// Zero if not specified or not applicable. In the example above, the scale is 2.
	Scale int
}

// typeSpecRegex matches type specifications like "VARCHAR(255)" or "DECIMAL(10,2)"
var typeSpecRegex = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_ ]*?)(?:\((\d+)(?:,\s*(\d+))?\))?$`)

// ParseTypeSpec parses a full type string into its components.
// Examples:
//   - "VARCHAR(255)" -> BaseType="VARCHAR", Length=255
//   - "DECIMAL(10,2)" -> BaseType="DECIMAL", Precision=10, Scale=2
//   - "INTEGER" -> BaseType="INTEGER"
func ParseTypeSpec(fullType string) TypeSpec {
	fullType = strings.TrimSpace(fullType)

	matches := typeSpecRegex.FindStringSubmatch(fullType)
	if matches == nil {
		return TypeSpec{BaseType: strings.ToUpper(fullType)}
	}

	spec := TypeSpec{
		BaseType: strings.ToUpper(strings.TrimSpace(matches[1])),
	}

	// Parse first number (length or precision)
	if matches[2] != "" {
		if n, err := strconv.Atoi(matches[2]); err == nil {
			if matches[3] != "" {
				spec.Precision = n
			} else {
				spec.Length = n
			}
		}
	}

	// Parse second number (scale)
	if matches[3] != "" {
		if n, err := strconv.Atoi(matches[3]); err == nil {
			spec.Scale = n
			if spec.Length > 0 {
				spec.Precision = spec.Length
				spec.Length = 0
			}
		}
	}

	return spec
}

// FormatTypeSpec formats a TypeSpec back into a type string.
func FormatTypeSpec(spec TypeSpec) string {
	if spec.Precision > 0 {
		return spec.BaseType + "(" +
			strconv.Itoa(spec.Precision) + "," +
			strconv.Itoa(spec.Scale) + ")"
	}

	if spec.Length > 0 {
		return spec.BaseType + "(" + strconv.Itoa(spec.Length) + ")"
	}

	return spec.BaseType
}

// NormalizeTypeWithMap normalizes a type name using the provided alias map.
// The alias map should map uppercase aliases to uppercase canonical names.
func NormalizeTypeWithMap(typeName string, aliasMap map[string]string) string {
	spec := ParseTypeSpec(typeName)
	upperBase := strings.ToUpper(spec.BaseType)

	if canonical, ok := aliasMap[upperBase]; ok {
		spec.BaseType = canonical
	} else {
		spec.BaseType = upperBase
	}

	return FormatTypeSpec(spec)
}
