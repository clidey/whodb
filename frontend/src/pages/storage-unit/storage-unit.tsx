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

import { Badge, Button, Checkbox, cn, Input, Label, SearchInput, SearchSelect, Separator, StackList, StackListItem, Table, TableCell, TableHead, Tabs, TabsContent, TabsList, TabsTrigger, toast, TableRow, VirtualizedTableBody, TableHeader, TableHeadRow, SheetTitle } from '@clidey/ux';
import { DatabaseType, RecordInput, StorageUnit, useAddStorageUnitMutation, useGetStorageUnitsQuery } from '@graphql';
import { ArrowPathRoundedSquareIcon, CheckCircleIcon, CircleStackIcon, CommandLineIcon, ListBulletIcon, MagnifyingGlassIcon, PlusCircleIcon, TableCellsIcon, XCircleIcon, XMarkIcon } from '../../components/heroicons';
import classNames from "classnames";
import clone from "lodash/clone";
import cloneDeep from "lodash/cloneDeep";
import filter from "lodash/filter";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Handle, Node, Position, useReactFlow } from "reactflow";
import { Card, ExpandableCard } from "../../components/card";
import { IGraphCardProps } from "../../components/graph/graph";
import { Loading, LoadingPage } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { databaseSupportsModifiers, getDatabaseDataTypes } from "../../utils/database-data-types";
import { databaseSupportsScratchpad } from "../../utils/database-features";
import { getDatabaseStorageUnitLabel, isNoSQL } from "../../utils/functions";
import { Tip } from '../../components/tip';
import { SettingsActions } from '../../store/settings';

const StorageUnitCard: FC<{ unit: StorageUnit, allTableNames: Set<string> }> = ({ unit, allTableNames }) => {
    const [expanded, setExpanded] = useState(false);
    const navigate = useNavigate();

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

    const isValidForeignKey = useCallback((key: string) => {
        // Check for both singular and plural table names
        if (key.endsWith("_id")) {
            const base = key.slice(0, -3);
            return allTableNames.has(base) || allTableNames.has(base + "s");
        }
        return false;
    }, [allTableNames]);

    return (<ExpandableCard key={unit.Name} isExpanded={expanded} setExpanded={setExpanded} icon={<TableCellsIcon className="w-4 h-4" />} className={cn({
        "shadow-2xl exploring-storage-unit": expanded,
    })} data-testid="storage-unit-card">
        <div className="flex flex-col grow mt-2" data-testid="storage-unit-card">
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
                        <p key={attribute.Key} className="text-xs">{attribute.Key}: {attribute.Value}</p>
                    ))
                }
            </div>
            <div className="flex flex-row justify-end gap-xs">
                <Button onClick={handleExpand} data-testid="explore-button" variant="secondary">
                    <MagnifyingGlassIcon className="w-4 h-4" /> Describe
                </Button>
                <Button onClick={handleNavigateToDatabase} data-testid="data-button" variant="secondary">
                    <CircleStackIcon className="w-4 h-4" /> Data
                </Button>
            </div>
        </div>
        <div className="flex flex-col grow gap-lg justify-between h-full overflow-y-auto">
            <SheetTitle className="flex items-center gap-2 mb-4">
                <TableCellsIcon className="w-5 h-5" />
                {unit.Name}
            </SheetTitle>
            <div className="w-full" data-testid="explore-fields">
                <div className="flex flex-col gap-xs2">
                    <StackList>
                        {
                            introAttributes.map(attribute => (
                                <StackListItem key={attribute.Key} item={attribute.Key}>
                                    {attribute.Value}
                                </StackListItem>
                            ))
                        }
                        {
                            expandedAttributes.map(attribute => (
                                <StackListItem key={attribute.Key} item={isValidForeignKey(attribute.Key) ?
                                    <Badge className="text-lg"
                                           data-testid="foreign-key-attribute">{attribute.Key}</Badge> : attribute.Key}>
                                    {attribute.Value}
                                </StackListItem>
                            ))
                        }
                    </StackList>
                </div>
            </div>
            <div className="flex items-end grow">
                <Button onClick={handleNavigateToDatabase} data-testid="data-button" variant="secondary" className="w-full">
                    <CircleStackIcon className="w-4 h-4" /> Data
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

    // For databases that don't have schemas (MongoDB, ClickHouse), pass the database name as the schema parameter
    // todo: is there a different way to do this? clickhouse doesn't have schemas as a table is considered a schema. people mainly switch between DB
    if (current?.Type === DatabaseType.ClickHouse || current?.Type === DatabaseType.MongoDb) {
        schema = current.Database
    }

    const { loading, data, refetch } = useGetStorageUnitsQuery({
        variables: {
            schema,
        },
    });
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
        setCreate(!create);
    }, [create]);

    const handleSubmit = useCallback(() => {
        if (storageUnitName.length === 0) {
            return setError("Name is required");
        }
        if (!isNoSQL(current?.Type as DatabaseType) && fields.some(field => field.Key.length === 0 || field.Value.length === 0)) {
            return setError("Fields cannot be empty");
        }
        setError(undefined);
        addStorageUnit({
            variables: {
                schema,
                storageUnit: storageUnitName,
                fields,
            },
            onCompleted() {
                toast.success(`${getDatabaseStorageUnitLabel(current?.Type, true)} ${storageUnitName} created successfully!`);
                setStorageUnitName("");
                setFields([]);
                refetch();
                setCreate(false);
            },
            onError(e) {
                toast.error(e.message);
            },
        });
    }, [addStorageUnit, current?.Type, fields, refetch, schema, storageUnitName]);

    const handleAddField = useCallback(() => {
        setFields(f => [...f, { Key: "", Value: "", Extra: [] }]);
    }, []);

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

    const storageUnitTypesDropdownItems = useMemo(() => {
        if (current?.Type == null || isNoSQL(current.Type)) {
            return [];
        }
        
        const dataTypes = getDatabaseDataTypes(current.Type);
        return dataTypes.map(item => ({
            id: item,
            label: item,
        }));
    }, [current?.Type]);
    
    useEffect(() => {
        refetch();
    }, [current, refetch]);

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

    const allTableNames = useMemo(() => {
        return new Set(Array.isArray(data?.StorageUnit) ? data.StorageUnit.map(unit => unit.Name) : []);
    }, [data?.StorageUnit]);

    const sharedAttributeKeys = useMemo(() => {
        if (!data?.StorageUnit || data.StorageUnit.length === 0) {
            return [];
        }
        
        // Get attributes that exist in ALL storage units (intersection of all attributes)
        const firstUnitAttributeKeys = new Set(data.StorageUnit[0].Attributes.map(attr => attr.Key));
        
        return Array.from(firstUnitAttributeKeys).filter(key => 
            data.StorageUnit.every(unit => 
                unit.Attributes.some(attr => attr.Key === key)
            )
        ).sort();
    }, [data?.StorageUnit]);

    const isValidForeignKey = useCallback((key: string) => {
        // Check for both singular and plural table names, case-insensitive
        if (key.toLowerCase().endsWith("_id")) {
            const base = key.slice(0, -3).toLowerCase();
            const allNamesLower = new Set(Array.from(allTableNames, name => name.toLowerCase()));
            return allNamesLower.has(base) || allNamesLower.has(base + "s");
        }
        return false;
    }, [allTableNames]);

    if (loading) {
        return <InternalPage routes={routes}>
            <LoadingPage />
        </InternalPage>
    }

    return <InternalPage routes={routes}>
        <div className="flex w-full h-fit my-2 gap-lg justify-between">
            <div className="flex justify-between items-center">
                <SearchInput value={filterValue} onChange={e => setFilterValue(e.target.value)} placeholder="Enter filter value..." />
            </div>
            <div className="flex items-center gap-2">
                {
                    databaseSupportsScratchpad(current?.Type) &&
                    <Button onClick={() => navigate(InternalRoutes.RawExecute.path)} data-testid="scratchpad-button" variant="secondary">
                        <CommandLineIcon className="w-4 h-4" /> Scratchpad
                    </Button>
                }
                <Tabs value={view} onValueChange={value => dispatch(SettingsActions.setStorageUnitView(value as 'list' | 'card'))}>
                    <TabsList>
                        <TabsTrigger value="card" data-testid="icon-button"><TableCellsIcon className="w-4 h-4" /></TabsTrigger>
                        <TabsTrigger value="list" data-testid="icon-button"><ListBulletIcon className="w-4 h-4" /></TabsTrigger>
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
                    <h1 className="text-lg"><span className="prefix-create-storage-unit">Create a</span> {getDatabaseStorageUnitLabel(current?.Type, true)}</h1>
                    <Button className="self-end" onClick={handleCreate} variant="secondary">
                        <PlusCircleIcon  className='w-4 h-4' /> Create
                    </Button>
                </div>
                <div className="flex grow flex-col my-2 gap-4">
                    <div className="flex flex-col gap-4">
                        <SheetTitle className="flex items-center gap-2">
                            <PlusCircleIcon className="w-5 h-5" />
                            Create a {getDatabaseStorageUnitLabel(current?.Type, true)}
                        </SheetTitle>
                        <div className="flex flex-col gap-2">
                            <Label>Name</Label>
                            <Input value={storageUnitName} onChange={e => setStorageUnitName(e.target.value)} placeholder="Enter name..." />
                        </div>
                        <div className={classNames("flex flex-col gap-sm overflow-y-auto max-h-[75vh]", {
                            "hidden": isNoSQL(current?.Type as DatabaseType),
                        })}>
                            <div className="flex flex-col gap-4">
                                {
                                    fields.map((field, index) => (
                                        <div className="flex flex-col gap-lg relative" key={`field-${index}`} data-testid="create-field-card">
                                            <Label>Field Name</Label>
                                            <Input value={field.Key} onChange={e => handleFieldValueChange("Key", index, e.target.value)} placeholder="Enter field name"/>
                                            <Label>Field Type</Label>
                                            <SearchSelect
                                                options={storageUnitTypesDropdownItems.map(item => ({
                                                    value: item.id,
                                                    label: item.label,
                                                }))}
                                                value={field.Value}
                                                onChange={value => handleFieldValueChange("Value", index, value)}
                                                placeholder="Select type"
                                                searchPlaceholder="Search type..."
                                                buttonProps={{
                                                    "data-testid": `field-type-${index}`,
                                                }}
                                            />

                                            {showModifiers && (
                                                <>
                                                    <Label>Modifiers</Label>
                                                    <div className="flex items-center w-1/3 justify-start gap-2">
                                                        <Checkbox checked={field.Extra?.find(extra => extra.Key === "Primary") != null} onCheckedChange={() => handleFieldValueChange("Primary", index, !field.Extra?.find(extra => extra.Key === "Primary") != null)}/>
                                                        <Label>Primary</Label>
                                                        <Checkbox checked={field.Extra?.find(extra => extra.Key === "Nullable") != null} onCheckedChange={() => handleFieldValueChange("Nullable", index, !field.Extra?.find(extra => extra.Key === "Nullable") != null)}/>
                                                        <Label>Nullable</Label>
                                                    </div>
                                                </>
                                            )}
                                            {
                                                fields.length > 1 &&
                                                <Button variant="destructive" onClick={() => handleRemove(index)} data-testid="remove-field-button" className="w-full mt-1">
                                                    <XCircleIcon className="w-4 h-4"/> <span>Remove</span>
                                                </Button>
                                            }
                                            {index !== fields.length - 1 && <Separator className="mt-2" />}
                                        </div>
                                    ))
                                } 
                            </div>
                            <Button className="self-end" onClick={handleAddField} data-testid="add-field-button" variant="secondary">
                                <PlusCircleIcon  className='w-4 h-4' /> Add field
                            </Button>
                        </div>
                    </div>
                    <div className="flex grow" />
                    <Button onClick={handleSubmit} data-testid="submit-button" className="w-full">
                        <CheckCircleIcon className="w-4 h-4" /> Submit
                    </Button>
                </div>
            </ExpandableCard>
            {
                data != null && data.StorageUnit.length > 0 && filterStorageUnits.map(unit => (
                    <StorageUnitCard key={unit.Name} unit={unit} allTableNames={allTableNames} />
                ))
            }
        </div>
        <div className={cn("flex flex-wrap gap-lg w-full h-[80vh]", {
            "hidden": view !== "list",
        })}>
            <Table>
                <TableHeader>
                    <TableHeadRow>
                        <TableHead>Name</TableHead>
                        {/** Dynamically render shared attribute keys as columns */}
                        {sharedAttributeKeys.map(key => (
                            <TableHead key={key}>{key}</TableHead>
                        ))}
                        <TableHead>Actions</TableHead>
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
                                        >
                                            <CircleStackIcon className="w-4 h-4" /> Data
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
                        
                        const [introAttributes, expandedAttributes] = [unit.Attributes.slice(0,4), unit.Attributes.slice(4)];
                        
                        return (
                            <Card className="p-6">
                                <div className="flex flex-col gap-4">
                                    <h2 className="text-2xl font-bold">{unit.Name}</h2>
                                    <StackList>
                                        {introAttributes.map(attribute => (
                                            <StackListItem key={attribute.Key} item={attribute.Key}>
                                                {attribute.Value}
                                            </StackListItem>
                                        ))}
                                        {expandedAttributes.map(attribute => (
                                            <StackListItem key={attribute.Key} item={isValidForeignKey(attribute.Key) ? <Badge className="text-lg" data-testid="foreign-key-attribute">{attribute.Key}</Badge> : attribute.Key}>
                                                {attribute.Value}
                                            </StackListItem>
                                        ))}
                                    </StackList>
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
                                            <CircleStackIcon className="w-4 h-4" /> Data
                                        </Button>
                                        <Button 
                                            onClick={() => setExpandedUnit(null)} 
                                            variant="outline"
                                        >
                                            <XMarkIcon className="w-4 h-4" /> Close
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

export const StorageUnitGraphCard: FC<IGraphCardProps<StorageUnit>> = ({ data }) => {
    const { getNodes } = useReactFlow();
    const navigate = useNavigate();

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

    const isValidForeignKey = useCallback((key: string) => {
        // Use node ids as table names
        if (key.endsWith("_id")) {
            const nodes = getNodes();
            const base = key.slice(0, -3);
            const nodeIds = new Set(nodes.map((node: Node) => node.id));
            return nodeIds.has(base) || nodeIds.has(base + "s");
        }
        return false;
    }, [getNodes]);

    if (data == null) {
        return (<Card icon={<ArrowPathRoundedSquareIcon className="w-4 h-4" />}>
            <Loading hideText={true} />
        </Card>)
    }

    return (
        <>
            <Handle className="dark:border-white/5" type="target" position={Position.Left} />
            <Card icon={<CircleStackIcon className="w-4 h-4" />} className="h-fit backdrop-blur-[2px] w-[400px] px-2 py-6">
                <div className="flex flex-col grow mt-2 gap-4">
                    <div className="flex flex-col grow">
                        <h2 className="text-3xl font-semibold mb-2 break-words">{data.Name}</h2>
                        <StackList>
                            {
                                data.Attributes.map(attribute => (
                                    <StackListItem rowClassName="items-start" key={attribute.Key}
                                                   item={isValidForeignKey(attribute.Key) ? <Badge
                                                       className="text-lg">{attribute.Key}</Badge> : attribute.Key}>
                                        {attribute.Value}
                                    </StackListItem>
                                ))
                            }
                        </StackList>
                    </div>
                    <Button onClick={handleNavigateTo} data-testid="data-button">
                        <CircleStackIcon className="w-4 h-4" /> Data
                    </Button>
                </div>
            </Card>
            <Handle className="dark:border-white/5" type="source" position={Position.Right} />
        </>
    );
}
