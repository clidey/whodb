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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/go-redis/redis/v8"
)

type RedisPlugin struct{}

var redisOperators = map[string]string{
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

func (p *RedisPlugin) IsAvailable(config *engine.PluginConfig) bool {
	ctx := context.Background()
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to connect to Redis for availability check")
		return false
	}
	defer client.Close()
	status := client.Ping(ctx)
	return status.Err() == nil
}

func (p *RedisPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	maxDatabases := 16
	var availableDatabases []string

	for i := 0; i < maxDatabases; i++ {
		dbConfig := *config
		dbConfig.Credentials.Database = strconv.Itoa(i)

		if p.IsAvailable(&dbConfig) {
			availableDatabases = append(availableDatabases, strconv.Itoa(i))
		}
	}

	if len(availableDatabases) == 0 {
		err := errors.New("no available databases found")
		log.Logger.WithError(err).Error("No Redis databases found during discovery")
		return nil, err
	}

	return availableDatabases, nil
}

func (p *RedisPlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *RedisPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	ctx := context.Background()

	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to connect to Redis for storage units retrieval")
		return nil, err
	}
	defer client.Close()

	var keys []string
	var cursor uint64

	for {
		var scanKeys []string
		scanKeys, cursor, err = client.Scan(ctx, cursor, "*", 0).Result() // count = 0 will use the redis default of 10
		if err != nil {
			log.Logger.WithError(err).Error("Failed to scan Redis keys")
			return nil, err
		}

		keys = append(keys, scanKeys...)

		// When cursor is 0, we've completed the full scan
		if cursor == 0 {
			break
		}
	}

	pipe := client.Pipeline()
	cmds := make(map[string]*redis.StatusCmd, len(keys))

	for _, key := range keys {
		cmds[key] = pipe.Type(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to execute Redis pipeline for key types")
		return nil, err
	}

	storageUnits := make([]engine.StorageUnit, 0, len(keys))
	for _, key := range keys {
		keyType, err := cmds[key].Result()
		if err != nil {
			log.Logger.WithError(err).WithField("key", key).Error("Failed to get Redis key type")
			return nil, err
		}

		var attributes []engine.Record
		switch keyType {
		case "string":
			sizeCmd := pipe.StrLen(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to execute pipeline for string key size")
				return nil, err
			}
			size, err := sizeCmd.Result()
			if err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to get string key size")
				return nil, err
			}
			attributes = []engine.Record{
				{Key: "Type", Value: "string"},
				{Key: "Size", Value: fmt.Sprintf("%d", size)},
			}
		case "hash":
			sizeCmd := pipe.HLen(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to execute pipeline for hash key size")
				return nil, err
			}
			size, err := sizeCmd.Result()
			if err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to get hash key size")
				return nil, err
			}
			attributes = []engine.Record{
				{Key: "Type", Value: "hash"},
				{Key: "Size", Value: fmt.Sprintf("%d", size)},
			}
		case "list":
			sizeCmd := pipe.LLen(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to execute pipeline for list key size")
				return nil, err
			}
			size, err := sizeCmd.Result()
			if err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to get list key size")
				return nil, err
			}
			attributes = []engine.Record{
				{Key: "Type", Value: "list"},
				{Key: "Size", Value: fmt.Sprintf("%d", size)},
			}
		case "set":
			sizeCmd := pipe.SCard(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to execute pipeline for set key size")
				return nil, err
			}
			size, err := sizeCmd.Result()
			if err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to get set key size")
				return nil, err
			}
			attributes = []engine.Record{
				{Key: "Type", Value: "set"},
				{Key: "Size", Value: fmt.Sprintf("%d", size)},
			}
		case "zset":
			sizeCmd := pipe.ZCard(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to execute pipeline for zset key size")
				return nil, err
			}
			size, err := sizeCmd.Result()
			if err != nil {
				log.Logger.WithError(err).WithField("key", key).Error("Failed to get zset key size")
				return nil, err
			}
			attributes = []engine.Record{
				{Key: "Type", Value: "zset"},
				{Key: "Size", Value: fmt.Sprintf("%d", size)},
			}
		default:
			attributes = []engine.Record{
				{Key: "Type", Value: "unknown"},
			}
		}

		storageUnits = append(storageUnits, engine.StorageUnit{
			Name:       key,
			Attributes: attributes,
		})
	}

	return storageUnits, nil
}

func (p *RedisPlugin) StorageUnitExists(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	ctx := context.Background()
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Close()

	exists, err := client.Exists(ctx, storageUnit).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (p *RedisPlugin) GetRows(
	config *engine.PluginConfig,
	schema, storageUnit string,
	where *model.WhereCondition,
	sortConditions []*model.SortCondition,
	pageSize, pageOffset int,
) (*engine.GetRowsResult, error) {
	ctx := context.Background()

	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to Redis for rows retrieval")
		return nil, err
	}
	defer client.Close()

	var result *engine.GetRowsResult

	keyType, err := client.Type(ctx, storageUnit).Result()
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Redis key type for rows retrieval")
		return nil, err
	}

	switch keyType {
	case "string":
		val, err := client.Get(ctx, storageUnit).Result()
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Redis string value")
			return nil, err
		}
		result = &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "value", Type: "string"}},
			Rows:    [][]string{{val}},
		}
	case "hash":
		hashValues, err := client.HGetAll(ctx, storageUnit).Result()
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Redis hash values")
			return nil, err
		}
		var rows [][]string
		for field, value := range hashValues {
			if where == nil || filterRedisHash(field, value, where) {
				rows = append(rows, []string{field, value})
			}
		}
		// Sort rows by field name (first column) alphabetically
		sort.Slice(rows, func(i, j int) bool {
			return rows[i][0] < rows[j][0]
		})
		result = &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "field", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:    rows,
		}
	case "list":
		listValues, err := client.LRange(ctx, storageUnit, 0, -1).Result()
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Redis list values")
			return nil, err
		}
		var rows [][]string
		for i, value := range listValues {
			if where == nil || filterRedisList(value, where) {
				rows = append(rows, []string{strconv.Itoa(i), value})
			}
		}
		result = &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "index", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:    rows,
		}
	case "set":
		setValues, err := client.SMembers(ctx, storageUnit).Result()
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Redis set values")
			return nil, err
		}
		var rows [][]string
		for i, value := range setValues {
			if where == nil || filterRedisSet(value, where) {
				rows = append(rows, []string{strconv.Itoa(i), value})
			}
		}
		result = &engine.GetRowsResult{
			Columns:       []engine.Column{{Name: "index", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:          rows,
			DisableUpdate: true,
		}
	case "zset":
		zsetValues, err := client.ZRangeWithScores(ctx, storageUnit, 0, -1).Result()
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Redis zset values")
			return nil, err
		}
		var rows [][]string
		for i, member := range zsetValues {
			value := member.Member.(string)
			scoreStr := fmt.Sprintf("%.2f", member.Score)
			if where == nil || filterRedisZSet(value, scoreStr, where) {
				rows = append(rows, []string{strconv.Itoa(i), value, scoreStr})
			}
		}
		result = &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "index", Type: "string"}, {Name: "member", Type: "string"}, {Name: "score", Type: "string"}},
			Rows:    rows,
		}
	default:
		err := errors.New("unsupported Redis data type")
		return nil, err
	}

	// Set TotalCount from the number of rows (Redis data is fully loaded)
	result.TotalCount = int64(len(result.Rows))

	return result, nil
}

func (p *RedisPlugin) GetRowCount(config *engine.PluginConfig, schema, storageUnit string, where *model.WhereCondition) (int64, error) {
	ctx := context.Background()

	client, err := DB(config)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	keyType, err := client.Type(ctx, storageUnit).Result()
	if err != nil {
		return 0, err
	}

	switch keyType {
	case "string":
		return 1, nil
	case "hash":
		count, err := client.HLen(ctx, storageUnit).Result()
		if err != nil {
			return 0, err
		}
		return count, nil
	case "list":
		count, err := client.LLen(ctx, storageUnit).Result()
		if err != nil {
			return 0, err
		}
		return count, nil
	case "set":
		count, err := client.SCard(ctx, storageUnit).Result()
		if err != nil {
			return 0, err
		}
		return count, nil
	case "zset":
		count, err := client.ZCard(ctx, storageUnit).Result()
		if err != nil {
			return 0, err
		}
		return count, nil
	default:
		return 0, errors.New("unsupported Redis data type")
	}
}

func (p *RedisPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	ctx := context.Background()

	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to Redis for columns retrieval")
		return nil, err
	}
	defer client.Close()

	keyType, err := client.Type(ctx, storageUnit).Result()
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Redis key type for columns retrieval")
		return nil, err
	}

	switch keyType {
	case "string":
		return []engine.Column{{Name: "value", Type: "string"}}, nil
	case "hash":
		return []engine.Column{{Name: "field", Type: "string"}, {Name: "value", Type: "string"}}, nil
	case "list":
		return []engine.Column{{Name: "index", Type: "string"}, {Name: "value", Type: "string"}}, nil
	case "set":
		return []engine.Column{{Name: "index", Type: "string"}, {Name: "value", Type: "string"}}, nil
	case "zset":
		return []engine.Column{{Name: "index", Type: "string"}, {Name: "member", Type: "string"}, {Name: "score", Type: "string"}}, nil
	default:
		return nil, errors.New("unsupported Redis data type")
	}
}

func filterRedisHash(field, value string, where *model.WhereCondition) bool {
	condition, err := convertWhereConditionToRedisFilter(where)
	if err != nil {
		return true // Ignore filtering on error
	}

	for key, op := range condition {
		if key == "field" {
			if !evaluateRedisCondition(field, op.Operator, op.Value) {
				return false
			}
		} else if key == "value" {
			if !evaluateRedisCondition(value, op.Operator, op.Value) {
				return false
			}
		}
	}
	return true
}

func filterRedisList(value string, where *model.WhereCondition) bool {
	condition, err := convertWhereConditionToRedisFilter(where)
	if err != nil {
		return true // Ignore filtering on error
	}

	for key, op := range condition {
		if key == "value" {
			if !evaluateRedisCondition(value, op.Operator, op.Value) {
				return false
			}
		}
	}
	return true
}

func filterRedisSet(value string, where *model.WhereCondition) bool {
	condition, err := convertWhereConditionToRedisFilter(where)
	if err != nil {
		return true
	}

	for key, op := range condition {
		if key == "value" || key == "member" {
			if !evaluateRedisCondition(value, op.Operator, op.Value) {
				return false
			}
		}
	}
	return true
}

func filterRedisZSet(member string, score string, where *model.WhereCondition) bool {
	condition, err := convertWhereConditionToRedisFilter(where)
	if err != nil {
		return true
	}

	for key, op := range condition {
		switch strings.ToLower(key) {
		case "member":
			if !evaluateRedisCondition(member, op.Operator, op.Value) {
				return false
			}
		case "score":
			if !evaluateRedisCondition(score, op.Operator, op.Value) {
				return false
			}
		}
	}
	return true
}

type redisFilter struct {
	Operator string
	Value    string
}

func (p *RedisPlugin) GetSupportedOperators() map[string]string {
	return redisOperators
}

func convertWhereConditionToRedisFilter(where *model.WhereCondition) (map[string]redisFilter, error) {
	if where == nil {
		return nil, nil
	}

	switch where.Type {
	case model.WhereConditionTypeAtomic:
		if where.Atomic == nil {
			return nil, fmt.Errorf("atomic condition must have an atomicwherecondition")
		}

		return map[string]redisFilter{
			where.Atomic.Key: {
				Operator: strings.ToUpper(where.Atomic.Operator),
				Value:    where.Atomic.Value,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported Redis filtering condition type: %v", where.Type)
	}
}

func evaluateRedisCondition(value, operator, target string) bool {
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
	}

	switch operator {
	default:
		return false
	}
}

func (p *RedisPlugin) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
	return nil, errors.ErrUnsupported
}

func (p *RedisPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return nil, errors.ErrUnsupported
}

func (p *RedisPlugin) RawExecuteWithParams(config *engine.PluginConfig, query string, params []any) (*engine.GetRowsResult, error) {
	return nil, errors.ErrUnsupported
}

func (p *RedisPlugin) Chat(config *engine.PluginConfig, schema string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return nil, errors.ErrUnsupported
}

func (p *RedisPlugin) FormatValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// GetColumnConstraints - not supported for Redis
func (p *RedisPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	return make(map[string]map[string]any), nil
}

// ClearTableData - not supported for Redis
func (p *RedisPlugin) ClearTableData(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	return false, errors.ErrUnsupported
}

// WithTransaction executes the operation directly since Redis doesn't support transactions in the same way as SQL databases
func (p *RedisPlugin) WithTransaction(config *engine.PluginConfig, operation func(tx any) error) error {
	// Redis doesn't support transactions in the same way as SQL databases
	// For now, just execute the operation directly
	return operation(nil)
}

func (p *RedisPlugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	return make(map[string]*engine.ForeignKeyRelationship), nil
}

// GetDatabaseMetadata returns Redis metadata for frontend configuration.
// Redis is a key-value store without traditional type definitions or operators.
func (p *RedisPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	ops := make([]string, 0, len(redisOperators))
	for op := range redisOperators {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return &engine.DatabaseMetadata{
		DatabaseType:    engine.DatabaseType_Redis,
		TypeDefinitions: []engine.TypeDefinition{},
		Operators:       ops,
		AliasMap:        map[string]string{},
	}
}

func NewRedisPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Redis,
		PluginFunctions: &RedisPlugin{},
	}
}
