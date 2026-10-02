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

import { skipToken, useQuery } from "@apollo/client/react";
import {
    SearchInput,
    Sidebar as SidebarComponent,
    SidebarContent,
    SidebarGroup,
    SidebarHeader,
} from "@clidey/ux";
import type { GetStorageUnitsQuery, SourceObjectRefInput} from "@graphql";
import {GetStorageUnitRowsDocument, GetStorageUnitsDocument, SourceAction} from "@graphql";
import {WhoDBChatIcon} from "./whodb-chat-icon";
import type {FC} from "react";
import { useCallback, useEffect, useMemo, useState} from "react";
import {useLocation, useNavigate} from "react-router-dom";
import {InternalRoutes} from "../config/routes";
import {useSourceContract} from "../hooks/useSourceContract";
import {useAppSelector} from "../store/hooks";
import {Loading} from "./loading";
import {useTranslation} from "@/hooks/use-translation";
import {buildSourceParentRef} from "@/utils/source-refs";
import {ph} from "@/utils/privacy";

type SourceBrowserObject = GetStorageUnitsQuery['StorageUnit'][number];

type SchemaViewerProps = {
    parentRef?: SourceObjectRefInput;
    selectedName?: string;
    trail?: SourceBrowserObject[];
};

const SchemaViewerUnit: FC<{unit: SourceBrowserObject; selected: boolean; onSelect: (name: string) => void}> = ({unit, selected, onSelect}) => {
    const metadataCount = unit.Attributes.find(attribute => attribute.Key === 'Count')?.Value;
    const knownCount = metadataCount != null && metadataCount !== 'unknown' ? metadataCount : undefined;
    const {data} = useQuery(GetStorageUnitRowsDocument, knownCount == null && unit.Actions.includes(SourceAction.ViewRows)
        ? {variables: {ref: unit.Ref, pageSize: 1, pageOffset: 0}}
        : skipToken);
    const count = knownCount ?? data?.Row.TotalCount;

    return <button
        type="button"
        className={selected ? 'is-selected' : ''}
        onClick={() => { onSelect(unit.Name); }}
        aria-current={selected ? 'page' : undefined}
    >
        <WhoDBChatIcon name="table" />
        <span>{unit.Name}</span>
        {count != null && <span className="ce-schema-row-count">{count}</span>}
    </button>;
};

export const SchemaViewer: FC<SchemaViewerProps> = ({ parentRef: explicitParentRef, selectedName, trail = [] }) => {
    const { t } = useTranslation('components/schema-viewer');
    const current = useAppSelector(state => state.auth.current);
    const currentType = current?.Type;
    const currentDatabase = current?.Database;
    const selectedSchema = useAppSelector(state => state.database.schema);
    const { item, storageUnitLabel, supportsSchema } = useSourceContract(currentType);
    const navigate = useNavigate();
    const state = useLocation().state as { unit?: SourceBrowserObject } | undefined;

    // Search state
    const [search, setSearch] = useState("");

    const parentRef = explicitParentRef ?? buildSourceParentRef(item, current, selectedSchema);
    const storageUnitsQueryOptions = current && (explicitParentRef != null || !supportsSchema || selectedSchema !== "")
        ? {
            variables: {
                parent: parentRef,
            },
        }
        : skipToken;

    // Query for storage units (tables, views, etc.)
    const {data, loading, refetch} = useQuery(GetStorageUnitsDocument, storageUnitsQueryOptions);

    // Refetch when the connection context changes (profile switch or database switch)
    const currentProfileId = current?.Id;
    useEffect(() => {
        if (currentProfileId) {
            void refetch();
        }
    }, [currentProfileId, currentDatabase, refetch]);

    const storageUnits = useMemo(() => {
        return (data?.StorageUnit ?? []) as SourceBrowserObject[];
    }, [data?.StorageUnit]);

    const filteredUnits = useMemo(() => storageUnits.filter(unit =>
        (unit.Name ?? "").toLowerCase().includes(search.trim().toLowerCase())
    ), [search, storageUnits]);

    const handleSelect = useCallback((tableId: string) => {
        if (tableId == null || tableId === (selectedName ?? state?.unit?.Name)) {
            return
        }
        const unit = storageUnits.find(u => u.Name === tableId);
        if (unit == null) {
            return;
        }
        if (unit.Actions.includes(SourceAction.Browse) && unit.HasChildren) {
            void navigate(InternalRoutes.Dashboard.StorageUnit.path, {
                state: {
                    parent: unit,
                    trail: [...trail, unit],
                },
            });
            return;
        }
        void navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
            state: {
                unit,
                parentRef,
                trail,
            },
        });
    }, [navigate, parentRef, selectedName, state?.unit?.Name, storageUnits, trail]);

    // Only hide sidebar if there's truly no data (no connection, no schema, etc.)
    // Don't hide when search returns empty results
    if (storageUnits.length === 0 && search.trim() === "") {
        return null;
    }

    return (
        <div className="flex h-full ce-schema-viewer" data-testid="schema-viewer">
            <SidebarComponent variant="embed" className="w-64 h-full flex flex-col">
                <SidebarContent>
                    <SidebarHeader>
                        <h1 className="text-lg font-semibold pt-8 px-4">
                            {storageUnitLabel}
                        </h1>
                    </SidebarHeader>
                    <div className="px-4">
                        <SearchInput
                            value={search}
                            onChange={(e) => { setSearch(e.target.value); }}
                            placeholder={t('searchTables')}
                            aria-label={t('searchTables')}
                        />
                    </div>
                    <SidebarGroup>
                        {
                            loading ? (
                                <div className="flex-1 flex items-center justify-center">
                                    <Loading />
                                </div>
                            ) : filteredUnits.length === 0 ? (
                                <div className="flex-1 flex items-center justify-center px-4 text-center text-sm text-muted-foreground mt-4">
                                    {t('noResults')}
                                </div>
                            ) : (
                                <div className={`ce-schema-list ${ph.mask}`}>
                                    <p className="ce-schema-list-label">{currentDatabase} · {storageUnits.length}</p>
                                    {filteredUnits.map(unit => <SchemaViewerUnit
                                        key={unit.Name}
                                        unit={unit}
                                        selected={unit.Name === (selectedName ?? state?.unit?.Name)}
                                        onSelect={handleSelect}
                                    />)}
                                </div>
                            )
                        }
                    </SidebarGroup>
                </SidebarContent>
            </SidebarComponent>
        </div>
    );
};
