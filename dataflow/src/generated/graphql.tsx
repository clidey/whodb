import { gql } from '@apollo/client';
import * as Apollo from '@apollo/client';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
const defaultOptions = {} as const;
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Upload: { input: any; output: any; }
};

export type AiChatMessage = {
  __typename?: 'AIChatMessage';
  RequiresConfirmation: Scalars['Boolean']['output'];
  Result?: Maybe<RowsResult>;
  Text: Scalars['String']['output'];
  Type: Scalars['String']['output'];
};

export type AiProvider = {
  __typename?: 'AIProvider';
  IsEnvironmentDefined: Scalars['Boolean']['output'];
  IsGeneric: Scalars['Boolean']['output'];
  Name: Scalars['String']['output'];
  ProviderId: Scalars['String']['output'];
  Type: Scalars['String']['output'];
};

export type AwsProvider = CloudProvider & {
  __typename?: 'AWSProvider';
  DiscoverDocumentDB: Scalars['Boolean']['output'];
  DiscoverElastiCache: Scalars['Boolean']['output'];
  DiscoverRDS: Scalars['Boolean']['output'];
  DiscoveredCount: Scalars['Int']['output'];
  Error?: Maybe<Scalars['String']['output']>;
  Id: Scalars['ID']['output'];
  LastDiscoveryAt?: Maybe<Scalars['String']['output']>;
  Name: Scalars['String']['output'];
  ProfileName?: Maybe<Scalars['String']['output']>;
  ProviderType: CloudProviderType;
  Region: Scalars['String']['output'];
  Status: CloudProviderStatus;
};

export type AwsProviderInput = {
  DiscoverDocumentDB?: InputMaybe<Scalars['Boolean']['input']>;
  DiscoverElastiCache?: InputMaybe<Scalars['Boolean']['input']>;
  DiscoverRDS?: InputMaybe<Scalars['Boolean']['input']>;
  Name: Scalars['String']['input'];
  ProfileName?: InputMaybe<Scalars['String']['input']>;
  Region: Scalars['String']['input'];
};

export type AwsRegion = {
  __typename?: 'AWSRegion';
  Description: Scalars['String']['output'];
  Id: Scalars['String']['output'];
  Partition: Scalars['String']['output'];
};

export type AtomicWhereCondition = {
  ColumnType: Scalars['String']['input'];
  Key: Scalars['String']['input'];
  Operator: Scalars['String']['input'];
  Value: Scalars['String']['input'];
};

export type AuthSessionPayload = {
  __typename?: 'AuthSessionPayload';
  database: Scalars['String']['output'];
  displayName: Scalars['String']['output'];
  expiresAt: Scalars['String']['output'];
  hostname: Scalars['String']['output'];
  port: Scalars['String']['output'];
  sessionToken: Scalars['String']['output'];
  type: Scalars['String']['output'];
};

export type Capabilities = {
  __typename?: 'Capabilities';
  supportsChat: Scalars['Boolean']['output'];
  supportsDatabaseSwitch: Scalars['Boolean']['output'];
  supportsGraph: Scalars['Boolean']['output'];
  supportsModifiers: Scalars['Boolean']['output'];
  supportsSchema: Scalars['Boolean']['output'];
  supportsScratchpad: Scalars['Boolean']['output'];
};

export type ChatInput = {
  Model: Scalars['String']['input'];
  PreviousConversation: Scalars['String']['input'];
  Query: Scalars['String']['input'];
  Token?: InputMaybe<Scalars['String']['input']>;
};

export type CloudProvider = {
  DiscoveredCount: Scalars['Int']['output'];
  Error?: Maybe<Scalars['String']['output']>;
  Id: Scalars['ID']['output'];
  LastDiscoveryAt?: Maybe<Scalars['String']['output']>;
  Name: Scalars['String']['output'];
  ProviderType: CloudProviderType;
  Region: Scalars['String']['output'];
  Status: CloudProviderStatus;
};

export enum CloudProviderStatus {
  Connected = 'Connected',
  Disconnected = 'Disconnected',
  Discovering = 'Discovering',
  Error = 'Error'
}

export enum CloudProviderType {
  Aws = 'AWS'
}

export type CollectionImportError = {
  __typename?: 'CollectionImportError';
  Index: Scalars['Int']['output'];
  Reason: Scalars['String']['output'];
};

export enum CollectionImportFormat {
  Csv = 'CSV',
  Excel = 'EXCEL',
  Json = 'JSON'
}

export type CollectionImportPreview = {
  __typename?: 'CollectionImportPreview';
  Columns: Array<Scalars['String']['output']>;
  Count?: Maybe<Scalars['Int']['output']>;
  Documents: Array<Scalars['String']['output']>;
  Format: CollectionImportFormat;
  Rows: Array<Array<Scalars['String']['output']>>;
  Sheet?: Maybe<Scalars['String']['output']>;
  Sheets: Array<Scalars['String']['output']>;
  Truncated: Scalars['Boolean']['output'];
  ValidationError?: Maybe<Scalars['String']['output']>;
};

export type CollectionImportResult = {
  __typename?: 'CollectionImportResult';
  Detail?: Maybe<Scalars['String']['output']>;
  Errors: Array<CollectionImportError>;
  ImportedCount: Scalars['Int']['output'];
  MatchedCount?: Maybe<Scalars['Int']['output']>;
  Message?: Maybe<Scalars['String']['output']>;
  ModifiedCount?: Maybe<Scalars['Int']['output']>;
  SkippedCount: Scalars['Int']['output'];
  Status: Scalars['Boolean']['output'];
  UpsertedCount?: Maybe<Scalars['Int']['output']>;
};

export type Column = {
  __typename?: 'Column';
  IsForeignKey: Scalars['Boolean']['output'];
  IsPrimary: Scalars['Boolean']['output'];
  Length?: Maybe<Scalars['Int']['output']>;
  Name: Scalars['String']['output'];
  Precision?: Maybe<Scalars['Int']['output']>;
  ReferencedColumn?: Maybe<Scalars['String']['output']>;
  ReferencedTable?: Maybe<Scalars['String']['output']>;
  Scale?: Maybe<Scalars['Int']['output']>;
  Type: Scalars['String']['output'];
};

export enum ConnectionStatus {
  Available = 'Available',
  Deleting = 'Deleting',
  Failed = 'Failed',
  Starting = 'Starting',
  Stopped = 'Stopped',
  Unknown = 'Unknown'
}

export type Dashboard = {
  __typename?: 'Dashboard';
  CreatedAt: Scalars['String']['output'];
  Description?: Maybe<Scalars['String']['output']>;
  ID: Scalars['ID']['output'];
  Name: Scalars['String']['output'];
  RefreshRule: Scalars['String']['output'];
  UpdatedAt: Scalars['String']['output'];
  Widgets: Array<DashboardWidget>;
};

export type DashboardWidget = {
  __typename?: 'DashboardWidget';
  Description?: Maybe<Scalars['String']['output']>;
  ID: Scalars['ID']['output'];
  Layout: Scalars['String']['output'];
  Query?: Maybe<Scalars['String']['output']>;
  QueryContext?: Maybe<Scalars['String']['output']>;
  Snapshot?: Maybe<Scalars['String']['output']>;
  SortOrder: Scalars['Int']['output'];
  Title: Scalars['String']['output'];
  Type: Scalars['String']['output'];
  Visualization?: Maybe<Scalars['String']['output']>;
};

export type DatabaseMetadata = {
  __typename?: 'DatabaseMetadata';
  aliasMap: Array<Record>;
  capabilities: Capabilities;
  databaseType: Scalars['String']['output'];
  operators: Array<Scalars['String']['output']>;
  systemSchemas: Array<Scalars['String']['output']>;
  typeDefinitions: Array<TypeDefinition>;
};

export type DatabaseQuerySuggestion = {
  __typename?: 'DatabaseQuerySuggestion';
  category: Scalars['String']['output'];
  description: Scalars['String']['output'];
};

export enum DatabaseType {
  ClickHouse = 'ClickHouse',
  ElasticSearch = 'ElasticSearch',
  MariaDb = 'MariaDB',
  MongoDb = 'MongoDB',
  MySql = 'MySQL',
  Postgres = 'Postgres',
  Redis = 'Redis',
  Sqlite3 = 'Sqlite3'
}

export type DiscoveredConnection = {
  __typename?: 'DiscoveredConnection';
  DatabaseType: Scalars['String']['output'];
  Id: Scalars['ID']['output'];
  Metadata: Array<Record>;
  Name: Scalars['String']['output'];
  ProviderID: Scalars['String']['output'];
  ProviderType: CloudProviderType;
  Region?: Maybe<Scalars['String']['output']>;
  Status: ConnectionStatus;
};

export type ExportDownload = {
  __typename?: 'ExportDownload';
  ContentType: Scalars['String']['output'];
  DownloadURL: Scalars['String']['output'];
  Filename: Scalars['String']['output'];
  ID: Scalars['ID']['output'];
  Size: Scalars['Int']['output'];
};

export type GenerateChatTitleInput = {
  Endpoint?: InputMaybe<Scalars['String']['input']>;
  Model: Scalars['String']['input'];
  ModelType: Scalars['String']['input'];
  ProviderId?: InputMaybe<Scalars['String']['input']>;
  Query: Scalars['String']['input'];
  Token?: InputMaybe<Scalars['String']['input']>;
};

export type GenerateChatTitleResponse = {
  __typename?: 'GenerateChatTitleResponse';
  Title: Scalars['String']['output'];
};

export type GraphUnit = {
  __typename?: 'GraphUnit';
  Relations: Array<GraphUnitRelationship>;
  Unit: StorageUnit;
};

export type GraphUnitRelationship = {
  __typename?: 'GraphUnitRelationship';
  Name: Scalars['String']['output'];
  Relationship: GraphUnitRelationshipType;
  SourceColumn?: Maybe<Scalars['String']['output']>;
  TargetColumn?: Maybe<Scalars['String']['output']>;
};

export enum GraphUnitRelationshipType {
  ManyToMany = 'ManyToMany',
  ManyToOne = 'ManyToOne',
  OneToMany = 'OneToMany',
  OneToOne = 'OneToOne',
  Unknown = 'Unknown'
}

export type HealthStatus = {
  __typename?: 'HealthStatus';
  Database: Scalars['String']['output'];
  Server: Scalars['String']['output'];
};

export type ImportCollectionFileInput = {
  Collection: Scalars['String']['input'];
  Delimiter?: InputMaybe<Scalars['String']['input']>;
  File: Scalars['Upload']['input'];
  Format: CollectionImportFormat;
  Mode: ImportMode;
  Schema: Scalars['String']['input'];
  Sheet?: InputMaybe<Scalars['String']['input']>;
  SkipColumns?: InputMaybe<Array<Scalars['String']['input']>>;
  UpsertKeys?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type ImportCollectionPreviewInput = {
  Delimiter?: InputMaybe<Scalars['String']['input']>;
  File: Scalars['Upload']['input'];
  Format: CollectionImportFormat;
  Sheet?: InputMaybe<Scalars['String']['input']>;
};

export type ImportColumnMapping = {
  Skip: Scalars['Boolean']['input'];
  SourceColumn: Scalars['String']['input'];
  TargetColumn?: InputMaybe<Scalars['String']['input']>;
};

export type ImportColumnMappingPreview = {
  __typename?: 'ImportColumnMappingPreview';
  SourceColumn: Scalars['String']['output'];
  TargetColumn: Scalars['String']['output'];
};

export type ImportExistingTableInput = {
  AllowAutoGenerated?: InputMaybe<Scalars['Boolean']['input']>;
  Mapping: Array<ImportColumnMapping>;
  Mode: ImportMode;
  Schema: Scalars['String']['input'];
  StorageUnit: Scalars['String']['input'];
};

export type ImportExistingTablePreviewInput = {
  Schema: Scalars['String']['input'];
  StorageUnit: Scalars['String']['input'];
  UseHeaderMapping: Scalars['Boolean']['input'];
};

export enum ImportFileFormat {
  Csv = 'CSV',
  Excel = 'EXCEL'
}

export type ImportFileInput = {
  ExistingTable?: InputMaybe<ImportExistingTableInput>;
  File: Scalars['Upload']['input'];
  NewTable?: InputMaybe<ImportNewTableInput>;
  Options: ImportFileOptions;
  TargetMode: ImportTargetMode;
};

export type ImportFileOptions = {
  Delimiter?: InputMaybe<Scalars['String']['input']>;
  Format: ImportFileFormat;
  Sheet?: InputMaybe<Scalars['String']['input']>;
};

export enum ImportMode {
  Append = 'APPEND',
  Overwrite = 'OVERWRITE',
  Upsert = 'UPSERT'
}

export type ImportNewTableColumnInput = {
  Nullable: Scalars['Boolean']['input'];
  Primary: Scalars['Boolean']['input'];
  Skip: Scalars['Boolean']['input'];
  SourceColumn: Scalars['String']['input'];
  TargetColumn?: InputMaybe<Scalars['String']['input']>;
  Type?: InputMaybe<Scalars['String']['input']>;
};

export type ImportNewTableColumnPreview = {
  __typename?: 'ImportNewTableColumnPreview';
  Nullable: Scalars['Boolean']['output'];
  Primary: Scalars['Boolean']['output'];
  Skip: Scalars['Boolean']['output'];
  SourceColumn: Scalars['String']['output'];
  TargetColumn: Scalars['String']['output'];
  Type: Scalars['String']['output'];
};

export type ImportNewTableInput = {
  Columns: Array<ImportNewTableColumnInput>;
  Schema: Scalars['String']['input'];
  TableName: Scalars['String']['input'];
};

export type ImportNewTablePreview = {
  __typename?: 'ImportNewTablePreview';
  Columns: Array<ImportNewTableColumnPreview>;
  Issues: Array<ImportSuggestionIssue>;
  TableName: Scalars['String']['output'];
};

export type ImportNewTablePreviewInput = {
  Schema: Scalars['String']['input'];
};

export type ImportPreview = {
  __typename?: 'ImportPreview';
  AutoGeneratedColumns: Array<Scalars['String']['output']>;
  Columns: Array<Scalars['String']['output']>;
  Mapping?: Maybe<Array<ImportColumnMappingPreview>>;
  NewTable?: Maybe<ImportNewTablePreview>;
  RequiresAllowAutoGenerated: Scalars['Boolean']['output'];
  Rows: Array<Array<Scalars['String']['output']>>;
  Sheet?: Maybe<Scalars['String']['output']>;
  Sheets: Array<Scalars['String']['output']>;
  Truncated: Scalars['Boolean']['output'];
  ValidationError?: Maybe<Scalars['String']['output']>;
};

export type ImportPreviewInput = {
  ExistingTable?: InputMaybe<ImportExistingTablePreviewInput>;
  File: Scalars['Upload']['input'];
  NewTable?: InputMaybe<ImportNewTablePreviewInput>;
  Options: ImportFileOptions;
  TargetMode: ImportTargetMode;
};

export type ImportResult = {
  __typename?: 'ImportResult';
  Detail?: Maybe<Scalars['String']['output']>;
  Message: Scalars['String']['output'];
  Status: Scalars['Boolean']['output'];
};

export type ImportSqlInput = {
  File?: InputMaybe<Scalars['Upload']['input']>;
  Filename?: InputMaybe<Scalars['String']['input']>;
  Script?: InputMaybe<Scalars['String']['input']>;
};

export type ImportSuggestionIssue = {
  __typename?: 'ImportSuggestionIssue';
  Field: Scalars['String']['output'];
  Key: Scalars['String']['output'];
  SourceColumn?: Maybe<Scalars['String']['output']>;
};

export enum ImportTargetMode {
  ExistingTable = 'EXISTING_TABLE',
  NewTable = 'NEW_TABLE'
}

export type LayoutInput = {
  Layout: Scalars['String']['input'];
  WidgetID: Scalars['ID']['input'];
};

export type LocalAwsProfile = {
  __typename?: 'LocalAWSProfile';
  AuthType: Scalars['String']['output'];
  IsDefault: Scalars['Boolean']['output'];
  Name: Scalars['String']['output'];
  Region?: Maybe<Scalars['String']['output']>;
  Source: Scalars['String']['output'];
};

export type LoginCredentials = {
  Advanced?: InputMaybe<Array<RecordInput>>;
  Database: Scalars['String']['input'];
  Hostname: Scalars['String']['input'];
  Id?: InputMaybe<Scalars['String']['input']>;
  Password: Scalars['String']['input'];
  Type: Scalars['String']['input'];
  Username: Scalars['String']['input'];
};

export type LoginProfile = {
  __typename?: 'LoginProfile';
  Alias?: Maybe<Scalars['String']['output']>;
  Database?: Maybe<Scalars['String']['output']>;
  Hostname?: Maybe<Scalars['String']['output']>;
  Id: Scalars['String']['output'];
  IsEnvironmentDefined: Scalars['Boolean']['output'];
  SSLConfigured: Scalars['Boolean']['output'];
  Source: Scalars['String']['output'];
  Type: DatabaseType;
};

export type LoginProfileInput = {
  Database?: InputMaybe<Scalars['String']['input']>;
  Id: Scalars['String']['input'];
  Type: DatabaseType;
};

export type MockDataDependencyAnalysis = {
  __typename?: 'MockDataDependencyAnalysis';
  Error?: Maybe<Scalars['String']['output']>;
  GenerationOrder: Array<Scalars['String']['output']>;
  Tables: Array<MockDataTableInfo>;
  TotalRows: Scalars['Int']['output'];
  Warnings: Array<Scalars['String']['output']>;
};

export type MockDataGenerationInput = {
  FkDensityRatio?: InputMaybe<Scalars['Int']['input']>;
  Method: Scalars['String']['input'];
  OverwriteExisting: Scalars['Boolean']['input'];
  RowCount: Scalars['Int']['input'];
  Schema: Scalars['String']['input'];
  StorageUnit: Scalars['String']['input'];
};

export type MockDataGenerationStatus = {
  __typename?: 'MockDataGenerationStatus';
  AmountGenerated: Scalars['Int']['output'];
  Details?: Maybe<Array<MockDataTableDetail>>;
};

export type MockDataTableDetail = {
  __typename?: 'MockDataTableDetail';
  RowsGenerated: Scalars['Int']['output'];
  Table: Scalars['String']['output'];
  UsedExistingData: Scalars['Boolean']['output'];
};

export type MockDataTableInfo = {
  __typename?: 'MockDataTableInfo';
  IsBlocked: Scalars['Boolean']['output'];
  RowsToGenerate: Scalars['Int']['output'];
  Table: Scalars['String']['output'];
  UsesExistingData: Scalars['Boolean']['output'];
};

export type Mutation = {
  __typename?: 'Mutation';
  AddAWSProvider: AwsProvider;
  AddRow: StatusResponse;
  AddStorageUnit: StatusResponse;
  AddWidget: DashboardWidget;
  BootstrapSealosSession: AuthSessionPayload;
  CreateDashboard: Dashboard;
  CreateSQLDataExport: ExportDownload;
  CreateStandaloneSession: AuthSessionPayload;
  DeleteDashboard: StatusResponse;
  DeleteRow: StatusResponse;
  DeleteWidget: StatusResponse;
  ExecuteConfirmedSQL: AiChatMessage;
  GenerateChatTitle: GenerateChatTitleResponse;
  GenerateMockData: MockDataGenerationStatus;
  GenerateRDSAuthToken: Scalars['String']['output'];
  ImportCollectionFile: CollectionImportResult;
  ImportCollectionPreview: CollectionImportPreview;
  ImportPreview: ImportPreview;
  ImportSQL: ImportResult;
  ImportTableFile: ImportResult;
  Login: StatusResponse;
  LoginWithProfile: StatusResponse;
  Logout: StatusResponse;
  RefreshCloudProvider: AwsProvider;
  RemoveCloudProvider: StatusResponse;
  ReplaceRow: StatusResponse;
  TestAWSCredentials: CloudProviderStatus;
  TestCloudProvider: CloudProviderStatus;
  UpdateAWSProvider: AwsProvider;
  UpdateDashboard: Dashboard;
  UpdateSettings: StatusResponse;
  UpdateStorageUnit: StatusResponse;
  UpdateWidget: DashboardWidget;
  UpdateWidgetLayouts: StatusResponse;
  UpdateWidgetSnapshot: StatusResponse;
};


export type MutationAddAwsProviderArgs = {
  input: AwsProviderInput;
};


export type MutationAddRowArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput>;
};


export type MutationAddStorageUnitArgs = {
  fields: Array<RecordInput>;
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
};


export type MutationAddWidgetArgs = {
  dashboardId: Scalars['ID']['input'];
  input: WidgetInput;
};


export type MutationBootstrapSealosSessionArgs = {
  input: SealosBootstrapInput;
};


export type MutationCreateDashboardArgs = {
  description?: InputMaybe<Scalars['String']['input']>;
  name: Scalars['String']['input'];
  refreshRule: Scalars['String']['input'];
};


export type MutationCreateSqlDataExportArgs = {
  input: SqlDataExportInput;
};


export type MutationCreateStandaloneSessionArgs = {
  credentials: LoginCredentials;
};


export type MutationDeleteDashboardArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteRowArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput>;
};


export type MutationDeleteWidgetArgs = {
  id: Scalars['ID']['input'];
};


export type MutationExecuteConfirmedSqlArgs = {
  operationType: Scalars['String']['input'];
  query: Scalars['String']['input'];
};


export type MutationGenerateChatTitleArgs = {
  input: GenerateChatTitleInput;
};


export type MutationGenerateMockDataArgs = {
  input: MockDataGenerationInput;
};


export type MutationGenerateRdsAuthTokenArgs = {
  endpoint: Scalars['String']['input'];
  port: Scalars['Int']['input'];
  providerID: Scalars['ID']['input'];
  region: Scalars['String']['input'];
  username: Scalars['String']['input'];
};


export type MutationImportCollectionFileArgs = {
  input: ImportCollectionFileInput;
};


export type MutationImportCollectionPreviewArgs = {
  input: ImportCollectionPreviewInput;
};


export type MutationImportPreviewArgs = {
  input: ImportPreviewInput;
};


export type MutationImportSqlArgs = {
  input: ImportSqlInput;
};


export type MutationImportTableFileArgs = {
  input: ImportFileInput;
};


export type MutationLoginArgs = {
  credentials: LoginCredentials;
};


export type MutationLoginWithProfileArgs = {
  profile: LoginProfileInput;
};


export type MutationRefreshCloudProviderArgs = {
  id: Scalars['ID']['input'];
};


export type MutationRemoveCloudProviderArgs = {
  id: Scalars['ID']['input'];
};


export type MutationReplaceRowArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput>;
};


export type MutationTestAwsCredentialsArgs = {
  input: AwsProviderInput;
};


export type MutationTestCloudProviderArgs = {
  id: Scalars['ID']['input'];
};


export type MutationUpdateAwsProviderArgs = {
  id: Scalars['ID']['input'];
  input: AwsProviderInput;
};


export type MutationUpdateDashboardArgs = {
  description?: InputMaybe<Scalars['String']['input']>;
  id: Scalars['ID']['input'];
  name?: InputMaybe<Scalars['String']['input']>;
  refreshRule?: InputMaybe<Scalars['String']['input']>;
};


export type MutationUpdateSettingsArgs = {
  newSettings: SettingsConfigInput;
};


export type MutationUpdateStorageUnitArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  updatedColumns: Array<Scalars['String']['input']>;
  values: Array<RecordInput>;
};


export type MutationUpdateWidgetArgs = {
  id: Scalars['ID']['input'];
  input: UpdateWidgetInput;
};


export type MutationUpdateWidgetLayoutsArgs = {
  dashboardId: Scalars['ID']['input'];
  layouts: Array<LayoutInput>;
};


export type MutationUpdateWidgetSnapshotArgs = {
  id: Scalars['ID']['input'];
  snapshot: SnapshotInput;
};

export type OperationWhereCondition = {
  Children: Array<WhereCondition>;
};

export type Query = {
  __typename?: 'Query';
  AIChat: Array<AiChatMessage>;
  AIModel: Array<Scalars['String']['output']>;
  AIProviders: Array<AiProvider>;
  AWSRegions: Array<AwsRegion>;
  AnalyzeMockDataDependencies: MockDataDependencyAnalysis;
  CloudProvider?: Maybe<AwsProvider>;
  CloudProviders: Array<AwsProvider>;
  Columns: Array<Column>;
  ColumnsBatch: Array<StorageUnitColumns>;
  Database: Array<Scalars['String']['output']>;
  DatabaseMetadata?: Maybe<DatabaseMetadata>;
  DatabaseQuerySuggestions: Array<DatabaseQuerySuggestion>;
  DiscoveredConnections: Array<DiscoveredConnection>;
  GetDashboards: Array<Dashboard>;
  Graph: Array<GraphUnit>;
  Health: HealthStatus;
  LocalAWSProfiles: Array<LocalAwsProfile>;
  MockDataMaxRowCount: Scalars['Int']['output'];
  Profiles: Array<LoginProfile>;
  ProviderConnections: Array<DiscoveredConnection>;
  RawExecute: RowsResult;
  Row: RowsResult;
  SSLStatus?: Maybe<SslStatus>;
  Schema: Array<Scalars['String']['output']>;
  SettingsConfig: SettingsConfig;
  StorageUnit: Array<StorageUnit>;
  UpdateInfo: UpdateInfo;
  Version: Scalars['String']['output'];
};


export type QueryAiChatArgs = {
  input: ChatInput;
  modelType: Scalars['String']['input'];
  providerId?: InputMaybe<Scalars['String']['input']>;
  schema: Scalars['String']['input'];
  token?: InputMaybe<Scalars['String']['input']>;
};


export type QueryAiModelArgs = {
  modelType: Scalars['String']['input'];
  providerId?: InputMaybe<Scalars['String']['input']>;
  token?: InputMaybe<Scalars['String']['input']>;
};


export type QueryAnalyzeMockDataDependenciesArgs = {
  fkDensityRatio?: InputMaybe<Scalars['Int']['input']>;
  rowCount: Scalars['Int']['input'];
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
};


export type QueryCloudProviderArgs = {
  id: Scalars['ID']['input'];
};


export type QueryColumnsArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
};


export type QueryColumnsBatchArgs = {
  schema: Scalars['String']['input'];
  storageUnits: Array<Scalars['String']['input']>;
};


export type QueryDatabaseArgs = {
  type: Scalars['String']['input'];
};


export type QueryDatabaseQuerySuggestionsArgs = {
  schema: Scalars['String']['input'];
};


export type QueryGraphArgs = {
  schema: Scalars['String']['input'];
};


export type QueryProviderConnectionsArgs = {
  providerID: Scalars['ID']['input'];
};


export type QueryRawExecuteArgs = {
  query: Scalars['String']['input'];
};


export type QueryRowArgs = {
  pageOffset: Scalars['Int']['input'];
  pageSize: Scalars['Int']['input'];
  schema: Scalars['String']['input'];
  sort?: InputMaybe<Array<SortCondition>>;
  storageUnit: Scalars['String']['input'];
  where?: InputMaybe<WhereCondition>;
};


export type QueryStorageUnitArgs = {
  schema: Scalars['String']['input'];
};

export type Record = {
  __typename?: 'Record';
  Key: Scalars['String']['output'];
  Value: Scalars['String']['output'];
};

export type RecordInput = {
  Extra?: InputMaybe<Array<RecordInput>>;
  Key: Scalars['String']['input'];
  Value: Scalars['String']['input'];
};

export type RowsResult = {
  __typename?: 'RowsResult';
  Columns: Array<Column>;
  DisableUpdate: Scalars['Boolean']['output'];
  Rows: Array<Array<Scalars['String']['output']>>;
  TotalCount: Scalars['Int']['output'];
};

export type SqlDataExportInput = {
  Limit?: InputMaybe<Scalars['Int']['input']>;
  Mode: SqlDataExportMode;
  Schema: Scalars['String']['input'];
  Sort?: InputMaybe<Array<SortCondition>>;
  StorageUnit: Scalars['String']['input'];
  Where?: InputMaybe<WhereCondition>;
};

export enum SqlDataExportMode {
  Insert = 'INSERT',
  Update = 'UPDATE'
}

export type SslStatus = {
  __typename?: 'SSLStatus';
  IsEnabled: Scalars['Boolean']['output'];
  Mode: Scalars['String']['output'];
};

export type SealosBootstrapInput = {
  databaseName?: InputMaybe<Scalars['String']['input']>;
  dbType: Scalars['String']['input'];
  host?: InputMaybe<Scalars['String']['input']>;
  kubeconfig: Scalars['String']['input'];
  namespace?: InputMaybe<Scalars['String']['input']>;
  port?: InputMaybe<Scalars['String']['input']>;
  resourceName: Scalars['String']['input'];
};

export type SettingsConfig = {
  __typename?: 'SettingsConfig';
  CloudProvidersEnabled: Scalars['Boolean']['output'];
  DisableCredentialForm: Scalars['Boolean']['output'];
  MaxPageSize: Scalars['Int']['output'];
  MetricsEnabled?: Maybe<Scalars['Boolean']['output']>;
  StandaloneLoginEnabled: Scalars['Boolean']['output'];
};

export type SettingsConfigInput = {
  MetricsEnabled?: InputMaybe<Scalars['String']['input']>;
};

export type SnapshotInput = {
  Config: Scalars['String']['input'];
  Data: Scalars['String']['input'];
  ExecutedAt: Scalars['String']['input'];
};

export type SortCondition = {
  Column: Scalars['String']['input'];
  Direction: SortDirection;
};

export enum SortDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export type StatusResponse = {
  __typename?: 'StatusResponse';
  Status: Scalars['Boolean']['output'];
};

export type StorageUnit = {
  __typename?: 'StorageUnit';
  Attributes: Array<Record>;
  IsMockDataGenerationAllowed: Scalars['Boolean']['output'];
  Name: Scalars['String']['output'];
};

export type StorageUnitColumns = {
  __typename?: 'StorageUnitColumns';
  Columns: Array<Column>;
  StorageUnit: Scalars['String']['output'];
};

export enum TypeCategory {
  Binary = 'binary',
  Boolean = 'boolean',
  Datetime = 'datetime',
  Json = 'json',
  Numeric = 'numeric',
  Other = 'other',
  Text = 'text'
}

export type TypeDefinition = {
  __typename?: 'TypeDefinition';
  category: TypeCategory;
  defaultLength?: Maybe<Scalars['Int']['output']>;
  defaultPrecision?: Maybe<Scalars['Int']['output']>;
  hasLength: Scalars['Boolean']['output'];
  hasPrecision: Scalars['Boolean']['output'];
  id: Scalars['String']['output'];
  label: Scalars['String']['output'];
};

export type UpdateInfo = {
  __typename?: 'UpdateInfo';
  currentVersion: Scalars['String']['output'];
  latestVersion: Scalars['String']['output'];
  releaseURL: Scalars['String']['output'];
  updateAvailable: Scalars['Boolean']['output'];
};

export type UpdateWidgetInput = {
  Description?: InputMaybe<Scalars['String']['input']>;
  Layout?: InputMaybe<Scalars['String']['input']>;
  Query?: InputMaybe<Scalars['String']['input']>;
  QueryContext?: InputMaybe<Scalars['String']['input']>;
  Snapshot?: InputMaybe<Scalars['String']['input']>;
  SortOrder?: InputMaybe<Scalars['Int']['input']>;
  Title?: InputMaybe<Scalars['String']['input']>;
  Visualization?: InputMaybe<Scalars['String']['input']>;
};

export type WhereCondition = {
  And?: InputMaybe<OperationWhereCondition>;
  Atomic?: InputMaybe<AtomicWhereCondition>;
  Or?: InputMaybe<OperationWhereCondition>;
  Type: WhereConditionType;
};

export enum WhereConditionType {
  And = 'And',
  Atomic = 'Atomic',
  Or = 'Or'
}

export type WidgetInput = {
  Description?: InputMaybe<Scalars['String']['input']>;
  Layout: Scalars['String']['input'];
  Query?: InputMaybe<Scalars['String']['input']>;
  QueryContext?: InputMaybe<Scalars['String']['input']>;
  Snapshot?: InputMaybe<Scalars['String']['input']>;
  SortOrder?: InputMaybe<Scalars['Int']['input']>;
  Title: Scalars['String']['input'];
  Type: Scalars['String']['input'];
  Visualization?: InputMaybe<Scalars['String']['input']>;
};

export type AddRowMutationVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput> | RecordInput;
}>;


export type AddRowMutation = { __typename?: 'Mutation', AddRow: { __typename?: 'StatusResponse', Status: boolean } };

export type AddStorageUnitMutationVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  fields: Array<RecordInput> | RecordInput;
}>;


export type AddStorageUnitMutation = { __typename?: 'Mutation', AddStorageUnit: { __typename?: 'StatusResponse', Status: boolean } };

export type AddWidgetMutationVariables = Exact<{
  dashboardId: Scalars['ID']['input'];
  input: WidgetInput;
}>;


export type AddWidgetMutation = { __typename?: 'Mutation', AddWidget: { __typename?: 'DashboardWidget', ID: string, Type: string, Title: string, Description?: string | null, Layout: string, Query?: string | null, QueryContext?: string | null, Visualization?: string | null, Snapshot?: string | null, SortOrder: number } };

export type BootstrapSealosSessionMutationVariables = Exact<{
  input: SealosBootstrapInput;
}>;


export type BootstrapSealosSessionMutation = { __typename?: 'Mutation', BootstrapSealosSession: { __typename?: 'AuthSessionPayload', sessionToken: string, expiresAt: string, type: string, hostname: string, port: string, database: string, displayName: string } };

export type CreateDashboardMutationVariables = Exact<{
  name: Scalars['String']['input'];
  description?: InputMaybe<Scalars['String']['input']>;
  refreshRule: Scalars['String']['input'];
}>;


export type CreateDashboardMutation = { __typename?: 'Mutation', CreateDashboard: { __typename?: 'Dashboard', ID: string, Name: string, Description?: string | null, RefreshRule: string, CreatedAt: string, UpdatedAt: string, Widgets: Array<{ __typename?: 'DashboardWidget', ID: string, Type: string, Title: string, Description?: string | null, Layout: string, Query?: string | null, QueryContext?: string | null, Visualization?: string | null, Snapshot?: string | null, SortOrder: number }> } };

export type CreateSqlDataExportMutationVariables = Exact<{
  input: SqlDataExportInput;
}>;


export type CreateSqlDataExportMutation = { __typename?: 'Mutation', CreateSQLDataExport: { __typename?: 'ExportDownload', ID: string, Filename: string, ContentType: string, DownloadURL: string, Size: number } };

export type CreateStandaloneSessionMutationVariables = Exact<{
  credentials: LoginCredentials;
}>;


export type CreateStandaloneSessionMutation = { __typename?: 'Mutation', CreateStandaloneSession: { __typename?: 'AuthSessionPayload', sessionToken: string, expiresAt: string, type: string, hostname: string, port: string, database: string, displayName: string } };

export type DeleteDashboardMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteDashboardMutation = { __typename?: 'Mutation', DeleteDashboard: { __typename?: 'StatusResponse', Status: boolean } };

export type DeleteRowMutationVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput> | RecordInput;
}>;


export type DeleteRowMutation = { __typename?: 'Mutation', DeleteRow: { __typename?: 'StatusResponse', Status: boolean } };

export type DeleteWidgetMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteWidgetMutation = { __typename?: 'Mutation', DeleteWidget: { __typename?: 'StatusResponse', Status: boolean } };

export type ExecuteConfirmedSqlMutationVariables = Exact<{
  query: Scalars['String']['input'];
  operationType: Scalars['String']['input'];
}>;


export type ExecuteConfirmedSqlMutation = { __typename?: 'Mutation', ExecuteConfirmedSQL: { __typename?: 'AIChatMessage', Type: string, Text: string, RequiresConfirmation: boolean, Result?: { __typename?: 'RowsResult', Rows: Array<Array<string>>, TotalCount: number, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } | null } };

export type ImportCollectionFileMutationVariables = Exact<{
  input: ImportCollectionFileInput;
}>;


export type ImportCollectionFileMutation = { __typename?: 'Mutation', ImportCollectionFile: { __typename?: 'CollectionImportResult', Status: boolean, ImportedCount: number, SkippedCount: number, MatchedCount?: number | null, ModifiedCount?: number | null, UpsertedCount?: number | null, Message?: string | null, Detail?: string | null, Errors: Array<{ __typename?: 'CollectionImportError', Index: number, Reason: string }> } };

export type ImportCollectionPreviewMutationVariables = Exact<{
  input: ImportCollectionPreviewInput;
}>;


export type ImportCollectionPreviewMutation = { __typename?: 'Mutation', ImportCollectionPreview: { __typename?: 'CollectionImportPreview', Format: CollectionImportFormat, Sheet?: string | null, Sheets: Array<string>, Columns: Array<string>, Rows: Array<Array<string>>, Documents: Array<string>, Count?: number | null, Truncated: boolean, ValidationError?: string | null } };

export type ImportPreviewMutationVariables = Exact<{
  input: ImportPreviewInput;
}>;


export type ImportPreviewMutation = { __typename?: 'Mutation', ImportPreview: { __typename?: 'ImportPreview', Sheet?: string | null, Sheets: Array<string>, Columns: Array<string>, Rows: Array<Array<string>>, Truncated: boolean, ValidationError?: string | null, RequiresAllowAutoGenerated: boolean, AutoGeneratedColumns: Array<string>, Mapping?: Array<{ __typename?: 'ImportColumnMappingPreview', SourceColumn: string, TargetColumn: string }> | null, NewTable?: { __typename?: 'ImportNewTablePreview', TableName: string, Columns: Array<{ __typename?: 'ImportNewTableColumnPreview', SourceColumn: string, TargetColumn: string, Type: string, Nullable: boolean, Primary: boolean, Skip: boolean }>, Issues: Array<{ __typename?: 'ImportSuggestionIssue', Key: string, Field: string, SourceColumn?: string | null }> } | null } };

export type ImportSqlMutationVariables = Exact<{
  input: ImportSqlInput;
}>;


export type ImportSqlMutation = { __typename?: 'Mutation', ImportSQL: { __typename?: 'ImportResult', Status: boolean, Message: string, Detail?: string | null } };

export type ImportTableFileMutationVariables = Exact<{
  input: ImportFileInput;
}>;


export type ImportTableFileMutation = { __typename?: 'Mutation', ImportTableFile: { __typename?: 'ImportResult', Status: boolean, Message: string, Detail?: string | null } };

export type LoginMutationVariables = Exact<{
  credentials: LoginCredentials;
}>;


export type LoginMutation = { __typename?: 'Mutation', Login: { __typename?: 'StatusResponse', Status: boolean } };

export type ReplaceRowMutationVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput> | RecordInput;
}>;


export type ReplaceRowMutation = { __typename?: 'Mutation', ReplaceRow: { __typename?: 'StatusResponse', Status: boolean } };

export type UpdateDashboardMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  name?: InputMaybe<Scalars['String']['input']>;
  description?: InputMaybe<Scalars['String']['input']>;
  refreshRule?: InputMaybe<Scalars['String']['input']>;
}>;


export type UpdateDashboardMutation = { __typename?: 'Mutation', UpdateDashboard: { __typename?: 'Dashboard', ID: string, Name: string, Description?: string | null, RefreshRule: string, CreatedAt: string, UpdatedAt: string, Widgets: Array<{ __typename?: 'DashboardWidget', ID: string, Type: string, Title: string, Description?: string | null, Layout: string, Query?: string | null, QueryContext?: string | null, Visualization?: string | null, Snapshot?: string | null, SortOrder: number }> } };

export type UpdateStorageUnitMutationVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput> | RecordInput;
  updatedColumns: Array<Scalars['String']['input']> | Scalars['String']['input'];
}>;


export type UpdateStorageUnitMutation = { __typename?: 'Mutation', UpdateStorageUnit: { __typename?: 'StatusResponse', Status: boolean } };

export type UpdateWidgetLayoutsMutationVariables = Exact<{
  dashboardId: Scalars['ID']['input'];
  layouts: Array<LayoutInput> | LayoutInput;
}>;


export type UpdateWidgetLayoutsMutation = { __typename?: 'Mutation', UpdateWidgetLayouts: { __typename?: 'StatusResponse', Status: boolean } };

export type UpdateWidgetSnapshotMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  snapshot: SnapshotInput;
}>;


export type UpdateWidgetSnapshotMutation = { __typename?: 'Mutation', UpdateWidgetSnapshot: { __typename?: 'StatusResponse', Status: boolean } };

export type UpdateWidgetMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: UpdateWidgetInput;
}>;


export type UpdateWidgetMutation = { __typename?: 'Mutation', UpdateWidget: { __typename?: 'DashboardWidget', ID: string, Type: string, Title: string, Description?: string | null, Layout: string, Query?: string | null, QueryContext?: string | null, Visualization?: string | null, Snapshot?: string | null, SortOrder: number } };

export type GetColumnsBatchQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnits: Array<Scalars['String']['input']> | Scalars['String']['input'];
}>;


export type GetColumnsBatchQuery = { __typename?: 'Query', ColumnsBatch: Array<{ __typename?: 'StorageUnitColumns', StorageUnit: string, Columns: Array<{ __typename?: 'Column', Type: string, Name: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> }> };

export type GetColumnsQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
}>;


export type GetColumnsQuery = { __typename?: 'Query', Columns: Array<{ __typename?: 'Column', Type: string, Name: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> };

export type GetDatabaseMetadataQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDatabaseMetadataQuery = { __typename?: 'Query', DatabaseMetadata?: { __typename?: 'DatabaseMetadata', databaseType: string, operators: Array<string>, systemSchemas: Array<string>, typeDefinitions: Array<{ __typename?: 'TypeDefinition', id: string, label: string, hasLength: boolean, hasPrecision: boolean, defaultLength?: number | null, defaultPrecision?: number | null, category: TypeCategory }>, aliasMap: Array<{ __typename?: 'Record', Key: string, Value: string }>, capabilities: { __typename?: 'Capabilities', supportsScratchpad: boolean, supportsChat: boolean, supportsGraph: boolean, supportsSchema: boolean, supportsDatabaseSwitch: boolean, supportsModifiers: boolean } } | null };

export type GetDatabaseQueryVariables = Exact<{
  type: Scalars['String']['input'];
}>;


export type GetDatabaseQuery = { __typename?: 'Query', Database: Array<string> };

export type GetDashboardsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDashboardsQuery = { __typename?: 'Query', GetDashboards: Array<{ __typename?: 'Dashboard', ID: string, Name: string, Description?: string | null, RefreshRule: string, CreatedAt: string, UpdatedAt: string, Widgets: Array<{ __typename?: 'DashboardWidget', ID: string, Type: string, Title: string, Description?: string | null, Layout: string, Query?: string | null, QueryContext?: string | null, Visualization?: string | null, Snapshot?: string | null, SortOrder: number }> }> };

export type RawExecuteQueryVariables = Exact<{
  query: Scalars['String']['input'];
}>;


export type RawExecuteQuery = { __typename?: 'Query', RawExecute: { __typename?: 'RowsResult', Rows: Array<Array<string>>, TotalCount: number, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } };

export type GetStorageUnitRowsQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  where?: InputMaybe<WhereCondition>;
  sort?: InputMaybe<Array<SortCondition> | SortCondition>;
  pageSize: Scalars['Int']['input'];
  pageOffset: Scalars['Int']['input'];
}>;


export type GetStorageUnitRowsQuery = { __typename?: 'Query', Row: { __typename?: 'RowsResult', Rows: Array<Array<string>>, DisableUpdate: boolean, TotalCount: number, Columns: Array<{ __typename?: 'Column', Type: string, Name: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> } };

export type GetSchemaQueryVariables = Exact<{ [key: string]: never; }>;


export type GetSchemaQuery = { __typename?: 'Query', Schema: Array<string> };

export type SettingsConfigQueryVariables = Exact<{ [key: string]: never; }>;


export type SettingsConfigQuery = { __typename?: 'Query', SettingsConfig: { __typename?: 'SettingsConfig', DisableCredentialForm: boolean, StandaloneLoginEnabled: boolean } };

export type GetStorageUnitsQueryVariables = Exact<{
  schema: Scalars['String']['input'];
}>;


export type GetStorageUnitsQuery = { __typename?: 'Query', StorageUnit: Array<{ __typename?: 'StorageUnit', Name: string, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };


export const AddRowDocument = gql`
    mutation AddRow($schema: String!, $storageUnit: String!, $values: [RecordInput!]!) {
  AddRow(schema: $schema, storageUnit: $storageUnit, values: $values) {
    Status
  }
}
    `;
export type AddRowMutationFn = Apollo.MutationFunction<AddRowMutation, AddRowMutationVariables>;

/**
 * __useAddRowMutation__
 *
 * To run a mutation, you first call `useAddRowMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useAddRowMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [addRowMutation, { data, loading, error }] = useAddRowMutation({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      values: // value for 'values'
 *   },
 * });
 */
export function useAddRowMutation(baseOptions?: Apollo.MutationHookOptions<AddRowMutation, AddRowMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<AddRowMutation, AddRowMutationVariables>(AddRowDocument, options);
      }
export type AddRowMutationHookResult = ReturnType<typeof useAddRowMutation>;
export type AddRowMutationResult = Apollo.MutationResult<AddRowMutation>;
export type AddRowMutationOptions = Apollo.BaseMutationOptions<AddRowMutation, AddRowMutationVariables>;
export const AddStorageUnitDocument = gql`
    mutation AddStorageUnit($schema: String!, $storageUnit: String!, $fields: [RecordInput!]!) {
  AddStorageUnit(schema: $schema, storageUnit: $storageUnit, fields: $fields) {
    Status
  }
}
    `;
export type AddStorageUnitMutationFn = Apollo.MutationFunction<AddStorageUnitMutation, AddStorageUnitMutationVariables>;

/**
 * __useAddStorageUnitMutation__
 *
 * To run a mutation, you first call `useAddStorageUnitMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useAddStorageUnitMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [addStorageUnitMutation, { data, loading, error }] = useAddStorageUnitMutation({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      fields: // value for 'fields'
 *   },
 * });
 */
export function useAddStorageUnitMutation(baseOptions?: Apollo.MutationHookOptions<AddStorageUnitMutation, AddStorageUnitMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<AddStorageUnitMutation, AddStorageUnitMutationVariables>(AddStorageUnitDocument, options);
      }
export type AddStorageUnitMutationHookResult = ReturnType<typeof useAddStorageUnitMutation>;
export type AddStorageUnitMutationResult = Apollo.MutationResult<AddStorageUnitMutation>;
export type AddStorageUnitMutationOptions = Apollo.BaseMutationOptions<AddStorageUnitMutation, AddStorageUnitMutationVariables>;
export const AddWidgetDocument = gql`
    mutation AddWidget($dashboardId: ID!, $input: WidgetInput!) {
  AddWidget(dashboardId: $dashboardId, input: $input) {
    ID
    Type
    Title
    Description
    Layout
    Query
    QueryContext
    Visualization
    Snapshot
    SortOrder
  }
}
    `;
export type AddWidgetMutationFn = Apollo.MutationFunction<AddWidgetMutation, AddWidgetMutationVariables>;

/**
 * __useAddWidgetMutation__
 *
 * To run a mutation, you first call `useAddWidgetMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useAddWidgetMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [addWidgetMutation, { data, loading, error }] = useAddWidgetMutation({
 *   variables: {
 *      dashboardId: // value for 'dashboardId'
 *      input: // value for 'input'
 *   },
 * });
 */
export function useAddWidgetMutation(baseOptions?: Apollo.MutationHookOptions<AddWidgetMutation, AddWidgetMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<AddWidgetMutation, AddWidgetMutationVariables>(AddWidgetDocument, options);
      }
export type AddWidgetMutationHookResult = ReturnType<typeof useAddWidgetMutation>;
export type AddWidgetMutationResult = Apollo.MutationResult<AddWidgetMutation>;
export type AddWidgetMutationOptions = Apollo.BaseMutationOptions<AddWidgetMutation, AddWidgetMutationVariables>;
export const BootstrapSealosSessionDocument = gql`
    mutation BootstrapSealosSession($input: SealosBootstrapInput!) {
  BootstrapSealosSession(input: $input) {
    sessionToken
    expiresAt
    type
    hostname
    port
    database
    displayName
  }
}
    `;
export type BootstrapSealosSessionMutationFn = Apollo.MutationFunction<BootstrapSealosSessionMutation, BootstrapSealosSessionMutationVariables>;

/**
 * __useBootstrapSealosSessionMutation__
 *
 * To run a mutation, you first call `useBootstrapSealosSessionMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useBootstrapSealosSessionMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [bootstrapSealosSessionMutation, { data, loading, error }] = useBootstrapSealosSessionMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useBootstrapSealosSessionMutation(baseOptions?: Apollo.MutationHookOptions<BootstrapSealosSessionMutation, BootstrapSealosSessionMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<BootstrapSealosSessionMutation, BootstrapSealosSessionMutationVariables>(BootstrapSealosSessionDocument, options);
      }
export type BootstrapSealosSessionMutationHookResult = ReturnType<typeof useBootstrapSealosSessionMutation>;
export type BootstrapSealosSessionMutationResult = Apollo.MutationResult<BootstrapSealosSessionMutation>;
export type BootstrapSealosSessionMutationOptions = Apollo.BaseMutationOptions<BootstrapSealosSessionMutation, BootstrapSealosSessionMutationVariables>;
export const CreateDashboardDocument = gql`
    mutation CreateDashboard($name: String!, $description: String, $refreshRule: String!) {
  CreateDashboard(
    name: $name
    description: $description
    refreshRule: $refreshRule
  ) {
    ID
    Name
    Description
    RefreshRule
    CreatedAt
    UpdatedAt
    Widgets {
      ID
      Type
      Title
      Description
      Layout
      Query
      QueryContext
      Visualization
      Snapshot
      SortOrder
    }
  }
}
    `;
export type CreateDashboardMutationFn = Apollo.MutationFunction<CreateDashboardMutation, CreateDashboardMutationVariables>;

/**
 * __useCreateDashboardMutation__
 *
 * To run a mutation, you first call `useCreateDashboardMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useCreateDashboardMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [createDashboardMutation, { data, loading, error }] = useCreateDashboardMutation({
 *   variables: {
 *      name: // value for 'name'
 *      description: // value for 'description'
 *      refreshRule: // value for 'refreshRule'
 *   },
 * });
 */
export function useCreateDashboardMutation(baseOptions?: Apollo.MutationHookOptions<CreateDashboardMutation, CreateDashboardMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<CreateDashboardMutation, CreateDashboardMutationVariables>(CreateDashboardDocument, options);
      }
export type CreateDashboardMutationHookResult = ReturnType<typeof useCreateDashboardMutation>;
export type CreateDashboardMutationResult = Apollo.MutationResult<CreateDashboardMutation>;
export type CreateDashboardMutationOptions = Apollo.BaseMutationOptions<CreateDashboardMutation, CreateDashboardMutationVariables>;
export const CreateSqlDataExportDocument = gql`
    mutation CreateSQLDataExport($input: SQLDataExportInput!) {
  CreateSQLDataExport(input: $input) {
    ID
    Filename
    ContentType
    DownloadURL
    Size
  }
}
    `;
export type CreateSqlDataExportMutationFn = Apollo.MutationFunction<CreateSqlDataExportMutation, CreateSqlDataExportMutationVariables>;

/**
 * __useCreateSqlDataExportMutation__
 *
 * To run a mutation, you first call `useCreateSqlDataExportMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useCreateSqlDataExportMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [createSqlDataExportMutation, { data, loading, error }] = useCreateSqlDataExportMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useCreateSqlDataExportMutation(baseOptions?: Apollo.MutationHookOptions<CreateSqlDataExportMutation, CreateSqlDataExportMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<CreateSqlDataExportMutation, CreateSqlDataExportMutationVariables>(CreateSqlDataExportDocument, options);
      }
export type CreateSqlDataExportMutationHookResult = ReturnType<typeof useCreateSqlDataExportMutation>;
export type CreateSqlDataExportMutationResult = Apollo.MutationResult<CreateSqlDataExportMutation>;
export type CreateSqlDataExportMutationOptions = Apollo.BaseMutationOptions<CreateSqlDataExportMutation, CreateSqlDataExportMutationVariables>;
export const CreateStandaloneSessionDocument = gql`
    mutation CreateStandaloneSession($credentials: LoginCredentials!) {
  CreateStandaloneSession(credentials: $credentials) {
    sessionToken
    expiresAt
    type
    hostname
    port
    database
    displayName
  }
}
    `;
export type CreateStandaloneSessionMutationFn = Apollo.MutationFunction<CreateStandaloneSessionMutation, CreateStandaloneSessionMutationVariables>;

/**
 * __useCreateStandaloneSessionMutation__
 *
 * To run a mutation, you first call `useCreateStandaloneSessionMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useCreateStandaloneSessionMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [createStandaloneSessionMutation, { data, loading, error }] = useCreateStandaloneSessionMutation({
 *   variables: {
 *      credentials: // value for 'credentials'
 *   },
 * });
 */
export function useCreateStandaloneSessionMutation(baseOptions?: Apollo.MutationHookOptions<CreateStandaloneSessionMutation, CreateStandaloneSessionMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<CreateStandaloneSessionMutation, CreateStandaloneSessionMutationVariables>(CreateStandaloneSessionDocument, options);
      }
export type CreateStandaloneSessionMutationHookResult = ReturnType<typeof useCreateStandaloneSessionMutation>;
export type CreateStandaloneSessionMutationResult = Apollo.MutationResult<CreateStandaloneSessionMutation>;
export type CreateStandaloneSessionMutationOptions = Apollo.BaseMutationOptions<CreateStandaloneSessionMutation, CreateStandaloneSessionMutationVariables>;
export const DeleteDashboardDocument = gql`
    mutation DeleteDashboard($id: ID!) {
  DeleteDashboard(id: $id) {
    Status
  }
}
    `;
export type DeleteDashboardMutationFn = Apollo.MutationFunction<DeleteDashboardMutation, DeleteDashboardMutationVariables>;

/**
 * __useDeleteDashboardMutation__
 *
 * To run a mutation, you first call `useDeleteDashboardMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useDeleteDashboardMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [deleteDashboardMutation, { data, loading, error }] = useDeleteDashboardMutation({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useDeleteDashboardMutation(baseOptions?: Apollo.MutationHookOptions<DeleteDashboardMutation, DeleteDashboardMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<DeleteDashboardMutation, DeleteDashboardMutationVariables>(DeleteDashboardDocument, options);
      }
export type DeleteDashboardMutationHookResult = ReturnType<typeof useDeleteDashboardMutation>;
export type DeleteDashboardMutationResult = Apollo.MutationResult<DeleteDashboardMutation>;
export type DeleteDashboardMutationOptions = Apollo.BaseMutationOptions<DeleteDashboardMutation, DeleteDashboardMutationVariables>;
export const DeleteRowDocument = gql`
    mutation DeleteRow($schema: String!, $storageUnit: String!, $values: [RecordInput!]!) {
  DeleteRow(schema: $schema, storageUnit: $storageUnit, values: $values) {
    Status
  }
}
    `;
export type DeleteRowMutationFn = Apollo.MutationFunction<DeleteRowMutation, DeleteRowMutationVariables>;

/**
 * __useDeleteRowMutation__
 *
 * To run a mutation, you first call `useDeleteRowMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useDeleteRowMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [deleteRowMutation, { data, loading, error }] = useDeleteRowMutation({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      values: // value for 'values'
 *   },
 * });
 */
export function useDeleteRowMutation(baseOptions?: Apollo.MutationHookOptions<DeleteRowMutation, DeleteRowMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<DeleteRowMutation, DeleteRowMutationVariables>(DeleteRowDocument, options);
      }
export type DeleteRowMutationHookResult = ReturnType<typeof useDeleteRowMutation>;
export type DeleteRowMutationResult = Apollo.MutationResult<DeleteRowMutation>;
export type DeleteRowMutationOptions = Apollo.BaseMutationOptions<DeleteRowMutation, DeleteRowMutationVariables>;
export const DeleteWidgetDocument = gql`
    mutation DeleteWidget($id: ID!) {
  DeleteWidget(id: $id) {
    Status
  }
}
    `;
export type DeleteWidgetMutationFn = Apollo.MutationFunction<DeleteWidgetMutation, DeleteWidgetMutationVariables>;

/**
 * __useDeleteWidgetMutation__
 *
 * To run a mutation, you first call `useDeleteWidgetMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useDeleteWidgetMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [deleteWidgetMutation, { data, loading, error }] = useDeleteWidgetMutation({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useDeleteWidgetMutation(baseOptions?: Apollo.MutationHookOptions<DeleteWidgetMutation, DeleteWidgetMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<DeleteWidgetMutation, DeleteWidgetMutationVariables>(DeleteWidgetDocument, options);
      }
export type DeleteWidgetMutationHookResult = ReturnType<typeof useDeleteWidgetMutation>;
export type DeleteWidgetMutationResult = Apollo.MutationResult<DeleteWidgetMutation>;
export type DeleteWidgetMutationOptions = Apollo.BaseMutationOptions<DeleteWidgetMutation, DeleteWidgetMutationVariables>;
export const ExecuteConfirmedSqlDocument = gql`
    mutation ExecuteConfirmedSQL($query: String!, $operationType: String!) {
  ExecuteConfirmedSQL(query: $query, operationType: $operationType) {
    Type
    Text
    Result {
      Columns {
        Type
        Name
      }
      Rows
      TotalCount
    }
    RequiresConfirmation
  }
}
    `;
export type ExecuteConfirmedSqlMutationFn = Apollo.MutationFunction<ExecuteConfirmedSqlMutation, ExecuteConfirmedSqlMutationVariables>;

/**
 * __useExecuteConfirmedSqlMutation__
 *
 * To run a mutation, you first call `useExecuteConfirmedSqlMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useExecuteConfirmedSqlMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [executeConfirmedSqlMutation, { data, loading, error }] = useExecuteConfirmedSqlMutation({
 *   variables: {
 *      query: // value for 'query'
 *      operationType: // value for 'operationType'
 *   },
 * });
 */
export function useExecuteConfirmedSqlMutation(baseOptions?: Apollo.MutationHookOptions<ExecuteConfirmedSqlMutation, ExecuteConfirmedSqlMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ExecuteConfirmedSqlMutation, ExecuteConfirmedSqlMutationVariables>(ExecuteConfirmedSqlDocument, options);
      }
export type ExecuteConfirmedSqlMutationHookResult = ReturnType<typeof useExecuteConfirmedSqlMutation>;
export type ExecuteConfirmedSqlMutationResult = Apollo.MutationResult<ExecuteConfirmedSqlMutation>;
export type ExecuteConfirmedSqlMutationOptions = Apollo.BaseMutationOptions<ExecuteConfirmedSqlMutation, ExecuteConfirmedSqlMutationVariables>;
export const ImportCollectionFileDocument = gql`
    mutation ImportCollectionFile($input: ImportCollectionFileInput!) {
  ImportCollectionFile(input: $input) {
    Status
    ImportedCount
    SkippedCount
    MatchedCount
    ModifiedCount
    UpsertedCount
    Errors {
      Index
      Reason
    }
    Message
    Detail
  }
}
    `;
export type ImportCollectionFileMutationFn = Apollo.MutationFunction<ImportCollectionFileMutation, ImportCollectionFileMutationVariables>;

/**
 * __useImportCollectionFileMutation__
 *
 * To run a mutation, you first call `useImportCollectionFileMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useImportCollectionFileMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [importCollectionFileMutation, { data, loading, error }] = useImportCollectionFileMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useImportCollectionFileMutation(baseOptions?: Apollo.MutationHookOptions<ImportCollectionFileMutation, ImportCollectionFileMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ImportCollectionFileMutation, ImportCollectionFileMutationVariables>(ImportCollectionFileDocument, options);
      }
export type ImportCollectionFileMutationHookResult = ReturnType<typeof useImportCollectionFileMutation>;
export type ImportCollectionFileMutationResult = Apollo.MutationResult<ImportCollectionFileMutation>;
export type ImportCollectionFileMutationOptions = Apollo.BaseMutationOptions<ImportCollectionFileMutation, ImportCollectionFileMutationVariables>;
export const ImportCollectionPreviewDocument = gql`
    mutation ImportCollectionPreview($input: ImportCollectionPreviewInput!) {
  ImportCollectionPreview(input: $input) {
    Format
    Sheet
    Sheets
    Columns
    Rows
    Documents
    Count
    Truncated
    ValidationError
  }
}
    `;
export type ImportCollectionPreviewMutationFn = Apollo.MutationFunction<ImportCollectionPreviewMutation, ImportCollectionPreviewMutationVariables>;

/**
 * __useImportCollectionPreviewMutation__
 *
 * To run a mutation, you first call `useImportCollectionPreviewMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useImportCollectionPreviewMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [importCollectionPreviewMutation, { data, loading, error }] = useImportCollectionPreviewMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useImportCollectionPreviewMutation(baseOptions?: Apollo.MutationHookOptions<ImportCollectionPreviewMutation, ImportCollectionPreviewMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ImportCollectionPreviewMutation, ImportCollectionPreviewMutationVariables>(ImportCollectionPreviewDocument, options);
      }
export type ImportCollectionPreviewMutationHookResult = ReturnType<typeof useImportCollectionPreviewMutation>;
export type ImportCollectionPreviewMutationResult = Apollo.MutationResult<ImportCollectionPreviewMutation>;
export type ImportCollectionPreviewMutationOptions = Apollo.BaseMutationOptions<ImportCollectionPreviewMutation, ImportCollectionPreviewMutationVariables>;
export const ImportPreviewDocument = gql`
    mutation ImportPreview($input: ImportPreviewInput!) {
  ImportPreview(input: $input) {
    Sheet
    Sheets
    Columns
    Rows
    Truncated
    ValidationError
    Mapping {
      SourceColumn
      TargetColumn
    }
    RequiresAllowAutoGenerated
    AutoGeneratedColumns
    NewTable {
      TableName
      Columns {
        SourceColumn
        TargetColumn
        Type
        Nullable
        Primary
        Skip
      }
      Issues {
        Key
        Field
        SourceColumn
      }
    }
  }
}
    `;
export type ImportPreviewMutationFn = Apollo.MutationFunction<ImportPreviewMutation, ImportPreviewMutationVariables>;

/**
 * __useImportPreviewMutation__
 *
 * To run a mutation, you first call `useImportPreviewMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useImportPreviewMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [importPreviewMutation, { data, loading, error }] = useImportPreviewMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useImportPreviewMutation(baseOptions?: Apollo.MutationHookOptions<ImportPreviewMutation, ImportPreviewMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ImportPreviewMutation, ImportPreviewMutationVariables>(ImportPreviewDocument, options);
      }
export type ImportPreviewMutationHookResult = ReturnType<typeof useImportPreviewMutation>;
export type ImportPreviewMutationResult = Apollo.MutationResult<ImportPreviewMutation>;
export type ImportPreviewMutationOptions = Apollo.BaseMutationOptions<ImportPreviewMutation, ImportPreviewMutationVariables>;
export const ImportSqlDocument = gql`
    mutation ImportSQL($input: ImportSQLInput!) {
  ImportSQL(input: $input) {
    Status
    Message
    Detail
  }
}
    `;
export type ImportSqlMutationFn = Apollo.MutationFunction<ImportSqlMutation, ImportSqlMutationVariables>;

/**
 * __useImportSqlMutation__
 *
 * To run a mutation, you first call `useImportSqlMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useImportSqlMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [importSqlMutation, { data, loading, error }] = useImportSqlMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useImportSqlMutation(baseOptions?: Apollo.MutationHookOptions<ImportSqlMutation, ImportSqlMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ImportSqlMutation, ImportSqlMutationVariables>(ImportSqlDocument, options);
      }
export type ImportSqlMutationHookResult = ReturnType<typeof useImportSqlMutation>;
export type ImportSqlMutationResult = Apollo.MutationResult<ImportSqlMutation>;
export type ImportSqlMutationOptions = Apollo.BaseMutationOptions<ImportSqlMutation, ImportSqlMutationVariables>;
export const ImportTableFileDocument = gql`
    mutation ImportTableFile($input: ImportFileInput!) {
  ImportTableFile(input: $input) {
    Status
    Message
    Detail
  }
}
    `;
export type ImportTableFileMutationFn = Apollo.MutationFunction<ImportTableFileMutation, ImportTableFileMutationVariables>;

/**
 * __useImportTableFileMutation__
 *
 * To run a mutation, you first call `useImportTableFileMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useImportTableFileMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [importTableFileMutation, { data, loading, error }] = useImportTableFileMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useImportTableFileMutation(baseOptions?: Apollo.MutationHookOptions<ImportTableFileMutation, ImportTableFileMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ImportTableFileMutation, ImportTableFileMutationVariables>(ImportTableFileDocument, options);
      }
export type ImportTableFileMutationHookResult = ReturnType<typeof useImportTableFileMutation>;
export type ImportTableFileMutationResult = Apollo.MutationResult<ImportTableFileMutation>;
export type ImportTableFileMutationOptions = Apollo.BaseMutationOptions<ImportTableFileMutation, ImportTableFileMutationVariables>;
export const LoginDocument = gql`
    mutation Login($credentials: LoginCredentials!) {
  Login(credentials: $credentials) {
    Status
  }
}
    `;
export type LoginMutationFn = Apollo.MutationFunction<LoginMutation, LoginMutationVariables>;

/**
 * __useLoginMutation__
 *
 * To run a mutation, you first call `useLoginMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useLoginMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [loginMutation, { data, loading, error }] = useLoginMutation({
 *   variables: {
 *      credentials: // value for 'credentials'
 *   },
 * });
 */
export function useLoginMutation(baseOptions?: Apollo.MutationHookOptions<LoginMutation, LoginMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<LoginMutation, LoginMutationVariables>(LoginDocument, options);
      }
export type LoginMutationHookResult = ReturnType<typeof useLoginMutation>;
export type LoginMutationResult = Apollo.MutationResult<LoginMutation>;
export type LoginMutationOptions = Apollo.BaseMutationOptions<LoginMutation, LoginMutationVariables>;
export const ReplaceRowDocument = gql`
    mutation ReplaceRow($schema: String!, $storageUnit: String!, $values: [RecordInput!]!) {
  ReplaceRow(schema: $schema, storageUnit: $storageUnit, values: $values) {
    Status
  }
}
    `;
export type ReplaceRowMutationFn = Apollo.MutationFunction<ReplaceRowMutation, ReplaceRowMutationVariables>;

/**
 * __useReplaceRowMutation__
 *
 * To run a mutation, you first call `useReplaceRowMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useReplaceRowMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [replaceRowMutation, { data, loading, error }] = useReplaceRowMutation({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      values: // value for 'values'
 *   },
 * });
 */
export function useReplaceRowMutation(baseOptions?: Apollo.MutationHookOptions<ReplaceRowMutation, ReplaceRowMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ReplaceRowMutation, ReplaceRowMutationVariables>(ReplaceRowDocument, options);
      }
export type ReplaceRowMutationHookResult = ReturnType<typeof useReplaceRowMutation>;
export type ReplaceRowMutationResult = Apollo.MutationResult<ReplaceRowMutation>;
export type ReplaceRowMutationOptions = Apollo.BaseMutationOptions<ReplaceRowMutation, ReplaceRowMutationVariables>;
export const UpdateDashboardDocument = gql`
    mutation UpdateDashboard($id: ID!, $name: String, $description: String, $refreshRule: String) {
  UpdateDashboard(
    id: $id
    name: $name
    description: $description
    refreshRule: $refreshRule
  ) {
    ID
    Name
    Description
    RefreshRule
    CreatedAt
    UpdatedAt
    Widgets {
      ID
      Type
      Title
      Description
      Layout
      Query
      QueryContext
      Visualization
      Snapshot
      SortOrder
    }
  }
}
    `;
export type UpdateDashboardMutationFn = Apollo.MutationFunction<UpdateDashboardMutation, UpdateDashboardMutationVariables>;

/**
 * __useUpdateDashboardMutation__
 *
 * To run a mutation, you first call `useUpdateDashboardMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateDashboardMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateDashboardMutation, { data, loading, error }] = useUpdateDashboardMutation({
 *   variables: {
 *      id: // value for 'id'
 *      name: // value for 'name'
 *      description: // value for 'description'
 *      refreshRule: // value for 'refreshRule'
 *   },
 * });
 */
export function useUpdateDashboardMutation(baseOptions?: Apollo.MutationHookOptions<UpdateDashboardMutation, UpdateDashboardMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateDashboardMutation, UpdateDashboardMutationVariables>(UpdateDashboardDocument, options);
      }
export type UpdateDashboardMutationHookResult = ReturnType<typeof useUpdateDashboardMutation>;
export type UpdateDashboardMutationResult = Apollo.MutationResult<UpdateDashboardMutation>;
export type UpdateDashboardMutationOptions = Apollo.BaseMutationOptions<UpdateDashboardMutation, UpdateDashboardMutationVariables>;
export const UpdateStorageUnitDocument = gql`
    mutation UpdateStorageUnit($schema: String!, $storageUnit: String!, $values: [RecordInput!]!, $updatedColumns: [String!]!) {
  UpdateStorageUnit(
    schema: $schema
    storageUnit: $storageUnit
    values: $values
    updatedColumns: $updatedColumns
  ) {
    Status
  }
}
    `;
export type UpdateStorageUnitMutationFn = Apollo.MutationFunction<UpdateStorageUnitMutation, UpdateStorageUnitMutationVariables>;

/**
 * __useUpdateStorageUnitMutation__
 *
 * To run a mutation, you first call `useUpdateStorageUnitMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateStorageUnitMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateStorageUnitMutation, { data, loading, error }] = useUpdateStorageUnitMutation({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      values: // value for 'values'
 *      updatedColumns: // value for 'updatedColumns'
 *   },
 * });
 */
export function useUpdateStorageUnitMutation(baseOptions?: Apollo.MutationHookOptions<UpdateStorageUnitMutation, UpdateStorageUnitMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateStorageUnitMutation, UpdateStorageUnitMutationVariables>(UpdateStorageUnitDocument, options);
      }
export type UpdateStorageUnitMutationHookResult = ReturnType<typeof useUpdateStorageUnitMutation>;
export type UpdateStorageUnitMutationResult = Apollo.MutationResult<UpdateStorageUnitMutation>;
export type UpdateStorageUnitMutationOptions = Apollo.BaseMutationOptions<UpdateStorageUnitMutation, UpdateStorageUnitMutationVariables>;
export const UpdateWidgetLayoutsDocument = gql`
    mutation UpdateWidgetLayouts($dashboardId: ID!, $layouts: [LayoutInput!]!) {
  UpdateWidgetLayouts(dashboardId: $dashboardId, layouts: $layouts) {
    Status
  }
}
    `;
export type UpdateWidgetLayoutsMutationFn = Apollo.MutationFunction<UpdateWidgetLayoutsMutation, UpdateWidgetLayoutsMutationVariables>;

/**
 * __useUpdateWidgetLayoutsMutation__
 *
 * To run a mutation, you first call `useUpdateWidgetLayoutsMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateWidgetLayoutsMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateWidgetLayoutsMutation, { data, loading, error }] = useUpdateWidgetLayoutsMutation({
 *   variables: {
 *      dashboardId: // value for 'dashboardId'
 *      layouts: // value for 'layouts'
 *   },
 * });
 */
export function useUpdateWidgetLayoutsMutation(baseOptions?: Apollo.MutationHookOptions<UpdateWidgetLayoutsMutation, UpdateWidgetLayoutsMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateWidgetLayoutsMutation, UpdateWidgetLayoutsMutationVariables>(UpdateWidgetLayoutsDocument, options);
      }
export type UpdateWidgetLayoutsMutationHookResult = ReturnType<typeof useUpdateWidgetLayoutsMutation>;
export type UpdateWidgetLayoutsMutationResult = Apollo.MutationResult<UpdateWidgetLayoutsMutation>;
export type UpdateWidgetLayoutsMutationOptions = Apollo.BaseMutationOptions<UpdateWidgetLayoutsMutation, UpdateWidgetLayoutsMutationVariables>;
export const UpdateWidgetSnapshotDocument = gql`
    mutation UpdateWidgetSnapshot($id: ID!, $snapshot: SnapshotInput!) {
  UpdateWidgetSnapshot(id: $id, snapshot: $snapshot) {
    Status
  }
}
    `;
export type UpdateWidgetSnapshotMutationFn = Apollo.MutationFunction<UpdateWidgetSnapshotMutation, UpdateWidgetSnapshotMutationVariables>;

/**
 * __useUpdateWidgetSnapshotMutation__
 *
 * To run a mutation, you first call `useUpdateWidgetSnapshotMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateWidgetSnapshotMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateWidgetSnapshotMutation, { data, loading, error }] = useUpdateWidgetSnapshotMutation({
 *   variables: {
 *      id: // value for 'id'
 *      snapshot: // value for 'snapshot'
 *   },
 * });
 */
export function useUpdateWidgetSnapshotMutation(baseOptions?: Apollo.MutationHookOptions<UpdateWidgetSnapshotMutation, UpdateWidgetSnapshotMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateWidgetSnapshotMutation, UpdateWidgetSnapshotMutationVariables>(UpdateWidgetSnapshotDocument, options);
      }
export type UpdateWidgetSnapshotMutationHookResult = ReturnType<typeof useUpdateWidgetSnapshotMutation>;
export type UpdateWidgetSnapshotMutationResult = Apollo.MutationResult<UpdateWidgetSnapshotMutation>;
export type UpdateWidgetSnapshotMutationOptions = Apollo.BaseMutationOptions<UpdateWidgetSnapshotMutation, UpdateWidgetSnapshotMutationVariables>;
export const UpdateWidgetDocument = gql`
    mutation UpdateWidget($id: ID!, $input: UpdateWidgetInput!) {
  UpdateWidget(id: $id, input: $input) {
    ID
    Type
    Title
    Description
    Layout
    Query
    QueryContext
    Visualization
    Snapshot
    SortOrder
  }
}
    `;
export type UpdateWidgetMutationFn = Apollo.MutationFunction<UpdateWidgetMutation, UpdateWidgetMutationVariables>;

/**
 * __useUpdateWidgetMutation__
 *
 * To run a mutation, you first call `useUpdateWidgetMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateWidgetMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateWidgetMutation, { data, loading, error }] = useUpdateWidgetMutation({
 *   variables: {
 *      id: // value for 'id'
 *      input: // value for 'input'
 *   },
 * });
 */
export function useUpdateWidgetMutation(baseOptions?: Apollo.MutationHookOptions<UpdateWidgetMutation, UpdateWidgetMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateWidgetMutation, UpdateWidgetMutationVariables>(UpdateWidgetDocument, options);
      }
export type UpdateWidgetMutationHookResult = ReturnType<typeof useUpdateWidgetMutation>;
export type UpdateWidgetMutationResult = Apollo.MutationResult<UpdateWidgetMutation>;
export type UpdateWidgetMutationOptions = Apollo.BaseMutationOptions<UpdateWidgetMutation, UpdateWidgetMutationVariables>;
export const GetColumnsBatchDocument = gql`
    query GetColumnsBatch($schema: String!, $storageUnits: [String!]!) {
  ColumnsBatch(schema: $schema, storageUnits: $storageUnits) {
    StorageUnit
    Columns {
      Type
      Name
      IsPrimary
      IsForeignKey
      ReferencedTable
      ReferencedColumn
      Length
      Precision
      Scale
    }
  }
}
    `;

/**
 * __useGetColumnsBatchQuery__
 *
 * To run a query within a React component, call `useGetColumnsBatchQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetColumnsBatchQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetColumnsBatchQuery({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnits: // value for 'storageUnits'
 *   },
 * });
 */
export function useGetColumnsBatchQuery(baseOptions: Apollo.QueryHookOptions<GetColumnsBatchQuery, GetColumnsBatchQueryVariables> & ({ variables: GetColumnsBatchQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetColumnsBatchQuery, GetColumnsBatchQueryVariables>(GetColumnsBatchDocument, options);
      }
export function useGetColumnsBatchLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetColumnsBatchQuery, GetColumnsBatchQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetColumnsBatchQuery, GetColumnsBatchQueryVariables>(GetColumnsBatchDocument, options);
        }
export function useGetColumnsBatchSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetColumnsBatchQuery, GetColumnsBatchQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetColumnsBatchQuery, GetColumnsBatchQueryVariables>(GetColumnsBatchDocument, options);
        }
export type GetColumnsBatchQueryHookResult = ReturnType<typeof useGetColumnsBatchQuery>;
export type GetColumnsBatchLazyQueryHookResult = ReturnType<typeof useGetColumnsBatchLazyQuery>;
export type GetColumnsBatchSuspenseQueryHookResult = ReturnType<typeof useGetColumnsBatchSuspenseQuery>;
export type GetColumnsBatchQueryResult = Apollo.QueryResult<GetColumnsBatchQuery, GetColumnsBatchQueryVariables>;
export const GetColumnsDocument = gql`
    query GetColumns($schema: String!, $storageUnit: String!) {
  Columns(schema: $schema, storageUnit: $storageUnit) {
    Type
    Name
    IsPrimary
    IsForeignKey
    ReferencedTable
    ReferencedColumn
    Length
    Precision
    Scale
  }
}
    `;

/**
 * __useGetColumnsQuery__
 *
 * To run a query within a React component, call `useGetColumnsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetColumnsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetColumnsQuery({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *   },
 * });
 */
export function useGetColumnsQuery(baseOptions: Apollo.QueryHookOptions<GetColumnsQuery, GetColumnsQueryVariables> & ({ variables: GetColumnsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetColumnsQuery, GetColumnsQueryVariables>(GetColumnsDocument, options);
      }
export function useGetColumnsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetColumnsQuery, GetColumnsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetColumnsQuery, GetColumnsQueryVariables>(GetColumnsDocument, options);
        }
export function useGetColumnsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetColumnsQuery, GetColumnsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetColumnsQuery, GetColumnsQueryVariables>(GetColumnsDocument, options);
        }
export type GetColumnsQueryHookResult = ReturnType<typeof useGetColumnsQuery>;
export type GetColumnsLazyQueryHookResult = ReturnType<typeof useGetColumnsLazyQuery>;
export type GetColumnsSuspenseQueryHookResult = ReturnType<typeof useGetColumnsSuspenseQuery>;
export type GetColumnsQueryResult = Apollo.QueryResult<GetColumnsQuery, GetColumnsQueryVariables>;
export const GetDatabaseMetadataDocument = gql`
    query GetDatabaseMetadata {
  DatabaseMetadata {
    databaseType
    typeDefinitions {
      id
      label
      hasLength
      hasPrecision
      defaultLength
      defaultPrecision
      category
    }
    operators
    aliasMap {
      Key
      Value
    }
    capabilities {
      supportsScratchpad
      supportsChat
      supportsGraph
      supportsSchema
      supportsDatabaseSwitch
      supportsModifiers
    }
    systemSchemas
  }
}
    `;

/**
 * __useGetDatabaseMetadataQuery__
 *
 * To run a query within a React component, call `useGetDatabaseMetadataQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetDatabaseMetadataQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetDatabaseMetadataQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetDatabaseMetadataQuery(baseOptions?: Apollo.QueryHookOptions<GetDatabaseMetadataQuery, GetDatabaseMetadataQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetDatabaseMetadataQuery, GetDatabaseMetadataQueryVariables>(GetDatabaseMetadataDocument, options);
      }
export function useGetDatabaseMetadataLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetDatabaseMetadataQuery, GetDatabaseMetadataQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetDatabaseMetadataQuery, GetDatabaseMetadataQueryVariables>(GetDatabaseMetadataDocument, options);
        }
export function useGetDatabaseMetadataSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetDatabaseMetadataQuery, GetDatabaseMetadataQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetDatabaseMetadataQuery, GetDatabaseMetadataQueryVariables>(GetDatabaseMetadataDocument, options);
        }
export type GetDatabaseMetadataQueryHookResult = ReturnType<typeof useGetDatabaseMetadataQuery>;
export type GetDatabaseMetadataLazyQueryHookResult = ReturnType<typeof useGetDatabaseMetadataLazyQuery>;
export type GetDatabaseMetadataSuspenseQueryHookResult = ReturnType<typeof useGetDatabaseMetadataSuspenseQuery>;
export type GetDatabaseMetadataQueryResult = Apollo.QueryResult<GetDatabaseMetadataQuery, GetDatabaseMetadataQueryVariables>;
export const GetDatabaseDocument = gql`
    query GetDatabase($type: String!) {
  Database(type: $type)
}
    `;

/**
 * __useGetDatabaseQuery__
 *
 * To run a query within a React component, call `useGetDatabaseQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetDatabaseQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetDatabaseQuery({
 *   variables: {
 *      type: // value for 'type'
 *   },
 * });
 */
export function useGetDatabaseQuery(baseOptions: Apollo.QueryHookOptions<GetDatabaseQuery, GetDatabaseQueryVariables> & ({ variables: GetDatabaseQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetDatabaseQuery, GetDatabaseQueryVariables>(GetDatabaseDocument, options);
      }
export function useGetDatabaseLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetDatabaseQuery, GetDatabaseQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetDatabaseQuery, GetDatabaseQueryVariables>(GetDatabaseDocument, options);
        }
export function useGetDatabaseSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetDatabaseQuery, GetDatabaseQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetDatabaseQuery, GetDatabaseQueryVariables>(GetDatabaseDocument, options);
        }
export type GetDatabaseQueryHookResult = ReturnType<typeof useGetDatabaseQuery>;
export type GetDatabaseLazyQueryHookResult = ReturnType<typeof useGetDatabaseLazyQuery>;
export type GetDatabaseSuspenseQueryHookResult = ReturnType<typeof useGetDatabaseSuspenseQuery>;
export type GetDatabaseQueryResult = Apollo.QueryResult<GetDatabaseQuery, GetDatabaseQueryVariables>;
export const GetDashboardsDocument = gql`
    query GetDashboards {
  GetDashboards {
    ID
    Name
    Description
    RefreshRule
    CreatedAt
    UpdatedAt
    Widgets {
      ID
      Type
      Title
      Description
      Layout
      Query
      QueryContext
      Visualization
      Snapshot
      SortOrder
    }
  }
}
    `;

/**
 * __useGetDashboardsQuery__
 *
 * To run a query within a React component, call `useGetDashboardsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetDashboardsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetDashboardsQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetDashboardsQuery(baseOptions?: Apollo.QueryHookOptions<GetDashboardsQuery, GetDashboardsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetDashboardsQuery, GetDashboardsQueryVariables>(GetDashboardsDocument, options);
      }
export function useGetDashboardsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetDashboardsQuery, GetDashboardsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetDashboardsQuery, GetDashboardsQueryVariables>(GetDashboardsDocument, options);
        }
export function useGetDashboardsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetDashboardsQuery, GetDashboardsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetDashboardsQuery, GetDashboardsQueryVariables>(GetDashboardsDocument, options);
        }
export type GetDashboardsQueryHookResult = ReturnType<typeof useGetDashboardsQuery>;
export type GetDashboardsLazyQueryHookResult = ReturnType<typeof useGetDashboardsLazyQuery>;
export type GetDashboardsSuspenseQueryHookResult = ReturnType<typeof useGetDashboardsSuspenseQuery>;
export type GetDashboardsQueryResult = Apollo.QueryResult<GetDashboardsQuery, GetDashboardsQueryVariables>;
export const RawExecuteDocument = gql`
    query RawExecute($query: String!) {
  RawExecute(query: $query) {
    Columns {
      Type
      Name
    }
    Rows
    TotalCount
  }
}
    `;

/**
 * __useRawExecuteQuery__
 *
 * To run a query within a React component, call `useRawExecuteQuery` and pass it any options that fit your needs.
 * When your component renders, `useRawExecuteQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useRawExecuteQuery({
 *   variables: {
 *      query: // value for 'query'
 *   },
 * });
 */
export function useRawExecuteQuery(baseOptions: Apollo.QueryHookOptions<RawExecuteQuery, RawExecuteQueryVariables> & ({ variables: RawExecuteQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<RawExecuteQuery, RawExecuteQueryVariables>(RawExecuteDocument, options);
      }
export function useRawExecuteLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<RawExecuteQuery, RawExecuteQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<RawExecuteQuery, RawExecuteQueryVariables>(RawExecuteDocument, options);
        }
export function useRawExecuteSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<RawExecuteQuery, RawExecuteQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<RawExecuteQuery, RawExecuteQueryVariables>(RawExecuteDocument, options);
        }
export type RawExecuteQueryHookResult = ReturnType<typeof useRawExecuteQuery>;
export type RawExecuteLazyQueryHookResult = ReturnType<typeof useRawExecuteLazyQuery>;
export type RawExecuteSuspenseQueryHookResult = ReturnType<typeof useRawExecuteSuspenseQuery>;
export type RawExecuteQueryResult = Apollo.QueryResult<RawExecuteQuery, RawExecuteQueryVariables>;
export const GetStorageUnitRowsDocument = gql`
    query GetStorageUnitRows($schema: String!, $storageUnit: String!, $where: WhereCondition, $sort: [SortCondition!], $pageSize: Int!, $pageOffset: Int!) {
  Row(
    schema: $schema
    storageUnit: $storageUnit
    where: $where
    sort: $sort
    pageSize: $pageSize
    pageOffset: $pageOffset
  ) {
    Columns {
      Type
      Name
      IsPrimary
      IsForeignKey
      ReferencedTable
      ReferencedColumn
      Length
      Precision
      Scale
    }
    Rows
    DisableUpdate
    TotalCount
  }
}
    `;

/**
 * __useGetStorageUnitRowsQuery__
 *
 * To run a query within a React component, call `useGetStorageUnitRowsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetStorageUnitRowsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetStorageUnitRowsQuery({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      where: // value for 'where'
 *      sort: // value for 'sort'
 *      pageSize: // value for 'pageSize'
 *      pageOffset: // value for 'pageOffset'
 *   },
 * });
 */
export function useGetStorageUnitRowsQuery(baseOptions: Apollo.QueryHookOptions<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables> & ({ variables: GetStorageUnitRowsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>(GetStorageUnitRowsDocument, options);
      }
export function useGetStorageUnitRowsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>(GetStorageUnitRowsDocument, options);
        }
export function useGetStorageUnitRowsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>(GetStorageUnitRowsDocument, options);
        }
export type GetStorageUnitRowsQueryHookResult = ReturnType<typeof useGetStorageUnitRowsQuery>;
export type GetStorageUnitRowsLazyQueryHookResult = ReturnType<typeof useGetStorageUnitRowsLazyQuery>;
export type GetStorageUnitRowsSuspenseQueryHookResult = ReturnType<typeof useGetStorageUnitRowsSuspenseQuery>;
export type GetStorageUnitRowsQueryResult = Apollo.QueryResult<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>;
export const GetSchemaDocument = gql`
    query GetSchema {
  Schema
}
    `;

/**
 * __useGetSchemaQuery__
 *
 * To run a query within a React component, call `useGetSchemaQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetSchemaQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetSchemaQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetSchemaQuery(baseOptions?: Apollo.QueryHookOptions<GetSchemaQuery, GetSchemaQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetSchemaQuery, GetSchemaQueryVariables>(GetSchemaDocument, options);
      }
export function useGetSchemaLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetSchemaQuery, GetSchemaQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetSchemaQuery, GetSchemaQueryVariables>(GetSchemaDocument, options);
        }
export function useGetSchemaSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSchemaQuery, GetSchemaQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetSchemaQuery, GetSchemaQueryVariables>(GetSchemaDocument, options);
        }
export type GetSchemaQueryHookResult = ReturnType<typeof useGetSchemaQuery>;
export type GetSchemaLazyQueryHookResult = ReturnType<typeof useGetSchemaLazyQuery>;
export type GetSchemaSuspenseQueryHookResult = ReturnType<typeof useGetSchemaSuspenseQuery>;
export type GetSchemaQueryResult = Apollo.QueryResult<GetSchemaQuery, GetSchemaQueryVariables>;
export const SettingsConfigDocument = gql`
    query SettingsConfig {
  SettingsConfig {
    DisableCredentialForm
    StandaloneLoginEnabled
  }
}
    `;

/**
 * __useSettingsConfigQuery__
 *
 * To run a query within a React component, call `useSettingsConfigQuery` and pass it any options that fit your needs.
 * When your component renders, `useSettingsConfigQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useSettingsConfigQuery({
 *   variables: {
 *   },
 * });
 */
export function useSettingsConfigQuery(baseOptions?: Apollo.QueryHookOptions<SettingsConfigQuery, SettingsConfigQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<SettingsConfigQuery, SettingsConfigQueryVariables>(SettingsConfigDocument, options);
      }
export function useSettingsConfigLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<SettingsConfigQuery, SettingsConfigQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<SettingsConfigQuery, SettingsConfigQueryVariables>(SettingsConfigDocument, options);
        }
export function useSettingsConfigSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<SettingsConfigQuery, SettingsConfigQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<SettingsConfigQuery, SettingsConfigQueryVariables>(SettingsConfigDocument, options);
        }
export type SettingsConfigQueryHookResult = ReturnType<typeof useSettingsConfigQuery>;
export type SettingsConfigLazyQueryHookResult = ReturnType<typeof useSettingsConfigLazyQuery>;
export type SettingsConfigSuspenseQueryHookResult = ReturnType<typeof useSettingsConfigSuspenseQuery>;
export type SettingsConfigQueryResult = Apollo.QueryResult<SettingsConfigQuery, SettingsConfigQueryVariables>;
export const GetStorageUnitsDocument = gql`
    query GetStorageUnits($schema: String!) {
  StorageUnit(schema: $schema) {
    Name
    Attributes {
      Key
      Value
    }
  }
}
    `;

/**
 * __useGetStorageUnitsQuery__
 *
 * To run a query within a React component, call `useGetStorageUnitsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetStorageUnitsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetStorageUnitsQuery({
 *   variables: {
 *      schema: // value for 'schema'
 *   },
 * });
 */
export function useGetStorageUnitsQuery(baseOptions: Apollo.QueryHookOptions<GetStorageUnitsQuery, GetStorageUnitsQueryVariables> & ({ variables: GetStorageUnitsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>(GetStorageUnitsDocument, options);
      }
export function useGetStorageUnitsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>(GetStorageUnitsDocument, options);
        }
export function useGetStorageUnitsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>(GetStorageUnitsDocument, options);
        }
export type GetStorageUnitsQueryHookResult = ReturnType<typeof useGetStorageUnitsQuery>;
export type GetStorageUnitsLazyQueryHookResult = ReturnType<typeof useGetStorageUnitsLazyQuery>;
export type GetStorageUnitsSuspenseQueryHookResult = ReturnType<typeof useGetStorageUnitsSuspenseQuery>;
export type GetStorageUnitsQueryResult = Apollo.QueryResult<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>;