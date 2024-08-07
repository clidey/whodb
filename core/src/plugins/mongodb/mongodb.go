package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBPlugin struct{}

func (p *MongoDBPlugin) IsAvailable(config *engine.PluginConfig) bool {
	client, err := DB(config)
	if err != nil {
		return false
	}
	defer client.Disconnect(context.TODO())
	return true
}

func (p *MongoDBPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *MongoDBPlugin) GetSchema(config *engine.PluginConfig) ([]string, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	databases, err := client.ListDatabaseNames(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	return databases, nil
}

func (p *MongoDBPlugin) GetStorageUnits(config *engine.PluginConfig, database string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	db := client.Database(database)
	collections, err := db.ListCollectionNames(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}

	storageUnits := []engine.StorageUnit{}
	for _, collectionName := range collections {
		stats := bson.M{}
		err := db.RunCommand(context.TODO(), bson.D{{Key: "collStats", Value: collectionName}}).Decode(&stats)
		if err != nil {
			return nil, err
		}

		storageUnits = append(storageUnits, engine.StorageUnit{
			Name: collectionName,
			Attributes: []engine.Record{
				{Key: "Storage Size", Value: fmt.Sprintf("%v", stats["storageSize"])},
				{Key: "Count", Value: fmt.Sprintf("%v", stats["count"])},
			},
		})
	}
	return storageUnits, nil
}

func (p *MongoDBPlugin) GetRows(config *engine.PluginConfig, database, collection, filter string, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	db := client.Database(database)
	coll := db.Collection(collection)

	var bsonFilter bson.M
	if len(filter) > 0 {
		if err := bson.UnmarshalExtJSON([]byte(filter), true, &bsonFilter); err != nil {
			return nil, fmt.Errorf("invalid filter format: %v", err)
		}
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(pageSize))
	findOptions.SetSkip(int64(pageOffset))

	cursor, err := coll.Find(context.TODO(), bsonFilter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var rowsResult []bson.M
	if err = cursor.All(context.TODO(), &rowsResult); err != nil {
		return nil, err
	}

	result := &engine.GetRowsResult{
		Columns: []engine.Column{
			{
				Name: "document",
				Type: "Document",
			},
		},
		Rows: [][]string{},
	}

	for _, doc := range rowsResult {
		jsonBytes, err := json.Marshal(doc)
		if err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, []string{
			string(jsonBytes),
		})
	}

	return result, nil
}

func (p *MongoDBPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return nil, errors.ErrUnsupported
}

func NewMongoDBPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_MongoDB,
		PluginFunctions: &MongoDBPlugin{},
	}
}
