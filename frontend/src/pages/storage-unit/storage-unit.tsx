/*
 * // Copyright 2025 Clidey, Inc.
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

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
import {
    DatabaseType,
    RecordInput,
    StorageUnit,
    useAddStorageUnitMutation,
    useGetColumnsBatchLazyQuery,
    useGetStorageUnitsQuery
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
import clone from "lodash/clone";
import cloneDeep from "lodash/cloneDeep";
import filter from "lodash/filter";
import {FC, useCallback, useEffect, useMemo, useState} from "react";
import {useNavigate, useSearchParams} from "react-router-dom";
import {Handle, Position} from "reactflow";
import {Card, ExpandableCard} from "../../components/card";
import {IGraphCardProps} from "../../components/graph/graph";
import {Loading, LoadingPage} from "../../components/loading";
import {InternalPage} from "../../components/page";
import {InternalRoutes} from "../../config/routes";
import {trackFrontendEvent} from "../../config/posthog";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {databaseSupportsModifiers} from "../../utils/database-data-types";
import {databaseSupportsScratchpad} from "../../utils/database-features";
import {getDatabaseStorageUnitLabel, isNoSQL} from "../../utils/functions";
import {Tip} from '../../components/tip';
import {SettingsActions} from '../../store/settings';
import {useTranslation} from '../../hooks/use-translation';

const StorageUnitCard: FC<{ unit: StorageUnit, columns?: any[] }> = ({ unit, columns }) => {
    const [expanded, setExpanded] = useState(false);
    const navigate = useNavigate();
    const { t } = useTranslation('pages/storage-unit');
    const current = useAppSelector(state => state.auth.current);

    const handleNavigateToDatabase = useCallback(() => {
        navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
            state: {
                unit,
            },
        })
    }, [navigate, unit]);

    const handleExpand = useCallback(() => {
        setExpanded(s => !s);
    }, []);

    const [introAttributes, expandedAttributes] = useMemo(() => {
        return [ unit.Attributes.slice(0,4), unit.Attributes.slice(4) ];
    }, [unit.Attributes]);

    return (<ExpandableCard key={unit.Name} isExpanded={expanded} setExpanded={setExpanded} icon={<TableCellsIcon className="w-4 h-4" />} className={cn({
        "shadow-2xl exploring-storage-unit": expanded,
    })} data-testid="storage-unit-card">
        <div className="flex flex-col grow mt-2 cursor-pointer" data-testid="storage-unit-card">
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
                        <p key={attribute.Key} className="text-xs">{attribute.Key}: {attribute.Value?.toLowerCase()}</p>
                    ))
                }
            </div>
            <div className="flex flex-row justify-end gap-xs" onClick={(e) => e.stopPropagation()}>
                <Button onClick={handleExpand} data-testid="explore-button" variant="secondary">
                    <MagnifyingGlassIcon className="w-4 h-4" /> {t('describe')}
                </Button>
                <Button onClick={handleNavigateToDatabase} data-testid="data-button" variant="secondary">
                    <CircleStackIcon className="w-4 h-4" /> {t('data')}
                </Button>
            </div>
        </div>
        <div className="flex flex-col grow gap-lg justify-between h-full overflow-y-auto">
            <SheetTitle className="flex items-center gap-2 mb-4">
                <TableCellsIcon className="w-5 h-5" />
                {unit.Name}
            </SheetTitle>
            {(current?.Type === DatabaseType.MongoDb || current?.Type === DatabaseType.ElasticSearch) && (
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
                            introAttributes.map(attribute => (
                                <StackListItem key={attribute.Key} item={attribute.Key}>
                                    {attribute.Value?.toLowerCase()}
                                </StackListItem>
                            ))
                        }
                        {
                            expandedAttributes.map(attribute => (
                                <StackListItem key={attribute.Key} item={attribute.Key}>
                                    {attribute.Value?.toLowerCase()}
                                </StackListItem>
                            ))
                        }
                    </StackList>
                    {columns && columns.length > 0 && (
                        <div className="mt-2">
                            <h3 className="text-xs font-semibold uppercase text-muted-foreground">{t('columnsTitle')}</h3>
                            <StackList>
                                {columns.map(col => {
                                    const isForeignKey = col.IsForeignKey;
                                    return (
                                        <StackListItem key={col.Name} item={isForeignKey ?
                                            <Badge className="text-lg" data-testid="foreign-key-attribute">{col.Name}</Badge> : col.Name}>
                                            {col.Type?.toLowerCase()}
                                        </StackListItem>
                                    );
                                })}
                            </StackList>
                        </div>
                    )}
                </div>
            </div>
            <div className="flex items-end grow">
                <Button onClick={handleNavigateToDatabase} data-testid="data-button" variant="secondary" className="w-full">
                    <CircleStackIcon className="w-4 h-4" /> {t('data')}
                </Button>
            </div>
        </div>
    </ExpandableCard>);
}

export const StorageUnitPage: FC = () => {
    const navigate = useNavigate();
    const [searchParams,] = useSearchParams();
    const [create, setCreate] = useState(searchParams.get("create") === "true");
    const [storageUnitName, setStorageUnitName] = useState("");
    const [fields, setFields] = useState<RecordInput[]>([ {Key: "", Value: "", Extra: [] }]);
    const [error, setError] = useState<string>();
    let schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const view = useAppSelector(state => state.settings.storageUnitView);
    const [addStorageUnit,] = useAddStorageUnitMutation();
    const [expandedUnit, setExpandedUnit] = useState<string | null>(null);
    const dispatch = useAppDispatch();
    const [tableColumns, setTableColumns] = useState<Record<string, any[]>>({});
    const [fetchColumnsBatch] = useGetColumnsBatchLazyQuery();
    const { t } = useTranslation('pages/storage-unit');

    useEffect(() => {
        void trackFrontendEvent('ui.storage_unit_viewed', {
            database_type: current?.Type ?? 'unknown',
            view_mode: view,
        });
    }, [current?.Type, trackFrontendEvent, view]);

    // TODO: ClickHouse/MongoDB use database name as schema parameter since they lack traditional schemas
    if (current?.Type === DatabaseType.ClickHouse || current?.Type === DatabaseType.MongoDb) {
        schema = current.Database
    }

    const { loading, data, refetch } = useGetStorageUnitsQuery({
        variables: {
            schema,
        },
    });

    // Refetch storage units when profile changes (current?.Id changes means different server/credentials)
    const currentProfileId = current?.Id;
    useEffect(() => {
        if (currentProfileId) {
            refetch();
        }
    }, [currentProfileId, refetch]);

    const [filterValue, setFilterValue] = useState("");

    const routes = useMemo(() => {
        const name = getDatabaseStorageUnitLabel(current?.Type);
        return [
            {
                ...InternalRoutes.Dashboard.StorageUnit,
                name,
            },
        ];
    }, [current]);

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
        if (!isNoSQL(current?.Type as DatabaseType) && fields.some(field => field.Key.length === 0 || field.Value.length === 0)) {
            return setError(t('fieldsCannotBeEmpty'));
        }
        setError(undefined);
        addStorageUnit({
            variables: {
                schema,
                storageUnit: storageUnitName,
                fields,
            },
            onCompleted() {
                const message = t('createSuccessMessage').replace('{storageUnit}', `${getDatabaseStorageUnitLabel(current?.Type, true)} ${storageUnitName}`);
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
    }, [addStorageUnit, current?.Type, fields, refetch, schema, storageUnitName, t, trackFrontendEvent]);

    const handleAddField = useCallback(() => {
        setFields(f => [...f, { Key: "", Value: "", Extra: [] }]);
        void trackFrontendEvent('ui.storage_unit_field_added', {
            database_type: current?.Type ?? 'unknown',
        });
    }, [current?.Type, trackFrontendEvent]);

    const handleFieldValueChange = useCallback((type: string, index: number, value: string | boolean) => {
        setFields(f => {
            const newF = cloneDeep(f);
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
            const newF = clone(f);
            newF.splice(index, 1);
            return newF;
        })
    }, [fields.length]);

    useEffect(() => {
        refetch();
    }, [current, refetch]);

    useEffect(() => {
        if (!data?.StorageUnit || data.StorageUnit.length === 0) return;

        const storageUnitNames = data.StorageUnit.map(unit => unit.Name);
        fetchColumnsBatch({
            variables: {
                schema,
                storageUnits: storageUnitNames,
            },
        }).then(result => {
            if (result.data?.ColumnsBatch) {
                const columnsMap: Record<string, any[]> = {};
                for (const item of result.data.ColumnsBatch) {
                    columnsMap[item.StorageUnit] = item.Columns;
                }
                setTableColumns(columnsMap);
            }
        }).catch(error => {
            console.error('Failed to fetch columns batch:', error);
            toast.error('Failed to load column information');
        });
    }, [data?.StorageUnit, fetchColumnsBatch, schema]);

    const filterStorageUnits = useMemo(() => {
        const lowerCaseFilterValue = filterValue.toLowerCase();
        return filter(data?.StorageUnit ?? [], unit => unit.Name.toLowerCase().includes(lowerCaseFilterValue))
            .sort((a, b) => a.Name.localeCompare(b.Name));
    }, [data?.StorageUnit, filterValue]);

    const showModifiers = useMemo(() => {
        if (!current?.Type) {
            return false;
        }
        return databaseSupportsModifiers(current.Type);
    }, [current?.Type]);

    const sharedAttributeKeys = useMemo(() => {
        if (!data?.StorageUnit || data.StorageUnit.length === 0) {
            return [];
        }

        // Get attributes that exist in ALL storage units (intersection of all attributes)
        // Preserve the order from the first unit (backend returns metadata like Type first)
        const firstUnitKeys = data.StorageUnit[0].Attributes.map(attr => attr.Key);

        return firstUnitKeys.filter(key =>
            data.StorageUnit.every(unit =>
                unit.Attributes.some(attr => attr.Key === key)
            )
        );
    }, [data?.StorageUnit]);

    if (loading) {
        return <InternalPage routes={routes}>
            <LoadingPage />
        </InternalPage>
    }

    return <InternalPage routes={routes}>
        <div className="flex w-full h-fit my-2 gap-lg justify-between">
            <div className="flex justify-between items-center">
                <SearchInput value={filterValue} onChange={e => setFilterValue(e.target.value)} placeholder={t('searchPlaceholder')} />
            </div>
            <div className="flex items-center gap-2">
                {
                    databaseSupportsScratchpad(current?.Type) &&
                    <Button onClick={() => navigate(InternalRoutes.RawExecute.path)} data-testid="scratchpad-button" variant="secondary">
                        <CommandLineIcon className="w-4 h-4" /> {t('scratchpad')}
                    </Button>
                }
                <Tabs value={view} onValueChange={value => dispatch(SettingsActions.setStorageUnitView(value as 'list' | 'card'))}>
                    <TabsList>
                        <TabsTrigger value="card" data-testid="icon-button"><Squares2X2Icon className="w-4 h-4" /></TabsTrigger>
                        <TabsTrigger value="list" data-testid="icon-button"><Bars3Icon className="w-4 h-4" /></TabsTrigger>
                    </TabsList>
                </Tabs>
            </div>
        </div>
        <div className={cn("flex flex-wrap gap-4", {
            "hidden": view !== "card",
        })} data-testid="storage-unit-card-list">
            <ExpandableCard className={classNames("overflow-visible min-w-[200px] max-w-[700px] h-full", {
                "hidden": current?.Type === DatabaseType.Redis,
            })} icon={<PlusCircleIcon className="w-4 h-4" />} isExpanded={create} setExpanded={setCreate} tag={<Badge variant="destructive">{error}</Badge>}>
                <div className="flex flex-col grow h-full justify-between mt-2 gap-2" data-testid="create-storage-unit-card">
                    <h1 className="text-lg"><span className="prefix-create-storage-unit">{t('createPrefix')}</span> {getDatabaseStorageUnitLabel(current?.Type, true)}</h1>
                    <Button className="self-end" onClick={handleCreate} variant="secondary">
                        <PlusCircleIcon  className='w-4 h-4' /> {t('create')}
                    </Button>
                </div>
                <div className="flex grow flex-col my-2 gap-4">
                    <div className="flex flex-col gap-4">
                        <SheetTitle className="flex items-center gap-2">
                            <PlusCircleIcon className="w-5 h-5" />
                            {t('createTitle')} {getDatabaseStorageUnitLabel(current?.Type, true)}
                        </SheetTitle>
                        <div className="flex flex-col gap-2">
                            <Label>{t('nameLabel')}</Label>
                            <Input value={storageUnitName} onChange={e => setStorageUnitName(e.target.value)} placeholder={t('namePlaceholder')} />
                        </div>
                        <div className={classNames("flex flex-col gap-sm overflow-y-auto max-h-[75vh]", {
                            "hidden": isNoSQL(current?.Type as DatabaseType),
                        })}>
                            <div className="flex flex-col gap-4">
                                {
                                    fields.map((field, index) => (
                                        <div className="flex flex-col gap-lg relative" key={`field-${index}`} data-testid="create-field-card">
                                            <Label>{t('fieldNameLabel')}</Label>
                                            <Input value={field.Key} onChange={e => handleFieldValueChange("Key", index, e.target.value)} placeholder={t('fieldNamePlaceholder')}/>
                                            <Label>{t('fieldTypeLabel')}</Label>
                                            <TypeSelector
                                                databaseType={current?.Type}
                                                value={field.Value}
                                                onChange={value => handleFieldValueChange("Value", index, value)}
                                                placeholder={t('fieldTypePlaceholder')}
                                                searchPlaceholder={t('searchTypePlaceholder')}
                                                buttonProps={{
                                                    "data-testid": `field-type-${index}`,
                                                }}
                                            />

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
                                                    <XCircleIcon className="w-4 h-4"/> <span>{t('removeField')}</span>
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
                        <CheckCircleIcon className="w-4 h-4" /> {t('createButton')}
                    </Button>
                </div>
            </ExpandableCard>
            {
                data != null && data.StorageUnit.length > 0 && filterStorageUnits.map(unit => (
                    <StorageUnitCard key={unit.Name} unit={unit} columns={tableColumns[unit.Name]} />
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
                        <TableHead>{t('actionsColumn')}</TableHead>
                    </TableHeadRow>
                </TableHeader>
                <VirtualizedTableBody
                    rowCount={filterStorageUnits.length}
                    rowHeight={40}>
                    {(rowIndex: number) => {
                        const unit = filterStorageUnits[rowIndex];
                        const attrMap = Object.fromEntries(unit.Attributes.map(attr => [attr.Key, attr.Value]));
                        return (
                            <TableRow key={unit.Name} className="group">
                                <TableCell>{unit.Name}</TableCell>
                                {sharedAttributeKeys.map(key => (
                                    <TableCell key={key}>{attrMap[key] ?? ""}</TableCell>
                                ))}
                                <TableCell className="relative">
                                    <div className="flex gap-xs opacity-0 group-hover:opacity-100 transition-opacity">
                                        <Button
                                            onClick={() => {
                                                navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
                                                    state: { unit },
                                                });
                                            }}
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

                        const columns = tableColumns[unit.Name];
                        const [introAttributes, expandedAttributes] = [unit.Attributes.slice(0,4), unit.Attributes.slice(4)];

                        return (
                            <Card className="p-6">
                                <div className="flex flex-col gap-4">
                                    <h2 className="text-2xl font-bold">{unit.Name}</h2>
                                    <StackList>
                                        {/* Metadata attributes */}
                                        {introAttributes.map(attribute => (
                                            <StackListItem key={attribute.Key} item={attribute.Key}>
                                                {attribute.Value?.toLowerCase()}
                                            </StackListItem>
                                        ))}
                                        {expandedAttributes.map(attribute => (
                                            <StackListItem key={attribute.Key} item={attribute.Key}>
                                                {attribute.Value?.toLowerCase()}
                                            </StackListItem>
                                        ))}
                                    </StackList>
                                    {columns && columns.length > 0 && (
                                        <div>
                                            <h3 className="text-xs font-semibold uppercase text-muted-foreground mb-2">{t('columnsTitle')}</h3>
                                            <StackList>
                                                {columns.map((col: any) => {
                                                    const isForeignKey = col.IsForeignKey;
                                                    return (
                                                        <StackListItem key={col.Name} item={isForeignKey ?
                                                            <Badge className="text-lg" data-testid="foreign-key-attribute">{col.Name}</Badge> : col.Name}>
                                                            {col.Type?.toLowerCase()}
                                                        </StackListItem>
                                                    );
                                                })}
                                            </StackList>
                                        </div>
                                    )}
                                    <div className="flex gap-sm mt-4">
                                        <Button
                                            onClick={() => {
                                                navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
                                                    state: { unit },
                                                });
                                            }}
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

export const StorageUnitGraphCard: FC<IGraphCardProps<StorageUnit & { columns?: any[] }>> = ({ data }) => {
    const navigate = useNavigate();
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
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
                                        <StackListItem key={`meta-${name}-${index}`} rowClassName="items-start" item={name}>
                                            {value}
                                        </StackListItem>
                                    );
                                })
                            }
                            {/* Show columns with FK/PK handles */}
                            {
                                columnItems.map((col: any, index: number) => {
                                    const name = col.Name;
                                    const value = col.Type?.toLowerCase();
                                    const isFKColumn = col.IsForeignKey || false;
                                    const isPKColumn = col.IsPrimary || false;

                                    return (
                                        <div key={`col-${name}-${index}`} className="relative">
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
                                                           item={isFKColumn ? <Badge
                                                               className="text-lg">{name}</Badge> : name}>
                                                {value}
                                            </StackListItem>
                                        </div>
                                    );
                                })
                            }
                        </StackList>
                    </div>
                    <Button onClick={handleNavigateTo} data-testid="data-button">
                        <CircleStackIcon className="w-4 h-4" /> {t('data')}
                    </Button>
                </div>
            </Card>
        </>
    );
}
