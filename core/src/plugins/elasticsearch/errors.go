/*
 * // Copyright 2025 Clidey, Inc.
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package elasticsearch

import (
	"bytes"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// formatElasticError extracts a readable error message from an Elasticsearch response.
func formatElasticError(res *esapi.Response) string {
	if res == nil {
		return "unknown error"
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res.String()
	}
	// Restore body for potential callers who still read it
	res.Body = io.NopCloser(bytes.NewBuffer(body))
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		return res.String()
	}
	return msg
}
