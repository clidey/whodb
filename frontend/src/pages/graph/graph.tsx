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

import {skipToken, useLazyQuery, useQuery} from "@apollo/client/react";
import {FC, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {Edge, Node, NodeMouseHandler, ReactFlowProvider, useEdgesState, useNodesState} from "reactflow";
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
    GetColumnsBatchDocument,
} from '@graphql';
import {useSourceContract} from "../../hooks/useSourceContract";
import {useAppSelector} from "../../store/hooks";
import {StorageUnitGraphCard} from "../storage-unit/storage-unit";
import {useTranslation} from '@/hooks/use-translation';
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
import {buildSourceScopeRef, getObjectNameFromRef} from "../../utils/source-refs";

type GraphStorageUnit = NonNullable<GetGraphQuery["Graph"]>[number]["Unit"];

// Helper function to group storage units by type
function groupByType(units: GraphStorageUnit[]) {
    const groups: Record<string, GraphStorageUnit[]> = {};
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
    storageUnits: GraphStorageUnit[];
    unitsLoading: boolean;
}

const GraphSidebar: FC<GraphSidebarProps> = ({
    current,
    search,
    setSearch,
    selectedUnits,
    setSelectedUnits,
    storageUnits,
    unitsLoading
}) => {
    const { t } = useTranslation('pages/graph');
    const { storageUnitLabel } = useSourceContract(current?.Type);
    const children = useMemo(() => {
        const units = storageUnits
            .filter((u: GraphStorageUnit) => u.Name.toLowerCase().includes(search.trim().toLowerCase()));
        const groups = groupByType(units);
        const groupEntries = Object.entries(groups);
        if (groupEntries.length === 0) {
            return <div className="text-sm text-muted-foreground px-2">{t('noItems')}</div>;
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
    }, [search, selectedUnits, setSelectedUnits, storageUnits, t]);

    return (
        <div className="dark flex grow">
            <SidebarComponent variant="embed" className="w-64 h-full flex flex-col">
                <SidebarContent data-testid="graph-sidebar-content">
                    <SidebarHeader>
                        <h1 className="text-lg font-semibold pt-8 px-4">
                            {storageUnitLabel}
                        </h1>
                    </SidebarHeader>
                    <div className="px-4">
                        <SearchInput
                            value={search}
                            onChange={e => setSearch(e.target.value)}
                            placeholder={t('searchTables')}
                            aria-label={t('searchTables')}
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
    const { t } = useTranslation('pages/graph');
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const reactFlowRef = useRef<IGraphInstance>();
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const { item, singularStorageUnitLabel, storageUnitLabel } = useSourceContract(current?.Type);
    const navigate = useNavigate();
    const [search, setSearch] = useState("");
    const [selectedUnits, setSelectedUnits] = useState<Set<string>>(new Set());
    const [isInitialized, setIsInitialized] = useState(false);
    const [tableColumns, setTableColumns] = useState<Record<string, any[]>>({});
    const [loadingColumns, setLoadingColumns] = useState<Record<string, boolean>>({});

    const [fetchColumnsBatch] = useLazyQuery(GetColumnsBatchDocument);
    const graphScopeRef = useMemo(() => buildSourceScopeRef(item, current, schema), [current, item, schema]);
    const shouldSkipGraph = !current || !item?.contract || (item.contract.GraphScopeKind != null && graphScopeRef == null);
    const graphQueryOptions = shouldSkipGraph
        ? skipToken
        : {
            variables: {
                ref: graphScopeRef,
            },
        };

    const {
        data: graphQueryData,
        loading: graphLoading,
        refetch: refetchGraph
    } = useQuery<GetGraphQuery, GetGraphQueryVariables>(GetGraphDocument, graphQueryOptions);
    const graphData = useMemo(() => {
        return (graphQueryData?.Graph ?? []) as GetGraphQuery["Graph"];
    }, [graphQueryData?.Graph]);
    const storageUnits = useMemo(() => {
        return graphData.map(node => node.Unit);
    }, [graphData]);

    // Clear graph-specific UI state when the connection context changes.
    const currentProfileId = current?.Id;
    const currentDatabase = current?.Database;
    useEffect(() => {
        setSelectedUnits(new Set());
        setTableColumns({});
        setLoadingColumns({});
    }, [currentProfileId, currentDatabase, schema]);

    // Refetch when the user switches profile or database while already on the graph page.
    // The initial fetch is handled by useQuery's variables changing.
    const prevProfileRef = useRef(currentProfileId);
    const prevDatabaseRef = useRef(currentDatabase);
    useEffect(() => {
        const profileChanged = prevProfileRef.current !== currentProfileId;
        const databaseChanged = prevDatabaseRef.current !== currentDatabase;
        prevProfileRef.current = currentProfileId;
        prevDatabaseRef.current = currentDatabase;
        if ((profileChanged || databaseChanged) && !shouldSkipGraph) {
            refetchGraph();
        }
    }, [currentProfileId, currentDatabase, shouldSkipGraph, refetchGraph]);

    // Default selection logic: auto-select up to 10 units
    useEffect(() => {
        const units = storageUnits;
        if (units.length === 0) return;
        setSelectedUnits(prev => {
            if (prev.size > 0) return prev;
            const toSelect = units.slice(0, 10);
            return new Set(toSelect.map(u => u.Name));
        });
    }, [storageUnits]);

    const tableColumnsRef = useRef(tableColumns);
    tableColumnsRef.current = tableColumns;
    const loadingColumnsRef = useRef(loadingColumns);
    loadingColumnsRef.current = loadingColumns;

    const loadColumnsForRefs = useCallback((refs: GraphStorageUnit["Ref"][]) => {
        const refsToFetch = refs.filter(ref => {
            const unitName = getObjectNameFromRef(ref);
            return !(unitName in tableColumnsRef.current) && !loadingColumnsRef.current[unitName];
        });
        if (refsToFetch.length === 0) {
            return;
        }

        const unitNames = refsToFetch.map(ref => getObjectNameFromRef(ref));
        setLoadingColumns(prev => {
            const next = { ...prev };
            for (const unitName of unitNames) {
                next[unitName] = true;
            }
            return next;
        });

        fetchColumnsBatch({
            variables: { refs: refsToFetch },
        }).then(result => {
            const batch = result.data?.ColumnsBatch;
            if (batch) {
                setTableColumns(prev => {
                    const next = { ...prev };
                    for (const item of batch) {
                        next[getObjectNameFromRef(item.StorageUnit)] = item.Columns;
                    }
                    return next;
                });
            }
        }).catch(error => {
            console.error('Failed to fetch columns batch:', error);
        }).finally(() => {
            setLoadingColumns(prev => {
                const next = { ...prev };
                for (const unitName of unitNames) {
                    delete next[unitName];
                }
                return next;
            });
        });
    }, [fetchColumnsBatch]);

    useEffect(() => {
        const refs = storageUnits
            .filter(unit => selectedUnits.has(unit.Name))
            .map(unit => unit.Ref);
        loadColumnsForRefs(refs);
    }, [loadColumnsForRefs, selectedUnits, storageUnits]);

    const handleNodeClick = useCallback<NodeMouseHandler>((_, node) => {
        const ref = node.data?.Ref;
        if (!ref) {
            return;
        }
        loadColumnsForRefs([ref]);
    }, [loadColumnsForRefs]);

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
            const columns = tableColumns[node.Unit.Name] || [];

            // Calculate node height based on content
            // Be generous with spacing to prevent any overlap
            // Base height: title + button + padding = 200px
            // Each item (metadata + columns): ~50px per item (includes row padding)
            const metadataCount = node.Unit.Attributes?.length || 0;
            const columnCount = columns.length;
            const itemCount = metadataCount + columnCount;
            const calculatedHeight = Math.max(250, 200 + (itemCount * 50));

                newNodes.push(createNode({
                    id: node.Unit.Name,
                    type: GraphElements.StorageUnit,
                    data: {
                        ...node.Unit,
                        columns: columns,
                        columnsLoading: loadingColumns[node.Unit.Name] || false,
                    },
                    width: 400,
                    height: calculatedHeight,
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
                        ...(sourceHandle != null && { sourceHandle }),
                        ...(targetHandle != null && { targetHandle }),
                    };

                    newEdgesSet.add(edgeId);
                    newEdges.push(newEdge);
                }
            }
        }

        return { computedNodes: newNodes, computedEdges: newEdges };
    }, [graphData, loadingColumns, selectedUnits, tableColumns]);

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
    }, [isInitialized, computedNodes, computedEdges]);

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
            storageUnits={storageUnits}
            unitsLoading={graphLoading}
        />
    }>
        <div className="flex-1 h-full">
            <ReactFlowProvider>
                {
                    !graphLoading && nodes.length === 0
                        ? <EmptyState
                            icon={<RectangleGroupIcon className="w-4 h-4" />}
                            title={t('noNodesTitle')}
                            description={t('noNodesDescription', { storageUnit: storageUnitLabel.toLowerCase() })}>
                            <Button onClick={() => navigate(InternalRoutes.Dashboard.StorageUnit.path + "?create=true")}>
                                {t('createButton', { storageUnit: singularStorageUnitLabel })}
                            </Button>
                        </EmptyState>
                        : <Graph nodes={nodes} edges={edges} nodeTypes={nodeTypes}
                                setNodes={setNodes} setEdges={setEdges}
                                onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
                                minZoom={0.1}
                                onNodeClick={handleNodeClick}
                                onReady={handleOnReady} />
                }
            </ReactFlowProvider>
        </div>
    </InternalPage>
}
