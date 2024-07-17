package mongodb

import (
	"context"
	"fmt"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DB(config *engine.PluginConfig) (*mongo.Client, error) {
	ctx := context.Background()
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "27017")
	queryParams := common.GetRecordValueOrDefault(config.Credentials.Advanced, "URL Params", "")
	dnsEnabled := common.GetRecordValueOrDefault(config.Credentials.Advanced, "DNS Enabled", "false")
	var connectionString string
	if dnsEnabled == "false" {
		connectionString = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s%s",
			config.Credentials.Username,
			config.Credentials.Password,
			config.Credentials.Hostname,
			port,
			config.Credentials.Database,
			queryParams)
	} else {
		connectionString = fmt.Sprintf("mongodb+srv://%s:%s@%s/%s%s",
			config.Credentials.Username,
			config.Credentials.Password,
			config.Credentials.Hostname,
			config.Credentials.Database,
			queryParams)
	}

	clientOptions := options.Client().ApplyURI(connectionString)
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
