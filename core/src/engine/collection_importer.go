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

package engine

// DocumentImportFailure describes a single source record that could not be
// written to a collection. Index is the zero-based position of the record in
// the parsed source (file order).
type DocumentImportFailure struct {
	Index  int
	Reason string
}

// DocumentUpsertResult reports the outcome counts of an upsert import.
type DocumentUpsertResult struct {
	Matched  int
	Modified int
	Upserted int
}

// CollectionImporter is an optional plugin capability for importing documents
// into a document collection. Only document databases (for example MongoDB)
// implement it; resolvers type-assert PluginFunctions to this interface and
// report an unsupported error when the assertion fails.
//
// Both methods continue past per-document write errors rather than aborting,
// returning the records they skipped so the caller can report them.
type CollectionImporter interface {
	// InsertDocuments inserts the given documents into the collection, skipping
	// (not aborting on) individual documents that fail. It returns the number
	// inserted and the failures encountered.
	InsertDocuments(config *PluginConfig, schema string, collection string, documents []map[string]any) (int, []DocumentImportFailure, error)

	// UpsertDocuments replaces or inserts each document, matching existing
	// documents by keys. When keys is empty it defaults to ["_id"]. It skips
	// individual documents that fail and returns the outcome counts plus failures.
	UpsertDocuments(config *PluginConfig, schema string, collection string, keys []string, documents []map[string]any) (DocumentUpsertResult, []DocumentImportFailure, error)
}
