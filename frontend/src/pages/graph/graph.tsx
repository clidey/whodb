// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

import { useQuery } from "@apollo/client";
import { FC, useCallback, useMemo, useRef } from "react";
import { Edge, Node, ReactFlowProvider, useEdgesState, useNodesState } from "reactflow";
import { EmptyMessage } from "../../components/common";
import { GraphElements } from "../../components/graph/constants";
import { Graph, IGraphInstance } from "../../components/graph/graph";
import { createEdge, createNode } from "../../components/graph/utils";
import { Icons } from "../../components/icons";
import { LoadingPage } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { GetGraphDocument, GetGraphQuery, GetGraphQueryVariables } from "../../generated/graphql";
import { useAppSelector } from "../../store/hooks";
import { getDatabaseStorageUnitLabel } from "../../utils/functions";
import { StorageUnitGraphCard } from "../storage-unit/storage-unit";

export const GraphPage: FC = () => {
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const reactFlowRef = useRef<IGraphInstance>();
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);

    const { loading } = useQuery<GetGraphQuery, GetGraphQueryVariables>(GetGraphDocument, {
        variables: {
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

    if (loading) {
        return <InternalPage routes={[InternalRoutes.Graph]}>
            <LoadingPage />
        </InternalPage>
    }

    return <InternalPage routes={[InternalRoutes.Graph]}>
        <ReactFlowProvider>
            {
                !loading && nodes.length === 0
                ? <EmptyMessage icon={Icons.SadSmile} label={`No ${getDatabaseStorageUnitLabel(current?.Type)} found`} />
                : <Graph nodes={nodes} edges={edges} nodeTypes={nodeTypes}
                    setNodes={setNodes} setEdges={setEdges}
                    onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}    
                    minZoom={0.1}
                    onReady={handleOnReady} />
            }
        </ReactFlowProvider>
    </InternalPage>
}