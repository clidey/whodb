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

package mockdata

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

const (
	// Default configuration values
	DefaultNullProbability = 0.2     // 20% chance of null for nullable columns
	DefaultStringMinLen    = 10      // Minimum string length
	DefaultStringMaxLen    = 255     // Maximum string length
	DefaultFloatMin        = 0.0     // Minimum float value
	DefaultFloatMax        = 10000.0 // Maximum float value
)

// Use shared database type sets
var (
	intTypes      = common.IntTypes
	uintTypes     = common.UintTypes
	floatTypes    = common.FloatTypes
	boolTypes     = common.BoolTypes
	dateTypes     = common.DateTypes
	dateTimeTypes = common.DateTimeTypes
	uuidTypes     = common.UuidTypes
	textTypes     = common.TextTypes
	jsonTypes     = common.JsonTypes
)

type Generator struct {
	faker *gofakeit.Faker
}

func NewGenerator() *Generator {
	return &Generator{
		faker: gofakeit.New(uint64(time.Now().UnixNano())),
	}
}

// detectDatabaseType returns the simplified database type for a column
func detectDatabaseType(columnType string) string {
	upperType := strings.ToUpper(columnType)

	// Handle PostgreSQL arrays first
	if strings.Contains(upperType, "[]") {
		return "array"
	}

	// Remove size specifiers like VARCHAR(255) -> VARCHAR
	if idx := strings.Index(upperType, "("); idx > 0 {
		upperType = upperType[:idx]
	}
	upperType = strings.TrimSpace(upperType)

	switch {
	case intTypes.Contains(upperType):
		return "int"
	case uintTypes.Contains(upperType):
		return "uint"
	case floatTypes.Contains(upperType):
		return "float"
	case boolTypes.Contains(upperType):
		return "bool"
	case dateTypes.Contains(upperType):
		return "date"
	case dateTimeTypes.Contains(upperType):
		return "datetime"
	case uuidTypes.Contains(upperType):
		return "uuid"
	case jsonTypes.Contains(upperType):
		return "json"
	case textTypes.Contains(upperType):
		return "text"
	default:
		return "text" // Default to text for unknown types
	}
}

// parseMaxLen extracts the max length from types like varchar(n)
func parseMaxLen(columnType string) int {
	if strings.Contains(columnType, "(") {
		start := strings.Index(columnType, "(")
		end := strings.Index(columnType, ")")
		if start != -1 && end != -1 && end > start+1 {
			var n int
			if _, err := fmt.Sscanf(columnType[start+1:end], "%d", &n); err == nil {
				return n
			}
		}
	}
	return 0
}

// generateByType generates mock data based on the detected database type
// Returns properly typed values that the database driver can handle
func (g *Generator) generateByType(dbType string, columnType string, constraints map[string]any) any {
	switch dbType {
	case "int":
		// Use Go's explicit int types for SQL integer types
		lowerType := strings.ToLower(columnType)
		switch {
		case strings.Contains(lowerType, "tinyint"):
			return g.faker.Int8()
		case strings.Contains(lowerType, "smallint"):
			return g.faker.Int16()
		case strings.Contains(lowerType, "bigint"):
			return g.faker.Int64()
		case strings.Contains(lowerType, "int32"):
			return g.faker.Int32()
		case strings.Contains(lowerType, "int16"):
			return g.faker.Int16()
		case strings.Contains(lowerType, "int8"):
			return g.faker.Int8()
		default:
			return int(g.faker.Int8())
		}
	case "uint":
		// Use Go's explicit uint types for SQL unsigned integer types
		lowerType := strings.ToLower(columnType)
		switch {
		case strings.Contains(lowerType, "tinyint unsigned") || strings.Contains(lowerType, "uint8"):
			return g.faker.Uint8()
		case strings.Contains(lowerType, "smallint unsigned") || strings.Contains(lowerType, "uint16"):
			return g.faker.Uint16()
		case strings.Contains(lowerType, "bigint unsigned") || strings.Contains(lowerType, "uint64"):
			return g.faker.Uint64()
		case strings.Contains(lowerType, "uint32"):
			return g.faker.Uint32()
		case strings.Contains(lowerType, "uint16"):
			return g.faker.Uint16()
		case strings.Contains(lowerType, "uint8"):
			return g.faker.Uint8()
		default:
			return uint(g.faker.Uint8())
		}
	case "float":
		return g.faker.Float32Range(DefaultFloatMin, DefaultFloatMax)
	case "bool":
		return g.faker.Bool()
	case "date":
		// Generate dates within a reasonable range (last 10 years to avoid timezone issues)
		now := time.Now()
		tenYearsAgo := now.AddDate(-10, 0, 0)
		return g.faker.DateRange(tenYearsAgo, now)
	case "datetime":
		// Generate datetimes within a reasonable range (last 10 years to avoid timezone issues)
		now := time.Now()
		tenYearsAgo := now.AddDate(-10, 0, 0)
		return g.faker.DateRange(tenYearsAgo, now)
	case "uuid":
		return g.faker.UUID()
	case "json":
		data := map[string]any{
			g.faker.Word(): g.faker.Word(),
			g.faker.Word(): g.faker.IntRange(1, 100),
		}
		jsonBytes, _ := json.Marshal(data)
		return string(jsonBytes)
	case "array":
		// PostgreSQL array format - needs to be string representation
		baseType := strings.ReplaceAll(columnType, "[]", "")
		arraySize := g.faker.IntRange(1, 5)
		elements := make([]string, arraySize)
		for i := range arraySize {
			val := g.generateByType(detectDatabaseType(baseType), baseType, constraints)
			elements[i] = fmt.Sprintf("%v", val)
		}
		return "{" + strings.Join(elements, ",") + "}"
	case "text":
		fallthrough
	default:
		// Check for IN constraint values
		if constraints != nil {
			if values, ok := constraints["check_values"].([]string); ok && len(values) > 0 {
				// Pick a random value from the allowed values
				return values[g.faker.IntRange(0, len(values)-1)]
			}
		}

		maxLen := parseMaxLen(columnType)
		if maxLen <= 0 {
			maxLen = g.faker.IntRange(DefaultStringMinLen, DefaultStringMaxLen)
		}
		text := g.faker.LoremIpsumSentence(g.faker.IntRange(1, 10))
		if len(text) > maxLen {
			text = text[:maxLen]
		}
		return text
	}
}

// GenerateValue generates a mock value based on column type only
// Returns properly typed values that the database driver can handle
func (g *Generator) GenerateValue(columnName string, columnType string, constraints map[string]any) (any, error) {
	columnTypeLower := strings.ToLower(columnType)
	log.Logger.WithField("column", columnName).WithField("type", columnType).Debug("Generating value for column")

	// Check constraints
	allowNull := false
	requireUnique := false
	if constraints != nil {
		if nullable, ok := constraints["nullable"]; ok {
			allowNull = nullable.(bool)
		}
		if unique, ok := constraints["unique"]; ok {
			requireUnique = unique.(bool)
		}
	}

	// Generate NULL for nullable columns with configured probability
	if allowNull && g.faker.Float64() < DefaultNullProbability {
		return nil, nil
	}

	// Generate value based on database type and constraints
	dbType := detectDatabaseType(columnTypeLower)
	log.Logger.WithField("column", columnName).WithField("dbType", dbType).Debug("Detected database type")
	value := g.generateByType(dbType, columnTypeLower, constraints)
	log.Logger.WithField("column", columnName).WithField("value", value).Debug("Generated value")

	// For columns that require uniqueness, use inherently unique generators
	if requireUnique {
		// For unique columns, prefer UUIDs or timestamp-based values
		switch dbType {
		case "text", "uuid":
			value = g.faker.UUID()
		case "int", "uint":
			// Use timestamp + random for unique integers
			value = int32(time.Now().UnixNano()/1000 + int64(g.faker.IntRange(0, 9999)))
		}
		// For other types, rely on random generation being unlikely to collide
	}

	return value, nil
}

// GenerateRowDataWithConstraints generates mock data for a complete row
func (g *Generator) GenerateRowDataWithConstraints(columns []engine.Column, colConstraints map[string]map[string]any) ([]engine.Record, error) {
	log.Logger.WithField("columnCount", len(columns)).Debug("Starting row data generation")

	records := make([]engine.Record, 0, len(columns))

	for _, col := range columns {
		// Skip serial columns - database generates these
		if strings.Contains(strings.ToLower(col.Type), "serial") {
			continue
		}

		constraints := make(map[string]any)
		if colConstraints != nil {
			if c, ok := colConstraints[col.Name]; ok {
				constraints = c
			}
		}

		value, err := g.GenerateValue(col.Name, col.Type, constraints)
		if err != nil {
			log.Logger.WithError(err).WithField("column", col.Name).WithField("type", col.Type).Error("Failed to generate value for column")
			return nil, fmt.Errorf("failed to generate value for column %s: %w", col.Name, err)
		}

		// TODO: Refactor engine.Record to support interface{}/any values instead of strings.
		// This would allow us to pass typed values directly to database plugins,
		// letting each plugin handle formatting according to its specific requirements.
		// Current approach requires converting typed values to strings here, which
		// defeats the purpose of returning typed values from generateByType().

		// Convert typed value to string for the Record
		var valueStr string
		extra := map[string]string{
			"Type": col.Type,
		}

		if value == nil {
			// Mark as NULL in Extra field, leave Value empty
			valueStr = ""
			extra["IsNull"] = "true"
		} else if t, ok := value.(time.Time); ok {
			// Format time values for MySQL compatibility
			if strings.Contains(strings.ToLower(col.Type), "date") && !strings.Contains(strings.ToLower(col.Type), "time") {
				valueStr = t.Format("2006-01-02")
			} else {
				// MySQL datetime format without timezone
				valueStr = t.Format("2006-01-02 15:04:05")
			}
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}

		records = append(records, engine.Record{
			Key:   col.Name,
			Value: valueStr,
			Extra: extra,
		})
	}

	return records, nil
}

// GenerateRowData generates mock data without constraints
func (g *Generator) GenerateRowData(columns []engine.Column) ([]engine.Record, error) {
	return g.GenerateRowDataWithConstraints(columns, nil)
}

// GenerateRowWithDefaults generates a row using safe default values
// This is used as a fallback when constraint violations occur
func (g *Generator) GenerateRowWithDefaults(columns []engine.Column) []engine.Record {
	log.Logger.WithField("columnCount", len(columns)).Info("Generating row with defaults")
	records := make([]engine.Record, 0, len(columns))

	for _, col := range columns {
		// Skip serial/auto-increment columns
		if strings.Contains(strings.ToLower(col.Type), "serial") {
			log.Logger.WithField("column", col.Name).WithField("type", col.Type).Debug("Skipping serial column")
			continue
		}

		colType := strings.ToLower(col.Type)
		var valueStr string
		extra := map[string]string{
			"Type": col.Type,
		}

		log.Logger.WithField("column", col.Name).WithField("type", col.Type).WithField("lowerType", colType).Debug("Processing column for default value")

		// Use safe defaults based on column type
		switch {
		case strings.Contains(colType, "int"):
			valueStr = "0"
		case strings.Contains(colType, "float") || strings.Contains(colType, "decimal") ||
			strings.Contains(colType, "numeric") || strings.Contains(colType, "number") ||
			strings.Contains(colType, "money") || strings.Contains(colType, "real") ||
			strings.Contains(colType, "double"):
			valueStr = "0.0"
		case strings.Contains(colType, "bool") || strings.Contains(colType, "bit"):
			valueStr = "false"
		case strings.Contains(colType, "date"):
			valueStr = time.Now().Format("2006-01-02")
		case strings.Contains(colType, "time"):
			valueStr = time.Now().Format("2006-01-02 15:04:05")
		case strings.Contains(colType, "uuid") || strings.Contains(colType, "uniqueidentifier"):
			valueStr = g.faker.UUID()
		case strings.Contains(colType, "json") || strings.Contains(colType, "jsonb"):
			valueStr = "{}"
		case strings.Contains(colType, "xml"):
			valueStr = "<root></root>"
		case strings.Contains(colType, "blob") || strings.Contains(colType, "bytea") ||
			strings.Contains(colType, "binary") || strings.Contains(colType, "varbinary") ||
			strings.Contains(colType, "image"):
			// Return hex representation of empty bytes for binary types
			valueStr = "\\x00"
		case strings.Contains(colType, "array"):
			valueStr = "{}"
		case strings.Contains(colType, "enum"):
			// Default enum value todo: make this actually work for enum as the db will likely enforce it to be one of the possible values
			valueStr = "value1"
		case strings.Contains(colType, "inet") || strings.Contains(colType, "cidr"):
			valueStr = "192.168.1.1"
		case strings.Contains(colType, "ipv4"):
			valueStr = "192.168.1.1"
		case strings.Contains(colType, "ipv6"):
			valueStr = "::1"
		case strings.Contains(colType, "geometry") || strings.Contains(colType, "geography") ||
			strings.Contains(colType, "point"):
			// Basic WKT format for a point
			valueStr = "POINT(0 0)"
		case strings.Contains(colType, "text") || strings.Contains(colType, "char") ||
			strings.Contains(colType, "varchar") || strings.Contains(colType, "string") ||
			strings.Contains(colType, "clob") || strings.Contains(colType, "long"):
			// For CHAR(n) or VARCHAR(n) types, try to respect the length constraint
			// This handles CHAR(2), VARCHAR(255), CHARACTER(10), etc.
			if strings.Contains(colType, "(") && strings.Contains(colType, ")") {
				// Extract the length from type(n)
				start := strings.Index(colType, "(")
				end := strings.Index(colType, ")")
				if start > -1 && end > start {
					lengthStr := colType[start+1 : end]
					// Handle potential comma (e.g., DECIMAL(10,2))
					if commaIdx := strings.Index(lengthStr, ","); commaIdx > -1 {
						lengthStr = lengthStr[:commaIdx]
					}
					lengthStr = strings.TrimSpace(lengthStr)
					if length, err := strconv.Atoi(lengthStr); err == nil && length > 0 {
						// Generate a random string of the appropriate length
						if length == 1 {
							valueStr = g.faker.Letter()
						} else if length == 2 {
							// For 2-char fields like country codes, use uppercase letters
							valueStr = g.faker.LetterN(2)
							valueStr = strings.ToUpper(valueStr)
						} else if length <= 10 {
							// For short strings, use random letters
							valueStr = g.faker.LetterN(uint(length))
						} else {
							// For longer strings, use Lorem Ipsum text
							valueStr = g.faker.LoremIpsumWord()
							if len(valueStr) > length {
								valueStr = valueStr[:length]
							} else {
								// If word is too short, pad with more text
								for len(valueStr) < length {
									valueStr += g.faker.Letter()
								}
							}
						}
					} else {
						// If we can't parse the length, use a random word
						valueStr = g.faker.LoremIpsumWord()
					}
				} else {
					// No length specified, use a random word
					valueStr = g.faker.LoremIpsumWord()
				}
			} else {
				// No parentheses, use a random word
				valueStr = g.faker.LoremIpsumWord()
			}
		default:
			// For any unknown type, use a random word
			valueStr = g.faker.LoremIpsumWord()
			log.Logger.WithField("column", col.Name).WithField("type", col.Type).Warn("Using random word for unknown column type")
		}

		log.Logger.WithField("column", col.Name).WithField("value", valueStr).WithField("type", col.Type).Info("Generated default value for column")

		records = append(records, engine.Record{
			Key:   col.Name,
			Value: valueStr,
			Extra: extra,
		})
	}

	log.Logger.WithField("recordCount", len(records)).Info("Completed generating row with defaults")

	return records
}
