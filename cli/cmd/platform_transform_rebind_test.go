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

package cmd

import (
	"encoding/json"
	"testing"
)

func TestValidateTransformRebindFlags(t *testing.T) {
	originalFileID := transformRebindFileID
	originalFilePath := transformRebindFilePath
	originalSourceID := transformRebindSourceID
	originalSourcePath := transformRebindSourcePath
	t.Cleanup(func() {
		transformRebindFileID = originalFileID
		transformRebindFilePath = originalFilePath
		transformRebindSourceID = originalSourceID
		transformRebindSourcePath = originalSourcePath
	})

	tests := []struct {
		name       string
		fileID     string
		filePath   string
		sourceID   string
		sourcePath []string
		wantError  bool
	}{
		{name: "file id", fileID: "file-1"},
		{name: "local file", filePath: "products.csv"},
		{name: "database source", sourceID: "source-1", sourcePath: []string{"public", "products"}},
		{name: "missing binding", wantError: true},
		{name: "mixed bindings", fileID: "file-1", sourceID: "source-1", sourcePath: []string{"products"}, wantError: true},
		{name: "partial database binding", sourceID: "source-1", wantError: true},
		{name: "two file bindings", fileID: "file-1", filePath: "products.csv", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transformRebindFileID = test.fileID
			transformRebindFilePath = test.filePath
			transformRebindSourceID = test.sourceID
			transformRebindSourcePath = test.sourcePath
			err := validateTransformRebindFlags()
			if (err != nil) != test.wantError {
				t.Fatalf("validateTransformRebindFlags() error = %v, wantError %v", err, test.wantError)
			}
		})
	}
}

func TestRebindTransformGraph(t *testing.T) {
	raw := `{"nodes":[{"id":"input","type":"file","config":{"fileId":"old"}},{"id":"output","type":"ontology","config":{"ontologyId":"onto-1"}}],"edges":[{"source":"input","target":"output"}]}`
	binding := transformSourceBinding{NodeType: "source", Config: map[string]any{"sourceId": "source-1"}}
	updated, nodeID, err := rebindTransformGraph(raw, "", binding)
	if err != nil {
		t.Fatalf("rebindTransformGraph() error = %v", err)
	}
	if nodeID != "input" {
		t.Fatalf("nodeID = %q, want input", nodeID)
	}
	var graph struct {
		Nodes []struct {
			ID     string         `json:"id"`
			Type   string         `json:"type"`
			Config map[string]any `json:"config"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(updated), &graph); err != nil {
		t.Fatalf("decode updated graph: %v", err)
	}
	if graph.Nodes[0].Type != "source" || graph.Nodes[0].Config["sourceId"] != "source-1" {
		t.Fatalf("source node = %#v", graph.Nodes[0])
	}
	if graph.Nodes[1].Type != "ontology" {
		t.Fatalf("output node was modified: %#v", graph.Nodes[1])
	}
}

func TestRebindTransformGraphRejectsUnknownNode(t *testing.T) {
	raw := `{"nodes":[{"id":"input","type":"file","config":{}}]}`
	_, _, err := rebindTransformGraph(raw, "missing", transformSourceBinding{NodeType: "file", Config: map[string]any{"fileId": "new"}})
	if err == nil {
		t.Fatal("rebindTransformGraph() error = nil, want unknown node error")
	}
}
