package common

import (
	"regexp"

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
