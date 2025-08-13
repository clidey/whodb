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

import { Badge, Button, Checkbox, cn, Input, Label, SearchInput, SearchSelect, Separator, StackList, StackListItem, toast } from '@clidey/ux';
import { DatabaseType, RecordInput, StorageUnit, useAddStorageUnitMutation, useGetStorageUnitsQuery } from '@graphql';
import classNames from "classnames";
import { clone, cloneDeep, filter } from "lodash";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Handle, Node, Position, useReactFlow } from "reactflow";
import { Card, ExpandableCard } from "../../components/card";
import { IGraphCardProps } from "../../components/graph/graph";
import { Icons } from "../../components/icons";
import { Loading, LoadingPage } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { useAppSelector } from "../../store/hooks";
import { databaseSupportsModifiers, getDatabaseDataTypes } from "../../utils/database-data-types";
import { databaseSupportsScratchpad } from "../../utils/database-features";
import { getDatabaseStorageUnitLabel, isNoSQL } from "../../utils/functions";

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

    return (<ExpandableCard key={unit.Name} isExpanded={expanded} setExpanded={setExpanded} icon={Icons.Tables} className={cn({
        "shadow-2xl": expanded,
    })}>
        <div className="flex flex-col grow mt-2">
            <div className="flex flex-col grow mb-2">
                <h1 className="text-sm font-semibold mb-2 break-words" data-testid="storage-unit-name">{unit.Name}</h1>
                {
                    introAttributes.slice(0,2).map(attribute => (
                        <p key={attribute.Key} className="text-xs">{attribute.Key}: {attribute.Value}</p>
                    ))
                }
            </div>
            <div className="flex flex-row justify-end gap-1">
                <Button onClick={handleExpand} data-testid="explore-button" variant="secondary">
                    {Icons.DocumentMagnify} Explore
                </Button>
                <Button onClick={handleNavigateToDatabase} data-testid="data-button" variant="secondary">
                    {Icons.Database} Data
                </Button>
            </div>
        </div>
        <div className="flex flex-col grow gap-4 justify-between h-full">
            <div className="w-full" data-testid="explore-fields">
                <div className="flex flex-col gap-12">
                    <h1 className="text-2xl font-bold mb-4">{unit.Name}</h1>
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
                                <StackListItem key={attribute.Key} item={isValidForeignKey(attribute.Key) ? <Badge className="text-lg">{attribute.Key}</Badge> : attribute.Key}>
                                    {attribute.Value}
                                </StackListItem>
                            ))
                        }
                    </StackList>
                </div>
            </div>
            <div className="flex items-end grow">
                <Button icon={Icons.Database} onClick={handleNavigateToDatabase} data-testid="data-button" variant="secondary" className="w-full">
                    {Icons.Database} Data
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
    const [addStorageUnit,] = useAddStorageUnitMutation();

    // todo: is there a different way to do this? clickhouse doesn't have schemas as a table is considered a schema. people mainly switch between DB
    if (current?.Type === DatabaseType.ClickHouse) {
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

    if (loading) {
        return <InternalPage routes={routes}>
            <LoadingPage />
        </InternalPage>
    }

    return <InternalPage routes={routes}>
        <div className="flex w-full h-fit my-2 gap-4 justify-between">
            <div>
                <SearchInput value={filterValue} onChange={e => setFilterValue(e.target.value)} placeholder="Enter filter value..." />
            </div>
            <div>
                {
                    databaseSupportsScratchpad(current?.Type) &&
                    <Button onClick={() => navigate(InternalRoutes.RawExecute.path)} data-testid="scratchpad-button" variant="secondary">
                        {Icons.Console} Scratchpad
                    </Button>
                }
            </div>
        </div>
        <div className="flex flex-wrap gap-4">
            <ExpandableCard className={classNames("overflow-visible min-w-[200px] max-w-[700px] h-full", {
                "hidden": current?.Type === DatabaseType.Redis,
            })} icon={Icons.Add} isExpanded={create} setExpanded={setCreate} tag={<Badge variant="destructive">{error}</Badge>}>
                <div className="flex flex-col grow h-full justify-between">
                    <h2 className="text-lg">Create a {getDatabaseStorageUnitLabel(current?.Type, true)}</h2>
                    <Button className="self-end" onClick={handleCreate} variant="secondary">
                        {Icons.Add} Create
                    </Button>
                </div>
                <div className="flex grow flex-col my-2 gap-4">
                    <div className="flex flex-col gap-2">
                        <h1 className="text-2xl font-bold mb-4">Create a {getDatabaseStorageUnitLabel(current?.Type, true)}</h1>
                        <div className="flex flex-col gap-2">
                            <Label>Name</Label>
                            <Input value={storageUnitName} onChange={e => setStorageUnitName(e.target.value)} />
                        </div>
                        <div className={classNames("flex flex-col gap-2 overflow-y-auto max-h-[75vh]", {
                            "hidden": isNoSQL(current?.Type as DatabaseType),
                        })}>
                            <div className="flex flex-col gap-4">
                                {
                                    fields.map((field, index) => (
                                        <div className="flex flex-col gap-2" key={`field-${index}`}>
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
                                                    {Icons.Delete} Remove
                                                </Button>
                                            }
                                            {index !== fields.length - 1 && <Separator className="mt-2" />}
                                        </div>
                                    ))
                                } 
                            </div>
                            <Button className="self-end" onClick={handleAddField} data-testid="add-field-button" variant="secondary">
                                {Icons.Add} Add field
                            </Button>
                        </div>
                    </div>
                    <div className="flex grow" />
                    <Button icon={Icons.CheckCircle} onClick={handleSubmit} data-testid="submit-button" className="w-full">
                        {Icons.CheckCircle} Submit
                    </Button>
                </div>
            </ExpandableCard>
            {
                data != null && data.StorageUnit.length > 0 && filterStorageUnits.map(unit => (
                    <StorageUnitCard key={unit.Name} unit={unit} allTableNames={allTableNames} />
                ))
            }
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
        return (<Card icon={Icons.Fetch}>
            <Loading hideText={true} />
        </Card>)
    }

    return (
        <>
            <Handle className="dark:border-white/5" type="target" position={Position.Left} />
            <Card icon={Icons.Database} className="h-fit backdrop-blur-[2px] w-[400px] px-2 py-6">
                <div className="flex flex-col grow mt-2 gap-4">
                    <div className="flex flex-col grow">
                        <h2 className="text-3xl font-semibold mb-2 break-words">{data.Name}</h2>
                        <StackList>
                            {
                                data.Attributes.map(attribute => (
                                    <StackListItem key={attribute.Key} item={isValidForeignKey(attribute.Key) ? <Badge className="text-lg">{attribute.Key}</Badge> : attribute.Key}>
                                        {attribute.Value}
                                    </StackListItem>
                                ))
                            }
                        </StackList>
                    </div>
                    <Button onClick={handleNavigateTo} data-testid="data-button">
                        {Icons.Database} Data
                    </Button>
                </div>
            </Card>
            <Handle className="dark:border-white/5" type="source" position={Position.Right} />
        </>
    );
}
