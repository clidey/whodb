package mongodb

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/bson"
)

func (p *MongoDBPlugin) UpdateStorageUnit(config *engine.PluginConfig, database string, storageUnit string, values map[string]string) (bool, error) {
	ctx := context.Background()
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Disconnect(ctx)

	db := client.Database(database)
	collection := db.Collection(storageUnit)

	var jsonValues bson.M
	if err := json.Unmarshal([]byte(values["document"]), &jsonValues); err != nil {
		return false, err
	}

	id, ok := jsonValues["_id"]
	if !ok {
		return false, errors.New("missing '_id' field in the document")
	}

	filter := bson.M{"_id": id}
	update := bson.M{"$set": jsonValues}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, err
	}

	if result.MatchedCount == 0 {
		return false, errors.New("no documents were updated")
	}

	return true, nil
}
