/*
 * Copyright 2026 Clidey, Inc.
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

package common

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// ContainsString checks if a string slice contains a specific element.
func ContainsString(slice []string, element string) bool {
	return slices.Contains(slice, element)
}

// GetRecordValueOrDefault searches for a key in a slice of Records and returns its value,
// or the provided default if the key is not found or has an empty value.
func GetRecordValueOrDefault(records []engine.Record, key string, defaultValue string) string {
	for _, record := range records {
		if record.Key == key && len(record.Value) > 0 {
			return record.Value
		}
	}
	return defaultValue
}

// IsRunningInsideDocker detects if the current process is running inside a Docker container
// by checking for the presence of the /.dockerenv file.
func IsRunningInsideDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return !os.IsNotExist(err)
}

// IsRunningInsideWSL2 detects if the current process is running inside WSL2
// by checking for "microsoft" or "WSL" in /proc/version.
func IsRunningInsideWSL2() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	version := strings.ToLower(string(data))
	return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
}

// GetWSL2WindowsHost returns the Windows host IP from inside WSL2
// by reading the default gateway from /proc/net/route. In WSL2, the
// default gateway is the Windows host. This is a file read only —
// no command execution.
func GetWSL2WindowsHost() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[1] != "00000000" {
			continue
		}
		// Default route found — parse gateway from hex to IP
		gw, err := strconv.ParseUint(fields[2], 16, 32)
		if err != nil {
			return ""
		}
		return fmt.Sprintf("%d.%d.%d.%d", gw&0xFF, (gw>>8)&0xFF, (gw>>16)&0xFF, (gw>>24)&0xFF)
	}
	return ""
}

// FilterList returns a new slice containing only the elements for which the predicate returns true.
func FilterList[T any](items []T, by func(input T) bool) []T {
	filteredItems := []T{}
	for _, item := range items {
		if by(item) {
			filteredItems = append(filteredItems, item)
		}
	}
	return filteredItems
}

// OpenBrowser opens the specified URL in the system's default browser.
// Supports Windows, macOS, and Linux. Logs a warning if the browser cannot be opened.
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
		// Unsupported platform - silently continue
	}
	if err != nil {
		log.Warnf("Failed to open browser: %v\n", err)
	}
}

// StrPtrToBool converts a string pointer to a boolean.
// Returns true if the string value is "true" (case-insensitive), false otherwise or if nil.
func StrPtrToBool(s *string) bool {
	if s == nil {
		return false
	}
	value := strings.ToLower(*s)
	return value == "true"
}
