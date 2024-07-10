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
	queryParams := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Query Params", "")
	var connectionString string
	// TODO: add TLS enabled logic to work instead of hard coded domains
	if config.Credentials.Hostname == "localhost" || config.Credentials.Hostname == "host.docker.internal" {
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
