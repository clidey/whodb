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
  AuthMethod: Scalars['String']['output'];
  DBUsername?: Maybe<Scalars['String']['output']>;
  DiscoverDocumentDB: Scalars['Boolean']['output'];
  DiscoverElastiCache: Scalars['Boolean']['output'];
  DiscoverRDS: Scalars['Boolean']['output'];
  DiscoveredCount: Scalars['Int']['output'];
  Error?: Maybe<Scalars['String']['output']>;
  HasCredentials: Scalars['Boolean']['output'];
  Id: Scalars['ID']['output'];
  LastDiscoveryAt?: Maybe<Scalars['String']['output']>;
  Name: Scalars['String']['output'];
  ProfileName?: Maybe<Scalars['String']['output']>;
  ProviderType: CloudProviderType;
  Region: Scalars['String']['output'];
  Status: CloudProviderStatus;
};

export type AwsProviderInput = {
  AccessKeyId?: InputMaybe<Scalars['String']['input']>;
  AuthMethod?: InputMaybe<Scalars['String']['input']>;
  DBUsername?: InputMaybe<Scalars['String']['input']>;
  DiscoverDocumentDB?: InputMaybe<Scalars['Boolean']['input']>;
  DiscoverElastiCache?: InputMaybe<Scalars['Boolean']['input']>;
  DiscoverRDS?: InputMaybe<Scalars['Boolean']['input']>;
  Name: Scalars['String']['input'];
  ProfileName?: InputMaybe<Scalars['String']['input']>;
  Region: Scalars['String']['input'];
  SecretAccessKey?: InputMaybe<Scalars['String']['input']>;
  SessionToken?: InputMaybe<Scalars['String']['input']>;
};

export type AwsRegion = {
  __typename?: 'AWSRegion';
  Description: Scalars['String']['output'];
  Id: Scalars['String']['output'];
};

export type AtomicWhereCondition = {
  ColumnType: Scalars['String']['input'];
  Key: Scalars['String']['input'];
  Operator: Scalars['String']['input'];
  Value: Scalars['String']['input'];
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
  CredentialsRequired = 'CredentialsRequired',
  Disconnected = 'Disconnected',
  Discovering = 'Discovering',
  Error = 'Error'
}

export enum CloudProviderType {
  Aws = 'AWS'
}

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

export type DatabaseMetadata = {
  __typename?: 'DatabaseMetadata';
  aliasMap: Array<Record>;
  databaseType: Scalars['String']['output'];
  operators: Array<Scalars['String']['output']>;
  typeDefinitions: Array<TypeDefinition>;
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

export enum ImportFileFormat {
  Csv = 'CSV',
  Excel = 'EXCEL'
}

export type ImportFileInput = {
  AllowAutoGenerated?: InputMaybe<Scalars['Boolean']['input']>;
  File: Scalars['Upload']['input'];
  Mapping: Array<ImportColumnMapping>;
  Mode: ImportMode;
  Options: ImportFileOptions;
  Schema: Scalars['String']['input'];
  StorageUnit: Scalars['String']['input'];
};

export type ImportFileOptions = {
  Delimiter?: InputMaybe<Scalars['String']['input']>;
  Format: ImportFileFormat;
  HasHeader: Scalars['Boolean']['input'];
  Sheet?: InputMaybe<Scalars['String']['input']>;
};

export enum ImportMode {
  Append = 'APPEND',
  Overwrite = 'OVERWRITE'
}

export type ImportPreview = {
  __typename?: 'ImportPreview';
  AutoGeneratedColumns: Array<Scalars['String']['output']>;
  Columns: Array<Scalars['String']['output']>;
  Mapping?: Maybe<Array<ImportColumnMappingPreview>>;
  RequiresAllowAutoGenerated: Scalars['Boolean']['output'];
  Rows: Array<Array<Scalars['String']['output']>>;
  Sheet?: Maybe<Scalars['String']['output']>;
  Truncated: Scalars['Boolean']['output'];
  ValidationError?: Maybe<Scalars['String']['output']>;
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

export type LocalAwsProfile = {
  __typename?: 'LocalAWSProfile';
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
  DeleteRow: StatusResponse;
  ExecuteConfirmedSQL: AiChatMessage;
  GenerateMockData: MockDataGenerationStatus;
  ImportPreview: ImportPreview;
  ImportSQL: ImportResult;
  ImportTableFile: ImportResult;
  Login: StatusResponse;
  LoginWithProfile: StatusResponse;
  Logout: StatusResponse;
  RefreshCloudProvider: AwsProvider;
  RemoveCloudProvider: StatusResponse;
  TestCloudProvider: CloudProviderStatus;
  UpdateAWSProvider: AwsProvider;
  UpdateSettings: StatusResponse;
  UpdateStorageUnit: StatusResponse;
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


export type MutationDeleteRowArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput>;
};


export type MutationExecuteConfirmedSqlArgs = {
  operationType: Scalars['String']['input'];
  query: Scalars['String']['input'];
};


export type MutationGenerateMockDataArgs = {
  input: MockDataGenerationInput;
};


export type MutationImportPreviewArgs = {
  file: Scalars['Upload']['input'];
  options: ImportFileOptions;
  schema?: InputMaybe<Scalars['String']['input']>;
  storageUnit?: InputMaybe<Scalars['String']['input']>;
  useHeaderMapping?: InputMaybe<Scalars['Boolean']['input']>;
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


export type MutationTestCloudProviderArgs = {
  id: Scalars['ID']['input'];
};


export type MutationUpdateAwsProviderArgs = {
  id: Scalars['ID']['input'];
  input: AwsProviderInput;
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
  DiscoveredConnections: Array<DiscoveredConnection>;
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

export type SslStatus = {
  __typename?: 'SSLStatus';
  IsEnabled: Scalars['Boolean']['output'];
  Mode: Scalars['String']['output'];
};

export type SettingsConfig = {
  __typename?: 'SettingsConfig';
  CloudProvidersEnabled: Scalars['Boolean']['output'];
  DisableCredentialForm: Scalars['Boolean']['output'];
  MetricsEnabled?: Maybe<Scalars['Boolean']['output']>;
};

export type SettingsConfigInput = {
  MetricsEnabled?: InputMaybe<Scalars['String']['input']>;
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

export type GetProfilesQueryVariables = Exact<{ [key: string]: never; }>;


export type GetProfilesQuery = { __typename?: 'Query', Profiles: Array<{ __typename?: 'LoginProfile', Alias?: string | null, Id: string, Type: DatabaseType, Hostname?: string | null, Database?: string | null, IsEnvironmentDefined: boolean, Source: string, SSLConfigured: boolean }> };

export type GetSchemaQueryVariables = Exact<{ [key: string]: never; }>;


export type GetSchemaQuery = { __typename?: 'Query', Schema: Array<string> };

export type GetVersionQueryVariables = Exact<{ [key: string]: never; }>;


export type GetVersionQuery = { __typename?: 'Query', Version: string };

export type GetDatabaseMetadataQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDatabaseMetadataQuery = { __typename?: 'Query', DatabaseMetadata?: { __typename?: 'DatabaseMetadata', databaseType: string, operators: Array<string>, typeDefinitions: Array<{ __typename?: 'TypeDefinition', id: string, label: string, hasLength: boolean, hasPrecision: boolean, defaultLength?: number | null, defaultPrecision?: number | null, category: TypeCategory }>, aliasMap: Array<{ __typename?: 'Record', Key: string, Value: string }> } | null };

export type ExecuteConfirmedSqlMutationVariables = Exact<{
  query: Scalars['String']['input'];
  operationType: Scalars['String']['input'];
}>;


export type ExecuteConfirmedSqlMutation = { __typename?: 'Mutation', ExecuteConfirmedSQL: { __typename?: 'AIChatMessage', Type: string, Text: string, RequiresConfirmation: boolean, Result?: { __typename?: 'RowsResult', Rows: Array<Array<string>>, DisableUpdate: boolean, TotalCount: number, Columns: Array<{ __typename?: 'Column', Type: string, Name: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> } | null } };

export type GetHealthQueryVariables = Exact<{ [key: string]: never; }>;


export type GetHealthQuery = { __typename?: 'Query', Health: { __typename?: 'HealthStatus', Server: string, Database: string } };

export type ImportPreviewMutationVariables = Exact<{
  file: Scalars['Upload']['input'];
  options: ImportFileOptions;
  schema?: InputMaybe<Scalars['String']['input']>;
  storageUnit?: InputMaybe<Scalars['String']['input']>;
  useHeaderMapping?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type ImportPreviewMutation = { __typename?: 'Mutation', ImportPreview: { __typename?: 'ImportPreview', Sheet?: string | null, Columns: Array<string>, Rows: Array<Array<string>>, Truncated: boolean, ValidationError?: string | null, RequiresAllowAutoGenerated: boolean, AutoGeneratedColumns: Array<string>, Mapping?: Array<{ __typename?: 'ImportColumnMappingPreview', SourceColumn: string, TargetColumn: string }> | null } };

export type ImportSqlMutationVariables = Exact<{
  input: ImportSqlInput;
}>;


export type ImportSqlMutation = { __typename?: 'Mutation', ImportSQL: { __typename?: 'ImportResult', Status: boolean, Message: string, Detail?: string | null } };

export type ImportTableFileMutationVariables = Exact<{
  input: ImportFileInput;
}>;


export type ImportTableFileMutation = { __typename?: 'Mutation', ImportTableFile: { __typename?: 'ImportResult', Status: boolean, Message: string, Detail?: string | null } };

export type GetSslStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type GetSslStatusQuery = { __typename?: 'Query', SSLStatus?: { __typename?: 'SSLStatus', IsEnabled: boolean, Mode: string } | null };

export type AnalyzeMockDataDependenciesQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  rowCount: Scalars['Int']['input'];
  fkDensityRatio?: InputMaybe<Scalars['Int']['input']>;
}>;


export type AnalyzeMockDataDependenciesQuery = { __typename?: 'Query', AnalyzeMockDataDependencies: { __typename?: 'MockDataDependencyAnalysis', GenerationOrder: Array<string>, TotalRows: number, Warnings: Array<string>, Error?: string | null, Tables: Array<{ __typename?: 'MockDataTableInfo', Table: string, RowsToGenerate: number, IsBlocked: boolean, UsesExistingData: boolean }> } };

export type MockDataMaxRowCountQueryVariables = Exact<{ [key: string]: never; }>;


export type MockDataMaxRowCountQuery = { __typename?: 'Query', MockDataMaxRowCount: number };

export type GenerateMockDataMutationVariables = Exact<{
  input: MockDataGenerationInput;
}>;


export type GenerateMockDataMutation = { __typename?: 'Mutation', GenerateMockData: { __typename?: 'MockDataGenerationStatus', AmountGenerated: number, Details?: Array<{ __typename?: 'MockDataTableDetail', Table: string, RowsGenerated: number, UsedExistingData: boolean }> | null } };

export type GetDatabaseQueryVariables = Exact<{
  type: Scalars['String']['input'];
}>;


export type GetDatabaseQuery = { __typename?: 'Query', Database: Array<string> };

export type LoginWithProfileMutationVariables = Exact<{
  profile: LoginProfileInput;
}>;


export type LoginWithProfileMutation = { __typename?: 'Mutation', LoginWithProfile: { __typename?: 'StatusResponse', Status: boolean } };

export type LoginMutationVariables = Exact<{
  credentials: LoginCredentials;
}>;


export type LoginMutation = { __typename?: 'Mutation', Login: { __typename?: 'StatusResponse', Status: boolean } };

export type LogoutMutationVariables = Exact<{ [key: string]: never; }>;


export type LogoutMutation = { __typename?: 'Mutation', Logout: { __typename?: 'StatusResponse', Status: boolean } };

export type GetAiProvidersQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAiProvidersQuery = { __typename?: 'Query', AIProviders: Array<{ __typename?: 'AIProvider', Type: string, Name: string, ProviderId: string, IsEnvironmentDefined: boolean, IsGeneric: boolean }> };

export type GetAiChatQueryVariables = Exact<{
  providerId?: InputMaybe<Scalars['String']['input']>;
  modelType: Scalars['String']['input'];
  token?: InputMaybe<Scalars['String']['input']>;
  schema: Scalars['String']['input'];
  previousConversation: Scalars['String']['input'];
  query: Scalars['String']['input'];
  model: Scalars['String']['input'];
}>;


export type GetAiChatQuery = { __typename?: 'Query', AIChat: Array<{ __typename?: 'AIChatMessage', Type: string, Text: string, Result?: { __typename?: 'RowsResult', Rows: Array<Array<string>>, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } | null }> };

export type GetAiModelsQueryVariables = Exact<{
  providerId?: InputMaybe<Scalars['String']['input']>;
  modelType: Scalars['String']['input'];
  token?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetAiModelsQuery = { __typename?: 'Query', AIModel: Array<string> };

export type GetColumnsQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
}>;


export type GetColumnsQuery = { __typename?: 'Query', Columns: Array<{ __typename?: 'Column', Name: string, Type: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> };

export type GetGraphQueryVariables = Exact<{
  schema: Scalars['String']['input'];
}>;


export type GetGraphQuery = { __typename?: 'Query', Graph: Array<{ __typename?: 'GraphUnit', Unit: { __typename?: 'StorageUnit', Name: string, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }, Relations: Array<{ __typename?: 'GraphUnitRelationship', Name: string, Relationship: GraphUnitRelationshipType, SourceColumn?: string | null, TargetColumn?: string | null }> }> };

export type ColumnsQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
}>;


export type ColumnsQuery = { __typename?: 'Query', Columns: Array<{ __typename?: 'Column', Name: string, Type: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> };

export type RawExecuteQueryVariables = Exact<{
  query: Scalars['String']['input'];
}>;


export type RawExecuteQuery = { __typename?: 'Query', RawExecute: { __typename?: 'RowsResult', Rows: Array<Array<string>>, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } };

export type GetCloudProvidersQueryVariables = Exact<{ [key: string]: never; }>;


export type GetCloudProvidersQuery = { __typename?: 'Query', CloudProviders: Array<{ __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, AuthMethod: string, ProfileName?: string | null, HasCredentials: boolean, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null }> };

export type GetCloudProviderQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetCloudProviderQuery = { __typename?: 'Query', CloudProvider?: { __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, AuthMethod: string, ProfileName?: string | null, HasCredentials: boolean, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null } | null };

export type GetDiscoveredConnectionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDiscoveredConnectionsQuery = { __typename?: 'Query', DiscoveredConnections: Array<{ __typename?: 'DiscoveredConnection', Id: string, ProviderType: CloudProviderType, ProviderID: string, Name: string, DatabaseType: string, Region?: string | null, Status: ConnectionStatus, Metadata: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };

export type GetProviderConnectionsQueryVariables = Exact<{
  providerId: Scalars['ID']['input'];
}>;


export type GetProviderConnectionsQuery = { __typename?: 'Query', ProviderConnections: Array<{ __typename?: 'DiscoveredConnection', Id: string, ProviderType: CloudProviderType, ProviderID: string, Name: string, DatabaseType: string, Region?: string | null, Status: ConnectionStatus, Metadata: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };

export type GetLocalAwsProfilesQueryVariables = Exact<{ [key: string]: never; }>;


export type GetLocalAwsProfilesQuery = { __typename?: 'Query', LocalAWSProfiles: Array<{ __typename?: 'LocalAWSProfile', Name: string, Region?: string | null, Source: string, IsDefault: boolean }> };

export type GetAwsRegionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAwsRegionsQuery = { __typename?: 'Query', AWSRegions: Array<{ __typename?: 'AWSRegion', Id: string, Description: string }> };

export type AddAwsProviderMutationVariables = Exact<{
  input: AwsProviderInput;
}>;


export type AddAwsProviderMutation = { __typename?: 'Mutation', AddAWSProvider: { __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, AuthMethod: string, ProfileName?: string | null, HasCredentials: boolean, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null } };

export type UpdateAwsProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: AwsProviderInput;
}>;


export type UpdateAwsProviderMutation = { __typename?: 'Mutation', UpdateAWSProvider: { __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, AuthMethod: string, ProfileName?: string | null, HasCredentials: boolean, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null } };

export type RemoveCloudProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type RemoveCloudProviderMutation = { __typename?: 'Mutation', RemoveCloudProvider: { __typename?: 'StatusResponse', Status: boolean } };

export type TestCloudProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type TestCloudProviderMutation = { __typename?: 'Mutation', TestCloudProvider: CloudProviderStatus };

export type RefreshCloudProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type RefreshCloudProviderMutation = { __typename?: 'Mutation', RefreshCloudProvider: { __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, AuthMethod: string, ProfileName?: string | null, HasCredentials: boolean, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null } };

export type SettingsConfigQueryVariables = Exact<{ [key: string]: never; }>;


export type SettingsConfigQuery = { __typename?: 'Query', SettingsConfig: { __typename?: 'SettingsConfig', MetricsEnabled?: boolean | null, CloudProvidersEnabled: boolean, DisableCredentialForm: boolean } };

export type UpdateSettingsMutationVariables = Exact<{
  newSettings: SettingsConfigInput;
}>;


export type UpdateSettingsMutation = { __typename?: 'Mutation', UpdateSettings: { __typename?: 'StatusResponse', Status: boolean } };

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

export type DeleteRowMutationVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput> | RecordInput;
}>;


export type DeleteRowMutation = { __typename?: 'Mutation', DeleteRow: { __typename?: 'StatusResponse', Status: boolean } };

export type GetColumnsBatchQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnits: Array<Scalars['String']['input']> | Scalars['String']['input'];
}>;


export type GetColumnsBatchQuery = { __typename?: 'Query', ColumnsBatch: Array<{ __typename?: 'StorageUnitColumns', StorageUnit: string, Columns: Array<{ __typename?: 'Column', Name: string, Type: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> }> };

export type GetStorageUnitRowsQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  where?: InputMaybe<WhereCondition>;
  sort?: InputMaybe<Array<SortCondition> | SortCondition>;
  pageSize: Scalars['Int']['input'];
  pageOffset: Scalars['Int']['input'];
}>;


export type GetStorageUnitRowsQuery = { __typename?: 'Query', Row: { __typename?: 'RowsResult', Rows: Array<Array<string>>, DisableUpdate: boolean, TotalCount: number, Columns: Array<{ __typename?: 'Column', Type: string, Name: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> } };

export type GetStorageUnitsQueryVariables = Exact<{
  schema: Scalars['String']['input'];
}>;


export type GetStorageUnitsQuery = { __typename?: 'Query', StorageUnit: Array<{ __typename?: 'StorageUnit', Name: string, IsMockDataGenerationAllowed: boolean, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };

export type UpdateStorageUnitMutationVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput> | RecordInput;
  updatedColumns: Array<Scalars['String']['input']> | Scalars['String']['input'];
}>;


export type UpdateStorageUnitMutation = { __typename?: 'Mutation', UpdateStorageUnit: { __typename?: 'StatusResponse', Status: boolean } };


export const GetProfilesDocument = gql`
    query GetProfiles {
  Profiles {
    Alias
    Id
    Type
    Hostname
    Database
    IsEnvironmentDefined
    Source
    SSLConfigured
  }
}
    `;

/**
 * __useGetProfilesQuery__
 *
 * To run a query within a React component, call `useGetProfilesQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetProfilesQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetProfilesQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetProfilesQuery(baseOptions?: Apollo.QueryHookOptions<GetProfilesQuery, GetProfilesQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetProfilesQuery, GetProfilesQueryVariables>(GetProfilesDocument, options);
      }
export function useGetProfilesLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetProfilesQuery, GetProfilesQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetProfilesQuery, GetProfilesQueryVariables>(GetProfilesDocument, options);
        }
export function useGetProfilesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetProfilesQuery, GetProfilesQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetProfilesQuery, GetProfilesQueryVariables>(GetProfilesDocument, options);
        }
export type GetProfilesQueryHookResult = ReturnType<typeof useGetProfilesQuery>;
export type GetProfilesLazyQueryHookResult = ReturnType<typeof useGetProfilesLazyQuery>;
export type GetProfilesSuspenseQueryHookResult = ReturnType<typeof useGetProfilesSuspenseQuery>;
export type GetProfilesQueryResult = Apollo.QueryResult<GetProfilesQuery, GetProfilesQueryVariables>;
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
export const GetVersionDocument = gql`
    query GetVersion {
  Version
}
    `;

/**
 * __useGetVersionQuery__
 *
 * To run a query within a React component, call `useGetVersionQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetVersionQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetVersionQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetVersionQuery(baseOptions?: Apollo.QueryHookOptions<GetVersionQuery, GetVersionQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetVersionQuery, GetVersionQueryVariables>(GetVersionDocument, options);
      }
export function useGetVersionLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetVersionQuery, GetVersionQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetVersionQuery, GetVersionQueryVariables>(GetVersionDocument, options);
        }
export function useGetVersionSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetVersionQuery, GetVersionQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetVersionQuery, GetVersionQueryVariables>(GetVersionDocument, options);
        }
export type GetVersionQueryHookResult = ReturnType<typeof useGetVersionQuery>;
export type GetVersionLazyQueryHookResult = ReturnType<typeof useGetVersionLazyQuery>;
export type GetVersionSuspenseQueryHookResult = ReturnType<typeof useGetVersionSuspenseQuery>;
export type GetVersionQueryResult = Apollo.QueryResult<GetVersionQuery, GetVersionQueryVariables>;
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
export const ExecuteConfirmedSqlDocument = gql`
    mutation ExecuteConfirmedSQL($query: String!, $operationType: String!) {
  ExecuteConfirmedSQL(query: $query, operationType: $operationType) {
    Type
    Text
    Result {
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
export const GetHealthDocument = gql`
    query GetHealth {
  Health {
    Server
    Database
  }
}
    `;

/**
 * __useGetHealthQuery__
 *
 * To run a query within a React component, call `useGetHealthQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetHealthQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetHealthQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetHealthQuery(baseOptions?: Apollo.QueryHookOptions<GetHealthQuery, GetHealthQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetHealthQuery, GetHealthQueryVariables>(GetHealthDocument, options);
      }
export function useGetHealthLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetHealthQuery, GetHealthQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetHealthQuery, GetHealthQueryVariables>(GetHealthDocument, options);
        }
export function useGetHealthSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetHealthQuery, GetHealthQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetHealthQuery, GetHealthQueryVariables>(GetHealthDocument, options);
        }
export type GetHealthQueryHookResult = ReturnType<typeof useGetHealthQuery>;
export type GetHealthLazyQueryHookResult = ReturnType<typeof useGetHealthLazyQuery>;
export type GetHealthSuspenseQueryHookResult = ReturnType<typeof useGetHealthSuspenseQuery>;
export type GetHealthQueryResult = Apollo.QueryResult<GetHealthQuery, GetHealthQueryVariables>;
export const ImportPreviewDocument = gql`
    mutation ImportPreview($file: Upload!, $options: ImportFileOptions!, $schema: String, $storageUnit: String, $useHeaderMapping: Boolean) {
  ImportPreview(
    file: $file
    options: $options
    schema: $schema
    storageUnit: $storageUnit
    useHeaderMapping: $useHeaderMapping
  ) {
    Sheet
    Columns
    Rows
    Truncated
    ValidationError
    RequiresAllowAutoGenerated
    AutoGeneratedColumns
    Mapping {
      SourceColumn
      TargetColumn
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
 *      file: // value for 'file'
 *      options: // value for 'options'
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      useHeaderMapping: // value for 'useHeaderMapping'
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
export const GetSslStatusDocument = gql`
    query GetSSLStatus {
  SSLStatus {
    IsEnabled
    Mode
  }
}
    `;

/**
 * __useGetSslStatusQuery__
 *
 * To run a query within a React component, call `useGetSslStatusQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetSslStatusQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetSslStatusQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetSslStatusQuery(baseOptions?: Apollo.QueryHookOptions<GetSslStatusQuery, GetSslStatusQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetSslStatusQuery, GetSslStatusQueryVariables>(GetSslStatusDocument, options);
      }
export function useGetSslStatusLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetSslStatusQuery, GetSslStatusQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetSslStatusQuery, GetSslStatusQueryVariables>(GetSslStatusDocument, options);
        }
export function useGetSslStatusSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSslStatusQuery, GetSslStatusQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetSslStatusQuery, GetSslStatusQueryVariables>(GetSslStatusDocument, options);
        }
export type GetSslStatusQueryHookResult = ReturnType<typeof useGetSslStatusQuery>;
export type GetSslStatusLazyQueryHookResult = ReturnType<typeof useGetSslStatusLazyQuery>;
export type GetSslStatusSuspenseQueryHookResult = ReturnType<typeof useGetSslStatusSuspenseQuery>;
export type GetSslStatusQueryResult = Apollo.QueryResult<GetSslStatusQuery, GetSslStatusQueryVariables>;
export const AnalyzeMockDataDependenciesDocument = gql`
    query AnalyzeMockDataDependencies($schema: String!, $storageUnit: String!, $rowCount: Int!, $fkDensityRatio: Int) {
  AnalyzeMockDataDependencies(
    schema: $schema
    storageUnit: $storageUnit
    rowCount: $rowCount
    fkDensityRatio: $fkDensityRatio
  ) {
    GenerationOrder
    Tables {
      Table
      RowsToGenerate
      IsBlocked
      UsesExistingData
    }
    TotalRows
    Warnings
    Error
  }
}
    `;

/**
 * __useAnalyzeMockDataDependenciesQuery__
 *
 * To run a query within a React component, call `useAnalyzeMockDataDependenciesQuery` and pass it any options that fit your needs.
 * When your component renders, `useAnalyzeMockDataDependenciesQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useAnalyzeMockDataDependenciesQuery({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      rowCount: // value for 'rowCount'
 *      fkDensityRatio: // value for 'fkDensityRatio'
 *   },
 * });
 */
export function useAnalyzeMockDataDependenciesQuery(baseOptions: Apollo.QueryHookOptions<AnalyzeMockDataDependenciesQuery, AnalyzeMockDataDependenciesQueryVariables> & ({ variables: AnalyzeMockDataDependenciesQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<AnalyzeMockDataDependenciesQuery, AnalyzeMockDataDependenciesQueryVariables>(AnalyzeMockDataDependenciesDocument, options);
      }
export function useAnalyzeMockDataDependenciesLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<AnalyzeMockDataDependenciesQuery, AnalyzeMockDataDependenciesQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<AnalyzeMockDataDependenciesQuery, AnalyzeMockDataDependenciesQueryVariables>(AnalyzeMockDataDependenciesDocument, options);
        }
export function useAnalyzeMockDataDependenciesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<AnalyzeMockDataDependenciesQuery, AnalyzeMockDataDependenciesQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<AnalyzeMockDataDependenciesQuery, AnalyzeMockDataDependenciesQueryVariables>(AnalyzeMockDataDependenciesDocument, options);
        }
export type AnalyzeMockDataDependenciesQueryHookResult = ReturnType<typeof useAnalyzeMockDataDependenciesQuery>;
export type AnalyzeMockDataDependenciesLazyQueryHookResult = ReturnType<typeof useAnalyzeMockDataDependenciesLazyQuery>;
export type AnalyzeMockDataDependenciesSuspenseQueryHookResult = ReturnType<typeof useAnalyzeMockDataDependenciesSuspenseQuery>;
export type AnalyzeMockDataDependenciesQueryResult = Apollo.QueryResult<AnalyzeMockDataDependenciesQuery, AnalyzeMockDataDependenciesQueryVariables>;
export const MockDataMaxRowCountDocument = gql`
    query MockDataMaxRowCount {
  MockDataMaxRowCount
}
    `;

/**
 * __useMockDataMaxRowCountQuery__
 *
 * To run a query within a React component, call `useMockDataMaxRowCountQuery` and pass it any options that fit your needs.
 * When your component renders, `useMockDataMaxRowCountQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useMockDataMaxRowCountQuery({
 *   variables: {
 *   },
 * });
 */
export function useMockDataMaxRowCountQuery(baseOptions?: Apollo.QueryHookOptions<MockDataMaxRowCountQuery, MockDataMaxRowCountQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<MockDataMaxRowCountQuery, MockDataMaxRowCountQueryVariables>(MockDataMaxRowCountDocument, options);
      }
export function useMockDataMaxRowCountLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<MockDataMaxRowCountQuery, MockDataMaxRowCountQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<MockDataMaxRowCountQuery, MockDataMaxRowCountQueryVariables>(MockDataMaxRowCountDocument, options);
        }
export function useMockDataMaxRowCountSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<MockDataMaxRowCountQuery, MockDataMaxRowCountQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<MockDataMaxRowCountQuery, MockDataMaxRowCountQueryVariables>(MockDataMaxRowCountDocument, options);
        }
export type MockDataMaxRowCountQueryHookResult = ReturnType<typeof useMockDataMaxRowCountQuery>;
export type MockDataMaxRowCountLazyQueryHookResult = ReturnType<typeof useMockDataMaxRowCountLazyQuery>;
export type MockDataMaxRowCountSuspenseQueryHookResult = ReturnType<typeof useMockDataMaxRowCountSuspenseQuery>;
export type MockDataMaxRowCountQueryResult = Apollo.QueryResult<MockDataMaxRowCountQuery, MockDataMaxRowCountQueryVariables>;
export const GenerateMockDataDocument = gql`
    mutation GenerateMockData($input: MockDataGenerationInput!) {
  GenerateMockData(input: $input) {
    AmountGenerated
    Details {
      Table
      RowsGenerated
      UsedExistingData
    }
  }
}
    `;
export type GenerateMockDataMutationFn = Apollo.MutationFunction<GenerateMockDataMutation, GenerateMockDataMutationVariables>;

/**
 * __useGenerateMockDataMutation__
 *
 * To run a mutation, you first call `useGenerateMockDataMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useGenerateMockDataMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [generateMockDataMutation, { data, loading, error }] = useGenerateMockDataMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useGenerateMockDataMutation(baseOptions?: Apollo.MutationHookOptions<GenerateMockDataMutation, GenerateMockDataMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<GenerateMockDataMutation, GenerateMockDataMutationVariables>(GenerateMockDataDocument, options);
      }
export type GenerateMockDataMutationHookResult = ReturnType<typeof useGenerateMockDataMutation>;
export type GenerateMockDataMutationResult = Apollo.MutationResult<GenerateMockDataMutation>;
export type GenerateMockDataMutationOptions = Apollo.BaseMutationOptions<GenerateMockDataMutation, GenerateMockDataMutationVariables>;
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
export const LoginWithProfileDocument = gql`
    mutation LoginWithProfile($profile: LoginProfileInput!) {
  LoginWithProfile(profile: $profile) {
    Status
  }
}
    `;
export type LoginWithProfileMutationFn = Apollo.MutationFunction<LoginWithProfileMutation, LoginWithProfileMutationVariables>;

/**
 * __useLoginWithProfileMutation__
 *
 * To run a mutation, you first call `useLoginWithProfileMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useLoginWithProfileMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [loginWithProfileMutation, { data, loading, error }] = useLoginWithProfileMutation({
 *   variables: {
 *      profile: // value for 'profile'
 *   },
 * });
 */
export function useLoginWithProfileMutation(baseOptions?: Apollo.MutationHookOptions<LoginWithProfileMutation, LoginWithProfileMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<LoginWithProfileMutation, LoginWithProfileMutationVariables>(LoginWithProfileDocument, options);
      }
export type LoginWithProfileMutationHookResult = ReturnType<typeof useLoginWithProfileMutation>;
export type LoginWithProfileMutationResult = Apollo.MutationResult<LoginWithProfileMutation>;
export type LoginWithProfileMutationOptions = Apollo.BaseMutationOptions<LoginWithProfileMutation, LoginWithProfileMutationVariables>;
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
export const LogoutDocument = gql`
    mutation Logout {
  Logout {
    Status
  }
}
    `;
export type LogoutMutationFn = Apollo.MutationFunction<LogoutMutation, LogoutMutationVariables>;

/**
 * __useLogoutMutation__
 *
 * To run a mutation, you first call `useLogoutMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useLogoutMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [logoutMutation, { data, loading, error }] = useLogoutMutation({
 *   variables: {
 *   },
 * });
 */
export function useLogoutMutation(baseOptions?: Apollo.MutationHookOptions<LogoutMutation, LogoutMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<LogoutMutation, LogoutMutationVariables>(LogoutDocument, options);
      }
export type LogoutMutationHookResult = ReturnType<typeof useLogoutMutation>;
export type LogoutMutationResult = Apollo.MutationResult<LogoutMutation>;
export type LogoutMutationOptions = Apollo.BaseMutationOptions<LogoutMutation, LogoutMutationVariables>;
export const GetAiProvidersDocument = gql`
    query GetAIProviders {
  AIProviders {
    Type
    Name
    ProviderId
    IsEnvironmentDefined
    IsGeneric
  }
}
    `;

/**
 * __useGetAiProvidersQuery__
 *
 * To run a query within a React component, call `useGetAiProvidersQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAiProvidersQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAiProvidersQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetAiProvidersQuery(baseOptions?: Apollo.QueryHookOptions<GetAiProvidersQuery, GetAiProvidersQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAiProvidersQuery, GetAiProvidersQueryVariables>(GetAiProvidersDocument, options);
      }
export function useGetAiProvidersLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAiProvidersQuery, GetAiProvidersQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAiProvidersQuery, GetAiProvidersQueryVariables>(GetAiProvidersDocument, options);
        }
export function useGetAiProvidersSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAiProvidersQuery, GetAiProvidersQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAiProvidersQuery, GetAiProvidersQueryVariables>(GetAiProvidersDocument, options);
        }
export type GetAiProvidersQueryHookResult = ReturnType<typeof useGetAiProvidersQuery>;
export type GetAiProvidersLazyQueryHookResult = ReturnType<typeof useGetAiProvidersLazyQuery>;
export type GetAiProvidersSuspenseQueryHookResult = ReturnType<typeof useGetAiProvidersSuspenseQuery>;
export type GetAiProvidersQueryResult = Apollo.QueryResult<GetAiProvidersQuery, GetAiProvidersQueryVariables>;
export const GetAiChatDocument = gql`
    query GetAIChat($providerId: String, $modelType: String!, $token: String, $schema: String!, $previousConversation: String!, $query: String!, $model: String!) {
  AIChat(
    providerId: $providerId
    modelType: $modelType
    token: $token
    schema: $schema
    input: {PreviousConversation: $previousConversation, Query: $query, Model: $model}
  ) {
    Type
    Result {
      Columns {
        Type
        Name
      }
      Rows
    }
    Text
  }
}
    `;

/**
 * __useGetAiChatQuery__
 *
 * To run a query within a React component, call `useGetAiChatQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAiChatQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAiChatQuery({
 *   variables: {
 *      providerId: // value for 'providerId'
 *      modelType: // value for 'modelType'
 *      token: // value for 'token'
 *      schema: // value for 'schema'
 *      previousConversation: // value for 'previousConversation'
 *      query: // value for 'query'
 *      model: // value for 'model'
 *   },
 * });
 */
export function useGetAiChatQuery(baseOptions: Apollo.QueryHookOptions<GetAiChatQuery, GetAiChatQueryVariables> & ({ variables: GetAiChatQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAiChatQuery, GetAiChatQueryVariables>(GetAiChatDocument, options);
      }
export function useGetAiChatLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAiChatQuery, GetAiChatQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAiChatQuery, GetAiChatQueryVariables>(GetAiChatDocument, options);
        }
export function useGetAiChatSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAiChatQuery, GetAiChatQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAiChatQuery, GetAiChatQueryVariables>(GetAiChatDocument, options);
        }
export type GetAiChatQueryHookResult = ReturnType<typeof useGetAiChatQuery>;
export type GetAiChatLazyQueryHookResult = ReturnType<typeof useGetAiChatLazyQuery>;
export type GetAiChatSuspenseQueryHookResult = ReturnType<typeof useGetAiChatSuspenseQuery>;
export type GetAiChatQueryResult = Apollo.QueryResult<GetAiChatQuery, GetAiChatQueryVariables>;
export const GetAiModelsDocument = gql`
    query GetAIModels($providerId: String, $modelType: String!, $token: String) {
  AIModel(providerId: $providerId, modelType: $modelType, token: $token)
}
    `;

/**
 * __useGetAiModelsQuery__
 *
 * To run a query within a React component, call `useGetAiModelsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAiModelsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAiModelsQuery({
 *   variables: {
 *      providerId: // value for 'providerId'
 *      modelType: // value for 'modelType'
 *      token: // value for 'token'
 *   },
 * });
 */
export function useGetAiModelsQuery(baseOptions: Apollo.QueryHookOptions<GetAiModelsQuery, GetAiModelsQueryVariables> & ({ variables: GetAiModelsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAiModelsQuery, GetAiModelsQueryVariables>(GetAiModelsDocument, options);
      }
export function useGetAiModelsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAiModelsQuery, GetAiModelsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAiModelsQuery, GetAiModelsQueryVariables>(GetAiModelsDocument, options);
        }
export function useGetAiModelsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAiModelsQuery, GetAiModelsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAiModelsQuery, GetAiModelsQueryVariables>(GetAiModelsDocument, options);
        }
export type GetAiModelsQueryHookResult = ReturnType<typeof useGetAiModelsQuery>;
export type GetAiModelsLazyQueryHookResult = ReturnType<typeof useGetAiModelsLazyQuery>;
export type GetAiModelsSuspenseQueryHookResult = ReturnType<typeof useGetAiModelsSuspenseQuery>;
export type GetAiModelsQueryResult = Apollo.QueryResult<GetAiModelsQuery, GetAiModelsQueryVariables>;
export const GetColumnsDocument = gql`
    query GetColumns($schema: String!, $storageUnit: String!) {
  Columns(schema: $schema, storageUnit: $storageUnit) {
    Name
    Type
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
export const GetGraphDocument = gql`
    query GetGraph($schema: String!) {
  Graph(schema: $schema) {
    Unit {
      Name
      Attributes {
        Key
        Value
      }
    }
    Relations {
      Name
      Relationship
      SourceColumn
      TargetColumn
    }
  }
}
    `;

/**
 * __useGetGraphQuery__
 *
 * To run a query within a React component, call `useGetGraphQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetGraphQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetGraphQuery({
 *   variables: {
 *      schema: // value for 'schema'
 *   },
 * });
 */
export function useGetGraphQuery(baseOptions: Apollo.QueryHookOptions<GetGraphQuery, GetGraphQueryVariables> & ({ variables: GetGraphQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetGraphQuery, GetGraphQueryVariables>(GetGraphDocument, options);
      }
export function useGetGraphLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetGraphQuery, GetGraphQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetGraphQuery, GetGraphQueryVariables>(GetGraphDocument, options);
        }
export function useGetGraphSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetGraphQuery, GetGraphQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetGraphQuery, GetGraphQueryVariables>(GetGraphDocument, options);
        }
export type GetGraphQueryHookResult = ReturnType<typeof useGetGraphQuery>;
export type GetGraphLazyQueryHookResult = ReturnType<typeof useGetGraphLazyQuery>;
export type GetGraphSuspenseQueryHookResult = ReturnType<typeof useGetGraphSuspenseQuery>;
export type GetGraphQueryResult = Apollo.QueryResult<GetGraphQuery, GetGraphQueryVariables>;
export const ColumnsDocument = gql`
    query Columns($schema: String!, $storageUnit: String!) {
  Columns(schema: $schema, storageUnit: $storageUnit) {
    Name
    Type
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
 * __useColumnsQuery__
 *
 * To run a query within a React component, call `useColumnsQuery` and pass it any options that fit your needs.
 * When your component renders, `useColumnsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useColumnsQuery({
 *   variables: {
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *   },
 * });
 */
export function useColumnsQuery(baseOptions: Apollo.QueryHookOptions<ColumnsQuery, ColumnsQueryVariables> & ({ variables: ColumnsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<ColumnsQuery, ColumnsQueryVariables>(ColumnsDocument, options);
      }
export function useColumnsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<ColumnsQuery, ColumnsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<ColumnsQuery, ColumnsQueryVariables>(ColumnsDocument, options);
        }
export function useColumnsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<ColumnsQuery, ColumnsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<ColumnsQuery, ColumnsQueryVariables>(ColumnsDocument, options);
        }
export type ColumnsQueryHookResult = ReturnType<typeof useColumnsQuery>;
export type ColumnsLazyQueryHookResult = ReturnType<typeof useColumnsLazyQuery>;
export type ColumnsSuspenseQueryHookResult = ReturnType<typeof useColumnsSuspenseQuery>;
export type ColumnsQueryResult = Apollo.QueryResult<ColumnsQuery, ColumnsQueryVariables>;
export const RawExecuteDocument = gql`
    query RawExecute($query: String!) {
  RawExecute(query: $query) {
    Columns {
      Type
      Name
    }
    Rows
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
export const GetCloudProvidersDocument = gql`
    query GetCloudProviders {
  CloudProviders {
    Id
    ProviderType
    Name
    Region
    AuthMethod
    ProfileName
    HasCredentials
    DiscoverRDS
    DiscoverElastiCache
    DiscoverDocumentDB
    Status
    LastDiscoveryAt
    DiscoveredCount
    Error
  }
}
    `;

/**
 * __useGetCloudProvidersQuery__
 *
 * To run a query within a React component, call `useGetCloudProvidersQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetCloudProvidersQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetCloudProvidersQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetCloudProvidersQuery(baseOptions?: Apollo.QueryHookOptions<GetCloudProvidersQuery, GetCloudProvidersQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetCloudProvidersQuery, GetCloudProvidersQueryVariables>(GetCloudProvidersDocument, options);
      }
export function useGetCloudProvidersLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetCloudProvidersQuery, GetCloudProvidersQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetCloudProvidersQuery, GetCloudProvidersQueryVariables>(GetCloudProvidersDocument, options);
        }
export function useGetCloudProvidersSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetCloudProvidersQuery, GetCloudProvidersQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetCloudProvidersQuery, GetCloudProvidersQueryVariables>(GetCloudProvidersDocument, options);
        }
export type GetCloudProvidersQueryHookResult = ReturnType<typeof useGetCloudProvidersQuery>;
export type GetCloudProvidersLazyQueryHookResult = ReturnType<typeof useGetCloudProvidersLazyQuery>;
export type GetCloudProvidersSuspenseQueryHookResult = ReturnType<typeof useGetCloudProvidersSuspenseQuery>;
export type GetCloudProvidersQueryResult = Apollo.QueryResult<GetCloudProvidersQuery, GetCloudProvidersQueryVariables>;
export const GetCloudProviderDocument = gql`
    query GetCloudProvider($id: ID!) {
  CloudProvider(id: $id) {
    Id
    ProviderType
    Name
    Region
    AuthMethod
    ProfileName
    HasCredentials
    DiscoverRDS
    DiscoverElastiCache
    DiscoverDocumentDB
    Status
    LastDiscoveryAt
    DiscoveredCount
    Error
  }
}
    `;

/**
 * __useGetCloudProviderQuery__
 *
 * To run a query within a React component, call `useGetCloudProviderQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetCloudProviderQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetCloudProviderQuery({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useGetCloudProviderQuery(baseOptions: Apollo.QueryHookOptions<GetCloudProviderQuery, GetCloudProviderQueryVariables> & ({ variables: GetCloudProviderQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetCloudProviderQuery, GetCloudProviderQueryVariables>(GetCloudProviderDocument, options);
      }
export function useGetCloudProviderLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetCloudProviderQuery, GetCloudProviderQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetCloudProviderQuery, GetCloudProviderQueryVariables>(GetCloudProviderDocument, options);
        }
export function useGetCloudProviderSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetCloudProviderQuery, GetCloudProviderQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetCloudProviderQuery, GetCloudProviderQueryVariables>(GetCloudProviderDocument, options);
        }
export type GetCloudProviderQueryHookResult = ReturnType<typeof useGetCloudProviderQuery>;
export type GetCloudProviderLazyQueryHookResult = ReturnType<typeof useGetCloudProviderLazyQuery>;
export type GetCloudProviderSuspenseQueryHookResult = ReturnType<typeof useGetCloudProviderSuspenseQuery>;
export type GetCloudProviderQueryResult = Apollo.QueryResult<GetCloudProviderQuery, GetCloudProviderQueryVariables>;
export const GetDiscoveredConnectionsDocument = gql`
    query GetDiscoveredConnections {
  DiscoveredConnections {
    Id
    ProviderType
    ProviderID
    Name
    DatabaseType
    Region
    Status
    Metadata {
      Key
      Value
    }
  }
}
    `;

/**
 * __useGetDiscoveredConnectionsQuery__
 *
 * To run a query within a React component, call `useGetDiscoveredConnectionsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetDiscoveredConnectionsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetDiscoveredConnectionsQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetDiscoveredConnectionsQuery(baseOptions?: Apollo.QueryHookOptions<GetDiscoveredConnectionsQuery, GetDiscoveredConnectionsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetDiscoveredConnectionsQuery, GetDiscoveredConnectionsQueryVariables>(GetDiscoveredConnectionsDocument, options);
      }
export function useGetDiscoveredConnectionsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetDiscoveredConnectionsQuery, GetDiscoveredConnectionsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetDiscoveredConnectionsQuery, GetDiscoveredConnectionsQueryVariables>(GetDiscoveredConnectionsDocument, options);
        }
export function useGetDiscoveredConnectionsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetDiscoveredConnectionsQuery, GetDiscoveredConnectionsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetDiscoveredConnectionsQuery, GetDiscoveredConnectionsQueryVariables>(GetDiscoveredConnectionsDocument, options);
        }
export type GetDiscoveredConnectionsQueryHookResult = ReturnType<typeof useGetDiscoveredConnectionsQuery>;
export type GetDiscoveredConnectionsLazyQueryHookResult = ReturnType<typeof useGetDiscoveredConnectionsLazyQuery>;
export type GetDiscoveredConnectionsSuspenseQueryHookResult = ReturnType<typeof useGetDiscoveredConnectionsSuspenseQuery>;
export type GetDiscoveredConnectionsQueryResult = Apollo.QueryResult<GetDiscoveredConnectionsQuery, GetDiscoveredConnectionsQueryVariables>;
export const GetProviderConnectionsDocument = gql`
    query GetProviderConnections($providerId: ID!) {
  ProviderConnections(providerID: $providerId) {
    Id
    ProviderType
    ProviderID
    Name
    DatabaseType
    Region
    Status
    Metadata {
      Key
      Value
    }
  }
}
    `;

/**
 * __useGetProviderConnectionsQuery__
 *
 * To run a query within a React component, call `useGetProviderConnectionsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetProviderConnectionsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetProviderConnectionsQuery({
 *   variables: {
 *      providerId: // value for 'providerId'
 *   },
 * });
 */
export function useGetProviderConnectionsQuery(baseOptions: Apollo.QueryHookOptions<GetProviderConnectionsQuery, GetProviderConnectionsQueryVariables> & ({ variables: GetProviderConnectionsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetProviderConnectionsQuery, GetProviderConnectionsQueryVariables>(GetProviderConnectionsDocument, options);
      }
export function useGetProviderConnectionsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetProviderConnectionsQuery, GetProviderConnectionsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetProviderConnectionsQuery, GetProviderConnectionsQueryVariables>(GetProviderConnectionsDocument, options);
        }
export function useGetProviderConnectionsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetProviderConnectionsQuery, GetProviderConnectionsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetProviderConnectionsQuery, GetProviderConnectionsQueryVariables>(GetProviderConnectionsDocument, options);
        }
export type GetProviderConnectionsQueryHookResult = ReturnType<typeof useGetProviderConnectionsQuery>;
export type GetProviderConnectionsLazyQueryHookResult = ReturnType<typeof useGetProviderConnectionsLazyQuery>;
export type GetProviderConnectionsSuspenseQueryHookResult = ReturnType<typeof useGetProviderConnectionsSuspenseQuery>;
export type GetProviderConnectionsQueryResult = Apollo.QueryResult<GetProviderConnectionsQuery, GetProviderConnectionsQueryVariables>;
export const GetLocalAwsProfilesDocument = gql`
    query GetLocalAWSProfiles {
  LocalAWSProfiles {
    Name
    Region
    Source
    IsDefault
  }
}
    `;

/**
 * __useGetLocalAwsProfilesQuery__
 *
 * To run a query within a React component, call `useGetLocalAwsProfilesQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetLocalAwsProfilesQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetLocalAwsProfilesQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetLocalAwsProfilesQuery(baseOptions?: Apollo.QueryHookOptions<GetLocalAwsProfilesQuery, GetLocalAwsProfilesQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetLocalAwsProfilesQuery, GetLocalAwsProfilesQueryVariables>(GetLocalAwsProfilesDocument, options);
      }
export function useGetLocalAwsProfilesLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetLocalAwsProfilesQuery, GetLocalAwsProfilesQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetLocalAwsProfilesQuery, GetLocalAwsProfilesQueryVariables>(GetLocalAwsProfilesDocument, options);
        }
export function useGetLocalAwsProfilesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetLocalAwsProfilesQuery, GetLocalAwsProfilesQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetLocalAwsProfilesQuery, GetLocalAwsProfilesQueryVariables>(GetLocalAwsProfilesDocument, options);
        }
export type GetLocalAwsProfilesQueryHookResult = ReturnType<typeof useGetLocalAwsProfilesQuery>;
export type GetLocalAwsProfilesLazyQueryHookResult = ReturnType<typeof useGetLocalAwsProfilesLazyQuery>;
export type GetLocalAwsProfilesSuspenseQueryHookResult = ReturnType<typeof useGetLocalAwsProfilesSuspenseQuery>;
export type GetLocalAwsProfilesQueryResult = Apollo.QueryResult<GetLocalAwsProfilesQuery, GetLocalAwsProfilesQueryVariables>;
export const GetAwsRegionsDocument = gql`
    query GetAWSRegions {
  AWSRegions {
    Id
    Description
  }
}
    `;

/**
 * __useGetAwsRegionsQuery__
 *
 * To run a query within a React component, call `useGetAwsRegionsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAwsRegionsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAwsRegionsQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetAwsRegionsQuery(baseOptions?: Apollo.QueryHookOptions<GetAwsRegionsQuery, GetAwsRegionsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAwsRegionsQuery, GetAwsRegionsQueryVariables>(GetAwsRegionsDocument, options);
      }
export function useGetAwsRegionsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAwsRegionsQuery, GetAwsRegionsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAwsRegionsQuery, GetAwsRegionsQueryVariables>(GetAwsRegionsDocument, options);
        }
export function useGetAwsRegionsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAwsRegionsQuery, GetAwsRegionsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAwsRegionsQuery, GetAwsRegionsQueryVariables>(GetAwsRegionsDocument, options);
        }
export type GetAwsRegionsQueryHookResult = ReturnType<typeof useGetAwsRegionsQuery>;
export type GetAwsRegionsLazyQueryHookResult = ReturnType<typeof useGetAwsRegionsLazyQuery>;
export type GetAwsRegionsSuspenseQueryHookResult = ReturnType<typeof useGetAwsRegionsSuspenseQuery>;
export type GetAwsRegionsQueryResult = Apollo.QueryResult<GetAwsRegionsQuery, GetAwsRegionsQueryVariables>;
export const AddAwsProviderDocument = gql`
    mutation AddAWSProvider($input: AWSProviderInput!) {
  AddAWSProvider(input: $input) {
    Id
    ProviderType
    Name
    Region
    AuthMethod
    ProfileName
    HasCredentials
    DiscoverRDS
    DiscoverElastiCache
    DiscoverDocumentDB
    Status
    LastDiscoveryAt
    DiscoveredCount
    Error
  }
}
    `;
export type AddAwsProviderMutationFn = Apollo.MutationFunction<AddAwsProviderMutation, AddAwsProviderMutationVariables>;

/**
 * __useAddAwsProviderMutation__
 *
 * To run a mutation, you first call `useAddAwsProviderMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useAddAwsProviderMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [addAwsProviderMutation, { data, loading, error }] = useAddAwsProviderMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useAddAwsProviderMutation(baseOptions?: Apollo.MutationHookOptions<AddAwsProviderMutation, AddAwsProviderMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<AddAwsProviderMutation, AddAwsProviderMutationVariables>(AddAwsProviderDocument, options);
      }
export type AddAwsProviderMutationHookResult = ReturnType<typeof useAddAwsProviderMutation>;
export type AddAwsProviderMutationResult = Apollo.MutationResult<AddAwsProviderMutation>;
export type AddAwsProviderMutationOptions = Apollo.BaseMutationOptions<AddAwsProviderMutation, AddAwsProviderMutationVariables>;
export const UpdateAwsProviderDocument = gql`
    mutation UpdateAWSProvider($id: ID!, $input: AWSProviderInput!) {
  UpdateAWSProvider(id: $id, input: $input) {
    Id
    ProviderType
    Name
    Region
    AuthMethod
    ProfileName
    HasCredentials
    DiscoverRDS
    DiscoverElastiCache
    DiscoverDocumentDB
    Status
    LastDiscoveryAt
    DiscoveredCount
    Error
  }
}
    `;
export type UpdateAwsProviderMutationFn = Apollo.MutationFunction<UpdateAwsProviderMutation, UpdateAwsProviderMutationVariables>;

/**
 * __useUpdateAwsProviderMutation__
 *
 * To run a mutation, you first call `useUpdateAwsProviderMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateAwsProviderMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateAwsProviderMutation, { data, loading, error }] = useUpdateAwsProviderMutation({
 *   variables: {
 *      id: // value for 'id'
 *      input: // value for 'input'
 *   },
 * });
 */
export function useUpdateAwsProviderMutation(baseOptions?: Apollo.MutationHookOptions<UpdateAwsProviderMutation, UpdateAwsProviderMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateAwsProviderMutation, UpdateAwsProviderMutationVariables>(UpdateAwsProviderDocument, options);
      }
export type UpdateAwsProviderMutationHookResult = ReturnType<typeof useUpdateAwsProviderMutation>;
export type UpdateAwsProviderMutationResult = Apollo.MutationResult<UpdateAwsProviderMutation>;
export type UpdateAwsProviderMutationOptions = Apollo.BaseMutationOptions<UpdateAwsProviderMutation, UpdateAwsProviderMutationVariables>;
export const RemoveCloudProviderDocument = gql`
    mutation RemoveCloudProvider($id: ID!) {
  RemoveCloudProvider(id: $id) {
    Status
  }
}
    `;
export type RemoveCloudProviderMutationFn = Apollo.MutationFunction<RemoveCloudProviderMutation, RemoveCloudProviderMutationVariables>;

/**
 * __useRemoveCloudProviderMutation__
 *
 * To run a mutation, you first call `useRemoveCloudProviderMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useRemoveCloudProviderMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [removeCloudProviderMutation, { data, loading, error }] = useRemoveCloudProviderMutation({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useRemoveCloudProviderMutation(baseOptions?: Apollo.MutationHookOptions<RemoveCloudProviderMutation, RemoveCloudProviderMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<RemoveCloudProviderMutation, RemoveCloudProviderMutationVariables>(RemoveCloudProviderDocument, options);
      }
export type RemoveCloudProviderMutationHookResult = ReturnType<typeof useRemoveCloudProviderMutation>;
export type RemoveCloudProviderMutationResult = Apollo.MutationResult<RemoveCloudProviderMutation>;
export type RemoveCloudProviderMutationOptions = Apollo.BaseMutationOptions<RemoveCloudProviderMutation, RemoveCloudProviderMutationVariables>;
export const TestCloudProviderDocument = gql`
    mutation TestCloudProvider($id: ID!) {
  TestCloudProvider(id: $id)
}
    `;
export type TestCloudProviderMutationFn = Apollo.MutationFunction<TestCloudProviderMutation, TestCloudProviderMutationVariables>;

/**
 * __useTestCloudProviderMutation__
 *
 * To run a mutation, you first call `useTestCloudProviderMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useTestCloudProviderMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [testCloudProviderMutation, { data, loading, error }] = useTestCloudProviderMutation({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useTestCloudProviderMutation(baseOptions?: Apollo.MutationHookOptions<TestCloudProviderMutation, TestCloudProviderMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<TestCloudProviderMutation, TestCloudProviderMutationVariables>(TestCloudProviderDocument, options);
      }
export type TestCloudProviderMutationHookResult = ReturnType<typeof useTestCloudProviderMutation>;
export type TestCloudProviderMutationResult = Apollo.MutationResult<TestCloudProviderMutation>;
export type TestCloudProviderMutationOptions = Apollo.BaseMutationOptions<TestCloudProviderMutation, TestCloudProviderMutationVariables>;
export const RefreshCloudProviderDocument = gql`
    mutation RefreshCloudProvider($id: ID!) {
  RefreshCloudProvider(id: $id) {
    Id
    ProviderType
    Name
    Region
    AuthMethod
    ProfileName
    HasCredentials
    DiscoverRDS
    DiscoverElastiCache
    DiscoverDocumentDB
    Status
    LastDiscoveryAt
    DiscoveredCount
    Error
  }
}
    `;
export type RefreshCloudProviderMutationFn = Apollo.MutationFunction<RefreshCloudProviderMutation, RefreshCloudProviderMutationVariables>;

/**
 * __useRefreshCloudProviderMutation__
 *
 * To run a mutation, you first call `useRefreshCloudProviderMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useRefreshCloudProviderMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [refreshCloudProviderMutation, { data, loading, error }] = useRefreshCloudProviderMutation({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useRefreshCloudProviderMutation(baseOptions?: Apollo.MutationHookOptions<RefreshCloudProviderMutation, RefreshCloudProviderMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<RefreshCloudProviderMutation, RefreshCloudProviderMutationVariables>(RefreshCloudProviderDocument, options);
      }
export type RefreshCloudProviderMutationHookResult = ReturnType<typeof useRefreshCloudProviderMutation>;
export type RefreshCloudProviderMutationResult = Apollo.MutationResult<RefreshCloudProviderMutation>;
export type RefreshCloudProviderMutationOptions = Apollo.BaseMutationOptions<RefreshCloudProviderMutation, RefreshCloudProviderMutationVariables>;
export const SettingsConfigDocument = gql`
    query SettingsConfig {
  SettingsConfig {
    MetricsEnabled
    CloudProvidersEnabled
    DisableCredentialForm
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
export const UpdateSettingsDocument = gql`
    mutation UpdateSettings($newSettings: SettingsConfigInput!) {
  UpdateSettings(newSettings: $newSettings) {
    Status
  }
}
    `;
export type UpdateSettingsMutationFn = Apollo.MutationFunction<UpdateSettingsMutation, UpdateSettingsMutationVariables>;

/**
 * __useUpdateSettingsMutation__
 *
 * To run a mutation, you first call `useUpdateSettingsMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateSettingsMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateSettingsMutation, { data, loading, error }] = useUpdateSettingsMutation({
 *   variables: {
 *      newSettings: // value for 'newSettings'
 *   },
 * });
 */
export function useUpdateSettingsMutation(baseOptions?: Apollo.MutationHookOptions<UpdateSettingsMutation, UpdateSettingsMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateSettingsMutation, UpdateSettingsMutationVariables>(UpdateSettingsDocument, options);
      }
export type UpdateSettingsMutationHookResult = ReturnType<typeof useUpdateSettingsMutation>;
export type UpdateSettingsMutationResult = Apollo.MutationResult<UpdateSettingsMutation>;
export type UpdateSettingsMutationOptions = Apollo.BaseMutationOptions<UpdateSettingsMutation, UpdateSettingsMutationVariables>;
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
export const GetColumnsBatchDocument = gql`
    query GetColumnsBatch($schema: String!, $storageUnits: [String!]!) {
  ColumnsBatch(schema: $schema, storageUnits: $storageUnits) {
    StorageUnit
    Columns {
      Name
      Type
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
export const GetStorageUnitsDocument = gql`
    query GetStorageUnits($schema: String!) {
  StorageUnit(schema: $schema) {
    Name
    Attributes {
      Key
      Value
    }
    IsMockDataGenerationAllowed
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