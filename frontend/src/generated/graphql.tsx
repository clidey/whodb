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
};

export type AiChatMessage = {
  __typename?: 'AIChatMessage';
  Result?: Maybe<RowsResult>;
  Text: Scalars['String']['output'];
  Type: Scalars['String']['output'];
};

export type AiProvider = {
  __typename?: 'AIProvider';
  IsEnvironmentDefined: Scalars['Boolean']['output'];
  ProviderId: Scalars['String']['output'];
  Type: Scalars['String']['output'];
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

export type Column = {
  __typename?: 'Column';
  Name: Scalars['String']['output'];
  Type: Scalars['String']['output'];
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

export type GraphUnit = {
  __typename?: 'GraphUnit';
  Relations: Array<GraphUnitRelationship>;
  Unit: StorageUnit;
};

export type GraphUnitRelationship = {
  __typename?: 'GraphUnitRelationship';
  Name: Scalars['String']['output'];
  Relationship: GraphUnitRelationshipType;
};

export enum GraphUnitRelationshipType {
  ManyToMany = 'ManyToMany',
  ManyToOne = 'ManyToOne',
  OneToMany = 'OneToMany',
  OneToOne = 'OneToOne',
  Unknown = 'Unknown'
}

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
  Id: Scalars['String']['output'];
  IsEnvironmentDefined: Scalars['Boolean']['output'];
  Source: Scalars['String']['output'];
  Type: DatabaseType;
};

export type LoginProfileInput = {
  Database?: InputMaybe<Scalars['String']['input']>;
  Id: Scalars['String']['input'];
  Type: DatabaseType;
};

export type MockDataGenerationInput = {
  Method: Scalars['String']['input'];
  OverwriteExisting: Scalars['Boolean']['input'];
  RowCount: Scalars['Int']['input'];
  Schema: Scalars['String']['input'];
  StorageUnit: Scalars['String']['input'];
};

export type MockDataGenerationStatus = {
  __typename?: 'MockDataGenerationStatus';
  AmountGenerated: Scalars['Int']['output'];
};

export type Mutation = {
  __typename?: 'Mutation';
  AddRow: StatusResponse;
  AddStorageUnit: StatusResponse;
  DeleteRow: StatusResponse;
  GenerateMockData: MockDataGenerationStatus;
  Login: StatusResponse;
  LoginWithProfile: StatusResponse;
  Logout: StatusResponse;
  UpdateSettings: StatusResponse;
  UpdateStorageUnit: StatusResponse;
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


export type MutationGenerateMockDataArgs = {
  input: MockDataGenerationInput;
};


export type MutationLoginArgs = {
  credentials: LoginCredentials;
};


export type MutationLoginWithProfileArgs = {
  profile: LoginProfileInput;
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
  Columns: Array<Column>;
  Database: Array<Scalars['String']['output']>;
  Graph: Array<GraphUnit>;
  MockDataMaxRowCount: Scalars['Int']['output'];
  Profiles: Array<LoginProfile>;
  RawExecute: RowsResult;
  Row: RowsResult;
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


export type QueryColumnsArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
};


export type QueryDatabaseArgs = {
  type: Scalars['String']['input'];
};


export type QueryGraphArgs = {
  schema: Scalars['String']['input'];
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
};

export type SettingsConfig = {
  __typename?: 'SettingsConfig';
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


export type GetProfilesQuery = { __typename?: 'Query', Profiles: Array<{ __typename?: 'LoginProfile', Alias?: string | null, Id: string, Type: DatabaseType, Database?: string | null, IsEnvironmentDefined: boolean, Source: string }> };

export type GetSchemaQueryVariables = Exact<{ [key: string]: never; }>;


export type GetSchemaQuery = { __typename?: 'Query', Schema: Array<string> };

export type GetVersionQueryVariables = Exact<{ [key: string]: never; }>;


export type GetVersionQuery = { __typename?: 'Query', Version: string };

export type MockDataMaxRowCountQueryVariables = Exact<{ [key: string]: never; }>;


export type MockDataMaxRowCountQuery = { __typename?: 'Query', MockDataMaxRowCount: number };

export type GenerateMockDataMutationVariables = Exact<{
  input: MockDataGenerationInput;
}>;


export type GenerateMockDataMutation = { __typename?: 'Mutation', GenerateMockData: { __typename?: 'MockDataGenerationStatus', AmountGenerated: number } };

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


export type GetAiProvidersQuery = { __typename?: 'Query', AIProviders: Array<{ __typename?: 'AIProvider', Type: string, ProviderId: string, IsEnvironmentDefined: boolean }> };

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

export type GetGraphQueryVariables = Exact<{
  schema: Scalars['String']['input'];
}>;


export type GetGraphQuery = { __typename?: 'Query', Graph: Array<{ __typename?: 'GraphUnit', Unit: { __typename?: 'StorageUnit', Name: string, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }, Relations: Array<{ __typename?: 'GraphUnitRelationship', Name: string, Relationship: GraphUnitRelationshipType }> }> };

export type ColumnsQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
}>;


export type ColumnsQuery = { __typename?: 'Query', Columns: Array<{ __typename?: 'Column', Name: string, Type: string }> };

export type RawExecuteQueryVariables = Exact<{
  query: Scalars['String']['input'];
}>;


export type RawExecuteQuery = { __typename?: 'Query', RawExecute: { __typename?: 'RowsResult', Rows: Array<Array<string>>, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } };

export type SettingsConfigQueryVariables = Exact<{ [key: string]: never; }>;


export type SettingsConfigQuery = { __typename?: 'Query', SettingsConfig: { __typename?: 'SettingsConfig', MetricsEnabled?: boolean | null } };

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

export type GetStorageUnitRowsQueryVariables = Exact<{
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  where?: InputMaybe<WhereCondition>;
  sort?: InputMaybe<Array<SortCondition> | SortCondition>;
  pageSize: Scalars['Int']['input'];
  pageOffset: Scalars['Int']['input'];
}>;


export type GetStorageUnitRowsQuery = { __typename?: 'Query', Row: { __typename?: 'RowsResult', Rows: Array<Array<string>>, DisableUpdate: boolean, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } };

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
    Database
    IsEnvironmentDefined
    Source
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
    ProviderId
    IsEnvironmentDefined
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
export const SettingsConfigDocument = gql`
    query SettingsConfig {
  SettingsConfig {
    MetricsEnabled
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
    }
    Rows
    DisableUpdate
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