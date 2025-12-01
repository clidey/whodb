/*
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

import {useQuery} from "@apollo/client";
import {FC, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {Edge, Node, ReactFlowProvider, useEdgesState, useNodesState} from "reactflow";
import {GraphElements} from "../../components/graph/constants";
import {Graph, IGraphInstance} from "../../components/graph/graph";
import {createEdge, createNode} from "../../components/graph/utils";
import {LoadingPage} from "../../components/loading";
import {InternalPage} from "../../components/page";
import {InternalRoutes} from "../../config/routes";
import {
    GetGraphDocument,
    GetGraphQuery,
    GetGraphQueryVariables,
    StorageUnit,
    useGetColumnsLazyQuery,
    useGetStorageUnitsQuery
} from '@graphql';
import {useAppSelector} from "../../store/hooks";
import {getDatabaseStorageUnitLabel} from "../../utils/functions";
import {StorageUnitGraphCard} from "../storage-unit/storage-unit";
import {
    Button,
    Checkbox,
    EmptyState,
    SearchInput,
    Sidebar as SidebarComponent,
    SidebarContent,
    SidebarGroup,
    SidebarHeader,
    toTitleCase
} from "@clidey/ux";
import {useNavigate} from "react-router-dom";
import {FolderIcon, RectangleGroupIcon, TableCellsIcon} from "../../components/heroicons";
import {databaseUsesSchemaForGraph} from "../../utils/database-features";

// Helper function to group storage units by type
function groupByType(units: StorageUnit[]) {
    const groups: Record<string, StorageUnit[]> = {};
    for (const unit of units) {
        const typeAttr = unit.Attributes.find(a => a.Key === "Type")?.Value ?? "";
        const type = toTitleCase(typeAttr);
        if (type === "") continue; // Ignore grouping if empty
        if (!groups[type]) groups[type] = [];
        groups[type].push(unit);
    }
    return groups;
}

// Sidebar component for graph page
interface GraphSidebarProps {
    current: any;
    search: string;
    setSearch: (search: string) => void;
    selectedUnits: Set<string>;
    setSelectedUnits: (units: Set<string> | ((prev: Set<string>) => Set<string>)) => void;
    storageUnitsData: any;
    unitsLoading: boolean;
}

const GraphSidebar: FC<GraphSidebarProps> = ({
    current,
    search,
    setSearch,
    selectedUnits,
    setSelectedUnits,
    storageUnitsData,
    unitsLoading
}) => {
    const children = useMemo(() => {
        const units: StorageUnit[] = (storageUnitsData?.StorageUnit ?? [])
            .filter((u: StorageUnit) => u.Name.toLowerCase().includes(search.trim().toLowerCase()));
        const groups = groupByType(units);
        const groupEntries = Object.entries(groups);
        if (groupEntries.length === 0) {
            return <div className="text-sm text-muted-foreground px-2">No items</div>;
        }
        return groupEntries.map(([type, units]) => (
            <div key={type} className="mb-3">
                <div className="flex items-center gap-2 px-2 py-1 font-medium">
                    <FolderIcon className="w-4 h-4" /> {type}
                </div>
                {units.map(u => {
                    const checked = selectedUnits.has(u.Name);
                    return (
                        <label key={u.Name} className="flex items-center gap-2 px-4 py-1 cursor-pointer select-none">
                            <Checkbox checked={checked} onCheckedChange={(checked) => {
                                setSelectedUnits((prev: Set<string>) => {
                                    const next = new Set(prev);
                                    if (checked) next.add(u.Name); else next.delete(u.Name);
                                    return next;
                                });
                            }} />
                            <TableCellsIcon className="w-4 h-4 min-w-4 min-h-4" />
                            <p className="text-sm truncate max-w-full overflow-hidden text-ellipsis whitespace-nowrap">
                                {u.Name}
                            </p>
                        </label>
                    );
                })}
            </div>
        ));
    }, [search, selectedUnits, setSelectedUnits, storageUnitsData]);

    return (
        <div className="dark flex grow">
            <SidebarComponent variant="embed" className="w-64 h-full flex flex-col">
                <SidebarContent data-testid="graph-sidebar-content">
                    <SidebarHeader>
                        <h1 className="text-lg font-semibold pt-8 px-4">
                            {getDatabaseStorageUnitLabel(current?.Type)}
                        </h1>
                    </SidebarHeader>
                    <div className="px-4">
                        <SearchInput
                            value={search}
                            onChange={e => setSearch(e.target.value)}
                            placeholder="Search tables..."
                            aria-label="Search tables"
                        />
                    </div>
                    <SidebarGroup>
                        {
                            unitsLoading
                                ? <div className="flex-1 flex items-center justify-center"><LoadingPage /></div>
                                : <div className="flex-1 overflow-y-auto px-2 py-2">
                                    {children}
                                </div>
                        }
                    </SidebarGroup>
                </SidebarContent>
            </SidebarComponent>
        </div>
    );
};

export const GraphPage: FC = () => {
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const reactFlowRef = useRef<IGraphInstance>();
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const navigate = useNavigate();
    const [search, setSearch] = useState("");
    const [selectedUnits, setSelectedUnits] = useState<Set<string>>(new Set());
    const [graphData, setGraphData] = useState<GetGraphQuery["Graph"]>([]);
    const [isInitialized, setIsInitialized] = useState(false);
    const [tableColumns, setTableColumns] = useState<Record<string, any[]>>({});

    const [fetchColumns] = useGetColumnsLazyQuery();

    const {
        loading: graphLoading,
        refetch: refetchGraph
    } = useQuery<GetGraphQuery, GetGraphQueryVariables>(GetGraphDocument, {
        variables: {
            schema: databaseUsesSchemaForGraph(current?.Type) ? schema : current?.Database ?? "",
        },
        onCompleted(data) {
            setGraphData(data.Graph);
            // Fetch columns for each table
            const fetchAllColumns = async () => {
                const columnsMap: Record<string, any[]> = {};
                for (const graphUnit of data.Graph) {
                    try {
                        const result = await fetchColumns({
                            variables: {
                                schema: databaseUsesSchemaForGraph(current?.Type) ? schema : current?.Database ?? "",
                                storageUnit: graphUnit.Unit.Name,
                            },
                        });
                        if (result.data?.Columns) {
                            columnsMap[graphUnit.Unit.Name] = result.data.Columns;
                        }
                    } catch (error) {
                        console.error(`Failed to fetch columns for ${graphUnit.Unit.Name}:`, error);
                    }
                }
                setTableColumns(columnsMap);
            };
            fetchAllColumns();
        },
    });

    // Fetch all storage units for sidebar selection
    const {data: storageUnitsData, loading: unitsLoading, refetch: refetchStorageUnits} = useGetStorageUnitsQuery({
        variables: {
            schema: databaseUsesSchemaForGraph(current?.Type) ? schema : current?.Database ?? "",
        },
        skip: !current,
        fetchPolicy: "cache-and-network",
    });

    // Refetch when profile changes (current?.Id changes means different server/credentials)
    const currentProfileId = current?.Id;
    useEffect(() => {
        if (currentProfileId) {
            refetchGraph();
            refetchStorageUnits();
        }
    }, [currentProfileId, refetchGraph, refetchStorageUnits]);

    // Default selection logic: auto-select all if < 10 units, else none
    useEffect(() => {
        const units = storageUnitsData?.StorageUnit ?? [];
        if (units.length === 0) return;
        setSelectedUnits(prev => {
            if (prev.size > 0) return prev;
            if (units.length < 10) {
                return new Set(units.map(u => u.Name));
            }
            return new Set();
        });
    }, [storageUnitsData?.StorageUnit]);

    // Build nodes and edges from graph data and selection
    const { computedNodes, computedEdges } = useMemo(() => {
        if (!graphData || selectedUnits.size === 0) {
            return { computedNodes: [], computedEdges: [] };
        }

        const newNodes: Node[] = [];
        const newEdges: Edge[] = [];
        const newEdgesSet = new Set<string>();

        // Create nodes for selected units with column data
        for (const node of graphData) {
            if (!selectedUnits.has(node.Unit.Name)) continue;
            const columns = tableColumns[node.Unit.Name];
            newNodes.push(createNode({
                id: node.Unit.Name,
                type: GraphElements.StorageUnit,
                data: {
                    ...node.Unit,
                    columns: columns || node.Unit.Attributes,
                },
            }));
        }
        
        // Create edges between selected nodes with column-level connections
        for (const node of graphData) {
            if (!selectedUnits.has(node.Unit.Name)) continue;
            for (const relation of node.Relations) {
                if (!selectedUnits.has(relation.Name)) continue;

                const sourceColumn = relation.SourceColumn;
                const targetColumn = relation.TargetColumn;

                // Generate unique edge ID based on columns if available
                const edgeId = sourceColumn && targetColumn
                    ? `${node.Unit.Name}-${sourceColumn}-${relation.Name}-${targetColumn}`
                    : `${node.Unit.Name}-${relation.Name}`;

                if (newEdgesSet.has(edgeId)) continue;

                if (relation.Relationship === "ManyToMany") {
                    const edge1 = {
                        ...createEdge(node.Unit.Name, relation.Name),
                        id: edgeId + '-1',
                    };
                    const edge2 = {
                        ...createEdge(relation.Name, node.Unit.Name),
                        id: edgeId + '-2',
                    };
                    newEdgesSet.add(edgeId);
                    newEdges.push(edge1, edge2);
                } else {
                    let source: string;
                    let target: string;
                    let sourceHandle: string | undefined;
                    let targetHandle: string | undefined;

                    if (relation.Relationship === "OneToMany") {
                        // OneToMany stored on referenced table (table with PK)
                        // node.Unit.Name = referenced table (has PK)
                        // relation.Name = referencing table (has FK)
                        // Arrow: referencing -> referenced (FK -> PK)
                        source = relation.Name;
                        target = node.Unit.Name;
                        // Only set handles if columns are loaded AND the specific columns exist
                        if (sourceColumn && targetColumn) {
                            const sourceColumns = tableColumns[source];
                            const targetColumns = tableColumns[target];
                            const sourceColExists = sourceColumns?.some(col => col.Name === sourceColumn && col.IsForeignKey);
                            const targetColExists = targetColumns?.some(col => col.Name === targetColumn && col.IsPrimary);

                            if (sourceColExists && targetColExists) {
                                sourceHandle = `${relation.Name}-${sourceColumn}`;
                                targetHandle = `${node.Unit.Name}-${targetColumn}`;
                            }
                        }
                    } else if (relation.Relationship === "ManyToOne") {
                        // ManyToOne stored on referencing table (table with FK)
                        // node.Unit.Name = referencing table (has FK)
                        // relation.Name = referenced table (has PK)
                        // Arrow: referencing -> referenced (FK -> PK)
                        source = node.Unit.Name;
                        target = relation.Name;
                        // Only set handles if columns are loaded AND the specific columns exist
                        if (sourceColumn && targetColumn) {
                            const sourceColumns = tableColumns[source];
                            const targetColumns = tableColumns[target];
                            const sourceColExists = sourceColumns?.some(col => col.Name === sourceColumn && col.IsForeignKey);
                            const targetColExists = targetColumns?.some(col => col.Name === targetColumn && col.IsPrimary);

                            if (sourceColExists && targetColExists) {
                                sourceHandle = `${node.Unit.Name}-${sourceColumn}`;
                                targetHandle = `${relation.Name}-${targetColumn}`;
                            }
                        }
                    } else {
                        // Unknown or OneToOne - default behavior
                        source = node.Unit.Name;
                        target = relation.Name;
                    }

                    const newEdge: Edge = {
                        ...createEdge(source, target),
                        id: edgeId,
                        sourceHandle,
                        targetHandle,
                    };

                    newEdgesSet.add(edgeId);
                    newEdges.push(newEdge);
                }
            }
        }

        return { computedNodes: newNodes, computedEdges: newEdges };
    }, [graphData, selectedUnits, tableColumns]);

    // Update nodes and edges when computed values change
    useEffect(() => {
        setNodes(computedNodes);
        setEdges(computedEdges);
    }, [computedNodes, computedEdges, setNodes, setEdges]);

    // Layout the graph when nodes change and graph is initialized
    useEffect(() => {
        if (isInitialized && computedNodes.length > 0) {
            // Wait for nodes to be rendered, then measure and layout
            const timer = setTimeout(() => {
                reactFlowRef.current?.layout("dagre");
            }, 50);
            return () => clearTimeout(timer);
        }
    }, [isInitialized, computedNodes]);

    const handleOnReady = useCallback((instance: IGraphInstance) => {
        reactFlowRef.current = instance;
        setIsInitialized(true);
    }, []);

    const nodeTypes = useMemo(() => ({
        [GraphElements.StorageUnit]: StorageUnitGraphCard,
    }), []);

    if (graphLoading) {
        return <InternalPage key="graph-loading" routes={[InternalRoutes.Graph]}>
            <LoadingPage />
        </InternalPage>
    }


    return <InternalPage key="graph" routes={[InternalRoutes.Graph]} sidebar={
        <GraphSidebar
            current={current}
            search={search}
            setSearch={setSearch}
            selectedUnits={selectedUnits}
            setSelectedUnits={setSelectedUnits}
            storageUnitsData={storageUnitsData}
            unitsLoading={unitsLoading}
        />
    }>
        <div className="flex-1 h-full">
            <ReactFlowProvider>
                {
                    !graphLoading && nodes.length === 0
                        ? <EmptyState
                            icon={<RectangleGroupIcon className="w-4 h-4" />}
                            title={`No nodes selected`}
                            description={`Select ${getDatabaseStorageUnitLabel(current?.Type).toLowerCase()} on the left to add them to the graph.`}>
                            <Button onClick={() => navigate(InternalRoutes.Dashboard.StorageUnit.path + "?create=true")}>
                                Create {getDatabaseStorageUnitLabel(current?.Type, true)}
                            </Button>
                        </EmptyState>
                        : <Graph nodes={nodes} edges={edges} nodeTypes={nodeTypes}
                                setNodes={setNodes} setEdges={setEdges}
                                onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
                                minZoom={0.1}
                                onReady={handleOnReady} />
                }
            </ReactFlowProvider>
        </div>
    </InternalPage>
}