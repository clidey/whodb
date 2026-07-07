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
	"context"

	"github.com/clidey/whodb/cli/internal/platform"
)

type platformDescribeRelated struct {
	Upstream   []platform.LineageNode `json:"upstream,omitempty"`
	Downstream []platform.LineageNode `json:"downstream,omitempty"`
}

type platformOntologyDescribe struct {
	platform.Ontology
	Related platformDescribeRelated `json:"related,omitempty"`
}

type platformDatasetDescribe struct {
	platform.Dataset
	Related platformDescribeRelated `json:"related,omitempty"`
}

type platformTransformDescribe struct {
	platform.Transform
	Related platformDescribeRelated `json:"related,omitempty"`
}

type platformFunctionDescribe struct {
	platform.Function
	Related platformDescribeRelated `json:"related,omitempty"`
}

type platformFileDescribe struct {
	platform.ProjectFile
	Related platformDescribeRelated `json:"related,omitempty"`
}

type platformSecretDescribe struct {
	platform.ProjectSecret
	Related platformDescribeRelated `json:"related,omitempty"`
}

type platformAIProviderDescribe struct {
	platform.AIProvider
	Related platformDescribeRelated `json:"related,omitempty"`
}

type platformFolderDescribe struct {
	platform.ProjectFolder
	Related platformDescribeRelated `json:"related,omitempty"`
}

func platformRelatedLineage(ctx context.Context, session *platformSession, projectID, id, nodeType string) platformDescribeRelated {
	graph, err := session.Client.ProjectLineage(ctx, projectID)
	if err != nil || graph == nil {
		return platformDescribeRelated{}
	}
	nodes := make(map[string]platform.LineageNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.NodeType+":"+node.ID] = node
	}
	related := platformDescribeRelated{}
	for _, edge := range graph.Edges {
		if edge.TargetID == id && edge.TargetType == nodeType {
			if node, ok := nodes[edge.SourceType+":"+edge.SourceID]; ok {
				related.Upstream = append(related.Upstream, node)
			}
		}
		if edge.SourceID == id && edge.SourceType == nodeType {
			if node, ok := nodes[edge.TargetType+":"+edge.TargetID]; ok {
				related.Downstream = append(related.Downstream, node)
			}
		}
	}
	return related
}
