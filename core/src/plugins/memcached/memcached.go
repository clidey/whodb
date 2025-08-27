// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memcached

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

type MemcachedPlugin struct{}

func (p *MemcachedPlugin) IsAvailable(config *engine.PluginConfig) bool {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to connect to Memcached for availability check")
		return false
	}
	
	// Try to get a non-existent key to test connectivity
	_, err = client.Get("__whodb_test__")
	if err != nil && err != memcache.ErrCacheMiss {
		log.Logger.WithError(err).Error("Failed to test Memcached connectivity")
		return false
	}
	
	return true
}

func (p *MemcachedPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	// Memcached doesn't have databases, return a single default
	return []string{"default"}, nil
}

func (p *MemcachedPlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	// Memcached doesn't support schemas
	return nil, errors.ErrUnsupported
}

func (p *MemcachedPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to connect to Memcached for storage units retrieval")
		return nil, err
	}
	
	// Get server statistics to show slabs information
	stats, err := client.Stats()
	if err != nil {
		log.Logger.WithError(err).Error("Failed to retrieve Memcached stats")
		return nil, err
	}
	
	storageUnits := []engine.StorageUnit{}
	slabMap := make(map[string]bool)
	
	// Parse stats to find slabs
	for _, serverStats := range stats {
		for key := range serverStats {
			// Look for slab-related keys
			if strings.HasPrefix(key, "items:") {
				parts := strings.Split(key, ":")
				if len(parts) >= 2 {
					slabID := parts[1]
					slabMap[slabID] = true
				}
			}
		}
	}
	
	// Sort slab IDs for consistent ordering
	slabIDs := make([]string, 0, len(slabMap))
	for id := range slabMap {
		slabIDs = append(slabIDs, id)
	}
	sort.Strings(slabIDs)
	
	// Create storage units for each slab
	for _, slabID := range slabIDs {
		attributes := []engine.Record{}
		
		// Get slab-specific stats
		for _, serverStats := range stats {
			// Number of items in this slab
			if itemCount, ok := serverStats[fmt.Sprintf("items:%s:number", slabID)]; ok {
				attributes = append(attributes, engine.Record{
					Key:   "Items",
					Value: itemCount,
				})
			}
			
			// Age of oldest item
			if age, ok := serverStats[fmt.Sprintf("items:%s:age", slabID)]; ok {
				attributes = append(attributes, engine.Record{
					Key:   "Oldest Item Age (seconds)",
					Value: age,
				})
			}
			
			// Memory used by this slab
			if memory, ok := serverStats[fmt.Sprintf("items:%s:mem_requested", slabID)]; ok {
				attributes = append(attributes, engine.Record{
					Key:   "Memory Used",
					Value: memory,
				})
			}
			
			break // Only need stats from one server
		}
		
		storageUnits = append(storageUnits, engine.StorageUnit{
			Name:       fmt.Sprintf("slab_%s", slabID),
			Attributes: attributes,
		})
	}
	
	// Add a general stats storage unit
	generalAttrs := []engine.Record{}
	for _, serverStats := range stats {
		if totalItems, ok := serverStats["total_items"]; ok {
			generalAttrs = append(generalAttrs, engine.Record{
				Key:   "Total Items",
				Value: totalItems,
			})
		}
		if currItems, ok := serverStats["curr_items"]; ok {
			generalAttrs = append(generalAttrs, engine.Record{
				Key:   "Current Items",
				Value: currItems,
			})
		}
		if bytes, ok := serverStats["bytes"]; ok {
			generalAttrs = append(generalAttrs, engine.Record{
				Key:   "Bytes Used",
				Value: bytes,
			})
		}
		if getHits, ok := serverStats["get_hits"]; ok {
			generalAttrs = append(generalAttrs, engine.Record{
				Key:   "Get Hits",
				Value: getHits,
			})
		}
		if getMisses, ok := serverStats["get_misses"]; ok {
			generalAttrs = append(generalAttrs, engine.Record{
				Key:   "Get Misses",
				Value: getMisses,
			})
		}
		break // Only need stats from one server
	}
	
	if len(generalAttrs) > 0 {
		storageUnits = append(storageUnits, engine.StorageUnit{
			Name:       "general_stats",
			Attributes: generalAttrs,
		})
	}
	
	// Add a keys storage unit for browsing cached keys
	// Note: Memcached doesn't provide a native way to list all keys
	// This is a placeholder that shows this limitation
	storageUnits = append(storageUnits, engine.StorageUnit{
		Name: "keys",
		Attributes: []engine.Record{
			{Key: "Type", Value: "key-value"},
			{Key: "Note", Value: "Use 'get' operation to retrieve specific keys"},
		},
	})
	
	return storageUnits, nil
}

func (p *MemcachedPlugin) GetRows(
	config *engine.PluginConfig,
	schema, storageUnit string,
	where *model.WhereCondition,
	sortConditions []*model.SortCondition,
	pageSize, pageOffset int,
) (*engine.GetRowsResult, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to Memcached for rows retrieval")
		return nil, err
	}
	
	switch storageUnit {
	case "general_stats":
		return p.getGeneralStats(client)
	case "keys":
		return p.getKeysView(client, where, pageSize, pageOffset)
	default:
		if strings.HasPrefix(storageUnit, "slab_") {
			slabID := strings.TrimPrefix(storageUnit, "slab_")
			return p.getSlabDetails(client, slabID)
		}
		return nil, fmt.Errorf("unknown storage unit: %s", storageUnit)
	}
}

func (p *MemcachedPlugin) getGeneralStats(client *memcache.Client) (*engine.GetRowsResult, error) {
	stats, err := client.Stats()
	if err != nil {
		log.Logger.WithError(err).Error("Failed to retrieve Memcached stats")
		return nil, err
	}
	
	rows := [][]string{}
	for server, serverStats := range stats {
		for key, value := range serverStats {
			// Filter to show only important general stats
			if isGeneralStat(key) {
				rows = append(rows, []string{server, key, value})
			}
		}
	}
	
	// Sort rows for consistent display
	sort.Slice(rows, func(i, j int) bool {
		if rows[i][0] != rows[j][0] {
			return rows[i][0] < rows[j][0]
		}
		return rows[i][1] < rows[j][1]
	})
	
	return &engine.GetRowsResult{
		Columns: []engine.Column{
			{Name: "server", Type: "string"},
			{Name: "stat", Type: "string"},
			{Name: "value", Type: "string"},
		},
		Rows: rows,
		DisableUpdate: true,
	}, nil
}

func (p *MemcachedPlugin) getSlabDetails(client *memcache.Client, slabID string) (*engine.GetRowsResult, error) {
	stats, err := client.Stats()
	if err != nil {
		log.Logger.WithError(err).Error("Failed to retrieve Memcached stats for slab")
		return nil, err
	}
	
	rows := [][]string{}
	for server, serverStats := range stats {
		for key, value := range serverStats {
			// Filter stats for this specific slab
			if strings.Contains(key, fmt.Sprintf(":%s:", slabID)) || strings.HasSuffix(key, fmt.Sprintf(":%s", slabID)) {
				statName := key
				// Clean up the stat name
				statName = strings.Replace(statName, fmt.Sprintf("items:%s:", slabID), "", 1)
				statName = strings.Replace(statName, fmt.Sprintf("slabs:%s:", slabID), "", 1)
				rows = append(rows, []string{server, statName, value})
			}
		}
	}
	
	// Sort rows for consistent display
	sort.Slice(rows, func(i, j int) bool {
		if rows[i][0] != rows[j][0] {
			return rows[i][0] < rows[j][0]
		}
		return rows[i][1] < rows[j][1]
	})
	
	return &engine.GetRowsResult{
		Columns: []engine.Column{
			{Name: "server", Type: "string"},
			{Name: "stat", Type: "string"},
			{Name: "value", Type: "string"},
		},
		Rows: rows,
		DisableUpdate: true,
	}, nil
}

func (p *MemcachedPlugin) getKeysView(client *memcache.Client, where *model.WhereCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	// If a specific key is requested via WHERE condition, fetch it
	if where != nil && where.Type == model.WhereConditionTypeAtomic && where.Atomic != nil && where.Atomic.Key == "key" && where.Atomic.Operator == "=" {
		key := where.Atomic.Value
		item, err := client.Get(key)
		if err != nil {
			if err == memcache.ErrCacheMiss {
				return &engine.GetRowsResult{
					Columns: []engine.Column{
						{Name: "key", Type: "string"},
						{Name: "value", Type: "string"},
						{Name: "flags", Type: "string"},
					},
					Rows: [][]string{},
				}, nil
			}
			log.Logger.WithError(err).WithField("key", key).Error("Failed to get Memcached key")
			return nil, err
		}
		
		return &engine.GetRowsResult{
			Columns: []engine.Column{
				{Name: "key", Type: "string"},
				{Name: "value", Type: "string"},
				{Name: "flags", Type: "string"},
			},
			Rows: [][]string{
				{item.Key, string(item.Value), strconv.Itoa(int(item.Flags))},
			},
		}, nil
	}
	
	// Without a WHERE clause, we can't list all keys in Memcached
	// Show a helpful message instead
	return &engine.GetRowsResult{
		Columns: []engine.Column{
			{Name: "info", Type: "string"},
		},
		Rows: [][]string{
			{"Memcached does not support listing all keys."},
			{"Use a WHERE clause with key = 'your_key' to fetch a specific key."},
		},
		DisableUpdate: true,
	}, nil
}

func isGeneralStat(key string) bool {
	generalStats := []string{
		"version", "uptime", "time", "pointer_size",
		"curr_items", "total_items", "bytes", "curr_connections",
		"total_connections", "get_hits", "get_misses", "evictions",
		"bytes_read", "bytes_written", "limit_maxbytes",
		"accepting_conns", "listen_disabled_num", "threads",
		"cmd_get", "cmd_set", "cmd_flush", "cmd_touch",
		"get_expired", "get_flushed", "touch_hits", "touch_misses",
	}
	
	for _, stat := range generalStats {
		if key == stat {
			return true
		}
	}
	return false
}

func (p *MemcachedPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	return false, errors.New("creating storage units is not supported for Memcached")
}

func (p *MemcachedPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	return false, errors.New("updating storage units is not supported for Memcached")
}

func (p *MemcachedPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	if storageUnit != "keys" {
		return false, errors.New("can only add rows to 'keys' storage unit")
	}
	
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to connect to Memcached for adding row")
		return false, err
	}
	
	var key, value string
	var flags, expiration int
	
	for _, record := range values {
		switch record.Key {
		case "key":
			key = record.Value
		case "value":
			value = record.Value
		case "flags":
			if f, err := strconv.Atoi(record.Value); err == nil {
				flags = f
			}
		case "expiration":
			if e, err := strconv.Atoi(record.Value); err == nil {
				expiration = e
			}
		}
	}
	
	if key == "" {
		return false, errors.New("key is required")
	}
	
	item := &memcache.Item{
		Key:        key,
		Value:      []byte(value),
		Flags:      uint32(flags),
		Expiration: int32(expiration),
	}
	
	err = client.Set(item)
	if err != nil {
		log.Logger.WithError(err).WithField("key", key).Error("Failed to set Memcached key")
		return false, err
	}
	
	return true, nil
}

func (p *MemcachedPlugin) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	if storageUnit != "keys" {
		return false, errors.New("can only delete rows from 'keys' storage unit")
	}
	
	key, ok := values["key"]
	if !ok {
		return false, errors.New("key is required for deletion")
	}
	
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to connect to Memcached for deleting row")
		return false, err
	}
	
	err = client.Delete(key)
	if err != nil && err != memcache.ErrCacheMiss {
		log.Logger.WithError(err).WithField("key", key).Error("Failed to delete Memcached key")
		return false, err
	}
	
	return true, nil
}

func (p *MemcachedPlugin) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
	return nil, errors.New("graph view is not supported for Memcached")
}

func (p *MemcachedPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to connect to Memcached for raw execute")
		return nil, err
	}
	
	// Parse simple memcached commands
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return nil, errors.New("empty command")
	}
	
	command := strings.ToLower(parts[0])
	switch command {
	case "get":
		if len(parts) < 2 {
			return nil, errors.New("get command requires a key")
		}
		key := parts[1]
		item, err := client.Get(key)
		if err != nil {
			if err == memcache.ErrCacheMiss {
				return &engine.GetRowsResult{
					Columns: []engine.Column{{Name: "result", Type: "string"}},
					Rows:    [][]string{{"NOT FOUND"}},
				}, nil
			}
			return nil, err
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{
				{Name: "key", Type: "string"},
				{Name: "value", Type: "string"},
				{Name: "flags", Type: "string"},
			},
			Rows: [][]string{
				{item.Key, string(item.Value), strconv.Itoa(int(item.Flags))},
			},
		}, nil
	
	case "set":
		if len(parts) < 3 {
			return nil, errors.New("set command requires key and value")
		}
		key := parts[1]
		value := strings.Join(parts[2:], " ")
		err := client.Set(&memcache.Item{
			Key:   key,
			Value: []byte(value),
		})
		if err != nil {
			return nil, err
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "result", Type: "string"}},
			Rows:    [][]string{{"STORED"}},
		}, nil
	
	case "delete":
		if len(parts) < 2 {
			return nil, errors.New("delete command requires a key")
		}
		key := parts[1]
		err := client.Delete(key)
		if err != nil && err != memcache.ErrCacheMiss {
			return nil, err
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "result", Type: "string"}},
			Rows:    [][]string{{"DELETED"}},
		}, nil
	
	case "flush_all":
		err := client.FlushAll()
		if err != nil {
			return nil, err
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "result", Type: "string"}},
			Rows:    [][]string{{"OK"}},
		}, nil
	
	case "stats":
		stats, err := client.Stats()
		if err != nil {
			return nil, err
		}
		rows := [][]string{}
		for server, serverStats := range stats {
			for key, value := range serverStats {
				rows = append(rows, []string{server, key, value})
			}
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{
				{Name: "server", Type: "string"},
				{Name: "stat", Type: "string"},
				{Name: "value", Type: "string"},
			},
			Rows: rows,
		}, nil
	
	default:
		return nil, fmt.Errorf("unsupported command: %s", command)
	}
}

func (p *MemcachedPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return nil, errors.ErrUnsupported
}

func (p *MemcachedPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error {
	return errors.New("export is not supported for Memcached")
}

func (p *MemcachedPlugin) FormatValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func (p *MemcachedPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	return make(map[string]map[string]any), nil
}

func (p *MemcachedPlugin) ClearTableData(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	if storageUnit == "keys" {
		client, err := DB(config)
		if err != nil {
			log.Logger.WithError(err).Error("Failed to connect to Memcached for clearing data")
			return false, err
		}
		err = client.FlushAll()
		if err != nil {
			log.Logger.WithError(err).Error("Failed to flush all Memcached data")
			return false, err
		}
		return true, nil
	}
	return false, errors.New("can only clear data from 'keys' storage unit")
}

func (p *MemcachedPlugin) WithTransaction(config *engine.PluginConfig, operation func(tx any) error) error {
	// Memcached doesn't support transactions
	return operation(nil)
}

func NewMemcachedPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Memcached,
		PluginFunctions: &MemcachedPlugin{},
	}
}