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

package clickhouse

import "strings"

// clickhouseTypeMap normalizes ClickHouse canonical types to standard SQL types
// that the base GormPlugin's ConvertStringValue understands.
var clickhouseTypeMap = map[string]string{
	"INT8":    "TINYINT",
	"INT16":   "SMALLINT",
	"INT32":   "INT",
	"INT64":   "BIGINT",
	"INT128":  "BIGINT",
	"INT256":  "BIGINT",
	"UINT8":   "TINYINT",
	"UINT16":  "SMALLINT",
	"UINT32":  "INT",
	"UINT64":  "BIGINT",
	"UINT128": "BIGINT",
	"UINT256": "BIGINT",
	"FLOAT32": "FLOAT",
	"FLOAT64": "DOUBLE",
}

// normalizeClickHouseType converts ClickHouse canonical types to standard SQL types
func normalizeClickHouseType(columnType string) string {
	upper := strings.ToUpper(columnType)
	if normalized, ok := clickhouseTypeMap[upper]; ok {
		return normalized
	}
	return columnType
}

func (p *ClickHousePlugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return p.ConvertStringValue(value, normalizeClickHouseType(columnType))
}

// Identifier quoting handled by GORM Dialector

func (p *ClickHousePlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT name
		FROM system.columns
		WHERE database = ? AND table = ? AND is_in_primary_key = 1
	`
}

