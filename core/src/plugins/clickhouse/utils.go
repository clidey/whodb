package clickhouse

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"strconv"
	"strings"
	"time"
)

func getColumnTypes(conn driver.Conn, schema string, table string) (map[string]string, error) {
	query := fmt.Sprintf(`
		SELECT 
			name,
			type
		FROM system.columns
		WHERE database = '%s' AND table = '%s'`,
		schema, table)

	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnTypes := make(map[string]string)
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, err
		}
		columnTypes[name] = dataType
	}

	return columnTypes, nil
}

func convertStringValue(value string, columnType string) (interface{}, error) {
	// Handle null values
	if value == "" || strings.ToLower(value) == "null" {
		return nil, nil
	}

	// Remove any outer Nullable() wrapper
	if strings.HasPrefix(columnType, "Nullable(") {
		columnType = strings.TrimPrefix(columnType, "Nullable(")
		columnType = strings.TrimSuffix(columnType, ")")
	}

	// Handle Array type
	if strings.HasPrefix(columnType, "Array(") {
		return convertArrayValue(value, columnType)
	}

	// Remove any LowCardinality() wrapper
	if strings.HasPrefix(columnType, "LowCardinality(") {
		columnType = strings.TrimPrefix(columnType, "LowCardinality(")
		columnType = strings.TrimSuffix(columnType, ")")
	}

	// Handle Enum type
	if strings.HasPrefix(columnType, "Enum") {
		// For enums, if it's a string value just pass it through
		// ClickHouse will handle the conversion internally
		return value, nil
	}

	// Handle basic types first
	switch {
	case strings.HasPrefix(columnType, "DateTime"):
		return parseDateTime(value)
	case columnType == "Date":
		return parseDate(value)
	}

	// Handle numeric + default
	switch {
	case strings.HasPrefix(columnType, "UInt"):
		bitSize := 64
		if len(columnType) > 4 {
			if size, err := strconv.Atoi(columnType[4:]); err == nil {
				bitSize = size
			}
		}
		val, err := strconv.ParseUint(value, 10, bitSize)
		if err != nil {
			return nil, fmt.Errorf("converting to %s: %w", columnType, err)
		}
		switch bitSize {
		case 8:
			return uint8(val), nil
		case 16:
			return uint16(val), nil
		case 32:
			return uint32(val), nil
		default:
			return val, nil
		}

	case strings.HasPrefix(columnType, "Int"):
		bitSize := 64
		if len(columnType) > 3 {
			if size, err := strconv.Atoi(columnType[3:]); err == nil {
				bitSize = size
			}
		}
		val, err := strconv.ParseInt(value, 10, bitSize)
		if err != nil {
			return nil, fmt.Errorf("converting to %s: %w", columnType, err)
		}
		switch bitSize {
		case 8:
			return int8(val), nil
		case 16:
			return int16(val), nil
		case 32:
			return int32(val), nil
		default:
			return val, nil
		}

	case strings.HasPrefix(columnType, "Float"):
		bitSize := 64
		if strings.HasSuffix(columnType, "32") {
			bitSize = 32
		}
		val, err := strconv.ParseFloat(value, bitSize)
		if err != nil {
			return nil, fmt.Errorf("converting to %s: %w", columnType, err)
		}
		if bitSize == 32 {
			return float32(val), nil
		}
		return val, nil

	default:
		// For any other types, pass through as string
		return value, nil
	}
}

func parseDateTime(value string) (time.Time, error) {
	// List of formats to try
	formats := []string{
		time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05Z", // UTC timezone
		"2006-01-02 15:04:05",  // No timezone
		"2006-01-02T15:04:05",  // No timezone with T
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, value)
		if err == nil {
			// Convert to UTC if it has a timezone
			return t.UTC(), nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("could not parse datetime '%s': %v", value, lastErr)
}

// parseDate converts a string to a time.Time object for ClickHouse Date
func parseDate(value string) (time.Time, error) {
	formats := []string{
		"2006-01-02", // Standard date format
		time.RFC3339, // Try full datetime format and truncate to date
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, value)
		if err == nil {
			// Truncate to date only (no time component)
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("could not parse date '%s': %v", value, lastErr)
}

// Helper function to format a time.Time as a ClickHouse DateTime string
func formatDateTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}

// Helper function to format a time.Time as a ClickHouse Date string
func formatDate(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

func convertArrayValue(value string, columnType string) (interface{}, error) {
	// Extract the element type from Array(Type)
	elementType := strings.TrimPrefix(columnType, "Array(")
	elementType = strings.TrimSuffix(elementType, ")")

	// Remove brackets and split by comma
	value = strings.Trim(value, "[]")
	if value == "" {
		return []interface{}{}, nil
	}

	elements := strings.Split(value, ",")
	result := make([]interface{}, 0, len(elements))

	for _, element := range elements {
		element = strings.TrimSpace(element)
		if element == "" {
			continue
		}

		converted, err := convertStringValue(element, elementType)
		if err != nil {
			return nil, fmt.Errorf("converting array element: %w", err)
		}
		result = append(result, converted)
	}

	return result, nil
}

func getPrimaryKeyColumns(conn driver.Conn, schema, table string) ([]string, error) {
	query := `
		SELECT name
		FROM system.columns
		WHERE database = ? AND table = ? AND is_in_primary_key = 1
	`

	rows, err := conn.Query(context.Background(), query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, column)
	}

	return primaryKeys, nil
}

func isPrimaryKey(column string, primaryKeys []string) bool {
	for _, pk := range primaryKeys {
		if column == pk {
			return true
		}
	}
	return false
}
