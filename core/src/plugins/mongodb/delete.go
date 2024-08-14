package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (p *MongoDBPlugin) DeleteRow(config *engine.PluginConfig, database string, storageUnit string, values map[string]string) (bool, error) {
	ctx := context.Background()
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Disconnect(ctx)

	db := client.Database(database)
	collection := db.Collection(storageUnit)

	documentJSON, ok := values["document"]
	if !ok {
		return false, errors.New("missing 'document' key in values map")
	}

	var jsonValues bson.M
	if err := json.Unmarshal([]byte(documentJSON), &jsonValues); err != nil {
		return false, err
	}

	id, ok := jsonValues["_id"]
	if !ok {
		return false, errors.New("missing '_id' field in the document")
	}

	objectID, err := primitive.ObjectIDFromHex(id.(string))
	if err != nil {
		return false, errors.New("invalid '_id' field; not a valid ObjectID")
	}

	delete(jsonValues, "_id")

	filter := bson.M{"_id": objectID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return false, err
	}

	if result.DeletedCount == 0 {
		return false, errors.New("no documents were deleted")
	}

	return true, nil
}
