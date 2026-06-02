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

package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// ExportData exports Redis data to tabular format
func (p *RedisPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error {
	client, err := DB(config)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	keyType, err := client.Type(client.Context(), storageUnit).Result()
	if err != nil {
		return err
	}

	switch keyType {
	case redisTypeString:
		headers := []string{common.FormatCSVHeader(redisKeyValue, "string")}
		if err := writer(headers); err != nil {
			return err
		}
		val, err := client.Get(client.Context(), storageUnit).Result()
		if err != nil {
			return err
		}
		return writer([]string{val})

	case redisTypeHash:
		headers := []string{common.FormatCSVHeader("field", "string"), common.FormatCSVHeader(redisKeyValue, "string")}
		if err := writer(headers); err != nil {
			return err
		}

		// If selected rows provided, use them
		if len(selectedRows) > 0 {
			for _, row := range selectedRows {
				if err := writer([]string{fmt.Sprintf("%v", row["field"]), fmt.Sprintf("%v", row[redisKeyValue])}); err != nil {
					return err
				}
			}
			return nil
		}

		values, err := client.HGetAll(client.Context(), storageUnit).Result()
		if err != nil {
			return err
		}
		fields := make([]string, 0, len(values))
		for f := range values {
			fields = append(fields, f)
		}
		sort.Strings(fields)
		for _, f := range fields {
			if err := writer([]string{f, values[f]}); err != nil {
				return err
			}
		}
		return nil

	case redisTypeList:
		headers := []string{common.FormatCSVHeader("index", "string"), common.FormatCSVHeader(redisKeyValue, "string")}
		if err := writer(headers); err != nil {
			return err
		}

		if len(selectedRows) > 0 {
			for _, row := range selectedRows {
				if err := writer([]string{fmt.Sprintf("%v", row["index"]), fmt.Sprintf("%v", row[redisKeyValue])}); err != nil {
					return err
				}
			}
			return nil
		}

		values, err := client.LRange(client.Context(), storageUnit, 0, -1).Result()
		if err != nil {
			return err
		}
		for i, v := range values {
			if err := writer([]string{strconv.Itoa(i), v}); err != nil {
				return err
			}
		}
		return nil

	case "set":
		headers := []string{common.FormatCSVHeader("index", "string"), common.FormatCSVHeader(redisKeyValue, "string")}
		if err := writer(headers); err != nil {
			return err
		}

		if len(selectedRows) > 0 {
			for _, row := range selectedRows {
				if err := writer([]string{fmt.Sprintf("%v", row["index"]), fmt.Sprintf("%v", row[redisKeyValue])}); err != nil {
					return err
				}
			}
			return nil
		}

		values, err := client.SMembers(client.Context(), storageUnit).Result()
		if err != nil {
			return err
		}
		sort.Strings(values)
		for i, v := range values {
			if err := writer([]string{strconv.Itoa(i), v}); err != nil {
				return err
			}
		}
		return nil

	case redisTypeZSet:
		headers := []string{
			common.FormatCSVHeader("index", "string"),
			common.FormatCSVHeader("member", "string"),
			common.FormatCSVHeader("score", "string"),
		}
		if err := writer(headers); err != nil {
			return err
		}

		if len(selectedRows) > 0 {
			for _, row := range selectedRows {
				if err := writer([]string{
					fmt.Sprintf("%v", row["index"]),
					fmt.Sprintf("%v", row["member"]),
					fmt.Sprintf("%v", row["score"]),
				}); err != nil {
					return err
				}
			}
			return nil
		}

		values, err := client.ZRangeWithScores(client.Context(), storageUnit, 0, -1).Result()
		if err != nil {
			return err
		}
		for i, m := range values {
			if err := writer([]string{
				strconv.Itoa(i),
				fmt.Sprintf("%v", m.Member),
				fmt.Sprintf("%.2f", m.Score),
			}); err != nil {
				return err
			}
		}
		return nil
	}

	return errors.New("unsupported Redis data type")
}

// ExportDataNDJSON streams Redis data as NDJSON.
func (p *RedisPlugin) ExportDataNDJSON(config *engine.PluginConfig, schema string, storageUnit string, writer func(string) error, selectedRows []map[string]any) error {
	client, err := DB(config)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	keyType, err := client.Type(client.Context(), storageUnit).Result()
	if err != nil {
		return err
	}

	emit := func(rows []map[string]any) error {
		for _, row := range rows {
			line, err := json.Marshal(row)
			if err != nil {
				return err
			}
			if err := writer(string(line)); err != nil {
				return err
			}
		}
		return nil
	}

	if len(selectedRows) > 0 {
		return emit(selectedRows)
	}

	ctx := client.Context()

	switch keyType {
	case redisTypeString:
		val, err := client.Get(ctx, storageUnit).Result()
		if err != nil {
			return err
		}
		return emit([]map[string]any{{redisKeyValue: val}})

	case redisTypeHash:
		values, err := client.HGetAll(ctx, storageUnit).Result()
		if err != nil {
			return err
		}
		rows := make([]map[string]any, 0, len(values))
		for field, value := range values {
			rows = append(rows, map[string]any{"field": field, redisKeyValue: value})
		}
		return emit(rows)

	case redisTypeList:
		values, err := client.LRange(ctx, storageUnit, 0, -1).Result()
		if err != nil {
			return err
		}
		rows := make([]map[string]any, 0, len(values))
		for i, v := range values {
			rows = append(rows, map[string]any{"index": i, redisKeyValue: v})
		}
		return emit(rows)

	case "set":
		values, err := client.SMembers(ctx, storageUnit).Result()
		if err != nil {
			return err
		}
		sort.Strings(values)
		rows := make([]map[string]any, 0, len(values))
		for i, v := range values {
			rows = append(rows, map[string]any{"index": i, redisKeyValue: v})
		}
		return emit(rows)

	case redisTypeZSet:
		values, err := client.ZRangeWithScores(ctx, storageUnit, 0, -1).Result()
		if err != nil {
			return err
		}
		rows := make([]map[string]any, 0, len(values))
		for i, m := range values {
			rows = append(rows, map[string]any{
				"index":  i,
				"member": m.Member,
				"score":  fmt.Sprintf("%.2f", m.Score),
			})
		}
		return emit(rows)
	}

	return errors.New("unsupported Redis data type")
}
