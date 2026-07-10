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

package mongodb

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type mongoShellCommand struct {
	Collection string
	Method     string
	RawArgs    string
}

var (
	mongoShellCollectionPattern    = regexp.MustCompile(`(?s)^\s*db\.([a-zA-Z_][\w.]*?)\.(\w+)\s*\((.*)\)\s*;?\s*$`)
	mongoShellGetCollectionPattern = regexp.MustCompile(`(?s)^\s*db\.getCollection\(\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')\s*\)\.(\w+)\s*\((.*)\)\s*;?\s*$`)
	mongoShellDatabasePattern      = regexp.MustCompile(`(?s)^\s*db\.(\w+)\s*\((.*)\)\s*;?\s*$`)
	mongoShellUnquotedKeyPattern   = regexp.MustCompile(`(?m)([\{,]\s*)([a-zA-Z_$][\w$]*)\s*:`)
	mongoShellTrailingCommaPattern = regexp.MustCompile(`,\s*([\}\]])`)
)

func parseMongoShellCommand(input string) (*mongoShellCommand, error) {
	cleaned := stripMongoShellComments(input)
	if matches := mongoShellCollectionPattern.FindStringSubmatch(cleaned); matches != nil {
		return &mongoShellCommand{Collection: matches[1], Method: matches[2], RawArgs: strings.TrimSpace(matches[3])}, nil
	}
	if matches := mongoShellGetCollectionPattern.FindStringSubmatch(cleaned); matches != nil {
		collection, err := strconv.Unquote(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid getCollection argument: %w", err)
		}
		return &mongoShellCommand{Collection: collection, Method: matches[2], RawArgs: strings.TrimSpace(matches[3])}, nil
	}
	if matches := mongoShellDatabasePattern.FindStringSubmatch(cleaned); matches != nil {
		return &mongoShellCommand{Method: matches[1], RawArgs: strings.TrimSpace(matches[2])}, nil
	}
	return nil, fmt.Errorf(
		"only single db.collection.method(...) calls are supported; JavaScript constructs such as variables, functions, and loops are not supported. Received: %s",
		truncateMongoShellInput(strings.TrimSpace(input), 120),
	)
}

func stripMongoShellComments(input string) string {
	var builder strings.Builder
	for i := 0; i < len(input); {
		if i+1 < len(input) && input[i] == '/' && input[i+1] == '*' {
			end := strings.Index(input[i+2:], "*/")
			if end == -1 {
				break
			}
			i = i + 2 + end + 2
			continue
		}
		if i+1 < len(input) && input[i] == '/' && input[i+1] == '/' {
			end := strings.IndexByte(input[i:], '\n')
			if end == -1 {
				break
			}
			i += end
			continue
		}
		builder.WriteByte(input[i])
		i++
	}
	return builder.String()
}

func truncateMongoShellInput(input string, maxRunes int) string {
	runes := []rune(input)
	if len(runes) <= maxRunes {
		return input
	}
	return string(runes[:maxRunes]) + "..."
}

func preprocessMongoShellJS(input string) string {
	input = regexp.MustCompile(`ObjectId\(\s*["']([^"']+)["']\s*\)`).ReplaceAllString(input, `{"$$oid":"$1"}`)
	input = regexp.MustCompile(`new\s+Date\(\s*["']([^"']*)["']\s*\)`).ReplaceAllString(input, `{"$$date":"$1"}`)
	input = regexp.MustCompile(`new\s+Date\(\s*\)`).ReplaceAllString(input, `{"$$date":"`+time.Now().UTC().Format(time.RFC3339)+`"}`)
	input = regexp.MustCompile(`ISODate\(\s*["']([^"']*)["']\s*\)`).ReplaceAllString(input, `{"$$date":"$1"}`)
	input = regexp.MustCompile(`NumberInt\(\s*(-?\d+)\s*\)`).ReplaceAllString(input, `$1`)
	input = regexp.MustCompile(`NumberLong\(\s*(-?\d+)\s*\)`).ReplaceAllString(input, `$1`)
	input = replaceMongoShellSingleQuotedStrings(input)
	input = mongoShellUnquotedKeyPattern.ReplaceAllString(input, `${1}"$2":`)
	return mongoShellTrailingCommaPattern.ReplaceAllString(input, `$1`)
}

func parseMongoShellArgs(rawArgs string) ([]any, error) {
	if strings.TrimSpace(rawArgs) == "" {
		return nil, nil
	}
	processed := preprocessMongoShellJS(rawArgs)
	var args bson.A
	if err := bson.UnmarshalExtJSON([]byte("["+processed+"]"), false, &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	return args, nil
}

func replaceMongoShellSingleQuotedStrings(input string) string {
	var builder strings.Builder
	builder.Grow(len(input))
	inDouble := false
	escaped := false
	for i := 0; i < len(input); i++ {
		character := input[i]
		if inDouble {
			builder.WriteByte(character)
			if escaped {
				escaped = false
				continue
			}
			if character == '\\' {
				escaped = true
				continue
			}
			if character == '"' {
				inDouble = false
			}
			continue
		}
		if character == '"' {
			inDouble = true
			builder.WriteByte(character)
			continue
		}
		if character != '\'' {
			builder.WriteByte(character)
			continue
		}
		builder.WriteByte('"')
		for i++; i < len(input); i++ {
			if input[i] == '\\' && i+1 < len(input) {
				builder.WriteByte(input[i])
				i++
				builder.WriteByte(input[i])
				continue
			}
			if input[i] == '\'' {
				break
			}
			builder.WriteByte(input[i])
		}
		builder.WriteByte('"')
	}
	return builder.String()
}
