package common

import (
	"regexp"
)

func IsValidSQLTableName(tableName string) bool {
	const pattern = `^[a-zA-Z0-9_]+$`
	matched, _ := regexp.MatchString(pattern, tableName)
	return matched
}
