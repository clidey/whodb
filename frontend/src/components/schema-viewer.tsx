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

import {
    SearchInput,
    Sidebar as SidebarComponent,
    SidebarContent,
    SidebarGroup,
    SidebarHeader,
    toTitleCase,
    Tree,
    TreeDataItem,
} from "@clidey/ux";
import { StorageUnit, useGetStorageUnitsQuery } from "@graphql";
import { FolderIcon, TableCellsIcon } from "@heroicons/react/24/outline";
import { FC, useCallback, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { InternalRoutes } from "../config/routes";
import { useAppSelector } from "../store/hooks";
import { databaseTypesThatUseDatabaseInsteadOfSchema } from "../utils/database-features";
import { getDatabaseStorageUnitLabel } from "../utils/functions";
import { Loading } from "./loading";

function groupByType(units: StorageUnit[]) {
    const groups: Record<string, any[]> = {};
    for (const unit of units) {
        const type = toTitleCase(unit.Attributes.find(a => a.Key === "Type")?.Value ?? "");
        if (type === "") continue; // Ignore grouping if empty
        if (!groups[type]) groups[type] = [];
        groups[type].push(unit);
    }
    return groups;
}

export const SchemaViewer: FC = () => {
    const current = useAppSelector(state => state.auth.current);
    const selectedSchema = useAppSelector(state => state.database.schema);
    const navigate = useNavigate();
    const state = useLocation().state as { unit: StorageUnit } | undefined;

    // Search state
    const [search, setSearch] = useState("");

    // Query for storage units (tables, views, etc.)
    const { data, loading } = useGetStorageUnitsQuery({
        variables: {
            schema: databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type) ? current?.Database ?? selectedSchema : selectedSchema,
        },
        skip: !current || !selectedSchema,
    });

    // Group storage units by type for tree display, with search filter
    const treeData: TreeDataItem[] = useMemo(() => {
        if (!data?.StorageUnit) return [];
        const grouped = groupByType(data.StorageUnit);

        // If searching, flatten all units and filter by name, then group again
        if (search.trim() !== "") {
            const searchLower = search.trim().toLowerCase();
            // Flatten all units
            const filteredUnits = data.StorageUnit.filter(unit =>
                unit.Name.toLowerCase().includes(searchLower)
            );
            const filteredGrouped = groupByType(filteredUnits);
            return Object.entries(filteredGrouped).map(([type, units]) => ({
                id: type,
                name: type,
                icon: FolderIcon as TreeDataItem["icon"],
                children: units.map(unit => ({
                    id: unit.Name,
                    name: unit.Name,
                    icon: TableCellsIcon as TreeDataItem["icon"],
                })),
            }));
        }

        // Default: show all grouped
        return Object.entries(grouped).map(([type, units]) => ({
            id: type,
            name: type,
            icon: FolderIcon as TreeDataItem["icon"],
            children: units.map(unit => ({
                id: unit.Name,
                name: unit.Name,
                icon: TableCellsIcon as TreeDataItem["icon"],
            })),
        }));
    }, [data, search]);

    const handleSelect = useCallback((item: TreeDataItem | undefined) => {
        // Only leaf nodes (tables) are selectable
        const tableId = item?.id;
        if (tableId == null || tableId === state?.unit.Name) {
            return
        }
        const unit = data?.StorageUnit.find(u => u.Name === tableId);
        if (unit == null) {
            return;
        }
        navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
            state: {
                unit,
            },
        });
    }, [navigate, state, data]);

    if (treeData.length === 0) {
        return null;
    }

    return (
        <div className="flex h-full dark" data-testid="schema-viewer">
            <SidebarComponent variant="embed" className="w-64 h-full flex flex-col">
                <SidebarContent>
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
                            loading ? (
                                <div className="flex-1 flex items-center justify-center">
                                    <Loading />
                                </div>
                            ) : (
                                <Tree
                                    className="flex-1 overflow-y-auto"  
                                    data={treeData}
                                    initialSelectedItemId={state?.unit.Name}
                                    onSelectChange={handleSelect}
                                    expandAll
                                />
                            )
                        }
                    </SidebarGroup>
                </SidebarContent>
            </SidebarComponent>
        </div>
    );
};