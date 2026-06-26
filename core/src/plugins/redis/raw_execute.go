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

package redis

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"

	"github.com/clidey/whodb/core/src/engine"
)

var redisKVPairCommands = map[string]bool{
	"HGETALL":   true,
	"CONFIG":    true,
	"XRANGE":    true,
	"XREVRANGE": true,
}

// RawExecute executes one Redis command and formats the reply as rows.
func (p *RedisPlugin) RawExecute(config *engine.PluginConfig, query string, _ ...any) (*engine.GetRowsResult, error) {
	tokens := tokenizeRedisCommand(query)
	if len(tokens) == 0 {
		return nil, errors.New("empty Redis command")
	}

	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()

	args := make([]any, len(tokens))
	for i, token := range tokens {
		args[i] = token
	}

	command := strings.ToUpper(tokens[0])
	result, err := client.Do(config.OperationContext(), args...).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return formatRedisRawResult(command, nil), nil
		}
		return nil, err
	}

	return formatRedisRawResult(command, result), nil
}

func formatRedisRawResult(command string, result any) *engine.GetRowsResult {
	switch v := result.(type) {
	case nil:
		return singleValueRedisResult("(nil)")
	case string:
		return singleValueRedisResult(v)
	case int:
		return singleValueRedisResult(strconv.Itoa(v))
	case int64:
		return singleValueRedisResult(strconv.FormatInt(v, 10))
	case []any:
		if redisKVPairCommands[command] && len(v)%2 == 0 {
			return redisKVPairResult(v)
		}
		return redisIndexedListResult(v)
	default:
		return singleValueRedisResult(fmt.Sprintf("%v", v))
	}
}

func singleValueRedisResult(value string) *engine.GetRowsResult {
	return &engine.GetRowsResult{
		Columns:       []engine.Column{{Name: redisKeyValue, Type: redisTypeString}},
		Rows:          [][]string{{value}},
		DisableUpdate: true,
		TotalCount:    1,
	}
}

func redisKVPairResult(flat []any) *engine.GetRowsResult {
	rows := make([][]string, 0, len(flat)/2)
	for i := 0; i+1 < len(flat); i += 2 {
		rows = append(rows, []string{fmt.Sprintf("%v", flat[i]), fmt.Sprintf("%v", flat[i+1])})
	}
	return &engine.GetRowsResult{
		Columns:       []engine.Column{{Name: "field", Type: redisTypeString}, {Name: redisKeyValue, Type: redisTypeString}},
		Rows:          rows,
		DisableUpdate: true,
		TotalCount:    int64(len(rows)),
	}
}

func redisIndexedListResult(items []any) *engine.GetRowsResult {
	rows := make([][]string, len(items))
	for i, item := range items {
		rows[i] = []string{strconv.Itoa(i), fmt.Sprintf("%v", item)}
	}
	return &engine.GetRowsResult{
		Columns:       []engine.Column{{Name: redisKeyIndex, Type: redisTypeString}, {Name: redisKeyValue, Type: redisTypeString}},
		Rows:          rows,
		DisableUpdate: true,
		TotalCount:    int64(len(rows)),
	}
}

func tokenizeRedisCommand(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	var tokens []string
	var current strings.Builder
	inDouble := false
	inSingle := false

	for i := range len(input) {
		b := input[i]
		switch {
		case b == '"' && !inSingle:
			inDouble = !inDouble
		case b == '\'' && !inDouble:
			inSingle = !inSingle
		case (b == ' ' || b == '\t' || b == '\n' || b == '\r') && !inDouble && !inSingle:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(b)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}
