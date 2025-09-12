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

package types

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
)

// InitializeDefaultTypes registers all default type definitions
func InitializeDefaultTypes(registry *TypeRegistry) {
	// Integer types
	registry.RegisterType(&TypeDefinition{
		Name:     "Integer",
		Category: TypeCategoryNumeric,
		SQLTypes: []string{
			"INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT", "MEDIUMINT",
			"INT2", "INT4", "INT8", "INT16", "INT32", "INT64",
		},
		FromString: func(s string) (any, error) {
			return cast.ToInt64E(s)
		},
		ToString: func(v any) (string, error) {
			return cast.ToStringE(v)
		},
		InputType: "number",
		Icon:      "number",
	})

	// Unsigned integer types
	registry.RegisterType(&TypeDefinition{
		Name:     "UnsignedInteger",
		Category: TypeCategoryNumeric,
		SQLTypes: []string{
			"UINT", "UINT8", "UINT16", "UINT32", "UINT64",
			"TINYINT UNSIGNED", "SMALLINT UNSIGNED", "MEDIUMINT UNSIGNED", "BIGINT UNSIGNED",
		},
		FromString: func(s string) (any, error) {
			return cast.ToUint64E(s)
		},
		ToString: func(v any) (string, error) {
			return cast.ToStringE(v)
		},
		InputType: "number",
		Icon:      "number",
	})

	// Float types
	registry.RegisterType(&TypeDefinition{
		Name:     "Float",
		Category: TypeCategoryNumeric,
		SQLTypes: []string{
			"REAL", "FLOAT", "DOUBLE", "DOUBLE PRECISION", "NUMBER",
		},
		FromString: func(s string) (any, error) {
			return cast.ToFloat64E(s)
		},
		ToString: func(v any) (string, error) {
			return cast.ToStringE(v)
		},
		InputType: "number",
		Icon:      "decimal",
	})

	// Decimal types
	registry.RegisterType(&TypeDefinition{
		Name:     "Decimal",
		Category: TypeCategoryNumeric,
		SQLTypes: []string{
			"DECIMAL", "NUMERIC", "MONEY", "SMALLMONEY",
		},
		FromString: func(s string) (any, error) {
			d, err := decimal.NewFromString(s)
			if err != nil {
				return nil, err
			}
			return d, nil
		},
		ToString: func(v any) (string, error) {
			switch val := v.(type) {
			case decimal.Decimal:
				return val.String(), nil
			default:
				return cast.ToStringE(v)
			}
		},
		InputType: "number",
		Icon:      "decimal",
	})

	// Boolean types
	registry.RegisterType(&TypeDefinition{
		Name:     "Boolean",
		Category: TypeCategoryBoolean,
		SQLTypes: []string{
			"BOOLEAN", "BOOL", "BIT",
		},
		FromString: func(s string) (any, error) {
			return cast.ToBoolE(s)
		},
		ToString: func(v any) (string, error) {
			return cast.ToStringE(v)
		},
		Validator: func(s string) error {
			_, err := cast.ToBoolE(s)
			return err
		},
		InputType: "checkbox",
		Icon:      "check",
	})

	// Date types
	registry.RegisterType(&TypeDefinition{
		Name:     "Date",
		Category: TypeCategoryDate,
		SQLTypes: []string{
			"DATE",
		},
		FromString: func(s string) (any, error) {
			formats := []string{
				"2006-01-02",
				time.RFC3339,
				"2006-01-02T15:04:05",
				"2006-01-02 15:04:05",
			}

			var lastErr error
			for _, format := range formats {
				t, err := time.Parse(format, s)
				if err == nil {
					// Truncate to date only
					return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
				}
				lastErr = err
			}
			return nil, fmt.Errorf("could not parse date '%s': %v", s, lastErr)
		},
		ToString: func(v any) (string, error) {
			t, err := cast.ToTimeE(v)
			if err != nil {
				return "", err
			}
			return t.Format("2006-01-02"), nil
		},
		InputType: "date",
		Icon:      "calendar",
	})

	// DateTime types
	registry.RegisterType(&TypeDefinition{
		Name:     "DateTime",
		Category: TypeCategoryDate,
		SQLTypes: []string{
			"DATETIME", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE",
			"DATETIME2", "SMALLDATETIME", "TIMETZ", "TIMESTAMPTZ",
		},
		FromString: func(s string) (any, error) {
			formats := []string{
				time.RFC3339,
				"2006-01-02T15:04:05Z",
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05",
			}

			var lastErr error
			for _, format := range formats {
				t, err := time.Parse(format, s)
				if err == nil {
					return t, nil
				}
				lastErr = err
			}
			return nil, fmt.Errorf("could not parse datetime '%s': %v", s, lastErr)
		},
		ToString: func(v any) (string, error) {
			t, err := cast.ToTimeE(v)
			if err != nil {
				return "", err
			}
			// Check if time component is zero
			if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
				return t.Format("2006-01-02"), nil
			}
			return t.Format("2006-01-02T15:04:05"), nil
		},
		InputType: "datetime-local",
		Icon:      "clock",
	})

	// Text types
	registry.RegisterType(&TypeDefinition{
		Name:     "Text",
		Category: TypeCategoryText,
		SQLTypes: []string{
			"TEXT", "STRING", "VARCHAR", "CHAR", "NCHAR", "NVARCHAR", "NTEXT",
			"TINYTEXT", "MEDIUMTEXT", "LONGTEXT", "CLOB",
		},
		FromString: func(s string) (any, error) {
			return s, nil
		},
		ToString: func(v any) (string, error) {
			return cast.ToStringE(v)
		},
		InputType: "text",
		Icon:      "text",
	})

	// UUID types
	registry.RegisterType(&TypeDefinition{
		Name:     "UUID",
		Category: TypeCategoryText,
		SQLTypes: []string{
			"UUID", "UNIQUEIDENTIFIER", "GUID",
		},
		FromString: func(s string) (any, error) {
			// Validate UUID format
			_, err := uuid.Parse(s)
			if err != nil {
				return nil, fmt.Errorf("invalid UUID format: %v", err)
			}
			return s, nil
		},
		ToString: func(v any) (string, error) {
			str := cast.ToString(v)
			// Ensure uppercase for consistency (e.g., SQL Server)
			return strings.ToUpper(str), nil
		},
		Validator: func(s string) error {
			_, err := uuid.Parse(s)
			return err
		},
		InputType: "text",
		Icon:      "key",
	})

	// Binary types
	registry.RegisterType(&TypeDefinition{
		Name:     "Binary",
		Category: TypeCategoryBinary,
		SQLTypes: []string{
			"BLOB", "BYTEA", "VARBINARY", "BINARY", "IMAGE",
			"TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
		},
		FromString: func(s string) (any, error) {
			// Convert string to byte array
			return []byte(s), nil
		},
		ToString: func(v any) (string, error) {
			switch val := v.(type) {
			case []byte:
				if len(val) == 0 {
					return "", nil
				}
				// Return as string for display
				return string(val), nil
			default:
				return cast.ToStringE(v)
			}
		},
		InputType: "text",
		Icon:      "binary",
	})

	// JSON types
	registry.RegisterType(&TypeDefinition{
		Name:     "JSON",
		Category: TypeCategoryJSON,
		SQLTypes: []string{
			"JSON", "JSONB",
		},
		FromString: func(s string) (any, error) {
			// Keep as string but could validate JSON format
			return s, nil
		},
		ToString: func(v any) (string, error) {
			return cast.ToStringE(v)
		},
		InputType: "text",
		Icon:      "json",
	})

	// Geometry types
	registry.RegisterType(&TypeDefinition{
		Name:     "Geometry",
		Category: TypeCategoryGeometry,
		SQLTypes: []string{
			"GEOMETRY", "GEOGRAPHY", "POINT", "LINESTRING", "POLYGON",
			"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON",
		},
		FromString: func(s string) (any, error) {
			// Keep as string for now - could parse WKT/WKB in future
			return s, nil
		},
		ToString: func(v any) (string, error) {
			return cast.ToStringE(v)
		},
		InputType: "text",
		Icon:      "map",
	})
}
