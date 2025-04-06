package common

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
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

func IsRunningInsideDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return !os.IsNotExist(err)
}

func FilterList[T any](items []T, by func(input T) bool) []T {
	filteredItems := []T{}
	for _, item := range items {
		if by(item) {
			filteredItems = append(filteredItems, item)
		}
	}
	return filteredItems
}

func OpenBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	default:
		log.Logger.Warnf("Unsupported platform. Please open the URL manually: %s\n", url)
	}
	if err != nil {
		log.Logger.Warnf("Failed to open browser: %v\n", err)
	}
}

func StrPtrToBool(s *string) bool {
	if s == nil {
		return false
	}
	value := strings.ToLower(*s)
	return value == "true"
}
