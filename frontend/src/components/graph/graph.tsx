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

import { Button, Tabs, TabsList, TabsTrigger } from '@clidey/ux';
import { ArrowDownTrayIcon, RectangleGroupIcon } from '../heroicons';
import classNames from 'classnames';
import { toPng } from 'html-to-image';
import { Dispatch, FC, ReactNode, SetStateAction, useCallback, useMemo, useRef, useState } from "react";
import ReactFlow, { Background, Controls, Edge, Node, NodeProps, NodeTypes, OnInit, ReactFlowInstance, ReactFlowProps, useReactFlow } from 'reactflow';
import { Tip } from '../tip';
import { GraphElements } from './constants';
import { FloatingGraphEdge, GraphEdgeConnectionLine } from './edge';
import { getDagreLayoutedElements } from './layouts';


export type IGraphCardProps<T extends unknown = any> = NodeProps<(T & {}) | undefined>;

export const createRedirectState = (nodes: {id: string, type: GraphElements}[]) => {
    return {
        nodes,
    };
}

export type IGraphInstance = {
    layout: (type?: "dagre", padding?: number) => void;
} & ReactFlowInstance;

export type IGraphProps<NodeData extends unknown = any, EdgeData extends unknown = any, ChangesType extends unknown = any> = {
    nodeTypes: NodeTypes;
    children?: ReactNode;
    nodes: Node<NodeData, string | undefined>[];
    setNodes: Dispatch<SetStateAction<Node<NodeData, string | undefined>[]>>;
    onNodesChange: (changes: ChangesType[]) => void;
    edges: Edge<EdgeData>[];
    setEdges: Dispatch<SetStateAction<Edge<EdgeData>[]>>;
    onEdgesChange: (changes: ChangesType[]) => void;
    onReady?: (instance: IGraphInstance) => void;
} & Partial<ReactFlowProps>;

export const Graph: FC<IGraphProps> = (props) => {
    const reactFlowWrapper = useRef<HTMLDivElement>(null);
    const { fitView } = useReactFlow();
    const [isLayingOut, setIsLayingOut] = useState(true);
    const { getNodes, getEdges, setNodes, setEdges } = useReactFlow();
    const [downloading, setDownloading] = useState(false);

    const edgeTypes = useMemo(() => ({
        floatingGraphEdge: FloatingGraphEdge,
    }), []);

    const onLayout = useCallback((type = "dagre", padding?: number) => {
        const nodes = getNodes();
        const edges = getEdges();

        if (nodes.length === 0) {
            return;
        }

        // Check if nodes have dimensions, if not, wait a bit more
        const nodesWithoutDimensions = nodes.some(node => !node.width || !node.height);
        if (nodesWithoutDimensions) {
            setTimeout(() => onLayout(type, padding), 100);
            return;
        }

        let layouted: { nodes: Node[], edges: Edge[] } = { nodes: [], edges: [] };
        if (type === "dagre") {
            layouted = getDagreLayoutedElements(nodes, edges);
        }

        setIsLayingOut(true);
        setNodes(layouted.nodes);
        setEdges(layouted.edges);

        setTimeout(() => {
            setIsLayingOut(false);
            fitView({
                duration: 300,
                padding,
            });
        }, 350);
    }, [fitView, getEdges, getNodes, setEdges, setNodes]);

    const handleInit: OnInit = useCallback((instance) => {
        setTimeout(() => {
            fitView({
                minZoom: 1,
                duration: 500,
                padding: 100,
            });
        }, 100);
        const graphInstance = instance as IGraphInstance;
        graphInstance.layout = onLayout;
        props.onReady?.(graphInstance);
    }, [fitView, onLayout, props]);

    const handleDownloadImage = useCallback(() => {
        if (reactFlowWrapper.current === null) {
            return;
          }
      
          setDownloading(true);
          toPng(reactFlowWrapper.current, {
            pixelRatio: 5,
          })
            .then((dataUrl) => {
              const link = document.createElement('a');
              link.download = 'clidey-whodb-diagram.png';
              link.href = dataUrl;
              link.click();
            })
            .catch((err) => {
              console.error('Could not capture the image', err);
            }).finally(() => {
                setDownloading(false);
            });
    }, []);

    return <ReactFlow
        ref={reactFlowWrapper}
        className={classNames("group rounded-lg transition-all", {
            "laying-out opacity-0": isLayingOut,
            "opacity-100": !isLayingOut,
        })}
        {...props}
        nodeTypes={props.nodeTypes}
        edgeTypes={edgeTypes}
        nodes={props.nodes}
        edges={props.edges}
        panOnScroll
        selectionOnDrag
        onNodesChange={props.onNodesChange}
        onEdgesChange={props.onEdgesChange}
        proOptions={{
            hideAttribution: true,
        }}
        fitView
        onInit={handleInit}
        connectionLineComponent={GraphEdgeConnectionLine}
    >
        <Background className="opacity-50" />
        {
            !downloading && <Controls />
        }
        <div className={classNames("flex flex-row gap-sm absolute bottom-4 right-4 z-10", {
            "hidden": downloading,
        })}>
            <div className="flex flex-col gap-2">
                <Tabs value={undefined} onValueChange={() => {}}>
                    <TabsList dir="column" className="px-1">
                        <TabsTrigger value="download" onClick={handleDownloadImage} asChild>
                            <Tip className="w-[30px]">
                                <Button data-testid="graph-download-button" variant="ghost">
                                    <ArrowDownTrayIcon className="w-4 h-4 dark:text-white" />
                                </Button>
                                Download
                            </Tip>
                        </TabsTrigger>
                        <TabsTrigger value="layout" onClick={() => onLayout("dagre")} asChild>
                            <Tip className="w-[30px]">
                                <Button data-testid="graph-layout-button" variant="ghost">
                                    <RectangleGroupIcon className="w-4 h-4 dark:text-white" />
                                </Button>
                                Layout
                            </Tip>
                        </TabsTrigger>
                    </TabsList>
                </Tabs>
            </div>
        </div>
        {props.children}
    </ReactFlow>
}