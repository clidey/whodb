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

package common

import (
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
		// Unsupported platform - silently continue
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
