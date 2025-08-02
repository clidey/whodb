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
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/clidey/whodb/core/src/engine"
)

type Generator struct {
	faker *gofakeit.Faker
}

func NewGenerator() *Generator {
	return &Generator{
		faker: gofakeit.New(time.Now().UnixNano()),
	}
}

// GenerateValue generates a mock value based on column type and name
func (g *Generator) GenerateValue(columnName string, columnType string, constraints map[string]interface{}) (string, error) {
	// Handle NOT NULL constraint
	isNotNull := false
	if constraints != nil {
		if notNull, ok := constraints["not_null"].(bool); ok {
			isNotNull = notNull
		}
	}

	// Normalize column type to lowercase for comparison
	columnType = strings.ToLower(columnType)
	columnNameLower := strings.ToLower(columnName)

	// First, try to generate based on column name patterns
	if value := g.generateByColumnName(columnNameLower); value != "" {
		return value, nil
	}

	// Then, generate based on column type
	value, err := g.generateByColumnType(columnType)
	if err != nil {
		return "", err
	}

	// Handle NULL values if allowed
	if !isNotNull && g.faker.Bool() && g.faker.IntRange(1, 10) > 8 { // 20% chance of NULL
		return "", nil
	}

	return value, nil
}

// generateByColumnName generates values based on common column name patterns
func (g *Generator) generateByColumnName(columnName string) string {
	switch {
	case strings.Contains(columnName, "email"):
		return g.faker.Email()
	case strings.Contains(columnName, "first_name") || columnName == "firstname":
		return g.faker.FirstName()
	case strings.Contains(columnName, "last_name") || columnName == "lastname":
		return g.faker.LastName()
	case strings.Contains(columnName, "name") && !strings.Contains(columnName, "username"):
		return g.faker.Name()
	case strings.Contains(columnName, "username") || strings.Contains(columnName, "user_name"):
		return g.faker.Username()
	case strings.Contains(columnName, "phone"):
		return g.faker.Phone()
	case strings.Contains(columnName, "address"):
		return g.faker.Address().Address
	case strings.Contains(columnName, "city"):
		return g.faker.City()
	case strings.Contains(columnName, "state"):
		return g.faker.State()
	case strings.Contains(columnName, "country"):
		return g.faker.Country()
	case strings.Contains(columnName, "zip") || strings.Contains(columnName, "postal"):
		return g.faker.Zip()
	case strings.Contains(columnName, "company"):
		return g.faker.Company()
	case strings.Contains(columnName, "job_title") || strings.Contains(columnName, "jobtitle"):
		return g.faker.JobTitle()
	case strings.Contains(columnName, "description") || strings.Contains(columnName, "desc"):
		return g.faker.Paragraph(1, 3, 10, " ")
	case strings.Contains(columnName, "url") || strings.Contains(columnName, "website"):
		return g.faker.URL()
	case strings.Contains(columnName, "uuid"):
		return g.faker.UUID()
	case strings.Contains(columnName, "price") || strings.Contains(columnName, "amount") || strings.Contains(columnName, "cost"):
		return fmt.Sprintf("%.2f", g.faker.Price(10.0, 1000.0))
	case strings.Contains(columnName, "created_at") || strings.Contains(columnName, "updated_at") || strings.Contains(columnName, "date"):
		return g.faker.Date().Format(time.RFC3339)
	}
	return ""
}

// generateByColumnType generates values based on SQL column types
func (g *Generator) generateByColumnType(columnType string) (string, error) {
	switch {
	// Integer types
	case strings.Contains(columnType, "int") || strings.Contains(columnType, "serial"):
		if strings.Contains(columnType, "bigint") || strings.Contains(columnType, "int8") {
			return fmt.Sprintf("%d", g.faker.Int64()), nil
		}
		if strings.Contains(columnType, "smallint") || strings.Contains(columnType, "int2") {
			return fmt.Sprintf("%d", g.faker.Int16()), nil
		}
		return fmt.Sprintf("%d", g.faker.Int32()), nil

	// Decimal/Numeric types
	case strings.Contains(columnType, "decimal") || strings.Contains(columnType, "numeric") || 
		 strings.Contains(columnType, "real") || strings.Contains(columnType, "double") || 
		 strings.Contains(columnType, "float"):
		return fmt.Sprintf("%.2f", g.faker.Float64Range(0.0, 10000.0)), nil

	// Text/String types
	case strings.Contains(columnType, "varchar") || strings.Contains(columnType, "text") || 
		 strings.Contains(columnType, "char"):
		// Extract length if specified
		maxLen := 255
		if strings.Contains(columnType, "(") {
			// Try to extract length from varchar(n)
			start := strings.Index(columnType, "(")
			end := strings.Index(columnType, ")")
			if start != -1 && end != -1 && end > start {
				lengthStr := columnType[start+1 : end]
				if len, err := fmt.Sscanf(lengthStr, "%d", &maxLen); err == nil && len == 1 {
					// Successfully parsed length
				}
			}
		}
		text := g.faker.LetterN(uint(g.faker.IntRange(10, maxLen)))
		if len(text) > maxLen {
			text = text[:maxLen]
		}
		return text, nil

	// Boolean type
	case strings.Contains(columnType, "bool"):
		return fmt.Sprintf("%t", g.faker.Bool()), nil

	// Date/Time types
	case strings.Contains(columnType, "timestamp"):
		return g.faker.Date().Format(time.RFC3339), nil
	case strings.Contains(columnType, "date"):
		return g.faker.Date().Format("2006-01-02"), nil
	case strings.Contains(columnType, "time"):
		return g.faker.Date().Format("15:04:05"), nil

	// UUID type
	case strings.Contains(columnType, "uuid"):
		return g.faker.UUID(), nil

	// JSON/JSONB types
	case strings.Contains(columnType, "json"):
		data := map[string]interface{}{
			g.faker.Word(): g.faker.Word(),
			g.faker.Word(): g.faker.IntRange(1, 100),
		}
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil

	// Array types (PostgreSQL)
	case strings.Contains(columnType, "[]"):
		// Generate array with 1-5 elements
		baseType := strings.Replace(columnType, "[]", "", -1)
		arraySize := g.faker.IntRange(1, 5)
		elements := make([]string, arraySize)
		for i := 0; i < arraySize; i++ {
			elem, err := g.generateByColumnType(baseType)
			if err != nil {
				return "", err
			}
			elements[i] = elem
		}
		return "{" + strings.Join(elements, ",") + "}", nil

	// Default to string
	default:
		return g.faker.Word(), nil
	}
}

// GenerateRowData generates mock data for a complete row
func (g *Generator) GenerateRowData(columns []engine.Column) ([]engine.Record, error) {
	records := make([]engine.Record, 0, len(columns))
	
	for _, col := range columns {
		constraints := make(map[string]interface{})
		// TODO: Parse constraints from column metadata
		
		value, err := g.GenerateValue(col.Name, col.Type, constraints)
		if err != nil {
			return nil, fmt.Errorf("failed to generate value for column %s: %w", col.Name, err)
		}
		
		records = append(records, engine.Record{
			Key:   col.Name,
			Value: value,
		})
	}
	
	return records, nil
}