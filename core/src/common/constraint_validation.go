package common

import (
	"regexp"
	"strings"
)

// ValidateColumnName validates that a column name is safe and doesn't contain SQL injection attempts
func ValidateColumnName(columnName string) bool {
	// Column names should only contain alphanumeric characters, underscores, and not start with a number
	validColumnPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	
	// Check for basic pattern
	if !validColumnPattern.MatchString(columnName) {
		return false
	}
	
	// Check for SQL keywords that should never appear in column names
	sqlKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER",
		"EXEC", "EXECUTE", "UNION", "GRANT", "REVOKE", "TRUNCATE",
		"MERGE", "CALL", "EXPLAIN", "LOCK", "COMMENT", "COMMIT", "ROLLBACK",
		"SAVEPOINT", "SET", "SHOW", "--", "/*", "*/", ";",
	}
	
	upperColumn := strings.ToUpper(columnName)
	for _, keyword := range sqlKeywords {
		if strings.Contains(upperColumn, keyword) {
			return false
		}
	}
	
	// Reasonable length limit for column names
	if len(columnName) > 128 {
		return false
	}
	
	return true
}

// SanitizeConstraintValue sanitizes a single constraint value to prevent SQL injection
func SanitizeConstraintValue(value string) (string, bool) {
	// Remove any SQL comment indicators
	if strings.Contains(value, "--") || strings.Contains(value, "/*") || strings.Contains(value, "*/") {
		return "", false
	}
	
	// Remove any semicolons which could be used to terminate statements
	if strings.Contains(value, ";") {
		return "", false
	}
	
	// Check for dangerous SQL keywords
	dangerousKeywords := []string{
		"DROP", "DELETE", "TRUNCATE", "EXEC", "EXECUTE",
		"CREATE", "ALTER", "GRANT", "REVOKE",
		"UNION", "INSERT", "UPDATE",
	}
	
	upperValue := strings.ToUpper(value)
	for _, keyword := range dangerousKeywords {
		// Check for whole word matches to avoid false positives
		if regexp.MustCompile(`\b` + keyword + `\b`).MatchString(upperValue) {
			return "", false
		}
	}
	
	// Limit value length to prevent buffer overflow attacks
	if len(value) > 1000 {
		return "", false
	}
	
	return value, true
}

// ValidateConstraintValues validates a slice of constraint values
func ValidateConstraintValues(values []string) ([]string, bool) {
	validatedValues := make([]string, 0, len(values))
	
	for _, value := range values {
		sanitized, ok := SanitizeConstraintValue(value)
		if !ok {
			return nil, false
		}
		validatedValues = append(validatedValues, sanitized)
	}
	
	return validatedValues, true
}