import classNames from "classnames";
import { clone } from "lodash";
import { FC, useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Handle, Position } from "reactflow";
import { ActionButton, AnimatedButton } from "../../components/button";
import { Card, ExpandableCard } from "../../components/card";
import { EmptyMessage } from "../../components/common";
import { createDropdownItem, Dropdown } from "../../components/dropdown";
import { IGraphCardProps } from "../../components/graph/graph";
import { Icons } from "../../components/icons";
import { Input, InputWithlabel, Label } from "../../components/input";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, RecordInput, StorageUnit, useAddStorageUnitMutation, useGetStorageUnitsQuery } from "../../generated/graphql";
import { notify } from "../../store/function";
import { useAppSelector } from "../../store/hooks";
import { getDatabaseStorageUnitLabel, isNoSQL } from "../../utils/functions";

const StorageUnitCard: FC<{ unit: StorageUnit }> = ({ unit }) => {
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
        return [ unit.Attributes.slice(0,5), unit.Attributes.slice(5) ];
    }, [unit.Attributes]);

    return (<ExpandableCard key={unit.Name} isExpanded={expanded} icon={{
        bgClassName: "bg-teal-500",
        component: Icons.Tables,
    }}>
        <div className="flex flex-col grow mt-2">
            <div className="flex flex-col grow mb-2">
                <div className="text-sm font-semibold mb-2 break-words dark:text-neutral-100">{unit.Name}</div>
                {
                    introAttributes.slice(0,2).map(attribute => (
                        <div key={attribute.Key} className="text-xs dark:text-neutral-300">{attribute.Key}: {attribute.Value}</div>
                    ))
                }
            </div>
            <div className="flex flex-row justify-end gap-1">
                <AnimatedButton icon={Icons.DocumentMagnify} label="Explore" onClick={handleExpand} />
                <AnimatedButton icon={Icons.Database} label="Data" onClick={handleNavigateToDatabase} />
            </div>
        </div>
        <div className="flex flex-col grow mt-2 gap-4">
            <div className="flex flex-row grow">
                <div className="flex flex-col grow">
                    <div className="text-md font-semibold mb-2 dark:text-neutral-100">{unit.Name}</div>
                    {
                        introAttributes.map(attribute => (
                            <div key={attribute.Key} className="text-xs dark:text-neutral-300"><span className="font-semibold">{attribute.Key}:</span> {attribute.Value}</div>
                        ))
                    }
                </div>
                <div className="flex flex-col grow mt-6">
                    {
                        expandedAttributes.map(attribute => (
                            <div key={attribute.Key} className="text-xs dark:text-neutral-300"><span className="font-semibold">{attribute.Key}:</span> {attribute.Value}</div>
                        ))
                    }
                </div>
            </div>
            <div className="flex flex-row justify-end gap-1">
                <AnimatedButton icon={Icons.DocumentMagnify} label={expanded ? "Hide" : "Explore"} onClick={handleExpand} />
                <AnimatedButton icon={Icons.Database} label="Data" onClick={handleNavigateToDatabase} />
            </div>
        </div>
    </ExpandableCard>);
}

export const StorageUnitPage: FC = () => {
    const navigate = useNavigate();
    const [create, setCreate] = useState(false);
    const [storageUnitName, setStorageUnitName] = useState("");
    const [fields, setFields] = useState<RecordInput[]>([ {Key: "", Value: "" }]);
    const [error, setError] = useState<string>();
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const [addStorageUnit,] = useAddStorageUnitMutation();
    const { loading, data, refetch } = useGetStorageUnitsQuery({
        variables: {
            type: current?.Type as DatabaseType,
            schema,
        },
    });

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
                type: current?.Type as DatabaseType,
                schema,
                storageUnit: storageUnitName,
                fields,
            },
            onCompleted() {
                notify(`${getDatabaseStorageUnitLabel(current?.Type, true)} ${storageUnitName} created successfully!`, "success");
                setStorageUnitName("");
                setFields([]);
                refetch();
                setCreate(false);
            },
            onError(e) {
                notify(e.message, "error");
            },
        });
    }, [addStorageUnit, current?.Type, fields, refetch, schema, storageUnitName]);

    const handleAddField = useCallback(() => {
        setFields(f => [...f, { Key: "", Value: "" }]);
    }, []);

    const handleFieldValueChange = useCallback((type: "Key" | "Value", index: number, value: string) => {
        setFields(f => {
            const newF = clone(f);
            newF[index][type] = value;
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
        let items: string[] = [];
    
        switch(current.Type) {
            case DatabaseType.MariaDb:
                items = [
                    "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "FLOAT", "DOUBLE", "DECIMAL",
                    "DATE", "DATETIME", "TIMESTAMP", "TIME", "YEAR",
                    "CHAR", "VARCHAR", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", 
                    "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT",
                    "ENUM", "SET", "JSON", "BOOLEAN"
                ];
                break;
            case DatabaseType.MySql:
                items = [
                    "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "FLOAT", "DOUBLE", "DECIMAL",
                    "DATE", "DATETIME", "TIMESTAMP", "TIME", "YEAR",
                    "CHAR", "VARCHAR(255)", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", 
                    "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT",
                    "ENUM", "SET", "JSON", "BOOLEAN", "VARCHAR(100)", "VARCHAR(1000)"
                ];
                break;
            case DatabaseType.Postgres:
                items = [
                    "SMALLINT", "INTEGER", "BIGINT", "DECIMAL", "NUMERIC", "REAL", "DOUBLE PRECISION", "SMALLSERIAL", 
                    "SERIAL", "BIGSERIAL", "MONEY",
                    "CHAR", "VARCHAR", "TEXT", "BYTEA",
                    "TIMESTAMP", "TIMESTAMPTZ", "DATE", "TIME", "TIMETZ", "INTERVAL",
                    "BOOLEAN", "POINT", "LINE", "LSEG", "BOX", "PATH", "POLYGON", "CIRCLE",
                    "CIDR", "INET", "MACADDR", "UUID", "XML", "JSON", "JSONB", "ARRAY", "HSTORE"
                ];
                break;
            case DatabaseType.Sqlite3:
                items = [
                    "NULL", "INTEGER", "REAL", "TEXT", "BLOB",
                    "NUMERIC", "BOOLEAN", "DATE", "DATETIME"
                ];
                break;
            default:
                items = [];
                break;
        }
        return items.map(item => createDropdownItem(item));
    }, [current?.Type]);
    

    if (loading) {
        return <InternalPage routes={routes}>
            <Loading />
        </InternalPage>
    }

    return <InternalPage routes={routes}>
        <div className="flex w-full h-fit my-2 gap-2">
            <AnimatedButton icon={Icons.Console} label="Scratchpad" onClick={() => navigate(InternalRoutes.RawExecute.path)} type="lg" />
        </div>
        {
            data != null && (
                data.StorageUnit.length === 0
                ? <>
                    <EmptyMessage icon={Icons.SadSmile} label={`No ${getDatabaseStorageUnitLabel(current?.Type)} found`} />
                </>
                : <>
                    <ExpandableCard className={classNames("overflow-visible", {
                        "hidden": current?.Type === DatabaseType.Redis,
                    })} icon={{
                        bgClassName: "bg-teal-500",
                        component: Icons.Add,
                    }} isExpanded={create} tag={<div className="text-red-700 dark:text-red-400 text-xs">
                        {error}
                    </div>}>
                        <div className="flex grow flex-col justify-between mt-3 text-neutral-800 dark:text-neutral-100">
                            Create a {getDatabaseStorageUnitLabel(current?.Type, true)}
                            <AnimatedButton className="self-end" icon={Icons.Add} label="Create" onClick={handleCreate} />
                        </div>
                        <div className="flex grow flex-col justify-between my-2 gap-4">
                            <div className="flex flex-col gap-2">
                                <InputWithlabel label="Name" value={storageUnitName} setValue={setStorageUnitName} />
                                <div className={classNames("flex flex-col gap-2", {
                                    "hidden": isNoSQL(current?.Type as DatabaseType),
                                })}>
                                    <div className="flex gap-2 justify-between">
                                        <Label label="Field Name" />
                                        <Label label="Value" />
                                        <div className="w-14" />
                                    </div>
                                    {
                                        fields.map((field, index) => (
                                            <div className="flex gap-2" key={field.Key}>
                                                <Input inputProps={{ className: "w-1/2" }} value={field.Key} setValue={(value) => handleFieldValueChange("Key", index, value)} placeholder="Enter field name" />
                                                <Dropdown className="w-1/2" items={storageUnitTypesDropdownItems} value={createDropdownItem(field.Value)}
                                                    onChange={(item) => handleFieldValueChange("Value", index, item.id)} />
                                                <div className="flex items-end mb-2">
                                                    <ActionButton disabled={fields.length === 1} containerClassName="w-6 h-6" icon={Icons.Delete} className={classNames({
                                                        "stroke-red-500 dark:stroke-red-400": fields.length > 1,
                                                        "stroke-neutral-300 dark:stroke-neutral-600": fields.length === 1,
                                                    })} onClick={() => handleRemove(index)} />
                                                </div>
                                            </div>
                                        ))
                                    }
                                    <AnimatedButton className="self-end" icon={Icons.Add} label="Add field" onClick={handleAddField} />
                                </div>
                            </div>
                            <div className="flex items-center justify-between">
                                <AnimatedButton icon={Icons.Cancel} label="Cancel" onClick={handleCreate} />
                                <AnimatedButton labelClassName="text-green-600 dark:text-green-300"
                                    iconClassName="stroke-green-600 dark:stroke-green-300" icon={Icons.Add}
                                    label="Submit" onClick={handleSubmit} />
                            </div>
                        </div>
                    </ExpandableCard>
                    {data.StorageUnit.map(unit => (
                        <StorageUnitCard key={unit.Name} unit={unit} />
                    ))}
                </>
            )
        }
    </InternalPage>
}

export const StorageUnitGraphCard: FC<IGraphCardProps<StorageUnit>> = ({ data }) => {
    const navigate = useNavigate();

    const handleNavigateTo = useCallback(() => {
        navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
            state: {
                unit: data,
            }
        });
    }, [navigate, data]);

    if (data == null) {
        return (<Card icon={{
            component: Icons.Fetch,
            bgClassName: "bg-green-500",
        }}>
            <Loading hideText={true} />
        </Card>)
    }

    return (
        <>
            <Handle className="dark:border-white/5" type="target" position={Position.Left} />
            <Card icon={{
                bgClassName: "bg-teal-500",
                component: Icons.Database,
            }} className="h-fit backdrop-blur-[2px]">
                <div className="flex flex-col grow mt-2 gap-4">
                    <div className="flex flex-col grow">
                        <div className="text-md font-semibold mb-2 break-words dark:text-neutral-300">{data.Name}</div>
                        {
                            data.Attributes.slice(0, 5).map(attribute => (
                                <div key={attribute.Key} className="text-xs dark:text-neutral-300"><span className="font-semibold">{attribute.Key}:</span> {attribute.Value}</div>
                            ))
                        }
                    </div>
                    <div className="flex flex-row justify-end gap-1">
                        <AnimatedButton icon={Icons.RightArrowUp} label="Data" onClick={handleNavigateTo} />
                    </div>
                </div>
            </Card>
            <Handle className="dark:border-white/5" type="source" position={Position.Right} />
        </>
    );
}