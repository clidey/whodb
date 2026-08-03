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
import type { Edge, Node } from 'reactflow';

type ILayoutOption = {
    direction: "TB" | "LR";
}

const PIPELINE_NODE_WIDTH = 208;
const PIPELINE_NODE_HEIGHT = 76;
const PIPELINE_BRANCH_GAP = 72;

function nodeWidth(node: Node): number {
    return node.width ?? (node.data?.nodeType ? PIPELINE_NODE_WIDTH : 400);
}

function nodeHeight(node: Node): number {
    return node.height ?? (node.data?.nodeType ? PIPELINE_NODE_HEIGHT : 200);
}

function isPipelineGraph(nodes: Node[]): boolean {
    return nodes.length > 0 && nodes.every(node => Boolean(node.data?.nodeType));
}

function sideBranchDirection(handle: string | null | undefined): -1 | 1 | null {
    if (handle === 'left' || handle === 'match') return -1;
    if (handle === 'right' || handle === 'otherwise') return 1;
    return null;
}

function positionPipelineTopology(nodes: Node[], edges: Edge[], direction: ILayoutOption['direction']): Node[] {
    if (!isPipelineGraph(nodes)) return nodes;

    const nodeByID = new Map(nodes.map(node => [node.id, node]));
    const positioned = new Map(nodes.map(node => [node.id, node]));

    const incoming = new Map<string, Edge[]>();
    const outgoing = new Map<string, Edge[]>();
    for (const edge of edges) {
        if (!nodeByID.has(edge.source) || !nodeByID.has(edge.target)) continue;
        incoming.set(edge.target, [...(incoming.get(edge.target) ?? []), edge]);
        outgoing.set(edge.source, [...(outgoing.get(edge.source) ?? []), edge]);
    }

    for (const target of nodes) {
        const inputs = incoming.get(target.id) ?? [];
        if (inputs.length < 2) continue;

        const sources = inputs
            .map(edge => nodeByID.get(edge.source))
            .filter((node): node is Node => node !== undefined)
            .filter(source => (outgoing.get(source.id) ?? []).length === 1);
        if (sources.length < 2) continue;

        const targetWidth = nodeWidth(target);
        const targetHeight = nodeHeight(target);
        const targetCenterX = target.position.x + targetWidth / 2;
        const targetCenterY = target.position.y + targetHeight / 2;
        const widestSource = Math.max(...sources.map(nodeWidth));

        sources.forEach((source, index) => {
            const sourceWidth = nodeWidth(source);
            const sourceHeight = nodeHeight(source);
            const offset = (index - (sources.length - 1) / 2) * (widestSource + PIPELINE_BRANCH_GAP);
            positioned.set(source.id, {
                ...source,
                position: direction === 'TB'
                    ? {
                        x: targetCenterX + offset - sourceWidth / 2,
                        y: target.position.y - PIPELINE_BRANCH_GAP - sourceHeight,
                    }
                    : {
                        x: target.position.x - PIPELINE_BRANCH_GAP - sourceWidth,
                        y: targetCenterY + offset - sourceHeight / 2,
                    },
            });
        });
    }

    for (const source of nodes) {
        const branches = outgoing.get(source.id) ?? [];
        if (branches.length < 2) continue;

        const targets = branches
            .map(edge => ({ edge, node: nodeByID.get(edge.target) }))
            .filter((branch): branch is { edge: Edge; node: Node } => branch.node !== undefined);
        if (targets.length < 2) continue;

        const sideBranches = targets
            .map(branch => ({ ...branch, direction: sideBranchDirection(branch.edge.sourceHandle) }))
            .filter((branch): branch is { edge: Edge; node: Node; direction: -1 | 1 } => branch.direction !== null);
        const sourceWidth = nodeWidth(source);
        const sourceHeight = nodeHeight(source);
        const sourceCenterX = source.position.x + sourceWidth / 2;
        const sourceCenterY = source.position.y + sourceHeight / 2;

        if (sideBranches.length === targets.length) {
            sideBranches.forEach(branch => {
                const targetWidth = nodeWidth(branch.node);
                const targetHeight = nodeHeight(branch.node);
                positioned.set(branch.node.id, {
                    ...branch.node,
                    position: direction === 'TB'
                        ? {
                            x: branch.direction < 0
                                ? source.position.x - PIPELINE_BRANCH_GAP - targetWidth
                                : source.position.x + sourceWidth + PIPELINE_BRANCH_GAP,
                            y: sourceCenterY - targetHeight / 2,
                        }
                        : {
                            x: sourceCenterX - targetWidth / 2,
                            y: branch.direction < 0
                                ? source.position.y - PIPELINE_BRANCH_GAP - targetHeight
                                : source.position.y + sourceHeight + PIPELINE_BRANCH_GAP,
                        },
                });
            });
            continue;
        }

        const widestTarget = Math.max(...targets.map(branch => nodeWidth(branch.node)));
        targets.forEach((branch, index) => {
            const targetWidth = nodeWidth(branch.node);
            const targetHeight = nodeHeight(branch.node);
            const offset = (index - (targets.length - 1) / 2) * (widestTarget + PIPELINE_BRANCH_GAP);
            positioned.set(branch.node.id, {
                ...branch.node,
                position: direction === 'TB'
                    ? {
                        x: sourceCenterX + offset - targetWidth / 2,
                        y: source.position.y + sourceHeight + PIPELINE_BRANCH_GAP,
                    }
                    : {
                        x: source.position.x + sourceWidth + PIPELINE_BRANCH_GAP,
                        y: sourceCenterY + offset - targetHeight / 2,
                    },
            });
        });
    }

    return nodes.map(node => positioned.get(node.id) ?? node);
}

/**
 * Find connected components in the graph using union-find
 */
const findConnectedComponents = (nodes: Node[], edges: Edge[]): Map<string, Set<string>> => {
    const nodeIds = new Set(nodes.map(n => n.id));
    const adjacency = new Map<string, Set<string>>();

    // Build adjacency list
    nodeIds.forEach(id => adjacency.set(id, new Set()));
    edges.forEach(edge => {
        adjacency.get(edge.source)?.add(edge.target);
        adjacency.get(edge.target)?.add(edge.source);
    });

    const visited = new Set<string>();
    const components = new Map<string, Set<string>>();
    let componentId = 0;

    // DFS to find connected components
    const dfs = (nodeId: string, component: Set<string>) => {
        visited.add(nodeId);
        component.add(nodeId);
        adjacency.get(nodeId)?.forEach(neighbor => {
            if (!visited.has(neighbor)) {
                dfs(neighbor, component);
            }
        });
    };

    nodeIds.forEach(nodeId => {
        if (!visited.has(nodeId)) {
            const component = new Set<string>();
            dfs(nodeId, component);
            components.set(`component-${componentId++}`, component);
        }
    });

    return components;
};

/**
 * Layout a single connected component using Dagre
 */
const layoutComponent = (
    nodes: Node[],
    edges: Edge[],
    options: ILayoutOption
): { nodes: Node[]; width: number; height: number; minX: number; minY: number } => {
    const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}));

    g.setGraph({
        rankdir: options.direction,
        edgesep: 50,
        nodesep: 80,
        ranksep: 120,
        marginx: 40,
        marginy: 40,
        align: "UL",
        acyclicer: "greedy",
        ranker: "network-simplex",
    });

    edges.forEach((edge) => g.setEdge(edge.source, edge.target));

    nodes.forEach((node) => {
        // Use node dimensions or fallback to defaults
        const width = nodeWidth(node);
        const height = nodeHeight(node);

        const nodeWithDimensions = {
            ...node,
            width,
            height,
        };
        g.setNode(node.id, nodeWithDimensions as any);
    });

    Dagre.layout(g);

    // Calculate component bounds including node dimensions
    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    const layoutedNodes = nodes.map((node) => {
        const dagreNode = g.node(node.id);
        const x = dagreNode?.x ?? 0;
        const y = dagreNode?.y ?? 0;

        // Use node dimensions or fallback to defaults
        const width = nodeWidth(node);
        const height = nodeHeight(node);

        minX = Math.min(minX, x - width / 2);
        minY = Math.min(minY, y - height / 2);
        maxX = Math.max(maxX, x + width / 2);
        maxY = Math.max(maxY, y + height / 2);

        return {
            ...node,
            position: { x, y }
        };
    });

    return {
        nodes: layoutedNodes,
        width: maxX - minX,
        height: maxY - minY,
        minX,
        minY,
    };
};

/**
 * Pack components into a grid layout with optimal spacing
 */
const packComponents = (
    components: Array<{ nodes: Node[]; width: number; height: number; minX: number; minY: number }>
): Node[] => {
    if (components.length === 0) return [];
    if (components.length === 1) {
        // Normalize single component to start at (0, 0)
        // Convert from Dagre center positions to React Flow top-left positions
        const component = components[0];
        return component.nodes.map(node => {
            const width = nodeWidth(node);
            const height = nodeHeight(node);

            // Dagre positions are centers, React Flow expects top-left
            const centerX = node.position.x - component.minX;
            const centerY = node.position.y - component.minY;
            const topLeftX = centerX - width / 2;
            const topLeftY = centerY - height / 2;

            return {
                ...node,
                position: {
                    x: topLeftX,
                    y: topLeftY,
                }
            };
        });
    }

    // Sort components by area (largest first) for better packing
    const sorted = [...components].sort((a, b) => (b.width * b.height) - (a.width * a.height));

    const COMPONENT_SPACING = 150; // Spacing between disconnected components

    // Calculate grid dimensions for optimal aspect ratio
    const targetAspectRatio = 1.5; // Slightly wider than tall
    const cols = Math.max(1, Math.ceil(Math.sqrt(sorted.length * targetAspectRatio)));

    // Build explicit grid structure
    const grid: Array<Array<typeof sorted[0]>> = [];
    let rowIndex = 0;
    let colIndex = 0;

    sorted.forEach((component) => {
        if (colIndex >= cols) {
            rowIndex++;
            colIndex = 0;
        }

        if (!grid[rowIndex]) {
            grid[rowIndex] = [];
        }

        grid[rowIndex][colIndex] = component;
        colIndex++;
    });

    // Calculate row heights (max height in each row)
    const rowHeights = grid.map(row =>
        Math.max(...row.map(comp => comp.height))
    );

    // Calculate Y position for each row
    const rowYPositions = rowHeights.reduce((acc, _height, index) => {
        if (index === 0) {
            acc.push(0);
        } else {
            acc.push(acc[index - 1] + rowHeights[index - 1] + COMPONENT_SPACING);
        }
        return acc;
    }, [] as number[]);

    // Position all components in the grid
    const allNodes: Node[] = [];

    grid.forEach((row, rIdx) => {
        let currentX = 0;
        const currentY = rowYPositions[rIdx];

        row.forEach((component) => {
            // Calculate offset to move component to grid position
            const offsetX = currentX - component.minX;
            const offsetY = currentY - component.minY;

            // Position all nodes in this component
            // Convert from Dagre center positions to React Flow top-left positions
            component.nodes.forEach(node => {
                const width = nodeWidth(node);
                const height = nodeHeight(node);

                // Dagre positions are centers, React Flow expects top-left
                const centerX = node.position.x + offsetX;
                const centerY = node.position.y + offsetY;
                const topLeftX = centerX - width / 2;
                const topLeftY = centerY - height / 2;

                allNodes.push({
                    ...node,
                    position: {
                        x: topLeftX,
                        y: topLeftY,
                    }
                });
            });

            // Move X position for next component in this row
            currentX += component.width + COMPONENT_SPACING;
        });
    });

    return allNodes;
};

export const getDagreLayoutedElements = (nodes: Node[] = [], edges: Edge[] = [], options: ILayoutOption = {
    direction: "LR",
}) => {
    if (nodes.length === 0) {
        return { nodes: [], edges };
    }

    // Find connected components
    const componentMap = findConnectedComponents(nodes, edges);

    // If only one component, use simple Dagre layout
    if (componentMap.size === 1) {
        const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}));

        g.setGraph({
            rankdir: options.direction,
            edgesep: 50,
            nodesep: 80,
            ranksep: 120,
            marginx: 50,
            marginy: 50,
            align: "UL",
            acyclicer: "greedy",
            ranker: "network-simplex",
        });

        edges.forEach((edge) => g.setEdge(edge.source, edge.target));

        nodes.forEach((node) => {
            // Use node dimensions or fallback to defaults
            const width = nodeWidth(node);
            const height = nodeHeight(node);

            const nodeWithDimensions = {
                ...node,
                width,
                height,
            };
            g.setNode(node.id, nodeWithDimensions as any);
        });

        Dagre.layout(g);

        const layoutedNodes = nodes.map((node) => {
                const dagreNode = g.node(node.id);
                const width = nodeWidth(node);
                const height = nodeHeight(node);

                // Dagre positions are centers, React Flow expects top-left
                const centerX = dagreNode?.x ?? 0;
                const centerY = dagreNode?.y ?? 0;
                const topLeftX = centerX - width / 2;
                const topLeftY = centerY - height / 2;

                return {
                    ...node,
                    position: {
                        x: topLeftX,
                        y: topLeftY
                    }
                };
            });
        return {
            nodes: positionPipelineTopology(layoutedNodes, edges, options.direction),
            edges,
        };
    }

    // Layout each component separately
    const layoutedComponents: Array<{ nodes: Node[]; width: number; height: number; minX: number; minY: number }> = [];

    componentMap.forEach((nodeIds) => {
        const componentNodes = nodes.filter(n => nodeIds.has(n.id));
        const componentEdges = edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target));

        const layouted = layoutComponent(componentNodes, componentEdges, options);
        layoutedComponents.push(layouted);
    });

    // Pack components into optimal grid layout
    const finalNodes = positionPipelineTopology(packComponents(layoutedComponents), edges, options.direction);

    return {
        nodes: finalNodes,
        edges,
    };
};
