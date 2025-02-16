// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package common

import (
	"fmt"
	"github.com/clidey/whodb/core/src/log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
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

type ExtractedText struct {
	Type string
	Text string
}

func ExtractCodeFromResponse(response string) []ExtractedText {
	tripleBacktickPattern := regexp.MustCompile("(?s)```(sql)?(.*?)```")

	codeBlocks := tripleBacktickPattern.FindAllStringSubmatchIndex(response, -1)

	var result []ExtractedText
	var lastIndex int

	for _, loc := range codeBlocks {
		start, end := loc[0], loc[1]
		codeTypeStart, codeTypeEnd, contentStart, contentEnd := loc[2], loc[3], loc[4], loc[5]

		codeContent := response[contentStart:contentEnd]

		codeType := "sql"
		if codeTypeStart != -1 && codeTypeEnd != -1 {
			codeType = response[codeTypeStart:codeTypeEnd]
		}

		if start > lastIndex {
			result = append(result, ExtractedText{Type: "message", Text: response[lastIndex:start]})
		}

		result = append(result, ExtractedText{Type: codeType, Text: codeContent})

		lastIndex = end
	}

	if lastIndex < len(response) {
		result = append(result, ExtractedText{Type: "message", Text: response[lastIndex:]})
	}

	return result
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
