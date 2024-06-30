package common

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

func IsValidSQLTableName(tableName string) bool {
	const pattern = `^[a-zA-Z0-9_]+$`
	matched, _ := regexp.MatchString(pattern, tableName)
	return matched
}

func IsValidMongoCollectionName(name string) error {
	if len(name) == 0 {
		return errors.New("collection name cannot be an empty string")
	}
	if utf8.RuneCountInString(name) > 120 {
		return errors.New("collection name cannot be longer than 120 bytes")
	}
	if strings.Contains(name, "\x00") {
		return errors.New("collection name cannot contain null character")
	}
	if strings.HasPrefix(name, "system.") {
		return errors.New("collection name cannot start with 'system.'")
	}
	return nil
}
