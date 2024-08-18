package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/go-redis/redis/v8"
)

type RedisPlugin struct{}

func (p *RedisPlugin) IsAvailable(config *engine.PluginConfig) bool {
	ctx := context.Background()
	client, err := DB(config)
	if err != nil {
		return false
	}
	defer client.Close()
	status := client.Ping(ctx)
	return status.Err() == nil
}

func (p *RedisPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.New("unsupported operation for Redis")
}

func (p *RedisPlugin) GetSchema(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *RedisPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	ctx := context.Background()

	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	keys, err := client.Keys(ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	pipe := client.Pipeline()
	cmds := make(map[string]*redis.StatusCmd, len(keys))

	for _, key := range keys {
		cmds[key] = pipe.Type(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	storageUnits := make([]engine.StorageUnit, 0, len(keys))
	for _, key := range keys {
		keyType, err := cmds[key].Result()
		if err != nil {
			return nil, err
		}

		var attributes []engine.Record
		switch keyType {
		case "string":
			sizeCmd := pipe.StrLen(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil {
				return nil, err
			}
			size, err := sizeCmd.Result()
			if err != nil {
				return nil, err
			}
			attributes = []engine.Record{
				{Key: "Type", Value: "string"},
				{Key: "Size", Value: fmt.Sprintf("%d", size)},
			}
		case "hash":
			sizeCmd := pipe.HLen(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil {
				return nil, err
			}
			size, err := sizeCmd.Result()
			if err != nil {
				return nil, err
			}
			attributes = []engine.Record{
				{Key: "Type", Value: "hash"},
				{Key: "Size", Value: fmt.Sprintf("%d", size)},
			}
		case "list":
			sizeCmd := pipe.LLen(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil {
				return nil, err
			}
			size, err := sizeCmd.Result()
			if err != nil {
				return nil, err
			}
			attributes = []engine.Record{
				{Key: "Type", Value: "list"},
				{Key: "Size", Value: fmt.Sprintf("%d", size)},
			}
		case "set":
			sizeCmd := pipe.SCard(ctx, key)
			if _, err := pipe.Exec(ctx); err != nil {
				return nil, err
			}
			size, err := sizeCmd.Result()
			if err != nil {
				return nil, err
			}
			attributes = []engine.Record{
				{Key: "Type", Value: "set"},
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

func (p *RedisPlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where string, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	ctx := context.Background()

	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var result *engine.GetRowsResult

	keyType, err := client.Type(ctx, storageUnit).Result()
	if err != nil {
		return nil, err
	}

	switch keyType {
	case "string":
		val, err := client.Get(ctx, storageUnit).Result()
		if err != nil {
			return nil, err
		}
		result = &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "value", Type: "string"}},
			Rows:    [][]string{{val}},
		}
	case "hash":
		hashValues, err := client.HGetAll(ctx, storageUnit).Result()
		if err != nil {
			return nil, err
		}
		rows := make([][]string, 0, len(hashValues))
		for field, value := range hashValues {
			rows = append(rows, []string{field, value})
		}
		result = &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "field", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:    rows,
		}
	case "list":
		listValues, err := client.LRange(ctx, storageUnit, 0, -1).Result()
		if err != nil {
			return nil, err
		}
		rows := make([][]string, 0, len(listValues))
		for i, value := range listValues {
			rows = append(rows, []string{strconv.Itoa(i), value})
		}
		result = &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "index", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:    rows,
		}
	case "set":
		setValues, err := client.SMembers(ctx, storageUnit).Result()
		if err != nil {
			return nil, err
		}
		rows := make([][]string, 0, len(setValues))
		for i, value := range setValues {
			rows = append(rows, []string{strconv.Itoa(i), value})
		}
		result = &engine.GetRowsResult{
			Columns:       []engine.Column{{Name: "index", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:          rows,
			DisableUpdate: true,
		}
	default:
		return nil, errors.New("unsupported Redis data type")
	}

	return result, nil
}

func (p *RedisPlugin) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
	return nil, errors.New("unsupported operation for Redis")
}

func (p *RedisPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return nil, errors.New("unsupported operation for Redis")
}

func (p *RedisPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return nil, errors.ErrUnsupported
}

func NewRedisPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Redis,
		PluginFunctions: &RedisPlugin{},
	}
}
