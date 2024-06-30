import { useQuery } from "@apollo/client";
import { FC, useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Handle, Position } from "reactflow";
import { AnimatedButton } from "../../components/button";
import { Card, ExpandableCard } from "../../components/card";
import { IGraphCardProps } from "../../components/graph/graph";
import { Icons } from "../../components/icons";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, GetStorageUnitsDocument, GetStorageUnitsQuery, GetStorageUnitsQueryVariables, StorageUnit } from "../../generated/graphql";
import { useAppSelector } from "../../store/hooks";
import { EmptyMessage } from "../../components/common";

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
            <div className="flex flex-col grow">
                <div className="text-md font-semibold mb-2 break-words">{unit.Name}</div>
                {
                    introAttributes.slice(0,2).map(attribute => (
                        <div className="text-sm">{attribute.Key}: {attribute.Value}</div>
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
                    <div className="text-md font-semibold mb-2">{unit.Name}</div>
                    {
                        introAttributes.map(attribute => (
                            <div className="text-xs"><span className="font-semibold">{attribute.Key}:</span> {attribute.Value}</div>
                        ))
                    }
                </div>
                <div className="flex flex-col grow mt-6">
                    {
                        expandedAttributes.map(attribute => (
                            <div className="text-xs"><span className="font-semibold">{attribute.Key}:</span> {attribute.Value}</div>
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
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const { loading, data } = useQuery<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>(GetStorageUnitsDocument, {
        variables: {
            type: current?.Type as DatabaseType,
            schema,
        },
    });

    if (loading) {
        return <InternalPage>
            <Loading />
        </InternalPage>
    }

    return <InternalPage routes={[InternalRoutes.Dashboard.StorageUnit]}>
        <div className="flex w-full h-fit my-2 gap-2">
            <AnimatedButton icon={Icons.Console} label="Raw Query" onClick={() => navigate(InternalRoutes.RawExecute.path)} type="lg" />
        </div>
        {
            data != null && (
                data.StorageUnit.length === 0
                ? <EmptyMessage icon={Icons.SadSmile} label="No tables found. Try changing schema." />
                : data.StorageUnit.map(unit => (
                    <StorageUnitCard unit={unit} />
                ))
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
            <Handle type="target" position={Position.Left} />
            <Card icon={{
                bgClassName: "bg-teal-500",
                component: Icons.Database,
            }} className="h-fit">
                <div className="flex flex-col grow mt-2 gap-4">
                    <div className="flex flex-col grow">
                        <div className="text-md font-semibold mb-2 break-words">{data.Name}</div>
                        {
                            data.Attributes.slice(0, 5).map(attribute => (
                                <div className="text-xs"><span className="font-semibold">{attribute.Key}:</span> {attribute.Value}</div>
                            ))
                        }
                    </div>
                    <div className="flex flex-row justify-end gap-1">
                        <AnimatedButton icon={Icons.RightArrowUp} label="Data" onClick={handleNavigateTo} />
                    </div>
                </div>
            </Card>
            <Handle type="source" position={Position.Right} />
        </>
    );
}