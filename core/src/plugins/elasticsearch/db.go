package elasticsearch

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/elastic/go-elasticsearch/v8"
)

func DB(config *engine.PluginConfig) (*elasticsearch.Client, error) {
	var addresses []string
	if config.Credentials.Hostname == "localhost" || config.Credentials.Hostname == "host.docker.internal" {
		addresses = []string{
			fmt.Sprintf("http://%s:%d", config.Credentials.Hostname, 9200),
		}
	} else {
		addresses = []string{
			fmt.Sprintf("https://%s:%d", config.Credentials.Hostname, 443),
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
