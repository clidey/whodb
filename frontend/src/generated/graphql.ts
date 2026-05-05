/* eslint-disable */
import type { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
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

export type AzureProvider = CloudProvider & {
  __typename?: 'AzureProvider';
  DiscoverCosmosDB: Scalars['Boolean']['output'];
  DiscoverMySQL: Scalars['Boolean']['output'];
  DiscoverPostgreSQL: Scalars['Boolean']['output'];
  DiscoverRedis: Scalars['Boolean']['output'];
  DiscoveredCount: Scalars['Int']['output'];
  Error?: Maybe<Scalars['String']['output']>;
  Id: Scalars['ID']['output'];
  LastDiscoveryAt?: Maybe<Scalars['String']['output']>;
  Name: Scalars['String']['output'];
  ProviderType: CloudProviderType;
  Region: Scalars['String']['output'];
  ResourceGroup?: Maybe<Scalars['String']['output']>;
  Status: CloudProviderStatus;
  SubscriptionID: Scalars['String']['output'];
  TenantID?: Maybe<Scalars['String']['output']>;
};

export type AzureProviderInput = {
  AuthMethod?: InputMaybe<Scalars['String']['input']>;
  ClientID?: InputMaybe<Scalars['String']['input']>;
  ClientSecret?: InputMaybe<Scalars['String']['input']>;
  DiscoverCosmosDB?: InputMaybe<Scalars['Boolean']['input']>;
  DiscoverMySQL?: InputMaybe<Scalars['Boolean']['input']>;
  DiscoverPostgreSQL?: InputMaybe<Scalars['Boolean']['input']>;
  DiscoverRedis?: InputMaybe<Scalars['Boolean']['input']>;
  Name: Scalars['String']['input'];
  ResourceGroup?: InputMaybe<Scalars['String']['input']>;
  SubscriptionID: Scalars['String']['input'];
  TenantID?: InputMaybe<Scalars['String']['input']>;
};

export type AzureRegion = {
  __typename?: 'AzureRegion';
  DisplayName: Scalars['String']['output'];
  Geography: Scalars['String']['output'];
  Id: Scalars['String']['output'];
};

export type AzureSubscription = {
  __typename?: 'AzureSubscription';
  DisplayName: Scalars['String']['output'];
  Id: Scalars['String']['output'];
  State: Scalars['String']['output'];
  TenantID: Scalars['String']['output'];
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
  Aws = 'AWS',
  Azure = 'Azure',
  Gcp = 'GCP'
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

export enum DataShape {
  Content = 'Content',
  Document = 'Document',
  Graph = 'Graph',
  Metadata = 'Metadata',
  Tabular = 'Tabular'
}

export type DiscoveredConnection = {
  __typename?: 'DiscoveredConnection';
  Id: Scalars['ID']['output'];
  Metadata: Array<Record>;
  Name: Scalars['String']['output'];
  ProviderID: Scalars['String']['output'];
  ProviderType: CloudProviderType;
  Region?: Maybe<Scalars['String']['output']>;
  SourceType: Scalars['String']['output'];
  Status: ConnectionStatus;
};

export type GcpProvider = CloudProvider & {
  __typename?: 'GCPProvider';
  DiscoverAlloyDB: Scalars['Boolean']['output'];
  DiscoverCloudSQL: Scalars['Boolean']['output'];
  DiscoverMemorystore: Scalars['Boolean']['output'];
  DiscoveredCount: Scalars['Int']['output'];
  Error?: Maybe<Scalars['String']['output']>;
  Id: Scalars['ID']['output'];
  LastDiscoveryAt?: Maybe<Scalars['String']['output']>;
  Name: Scalars['String']['output'];
  ProjectID: Scalars['String']['output'];
  ProviderType: CloudProviderType;
  Region: Scalars['String']['output'];
  ServiceAccountKeyPath?: Maybe<Scalars['String']['output']>;
  Status: CloudProviderStatus;
};

export type GcpProviderInput = {
  DiscoverAlloyDB?: InputMaybe<Scalars['Boolean']['input']>;
  DiscoverCloudSQL?: InputMaybe<Scalars['Boolean']['input']>;
  DiscoverMemorystore?: InputMaybe<Scalars['Boolean']['input']>;
  Name: Scalars['String']['input'];
  ProjectID: Scalars['String']['input'];
  Region: Scalars['String']['input'];
  ServiceAccountKeyPath?: InputMaybe<Scalars['String']['input']>;
};

export type GcpRegion = {
  __typename?: 'GCPRegion';
  Description: Scalars['String']['output'];
  Id: Scalars['String']['output'];
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
  Unit: SourceObject;
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
  Ref: SourceObjectRefInput;
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
  AuthType: Scalars['String']['output'];
  IsDefault: Scalars['Boolean']['output'];
  Name: Scalars['String']['output'];
  Region?: Maybe<Scalars['String']['output']>;
  Source: Scalars['String']['output'];
};

export type LocalGcpProject = {
  __typename?: 'LocalGCPProject';
  IsDefault: Scalars['Boolean']['output'];
  Name: Scalars['String']['output'];
  ProjectID: Scalars['String']['output'];
  Source: Scalars['String']['output'];
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
  Ref: SourceObjectRefInput;
  RowCount: Scalars['Int']['input'];
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
  AddAzureProvider: AzureProvider;
  AddGCPProvider: GcpProvider;
  AddSourceRow: StatusResponse;
  CreateSourceObject: StatusResponse;
  DeleteSourceRow: StatusResponse;
  ExecuteConfirmedSQL: AiChatMessage;
  GenerateAzureADToken: Scalars['String']['output'];
  GenerateChatTitle: GenerateChatTitleResponse;
  GenerateCloudSQLIAMAuthToken: Scalars['String']['output'];
  GenerateMockData: MockDataGenerationStatus;
  GenerateRDSAuthToken: Scalars['String']['output'];
  ImportPreview: ImportPreview;
  ImportSQL: ImportResult;
  ImportSourceObjectFile: ImportResult;
  LoginSource: StatusResponse;
  LoginWithSourceProfile: StatusResponse;
  Logout: StatusResponse;
  RefreshAzureProvider: AzureProvider;
  RefreshCloudProvider: CloudProvider;
  RefreshGCPProvider: GcpProvider;
  RemoveCloudProvider: StatusResponse;
  TestAWSCredentials: CloudProviderStatus;
  TestAzureCredentials: CloudProviderStatus;
  TestCloudProvider: CloudProviderStatus;
  TestGCPCredentials: CloudProviderStatus;
  UpdateAWSProvider: AwsProvider;
  UpdateAzureProvider: AzureProvider;
  UpdateGCPProvider: GcpProvider;
  UpdateSettings: StatusResponse;
  UpdateSourceObject: StatusResponse;
};


export type MutationAddAwsProviderArgs = {
  input: AwsProviderInput;
};


export type MutationAddAzureProviderArgs = {
  input: AzureProviderInput;
};


export type MutationAddGcpProviderArgs = {
  input: GcpProviderInput;
};


export type MutationAddSourceRowArgs = {
  ref: SourceObjectRefInput;
  values: Array<RecordInput>;
};


export type MutationCreateSourceObjectArgs = {
  fields: Array<RecordInput>;
  name: Scalars['String']['input'];
  parent?: InputMaybe<SourceObjectRefInput>;
};


export type MutationDeleteSourceRowArgs = {
  ref: SourceObjectRefInput;
  values: Array<RecordInput>;
};


export type MutationExecuteConfirmedSqlArgs = {
  operationType: Scalars['String']['input'];
  query: Scalars['String']['input'];
};


export type MutationGenerateAzureAdTokenArgs = {
  providerID: Scalars['ID']['input'];
  sourceType: Scalars['String']['input'];
};


export type MutationGenerateChatTitleArgs = {
  input: GenerateChatTitleInput;
};


export type MutationGenerateCloudSqliamAuthTokenArgs = {
  providerID: Scalars['ID']['input'];
  username: Scalars['String']['input'];
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


export type MutationImportPreviewArgs = {
  file: Scalars['Upload']['input'];
  options: ImportFileOptions;
  ref?: InputMaybe<SourceObjectRefInput>;
  useHeaderMapping?: InputMaybe<Scalars['Boolean']['input']>;
};


export type MutationImportSqlArgs = {
  input: ImportSqlInput;
};


export type MutationImportSourceObjectFileArgs = {
  input: ImportFileInput;
};


export type MutationLoginSourceArgs = {
  credentials: SourceLoginInput;
};


export type MutationLoginWithSourceProfileArgs = {
  profile: SourceProfileLoginInput;
};


export type MutationRefreshAzureProviderArgs = {
  id: Scalars['ID']['input'];
};


export type MutationRefreshCloudProviderArgs = {
  id: Scalars['ID']['input'];
};


export type MutationRefreshGcpProviderArgs = {
  id: Scalars['ID']['input'];
};


export type MutationRemoveCloudProviderArgs = {
  id: Scalars['ID']['input'];
};


export type MutationTestAwsCredentialsArgs = {
  input: AwsProviderInput;
};


export type MutationTestAzureCredentialsArgs = {
  input: AzureProviderInput;
};


export type MutationTestCloudProviderArgs = {
  id: Scalars['ID']['input'];
};


export type MutationTestGcpCredentialsArgs = {
  input: GcpProviderInput;
};


export type MutationUpdateAwsProviderArgs = {
  id: Scalars['ID']['input'];
  input: AwsProviderInput;
};


export type MutationUpdateAzureProviderArgs = {
  id: Scalars['ID']['input'];
  input: AzureProviderInput;
};


export type MutationUpdateGcpProviderArgs = {
  id: Scalars['ID']['input'];
  input: GcpProviderInput;
};


export type MutationUpdateSettingsArgs = {
  newSettings: SettingsConfigInput;
};


export type MutationUpdateSourceObjectArgs = {
  ref: SourceObjectRefInput;
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
  AzureProvider?: Maybe<AzureProvider>;
  AzureProviders: Array<AzureProvider>;
  AzureRegions: Array<AzureRegion>;
  AzureSubscriptions: Array<AzureSubscription>;
  CloudProvider?: Maybe<CloudProvider>;
  CloudProviders: Array<CloudProvider>;
  DiscoveredConnections: Array<DiscoveredConnection>;
  GCPProvider?: Maybe<GcpProvider>;
  GCPProviders: Array<GcpProvider>;
  GCPRegions: Array<GcpRegion>;
  Health: HealthStatus;
  LocalAWSProfiles: Array<LocalAwsProfile>;
  LocalGCPProjects: Array<LocalGcpProject>;
  MockDataMaxRowCount: Scalars['Int']['output'];
  ProviderConnections: Array<DiscoveredConnection>;
  RunSourceQuery: RowsResult;
  SSLStatus?: Maybe<SslStatus>;
  SettingsConfig: SettingsConfig;
  SourceColumns: Array<Column>;
  SourceColumnsBatch: Array<SourceObjectColumns>;
  SourceContent?: Maybe<SourceContent>;
  SourceFieldOptions: Array<Scalars['String']['output']>;
  SourceGraph: Array<GraphUnit>;
  SourceObject?: Maybe<SourceObject>;
  SourceObjects: Array<SourceObject>;
  SourceProfiles: Array<SourceProfile>;
  SourceQuerySuggestions: Array<SourceQuerySuggestion>;
  SourceRows: RowsResult;
  SourceSessionMetadata?: Maybe<SourceSessionMetadata>;
  SourceTypes: Array<SourceType>;
  UpdateInfo: UpdateInfo;
  Version: Scalars['String']['output'];
};


export type QueryAiChatArgs = {
  input: ChatInput;
  modelType: Scalars['String']['input'];
  providerId?: InputMaybe<Scalars['String']['input']>;
  ref?: InputMaybe<SourceObjectRefInput>;
  token?: InputMaybe<Scalars['String']['input']>;
};


export type QueryAiModelArgs = {
  modelType: Scalars['String']['input'];
  providerId?: InputMaybe<Scalars['String']['input']>;
  token?: InputMaybe<Scalars['String']['input']>;
};


export type QueryAnalyzeMockDataDependenciesArgs = {
  fkDensityRatio?: InputMaybe<Scalars['Int']['input']>;
  ref: SourceObjectRefInput;
  rowCount: Scalars['Int']['input'];
};


export type QueryAzureProviderArgs = {
  id: Scalars['ID']['input'];
};


export type QueryCloudProviderArgs = {
  id: Scalars['ID']['input'];
};


export type QueryGcpProviderArgs = {
  id: Scalars['ID']['input'];
};


export type QueryProviderConnectionsArgs = {
  providerID: Scalars['ID']['input'];
};


export type QueryRunSourceQueryArgs = {
  query: Scalars['String']['input'];
};


export type QuerySourceColumnsArgs = {
  ref: SourceObjectRefInput;
};


export type QuerySourceColumnsBatchArgs = {
  refs: Array<SourceObjectRefInput>;
};


export type QuerySourceContentArgs = {
  ref: SourceObjectRefInput;
};


export type QuerySourceFieldOptionsArgs = {
  fieldKey: Scalars['String']['input'];
  sourceType: Scalars['String']['input'];
  values?: InputMaybe<Array<RecordInput>>;
};


export type QuerySourceGraphArgs = {
  ref?: InputMaybe<SourceObjectRefInput>;
};


export type QuerySourceObjectArgs = {
  ref: SourceObjectRefInput;
};


export type QuerySourceObjectsArgs = {
  kinds?: InputMaybe<Array<SourceObjectKind>>;
  parent?: InputMaybe<SourceObjectRefInput>;
};


export type QuerySourceQuerySuggestionsArgs = {
  ref?: InputMaybe<SourceObjectRefInput>;
};


export type QuerySourceRowsArgs = {
  pageOffset: Scalars['Int']['input'];
  pageSize: Scalars['Int']['input'];
  ref: SourceObjectRefInput;
  sort?: InputMaybe<Array<SortCondition>>;
  where?: InputMaybe<WhereCondition>;
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
  EnableNewUI: Scalars['Boolean']['output'];
  MaxPageSize: Scalars['Int']['output'];
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

export enum SourceAction {
  Browse = 'Browse',
  CreateChild = 'CreateChild',
  Delete = 'Delete',
  DeleteData = 'DeleteData',
  Execute = 'Execute',
  GenerateMockData = 'GenerateMockData',
  ImportData = 'ImportData',
  InsertData = 'InsertData',
  Inspect = 'Inspect',
  UpdateData = 'UpdateData',
  ViewContent = 'ViewContent',
  ViewDefinition = 'ViewDefinition',
  ViewGraph = 'ViewGraph',
  ViewRows = 'ViewRows'
}

export enum SourceCategory {
  Cache = 'Cache',
  Database = 'Database',
  FileStore = 'FileStore',
  ObjectStore = 'ObjectStore',
  Search = 'Search'
}

export type SourceConnectionField = {
  __typename?: 'SourceConnectionField';
  DefaultValue?: Maybe<Scalars['String']['output']>;
  Key: Scalars['String']['output'];
  Kind: SourceConnectionFieldKind;
  LabelKey: Scalars['String']['output'];
  PlaceholderKey?: Maybe<Scalars['String']['output']>;
  Required: Scalars['Boolean']['output'];
  Section: SourceConnectionFieldSection;
  SupportsOptions: Scalars['Boolean']['output'];
};

export enum SourceConnectionFieldKind {
  Boolean = 'Boolean',
  FilePath = 'FilePath',
  Password = 'Password',
  Text = 'Text'
}

export enum SourceConnectionFieldSection {
  Advanced = 'Advanced',
  Primary = 'Primary'
}

export type SourceConnectionTraits = {
  __typename?: 'SourceConnectionTraits';
  HostInputMode: SourceHostInputMode;
  HostInputURLParser: SourceHostInputUrlParser;
  SupportsCustomCAContent: Scalars['Boolean']['output'];
  Transport: SourceConnectionTransport;
};

export enum SourceConnectionTransport {
  Bridge = 'Bridge',
  File = 'File',
  Network = 'Network'
}

export type SourceContent = {
  __typename?: 'SourceContent';
  FileName: Scalars['String']['output'];
  IsBinary: Scalars['Boolean']['output'];
  MIMEType: Scalars['String']['output'];
  ModifiedAt?: Maybe<Scalars['String']['output']>;
  SizeBytes: Scalars['String']['output'];
  Text?: Maybe<Scalars['String']['output']>;
  Truncated: Scalars['Boolean']['output'];
};

export type SourceContract = {
  __typename?: 'SourceContract';
  BrowsePath: Array<SourceObjectKind>;
  DefaultObjectKind: SourceObjectKind;
  GraphScopeKind?: Maybe<SourceObjectKind>;
  Model: SourceModel;
  ObjectTypes: Array<SourceObjectType>;
  RootActions: Array<SourceAction>;
  Surfaces: Array<SourceSurface>;
};

export type SourceDiscoveryAdvancedDefault = {
  __typename?: 'SourceDiscoveryAdvancedDefault';
  Conditions: Array<SourceDiscoveryMetadataCondition>;
  DefaultValue: Scalars['String']['output'];
  Key: Scalars['String']['output'];
  MetadataKey: Scalars['String']['output'];
  ProviderTypes: Array<Scalars['String']['output']>;
  Value: Scalars['String']['output'];
};

export type SourceDiscoveryMetadataCondition = {
  __typename?: 'SourceDiscoveryMetadataCondition';
  Key: Scalars['String']['output'];
  Value: Scalars['String']['output'];
};

export type SourceDiscoveryPrefill = {
  __typename?: 'SourceDiscoveryPrefill';
  AdvancedDefaults: Array<SourceDiscoveryAdvancedDefault>;
};

export enum SourceHostInputMode {
  Hostname = 'Hostname',
  HostnameOrUrl = 'HostnameOrURL',
  None = 'None'
}

export enum SourceHostInputUrlParser {
  MongoSrv = 'MongoSRV',
  None = 'None',
  Postgres = 'Postgres'
}

export type SourceLoginInput = {
  AccessToken?: InputMaybe<Scalars['String']['input']>;
  Id?: InputMaybe<Scalars['String']['input']>;
  SourceType: Scalars['String']['input'];
  Values: Array<RecordInput>;
};

export type SourceMockDataTraits = {
  __typename?: 'SourceMockDataTraits';
  SupportsRelationalDependencies: Scalars['Boolean']['output'];
};

export enum SourceModel {
  Document = 'Document',
  Graph = 'Graph',
  KeyValue = 'KeyValue',
  Object = 'Object',
  Relational = 'Relational',
  Search = 'Search'
}

export type SourceObject = {
  __typename?: 'SourceObject';
  Actions: Array<SourceAction>;
  HasChildren: Scalars['Boolean']['output'];
  Kind: SourceObjectKind;
  Metadata: Array<Record>;
  Name: Scalars['String']['output'];
  Path: Array<Scalars['String']['output']>;
  Ref: SourceObjectRef;
};

export type SourceObjectColumns = {
  __typename?: 'SourceObjectColumns';
  Columns: Array<Column>;
  Ref: SourceObjectRef;
};

export enum SourceObjectKind {
  Collection = 'Collection',
  Database = 'Database',
  Function = 'Function',
  Index = 'Index',
  Item = 'Item',
  Key = 'Key',
  Procedure = 'Procedure',
  Schema = 'Schema',
  Sequence = 'Sequence',
  Table = 'Table',
  Trigger = 'Trigger',
  View = 'View'
}

export type SourceObjectRef = {
  __typename?: 'SourceObjectRef';
  Kind: SourceObjectKind;
  Locator: Scalars['String']['output'];
  Path: Array<Scalars['String']['output']>;
};

export type SourceObjectRefInput = {
  Kind: SourceObjectKind;
  Locator?: InputMaybe<Scalars['String']['input']>;
  Path: Array<Scalars['String']['input']>;
};

export type SourceObjectType = {
  __typename?: 'SourceObjectType';
  Actions: Array<SourceAction>;
  DataShape: DataShape;
  Kind: SourceObjectKind;
  PluralLabel: Scalars['String']['output'];
  SingularLabel: Scalars['String']['output'];
  Views: Array<SourceView>;
};

export type SourcePresentationTraits = {
  __typename?: 'SourcePresentationTraits';
  ProfileLabelStrategy: SourceProfileLabelStrategy;
  SchemaFidelity: SourceSchemaFidelity;
};

export type SourceProfile = {
  __typename?: 'SourceProfile';
  DisplayName: Scalars['String']['output'];
  Id: Scalars['String']['output'];
  IsEnvironmentDefined: Scalars['Boolean']['output'];
  SSLConfigured: Scalars['Boolean']['output'];
  Source: Scalars['String']['output'];
  SourceType: Scalars['String']['output'];
  Values: Array<Record>;
};

export enum SourceProfileLabelStrategy {
  Database = 'Database',
  Default = 'Default',
  Hostname = 'Hostname'
}

export type SourceProfileLoginInput = {
  Id: Scalars['String']['input'];
  Values?: InputMaybe<Array<RecordInput>>;
};

export enum SourceQueryExplainMode {
  Explain = 'Explain',
  ExplainAnalyze = 'ExplainAnalyze',
  ExplainPipeline = 'ExplainPipeline',
  None = 'None'
}

export type SourceQuerySuggestion = {
  __typename?: 'SourceQuerySuggestion';
  category: Scalars['String']['output'];
  description: Scalars['String']['output'];
};

export type SourceQueryTraits = {
  __typename?: 'SourceQueryTraits';
  ExplainMode: SourceQueryExplainMode;
  SupportsAnalyze: Scalars['Boolean']['output'];
};

export type SourceSslMode = {
  __typename?: 'SourceSSLMode';
  aliases: Array<Scalars['String']['output']>;
  value: Scalars['String']['output'];
};

export enum SourceSchemaFidelity {
  Exact = 'Exact',
  Sampled = 'Sampled'
}

export type SourceSessionMetadata = {
  __typename?: 'SourceSessionMetadata';
  AliasMap: Array<Record>;
  Operators: Array<Scalars['String']['output']>;
  QueryLanguages: Array<Scalars['String']['output']>;
  SourceType: Scalars['String']['output'];
  TypeDefinitions: Array<TypeDefinition>;
};

export enum SourceSurface {
  Browser = 'Browser',
  Chat = 'Chat',
  Graph = 'Graph',
  Query = 'Query'
}

export type SourceTraits = {
  __typename?: 'SourceTraits';
  Connection: SourceConnectionTraits;
  MockData: SourceMockDataTraits;
  Presentation: SourcePresentationTraits;
  Query: SourceQueryTraits;
};

export type SourceType = {
  __typename?: 'SourceType';
  Category: SourceCategory;
  ConnectionFields: Array<SourceConnectionField>;
  Connector: Scalars['String']['output'];
  Contract: SourceContract;
  DiscoveryPrefill: SourceDiscoveryPrefill;
  Id: Scalars['String']['output'];
  IsAWSManaged: Scalars['Boolean']['output'];
  Label: Scalars['String']['output'];
  SSLModes: Array<SourceSslMode>;
  Traits: SourceTraits;
};

export enum SourceView {
  Binary = 'Binary',
  Graph = 'Graph',
  Grid = 'Grid',
  Json = 'JSON',
  Metadata = 'Metadata',
  Sql = 'SQL',
  Text = 'Text'
}

export type StatusResponse = {
  __typename?: 'StatusResponse';
  Status: Scalars['Boolean']['output'];
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
  insertFunc?: Maybe<Scalars['String']['output']>;
  label: Scalars['String']['output'];
  tableModel?: Maybe<Scalars['String']['output']>;
};

export type UpdateInfo = {
  __typename?: 'UpdateInfo';
  currentVersion: Scalars['String']['output'];
  latestVersion: Scalars['String']['output'];
  releaseURL: Scalars['String']['output'];
  updateAvailable: Scalars['Boolean']['output'];
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

export type SourceProfilesQueryVariables = Exact<{ [key: string]: never; }>;


export type SourceProfilesQuery = { __typename?: 'Query', SourceProfiles: Array<{ __typename?: 'SourceProfile', Id: string, IsEnvironmentDefined: boolean, Source: string, SSLConfigured: boolean, Alias: string, Type: string, Values: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };

export type GetSchemaQueryVariables = Exact<{
  parent?: InputMaybe<SourceObjectRefInput>;
  kinds?: InputMaybe<Array<SourceObjectKind> | SourceObjectKind>;
}>;


export type GetSchemaQuery = { __typename?: 'Query', Schema: Array<{ __typename?: 'SourceObject', Name: string, Kind: SourceObjectKind, Ref: { __typename?: 'SourceObjectRef', Kind: SourceObjectKind, Locator: string, Path: Array<string> } }> };

export type GetUpdateInfoQueryVariables = Exact<{ [key: string]: never; }>;


export type GetUpdateInfoQuery = { __typename?: 'Query', UpdateInfo: { __typename?: 'UpdateInfo', currentVersion: string, latestVersion: string, updateAvailable: boolean, releaseURL: string } };

export type GetVersionQueryVariables = Exact<{ [key: string]: never; }>;


export type GetVersionQuery = { __typename?: 'Query', Version: string };

export type ExecuteConfirmedSqlMutationVariables = Exact<{
  query: Scalars['String']['input'];
  operationType: Scalars['String']['input'];
}>;


export type ExecuteConfirmedSqlMutation = { __typename?: 'Mutation', ExecuteConfirmedSQL: { __typename?: 'AIChatMessage', Type: string, Text: string, RequiresConfirmation: boolean, Result?: { __typename?: 'RowsResult', Rows: Array<Array<string>>, DisableUpdate: boolean, TotalCount: number, Columns: Array<{ __typename?: 'Column', Type: string, Name: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> } | null } };

export type GenerateChatTitleMutationVariables = Exact<{
  input: GenerateChatTitleInput;
}>;


export type GenerateChatTitleMutation = { __typename?: 'Mutation', GenerateChatTitle: { __typename?: 'GenerateChatTitleResponse', Title: string } };

export type GetHealthQueryVariables = Exact<{ [key: string]: never; }>;


export type GetHealthQuery = { __typename?: 'Query', Health: { __typename?: 'HealthStatus', Server: string, Database: string } };

export type ImportPreviewMutationVariables = Exact<{
  file: Scalars['Upload']['input'];
  options: ImportFileOptions;
  ref?: InputMaybe<SourceObjectRefInput>;
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

export type SourceSessionMetadataQueryVariables = Exact<{ [key: string]: never; }>;


export type SourceSessionMetadataQuery = { __typename?: 'Query', SourceSessionMetadata?: { __typename?: 'SourceSessionMetadata', sourceType: string, queryLanguages: Array<string>, operators: Array<string>, typeDefinitions: Array<{ __typename?: 'TypeDefinition', id: string, label: string, hasLength: boolean, hasPrecision: boolean, defaultLength?: number | null, defaultPrecision?: number | null, category: TypeCategory }>, aliasMap: Array<{ __typename?: 'Record', Key: string, Value: string }> } | null };

export type SourceTypesQueryVariables = Exact<{ [key: string]: never; }>;


export type SourceTypesQuery = { __typename?: 'Query', SourceTypes: Array<{ __typename?: 'SourceType', id: string, label: string, connector: string, category: SourceCategory, isAwsManaged: boolean, traits: { __typename?: 'SourceTraits', connection: { __typename?: 'SourceConnectionTraits', transport: SourceConnectionTransport, hostInputMode: SourceHostInputMode, hostInputUrlParser: SourceHostInputUrlParser, supportsCustomCAContent: boolean }, presentation: { __typename?: 'SourcePresentationTraits', profileLabelStrategy: SourceProfileLabelStrategy, schemaFidelity: SourceSchemaFidelity }, query: { __typename?: 'SourceQueryTraits', supportsAnalyze: boolean, explainMode: SourceQueryExplainMode }, mockData: { __typename?: 'SourceMockDataTraits', supportsRelationalDependencies: boolean } }, connectionFields: Array<{ __typename?: 'SourceConnectionField', Key: string, Kind: SourceConnectionFieldKind, Section: SourceConnectionFieldSection, Required: boolean, LabelKey: string, PlaceholderKey?: string | null, DefaultValue?: string | null, SupportsOptions: boolean }>, contract: { __typename?: 'SourceContract', Model: SourceModel, Surfaces: Array<SourceSurface>, RootActions: Array<SourceAction>, BrowsePath: Array<SourceObjectKind>, DefaultObjectKind: SourceObjectKind, GraphScopeKind?: SourceObjectKind | null, ObjectTypes: Array<{ __typename?: 'SourceObjectType', Kind: SourceObjectKind, DataShape: DataShape, Actions: Array<SourceAction>, Views: Array<SourceView>, SingularLabel: string, PluralLabel: string }> }, discoveryPrefill: { __typename?: 'SourceDiscoveryPrefill', AdvancedDefaults: Array<{ __typename?: 'SourceDiscoveryAdvancedDefault', Key: string, Value: string, MetadataKey: string, DefaultValue: string, ProviderTypes: Array<string>, Conditions: Array<{ __typename?: 'SourceDiscoveryMetadataCondition', Key: string, Value: string }> }> }, sslModes: Array<{ __typename?: 'SourceSSLMode', value: string, aliases: Array<string> }> }> };

export type GetSslStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type GetSslStatusQuery = { __typename?: 'Query', SSLStatus?: { __typename?: 'SSLStatus', IsEnabled: boolean, Mode: string } | null };

export type AnalyzeMockDataDependenciesQueryVariables = Exact<{
  ref: SourceObjectRefInput;
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

export type SourceFieldOptionsQueryVariables = Exact<{
  sourceType: Scalars['String']['input'];
  values?: InputMaybe<Array<RecordInput> | RecordInput>;
}>;


export type SourceFieldOptionsQuery = { __typename?: 'Query', SourceFieldOptions: Array<string> };

export type LoginWithSourceProfileMutationVariables = Exact<{
  profile: SourceProfileLoginInput;
}>;


export type LoginWithSourceProfileMutation = { __typename?: 'Mutation', LoginWithSourceProfile: { __typename?: 'StatusResponse', Status: boolean } };

export type LoginSourceMutationVariables = Exact<{
  credentials: SourceLoginInput;
}>;


export type LoginSourceMutation = { __typename?: 'Mutation', LoginSource: { __typename?: 'StatusResponse', Status: boolean } };

export type LogoutMutationVariables = Exact<{ [key: string]: never; }>;


export type LogoutMutation = { __typename?: 'Mutation', Logout: { __typename?: 'StatusResponse', Status: boolean } };

export type GetAiProvidersQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAiProvidersQuery = { __typename?: 'Query', AIProviders: Array<{ __typename?: 'AIProvider', Type: string, Name: string, ProviderId: string, IsEnvironmentDefined: boolean, IsGeneric: boolean }> };

export type GetAiChatQueryVariables = Exact<{
  providerId?: InputMaybe<Scalars['String']['input']>;
  modelType: Scalars['String']['input'];
  token?: InputMaybe<Scalars['String']['input']>;
  ref?: InputMaybe<SourceObjectRefInput>;
  previousConversation: Scalars['String']['input'];
  query: Scalars['String']['input'];
  model: Scalars['String']['input'];
}>;


export type GetAiChatQuery = { __typename?: 'Query', AIChat: Array<{ __typename?: 'AIChatMessage', Type: string, Text: string, Result?: { __typename?: 'RowsResult', Rows: Array<Array<string>>, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } | null }> };

export type GetDatabaseQuerySuggestionsQueryVariables = Exact<{
  ref?: InputMaybe<SourceObjectRefInput>;
}>;


export type GetDatabaseQuerySuggestionsQuery = { __typename?: 'Query', DatabaseQuerySuggestions: Array<{ __typename?: 'SourceQuerySuggestion', description: string, category: string }> };

export type GetAiModelsQueryVariables = Exact<{
  providerId?: InputMaybe<Scalars['String']['input']>;
  modelType: Scalars['String']['input'];
  token?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetAiModelsQuery = { __typename?: 'Query', AIModel: Array<string> };

export type GetColumnsQueryVariables = Exact<{
  ref: SourceObjectRefInput;
}>;


export type GetColumnsQuery = { __typename?: 'Query', Columns: Array<{ __typename?: 'Column', Name: string, Type: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> };

export type GetGraphQueryVariables = Exact<{
  ref?: InputMaybe<SourceObjectRefInput>;
}>;


export type GetGraphQuery = { __typename?: 'Query', Graph: Array<{ __typename?: 'GraphUnit', Unit: { __typename?: 'SourceObject', Kind: SourceObjectKind, Name: string, Ref: { __typename?: 'SourceObjectRef', Kind: SourceObjectKind, Locator: string, Path: Array<string> }, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }, Relations: Array<{ __typename?: 'GraphUnitRelationship', Name: string, Relationship: GraphUnitRelationshipType, SourceColumn?: string | null, TargetColumn?: string | null }> }> };

export type ColumnsQueryVariables = Exact<{
  ref: SourceObjectRefInput;
}>;


export type ColumnsQuery = { __typename?: 'Query', Columns: Array<{ __typename?: 'Column', Name: string, Type: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> };

export type RawExecuteQueryVariables = Exact<{
  query: Scalars['String']['input'];
}>;


export type RawExecuteQuery = { __typename?: 'Query', RawExecute: { __typename?: 'RowsResult', Rows: Array<Array<string>>, TotalCount: number, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } };

export type GetCloudProvidersQueryVariables = Exact<{ [key: string]: never; }>;


export type GetCloudProvidersQuery = { __typename?: 'Query', CloudProviders: Array<{ __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProfileName?: string | null, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean } | { __typename?: 'AzureProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, SubscriptionID: string, TenantID?: string | null, ResourceGroup?: string | null, DiscoverPostgreSQL: boolean, DiscoverMySQL: boolean, DiscoverRedis: boolean, DiscoverCosmosDB: boolean } | { __typename?: 'GCPProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProjectID: string, ServiceAccountKeyPath?: string | null, DiscoverCloudSQL: boolean, DiscoverAlloyDB: boolean, DiscoverMemorystore: boolean }> };

export type GetCloudProviderQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetCloudProviderQuery = { __typename?: 'Query', CloudProvider?: { __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProfileName?: string | null, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean } | { __typename?: 'AzureProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, SubscriptionID: string, TenantID?: string | null, ResourceGroup?: string | null, DiscoverPostgreSQL: boolean, DiscoverMySQL: boolean, DiscoverRedis: boolean, DiscoverCosmosDB: boolean } | { __typename?: 'GCPProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProjectID: string, ServiceAccountKeyPath?: string | null, DiscoverCloudSQL: boolean, DiscoverAlloyDB: boolean, DiscoverMemorystore: boolean } | null };

export type GetDiscoveredConnectionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDiscoveredConnectionsQuery = { __typename?: 'Query', DiscoveredConnections: Array<{ __typename?: 'DiscoveredConnection', Id: string, ProviderType: CloudProviderType, ProviderID: string, Name: string, Region?: string | null, Status: ConnectionStatus, DatabaseType: string, Metadata: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };

export type GetProviderConnectionsQueryVariables = Exact<{
  providerId: Scalars['ID']['input'];
}>;


export type GetProviderConnectionsQuery = { __typename?: 'Query', ProviderConnections: Array<{ __typename?: 'DiscoveredConnection', Id: string, ProviderType: CloudProviderType, ProviderID: string, Name: string, Region?: string | null, Status: ConnectionStatus, DatabaseType: string, Metadata: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };

export type GetLocalAwsProfilesQueryVariables = Exact<{ [key: string]: never; }>;


export type GetLocalAwsProfilesQuery = { __typename?: 'Query', LocalAWSProfiles: Array<{ __typename?: 'LocalAWSProfile', Name: string, Region?: string | null, Source: string, AuthType: string, IsDefault: boolean }> };

export type GetAwsRegionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAwsRegionsQuery = { __typename?: 'Query', AWSRegions: Array<{ __typename?: 'AWSRegion', Id: string, Description: string, Partition: string }> };

export type AddAwsProviderMutationVariables = Exact<{
  input: AwsProviderInput;
}>;


export type AddAwsProviderMutation = { __typename?: 'Mutation', AddAWSProvider: { __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, ProfileName?: string | null, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null } };

export type UpdateAwsProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: AwsProviderInput;
}>;


export type UpdateAwsProviderMutation = { __typename?: 'Mutation', UpdateAWSProvider: { __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, ProfileName?: string | null, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null } };

export type RemoveCloudProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type RemoveCloudProviderMutation = { __typename?: 'Mutation', RemoveCloudProvider: { __typename?: 'StatusResponse', Status: boolean } };

export type TestCloudProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type TestCloudProviderMutation = { __typename?: 'Mutation', TestCloudProvider: CloudProviderStatus };

export type TestAwsCredentialsMutationVariables = Exact<{
  input: AwsProviderInput;
}>;


export type TestAwsCredentialsMutation = { __typename?: 'Mutation', TestAWSCredentials: CloudProviderStatus };

export type GenerateRdsAuthTokenMutationVariables = Exact<{
  providerID: Scalars['ID']['input'];
  endpoint: Scalars['String']['input'];
  port: Scalars['Int']['input'];
  region: Scalars['String']['input'];
  username: Scalars['String']['input'];
}>;


export type GenerateRdsAuthTokenMutation = { __typename?: 'Mutation', GenerateRDSAuthToken: string };

export type RefreshCloudProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type RefreshCloudProviderMutation = { __typename?: 'Mutation', RefreshCloudProvider: { __typename?: 'AWSProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProfileName?: string | null, DiscoverRDS: boolean, DiscoverElastiCache: boolean, DiscoverDocumentDB: boolean } | { __typename?: 'AzureProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, SubscriptionID: string, TenantID?: string | null, ResourceGroup?: string | null, DiscoverPostgreSQL: boolean, DiscoverMySQL: boolean, DiscoverRedis: boolean, DiscoverCosmosDB: boolean } | { __typename?: 'GCPProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProjectID: string, ServiceAccountKeyPath?: string | null, DiscoverCloudSQL: boolean, DiscoverAlloyDB: boolean, DiscoverMemorystore: boolean } };

export type GetAzureProvidersQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAzureProvidersQuery = { __typename?: 'Query', AzureProviders: Array<{ __typename?: 'AzureProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, SubscriptionID: string, TenantID?: string | null, ResourceGroup?: string | null, DiscoverPostgreSQL: boolean, DiscoverMySQL: boolean, DiscoverRedis: boolean, DiscoverCosmosDB: boolean, Status: CloudProviderStatus, DiscoveredCount: number, LastDiscoveryAt?: string | null, Error?: string | null }> };

export type GetAzureProviderQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetAzureProviderQuery = { __typename?: 'Query', AzureProvider?: { __typename?: 'AzureProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, SubscriptionID: string, TenantID?: string | null, ResourceGroup?: string | null, DiscoverPostgreSQL: boolean, DiscoverMySQL: boolean, DiscoverRedis: boolean, DiscoverCosmosDB: boolean, Status: CloudProviderStatus, DiscoveredCount: number, LastDiscoveryAt?: string | null, Error?: string | null } | null };

export type GetAzureSubscriptionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAzureSubscriptionsQuery = { __typename?: 'Query', AzureSubscriptions: Array<{ __typename?: 'AzureSubscription', Id: string, DisplayName: string, State: string, TenantID: string }> };

export type GetAzureRegionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAzureRegionsQuery = { __typename?: 'Query', AzureRegions: Array<{ __typename?: 'AzureRegion', Id: string, DisplayName: string, Geography: string }> };

export type AddAzureProviderMutationVariables = Exact<{
  input: AzureProviderInput;
}>;


export type AddAzureProviderMutation = { __typename?: 'Mutation', AddAzureProvider: { __typename?: 'AzureProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, SubscriptionID: string, TenantID?: string | null, ResourceGroup?: string | null, DiscoverPostgreSQL: boolean, DiscoverMySQL: boolean, DiscoverRedis: boolean, DiscoverCosmosDB: boolean, Status: CloudProviderStatus, DiscoveredCount: number, LastDiscoveryAt?: string | null, Error?: string | null } };

export type UpdateAzureProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: AzureProviderInput;
}>;


export type UpdateAzureProviderMutation = { __typename?: 'Mutation', UpdateAzureProvider: { __typename?: 'AzureProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, SubscriptionID: string, TenantID?: string | null, ResourceGroup?: string | null, DiscoverPostgreSQL: boolean, DiscoverMySQL: boolean, DiscoverRedis: boolean, DiscoverCosmosDB: boolean, Status: CloudProviderStatus, DiscoveredCount: number, LastDiscoveryAt?: string | null, Error?: string | null } };

export type TestAzureCredentialsMutationVariables = Exact<{
  input: AzureProviderInput;
}>;


export type TestAzureCredentialsMutation = { __typename?: 'Mutation', TestAzureCredentials: CloudProviderStatus };

export type GenerateAzureAdTokenMutationVariables = Exact<{
  providerID: Scalars['ID']['input'];
  sourceType: Scalars['String']['input'];
}>;


export type GenerateAzureAdTokenMutation = { __typename?: 'Mutation', GenerateAzureADToken: string };

export type RefreshAzureProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type RefreshAzureProviderMutation = { __typename?: 'Mutation', RefreshAzureProvider: { __typename?: 'AzureProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, SubscriptionID: string, TenantID?: string | null, ResourceGroup?: string | null, DiscoverPostgreSQL: boolean, DiscoverMySQL: boolean, DiscoverRedis: boolean, DiscoverCosmosDB: boolean, Status: CloudProviderStatus, DiscoveredCount: number, LastDiscoveryAt?: string | null, Error?: string | null } };

export type GetGcpProvidersQueryVariables = Exact<{ [key: string]: never; }>;


export type GetGcpProvidersQuery = { __typename?: 'Query', GCPProviders: Array<{ __typename?: 'GCPProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProjectID: string, ServiceAccountKeyPath?: string | null, DiscoverCloudSQL: boolean, DiscoverAlloyDB: boolean, DiscoverMemorystore: boolean }> };

export type GetGcpProviderQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetGcpProviderQuery = { __typename?: 'Query', GCPProvider?: { __typename?: 'GCPProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProjectID: string, ServiceAccountKeyPath?: string | null, DiscoverCloudSQL: boolean, DiscoverAlloyDB: boolean, DiscoverMemorystore: boolean } | null };

export type GetLocalGcpProjectsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetLocalGcpProjectsQuery = { __typename?: 'Query', LocalGCPProjects: Array<{ __typename?: 'LocalGCPProject', ProjectID: string, Name: string, Source: string, IsDefault: boolean }> };

export type GetGcpRegionsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetGcpRegionsQuery = { __typename?: 'Query', GCPRegions: Array<{ __typename?: 'GCPRegion', Id: string, Description: string }> };

export type AddGcpProviderMutationVariables = Exact<{
  input: GcpProviderInput;
}>;


export type AddGcpProviderMutation = { __typename?: 'Mutation', AddGCPProvider: { __typename?: 'GCPProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProjectID: string, ServiceAccountKeyPath?: string | null, DiscoverCloudSQL: boolean, DiscoverAlloyDB: boolean, DiscoverMemorystore: boolean } };

export type UpdateGcpProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: GcpProviderInput;
}>;


export type UpdateGcpProviderMutation = { __typename?: 'Mutation', UpdateGCPProvider: { __typename?: 'GCPProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProjectID: string, ServiceAccountKeyPath?: string | null, DiscoverCloudSQL: boolean, DiscoverAlloyDB: boolean, DiscoverMemorystore: boolean } };

export type TestGcpCredentialsMutationVariables = Exact<{
  input: GcpProviderInput;
}>;


export type TestGcpCredentialsMutation = { __typename?: 'Mutation', TestGCPCredentials: CloudProviderStatus };

export type RefreshGcpProviderMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type RefreshGcpProviderMutation = { __typename?: 'Mutation', RefreshGCPProvider: { __typename?: 'GCPProvider', Id: string, ProviderType: CloudProviderType, Name: string, Region: string, Status: CloudProviderStatus, LastDiscoveryAt?: string | null, DiscoveredCount: number, Error?: string | null, ProjectID: string, ServiceAccountKeyPath?: string | null, DiscoverCloudSQL: boolean, DiscoverAlloyDB: boolean, DiscoverMemorystore: boolean } };

export type GenerateCloudSqliamAuthTokenMutationVariables = Exact<{
  providerID: Scalars['ID']['input'];
  username: Scalars['String']['input'];
}>;


export type GenerateCloudSqliamAuthTokenMutation = { __typename?: 'Mutation', GenerateCloudSQLIAMAuthToken: string };

export type SettingsConfigQueryVariables = Exact<{ [key: string]: never; }>;


export type SettingsConfigQuery = { __typename?: 'Query', SettingsConfig: { __typename?: 'SettingsConfig', MetricsEnabled?: boolean | null, CloudProvidersEnabled: boolean, DisableCredentialForm: boolean, EnableNewUI: boolean, MaxPageSize: number } };

export type UpdateSettingsMutationVariables = Exact<{
  newSettings: SettingsConfigInput;
}>;


export type UpdateSettingsMutation = { __typename?: 'Mutation', UpdateSettings: { __typename?: 'StatusResponse', Status: boolean } };

export type AddRowMutationVariables = Exact<{
  ref: SourceObjectRefInput;
  values: Array<RecordInput> | RecordInput;
}>;


export type AddRowMutation = { __typename?: 'Mutation', AddRow: { __typename?: 'StatusResponse', Status: boolean } };

export type AddStorageUnitMutationVariables = Exact<{
  parent?: InputMaybe<SourceObjectRefInput>;
  storageUnit: Scalars['String']['input'];
  fields: Array<RecordInput> | RecordInput;
}>;


export type AddStorageUnitMutation = { __typename?: 'Mutation', AddStorageUnit: { __typename?: 'StatusResponse', Status: boolean } };

export type DeleteRowMutationVariables = Exact<{
  ref: SourceObjectRefInput;
  values: Array<RecordInput> | RecordInput;
}>;


export type DeleteRowMutation = { __typename?: 'Mutation', DeleteRow: { __typename?: 'StatusResponse', Status: boolean } };

export type GetColumnsBatchQueryVariables = Exact<{
  refs: Array<SourceObjectRefInput> | SourceObjectRefInput;
}>;


export type GetColumnsBatchQuery = { __typename?: 'Query', ColumnsBatch: Array<{ __typename?: 'SourceObjectColumns', StorageUnit: { __typename?: 'SourceObjectRef', Kind: SourceObjectKind, Locator: string, Path: Array<string> }, Columns: Array<{ __typename?: 'Column', Name: string, Type: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> }> };

export type GetStorageUnitRowsQueryVariables = Exact<{
  ref: SourceObjectRefInput;
  where?: InputMaybe<WhereCondition>;
  sort?: InputMaybe<Array<SortCondition> | SortCondition>;
  pageSize: Scalars['Int']['input'];
  pageOffset: Scalars['Int']['input'];
}>;


export type GetStorageUnitRowsQuery = { __typename?: 'Query', Row: { __typename?: 'RowsResult', Rows: Array<Array<string>>, DisableUpdate: boolean, TotalCount: number, Columns: Array<{ __typename?: 'Column', Type: string, Name: string, IsPrimary: boolean, IsForeignKey: boolean, ReferencedTable?: string | null, ReferencedColumn?: string | null, Length?: number | null, Precision?: number | null, Scale?: number | null }> } };

export type GetSourceContentQueryVariables = Exact<{
  ref: SourceObjectRefInput;
}>;


export type GetSourceContentQuery = { __typename?: 'Query', Content?: { __typename?: 'SourceContent', Text?: string | null, MIMEType: string, IsBinary: boolean, SizeBytes: string, Truncated: boolean, FileName: string, ModifiedAt?: string | null } | null };

export type GetStorageUnitsQueryVariables = Exact<{
  parent?: InputMaybe<SourceObjectRefInput>;
}>;


export type GetStorageUnitsQuery = { __typename?: 'Query', StorageUnit: Array<{ __typename?: 'SourceObject', Kind: SourceObjectKind, Name: string, Actions: Array<SourceAction>, HasChildren: boolean, Ref: { __typename?: 'SourceObjectRef', Kind: SourceObjectKind, Locator: string, Path: Array<string> }, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };

export type UpdateStorageUnitMutationVariables = Exact<{
  ref: SourceObjectRefInput;
  values: Array<RecordInput> | RecordInput;
  updatedColumns: Array<Scalars['String']['input']> | Scalars['String']['input'];
}>;


export type UpdateStorageUnitMutation = { __typename?: 'Mutation', UpdateStorageUnit: { __typename?: 'StatusResponse', Status: boolean } };


export const SourceProfilesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SourceProfiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"SourceProfiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"Alias"},"name":{"kind":"Name","value":"DisplayName"}},{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","alias":{"kind":"Name","value":"Type"},"name":{"kind":"Name","value":"SourceType"}},{"kind":"Field","name":{"kind":"Name","value":"Values"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Key"}},{"kind":"Field","name":{"kind":"Name","value":"Value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"IsEnvironmentDefined"}},{"kind":"Field","name":{"kind":"Name","value":"Source"}},{"kind":"Field","name":{"kind":"Name","value":"SSLConfigured"}}]}}]}}]} as unknown as DocumentNode<SourceProfilesQuery, SourceProfilesQueryVariables>;
export const GetSchemaDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSchema"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"parent"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"kinds"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectKind"}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"Schema"},"name":{"kind":"Name","value":"SourceObjects"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"parent"},"value":{"kind":"Variable","name":{"kind":"Name","value":"parent"}}},{"kind":"Argument","name":{"kind":"Name","value":"kinds"},"value":{"kind":"Variable","name":{"kind":"Name","value":"kinds"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Kind"}},{"kind":"Field","name":{"kind":"Name","value":"Ref"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Kind"}},{"kind":"Field","name":{"kind":"Name","value":"Locator"}},{"kind":"Field","name":{"kind":"Name","value":"Path"}}]}}]}}]}}]} as unknown as DocumentNode<GetSchemaQuery, GetSchemaQueryVariables>;
export const GetUpdateInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetUpdateInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"UpdateInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"currentVersion"}},{"kind":"Field","name":{"kind":"Name","value":"latestVersion"}},{"kind":"Field","name":{"kind":"Name","value":"updateAvailable"}},{"kind":"Field","name":{"kind":"Name","value":"releaseURL"}}]}}]}}]} as unknown as DocumentNode<GetUpdateInfoQuery, GetUpdateInfoQueryVariables>;
export const GetVersionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetVersion"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Version"}}]}}]} as unknown as DocumentNode<GetVersionQuery, GetVersionQueryVariables>;
export const ExecuteConfirmedSqlDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ExecuteConfirmedSQL"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"query"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"operationType"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ExecuteConfirmedSQL"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"query"}}},{"kind":"Argument","name":{"kind":"Name","value":"operationType"},"value":{"kind":"Variable","name":{"kind":"Name","value":"operationType"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"Text"}},{"kind":"Field","name":{"kind":"Name","value":"Result"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Columns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"IsPrimary"}},{"kind":"Field","name":{"kind":"Name","value":"IsForeignKey"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedTable"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedColumn"}},{"kind":"Field","name":{"kind":"Name","value":"Length"}},{"kind":"Field","name":{"kind":"Name","value":"Precision"}},{"kind":"Field","name":{"kind":"Name","value":"Scale"}}]}},{"kind":"Field","name":{"kind":"Name","value":"Rows"}},{"kind":"Field","name":{"kind":"Name","value":"DisableUpdate"}},{"kind":"Field","name":{"kind":"Name","value":"TotalCount"}}]}},{"kind":"Field","name":{"kind":"Name","value":"RequiresConfirmation"}}]}}]}}]} as unknown as DocumentNode<ExecuteConfirmedSqlMutation, ExecuteConfirmedSqlMutationVariables>;
export const GenerateChatTitleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"GenerateChatTitle"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"GenerateChatTitleInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"GenerateChatTitle"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Title"}}]}}]}}]} as unknown as DocumentNode<GenerateChatTitleMutation, GenerateChatTitleMutationVariables>;
export const GetHealthDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetHealth"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Server"}},{"kind":"Field","name":{"kind":"Name","value":"Database"}}]}}]}}]} as unknown as DocumentNode<GetHealthQuery, GetHealthQueryVariables>;
export const ImportPreviewDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ImportPreview"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"file"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Upload"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"options"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ImportFileOptions"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"useHeaderMapping"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ImportPreview"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"file"},"value":{"kind":"Variable","name":{"kind":"Name","value":"file"}}},{"kind":"Argument","name":{"kind":"Name","value":"options"},"value":{"kind":"Variable","name":{"kind":"Name","value":"options"}}},{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"useHeaderMapping"},"value":{"kind":"Variable","name":{"kind":"Name","value":"useHeaderMapping"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Sheet"}},{"kind":"Field","name":{"kind":"Name","value":"Columns"}},{"kind":"Field","name":{"kind":"Name","value":"Rows"}},{"kind":"Field","name":{"kind":"Name","value":"Truncated"}},{"kind":"Field","name":{"kind":"Name","value":"ValidationError"}},{"kind":"Field","name":{"kind":"Name","value":"RequiresAllowAutoGenerated"}},{"kind":"Field","name":{"kind":"Name","value":"AutoGeneratedColumns"}},{"kind":"Field","name":{"kind":"Name","value":"Mapping"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"SourceColumn"}},{"kind":"Field","name":{"kind":"Name","value":"TargetColumn"}}]}}]}}]}}]} as unknown as DocumentNode<ImportPreviewMutation, ImportPreviewMutationVariables>;
export const ImportSqlDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ImportSQL"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ImportSQLInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ImportSQL"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"Message"}},{"kind":"Field","name":{"kind":"Name","value":"Detail"}}]}}]}}]} as unknown as DocumentNode<ImportSqlMutation, ImportSqlMutationVariables>;
export const ImportTableFileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ImportTableFile"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ImportFileInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"ImportTableFile"},"name":{"kind":"Name","value":"ImportSourceObjectFile"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"Message"}},{"kind":"Field","name":{"kind":"Name","value":"Detail"}}]}}]}}]} as unknown as DocumentNode<ImportTableFileMutation, ImportTableFileMutationVariables>;
export const SourceSessionMetadataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SourceSessionMetadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"SourceSessionMetadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"sourceType"},"name":{"kind":"Name","value":"SourceType"}},{"kind":"Field","alias":{"kind":"Name","value":"queryLanguages"},"name":{"kind":"Name","value":"QueryLanguages"}},{"kind":"Field","alias":{"kind":"Name","value":"typeDefinitions"},"name":{"kind":"Name","value":"TypeDefinitions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"hasLength"}},{"kind":"Field","name":{"kind":"Name","value":"hasPrecision"}},{"kind":"Field","name":{"kind":"Name","value":"defaultLength"}},{"kind":"Field","name":{"kind":"Name","value":"defaultPrecision"}},{"kind":"Field","name":{"kind":"Name","value":"category"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"operators"},"name":{"kind":"Name","value":"Operators"}},{"kind":"Field","alias":{"kind":"Name","value":"aliasMap"},"name":{"kind":"Name","value":"AliasMap"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Key"}},{"kind":"Field","name":{"kind":"Name","value":"Value"}}]}}]}}]}}]} as unknown as DocumentNode<SourceSessionMetadataQuery, SourceSessionMetadataQueryVariables>;
export const SourceTypesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SourceTypes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"SourceTypes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"id"},"name":{"kind":"Name","value":"Id"}},{"kind":"Field","alias":{"kind":"Name","value":"label"},"name":{"kind":"Name","value":"Label"}},{"kind":"Field","alias":{"kind":"Name","value":"connector"},"name":{"kind":"Name","value":"Connector"}},{"kind":"Field","alias":{"kind":"Name","value":"category"},"name":{"kind":"Name","value":"Category"}},{"kind":"Field","alias":{"kind":"Name","value":"traits"},"name":{"kind":"Name","value":"Traits"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"connection"},"name":{"kind":"Name","value":"Connection"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"transport"},"name":{"kind":"Name","value":"Transport"}},{"kind":"Field","alias":{"kind":"Name","value":"hostInputMode"},"name":{"kind":"Name","value":"HostInputMode"}},{"kind":"Field","alias":{"kind":"Name","value":"hostInputUrlParser"},"name":{"kind":"Name","value":"HostInputURLParser"}},{"kind":"Field","alias":{"kind":"Name","value":"supportsCustomCAContent"},"name":{"kind":"Name","value":"SupportsCustomCAContent"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"presentation"},"name":{"kind":"Name","value":"Presentation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"profileLabelStrategy"},"name":{"kind":"Name","value":"ProfileLabelStrategy"}},{"kind":"Field","alias":{"kind":"Name","value":"schemaFidelity"},"name":{"kind":"Name","value":"SchemaFidelity"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"query"},"name":{"kind":"Name","value":"Query"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"supportsAnalyze"},"name":{"kind":"Name","value":"SupportsAnalyze"}},{"kind":"Field","alias":{"kind":"Name","value":"explainMode"},"name":{"kind":"Name","value":"ExplainMode"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"mockData"},"name":{"kind":"Name","value":"MockData"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"supportsRelationalDependencies"},"name":{"kind":"Name","value":"SupportsRelationalDependencies"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"connectionFields"},"name":{"kind":"Name","value":"ConnectionFields"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Key"}},{"kind":"Field","name":{"kind":"Name","value":"Kind"}},{"kind":"Field","name":{"kind":"Name","value":"Section"}},{"kind":"Field","name":{"kind":"Name","value":"Required"}},{"kind":"Field","name":{"kind":"Name","value":"LabelKey"}},{"kind":"Field","name":{"kind":"Name","value":"PlaceholderKey"}},{"kind":"Field","name":{"kind":"Name","value":"DefaultValue"}},{"kind":"Field","name":{"kind":"Name","value":"SupportsOptions"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"contract"},"name":{"kind":"Name","value":"Contract"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Model"}},{"kind":"Field","name":{"kind":"Name","value":"Surfaces"}},{"kind":"Field","name":{"kind":"Name","value":"RootActions"}},{"kind":"Field","name":{"kind":"Name","value":"BrowsePath"}},{"kind":"Field","name":{"kind":"Name","value":"DefaultObjectKind"}},{"kind":"Field","name":{"kind":"Name","value":"GraphScopeKind"}},{"kind":"Field","name":{"kind":"Name","value":"ObjectTypes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Kind"}},{"kind":"Field","name":{"kind":"Name","value":"DataShape"}},{"kind":"Field","name":{"kind":"Name","value":"Actions"}},{"kind":"Field","name":{"kind":"Name","value":"Views"}},{"kind":"Field","name":{"kind":"Name","value":"SingularLabel"}},{"kind":"Field","name":{"kind":"Name","value":"PluralLabel"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"discoveryPrefill"},"name":{"kind":"Name","value":"DiscoveryPrefill"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AdvancedDefaults"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Key"}},{"kind":"Field","name":{"kind":"Name","value":"Value"}},{"kind":"Field","name":{"kind":"Name","value":"MetadataKey"}},{"kind":"Field","name":{"kind":"Name","value":"DefaultValue"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderTypes"}},{"kind":"Field","name":{"kind":"Name","value":"Conditions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Key"}},{"kind":"Field","name":{"kind":"Name","value":"Value"}}]}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"isAwsManaged"},"name":{"kind":"Name","value":"IsAWSManaged"}},{"kind":"Field","alias":{"kind":"Name","value":"sslModes"},"name":{"kind":"Name","value":"SSLModes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"aliases"}}]}}]}}]}}]} as unknown as DocumentNode<SourceTypesQuery, SourceTypesQueryVariables>;
export const GetSslStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSSLStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"SSLStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"IsEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"Mode"}}]}}]}}]} as unknown as DocumentNode<GetSslStatusQuery, GetSslStatusQueryVariables>;
export const AnalyzeMockDataDependenciesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AnalyzeMockDataDependencies"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"rowCount"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fkDensityRatio"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AnalyzeMockDataDependencies"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"rowCount"},"value":{"kind":"Variable","name":{"kind":"Name","value":"rowCount"}}},{"kind":"Argument","name":{"kind":"Name","value":"fkDensityRatio"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fkDensityRatio"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"GenerationOrder"}},{"kind":"Field","name":{"kind":"Name","value":"Tables"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Table"}},{"kind":"Field","name":{"kind":"Name","value":"RowsToGenerate"}},{"kind":"Field","name":{"kind":"Name","value":"IsBlocked"}},{"kind":"Field","name":{"kind":"Name","value":"UsesExistingData"}}]}},{"kind":"Field","name":{"kind":"Name","value":"TotalRows"}},{"kind":"Field","name":{"kind":"Name","value":"Warnings"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}}]}}]}}]} as unknown as DocumentNode<AnalyzeMockDataDependenciesQuery, AnalyzeMockDataDependenciesQueryVariables>;
export const MockDataMaxRowCountDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"MockDataMaxRowCount"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"MockDataMaxRowCount"}}]}}]} as unknown as DocumentNode<MockDataMaxRowCountQuery, MockDataMaxRowCountQueryVariables>;
export const GenerateMockDataDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"GenerateMockData"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"MockDataGenerationInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"GenerateMockData"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AmountGenerated"}},{"kind":"Field","name":{"kind":"Name","value":"Details"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Table"}},{"kind":"Field","name":{"kind":"Name","value":"RowsGenerated"}},{"kind":"Field","name":{"kind":"Name","value":"UsedExistingData"}}]}}]}}]}}]} as unknown as DocumentNode<GenerateMockDataMutation, GenerateMockDataMutationVariables>;
export const SourceFieldOptionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SourceFieldOptions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sourceType"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"values"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RecordInput"}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"SourceFieldOptions"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"sourceType"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sourceType"}}},{"kind":"Argument","name":{"kind":"Name","value":"fieldKey"},"value":{"kind":"StringValue","value":"Database","block":false}},{"kind":"Argument","name":{"kind":"Name","value":"values"},"value":{"kind":"Variable","name":{"kind":"Name","value":"values"}}}]}]}}]} as unknown as DocumentNode<SourceFieldOptionsQuery, SourceFieldOptionsQueryVariables>;
export const LoginWithSourceProfileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"LoginWithSourceProfile"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"profile"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceProfileLoginInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"LoginWithSourceProfile"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"profile"},"value":{"kind":"Variable","name":{"kind":"Name","value":"profile"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}}]}}]}}]} as unknown as DocumentNode<LoginWithSourceProfileMutation, LoginWithSourceProfileMutationVariables>;
export const LoginSourceDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"LoginSource"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"credentials"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceLoginInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"LoginSource"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"credentials"},"value":{"kind":"Variable","name":{"kind":"Name","value":"credentials"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}}]}}]}}]} as unknown as DocumentNode<LoginSourceMutation, LoginSourceMutationVariables>;
export const LogoutDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"Logout"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Logout"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}}]}}]}}]} as unknown as DocumentNode<LogoutMutation, LogoutMutationVariables>;
export const GetAiProvidersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAIProviders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AIProviders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderId"}},{"kind":"Field","name":{"kind":"Name","value":"IsEnvironmentDefined"}},{"kind":"Field","name":{"kind":"Name","value":"IsGeneric"}}]}}]}}]} as unknown as DocumentNode<GetAiProvidersQuery, GetAiProvidersQueryVariables>;
export const GetAiChatDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAIChat"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"providerId"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"modelType"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"token"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"previousConversation"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"query"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"model"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AIChat"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"providerId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"providerId"}}},{"kind":"Argument","name":{"kind":"Name","value":"modelType"},"value":{"kind":"Variable","name":{"kind":"Name","value":"modelType"}}},{"kind":"Argument","name":{"kind":"Name","value":"token"},"value":{"kind":"Variable","name":{"kind":"Name","value":"token"}}},{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"PreviousConversation"},"value":{"kind":"Variable","name":{"kind":"Name","value":"previousConversation"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"Query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"query"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"Model"},"value":{"kind":"Variable","name":{"kind":"Name","value":"model"}}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"Result"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Columns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"Rows"}}]}},{"kind":"Field","name":{"kind":"Name","value":"Text"}}]}}]}}]} as unknown as DocumentNode<GetAiChatQuery, GetAiChatQueryVariables>;
export const GetDatabaseQuerySuggestionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDatabaseQuerySuggestions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"DatabaseQuerySuggestions"},"name":{"kind":"Name","value":"SourceQuerySuggestions"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"category"}}]}}]}}]} as unknown as DocumentNode<GetDatabaseQuerySuggestionsQuery, GetDatabaseQuerySuggestionsQueryVariables>;
export const GetAiModelsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAIModels"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"providerId"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"modelType"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"token"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AIModel"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"providerId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"providerId"}}},{"kind":"Argument","name":{"kind":"Name","value":"modelType"},"value":{"kind":"Variable","name":{"kind":"Name","value":"modelType"}}},{"kind":"Argument","name":{"kind":"Name","value":"token"},"value":{"kind":"Variable","name":{"kind":"Name","value":"token"}}}]}]}}]} as unknown as DocumentNode<GetAiModelsQuery, GetAiModelsQueryVariables>;
export const GetColumnsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetColumns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"Columns"},"name":{"kind":"Name","value":"SourceColumns"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"IsPrimary"}},{"kind":"Field","name":{"kind":"Name","value":"IsForeignKey"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedTable"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedColumn"}},{"kind":"Field","name":{"kind":"Name","value":"Length"}},{"kind":"Field","name":{"kind":"Name","value":"Precision"}},{"kind":"Field","name":{"kind":"Name","value":"Scale"}}]}}]}}]} as unknown as DocumentNode<GetColumnsQuery, GetColumnsQueryVariables>;
export const GetGraphDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetGraph"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"Graph"},"name":{"kind":"Name","value":"SourceGraph"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Unit"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Ref"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Kind"}},{"kind":"Field","name":{"kind":"Name","value":"Locator"}},{"kind":"Field","name":{"kind":"Name","value":"Path"}}]}},{"kind":"Field","name":{"kind":"Name","value":"Kind"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","alias":{"kind":"Name","value":"Attributes"},"name":{"kind":"Name","value":"Metadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Key"}},{"kind":"Field","name":{"kind":"Name","value":"Value"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"Relations"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Relationship"}},{"kind":"Field","name":{"kind":"Name","value":"SourceColumn"}},{"kind":"Field","name":{"kind":"Name","value":"TargetColumn"}}]}}]}}]}}]} as unknown as DocumentNode<GetGraphQuery, GetGraphQueryVariables>;
export const ColumnsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Columns"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"Columns"},"name":{"kind":"Name","value":"SourceColumns"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"IsPrimary"}},{"kind":"Field","name":{"kind":"Name","value":"IsForeignKey"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedTable"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedColumn"}},{"kind":"Field","name":{"kind":"Name","value":"Length"}},{"kind":"Field","name":{"kind":"Name","value":"Precision"}},{"kind":"Field","name":{"kind":"Name","value":"Scale"}}]}}]}}]} as unknown as DocumentNode<ColumnsQuery, ColumnsQueryVariables>;
export const RawExecuteDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"RawExecute"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"query"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"RawExecute"},"name":{"kind":"Name","value":"RunSourceQuery"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"query"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Columns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"Rows"}},{"kind":"Field","name":{"kind":"Name","value":"TotalCount"}}]}}]}}]} as unknown as DocumentNode<RawExecuteQuery, RawExecuteQueryVariables>;
export const GetCloudProvidersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetCloudProviders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"CloudProviders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"AWSProvider"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProfileName"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRDS"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverElastiCache"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverDocumentDB"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"AzureProvider"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"SubscriptionID"}},{"kind":"Field","name":{"kind":"Name","value":"TenantID"}},{"kind":"Field","name":{"kind":"Name","value":"ResourceGroup"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverPostgreSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMySQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRedis"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCosmosDB"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"GCPProvider"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"ServiceAccountKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCloudSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverAlloyDB"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMemorystore"}}]}}]}}]}}]} as unknown as DocumentNode<GetCloudProvidersQuery, GetCloudProvidersQueryVariables>;
export const GetCloudProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetCloudProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"CloudProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"AWSProvider"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProfileName"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRDS"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverElastiCache"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverDocumentDB"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"AzureProvider"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"SubscriptionID"}},{"kind":"Field","name":{"kind":"Name","value":"TenantID"}},{"kind":"Field","name":{"kind":"Name","value":"ResourceGroup"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverPostgreSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMySQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRedis"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCosmosDB"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"GCPProvider"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"ServiceAccountKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCloudSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverAlloyDB"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMemorystore"}}]}}]}}]}}]} as unknown as DocumentNode<GetCloudProviderQuery, GetCloudProviderQueryVariables>;
export const GetDiscoveredConnectionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDiscoveredConnections"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"DiscoveredConnections"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderID"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","alias":{"kind":"Name","value":"DatabaseType"},"name":{"kind":"Name","value":"SourceType"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"Metadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Key"}},{"kind":"Field","name":{"kind":"Name","value":"Value"}}]}}]}}]}}]} as unknown as DocumentNode<GetDiscoveredConnectionsQuery, GetDiscoveredConnectionsQueryVariables>;
export const GetProviderConnectionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetProviderConnections"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"providerId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ProviderConnections"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"providerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"providerId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderID"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","alias":{"kind":"Name","value":"DatabaseType"},"name":{"kind":"Name","value":"SourceType"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"Metadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Key"}},{"kind":"Field","name":{"kind":"Name","value":"Value"}}]}}]}}]}}]} as unknown as DocumentNode<GetProviderConnectionsQuery, GetProviderConnectionsQueryVariables>;
export const GetLocalAwsProfilesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetLocalAWSProfiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"LocalAWSProfiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Source"}},{"kind":"Field","name":{"kind":"Name","value":"AuthType"}},{"kind":"Field","name":{"kind":"Name","value":"IsDefault"}}]}}]}}]} as unknown as DocumentNode<GetLocalAwsProfilesQuery, GetLocalAwsProfilesQueryVariables>;
export const GetAwsRegionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAWSRegions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AWSRegions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"Description"}},{"kind":"Field","name":{"kind":"Name","value":"Partition"}}]}}]}}]} as unknown as DocumentNode<GetAwsRegionsQuery, GetAwsRegionsQueryVariables>;
export const AddAwsProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddAWSProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AWSProviderInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AddAWSProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"ProfileName"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRDS"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverElastiCache"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverDocumentDB"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}}]}}]}}]} as unknown as DocumentNode<AddAwsProviderMutation, AddAwsProviderMutationVariables>;
export const UpdateAwsProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAWSProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AWSProviderInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"UpdateAWSProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"ProfileName"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRDS"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverElastiCache"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverDocumentDB"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}}]}}]}}]} as unknown as DocumentNode<UpdateAwsProviderMutation, UpdateAwsProviderMutationVariables>;
export const RemoveCloudProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveCloudProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"RemoveCloudProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}}]}}]}}]} as unknown as DocumentNode<RemoveCloudProviderMutation, RemoveCloudProviderMutationVariables>;
export const TestCloudProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TestCloudProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"TestCloudProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}]}]}}]} as unknown as DocumentNode<TestCloudProviderMutation, TestCloudProviderMutationVariables>;
export const TestAwsCredentialsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TestAWSCredentials"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AWSProviderInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"TestAWSCredentials"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}]}]}}]} as unknown as DocumentNode<TestAwsCredentialsMutation, TestAwsCredentialsMutationVariables>;
export const GenerateRdsAuthTokenDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"GenerateRDSAuthToken"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"providerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"endpoint"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"port"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"region"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"username"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"GenerateRDSAuthToken"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"providerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"providerID"}}},{"kind":"Argument","name":{"kind":"Name","value":"endpoint"},"value":{"kind":"Variable","name":{"kind":"Name","value":"endpoint"}}},{"kind":"Argument","name":{"kind":"Name","value":"port"},"value":{"kind":"Variable","name":{"kind":"Name","value":"port"}}},{"kind":"Argument","name":{"kind":"Name","value":"region"},"value":{"kind":"Variable","name":{"kind":"Name","value":"region"}}},{"kind":"Argument","name":{"kind":"Name","value":"username"},"value":{"kind":"Variable","name":{"kind":"Name","value":"username"}}}]}]}}]} as unknown as DocumentNode<GenerateRdsAuthTokenMutation, GenerateRdsAuthTokenMutationVariables>;
export const RefreshCloudProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshCloudProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"RefreshCloudProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"AWSProvider"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProfileName"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRDS"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverElastiCache"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverDocumentDB"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"AzureProvider"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"SubscriptionID"}},{"kind":"Field","name":{"kind":"Name","value":"TenantID"}},{"kind":"Field","name":{"kind":"Name","value":"ResourceGroup"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverPostgreSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMySQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRedis"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCosmosDB"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"GCPProvider"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"ServiceAccountKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCloudSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverAlloyDB"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMemorystore"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshCloudProviderMutation, RefreshCloudProviderMutationVariables>;
export const GetAzureProvidersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAzureProviders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AzureProviders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"SubscriptionID"}},{"kind":"Field","name":{"kind":"Name","value":"TenantID"}},{"kind":"Field","name":{"kind":"Name","value":"ResourceGroup"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverPostgreSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMySQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRedis"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCosmosDB"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}}]}}]}}]} as unknown as DocumentNode<GetAzureProvidersQuery, GetAzureProvidersQueryVariables>;
export const GetAzureProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAzureProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AzureProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"SubscriptionID"}},{"kind":"Field","name":{"kind":"Name","value":"TenantID"}},{"kind":"Field","name":{"kind":"Name","value":"ResourceGroup"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverPostgreSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMySQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRedis"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCosmosDB"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}}]}}]}}]} as unknown as DocumentNode<GetAzureProviderQuery, GetAzureProviderQueryVariables>;
export const GetAzureSubscriptionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAzureSubscriptions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AzureSubscriptions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"DisplayName"}},{"kind":"Field","name":{"kind":"Name","value":"State"}},{"kind":"Field","name":{"kind":"Name","value":"TenantID"}}]}}]}}]} as unknown as DocumentNode<GetAzureSubscriptionsQuery, GetAzureSubscriptionsQueryVariables>;
export const GetAzureRegionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAzureRegions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AzureRegions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"DisplayName"}},{"kind":"Field","name":{"kind":"Name","value":"Geography"}}]}}]}}]} as unknown as DocumentNode<GetAzureRegionsQuery, GetAzureRegionsQueryVariables>;
export const AddAzureProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddAzureProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AzureProviderInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AddAzureProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"SubscriptionID"}},{"kind":"Field","name":{"kind":"Name","value":"TenantID"}},{"kind":"Field","name":{"kind":"Name","value":"ResourceGroup"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverPostgreSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMySQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRedis"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCosmosDB"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}}]}}]}}]} as unknown as DocumentNode<AddAzureProviderMutation, AddAzureProviderMutationVariables>;
export const UpdateAzureProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAzureProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AzureProviderInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"UpdateAzureProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"SubscriptionID"}},{"kind":"Field","name":{"kind":"Name","value":"TenantID"}},{"kind":"Field","name":{"kind":"Name","value":"ResourceGroup"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverPostgreSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMySQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRedis"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCosmosDB"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}}]}}]}}]} as unknown as DocumentNode<UpdateAzureProviderMutation, UpdateAzureProviderMutationVariables>;
export const TestAzureCredentialsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TestAzureCredentials"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"AzureProviderInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"TestAzureCredentials"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}]}]}}]} as unknown as DocumentNode<TestAzureCredentialsMutation, TestAzureCredentialsMutationVariables>;
export const GenerateAzureAdTokenDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"GenerateAzureADToken"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"providerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sourceType"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"GenerateAzureADToken"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"providerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"providerID"}}},{"kind":"Argument","name":{"kind":"Name","value":"sourceType"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sourceType"}}}]}]}}]} as unknown as DocumentNode<GenerateAzureAdTokenMutation, GenerateAzureAdTokenMutationVariables>;
export const RefreshAzureProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshAzureProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"RefreshAzureProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"SubscriptionID"}},{"kind":"Field","name":{"kind":"Name","value":"TenantID"}},{"kind":"Field","name":{"kind":"Name","value":"ResourceGroup"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverPostgreSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMySQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverRedis"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCosmosDB"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}}]}}]}}]} as unknown as DocumentNode<RefreshAzureProviderMutation, RefreshAzureProviderMutationVariables>;
export const GetGcpProvidersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetGCPProviders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"GCPProviders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"ServiceAccountKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCloudSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverAlloyDB"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMemorystore"}}]}}]}}]} as unknown as DocumentNode<GetGcpProvidersQuery, GetGcpProvidersQueryVariables>;
export const GetGcpProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetGCPProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"GCPProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"ServiceAccountKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCloudSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverAlloyDB"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMemorystore"}}]}}]}}]} as unknown as DocumentNode<GetGcpProviderQuery, GetGcpProviderQueryVariables>;
export const GetLocalGcpProjectsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetLocalGCPProjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"LocalGCPProjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Source"}},{"kind":"Field","name":{"kind":"Name","value":"IsDefault"}}]}}]}}]} as unknown as DocumentNode<GetLocalGcpProjectsQuery, GetLocalGcpProjectsQueryVariables>;
export const GetGcpRegionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetGCPRegions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"GCPRegions"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"Description"}}]}}]}}]} as unknown as DocumentNode<GetGcpRegionsQuery, GetGcpRegionsQueryVariables>;
export const AddGcpProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddGCPProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"GCPProviderInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"AddGCPProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"ServiceAccountKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCloudSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverAlloyDB"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMemorystore"}}]}}]}}]} as unknown as DocumentNode<AddGcpProviderMutation, AddGcpProviderMutationVariables>;
export const UpdateGcpProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateGCPProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"GCPProviderInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"UpdateGCPProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"ServiceAccountKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCloudSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverAlloyDB"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMemorystore"}}]}}]}}]} as unknown as DocumentNode<UpdateGcpProviderMutation, UpdateGcpProviderMutationVariables>;
export const TestGcpCredentialsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TestGCPCredentials"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"GCPProviderInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"TestGCPCredentials"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}]}]}}]} as unknown as DocumentNode<TestGcpCredentialsMutation, TestGcpCredentialsMutationVariables>;
export const RefreshGcpProviderDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshGCPProvider"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"RefreshGCPProvider"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Id"}},{"kind":"Field","name":{"kind":"Name","value":"ProviderType"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Region"}},{"kind":"Field","name":{"kind":"Name","value":"Status"}},{"kind":"Field","name":{"kind":"Name","value":"LastDiscoveryAt"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoveredCount"}},{"kind":"Field","name":{"kind":"Name","value":"Error"}},{"kind":"Field","name":{"kind":"Name","value":"ProjectID"}},{"kind":"Field","name":{"kind":"Name","value":"ServiceAccountKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverCloudSQL"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverAlloyDB"}},{"kind":"Field","name":{"kind":"Name","value":"DiscoverMemorystore"}}]}}]}}]} as unknown as DocumentNode<RefreshGcpProviderMutation, RefreshGcpProviderMutationVariables>;
export const GenerateCloudSqliamAuthTokenDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"GenerateCloudSQLIAMAuthToken"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"providerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"username"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"GenerateCloudSQLIAMAuthToken"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"providerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"providerID"}}},{"kind":"Argument","name":{"kind":"Name","value":"username"},"value":{"kind":"Variable","name":{"kind":"Name","value":"username"}}}]}]}}]} as unknown as DocumentNode<GenerateCloudSqliamAuthTokenMutation, GenerateCloudSqliamAuthTokenMutationVariables>;
export const SettingsConfigDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SettingsConfig"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"SettingsConfig"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"MetricsEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"CloudProvidersEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"DisableCredentialForm"}},{"kind":"Field","name":{"kind":"Name","value":"EnableNewUI"}},{"kind":"Field","name":{"kind":"Name","value":"MaxPageSize"}}]}}]}}]} as unknown as DocumentNode<SettingsConfigQuery, SettingsConfigQueryVariables>;
export const UpdateSettingsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateSettings"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"newSettings"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SettingsConfigInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"UpdateSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"newSettings"},"value":{"kind":"Variable","name":{"kind":"Name","value":"newSettings"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}}]}}]}}]} as unknown as DocumentNode<UpdateSettingsMutation, UpdateSettingsMutationVariables>;
export const AddRowDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddRow"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"values"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RecordInput"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"AddRow"},"name":{"kind":"Name","value":"AddSourceRow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"values"},"value":{"kind":"Variable","name":{"kind":"Name","value":"values"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}}]}}]}}]} as unknown as DocumentNode<AddRowMutation, AddRowMutationVariables>;
export const AddStorageUnitDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddStorageUnit"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"parent"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"storageUnit"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"fields"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RecordInput"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"AddStorageUnit"},"name":{"kind":"Name","value":"CreateSourceObject"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"parent"},"value":{"kind":"Variable","name":{"kind":"Name","value":"parent"}}},{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"storageUnit"}}},{"kind":"Argument","name":{"kind":"Name","value":"fields"},"value":{"kind":"Variable","name":{"kind":"Name","value":"fields"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}}]}}]}}]} as unknown as DocumentNode<AddStorageUnitMutation, AddStorageUnitMutationVariables>;
export const DeleteRowDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteRow"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"values"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RecordInput"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"DeleteRow"},"name":{"kind":"Name","value":"DeleteSourceRow"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"values"},"value":{"kind":"Variable","name":{"kind":"Name","value":"values"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}}]}}]}}]} as unknown as DocumentNode<DeleteRowMutation, DeleteRowMutationVariables>;
export const GetColumnsBatchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetColumnsBatch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"refs"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"ColumnsBatch"},"name":{"kind":"Name","value":"SourceColumnsBatch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"refs"},"value":{"kind":"Variable","name":{"kind":"Name","value":"refs"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"StorageUnit"},"name":{"kind":"Name","value":"Ref"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Kind"}},{"kind":"Field","name":{"kind":"Name","value":"Locator"}},{"kind":"Field","name":{"kind":"Name","value":"Path"}}]}},{"kind":"Field","name":{"kind":"Name","value":"Columns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"IsPrimary"}},{"kind":"Field","name":{"kind":"Name","value":"IsForeignKey"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedTable"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedColumn"}},{"kind":"Field","name":{"kind":"Name","value":"Length"}},{"kind":"Field","name":{"kind":"Name","value":"Precision"}},{"kind":"Field","name":{"kind":"Name","value":"Scale"}}]}}]}}]}}]} as unknown as DocumentNode<GetColumnsBatchQuery, GetColumnsBatchQueryVariables>;
export const GetStorageUnitRowsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetStorageUnitRows"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"where"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"WhereCondition"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sort"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SortCondition"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageOffset"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"Row"},"name":{"kind":"Name","value":"SourceRows"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"where"},"value":{"kind":"Variable","name":{"kind":"Name","value":"where"}}},{"kind":"Argument","name":{"kind":"Name","value":"sort"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sort"}}},{"kind":"Argument","name":{"kind":"Name","value":"pageSize"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"pageOffset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageOffset"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Columns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Type"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","name":{"kind":"Name","value":"IsPrimary"}},{"kind":"Field","name":{"kind":"Name","value":"IsForeignKey"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedTable"}},{"kind":"Field","name":{"kind":"Name","value":"ReferencedColumn"}},{"kind":"Field","name":{"kind":"Name","value":"Length"}},{"kind":"Field","name":{"kind":"Name","value":"Precision"}},{"kind":"Field","name":{"kind":"Name","value":"Scale"}}]}},{"kind":"Field","name":{"kind":"Name","value":"Rows"}},{"kind":"Field","name":{"kind":"Name","value":"DisableUpdate"}},{"kind":"Field","name":{"kind":"Name","value":"TotalCount"}}]}}]}}]} as unknown as DocumentNode<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>;
export const GetSourceContentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSourceContent"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"Content"},"name":{"kind":"Name","value":"SourceContent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Text"}},{"kind":"Field","name":{"kind":"Name","value":"MIMEType"}},{"kind":"Field","name":{"kind":"Name","value":"IsBinary"}},{"kind":"Field","name":{"kind":"Name","value":"SizeBytes"}},{"kind":"Field","name":{"kind":"Name","value":"Truncated"}},{"kind":"Field","name":{"kind":"Name","value":"FileName"}},{"kind":"Field","name":{"kind":"Name","value":"ModifiedAt"}}]}}]}}]} as unknown as DocumentNode<GetSourceContentQuery, GetSourceContentQueryVariables>;
export const GetStorageUnitsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetStorageUnits"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"parent"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"StorageUnit"},"name":{"kind":"Name","value":"SourceObjects"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"parent"},"value":{"kind":"Variable","name":{"kind":"Name","value":"parent"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Ref"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Kind"}},{"kind":"Field","name":{"kind":"Name","value":"Locator"}},{"kind":"Field","name":{"kind":"Name","value":"Path"}}]}},{"kind":"Field","name":{"kind":"Name","value":"Kind"}},{"kind":"Field","name":{"kind":"Name","value":"Name"}},{"kind":"Field","alias":{"kind":"Name","value":"Attributes"},"name":{"kind":"Name","value":"Metadata"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Key"}},{"kind":"Field","name":{"kind":"Name","value":"Value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"Actions"}},{"kind":"Field","name":{"kind":"Name","value":"HasChildren"}}]}}]}}]} as unknown as DocumentNode<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>;
export const UpdateStorageUnitDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateStorageUnit"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SourceObjectRefInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"values"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"RecordInput"}}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"updatedColumns"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"UpdateStorageUnit"},"name":{"kind":"Name","value":"UpdateSourceObject"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"values"},"value":{"kind":"Variable","name":{"kind":"Name","value":"values"}}},{"kind":"Argument","name":{"kind":"Name","value":"updatedColumns"},"value":{"kind":"Variable","name":{"kind":"Name","value":"updatedColumns"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"Status"}}]}}]}}]} as unknown as DocumentNode<UpdateStorageUnitMutation, UpdateStorageUnitMutationVariables>;