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

import {useLazyQuery, useMutation, useQuery} from "@apollo/client/react";
import {
    Badge,
    Button,
    Checkbox,
    cn,
    Input,
    Label,
    SearchInput,
    Separator,
    SheetTitle,
    StackList,
    StackListItem,
    Table,
    TableCell,
    TableHead,
    TableHeader,
    TableHeadRow,
    TableRow,
    Tabs,
    TabsList,
    TabsTrigger,
    toast,
    VirtualizedTableBody
} from '@clidey/ux';
import {TypeSelector} from "../../components/type-selector";
import {findColumnTypeDefinition} from "../../utils/source-column-types";
import {
    AddStorageUnitDocument,
    DataShape,
    GetColumnsBatchDocument,
    GetGraphQuery,
    GetStorageUnitsQuery,
    GetStorageUnitsDocument,
    RecordInput,
    SourceSchemaFidelity,
    SourceAction,
} from '@graphql';
import {
    ArrowPathRoundedSquareIcon,
    Bars3Icon,
    CheckCircleIcon,
    CircleStackIcon,
    CommandLineIcon,
    InformationCircleIcon,
    MagnifyingGlassIcon,
    PlusCircleIcon,
    Squares2X2Icon,
    TableCellsIcon,
    XCircleIcon,
    XMarkIcon
} from '../../components/heroicons';
import classNames from "classnames";
import {FC, useCallback, useEffect, useMemo, useState} from "react";
import {useLocation, useNavigate, useSearchParams} from "react-router-dom";
import {Handle, Position} from "reactflow";
import {Card, ExpandableCard} from "../../components/card";
import {IGraphCardProps} from "../../components/graph/graph";
import {Loading, LoadingPage} from "../../components/loading";
import {InternalPage} from "../../components/page";
import {InternalRoutes} from "../../config/routes";
import {useSourceContract} from "../../hooks/useSourceContract";
import {trackFrontendEvent} from "../../config/posthog";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {Tip} from '../../components/tip';
import {SettingsActions} from '../../store/settings';
import {useTranslation} from '../../hooks/use-translation';
import {buildSourceParentObjectRef, buildSourceParentRef} from '../../utils/source-refs';
import {formatAttributeValue} from '../../utils/functions';
import { findSourceObjectType, type SourceTypeItem } from '../../config/source-types';

type SourceBrowserObject = GetStorageUnitsQuery['StorageUnit'][number];

type StorageBrowserState = {
    parent?: SourceBrowserObject;
    trail?: SourceBrowserObject[];
};

function isTabularSourceObject(
    item: SourceTypeItem | undefined,
    unit: SourceBrowserObject
): boolean {
    const dataShape = findSourceObjectType(item, unit.Kind)?.DataShape;
    return dataShape === DataShape.Tabular || dataShape === DataShape.Document;
}

function nextBrowserState(
    trail: SourceBrowserObject[],
    unit: SourceBrowserObject
): StorageBrowserState {
    return {
        parent: unit,
        trail: [...trail, unit],
    };
}

const StorageUnitCard: FC<{
    unit: SourceBrowserObject;
    trail: SourceBrowserObject[];
}> = ({ unit, trail }) => {
    const [expanded, setExpanded] = useState(false);
    const navigate = useNavigate();
    const { t } = useTranslation('pages/storage-unit');
    const current = useAppSelector(state => state.auth.current);
    const { item, schemaFidelity } = useSourceContract(current?.Type);
    const [columns, setColumns] = useState<any[] | undefined>(undefined);
    const [columnsLoading, setColumnsLoading] = useState(false);
    const [fetchColumnsBatch] = useLazyQuery(GetColumnsBatchDocument);
    const canBrowse = unit.Actions.includes(SourceAction.Browse) && unit.HasChildren;
    const shouldFetchColumns = isTabularSourceObject(item, unit);

    const handleNavigateToObject = useCallback(() => {
        if (canBrowse) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path, {
                state: nextBrowserState(trail, unit),
            });
            return;
        }

        navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
            state: {
                unit,
                parentRef: buildSourceParentObjectRef(item, unit.Ref),
                trail,
            },
        });
    }, [canBrowse, item, navigate, trail, unit]);

    const fetchColumns = useCallback(() => {
        if (!shouldFetchColumns || columns !== undefined) return;
        setColumnsLoading(true);
        fetchColumnsBatch({
            variables: { refs: [unit.Ref] },
        }).then(result => {
            const batch = result.data?.ColumnsBatch;
            if (batch && batch.length > 0 && batch[0].Columns.length > 0) {
                setColumns(batch[0].Columns);
            } else {
                setColumns([]);
            }
        }).catch(() => {
            setColumns([]);
        }).finally(() => {
            setColumnsLoading(false);
        });
    }, [columns, fetchColumnsBatch, shouldFetchColumns, unit.Ref]);

    const handleSetExpanded = useCallback((status: boolean) => {
        setExpanded(status);
        if (status && shouldFetchColumns) requestAnimationFrame(fetchColumns);
    }, [fetchColumns, shouldFetchColumns]);

    useEffect(() => {
        if (expanded && shouldFetchColumns && columns === undefined) {
            requestAnimationFrame(fetchColumns);
        }
    }, [columns, expanded, fetchColumns, shouldFetchColumns]);

    const handleExpand = useCallback(() => {
        const next = !expanded;
        setExpanded(next);
        if (next && shouldFetchColumns) requestAnimationFrame(fetchColumns);
    }, [expanded, fetchColumns, shouldFetchColumns]);

    const [introAttributes, expandedAttributes] = useMemo(() => {
        return [ unit.Attributes.slice(0,4), unit.Attributes.slice(4) ];
    }, [unit.Attributes]);

    return (<ExpandableCard key={unit.Name} isExpanded={expanded} setExpanded={handleSetExpanded} icon={<TableCellsIcon className="w-4 h-4" />} className={cn({
        "shadow-2xl exploring-storage-unit": expanded,
    })} data-testid="storage-unit-card" data-table-name={unit.Name}>
        <div className="flex flex-col grow mt-2 cursor-pointer" data-testid="storage-unit-card" data-table-name={unit.Name}>
            <div className="flex flex-col grow mb-2 w-full overflow-x-hidden">
                <Tip className="w-fit">
                    <h1
                        className="text-sm font-semibold mb-2 overflow-hidden text-ellipsis whitespace-nowrap max-w-[190px]"
                        data-testid="storage-unit-name"
                        title={unit.Name}
                    >
                        {unit.Name}
                    </h1>
                    <p className="text-xs">{unit.Name}</p>
                </Tip>
                {
                    introAttributes.slice(0,2).map(attribute => (
                        <p key={attribute.Key} className="text-xs">{attribute.Key}: {formatAttributeValue(attribute.Key, attribute.Value)}</p>
                    ))
                }
            </div>
            <div className="flex flex-row justify-end gap-xs" onClick={(e) => e.stopPropagation()}>
                <Button onClick={handleExpand} data-testid="explore-button" variant="secondary">
                    <MagnifyingGlassIcon className="w-4 h-4" /> {t('describe')}
                </Button>
                <Button onClick={handleNavigateToObject} data-testid="data-button" variant="secondary">
                    <CircleStackIcon className="w-4 h-4" /> {t('data')}
                </Button>
            </div>
        </div>
        <div className="flex flex-col grow gap-lg justify-between h-full overflow-y-auto">
            <SheetTitle className="flex items-center gap-2 mb-4">
                <TableCellsIcon className="w-5 h-5" />
                {unit.Name}
            </SheetTitle>
            {schemaFidelity === SourceSchemaFidelity.Sampled && (
                <div className="mb-2" data-testid="sampled-schema-warning">
                    <div className="flex items-center gap-xs text-sm">
                        <InformationCircleIcon className="w-4 h-4" />
                        <span>{t('sampledSchemaBadge')}</span>
                    </div>
                </div>
            )}
            <div className="w-full" data-testid="explore-fields">
                <div className="flex flex-col gap-xs2">
                    <StackList>
                        {/* Metadata attributes (Type, Total Size, etc.) */}
                        {
                            introAttributes.map(attribute => {
                                const display = formatAttributeValue(attribute.Key, attribute.Value);
                                return (
                                    <div key={attribute.Key} data-field-key={attribute.Key} data-field-value={display}>
                                        <StackListItem item={attribute.Key}>
                                            {display}
                                        </StackListItem>
                                    </div>
                                );
                            })
                        }
                        {
                            expandedAttributes.map(attribute => {
                                const display = formatAttributeValue(attribute.Key, attribute.Value);
                                return (
                                    <div key={attribute.Key} data-field-key={attribute.Key} data-field-value={display}>
                                        <StackListItem item={attribute.Key}>
                                            {display}
                                        </StackListItem>
                                    </div>
                                );
                            })
                        }
                    </StackList>
                    {columnsLoading && shouldFetchColumns && (
                        <div className="mt-8">
                            <Loading hideText={true} />
                        </div>
                    )}
                    {!columnsLoading && shouldFetchColumns && columns && columns.length > 0 && (
                        <div className="mt-8">
                            <h3 className="text-xs font-semibold uppercase text-muted-foreground">{t('columns')}</h3>
                            <StackList>
                                {columns.map(col => {
                                    const isForeignKey = col.IsForeignKey;
                                    return (
                                        <div key={col.Name} data-field-key={col.Name} data-field-value={col.Type?.toLowerCase()} data-is-foreign-key={isForeignKey || undefined}>
                                            <StackListItem item={isForeignKey ?
                                                <Badge className="text-lg" data-testid="foreign-key-attribute" variant="secondary">{col.Name}</Badge> : col.Name}>
                                                {col.Type?.toLowerCase()}
                                            </StackListItem>
                                        </div>
                                    );
                                })}
                            </StackList>
                        </div>
                    )}
                </div>
            </div>
            <div className="flex items-end grow">
                <Button onClick={handleNavigateToObject} data-testid="data-button" variant="secondary" className="w-full">
                    <CircleStackIcon className="w-4 h-4" /> {t('data')}
                </Button>
            </div>
        </div>
    </ExpandableCard>);
}

export const StorageUnitPage: FC = () => {
    const location = useLocation();
    const navigate = useNavigate();
    const [searchParams,] = useSearchParams();
    const [create, setCreate] = useState(searchParams.get("create") === "true");
    const [storageUnitName, setStorageUnitName] = useState("");
    const [fields, setFields] = useState<RecordInput[]>([ {Key: "", Value: "", Extra: [] }]);
    const [error, setError] = useState<string>();
    let schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const {
        item,
        isNoSQL,
        singularStorageUnitLabel,
        storageUnitLabel,
        supportsModifiers,
        supportsScratchpad,
        usesDatabaseInsteadOfSchema,
    } = useSourceContract(current?.Type);
    const view = useAppSelector(state => state.settings.storageUnitView);
    const [addStorageUnit,] = useMutation(AddStorageUnitDocument);
    const [expandedUnit, setExpandedUnit] = useState<string | null>(null);
    const [expandedUnitColumns, setExpandedUnitColumns] = useState<{ name: string; columns: any[] } | null>(null);
    const [expandedUnitColumnsLoading, setExpandedUnitColumnsLoading] = useState(false);
    const [fetchColumnsBatchForList] = useLazyQuery(GetColumnsBatchDocument);
    const dispatch = useAppDispatch();
    const { t } = useTranslation('pages/storage-unit');
    const { t: tCommon } = useTranslation('common');
    const locationState = location.state as StorageBrowserState | undefined;
    const trail = locationState?.trail ?? [];
    const currentParent = locationState?.parent;

    useEffect(() => {
        void trackFrontendEvent('ui.storage_unit_viewed', {
            database_type: current?.Type ?? 'unknown',
            view_mode: view,
        });
    }, [current?.Type, trackFrontendEvent, view]);

    // Databases like MySQL, MariaDB, ClickHouse, MongoDB use database name as schema parameter since they treat database=schema
    if (usesDatabaseInsteadOfSchema) {
        schema = current?.Database ?? '';
    }

    const initialParentRef = useMemo(() => buildSourceParentRef(item, current, schema), [current, item, schema]);
    const canCreateObjects = useMemo(() => {
        const parentKind = currentParent?.Kind ?? initialParentRef?.Kind;
        if (parentKind) {
            return findSourceObjectType(item, parentKind)?.Actions.includes(SourceAction.CreateChild) ?? false;
        }
        return item?.contract?.RootActions?.includes(SourceAction.CreateChild) ?? false;
    }, [currentParent?.Kind, initialParentRef?.Kind, item]);
    const parentRef = currentParent?.Ref ?? initialParentRef;

    const { loading, data, refetch } = useQuery(GetStorageUnitsDocument, {
        variables: {
            parent: parentRef,
        },
    });

    const storageUnits = useMemo(() => {
        return (data?.StorageUnit ?? []) as SourceBrowserObject[];
    }, [data?.StorageUnit]);

    // Refetch storage units when the connection context changes (profile switch or database switch)
    const currentProfileId = current?.Id;
    const currentDatabase = current?.Database;
    useEffect(() => {
        if (currentProfileId) {
            refetch();
        }
    }, [currentProfileId, currentDatabase, refetch]);

    // Lazy-load columns for list view expanded detail
    useEffect(() => {
        if (!expandedUnit) return;
        if (expandedUnitColumns?.name === expandedUnit) return;
        const expandedSourceUnit = storageUnits.find(unit => unit.Name === expandedUnit);
        if (!expandedSourceUnit) return;
        if (!isTabularSourceObject(item, expandedSourceUnit)) {
            setExpandedUnitColumns({ name: expandedUnit, columns: [] });
            return;
        }
        setExpandedUnitColumnsLoading(true);
        fetchColumnsBatchForList({
            variables: { refs: [expandedSourceUnit.Ref] },
        }).then(result => {
            const batch = result.data?.ColumnsBatch;
            if (batch && batch.length > 0 && batch[0].Columns.length > 0) {
                setExpandedUnitColumns({ name: expandedUnit, columns: batch[0].Columns });
            } else {
                setExpandedUnitColumns({ name: expandedUnit, columns: [] });
            }
        }).catch(() => {
            setExpandedUnitColumns({ name: expandedUnit, columns: [] });
        }).finally(() => {
            setExpandedUnitColumnsLoading(false);
        });
    }, [expandedUnit, expandedUnitColumns?.name, fetchColumnsBatchForList, item, storageUnits]);

    const [filterValue, setFilterValue] = useState("");

    const routes = useMemo(() => {
        return [
            {
                ...InternalRoutes.Dashboard.StorageUnit,
                name: storageUnitLabel,
            },
        ];
    }, [storageUnitLabel]);

    const handleCreate = useCallback(() => {
        const next = !create;
        setCreate(next);
        void trackFrontendEvent('ui.storage_unit_create_toggle', {
            database_type: current?.Type ?? 'unknown',
            open: next,
        });
    }, [create, current?.Type, trackFrontendEvent]);

    const handleSubmit = useCallback(() => {
        if (storageUnitName.length === 0) {
            return setError(t('nameRequired'));
        }
        if (!isNoSQL && fields.some(field => field.Key.length === 0 || field.Value.length === 0)) {
            return setError(t('fieldsCannotBeEmpty'));
        }
        setError(undefined);
        addStorageUnit({
            variables: {
                parent: parentRef,
                storageUnit: storageUnitName,
                fields,
            },
            onCompleted() {
                const message = t('createSuccessMessage').replace('{storageUnit}', `${singularStorageUnitLabel} ${storageUnitName}`);
                toast.success(message);
                void trackFrontendEvent('ui.storage_unit_created', {
                    database_type: current?.Type ?? 'unknown',
                    field_count: fields.length,
                });
                setStorageUnitName("");
                setFields([]);
                refetch();
                setCreate(false);
            },
            onError(e) {
                toast.error(e.message);
            },
        });
    }, [addStorageUnit, current?.Type, fields, isNoSQL, parentRef, refetch, singularStorageUnitLabel, storageUnitName, t, trackFrontendEvent]);

    const handleAddField = useCallback(() => {
        setFields(f => [...f, { Key: "", Value: "", Extra: [] }]);
        void trackFrontendEvent('ui.storage_unit_field_added', {
            database_type: current?.Type ?? 'unknown',
        });
    }, [current?.Type, trackFrontendEvent]);

    const handleFieldValueChange = useCallback((type: string, index: number, value: string | boolean) => {
        setFields(f => {
            const newF = structuredClone(f);
            if (type === "Key" || type === "Value") {
                newF[index][type] = value as string;
            } else {
                if (newF[index].Extra == null) {
                    newF[index].Extra = [];
                }
                const extraIndex = newF[index].Extra.findIndex(extra => extra.Key === type);
                if (value && extraIndex === -1) {
                    newF[index].Extra = [...newF[index].Extra, { Key: type, Value: "true" }];
                } else {
                    newF[index].Extra = newF[index].Extra.filter((_, i) => i !== extraIndex);
                }
            }
            return newF;
        });
    }, []);

    const handleRemove = useCallback((index: number) => {
        if (fields.length <= 1) {
            return;
        }
        setFields(f => {
            const newF = [...f];
            newF.splice(index, 1);
            return newF;
        })
    }, [fields.length]);

    const filterStorageUnits = useMemo(() => {
        const lowerCaseFilterValue = filterValue.toLowerCase();
        return storageUnits.filter(unit => (unit.Name ?? "").toLowerCase().includes(lowerCaseFilterValue))
            .sort((a, b) => {
                if (a.HasChildren !== b.HasChildren) {
                    return a.HasChildren ? -1 : 1;
                }
                return (a.Name ?? "").localeCompare(b.Name ?? "");
            });
    }, [filterValue, storageUnits]);

    const showModifiers = useMemo(() => {
        return supportsModifiers;
    }, [supportsModifiers]);

    const handleOpenUnit = useCallback((unit: SourceBrowserObject) => {
        if (unit.Actions.includes(SourceAction.Browse) && unit.HasChildren) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path, {
                state: nextBrowserState(trail, unit),
            });
            return;
        }

        navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
            state: {
                unit,
                parentRef: buildSourceParentObjectRef(item, unit.Ref),
                trail,
            },
        });
    }, [item, navigate, trail]);

    const previousBrowserState = useMemo<StorageBrowserState | undefined>(() => {
        if (trail.length === 0) {
            return undefined;
        }
        if (trail.length === 1) {
            return {};
        }
        return {
            parent: trail[trail.length - 2],
            trail: trail.slice(0, -1),
        };
    }, [trail]);

    const sharedAttributeKeys = useMemo(() => {
        if (storageUnits.length === 0) {
            return [];
        }

        // Get attributes that exist in ALL storage units (intersection of all attributes)
        // Preserve the order from the first unit (backend returns metadata like Type first)
        const firstUnitKeys = storageUnits[0]?.Attributes.map(attr => attr.Key) ?? [];

        return firstUnitKeys.filter(key =>
            storageUnits.every(unit =>
                unit.Attributes.some(attr => attr.Key === key)
            )
        );
    }, [storageUnits]);

    if (loading) {
        return <InternalPage routes={routes}>
            <LoadingPage />
        </InternalPage>
    }

    return <InternalPage routes={routes}>
        <div className="flex w-full h-fit my-2 gap-lg justify-between">
            <div className="flex justify-between items-center">
                {previousBrowserState != null && (
                    <Button
                        variant="secondary"
                        className="mr-2"
                        onClick={() => navigate(InternalRoutes.Dashboard.StorageUnit.path, { state: previousBrowserState })}
                    >
                        {tCommon('back')}
                    </Button>
                )}
                <SearchInput value={filterValue} onChange={e => setFilterValue(e.target.value)} placeholder={t('searchPlaceholder')} />
            </div>
            <div className="flex items-center gap-2">
                {
                    supportsScratchpad &&
                    <Button onClick={() => navigate(InternalRoutes.RawExecute.path)} data-testid="scratchpad-button" variant="secondary">
                        <CommandLineIcon className="w-4 h-4" /> {t('scratchpad')}
                    </Button>
                }
                <Tabs value={view} onValueChange={value => dispatch(SettingsActions.setStorageUnitView(value as 'list' | 'card'))}>
                    <TabsList>
                        <Tip className="w-fit">
                            <TabsTrigger value="card" data-testid="icon-button" aria-label={t('cardView')}><Squares2X2Icon className="w-4 h-4" /></TabsTrigger>
                            <p>{t('cardView')}</p>
                        </Tip>
                        <Tip className="w-fit">
                            <TabsTrigger value="list" data-testid="icon-button" aria-label={t('listView')}><Bars3Icon className="w-4 h-4" /></TabsTrigger>
                            <p>{t('listView')}</p>
                        </Tip>
                    </TabsList>
                </Tabs>
            </div>
        </div>
        <div className={cn("flex flex-wrap gap-4", {
            "hidden": view !== "card",
        })} data-testid="storage-unit-card-list">
            {canCreateObjects && <ExpandableCard className="overflow-visible min-w-[200px] max-w-[700px] h-full" icon={<PlusCircleIcon className="w-4 h-4" />} isExpanded={create} setExpanded={setCreate} tag={<Badge variant="destructive">{error}</Badge>}>
                <div className="flex flex-col grow h-full justify-between mt-2 gap-2" data-testid="create-storage-unit-card">
                    <h1 className="text-lg"><span className="prefix-create-storage-unit">{t('createTitle', { storageUnit: singularStorageUnitLabel })}</span></h1>
                    <Button className="self-end" onClick={e => { e.stopPropagation(); handleCreate(); }} variant="secondary">
                        <PlusCircleIcon  className='w-4 h-4' /> {t('create')}
                    </Button>
                </div>
                <div className="flex grow flex-col my-2 gap-4">
                    <div className="flex flex-col gap-4">
                        <SheetTitle className="flex items-center gap-2">
                            <PlusCircleIcon className="w-5 h-5" />
                            {t('createTitle', { storageUnit: singularStorageUnitLabel })}
                        </SheetTitle>
                        <div className="flex flex-col gap-2">
                            <Label>{t('nameLabel')}</Label>
                            <Input value={storageUnitName} onChange={e => setStorageUnitName(e.target.value)} placeholder={t('namePlaceholder')} />
                        </div>
                        <div className={classNames("flex flex-col gap-sm overflow-y-auto max-h-[75vh]", {
                            "hidden": isNoSQL,
                        })}>
                            <div className="flex flex-col gap-4">
                                {
                                    fields.map((field, index) => (
                                        <div className="flex flex-col gap-lg relative" key={`field-${index}`} data-testid="create-field-card">
                                            <Label>{t('fieldNameLabel')}</Label>
                                            <Input value={field.Key} onChange={e => handleFieldValueChange("Key", index, e.target.value)} placeholder={t('fieldNamePlaceholder')}/>
                                            <Label>{t('fieldTypeLabel')}</Label>
                                            <TypeSelector
                                                sourceType={current?.Type}
                                                value={field.Value}
                                                onChange={value => handleFieldValueChange("Value", index, value)}
                                                placeholder={t('fieldTypePlaceholder')}
                                                searchPlaceholder={t('searchTypePlaceholder')}
                                                buttonProps={{
                                                    "data-testid": `field-type-${index}`,
                                                }}
                                            />

                                            {(() => {
                                                const typeDef = current?.Type && field.Value
                                                    ? findColumnTypeDefinition(field.Value, current.Type)
                                                    : undefined;
                                                return typeDef?.tableModel ? (
                                                    <p className="text-xs text-muted-foreground">
                                                        {t('aggregateKeyHint')}
                                                    </p>
                                                ) : null;
                                            })()}

                                            {showModifiers && (
                                                <>
                                                    <Label>{t('modifiersLabel')}</Label>
                                                    <div className="flex items-center w-1/3 justify-start gap-2">
                                                        <Checkbox checked={field.Extra?.find(extra => extra.Key === "Primary") != null} onCheckedChange={() => handleFieldValueChange("Primary", index, !field.Extra?.find(extra => extra.Key === "Primary") != null)}/>
                                                        <Label>{t('primaryModifier')}</Label>
                                                        <Checkbox checked={field.Extra?.find(extra => extra.Key === "Nullable") != null} onCheckedChange={() => handleFieldValueChange("Nullable", index, !field.Extra?.find(extra => extra.Key === "Nullable") != null)}/>
                                                        <Label>{t('nullableModifier')}</Label>
                                                    </div>
                                                </>
                                            )}
                                            {
                                                fields.length > 1 &&
                                                <Button variant="destructive" onClick={() => handleRemove(index)} data-testid="remove-field-button" className="w-full mt-1">
                                                    <XCircleIcon className="w-4 h-4"/> <span>{t('remove')}</span>
                                                </Button>
                                            }
                                            {index !== fields.length - 1 && <Separator className="mt-2" />}
                                        </div>
                                    ))
                                } 
                            </div>
                            <Button className="self-end" onClick={handleAddField} data-testid="add-field-button" variant="secondary">
                                <PlusCircleIcon  className='w-4 h-4' /> {t('addField')}
                            </Button>
                        </div>
                    </div>
                    <div className="flex grow" />
                    <Button onClick={handleSubmit} data-testid="submit-button" className="w-full">
                        <CheckCircleIcon className="w-4 h-4" /> {t('create')}
                    </Button>
                </div>
            </ExpandableCard>}
            {
                storageUnits.length > 0 && filterStorageUnits.map(unit => (
                    <StorageUnitCard key={`${unit.Name}-${unit.Kind}`} unit={unit} trail={trail} />
                ))
            }
        </div>
        <div className={cn("flex flex-wrap gap-lg w-full h-[80vh]", {
            "hidden": view !== "list",
        })}>
            <Table>
                <TableHeader>
                    <TableHeadRow>
                        <TableHead>{t('nameLabel')}</TableHead>
                        {/** Dynamically render shared attribute keys as columns */}
                        {sharedAttributeKeys.map(key => (
                            <TableHead key={key}>{key}</TableHead>
                        ))}
                        <TableHead>{t('actions')}</TableHead>
                    </TableHeadRow>
                </TableHeader>
                <VirtualizedTableBody
                    rowCount={filterStorageUnits.length}
                    rowHeight={40}>
                    {(rowIndex: number) => {
                        const unit = filterStorageUnits[rowIndex];
                        if (!unit) {
                            return null;
                        }
                        const attrMap = Object.fromEntries(unit.Attributes.map(attr => [attr.Key, attr.Value]));
                        return (
                            <TableRow key={unit.Name} className="group">
                                <TableCell>{unit.Name}</TableCell>
                                {sharedAttributeKeys.map(key => (
                                    <TableCell key={key}>{formatAttributeValue(key, attrMap[key])}</TableCell>
                                ))}
                                <TableCell className="relative">
                                    <div className="flex gap-xs opacity-0 group-hover:opacity-100 transition-opacity">
                                        <Button
                                            onClick={() => handleOpenUnit(unit)}
                                            data-testid="data-button"
                                            variant="secondary"
                                            size="sm"
                                            className="!cursor-pointer"
                                        >
                                            <CircleStackIcon className="w-4 h-4" /> {t('data')}
                                        </Button>
                                    </div>
                                </TableCell>
                            </TableRow>
                        );
                    }}
                </VirtualizedTableBody>
            </Table>
            {expandedUnit && (
                <div className="w-full mt-4">
                    {(() => {
                        const unit = filterStorageUnits.find(u => u.Name === expandedUnit);
                        if (!unit) return null;

                        const columns = expandedUnitColumns?.name === unit.Name ? expandedUnitColumns.columns : undefined;
                        const [introAttributes, expandedAttributes] = [unit.Attributes.slice(0,4), unit.Attributes.slice(4)];

                        return (
                            <Card className="p-6">
                                <div className="flex flex-col gap-4">
                                    <h2 className="text-2xl font-bold">{unit.Name}</h2>
                                    <StackList>
                                        {/* Metadata attributes */}
                                        {introAttributes.map(attribute => {
                                            const display = formatAttributeValue(attribute.Key, attribute.Value);
                                            return (
                                                <div key={attribute.Key} data-field-key={attribute.Key} data-field-value={display}>
                                                    <StackListItem item={attribute.Key}>
                                                        {display}
                                                    </StackListItem>
                                                </div>
                                            );
                                        })}
                                        {expandedAttributes.map(attribute => {
                                            const display = formatAttributeValue(attribute.Key, attribute.Value);
                                            return (
                                                <div key={attribute.Key} data-field-key={attribute.Key} data-field-value={display}>
                                                    <StackListItem item={attribute.Key}>
                                                        {display}
                                                    </StackListItem>
                                                </div>
                                            );
                                        })}
                                    </StackList>
                                    {expandedUnitColumnsLoading && <Loading hideText={true} />}
                                    {!expandedUnitColumnsLoading && columns && columns.length > 0 && (
                                        <div>
                                            <h3 className="text-xs font-semibold uppercase text-muted-foreground mb-2">{t('columns')}</h3>
                                            <StackList>
                                                {columns.map((col: any) => {
                                                    const isForeignKey = col.IsForeignKey;
                                                    return (
                                                        <div key={col.Name} data-field-key={col.Name} data-field-value={col.Type?.toLowerCase()} data-is-foreign-key={isForeignKey || undefined}>
                                                            <StackListItem item={isForeignKey ?
                                                                <Badge className="text-lg" data-testid="foreign-key-attribute">{col.Name}</Badge> : col.Name}>
                                                                {col.Type?.toLowerCase()}
                                                            </StackListItem>
                                                        </div>
                                                    );
                                                })}
                                            </StackList>
                                        </div>
                                    )}
                                    <div className="flex gap-sm mt-4">
                                        <Button
                                            onClick={() => handleOpenUnit(unit)}
                                            data-testid="data-button"
                                            variant="secondary"
                                        >
                                            <CircleStackIcon className="w-4 h-4" /> {t('data')}
                                        </Button>
                                        <Button
                                            onClick={() => setExpandedUnit(null)}
                                            variant="outline"
                                        >
                                            <XMarkIcon className="w-4 h-4" /> {t('close')}
                                        </Button>
                                    </div>
                                </div>
                            </Card>
                        );
                    })()}
                </div>
            )}
        </div>
    </InternalPage>
}

type StorageUnitGraphCardData = GetGraphQuery['Graph'][number]['Unit'] & {
    columns?: any[];
    columnsLoading?: boolean;
};

export const StorageUnitGraphCard: FC<IGraphCardProps<StorageUnitGraphCardData>> = ({ data }) => {
    const navigate = useNavigate();
    const { t } = useTranslation('pages/storage-unit');

    const handleNavigateTo = useCallback(() => {
        if (data == null) {
            return;
        }
        navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
            state: {
                unit: data,
            }
        });
    }, [navigate, data]);

    // Attributes contains metadata (Type, Total Size, etc.)
    // Columns contains field definitions with FK/PK info for handles
    const metadataItems = data?.Attributes || [];
    const columnItems = data?.columns || [];
    const columnsLoading = data?.columnsLoading || false;

    if (data == null) {
        return (<Card icon={<ArrowPathRoundedSquareIcon className="w-4 h-4" />}>
            <Loading hideText={true} />
        </Card>)
    }

    return (
        <>
            <Card icon={<CircleStackIcon className="w-4 h-4" />} className="h-fit backdrop-blur-[2px] w-[400px] px-2 py-6">
                <div className="flex flex-col grow mt-2 gap-lg" data-testid="storage-unit-graph-card">
                    <div className="flex flex-col grow">
                        <h2 className="text-3xl font-semibold mb-2 break-words">{data.Name}</h2>
                        <StackList>
                            {/* Show metadata first (Type, Total Size, etc.) */}
                            {
                                metadataItems.map((item: any, index: number) => {
                                    const name = item.Key;
                                    const value = item.Value?.toLowerCase();
                                    return (
                                        <div key={`meta-${name}-${index}`} data-field-key={name} data-field-value={value}>
                                            <StackListItem rowClassName="items-start" item={name}>
                                                {value}
                                            </StackListItem>
                                        </div>
                                    );
                                })
                            }
                            {
                                columnsLoading && (
                                    <div className="py-4">
                                        <Loading hideText={true} />
                                    </div>
                                )
                            }
                            {/* Show columns with FK/PK handles */}
                            {
                                !columnsLoading && columnItems.map((col: any, index: number) => {
                                    const name = col.Name;
                                    const value = col.Type?.toLowerCase();
                                    const isFKColumn = col.IsForeignKey || false;
                                    const isPKColumn = col.IsPrimary || false;

                                    return (
                                        <div key={`col-${name}-${index}`} className="relative" data-field-key={name} data-field-value={value} data-is-foreign-key={isFKColumn || undefined} data-is-primary-key={isPKColumn || undefined}>
                                            {isFKColumn && (
                                                <Handle
                                                    type="source"
                                                    position={Position.Right}
                                                    id={`${data.Name}-${name}`}
                                                    className="right-0 translate-x-6 w-3 h-3 border-2 border-border bg-background dark:bg-background"
                                                />
                                            )}
                                            {isPKColumn && (
                                                <Handle
                                                    type="target"
                                                    position={Position.Left}
                                                    id={`${data.Name}-${name}`}
                                                    className="left-0 -translate-x-6 w-3 h-3 border-2 border-border bg-background dark:bg-background"
                                                />
                                            )}
                                            <StackListItem rowClassName="items-start"
                                                           item={isFKColumn ? <Badge variant="secondary" className="text-lg">{name}</Badge> : name}>
                                                {value}
                                            </StackListItem>
                                        </div>
                                    );
                                })
                            }
                        </StackList>
                    </div>
                    <Button onClick={handleNavigateTo} data-testid="data-button" variant="secondary">
                        <CircleStackIcon className="w-4 h-4" /> {t('data')}
                    </Button>
                </div>
            </Card>
        </>
    );
}
