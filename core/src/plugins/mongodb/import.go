/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mongodb

import (
	"context"
	"errors"

	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBPlugin provides document imports for Collection File Import.
var _ engine.CollectionImporter = (*MongoDBPlugin)(nil)

const documentImportBatchSize = 1000

// InsertDocuments inserts documents into the collection using unordered bulk
// writes, so a document that fails (for example a duplicate _id) is skipped and
// reported rather than aborting the whole import. It implements
// engine.CollectionImporter.
func (p *MongoDBPlugin) InsertDocuments(config *engine.PluginConfig, schema string, collection string, documents []map[string]any) (int, []engine.DocumentImportFailure, error) {
	if len(documents) == 0 {
		return 0, nil, nil
	}

	client, err := DB(config)
	if err != nil {
		return 0, nil, err
	}
	defer disconnectClient(client)

	coll := client.Database(schema).Collection(collection)

	inserted := 0
	var failures []engine.DocumentImportFailure
	for start := 0; start < len(documents); start += documentImportBatchSize {
		end := start + documentImportBatchSize
		if end > len(documents) {
			end = len(documents)
		}

		models := make([]mongo.WriteModel, 0, end-start)
		for _, doc := range documents[start:end] {
			models = append(models, mongo.NewInsertOneModel().SetDocument(normalizeDocumentID(doc)))
		}

		result, batchFailures, err := bulkWriteUnordered(coll, models, start)
		if err != nil {
			return inserted, failures, err
		}
		inserted += int(result.InsertedCount)
		failures = append(failures, batchFailures...)
	}

	return inserted, failures, nil
}

// UpsertDocuments replaces or inserts each document, matching existing documents
// by keys (defaulting to ["_id"]). Documents missing a key are inserted instead
// of matched, so a missing key never produces an empty filter. Per-document
// failures are skipped and reported. It implements engine.CollectionImporter.
func (p *MongoDBPlugin) UpsertDocuments(config *engine.PluginConfig, schema string, collection string, keys []string, documents []map[string]any) (engine.DocumentUpsertResult, []engine.DocumentImportFailure, error) {
	var outcome engine.DocumentUpsertResult
	if len(documents) == 0 {
		return outcome, nil, nil
	}
	if len(keys) == 0 {
		keys = []string{"_id"}
	}

	client, err := DB(config)
	if err != nil {
		return outcome, nil, err
	}
	defer disconnectClient(client)

	coll := client.Database(schema).Collection(collection)

	var failures []engine.DocumentImportFailure
	for start := 0; start < len(documents); start += documentImportBatchSize {
		end := start + documentImportBatchSize
		if end > len(documents) {
			end = len(documents)
		}

		models := make([]mongo.WriteModel, 0, end-start)
		for _, doc := range documents[start:end] {
			normalized := normalizeDocumentID(doc)
			if filter, ok := upsertFilter(normalized, keys); ok {
				models = append(models, mongo.NewReplaceOneModel().SetFilter(filter).SetReplacement(normalized).SetUpsert(true))
			} else {
				models = append(models, mongo.NewInsertOneModel().SetDocument(normalized))
			}
		}

		result, batchFailures, err := bulkWriteUnordered(coll, models, start)
		if err != nil {
			return outcome, failures, err
		}
		outcome.Matched += int(result.MatchedCount)
		outcome.Modified += int(result.ModifiedCount)
		outcome.Upserted += int(result.UpsertedCount) + int(result.InsertedCount)
		failures = append(failures, batchFailures...)
	}

	return outcome, failures, nil
}

// bulkWriteUnordered runs an unordered bulk write and treats per-document write
// errors as skipped records rather than a fatal error. offset is added to each
// write-error index so failures map to positions in the full document slice.
func bulkWriteUnordered(coll *mongo.Collection, models []mongo.WriteModel, offset int) (*mongo.BulkWriteResult, []engine.DocumentImportFailure, error) {
	result, err := coll.BulkWrite(context.Background(), models, options.BulkWrite().SetOrdered(false))
	if err != nil {
		var bwe mongo.BulkWriteException
		if errors.As(err, &bwe) {
			failures := make([]engine.DocumentImportFailure, 0, len(bwe.WriteErrors))
			for _, we := range bwe.WriteErrors {
				failures = append(failures, engine.DocumentImportFailure{
					Index:  offset + we.Index,
					Reason: we.Message,
				})
			}
			return result, failures, nil
		}
		return result, nil, handleMongoError(err)
	}
	return result, nil, nil
}

// normalizeDocumentID converts a string _id that is a valid hex ObjectID into an
// ObjectID, matching how documents added through the editor are stored. Values
// already of a non-string type (such as an Extended JSON ObjectID) are left
// unchanged. The document is modified in place and returned for convenience.
func normalizeDocumentID(doc map[string]any) map[string]any {
	if rawID, ok := doc["_id"]; ok {
		if id, err := normalizeMongoID(rawID); err == nil {
			doc["_id"] = id
		}
	}
	return doc
}

// upsertFilter builds an equality filter from keys. It returns ok=false when any
// key is absent from the document, signalling the caller to insert the document
// instead of matching an existing one (avoiding an empty or partial filter).
func upsertFilter(doc map[string]any, keys []string) (bson.D, bool) {
	filter := make(bson.D, 0, len(keys))
	for _, key := range keys {
		value, ok := doc[key]
		if !ok {
			return nil, false
		}
		filter = append(filter, bson.E{Key: key, Value: value})
	}
	return filter, true
}
