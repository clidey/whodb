import {
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
import { FC, useCallback, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { InternalRoutes } from "../config/routes";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import { getDatabaseStorageUnitLabel } from "../utils/functions";
import { Loading } from "./loading";

function groupByType(units: StorageUnit[]) {
    const groups: Record<string, any[]> = {};
    for (const unit of units) {
        const type = toTitleCase(unit.Attributes.find(a => a.Key === "Type")?.Value ?? "");
        console.log(type);
        if (!groups[type]) groups[type] = [];
        groups[type].push(unit);
    }
    return groups;
}

export const SchemaViewer: FC = () => {
    const current = useAppSelector(state => state.auth.current);
    const selectedSchema = useAppSelector(state => state.database.schema);
    const dispatch = useAppDispatch();
    const { storageUnitId } = useParams();
    const navigate = useNavigate();

    // Query for storage units (tables, views, etc.)
    const { data, loading } = useGetStorageUnitsQuery({
        variables: {
            schema: selectedSchema,
        },
        skip: !current || !selectedSchema,
    });

    // Group storage units by type for tree display
    const treeData: TreeDataItem[] = useMemo(() => {
        if (!data?.StorageUnit) return [];
        const grouped = groupByType(data.StorageUnit);
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
    }, [data]);

    const handleSelect = useCallback((item: TreeDataItem | undefined) => {
        // Only leaf nodes (tables) are selectable
        const tableId = item?.id;
        if (tableId && tableId !== storageUnitId) {
            navigate(InternalRoutes.Dashboard.ExploreStorageUnit(tableId).path);
        }
    }, [navigate, storageUnitId]);

    return (
        <div className="flex h-full">
            <SidebarComponent variant="embed" className="w-64 h-full flex flex-col">
                <SidebarContent>
                    <SidebarHeader>
                        <h1 className="text-lg font-semibold">
                            Tables
                        </h1>
                    </SidebarHeader>
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
                                    initialSelectedItemId={storageUnitId}
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