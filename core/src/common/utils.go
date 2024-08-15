package common

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

func ContainsString(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}

func GetRecordValueOrDefault(records []engine.Record, key string, defaultValue string) string {
	for _, record := range records {
		if record.Key == key && len(record.Value) > 0 {
			return record.Value
		}
	}
	return defaultValue
}

func JoinWithQuotes(arr []string) string {
	quotedStrings := make([]string, len(arr))

	for i, str := range arr {
		quotedStrings[i] = fmt.Sprintf("\"%s\"", str)
	}

	return strings.Join(quotedStrings, ", ")
}
