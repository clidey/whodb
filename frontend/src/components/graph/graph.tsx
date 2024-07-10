import classNames from 'classnames';
import { Dispatch, FC, ReactNode, SetStateAction, useCallback, useMemo, useRef, useState } from "react";
import ReactFlow, { Background, Controls, Edge, Node, NodeProps, NodeTypes, OnInit, ReactFlowInstance, ReactFlowProps, useReactFlow } from 'reactflow';
import { ActionButton } from '../button';
import { Icons } from '../icons';
import { GraphElements } from './constants';
import { FloatingGraphEdge, GraphEdgeConnectionLine } from './edge';
import { getDagreLayoutedElements } from './layouts';
import { toPng } from 'html-to-image';


export type IGraphCardProps<T extends unknown = any> = NodeProps<(T & {}) | undefined>;

export const createRedirectState = (nodes: {id: string, type: GraphElements}[]) => {
    return {
        nodes,
    };
}

export type IGraphInstance = {
    layout: (type?: "dagre") => void;
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
    const [isLayingOut, setIsLayingOut] = useState(false);
    const { getNodes, getEdges, setNodes, setEdges } = useReactFlow();
    const [downloading, setDownloading] = useState(false);

    const edgeTypes = useMemo(() => ({
        floatingGraphEdge: FloatingGraphEdge,
    }), []);

    const onLayout = useCallback((type = "dagre") => {
        const nodes = getNodes();
        const edges = getEdges();

        if (nodes.length === 0) {
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
          toPng(reactFlowWrapper.current)
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
        className={classNames("group bg-white", {
            "laying-out": isLayingOut,
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
        <Background />
        {
            !downloading && <Controls />
        }
        <div className={classNames("flex flex-row gap-2 absolute bottom-8 right-5 z-10", {
            "hidden": downloading,
        })}>
            <div className="flex flex-col gap-2">
                <ActionButton icon={Icons.Download} onClick={handleDownloadImage} />
                <ActionButton icon={Icons.GraphLayout} onClick={() => onLayout("dagre")} />
            </div>
        </div>
        {props.children}
    </ReactFlow>
}