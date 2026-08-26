import type { Transport } from '../transport.js';
import type { Column, Dataset, DatasetQueryResult, OntologyAddRowsResult, OntologyAggregateMetricInput, OntologyCreateRowInput, OntologyDescribeInput, OntologyDescription, OntologyFastLookup, OntologyObjectType, OntologyQueryInput, OntologySimilarInput, OntologySimilarityResult, OntologyStatsResult, Organization, PlatformSource, Project, QueryDatasetInput, RecordInput, RowsResult, SortCondition, SourceObject, SourceObjectKind, SourceObjectRefInput, StatusResponse, WhereCondition, Workspace } from './types.js';
export interface OntologyEntitiesVars {
    projectId: string;
}
export declare const OntologyEntitiesDocument = "query OntologyEntities($projectId: ID!) { OntologyEntities(projectId: $projectId) { id projectId apiName displayName pluralDisplayName description primaryKey sourceId tableName schemaName storageMode provisioningStatus embeddingEnabled embeddingProperties status icon color createdAt updatedAt properties { id apiName displayName description columnName dataType arrayElementType isRequired visibility isSearchable isSortable isEditOnly sortOrder } links { id apiName targetOntologyApiName cardinality foreignKeyProperty targetForeignKeyProperty joinTable sourceColumnInJoinTable targetColumnInJoinTable displayName pluralDisplayName reverseDisplayName } } }";
/** Generated wire call for Query OntologyEntities. */
export declare function ontologyEntities(transport: Transport, vars: OntologyEntitiesVars): Promise<Array<OntologyObjectType>>;
export interface OntologyEntityVars {
    projectId: string;
    id: string;
}
export declare const OntologyEntityDocument = "query OntologyEntity($projectId: ID!, $id: ID!) { OntologyEntity(projectId: $projectId, id: $id) { id projectId apiName displayName pluralDisplayName description primaryKey sourceId tableName schemaName storageMode provisioningStatus embeddingEnabled embeddingProperties status icon color createdAt updatedAt properties { id apiName displayName description columnName dataType arrayElementType isRequired visibility isSearchable isSortable isEditOnly sortOrder } links { id apiName targetOntologyApiName cardinality foreignKeyProperty targetForeignKeyProperty joinTable sourceColumnInJoinTable targetColumnInJoinTable displayName pluralDisplayName reverseDisplayName } } }";
/** Generated wire call for Query OntologyEntity. */
export declare function ontologyEntity(transport: Transport, vars: OntologyEntityVars): Promise<OntologyObjectType>;
export interface OntologyDescribeVars {
    projectId: string;
    input: OntologyDescribeInput;
}
export declare const OntologyDescribeDocument = "query OntologyDescribe($projectId: ID!, $input: OntologyDescribeInput!) { OntologyDescribe(projectId: $projectId, input: $input) { text entities { name displayName description primaryKey fields { name displayName description type required searchable sortable } } relationships { name source sourceField target targetField cardinality kind } } }";
/** Generated wire call for Query OntologyDescribe. */
export declare function ontologyDescribe(transport: Transport, vars: OntologyDescribeVars): Promise<OntologyDescription>;
export interface OntologyRowsVars {
    projectId: string;
    id: string;
    pageSize: number;
    pageOffset: number;
}
export declare const OntologyRowsDocument = "query OntologyRows($projectId: ID!, $id: ID!, $pageSize: Int!, $pageOffset: Int!) { OntologyRows(projectId: $projectId, id: $id, pageSize: $pageSize, pageOffset: $pageOffset) { columns rows total } }";
/** Generated wire call for Query OntologyRows. */
export declare function ontologyRows(transport: Transport, vars: OntologyRowsVars): Promise<DatasetQueryResult>;
export interface OntologyQueryVars {
    projectId: string;
    input: OntologyQueryInput;
}
export declare const OntologyQueryDocument = "query OntologyQuery($projectId: ID!, $input: OntologyQueryInput!) { OntologyQuery(projectId: $projectId, input: $input) { columns rows total } }";
/** Generated wire call for Query OntologyQuery. */
export declare function ontologyQuery(transport: Transport, vars: OntologyQueryVars): Promise<DatasetQueryResult>;
export interface OntologyAggregateVars {
    projectId: string;
    id: string;
    groupBy: Array<string>;
    metrics: Array<OntologyAggregateMetricInput>;
    where?: WhereCondition | null;
    sort?: Array<SortCondition> | null;
    pageSize: number;
    pageOffset: number;
}
export declare const OntologyAggregateDocument = "query OntologyAggregate($projectId: ID!, $id: ID!, $groupBy: [String!]!, $metrics: [OntologyAggregateMetricInput!]!, $where: WhereCondition, $sort: [SortCondition!], $pageSize: Int!, $pageOffset: Int!) { OntologyAggregate(projectId: $projectId, id: $id, groupBy: $groupBy, metrics: $metrics, where: $where, sort: $sort, pageSize: $pageSize, pageOffset: $pageOffset) { columns rows total } }";
/** Generated wire call for Query OntologyAggregate. */
export declare function ontologyAggregate(transport: Transport, vars: OntologyAggregateVars): Promise<DatasetQueryResult>;
export interface OntologyStatsVars {
    projectId: string;
    id: string;
    property: string;
    where?: WhereCondition | null;
}
export declare const OntologyStatsDocument = "query OntologyStats($projectId: ID!, $id: ID!, $property: String!, $where: WhereCondition) { OntologyStats(projectId: $projectId, id: $id, property: $property, where: $where) { property min max total nonNull } }";
/** Generated wire call for Query OntologyStats. */
export declare function ontologyStats(transport: Transport, vars: OntologyStatsVars): Promise<OntologyStatsResult>;
export interface OntologySimilarVars {
    projectId: string;
    input: OntologySimilarInput;
}
export declare const OntologySimilarDocument = "query OntologySimilar($projectId: ID!, $input: OntologySimilarInput!) { OntologySimilar(projectId: $projectId, input: $input) { rows { score values { key value } } } }";
/** Generated wire call for Query OntologySimilar. */
export declare function ontologySimilar(transport: Transport, vars: OntologySimilarVars): Promise<OntologySimilarityResult>;
export interface OntologyFollowLinkVars {
    projectId: string;
    entityId: string;
    pk: string;
    linkApiName: string;
    pageSize: number;
    pageOffset: number;
}
export declare const OntologyFollowLinkDocument = "query OntologyFollowLink($projectId: ID!, $entityId: ID!, $pk: String!, $linkApiName: String!, $pageSize: Int!, $pageOffset: Int!) { OntologyFollowLink(projectId: $projectId, entityId: $entityId, pk: $pk, linkApiName: $linkApiName, pageSize: $pageSize, pageOffset: $pageOffset) { columns rows total } }";
/** Generated wire call for Query OntologyFollowLink. */
export declare function ontologyFollowLink(transport: Transport, vars: OntologyFollowLinkVars): Promise<DatasetQueryResult>;
export interface OntologyFollowIncomingLinkVars {
    projectId: string;
    entityId: string;
    pk: string;
    sourceEntityId: string;
    linkApiName: string;
    pageSize: number;
    pageOffset: number;
}
export declare const OntologyFollowIncomingLinkDocument = "query OntologyFollowIncomingLink($projectId: ID!, $entityId: ID!, $pk: String!, $sourceEntityId: ID!, $linkApiName: String!, $pageSize: Int!, $pageOffset: Int!) { OntologyFollowIncomingLink(projectId: $projectId, entityId: $entityId, pk: $pk, sourceEntityId: $sourceEntityId, linkApiName: $linkApiName, pageSize: $pageSize, pageOffset: $pageOffset) { columns rows total } }";
/** Generated wire call for Query OntologyFollowIncomingLink. */
export declare function ontologyFollowIncomingLink(transport: Transport, vars: OntologyFollowIncomingLinkVars): Promise<DatasetQueryResult>;
export interface OntologyFastLookupsVars {
    projectId: string;
    entityId: string;
}
export declare const OntologyFastLookupsDocument = "query OntologyFastLookups($projectId: ID!, $entityId: ID!) { OntologyFastLookups(projectId: $projectId, entityId: $entityId) { id entityId displayName fields status reason createdAt updatedAt } }";
/** Generated wire call for Query OntologyFastLookups. */
export declare function ontologyFastLookups(transport: Transport, vars: OntologyFastLookupsVars): Promise<Array<OntologyFastLookup>>;
export interface OntologyAddRowVars {
    projectId: string;
    entityId: string;
    values: Array<RecordInput>;
}
export declare const OntologyAddRowDocument = "mutation OntologyAddRow($projectId: ID!, $entityId: ID!, $values: [RecordInput!]!) { OntologyAddRow(projectId: $projectId, entityId: $entityId, values: $values) { Status } }";
/** Generated wire call for Mutation OntologyAddRow. */
export declare function ontologyAddRow(transport: Transport, vars: OntologyAddRowVars): Promise<StatusResponse>;
export interface OntologyAddRowsVars {
    projectId: string;
    entityId: string;
    rows: Array<OntologyCreateRowInput>;
    idempotencyKey?: string | null;
}
export declare const OntologyAddRowsDocument = "mutation OntologyAddRows($projectId: ID!, $entityId: ID!, $rows: [OntologyCreateRowInput!]!, $idempotencyKey: String) { OntologyAddRows(projectId: $projectId, entityId: $entityId, rows: $rows, idempotencyKey: $idempotencyKey) { inserted ids } }";
/** Generated wire call for Mutation OntologyAddRows. */
export declare function ontologyAddRows(transport: Transport, vars: OntologyAddRowsVars): Promise<OntologyAddRowsResult>;
export interface OntologyUpdateRowVars {
    projectId: string;
    entityId: string;
    values: Array<RecordInput>;
    updatedColumns: Array<string>;
}
export declare const OntologyUpdateRowDocument = "mutation OntologyUpdateRow($projectId: ID!, $entityId: ID!, $values: [RecordInput!]!, $updatedColumns: [String!]!) { OntologyUpdateRow(projectId: $projectId, entityId: $entityId, values: $values, updatedColumns: $updatedColumns) { Status } }";
/** Generated wire call for Mutation OntologyUpdateRow. */
export declare function ontologyUpdateRow(transport: Transport, vars: OntologyUpdateRowVars): Promise<StatusResponse>;
export interface OntologyDeleteRowVars {
    projectId: string;
    entityId: string;
    values: Array<RecordInput>;
}
export declare const OntologyDeleteRowDocument = "mutation OntologyDeleteRow($projectId: ID!, $entityId: ID!, $values: [RecordInput!]!) { OntologyDeleteRow(projectId: $projectId, entityId: $entityId, values: $values) { Status } }";
/** Generated wire call for Mutation OntologyDeleteRow. */
export declare function ontologyDeleteRow(transport: Transport, vars: OntologyDeleteRowVars): Promise<StatusResponse>;
export interface ProjectDatasetsVars {
    projectId: string;
}
export declare const ProjectDatasetsDocument = "query ProjectDatasets($projectId: ID!) { ProjectDatasets(projectId: $projectId) { id projectId sourceId name description schema { name type isNullable isPrimary } schemaMode ownerId rowCount sizeBytes createdAt updatedAt } }";
/** Generated wire call for Query ProjectDatasets. */
export declare function projectDatasets(transport: Transport, vars: ProjectDatasetsVars): Promise<Array<Dataset>>;
export interface QueryDatasetVars {
    input: QueryDatasetInput;
}
export declare const QueryDatasetDocument = "query QueryDataset($input: QueryDatasetInput!) { QueryDataset(input: $input) { columns rows total } }";
/** Generated wire call for Query QueryDataset. */
export declare function queryDataset(transport: Transport, vars: QueryDatasetVars): Promise<DatasetQueryResult>;
export interface ProjectSourcesVars {
    projectId: string;
}
export declare const ProjectSourcesDocument = "query ProjectSources($projectId: ID!) { ProjectSources(projectId: $projectId) { id projectId name databaseType createdBy createdAt ingestionStatus lastEventAt ingestionError eventsReceived } }";
/** Generated wire call for Query ProjectSources. */
export declare function projectSources(transport: Transport, vars: ProjectSourcesVars): Promise<Array<PlatformSource>>;
export interface PlatformSourceRowsVars {
    projectId: string;
    sourceId: string;
    ref: SourceObjectRefInput;
    where?: WhereCondition | null;
    sort?: Array<SortCondition> | null;
    pageSize: number;
    pageOffset: number;
}
export declare const PlatformSourceRowsDocument = "query PlatformSourceRows($projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!, $where: WhereCondition, $sort: [SortCondition!], $pageSize: Int!, $pageOffset: Int!) { PlatformSourceRows(projectId: $projectId, sourceId: $sourceId, ref: $ref, where: $where, sort: $sort, pageSize: $pageSize, pageOffset: $pageOffset) { Columns { Type Name MetadataFidelity IsPrimary IsForeignKey ReferencedTable ReferencedColumn Length Precision Scale } Rows DisableUpdate TotalCount } }";
/** Generated wire call for Query PlatformSourceRows. */
export declare function platformSourceRows(transport: Transport, vars: PlatformSourceRowsVars): Promise<RowsResult>;
export interface PlatformSourceObjectsVars {
    projectId: string;
    sourceId: string;
    parent?: SourceObjectRefInput | null;
    kinds?: Array<SourceObjectKind> | null;
    pageSize?: number | null;
    pageOffset?: number | null;
}
export declare const PlatformSourceObjectsDocument = "query PlatformSourceObjects($projectId: ID!, $sourceId: ID!, $parent: SourceObjectRefInput, $kinds: [SourceObjectKind!], $pageSize: Int, $pageOffset: Int) { PlatformSourceObjects(projectId: $projectId, sourceId: $sourceId, parent: $parent, kinds: $kinds, pageSize: $pageSize, pageOffset: $pageOffset) { Ref { Kind Locator Path } Kind Name Path HasChildren Actions Metadata { Key Value } } }";
/** Generated wire call for Query PlatformSourceObjects. */
export declare function platformSourceObjects(transport: Transport, vars: PlatformSourceObjectsVars): Promise<Array<SourceObject>>;
export interface PlatformSourceColumnsVars {
    projectId: string;
    sourceId: string;
    ref: SourceObjectRefInput;
}
export declare const PlatformSourceColumnsDocument = "query PlatformSourceColumns($projectId: ID!, $sourceId: ID!, $ref: SourceObjectRefInput!) { PlatformSourceColumns(projectId: $projectId, sourceId: $sourceId, ref: $ref) { Type Name MetadataFidelity IsPrimary IsForeignKey ReferencedTable ReferencedColumn Length Precision Scale } }";
/** Generated wire call for Query PlatformSourceColumns. */
export declare function platformSourceColumns(transport: Transport, vars: PlatformSourceColumnsVars): Promise<Array<Column>>;
export interface MyOrganizationsVars {
}
export declare const MyOrganizationsDocument = "query MyOrganizations { MyOrganizations { id name slug createdAt } }";
/** Generated wire call for Query MyOrganizations. */
export declare function myOrganizations(transport: Transport, vars: MyOrganizationsVars): Promise<Array<Organization>>;
export interface ProjectsVars {
    orgId: string;
}
export declare const ProjectsDocument = "query Projects($orgId: ID!) { Projects(orgId: $orgId) { id orgId name slug description createdAt } }";
/** Generated wire call for Query Projects. */
export declare function projects(transport: Transport, vars: ProjectsVars): Promise<Array<Project>>;
export interface MyWorkspaceVars {
}
export declare const MyWorkspaceDocument = "query MyWorkspace { MyWorkspace { orgId projectId } }";
/** Generated wire call for Query MyWorkspace. */
export declare function myWorkspace(transport: Transport, vars: MyWorkspaceVars): Promise<Workspace>;
