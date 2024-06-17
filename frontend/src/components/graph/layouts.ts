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
        edgesep: 10,
        nodesep: 20,
        ranksep: 50,
        marginx: 20,
        marginy: 20,
        align: "UL",
    });

    edges.forEach((edge) => g.setEdge(edge.source, edge.target));
    nodes.forEach((node) => g.setNode(node.id, node as any));

    Dagre.layout(g);

    return {
        nodes: nodes.map((node) => {
            const { x, y } = g.node(node.id);
            return { ...node, position: { x, y } };
        }),
        edges,
    };
};