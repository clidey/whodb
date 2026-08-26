export interface AtomicWhereCondition {
    ColumnType: string;
    Key: string;
    Operator: string;
    Value: string;
}
export interface Column {
    Type: string;
    Name: string;
    MetadataFidelity: SourceMetadataFidelity;
    IsPrimary: boolean;
    IsForeignKey: boolean;
    ReferencedTable?: string | null;
    ReferencedColumn?: string | null;
    Length?: number | null;
    Precision?: number | null;
    Scale?: number | null;
}
export interface ColumnDef {
    name: string;
    type: string;
    isNullable: boolean;
    isPrimary: boolean;
}
export interface Dataset {
    id: string;
    projectId: string;
    sourceId: string;
    name: string;
    description: string;
    schema: Array<ColumnDef>;
    schemaMode: string;
    ownerId: string;
    rowCount: number;
    sizeBytes: number;
    createdAt: string;
    updatedAt: string;
}
export interface DatasetQueryResult {
    columns: Array<string>;
    rows: Array<Array<string>>;
    total: number;
}
export interface OntologyAddRowsResult {
    inserted: number;
    ids: Array<string>;
}
export interface OntologyAggregateMetricInput {
    field?: string | null;
    op: string;
    as?: string | null;
}
export interface OntologyCreateRowInput {
    values: Array<RecordInput>;
}
export type OntologyDataType = 'String' | 'Integer' | 'Long' | 'Double' | 'Float' | 'Boolean' | 'Date' | 'Timestamp' | 'Array' | 'Struct' | 'UUID';
export interface OntologyDescribeInput {
    entities: Array<string>;
    includeInferred: boolean;
}
export interface OntologyDescription {
    text: string;
    entities: Array<OntologyDescriptionEntity>;
    relationships: Array<OntologyDescriptionRelationship>;
}
export interface OntologyDescriptionEntity {
    name: string;
    displayName: string;
    description: string;
    primaryKey: string;
    fields: Array<OntologyDescriptionField>;
}
export interface OntologyDescriptionField {
    name: string;
    displayName: string;
    description: string;
    type: string;
    required: boolean;
    searchable: boolean;
    sortable: boolean;
}
export interface OntologyDescriptionRelationship {
    name: string;
    source: string;
    sourceField: string;
    target: string;
    targetField: string;
    cardinality: string;
    kind: string;
}
export interface OntologyFastLookup {
    id: string;
    entityId: string;
    displayName: string;
    fields: Array<string>;
    status: string;
    reason: string;
    createdAt: string;
    updatedAt: string;
}
export interface OntologyLink {
    id: string;
    apiName: string;
    targetOntologyApiName: string;
    cardinality: OntologyLinkCardinality;
    foreignKeyProperty: string;
    targetForeignKeyProperty: string;
    joinTable: string;
    sourceColumnInJoinTable: string;
    targetColumnInJoinTable: string;
    displayName: string;
    pluralDisplayName: string;
    reverseDisplayName: string;
}
export type OntologyLinkCardinality = 'ONE_TO_ONE' | 'ONE_TO_MANY' | 'MANY_TO_ONE' | 'MANY_TO_MANY';
export interface OntologyObjectType {
    id: string;
    projectId: string;
    apiName: string;
    displayName: string;
    pluralDisplayName: string;
    description: string;
    primaryKey: string;
    sourceId?: string | null;
    tableName: string;
    schemaName: string;
    storageMode: OntologyStorageMode;
    provisioningStatus: OntologyProvisioningStatus;
    embeddingEnabled: boolean;
    embeddingProperties: Array<string>;
    status: OntologyStatus;
    icon: string;
    color: string;
    createdAt: string;
    updatedAt: string;
    properties: Array<OntologyProperty>;
    links: Array<OntologyLink>;
}
export interface OntologyProperty {
    id: string;
    apiName: string;
    displayName: string;
    description: string;
    columnName: string;
    dataType: OntologyDataType;
    arrayElementType: string;
    isRequired: boolean;
    visibility: OntologyVisibility;
    isSearchable: boolean;
    isSortable: boolean;
    isEditOnly: boolean;
    sortOrder: number;
}
export type OntologyProvisioningStatus = 'provisioning' | 'ready' | 'failed';
export interface OntologyQueryInput {
    entity: string;
    whereJson?: string | null;
    search?: string | null;
    searchFields?: Array<string> | null;
    joins?: Array<OntologyQueryJoinInput> | null;
    groupBy?: Array<string> | null;
    metrics?: Array<OntologyQueryMetricInput> | null;
    sort?: Array<OntologyQuerySortInput> | null;
    pageSize?: number | null;
    offset?: number | null;
    scanLimit?: number | null;
}
export interface OntologyQueryJoinInput {
    entity: string;
    leftField: string;
    rightField: string;
    as?: string | null;
    kind?: string | null;
}
export interface OntologyQueryMetricInput {
    op: string;
    field?: string | null;
    as?: string | null;
}
export interface OntologyQuerySortInput {
    field: string;
    desc?: boolean | null;
}
export interface OntologySimilarInput {
    entityId: string;
    rowId: string;
    topK: number;
    properties?: Array<string> | null;
    where?: WhereCondition | null;
}
export interface OntologySimilarityResult {
    rows: Array<OntologySimilarityRow>;
}
export interface OntologySimilarityRow {
    score: number;
    values: Array<OntologySimilarityValue>;
}
export interface OntologySimilarityValue {
    key: string;
    value: string;
}
export interface OntologyStatsResult {
    property: string;
    min?: string | null;
    max?: string | null;
    total: number;
    nonNull: number;
}
export type OntologyStatus = 'active' | 'experimental' | 'deprecated';
export type OntologyStorageMode = 'operational' | 'analytical';
export type OntologyVisibility = 'prominent' | 'normal' | 'hidden';
export interface OperationWhereCondition {
    Children: Array<WhereCondition>;
}
export interface Organization {
    id: string;
    name: string;
    slug: string;
    createdAt: string;
}
export interface PlatformSource {
    id: string;
    projectId: string;
    name: string;
    databaseType: string;
    createdBy: string;
    createdAt: string;
    ingestionStatus: string;
    lastEventAt?: string | null;
    ingestionError: string;
    eventsReceived: string;
}
export interface Project {
    id: string;
    orgId: string;
    name: string;
    slug: string;
    description: string;
    createdAt: string;
}
export interface QueryDatasetInput {
    projectId: string;
    datasetId: string;
    pageSize: number;
    pageOffset: number;
}
export interface Record {
    Key: string;
    Value: string;
}
export interface RecordInput {
    Key: string;
    Value: string;
    Extra?: Array<RecordInput> | null;
}
export interface RowsResult {
    Columns: Array<Column>;
    Rows: Array<Array<string>>;
    DisableUpdate: boolean;
    TotalCount: number;
}
export interface SortCondition {
    Column: string;
    Direction: SortDirection;
}
export type SortDirection = 'ASC' | 'DESC';
export type SourceAction = 'Browse' | 'Inspect' | 'ViewRows' | 'ViewContent' | 'ViewDefinition' | 'CreateChild' | 'Delete' | 'InsertData' | 'UpdateData' | 'DeleteData' | 'ImportData' | 'GenerateMockData' | 'Execute' | 'ViewGraph';
export type SourceMetadataFidelity = 'Exact' | 'Driver' | 'Sampled' | 'Inferred' | 'Synthetic' | 'Unsupported' | 'Unknown';
export interface SourceObject {
    Ref: SourceObjectRef;
    Kind: SourceObjectKind;
    Name: string;
    Path: Array<string>;
    HasChildren: boolean;
    Actions: Array<SourceAction>;
    Metadata: Array<Record>;
}
export type SourceObjectKind = 'Database' | 'Schema' | 'Table' | 'View' | 'Collection' | 'Index' | 'Key' | 'Item' | 'Function' | 'Procedure' | 'Trigger' | 'Sequence';
export interface SourceObjectRef {
    Kind: SourceObjectKind;
    Locator: string;
    Path: Array<string>;
}
export interface SourceObjectRefInput {
    Kind: SourceObjectKind;
    Locator?: string | null;
    Path: Array<string>;
}
export interface StatusResponse {
    Status: boolean;
}
export interface WhereCondition {
    Type: WhereConditionType;
    Atomic?: AtomicWhereCondition | null;
    And?: OperationWhereCondition | null;
    Or?: OperationWhereCondition | null;
}
export type WhereConditionType = 'Atomic' | 'And' | 'Or';
export interface Workspace {
    orgId?: string | null;
    projectId?: string | null;
}
