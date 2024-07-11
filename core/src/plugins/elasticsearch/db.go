package elasticsearch

import (
	"fmt"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/elastic/go-elasticsearch/v8"
)

func DB(config *engine.PluginConfig) (*elasticsearch.Client, error) {
	var addresses []string
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "9200")
	sslMode := common.GetRecordValueOrDefault(config.Credentials.Advanced, "SSL Mode", "disable")
	if sslMode == "enable" {
		addresses = []string{
			fmt.Sprintf("https://%s:%s", config.Credentials.Hostname, port),
		}
	} else {
		addresses = []string{
			fmt.Sprintf("http://%s:%s", config.Credentials.Hostname, port),
		}
	}

	cfg := elasticsearch.Config{
		Addresses: addresses,
		Username:  config.Credentials.Username,
		Password:  config.Credentials.Password,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	res, err := client.Info()
	if err != nil || res.IsError() {
		return nil, fmt.Errorf("error pinging Elasticsearch: %v", err)
	}

	return client, nil
}
