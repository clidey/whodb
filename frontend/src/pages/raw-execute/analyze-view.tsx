import React, { useEffect, useState, useCallback, useMemo, useRef, FC } from "react";
import ReactFlow, {
    Controls,
    Background,
    ReactFlowProvider,
    useNodesState,
    useEdgesState,
    Handle,
    Position,
} from "reactflow";
import "reactflow/dist/style.css";
import { v4 as uuidv4 } from "uuid";
import { Graph, IGraphInstance } from "../../components/graph/graph";
import { Card } from "../../components/card";
import { Icons } from "../../components/icons";
import { ClassNames } from "../../components/classes";
import classNames from "classnames";
import { createEdge, createNode } from "../../components/graph/utils";

type IPlanNode = {
    "Node Type": string;
    "Hash Cond"?: string;
    "Join Type"?: string;
    "Relation Name"?: string;
    "Actual Rows"?: number;
    "Actual Total Time"?: number;
    Plans?: IPlanNode[];
}

type IExplainAnalyzeResult = {
    Plan: IPlanNode;
    "Execution Time": number;
}


enum GraphElements {
    ScanNode = "ScanNode",
    JoinNode = "JoinNode",
    HashNode = "HashNode",
    AggregateNode = "AggregateNode",
}

export const ScanNode: FC<{ data: any }> = ({ data }) => {
    return (
        <>
            <Handle className="dark:border-white/5" type="target" position={Position.Left} />
            <Card icon={{ component: Icons.Search, bgClassName: ClassNames.IconBackground }}
                className="h-fit backdrop-blur-[2px] bg-transparent">
                <div className="flex flex-col grow mt-2">
                    <div className={classNames(ClassNames.Text, "text-md font-semibold mb-2")}>
                        {data["Node Type"]} on {data["Relation Name"]}
                    </div>
                    <div className={classNames(ClassNames.Text, "text-xs")}>
                        <span className="font-semibold">Rows:</span> {data["Actual Rows"]}
                    </div>
                    <div className={classNames(ClassNames.Text, "text-xs")}>
                        <span className="font-semibold">Time:</span> {data["Actual Total Time"]} ms
                    </div>
                </div>
            </Card>
            <Handle className="dark:border-white/5" type="source" position={Position.Right} />
        </>
    );
};


export const JoinNode: FC<{ data: any }> = ({ data }) => {
    return (
        <>
            <Handle className="dark:border-white/5" type="target" position={Position.Left} />
            <Card icon={{ component: Icons.Link, bgClassName: ClassNames.IconBackground }}
                className="h-fit backdrop-blur-[2px] bg-transparent">
                <div className="flex flex-col grow mt-2">
                    <div className={classNames(ClassNames.Text, "text-md font-semibold mb-2")}>
                        {data["Node Type"]} ({data["Join Type"]})
                    </div>
                    <div className={classNames(ClassNames.Text, "text-xs")}>
                        <span className="font-semibold">Rows:</span> {data["Actual Rows"]}
                    </div>
                    {data["Hash Cond"] && (
                        <div className={classNames(ClassNames.Text, "text-xs")}>
                            <span className="font-semibold">Condition:</span> {data["Hash Cond"]}
                        </div>
                    )}
                    <div className={classNames(ClassNames.Text, "text-xs")}>
                        <span className="font-semibold">Time:</span> {data["Actual Total Time"]} ms
                    </div>
                </div>
            </Card>
            <Handle className="dark:border-white/5" type="source" position={Position.Right} />
        </>
    );
};


export const HashNode: FC<{ data: any }> = ({ data }) => {
    return (
        <>
            <Handle className="dark:border-white/5" type="target" position={Position.Left} />
            <Card icon={{ component: Icons.Hash, bgClassName: ClassNames.IconBackground }}
                className="h-fit backdrop-blur-[2px] bg-transparent">
                <div className="flex flex-col grow mt-2">
                    <div className={classNames(ClassNames.Text, "text-md font-semibold mb-2")}>
                        {data["Node Type"]}
                    </div>
                    <div className={classNames(ClassNames.Text, "text-xs")}>
                        <span className="font-semibold">Buckets:</span> {data["Hash Buckets"]}
                    </div>
                    <div className={classNames(ClassNames.Text, "text-xs")}>
                        <span className="font-semibold">Memory:</span> {data["Peak Memory Usage"]} KB
                    </div>
                </div>
            </Card>
            <Handle className="dark:border-white/5" type="source" position={Position.Right} />
        </>
    );
};

export const AggregateNode: FC<{ data: any }> = ({ data }) => {
    return (
        <>
            <Handle className="dark:border-white/5" type="target" position={Position.Left} />
            <Card icon={{ component: Icons.Chart, bgClassName: ClassNames.IconBackground }}
                className="h-fit backdrop-blur-[2px] bg-transparent">
                <div className="flex flex-col grow mt-2">
                    <div className={classNames(ClassNames.Text, "text-md font-semibold mb-2")}>
                        {data["Node Type"]}
                    </div>
                    <div className={classNames(ClassNames.Text, "text-xs")}>
                        <span className="font-semibold">Rows:</span> {data["Actual Rows"]}
                    </div>
                    <div className={classNames(ClassNames.Text, "text-xs")}>
                        <span className="font-semibold">Time:</span> {data["Actual Total Time"]} ms
                    </div>
                </div>
            </Card>
            <Handle className="dark:border-white/5" type="source" position={Position.Right} />
        </>
    );
};



const mapNodeType = (node: any): GraphElements => {
    switch (node["Node Type"]) {
        case "Seq Scan":
        case "Index Scan":
            return GraphElements.ScanNode;
        case "Hash Join":
        case "Nested Loop":
            return GraphElements.JoinNode;
        case "Hash":
            return GraphElements.HashNode;
        case "Aggregate":
        case "Sort":
            return GraphElements.AggregateNode;
        default:
            return GraphElements.ScanNode;
    }
};

const convertPlanToGraph = (plan: IPlanNode, parentId: string | null = null) => {
    const id = uuidv4();
    const node =  createNode({
        id,
        type: mapNodeType(plan),
        data: plan,
    });

    const edges = parentId
        ? [createEdge(parentId, id)]
        : [];

    let childNodes: any[] = [];
    let childEdges: any[] = [];

    if (plan.Plans) {
        for (const child of plan.Plans) {
            const { nodes, edges: newEdges } = convertPlanToGraph(child, id);
            childNodes.push(...nodes);
            childEdges.push(...newEdges);
        }
    }

    return { nodes: [node, ...childNodes], edges: [...edges, ...childEdges] };
};

export const AnalyzeGraph: React.FC<{ data: IExplainAnalyzeResult }> = ({ data }) => {
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const reactFlowRef = useRef<IGraphInstance>();

    useEffect(() => {
        if (data == null) {
            return;
        }
        if (data.Plan != null) {
            const { nodes, edges } = convertPlanToGraph(data.Plan);
            setNodes(nodes);
            setEdges(edges);
            setTimeout(() => {
                reactFlowRef.current?.layout("dagre");
            }, 500);
        }
    }, [data]);

    const handleOnReady = useCallback((instance: IGraphInstance) => {
        reactFlowRef.current = instance;
    }, []);

    const nodeTypes = useMemo(() => ({
        [GraphElements.ScanNode]: ScanNode,
        [GraphElements.JoinNode]: JoinNode,
        [GraphElements.HashNode]: HashNode,
        [GraphElements.AggregateNode]: AggregateNode,
    }), []);

    return (
        <ReactFlowProvider>
            <Graph nodes={nodes} edges={edges} nodeTypes={nodeTypes}
                setNodes={setNodes} setEdges={setEdges}
                onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
                minZoom={0.1}
                onReady={handleOnReady} />
        </ReactFlowProvider>
    );
};
