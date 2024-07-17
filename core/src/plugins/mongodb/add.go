package mongodb

import (
	"context"
	"errors"

	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/mongo"
)

func (p *MongoDBPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Disconnect(context.Background())

	database := client.Database(schema)

	err = createCollectionIfNotExists(database, storageUnit)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (p *MongoDBPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(schema).Collection(storageUnit)

	document := make(map[string]interface{})
	for k, v := range values {
		document[k] = v
	}

	_, err = collection.InsertOne(context.Background(), document)
	if err != nil {
		return false, err
	}

	return true, nil
}

func createCollectionIfNotExists(database *mongo.Database, collectionName string) error {
	collections, err := database.ListCollectionNames(context.Background(), nil)
	if err != nil {
		return err
	}

	for _, col := range collections {
		if col == collectionName {
			return errors.New("collection already exists")
		}
	}

	err = database.CreateCollection(context.Background(), collectionName)
	if err != nil {
		return err
	}

	return nil
}
