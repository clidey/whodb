/**
 * Copyright 2025 Clidey, Inc.
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

import Dagre from '@dagrejs/dagre';
import { Edge, Node } from 'reactflow';

type ILayoutOption = {
    direction: "TB" | "LR";
}

const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}));
export const getDagreLayoutedElements = (nodes: Node[] = [], edges: Edge[] = [], options: ILayoutOption = {
    direction: "LR",
}) => {
    g.setGraph({
        rankdir: options.direction,
        edgesep: 60,
        nodesep: 100,
        ranksep: 180,
        marginx: 80,
        marginy: 80,
        align: "UL",
        acyclicer: "greedy",
        ranker: "tight-tree",
    });

    edges.forEach((edge) => g.setEdge(edge.source, edge.target));
    
    // Ensure nodes have proper dimensions before layout
    nodes.forEach((node) => {
        const nodeWithDimensions = {
            ...node,
            width: node.width || 250,
            height: node.height || 120,
        };
        g.setNode(node.id, nodeWithDimensions as any);
    });

    Dagre.layout(g);

    return {
        nodes: nodes.map((node) => {
            const dagreNode = g.node(node.id);
            return { 
                ...node, 
                position: { 
                    x: dagreNode?.x || 0, 
                    y: dagreNode?.y || 0 
                } 
            };
        }),
        edges,
    };
};