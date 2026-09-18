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
	"os"
	"path/filepath"
	"testing"

	"github.com/clidey/whodb/cli/internal/platform"
)

func TestValidateTransformPlan(t *testing.T) {
	run := true
	valid := transformPlan{
		Kind: transformPlanKind,
		Transforms: []transformPlanEntry{{
			Name: "Load products", Ontology: "product", File: "products.parquet", WriteMode: "truncate_and_insert", Run: &run,
		}},
	}
	if err := validateTransformPlan(valid); err != nil {
		t.Fatalf("validateTransformPlan() error = %v", err)
	}

	duplicate := valid
	duplicate.Transforms = append(duplicate.Transforms, duplicate.Transforms[0])
	if err := validateTransformPlan(duplicate); err == nil {
		t.Fatal("validateTransformPlan() accepted duplicate names")
	}

	invalidMode := valid
	invalidMode.Transforms = append([]transformPlanEntry(nil), valid.Transforms...)
	invalidMode.Transforms[0].WriteMode = "drop_everything"
	if err := validateTransformPlan(invalidMode); err == nil {
		t.Fatal("validateTransformPlan() accepted unsupported write mode")
	}
}

func TestBuildFileOntologyGraph(t *testing.T) {
	raw, err := buildFileOntologyGraph("file-1", platform.Ontology{ID: "ontology-1", DisplayName: "Product"}, "truncate_and_insert")
	if err != nil {
		t.Fatalf("buildFileOntologyGraph() error = %v", err)
	}
	var graph struct {
		Nodes []struct {
			ID     string         `json:"id"`
			Type   string         `json:"type"`
			Config map[string]any `json:"config"`
		} `json:"nodes"`
		Edges []map[string]string `json:"edges"`
	}
	if err := json.Unmarshal([]byte(raw), &graph); err != nil {
		t.Fatalf("graph is not JSON: %v", err)
	}
	if len(graph.Nodes) != 2 || graph.Nodes[0].Type != "file" || graph.Nodes[1].Type != "ontology" {
		t.Fatalf("unexpected nodes: %#v", graph.Nodes)
	}
	if graph.Nodes[0].Config["fileId"] != "file-1" || graph.Nodes[1].Config["ontologyId"] != "ontology-1" {
		t.Fatalf("unexpected node config: %#v", graph.Nodes)
	}
	if len(graph.Edges) != 1 || graph.Edges[0]["source"] != "file" || graph.Edges[0]["target"] != "ontology" {
		t.Fatalf("unexpected edges: %#v", graph.Edges)
	}
}

func TestPreviewTransformApplyAction(t *testing.T) {
	tests := []struct {
		name                    string
		file, transform, update bool
		run                     bool
		want                    string
	}{
		{"new", false, false, false, true, "upload+create+run"},
		{"resume", true, true, false, true, "reuse+run"},
		{"overwrite", true, true, true, false, "upload+update"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := previewTransformApplyAction(test.file, test.transform, test.update, test.run); got != test.want {
				t.Fatalf("previewTransformApplyAction() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestTransformRunStatusClassification(t *testing.T) {
	for _, status := range []string{"completed", "succeeded", "success", "failed", "cancelled", "canceled"} {
		if !isTerminalTransformRunStatus(status) {
			t.Errorf("isTerminalTransformRunStatus(%q) = false", status)
		}
	}
	for _, status := range []string{"queued", "pending", "running", ""} {
		if isTerminalTransformRunStatus(status) {
			t.Errorf("isTerminalTransformRunStatus(%q) = true", status)
		}
	}
	if !isFailedTransformRunStatus("failed") || isFailedTransformRunStatus("completed") {
		t.Fatal("failed status classification is incorrect")
	}
}

func TestReadTransformCatalogsSupportsSeedAndAnalyticsShapes(t *testing.T) {
	dir := t.TempDir()
	seedPath := filepath.Join(dir, "seed.json")
	analyticsPath := filepath.Join(dir, "analytics.json")
	if err := os.WriteFile(seedPath, []byte(`{"files":[{"api_name":"product","file":"product.parquet"}],"reference_files":[{"api_name":"store","file":"store.parquet"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(analyticsPath, []byte(`{"ontologies":[{"apiName":"sales_analytics","source":"sales.parquet"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	originalPrefix := transformApplyNamePrefix
	transformApplyNamePrefix = "RetailOS Fixture"
	t.Cleanup(func() { transformApplyNamePrefix = originalPrefix })
	plan, err := readTransformCatalogs([]string{seedPath, analyticsPath}, []platform.Ontology{
		{APIName: "sales_analytics", DisplayName: "Sales Analytics"},
		{APIName: "product", DisplayName: "Product"},
		{APIName: "unmapped", DisplayName: "Unmapped"},
		{APIName: "store", DisplayName: "Store"},
	})
	if err != nil {
		t.Fatalf("readTransformCatalogs() error = %v", err)
	}
	if len(plan.Transforms) != 3 {
		t.Fatalf("transform count = %d, want 3", len(plan.Transforms))
	}
	if plan.Transforms[0].Ontology != "product" || plan.Transforms[0].Name != "RetailOS Fixture Product" {
		t.Fatalf("first transform = %#v", plan.Transforms[0])
	}
	if plan.Transforms[1].File != filepath.Join(dir, "sales.parquet") {
		t.Fatalf("analytics file = %q", plan.Transforms[1].File)
	}
}
