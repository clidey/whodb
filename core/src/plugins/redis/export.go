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
	"fmt"
	"sort"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// ExportData exports Redis data to tabular format
func (p *RedisPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error {
	client, err := DB(config)
	if err != nil {
		return err
	}
	defer client.Close()

	keyType, err := client.Type(client.Context(), storageUnit).Result()
	if err != nil {
		return err
	}

	switch keyType {
	case "string":
		headers := []string{common.FormatCSVHeader("value", "string")}
		if err := writer(headers); err != nil {
			return err
		}
		val, err := client.Get(client.Context(), storageUnit).Result()
		if err != nil {
			return err
		}
		return writer([]string{val})

	case "hash":
		headers := []string{common.FormatCSVHeader("field", "string"), common.FormatCSVHeader("value", "string")}
		if err := writer(headers); err != nil {
			return err
		}

		// If selected rows provided, use them
		if len(selectedRows) > 0 {
			for _, row := range selectedRows {
				writer([]string{fmt.Sprintf("%v", row["field"]), fmt.Sprintf("%v", row["value"])})
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

	case "list":
		headers := []string{common.FormatCSVHeader("index", "string"), common.FormatCSVHeader("value", "string")}
		if err := writer(headers); err != nil {
			return err
		}

		if len(selectedRows) > 0 {
			for _, row := range selectedRows {
				writer([]string{fmt.Sprintf("%v", row["index"]), fmt.Sprintf("%v", row["value"])})
			}
			return nil
		}

		values, err := client.LRange(client.Context(), storageUnit, 0, -1).Result()
		if err != nil {
			return err
		}
		for i, v := range values {
			if err := writer([]string{fmt.Sprintf("%d", i), v}); err != nil {
				return err
			}
		}
		return nil

	case "set":
		headers := []string{common.FormatCSVHeader("index", "string"), common.FormatCSVHeader("value", "string")}
		if err := writer(headers); err != nil {
			return err
		}

		if len(selectedRows) > 0 {
			for _, row := range selectedRows {
				writer([]string{fmt.Sprintf("%v", row["index"]), fmt.Sprintf("%v", row["value"])})
			}
			return nil
		}

		values, err := client.SMembers(client.Context(), storageUnit).Result()
		if err != nil {
			return err
		}
		sort.Strings(values)
		for i, v := range values {
			if err := writer([]string{fmt.Sprintf("%d", i), v}); err != nil {
				return err
			}
		}
		return nil

	case "zset":
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
				writer([]string{
					fmt.Sprintf("%v", row["index"]),
					fmt.Sprintf("%v", row["member"]),
					fmt.Sprintf("%v", row["score"]),
				})
			}
			return nil
		}

		values, err := client.ZRangeWithScores(client.Context(), storageUnit, 0, -1).Result()
		if err != nil {
			return err
		}
		for i, m := range values {
			if err := writer([]string{
				fmt.Sprintf("%d", i),
				fmt.Sprintf("%v", m.Member),
				fmt.Sprintf("%.2f", m.Score),
			}); err != nil {
				return err
			}
		}
		return nil
	}

	return fmt.Errorf("unsupported Redis data type")
}
