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

import { Badge, cn, SearchInput } from '@clidey/ux';
import { 
    useGetFunctionsQuery, 
    useGetProceduresQuery, 
    useGetTriggersQuery, 
    useGetIndexesQuery, 
    useGetSequencesQuery, 
    useGetTypesQuery 
} from '@graphql';
import { BeakerIcon, BoltIcon, CodeBracketIcon, CubeIcon, DocumentDuplicateIcon, HashtagIcon, MagnifyingGlassIcon } from '@heroicons/react/24/outline';
import { FC, ReactNode, useCallback, useMemo, useState } from "react";
import { Card, ExpandableCard } from "../../components/card";
import { LoadingPage } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { useAppSelector } from "../../store/hooks";

interface EntityCardProps {
    entity: any;
    entityType: string;
}

const EntityCard: FC<EntityCardProps> = ({ entity, entityType }) => {
    const [expanded, setExpanded] = useState(false);

    const handleExpand = useCallback(() => {
        setExpanded(s => !s);
    }, []);

    const getIcon = () => {
        switch(entityType) {
            case 'functions': return <BeakerIcon className="w-4 h-4" />;
            case 'procedures': return <CodeBracketIcon className="w-4 h-4" />;
            case 'triggers': return <BoltIcon className="w-4 h-4" />;
            case 'indexes': return <DocumentDuplicateIcon className="w-4 h-4" />;
            case 'sequences': return <HashtagIcon className="w-4 h-4" />;
            case 'types': return <CubeIcon className="w-4 h-4" />;
            default: return null;
        }
    };

    const getAttributes = (): Array<{label: string, value: ReactNode}> => {
        switch(entityType) {
            case 'functions':
                return [
                    { label: 'Return Type', value: entity.ReturnType },
                    { label: 'Language', value: entity.Language },
                    { label: 'Aggregate', value: entity.IsAggregate ? 'Yes' : 'No' },
                    { label: 'Parameters', value: entity.Parameters?.map(p => `${p.Key} ${p.Value}`).join(', ') || 'None' },
                ];
            case 'procedures':
                return [
                    { label: 'Language', value: entity.Language },
                    { label: 'Parameters', value: entity.Parameters?.map(p => `${p.Key} ${p.Value}`).join(', ') || 'None' },
                ];
            case 'triggers':
                return [
                    { label: 'Table', value: entity.TableName },
                    { label: 'Event', value: entity.Event },
                    { label: 'Timing', value: entity.Timing },
                ];
            case 'indexes':
                return [
                    { label: 'Table', value: entity.TableName },
                    { label: 'Columns', value: entity.Columns?.join(', ') || '' },
                    { label: 'Type', value: entity.Type },
                    { label: 'Unique', value: entity.IsUnique ? 'Yes' : 'No' },
                    { label: 'Primary', value: entity.IsPrimary ? 'Yes' : 'No' },
                    ...(entity.Size ? [{ label: 'Size', value: entity.Size }] : []),
                ];
            case 'sequences':
                return [
                    { label: 'Data Type', value: entity.DataType },
                    { label: 'Current Value', value: entity.StartValue },
                    { label: 'Increment', value: entity.Increment },
                    { label: 'Range', value: `${entity.MinValue} - ${entity.MaxValue}` },
                    { label: 'Cache', value: entity.CacheSize },
                    { label: 'Cycle', value: entity.IsCycle ? 'Yes' : 'No' },
                ];
            case 'types':
                return [
                    { label: 'Schema', value: entity.Schema },
                    { label: 'Type', value: entity.Type },
                ];
            default:
                return [];
        }
    };

    const hasDefinition = ['functions', 'procedures', 'triggers', 'types'].includes(entityType) && entity.Definition;

    return (
        <ExpandableCard
            title={entity.Name}
            attributes={getAttributes()}
            expanded={expanded}
            onToggle={handleExpand}
            icon={getIcon()}
            className="h-fit"
        >
            {hasDefinition && expanded && (
                <div className="p-4 bg-muted/50 rounded-md font-mono text-sm overflow-x-auto">
                    <pre className="whitespace-pre-wrap">{entity.Definition}</pre>
                </div>
            )}
        </ExpandableCard>
    );
};

export const EntityPage: FC<{ entityType: string }> = ({ entityType }) => {
    const [search, setSearch] = useState("");
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);

    // Query hooks based on entity type
    const functionsQuery = useGetFunctionsQuery({
        variables: { schema },
        skip: entityType !== 'functions' || !schema
    });
    const proceduresQuery = useGetProceduresQuery({
        variables: { schema },
        skip: entityType !== 'procedures' || !schema
    });
    const triggersQuery = useGetTriggersQuery({
        variables: { schema },
        skip: entityType !== 'triggers' || !schema
    });
    const indexesQuery = useGetIndexesQuery({
        variables: { schema },
        skip: entityType !== 'indexes' || !schema
    });
    const sequencesQuery = useGetSequencesQuery({
        variables: { schema },
        skip: entityType !== 'sequences' || !schema
    });
    const typesQuery = useGetTypesQuery({
        variables: { schema },
        skip: entityType !== 'types' || !schema
    });

    // Get the appropriate query data and loading state
    const { data, loading } = useMemo(() => {
        switch(entityType) {
            case 'functions':
                return { data: functionsQuery.data?.Functions, loading: functionsQuery.loading };
            case 'procedures':
                return { data: proceduresQuery.data?.Procedures, loading: proceduresQuery.loading };
            case 'triggers':
                return { data: triggersQuery.data?.Triggers, loading: triggersQuery.loading };
            case 'indexes':
                return { data: indexesQuery.data?.Indexes, loading: indexesQuery.loading };
            case 'sequences':
                return { data: sequencesQuery.data?.Sequences, loading: sequencesQuery.loading };
            case 'types':
                return { data: typesQuery.data?.Types, loading: typesQuery.loading };
            default:
                return { data: undefined, loading: false };
        }
    }, [entityType, functionsQuery, proceduresQuery, triggersQuery, indexesQuery, sequencesQuery, typesQuery]);

    // Filter entities based on search
    const filteredEntities = useMemo(() => {
        if (!data) return [];
        if (!search) return data;
        
        return data.filter((entity: any) => 
            entity.Name?.toLowerCase().includes(search.toLowerCase()) ||
            (entity.TableName && entity.TableName.toLowerCase().includes(search.toLowerCase()))
        );
    }, [data, search]);

    // Get page title
    const pageTitle = useMemo(() => {
        switch(entityType) {
            case 'functions': return 'Functions';
            case 'procedures': return 'Procedures';
            case 'triggers': return 'Triggers';
            case 'indexes': return 'Indexes';
            case 'sequences': return 'Sequences';
            case 'types': return 'User Types';
            default: return 'Database Entities';
        }
    }, [entityType]);

    if (loading || !current) {
        return <LoadingPage />;
    }

    if (!schema) {
        return (
            <InternalPage title={pageTitle} current={current}>
                <Card className="p-8 text-center">
                    <p className="text-muted-foreground">Please select a schema to view {pageTitle.toLowerCase()}</p>
                </Card>
            </InternalPage>
        );
    }

    return (
        <InternalPage title={pageTitle} current={current}>
            <div className="flex flex-col gap-4">
                <div className="flex justify-between items-center gap-4">
                    <SearchInput
                        placeholder={`Search ${pageTitle.toLowerCase()}...`}
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        className="max-w-md"
                        icon={<MagnifyingGlassIcon className="w-4 h-4" />}
                    />
                    <Badge variant="secondary">{filteredEntities.length} {pageTitle}</Badge>
                </div>
                
                {filteredEntities.length === 0 ? (
                    <Card className="p-8 text-center">
                        <p className="text-muted-foreground">
                            {search ? `No ${pageTitle.toLowerCase()} found matching "${search}"` : `No ${pageTitle.toLowerCase()} found in this schema`}
                        </p>
                    </Card>
                ) : (
                    <div className={cn("grid gap-4 pb-8", "grid-cols-1 lg:grid-cols-2 xl:grid-cols-3")}>
                        {filteredEntities.map((entity: any) => (
                            <EntityCard 
                                key={entity.Name} 
                                entity={entity} 
                                entityType={entityType}
                            />
                        ))}
                    </div>
                )}
            </div>
        </InternalPage>
    );
};