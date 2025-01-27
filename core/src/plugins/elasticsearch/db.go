package elasticsearch

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/elastic/go-elasticsearch/v8"
)

func DB(config *engine.PluginConfig) (*elasticsearch.Client, error) {
	var addresses []string
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "9200"))
	if err != nil {
		return nil, err
	}
	sslMode := common.GetRecordValueOrDefault(config.Credentials.Advanced, "SSL Mode", "disable")

	hostName := url.QueryEscape(config.Credentials.Hostname)

	scheme := "https"
	if sslMode == "disable" {
		scheme = "http"
	}

	addressUrl := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(hostName, strconv.Itoa(port)),
	}

	addresses = []string{
		addressUrl.String(),
	}

	cfg := elasticsearch.Config{
		Addresses: addresses,
		Username:  url.QueryEscape(config.Credentials.Username),
		Password:  url.QueryEscape(config.Credentials.Password),
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
