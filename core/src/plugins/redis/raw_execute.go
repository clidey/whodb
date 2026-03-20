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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/go-redis/redis/v8"
)

// RawExecute parses and executes Redis commands.
// The command string is split into parts respecting quoted strings,
// then dispatched to the appropriate Redis client method.
func (p *RedisPlugin) RawExecute(config *engine.PluginConfig, query string, _ ...any) (*engine.GetRowsResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty Redis command")
	}

	parts := splitCommand(query)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty Redis command")
	}

	command := strings.ToUpper(parts[0])
	args := parts[1:]

	client, err := DB(config)
	if err != nil {
		log.WithError(err).Error("Failed to connect to Redis for raw execute")
		return nil, err
	}
	defer client.Close()

	ctx := context.Background()

	switch command {
	case "GET":
		if len(args) < 1 {
			return nil, fmt.Errorf("GET requires a key argument")
		}
		val, err := client.Get(ctx, args[0]).Result()
		if err == redis.Nil {
			return singleValueResult("(nil)"), nil
		}
		if err != nil {
			return nil, fmt.Errorf("GET failed: %v", err)
		}
		return singleValueResult(val), nil

	case "SET":
		if len(args) < 2 {
			return nil, fmt.Errorf("SET requires key and value arguments")
		}
		err := client.Set(ctx, args[0], args[1], 0).Err()
		if err != nil {
			return nil, fmt.Errorf("SET failed: %v", err)
		}
		return singleResultMessage("OK"), nil

	case "DEL":
		if len(args) < 1 {
			return nil, fmt.Errorf("DEL requires at least one key argument")
		}
		count, err := client.Del(ctx, args...).Result()
		if err != nil {
			return nil, fmt.Errorf("DEL failed: %v", err)
		}
		return singleResultMessage(strconv.FormatInt(count, 10)), nil

	case "TYPE":
		if len(args) < 1 {
			return nil, fmt.Errorf("TYPE requires a key argument")
		}
		val, err := client.Type(ctx, args[0]).Result()
		if err != nil {
			return nil, fmt.Errorf("TYPE failed: %v", err)
		}
		return singleValueResult(val), nil

	case "TTL":
		if len(args) < 1 {
			return nil, fmt.Errorf("TTL requires a key argument")
		}
		val, err := client.TTL(ctx, args[0]).Result()
		if err != nil {
			return nil, fmt.Errorf("TTL failed: %v", err)
		}
		return singleValueResult(val.String()), nil

	case "KEYS":
		pattern := "*"
		if len(args) > 0 {
			pattern = args[0]
		}
		keys, err := client.Keys(ctx, pattern).Result()
		if err != nil {
			return nil, fmt.Errorf("KEYS failed: %v", err)
		}
		return stringListResult(keys), nil

	case "HGETALL":
		if len(args) < 1 {
			return nil, fmt.Errorf("HGETALL requires a key argument")
		}
		vals, err := client.HGetAll(ctx, args[0]).Result()
		if err != nil {
			return nil, fmt.Errorf("HGETALL failed: %v", err)
		}
		return hashResult(vals), nil

	case "HSET":
		if len(args) < 3 {
			return nil, fmt.Errorf("HSET requires key, field, and value arguments")
		}
		err := client.HSet(ctx, args[0], args[1], args[2]).Err()
		if err != nil {
			return nil, fmt.Errorf("HSET failed: %v", err)
		}
		return singleResultMessage("OK"), nil

	case "HGET":
		if len(args) < 2 {
			return nil, fmt.Errorf("HGET requires key and field arguments")
		}
		val, err := client.HGet(ctx, args[0], args[1]).Result()
		if err == redis.Nil {
			return singleValueResult("(nil)"), nil
		}
		if err != nil {
			return nil, fmt.Errorf("HGET failed: %v", err)
		}
		return singleValueResult(val), nil

	case "SMEMBERS":
		if len(args) < 1 {
			return nil, fmt.Errorf("SMEMBERS requires a key argument")
		}
		vals, err := client.SMembers(ctx, args[0]).Result()
		if err != nil {
			return nil, fmt.Errorf("SMEMBERS failed: %v", err)
		}
		return stringListResult(vals), nil

	case "SADD":
		if len(args) < 2 {
			return nil, fmt.Errorf("SADD requires key and member arguments")
		}
		members := make([]any, len(args)-1)
		for i, a := range args[1:] {
			members[i] = a
		}
		count, err := client.SAdd(ctx, args[0], members...).Result()
		if err != nil {
			return nil, fmt.Errorf("SADD failed: %v", err)
		}
		return singleResultMessage(strconv.FormatInt(count, 10)), nil

	case "LRANGE":
		if len(args) < 3 {
			return nil, fmt.Errorf("LRANGE requires key, start, and stop arguments")
		}
		start, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid start index: %v", err)
		}
		stop, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid stop index: %v", err)
		}
		vals, err := client.LRange(ctx, args[0], start, stop).Result()
		if err != nil {
			return nil, fmt.Errorf("LRANGE failed: %v", err)
		}
		return listResult(vals), nil

	case "LPUSH", "RPUSH":
		if len(args) < 2 {
			return nil, fmt.Errorf("%s requires key and value arguments", command)
		}
		values := make([]any, len(args)-1)
		for i, a := range args[1:] {
			values[i] = a
		}
		var count int64
		if command == "LPUSH" {
			count, err = client.LPush(ctx, args[0], values...).Result()
		} else {
			count, err = client.RPush(ctx, args[0], values...).Result()
		}
		if err != nil {
			return nil, fmt.Errorf("%s failed: %v", command, err)
		}
		return singleResultMessage(strconv.FormatInt(count, 10)), nil

	case "ZRANGEBYSCORE":
		if len(args) < 3 {
			return nil, fmt.Errorf("ZRANGEBYSCORE requires key, min, and max arguments")
		}
		withScores := len(args) > 3 && strings.ToUpper(args[3]) == "WITHSCORES"
		if withScores {
			vals, err := client.ZRangeByScoreWithScores(ctx, args[0], &redis.ZRangeBy{
				Min: args[1],
				Max: args[2],
			}).Result()
			if err != nil {
				return nil, fmt.Errorf("ZRANGEBYSCORE failed: %v", err)
			}
			return zsetResult(vals), nil
		}
		vals, err := client.ZRangeByScore(ctx, args[0], &redis.ZRangeBy{
			Min: args[1],
			Max: args[2],
		}).Result()
		if err != nil {
			return nil, fmt.Errorf("ZRANGEBYSCORE failed: %v", err)
		}
		return stringListResult(vals), nil

	case "ZRANGE":
		if len(args) < 3 {
			return nil, fmt.Errorf("ZRANGE requires key, start, and stop arguments")
		}
		start, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid start index: %v", err)
		}
		stop, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid stop index: %v", err)
		}
		withScores := len(args) > 3 && strings.ToUpper(args[3]) == "WITHSCORES"
		if withScores {
			vals, err := client.ZRangeWithScores(ctx, args[0], start, stop).Result()
			if err != nil {
				return nil, fmt.Errorf("ZRANGE failed: %v", err)
			}
			return zsetResult(vals), nil
		}
		vals, err := client.ZRange(ctx, args[0], start, stop).Result()
		if err != nil {
			return nil, fmt.Errorf("ZRANGE failed: %v", err)
		}
		return stringListResult(vals), nil

	case "EXISTS":
		if len(args) < 1 {
			return nil, fmt.Errorf("EXISTS requires at least one key argument")
		}
		count, err := client.Exists(ctx, args...).Result()
		if err != nil {
			return nil, fmt.Errorf("EXISTS failed: %v", err)
		}
		return singleResultMessage(strconv.FormatInt(count, 10)), nil

	case "EXPIRE":
		if len(args) < 2 {
			return nil, fmt.Errorf("EXPIRE requires key and seconds arguments")
		}
		seconds, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid seconds value: %v", err)
		}
		ok, err := client.Expire(ctx, args[0], time.Duration(seconds)*time.Second).Result()
		if err != nil {
			return nil, fmt.Errorf("EXPIRE failed: %v", err)
		}
		if ok {
			return singleResultMessage("1"), nil
		}
		return singleResultMessage("0"), nil

	default:
		// Fallback: use generic Do for any unhandled command
		cmdArgs := make([]any, len(parts))
		for i, p := range parts {
			cmdArgs[i] = p
		}
		val, err := client.Do(ctx, cmdArgs...).Result()
		if err == redis.Nil {
			return singleValueResult("(nil)"), nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s failed: %v", command, err)
		}
		return singleValueResult(fmt.Sprintf("%v", val)), nil
	}
}

// splitCommand splits a Redis command string into parts, respecting quoted strings.
func splitCommand(s string) []string {
	var parts []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escape := false

	for _, ch := range s {
		if escape {
			current.WriteRune(ch)
			escape = false
			continue
		}
		if ch == '\\' && (inSingleQuote || inDoubleQuote) {
			escape = true
			continue
		}
		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}
		if ch == ' ' && !inSingleQuote && !inDoubleQuote {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(ch)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// singleValueResult creates a 1-column "value" result with a single row.
func singleValueResult(val string) *engine.GetRowsResult {
	return &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "value", Type: "string"}},
		Rows:    [][]string{{val}},
	}
}

// singleResultMessage creates a 1-column "result" result with a single row.
func singleResultMessage(msg string) *engine.GetRowsResult {
	return &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "result", Type: "string"}},
		Rows:    [][]string{{msg}},
	}
}

// stringListResult creates a 1-column "value" result with one row per string.
func stringListResult(vals []string) *engine.GetRowsResult {
	rows := make([][]string, 0, len(vals))
	for _, v := range vals {
		rows = append(rows, []string{v})
	}
	return &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "value", Type: "string"}},
		Rows:    rows,
	}
}

// hashResult creates a 2-column "field"/"value" result from a hash map.
func hashResult(vals map[string]string) *engine.GetRowsResult {
	rows := make([][]string, 0, len(vals))
	for field, value := range vals {
		rows = append(rows, []string{field, value})
	}
	return &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "field", Type: "string"}, {Name: "value", Type: "string"}},
		Rows:    rows,
	}
}

// listResult creates a 2-column "index"/"value" result from a list.
func listResult(vals []string) *engine.GetRowsResult {
	rows := make([][]string, 0, len(vals))
	for i, v := range vals {
		rows = append(rows, []string{strconv.Itoa(i), v})
	}
	return &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "index", Type: "string"}, {Name: "value", Type: "string"}},
		Rows:    rows,
	}
}

// zsetResult creates a 2-column "member"/"score" result from sorted set members.
func zsetResult(vals []redis.Z) *engine.GetRowsResult {
	rows := make([][]string, 0, len(vals))
	for _, z := range vals {
		member := fmt.Sprintf("%v", z.Member)
		score := fmt.Sprintf("%.2f", z.Score)
		rows = append(rows, []string{member, score})
	}
	return &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "member", Type: "string"}, {Name: "score", Type: "string"}},
		Rows:    rows,
	}
}
