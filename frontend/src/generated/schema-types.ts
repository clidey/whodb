export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Upload: { input: unknown; output: unknown; }
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
  Icon?: Maybe<Scalars['String']['output']>;
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
  DiscoverS3: Scalars['Boolean']['output'];
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
  DiscoverS3?: InputMaybe<Scalars['Boolean']['input']>;
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
  MetadataFidelity: SourceMetadataFidelity;
  Name: Scalars['String']['output'];
  Precision?: Maybe<Scalars['Int']['output']>;
  ReferencedColumn?: Maybe<Scalars['String']['output']>;
  ReferencedTable?: Maybe<Scalars['String']['output']>;
  Scale?: Maybe<Scalars['Int']['output']>;
  Type: Scalars['String']['output'];
};

export type ColumnCreationCapabilities = {
  __typename?: 'ColumnCreationCapabilities';
  CheckMinMax: Scalars['Boolean']['output'];
  CheckValues: Scalars['Boolean']['output'];
  CompositePrimaryKey: Scalars['Boolean']['output'];
  DefaultValue: Scalars['Boolean']['output'];
  ForeignKey: Scalars['Boolean']['output'];
  Identity: Scalars['Boolean']['output'];
  Nullable: Scalars['Boolean']['output'];
  PrimaryKey: Scalars['Boolean']['output'];
  Types: Scalars['Boolean']['output'];
  Unique: Scalars['Boolean']['output'];
};

export type ColumnCreationLabels = {
  __typename?: 'ColumnCreationLabels';
  CheckMax: Scalars['String']['output'];
  CheckMin: Scalars['String']['output'];
  CheckValues: Scalars['String']['output'];
  DefaultValue: Scalars['String']['output'];
  ForeignKey: Scalars['String']['output'];
  Identity: Scalars['String']['output'];
  Nullable: Scalars['String']['output'];
  PrimaryKey: Scalars['String']['output'];
  Unique: Scalars['String']['output'];
};

export type ColumnDefinitionInput = {
  CheckMax?: InputMaybe<Scalars['Float']['input']>;
  CheckMin?: InputMaybe<Scalars['Float']['input']>;
  CheckValues?: InputMaybe<Array<Scalars['String']['input']>>;
  DefaultValue?: InputMaybe<Scalars['String']['input']>;
  ForeignKey?: InputMaybe<ForeignKeyDefinitionInput>;
  Identity: Scalars['Boolean']['input'];
  Name: Scalars['String']['input'];
  Nullable?: InputMaybe<Scalars['Boolean']['input']>;
  Primary: Scalars['Boolean']['input'];
  Type: Scalars['String']['input'];
  Unique: Scalars['Boolean']['input'];
};

export enum ConnectionStatus {
  Available = 'Available',
  Deleting = 'Deleting',
  Failed = 'Failed',
  Starting = 'Starting',
  Stopped = 'Stopped',
  Unknown = 'Unknown'
}

export type CreationOptionDefinition = {
  __typename?: 'CreationOptionDefinition';
  Key: Scalars['String']['output'];
  Label: Scalars['String']['output'];
  Required: Scalars['Boolean']['output'];
  Values: Array<Scalars['String']['output']>;
};

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

export type ForeignKeyDefinition = {
  __typename?: 'ForeignKeyDefinition';
  Column: Scalars['String']['output'];
  Table: Scalars['String']['output'];
};

export type ForeignKeyDefinitionInput = {
  Column: Scalars['String']['input'];
  Table: Scalars['String']['input'];
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
  SessionId?: InputMaybe<Scalars['String']['input']>;
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
  MetadataFidelity: SourceMetadataFidelity;
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
  CreateSourceObjectFromDefinition: StatusResponse;
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
  TestSourceConnection: StatusResponse;
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


export type MutationCreateSourceObjectFromDefinitionArgs = {
  definition: SourceObjectDefinitionInput;
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


export type MutationTestSourceConnectionArgs = {
  credentials: SourceLoginInput;
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

export type ObjectCreationMetadata = {
  __typename?: 'ObjectCreationMetadata';
  ColumnCapabilities: ColumnCreationCapabilities;
  ColumnLabels: ColumnCreationLabels;
  ObjectKind: SourceObjectKind;
  RequiresColumns: Scalars['Boolean']['output'];
  Supported: Scalars['Boolean']['output'];
  TableCapabilities: TableCreationCapabilities;
  TableOptions: Array<CreationOptionDefinition>;
  TypeDefinitions: Array<TypeDefinition>;
};

export type OperationWhereCondition = {
  Children: Array<WhereCondition>;
};

export type Query = {
  __typename?: 'Query';
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
  SourceFieldConstraints: Array<SourceFieldConstraints>;
  SourceFieldOptions: Array<Scalars['String']['output']>;
  SourceGraph: Array<GraphUnit>;
  SourceObject?: Maybe<SourceObject>;
  SourceObjectCreationMetadata: ObjectCreationMetadata;
  SourceObjects: Array<SourceObject>;
  SourceProfiles: Array<SourceProfile>;
  SourceQuerySuggestions: Array<SourceQuerySuggestion>;
  SourceRows: RowsResult;
  SourceSessionMetadata?: Maybe<SourceSessionMetadata>;
  SourceTypes: Array<SourceType>;
  UpdateInfo: UpdateInfo;
  Version: Scalars['String']['output'];
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


export type QuerySourceFieldConstraintsArgs = {
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


export type QuerySourceObjectCreationMetadataArgs = {
  parent?: InputMaybe<SourceObjectRefInput>;
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
  AWSProviderEnabled: Scalars['Boolean']['output'];
  AzureProviderEnabled: Scalars['Boolean']['output'];
  CloudProvidersEnabled: Scalars['Boolean']['output'];
  DisableCredentialForm: Scalars['Boolean']['output'];
  EnableNewUI: Scalars['Boolean']['output'];
  GCPProviderEnabled: Scalars['Boolean']['output'];
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

export type SourceFieldConstraints = {
  __typename?: 'SourceFieldConstraints';
  AllowedValues: Array<Scalars['String']['output']>;
  CheckMax?: Maybe<Scalars['Float']['output']>;
  CheckMin?: Maybe<Scalars['Float']['output']>;
  DefaultValue?: Maybe<Scalars['String']['output']>;
  ForeignKey?: Maybe<ForeignKeyDefinition>;
  Identity: Scalars['Boolean']['output'];
  Length?: Maybe<Scalars['Int']['output']>;
  MetadataFidelity: SourceMetadataFidelity;
  Name: Scalars['String']['output'];
  Nullable?: Maybe<Scalars['Boolean']['output']>;
  Precision?: Maybe<Scalars['Int']['output']>;
  Primary: Scalars['Boolean']['output'];
  Scale?: Maybe<Scalars['Int']['output']>;
  Type: Scalars['String']['output'];
  Unique: Scalars['Boolean']['output'];
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

export enum SourceMetadataFidelity {
  Driver = 'Driver',
  Exact = 'Exact',
  Inferred = 'Inferred',
  Sampled = 'Sampled',
  Synthetic = 'Synthetic',
  Unknown = 'Unknown',
  Unsupported = 'Unsupported'
}

export type SourceMetadataTraits = {
  __typename?: 'SourceMetadataTraits';
  Columns: SourceMetadataFidelity;
  Constraints: SourceMetadataFidelity;
  Graph: SourceMetadataFidelity;
  SystemObjectFiltering: SourceMetadataFidelity;
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

export type SourceObjectDefinitionInput = {
  Columns: Array<ColumnDefinitionInput>;
  Name: Scalars['String']['input'];
  TableOptions?: InputMaybe<Array<RecordInput>>;
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
  SupportsMultiStatement: Scalars['Boolean']['output'];
  SupportsScripts: Scalars['Boolean']['output'];
  SupportsSqlImport: Scalars['Boolean']['output'];
  SupportsStreaming: Scalars['Boolean']['output'];
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
  Metadata: SourceMetadataTraits;
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

export type TableCreationCapabilities = {
  __typename?: 'TableCreationCapabilities';
  ClusteringKey: Scalars['Boolean']['output'];
  KeyValueType: Scalars['Boolean']['output'];
  OrderKey: Scalars['Boolean']['output'];
  PartitionKey: Scalars['Boolean']['output'];
  RequiresPrimaryKey: Scalars['Boolean']['output'];
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
