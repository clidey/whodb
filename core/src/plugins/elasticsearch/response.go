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

package elasticsearch

import "errors"

// errInvalidResponse indicates an Elasticsearch response did not have the
// structure the plugin expects. Elasticsearch/OpenSearch can return
// error-shaped or otherwise unexpected bodies that still parse as JSON, so
// response fields must be accessed defensively rather than via bare type
// assertions that would panic.
var errInvalidResponse = errors.New("invalid Elasticsearch response structure")

// responseHits extracts the hits.hits array from a decoded search response.
// It returns errInvalidResponse if either level is missing or not the
// expected type.
func responseHits(searchResult map[string]any) ([]any, error) {
	hitsObj, ok := searchResult["hits"].(map[string]any)
	if !ok {
		return nil, errInvalidResponse
	}
	hits, ok := hitsObj["hits"].([]any)
	if !ok {
		return nil, errInvalidResponse
	}
	return hits, nil
}

// hitSource extracts the _source object from a single hit. It returns false
// when the hit or its _source is missing or not an object, so callers can skip
// malformed hits instead of panicking.
func hitSource(hit any) (map[string]any, bool) {
	hitMap, ok := hit.(map[string]any)
	if !ok {
		return nil, false
	}
	source, ok := hitMap["_source"].(map[string]any)
	if !ok {
		return nil, false
	}
	return source, true
}

// indicesStats extracts the indices object from a decoded stats response.
func indicesStats(stats map[string]any) (map[string]any, error) {
	indices, ok := stats["indices"].(map[string]any)
	if !ok {
		return nil, errInvalidResponse
	}
	return indices, nil
}
