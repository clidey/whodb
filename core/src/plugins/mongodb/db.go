package mongodb

import (
	"context"
	"fmt"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"strconv"
	"strings"
)

func DB(config *engine.PluginConfig) (*mongo.Client, error) {
	ctx := context.Background()
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "27017"))
	if err != nil {
		return nil, err
	}
	queryParams := common.GetRecordValueOrDefault(config.Credentials.Advanced, "URL Params", "")
	dnsEnabled := common.GetRecordValueOrDefault(config.Credentials.Advanced, "DNS Enabled", "false")

	connectionURI := strings.Builder{}
	clientOptions := options.Client()

	if strings.ToLower(dnsEnabled) == "true" {
		connectionURI.WriteString("mongodb+srv://")
		connectionURI.WriteString(fmt.Sprintf("%s/", config.Credentials.Hostname))
		connectionURI.WriteString(config.Credentials.Database)
		connectionURI.WriteString(queryParams)
	} else {
		connectionURI.WriteString("mongodb://")
		connectionURI.WriteString(fmt.Sprintf("%s:%d/", config.Credentials.Hostname, port))
		connectionURI.WriteString(config.Credentials.Database)
		connectionURI.WriteString(queryParams)
	}

	clientOptions.ApplyURI(connectionURI.String())
	clientOptions.SetAuth(options.Credential{
		Username: url.QueryEscape(config.Credentials.Username),
		Password: url.QueryEscape(config.Credentials.Password),
	})

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}
