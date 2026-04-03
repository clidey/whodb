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
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// MemcachedPlugin implements PluginFunctions for Memcached.
type MemcachedPlugin struct {
	engine.BasePlugin
}

var memcachedOperators = map[string]string{
	"=":           "=",
	"!=":          "!=",
	"<>":          "!=",
	">":           ">",
	">=":          ">=",
	"<":           "<",
	"<=":          "<=",
	"CONTAINS":    "CONTAINS",
	"STARTS WITH": "STARTS WITH",
	"ENDS WITH":   "ENDS WITH",
	"IN":          "IN",
	"NOT IN":      "NOT IN",
}

// IsAvailable checks if the Memcached server is reachable.
func (p *MemcachedPlugin) IsAvailable(ctx context.Context, config *engine.PluginConfig) bool {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).Error("Failed to connect to Memcached for availability check")
		return false
	}
	defer client.Close()
	return client.Ping() == nil
}

// GetDatabases returns a single entry since Memcached has a flat keyspace with no database concept.
func (p *MemcachedPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return []string{"default"}, nil
}

// GetStorageUnits lists all items in Memcached via lru_crawler metadump.
func (p *MemcachedPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).Error("Failed to connect to Memcached for storage units retrieval")
		return nil, err
	}
	defer client.Close()

	entries, err := client.Metadump()
	if err != nil {
		log.WithError(err).Error("Failed to enumerate Memcached keys via metadump")
		return nil, err
	}

	storageUnits := make([]engine.StorageUnit, 0, len(entries))
	for _, entry := range entries {
		if entry.Key == "auth" {
			continue
		}
		expValue := "never"
		if entry.Expiration > 0 {
			expValue = time.Unix(entry.Expiration, 0).UTC().Format(time.RFC3339)
		}
		storageUnits = append(storageUnits, engine.StorageUnit{
			Name: entry.Key,
			Attributes: []engine.Record{
				{Key: "Type", Value: "string"},
				{Key: "Size", Value: strconv.Itoa(entry.Size)},
				{Key: "Expires", Value: expValue},
			},
		})
	}

	return storageUnits, nil
}

// StorageUnitExists checks if a key exists in Memcached.
func (p *MemcachedPlugin) StorageUnitExists(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Close()

	item, err := client.Get(storageUnit)
	if err != nil {
		return false, err
	}
	return item != nil, nil
}

// GetRows retrieves a single item's data as a row with columns: Value, Flags.
func (p *MemcachedPlugin) GetRows(
	config *engine.PluginConfig,
	req *engine.GetRowsRequest,
) (*engine.GetRowsResult, error) {
	storageUnit := req.StorageUnit
	where := req.Where

	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to Memcached for rows retrieval")
		return nil, err
	}
	defer client.Close()

	item, err := client.Get(storageUnit)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Memcached item")
		return nil, err
	}
	if item == nil {
		return &engine.GetRowsResult{
			Columns:    memcachedColumns(),
			Rows:       [][]string{},
			TotalCount: 0,
		}, nil
	}

	row := []string{
		string(item.Value),
		strconv.FormatUint(uint64(item.Flags), 10),
	}

	rows := [][]string{row}

	// Apply filtering if where condition is set
	if where != nil {
		rows = filterMemcachedRows(rows, where)
	}

	return &engine.GetRowsResult{
		Columns:    memcachedColumns(),
		Rows:       rows,
		TotalCount: int64(len(rows)),
	}, nil
}

// GetRowCount returns the count of items for a key (always 0 or 1 for Memcached).
func (p *MemcachedPlugin) GetRowCount(config *engine.PluginConfig, schema, storageUnit string, where *model.WhereCondition) (int64, error) {
	client, err := DB(config)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	item, err := client.Get(storageUnit)
	if err != nil {
		return 0, err
	}
	if item == nil {
		return 0, nil
	}
	return 1, nil
}

// GetColumnsForTable returns the column definitions for a Memcached item.
func (p *MemcachedPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	return memcachedColumns(), nil
}

// FormatValue converts a value to its string representation.
func (p *MemcachedPlugin) FormatValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// GetDatabaseMetadata returns Memcached metadata for frontend configuration.
func (p *MemcachedPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	ops := make([]string, 0, len(memcachedOperators))
	for op := range memcachedOperators {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return &engine.DatabaseMetadata{
		DatabaseType:    engine.DatabaseType_Memcached,
		TypeDefinitions: []engine.TypeDefinition{},
		Operators:       ops,
		AliasMap:        map[string]string{},
		Capabilities:    engine.Capabilities{},
	}
}

func init() {
	engine.RegisterPlugin(NewMemcachedPlugin())
}

// NewMemcachedPlugin creates a new Memcached plugin.
func NewMemcachedPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Memcached,
		PluginFunctions: &MemcachedPlugin{},
	}
}

// memcachedColumns returns the standard column definitions for a Memcached item.
func memcachedColumns() []engine.Column {
	return []engine.Column{
		{Name: "Value", Type: "string"},
		{Name: "Flags", Type: "uint32"},
	}
}

// filterMemcachedRows applies a where condition to memcached rows.
func filterMemcachedRows(rows [][]string, where *model.WhereCondition) [][]string {
	condition, err := convertWhereCondition(where)
	if err != nil {
		return rows
	}

	var filtered [][]string
	columnIndex := map[string]int{
		"value": 0,
		"flags": 1,
	}

	for _, row := range rows {
		match := true
		for col, filter := range condition {
			idx, ok := columnIndex[strings.ToLower(col)]
			if !ok {
				continue
			}
			if !evaluateCondition(row[idx], filter.Operator, filter.Value) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

type memcachedFilter struct {
	Operator string
	Value    string
}

func convertWhereCondition(where *model.WhereCondition) (map[string]memcachedFilter, error) {
	if where == nil {
		return nil, nil
	}

	switch where.Type {
	case model.WhereConditionTypeAtomic:
		if where.Atomic == nil {
			return nil, fmt.Errorf("atomic condition must have an atomicwherecondition")
		}
		return map[string]memcachedFilter{
			where.Atomic.Key: {
				Operator: strings.ToUpper(where.Atomic.Operator),
				Value:    where.Atomic.Value,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported Memcached filtering condition type: %v", where.Type)
	}
}

func evaluateCondition(value, operator, target string) bool {
	switch operator {
	case "=", "EQ":
		return value == target
	case "!=", "NE", "<>":
		return value != target
	case ">":
		return value > target
	case ">=":
		return value >= target
	case "<":
		return value < target
	case "<=":
		return value <= target
	case "CONTAINS":
		return strings.Contains(value, target)
	case "STARTS WITH":
		return strings.HasPrefix(value, target)
	case "ENDS WITH":
		return strings.HasSuffix(value, target)
	case "IN":
		parts := strings.Split(target, ",")
		for _, p := range parts {
			if value == strings.TrimSpace(p) {
				return true
			}
		}
		return false
	case "NOT IN":
		parts := strings.Split(target, ",")
		for _, p := range parts {
			if value == strings.TrimSpace(p) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
