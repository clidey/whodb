package sqlite3

import (
	"path/filepath"
	"strings"
)

func isValidDatabaseFileName(fileName string) bool {
	cleanedPath := filepath.Clean(fileName)
	if strings.Contains(cleanedPath, "..") || filepath.IsAbs(cleanedPath) {
		return false
	}
	if cleanedPath != fileName {
		return false
	}
	return true
}
