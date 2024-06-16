import { useQuery } from "@apollo/client";
import { FC, useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AnimatedButton } from "../../components/button";
import { ExpandableCard } from "../../components/card";
import { Icons } from "../../components/icons";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, GetStorageUnitsDocument, GetStorageUnitsQuery, GetStorageUnitsQueryVariables, StorageUnit } from "../../generated/graphql";

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
        return [ unit.Attributes.slice(0,6), unit.Attributes.slice(6) ];
    }, [unit.Attributes]);

    return (<ExpandableCard key={unit.Name} isExpanded={expanded} icon={{
        bgClassName: "bg-teal-500",
        component: Icons.Tables,
    }}>
        <div className="flex flex-col grow mt-2">
            <div className="flex flex-col grow">
                <div className="text-md font-semibold mb-2">{unit.Name}</div>
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
        <div className="flex flex-col grow mt-2">
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
    const { loading, data } = useQuery<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>(GetStorageUnitsDocument, {
        variables: {
            type: DatabaseType.Postgres,
        }
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
            data?.StorageUnit.map(unit => (
                <StorageUnitCard unit={unit} />
            ))
        }
    </InternalPage>
}