package mongodb

import (
	"context"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DB(config *engine.PluginConfig) (*mongo.Client, error) {
	ctx := context.Background()
	var connectionString string
	if config.Credentials.Hostname == "localhost" {
		connectionString = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
			config.Credentials.Username,
			config.Credentials.Password,
			config.Credentials.Hostname,
			27017,
			config.Credentials.Database)
	} else {
		connectionString = fmt.Sprintf("mongodb+srv://%s:%s@%s/%s",
			config.Credentials.Username,
			config.Credentials.Password,
			config.Credentials.Hostname,
			config.Credentials.Database)
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
