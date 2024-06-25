import { useQuery } from "@apollo/client";
import { FC, useCallback, useMemo, useRef } from "react";
import { Edge, Node, ReactFlowProvider, useEdgesState, useNodesState } from "reactflow";
import { GraphElements } from "../../components/graph/constants";
import { Graph, IGraphInstance } from "../../components/graph/graph";
import { createEdge, createNode } from "../../components/graph/utils";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, GetGraphDocument, GetGraphQuery, GetGraphQueryVariables } from "../../generated/graphql";
import { StorageUnitGraphCard } from "../storage-unit/storage-unit";
import { useAppSelector } from "../../store/hooks";
import { EmptyMessage } from "../../components/common";
import { Icons } from "../../components/icons";

export const GraphPage: FC = () => {
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const reactFlowRef = useRef<IGraphInstance>();
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);

    const { data, loading } = useQuery<GetGraphQuery, GetGraphQueryVariables>(GetGraphDocument, {
        variables: {
            type: current?.Type as DatabaseType,
            schema,
        },
        onCompleted(data) {
            const newNodes: Node[] = [];
            const newEdges: Edge[] = [];
            const newEdgesSet = new Set<string>();
            for (const node of data.Graph) {
                newNodes.push(createNode({
                    id: node.Unit.Name,
                    type: GraphElements.StorageUnit,
                    data: node.Unit,
                }));
                for (const edge of node.Relations) {
                    const newEdge = createEdge(node.Unit.Name, edge.Name);
                    if (newEdgesSet.has(newEdge.id)) {
                        continue;
                    }
                    newEdgesSet.add(newEdge.id);
                    newEdges.push(newEdge);
                }
            }
            setNodes(newNodes);
            setEdges(newEdges);
            setTimeout(() => {
                reactFlowRef.current?.layout("dagre");
            }, 300);
        },
    });

    const handleOnReady = useCallback((instance: IGraphInstance) => {
        reactFlowRef.current = instance;
    }, []);

    const nodeTypes = useMemo(() => ({
        [GraphElements.StorageUnit]: StorageUnitGraphCard,
    }), []);

    if (loading || data == null) {
        return <InternalPage routes={[InternalRoutes.Graph]}>
            <Loading />
        </InternalPage>
    }

    return <InternalPage routes={[InternalRoutes.Graph]}>
        <ReactFlowProvider>
            {
                !loading && nodes.length === 0
                ? <EmptyMessage icon={Icons.SadSmile} label="No tables found. Try changing schema." />
                : <Graph nodes={nodes} edges={edges} nodeTypes={nodeTypes}
                    setNodes={setNodes} setEdges={setEdges}
                    onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}    
                    minZoom={0.1}
                    onReady={handleOnReady} />
            }
        </ReactFlowProvider>
    </InternalPage>
}