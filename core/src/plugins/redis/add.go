package redis

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *RedisPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}

	ctx := context.Background()

	for fieldType, value := range fields {
		switch fieldType {
		case "string":
			if err := client.Set(ctx, storageUnit, value, 0).Err(); err != nil {
				return false, err
			}
		case "hash":
			var hashFields map[string]string
			if err := json.Unmarshal([]byte(value), &hashFields); err != nil {
				return false, err
			}
			hashFieldsInterface := make(map[string]interface{}, len(hashFields))
			for k, v := range hashFields {
				hashFieldsInterface[k] = v
			}
			if err := client.HMSet(ctx, storageUnit, hashFieldsInterface).Err(); err != nil {
				return false, err
			}
		case "list":
			var listValues []string
			if err := json.Unmarshal([]byte(value), &listValues); err != nil {
				return false, err
			}
			listValuesInterface := make([]interface{}, len(listValues))
			for i, v := range listValues {
				listValuesInterface[i] = v
			}
			if err := client.RPush(ctx, storageUnit, listValuesInterface...).Err(); err != nil {
				return false, err
			}
		case "set":
			var setValues []string
			if err := json.Unmarshal([]byte(value), &setValues); err != nil {
				return false, err
			}
			setValuesInterface := make([]interface{}, len(setValues))
			for i, v := range setValues {
				setValuesInterface[i] = v
			}
			if err := client.SAdd(ctx, storageUnit, setValuesInterface...).Err(); err != nil {
				return false, err
			}
		default:
			return false, errors.New("unsupported field type")
		}
	}

	return true, nil
}

func (p *RedisPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}

	if len(values) == 0 {
		return false, errors.New("no values provided to insert into the table")
	}

	ctx := context.Background()

	keyType, err := client.Type(ctx, storageUnit).Result()
	if err != nil {
		return false, err
	}

	switch keyType {
	case "hash":
		hashFieldsInterface := make(map[string]interface{}, len(values))
		for _, value := range values {
			hashFieldsInterface[value.Key] = value.Value
		}
		if err := client.HMSet(ctx, storageUnit, hashFieldsInterface).Err(); err != nil {
			return false, err
		}
	case "list":
		listValuesInterface := make([]interface{}, len(values))
		for _, value := range values {
			listValuesInterface = append(listValuesInterface, value.Value)
		}
		if err := client.RPush(ctx, storageUnit, listValuesInterface...).Err(); err != nil {
			return false, err
		}
	case "set":
		setValuesInterface := make([]interface{}, len(values))
		for _, value := range values {
			setValuesInterface = append(setValuesInterface, value.Value)
		}
		if err := client.SAdd(ctx, storageUnit, setValuesInterface...).Err(); err != nil {
			return false, err
		}
	default:
		return false, errors.New("unsupported storage unit type for adding rows")
	}

	return true, nil
}
