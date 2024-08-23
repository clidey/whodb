package memcached

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/clidey/whodb/core/src/engine"
)

type MemcachedPlugin struct{}

func (p *MemcachedPlugin) IsAvailable(config *engine.PluginConfig) bool {
	client, err := DB(config)
	if err != nil {
		return false
	}
	_, err = client.Get("test_key")
	if err != nil && err != memcache.ErrCacheMiss {
		return false
	}
	return true
}

func (p *MemcachedPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.New("unsupported operation for Memcached")
}

func (p *MemcachedPlugin) GetSchema(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.New("unsupported operation for Memcached")
}

func (p *MemcachedPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	rows, err := p.GetRows(config, schema, "", "", 0, 0)
	if err != nil {
		return nil, err
	}

	count := len(rows.Rows)

	return []engine.StorageUnit{
		{
			Name: "default",
			Attributes: []engine.Record{
				{
					Key:   "Count",
					Value: fmt.Sprintf("%v", count),
				},
			},
		},
	}, nil
}

func (p *MemcachedPlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where string, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	conn, err := net.Dial("tcp", getAddress(config))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Memcached server: %v", err)
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	slabs := make(map[int]int)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "END" {
			break
		}

		if strings.HasPrefix(line, "STAT items") {
			parts := strings.Split(line, ":")
			if len(parts) > 2 {
				slabID := 0
				number := 0
				fmt.Sscanf(parts[1], "%d", &slabID)
				fmt.Sscanf(parts[2], "number %d", &number)
				slabs[slabID] = number
			}
		}
	}

	rows := [][]string{}

	for range slabs {
		for scanner.Scan() {
			line := scanner.Text()
			if line == "END" {
				break
			}

			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				key := parts[1]

				item, err := client.Get(key)
				if err != nil {
					return nil, fmt.Errorf("failed to retrieve value for key %s: %v", key, err)
				}
				rows = append(rows, []string{string(item.Value)})
			}
		}
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("error reading from connection: %v", scanner.Err())
	}

	return &engine.GetRowsResult{
		Columns: []engine.Column{
			{
				Name: "Value",
				Type: "string",
			},
		},
		Rows: rows,
	}, nil
}

func (p *MemcachedPlugin) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
	return nil, errors.New("unsupported operation for Memcached")
}

func (p *MemcachedPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return nil, errors.New("unsupported operation for Memcached")
}

func (p *MemcachedPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return nil, errors.New("unsupported operation for Memcached")
}

func NewMemcachedPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Memcached,
		PluginFunctions: &MemcachedPlugin{},
	}
}
