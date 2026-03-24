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

package memcached

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// ExportData exports a Memcached item to tabular format.
func (p *MemcachedPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error {
	headers := []string{
		common.FormatCSVHeader("Value", "string"),
		common.FormatCSVHeader("Flags", "uint32"),
	}
	if err := writer(headers); err != nil {
		return err
	}

	if len(selectedRows) > 0 {
		for _, row := range selectedRows {
			if err := writer([]string{
				fmt.Sprintf("%v", row["Value"]),
				fmt.Sprintf("%v", row["Flags"]),
			}); err != nil {
				return err
			}
		}
		return nil
	}

	client, err := DB(config)
	if err != nil {
		return err
	}
	defer client.Close()

	item, err := client.Get(storageUnit)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}

	return writer([]string{
		string(item.Value),
		strconv.FormatUint(uint64(item.Flags), 10),
	})
}

// ExportDataNDJSON streams a Memcached item as NDJSON.
func (p *MemcachedPlugin) ExportDataNDJSON(config *engine.PluginConfig, schema string, storageUnit string, writer func(string) error, selectedRows []map[string]any) error {
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

	client, err := DB(config)
	if err != nil {
		return err
	}
	defer client.Close()

	item, err := client.Get(storageUnit)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}

	return emit([]map[string]any{{
		"Value": string(item.Value),
		"Flags": item.Flags,
	}})
}
