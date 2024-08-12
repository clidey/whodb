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

export type Column = {
  __typename?: 'Column';
  Name: Scalars['String']['output'];
  Type: Scalars['String']['output'];
};

export enum DatabaseType {
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
  Database?: Maybe<Scalars['String']['output']>;
  Id: Scalars['String']['output'];
  Type: DatabaseType;
};

export type LoginProfileInput = {
  Database?: InputMaybe<Scalars['String']['input']>;
  Id: Scalars['String']['input'];
  Type: DatabaseType;
};

export type Mutation = {
  __typename?: 'Mutation';
  AddRow: StatusResponse;
  AddStorageUnit: StatusResponse;
  DeleteRow: StatusResponse;
  Login: StatusResponse;
  LoginWithProfile: StatusResponse;
  Logout: StatusResponse;
  UpdateStorageUnit: StatusResponse;
};


export type MutationAddRowArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  type: DatabaseType;
  values: Array<RecordInput>;
};


export type MutationAddStorageUnitArgs = {
  fields: Array<RecordInput>;
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  type: DatabaseType;
};


export type MutationDeleteRowArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  type: DatabaseType;
  values: Array<RecordInput>;
};


export type MutationLoginArgs = {
  credentials: LoginCredentials;
};


export type MutationLoginWithProfileArgs = {
  profile: LoginProfileInput;
};


export type MutationUpdateStorageUnitArgs = {
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  type: DatabaseType;
  values: Array<RecordInput>;
};

export type Query = {
  __typename?: 'Query';
  Database: Array<Scalars['String']['output']>;
  Graph: Array<GraphUnit>;
  Profiles: Array<LoginProfile>;
  RawExecute: RowsResult;
  Row: RowsResult;
  Schema: Array<Scalars['String']['output']>;
  StorageUnit: Array<StorageUnit>;
};


export type QueryDatabaseArgs = {
  type: DatabaseType;
};


export type QueryGraphArgs = {
  schema: Scalars['String']['input'];
  type: DatabaseType;
};


export type QueryRawExecuteArgs = {
  query: Scalars['String']['input'];
  type: DatabaseType;
};


export type QueryRowArgs = {
  pageOffset: Scalars['Int']['input'];
  pageSize: Scalars['Int']['input'];
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  type: DatabaseType;
  where: Scalars['String']['input'];
};


export type QuerySchemaArgs = {
  type: DatabaseType;
};


export type QueryStorageUnitArgs = {
  schema: Scalars['String']['input'];
  type: DatabaseType;
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

export type StatusResponse = {
  __typename?: 'StatusResponse';
  Status: Scalars['Boolean']['output'];
};

export type StorageUnit = {
  __typename?: 'StorageUnit';
  Attributes: Array<Record>;
  Name: Scalars['String']['output'];
};

export type GetProfilesQueryVariables = Exact<{ [key: string]: never; }>;


export type GetProfilesQuery = { __typename?: 'Query', Profiles: Array<{ __typename?: 'LoginProfile', Id: string, Type: DatabaseType }> };

export type GetSchemaQueryVariables = Exact<{
  type: DatabaseType;
}>;


export type GetSchemaQuery = { __typename?: 'Query', Schema: Array<string> };

export type GetDatabaseQueryVariables = Exact<{
  type: DatabaseType;
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

export type GetGraphQueryVariables = Exact<{
  type: DatabaseType;
  schema: Scalars['String']['input'];
}>;


export type GetGraphQuery = { __typename?: 'Query', Graph: Array<{ __typename?: 'GraphUnit', Unit: { __typename?: 'StorageUnit', Name: string, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }, Relations: Array<{ __typename?: 'GraphUnitRelationship', Name: string, Relationship: GraphUnitRelationshipType }> }> };

export type RawExecuteQueryVariables = Exact<{
  type: DatabaseType;
  query: Scalars['String']['input'];
}>;


export type RawExecuteQuery = { __typename?: 'Query', RawExecute: { __typename?: 'RowsResult', Rows: Array<Array<string>>, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } };

export type AddRowMutationVariables = Exact<{
  type: DatabaseType;
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput> | RecordInput;
}>;


export type AddRowMutation = { __typename?: 'Mutation', AddRow: { __typename?: 'StatusResponse', Status: boolean } };

export type AddStorageUnitMutationVariables = Exact<{
  type: DatabaseType;
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  fields: Array<RecordInput> | RecordInput;
}>;


export type AddStorageUnitMutation = { __typename?: 'Mutation', AddStorageUnit: { __typename?: 'StatusResponse', Status: boolean } };

export type DeleteRowMutationVariables = Exact<{
  type: DatabaseType;
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput> | RecordInput;
}>;


export type DeleteRowMutation = { __typename?: 'Mutation', DeleteRow: { __typename?: 'StatusResponse', Status: boolean } };

export type GetStorageUnitRowsQueryVariables = Exact<{
  type: DatabaseType;
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  where: Scalars['String']['input'];
  pageSize: Scalars['Int']['input'];
  pageOffset: Scalars['Int']['input'];
}>;


export type GetStorageUnitRowsQuery = { __typename?: 'Query', Row: { __typename?: 'RowsResult', Rows: Array<Array<string>>, DisableUpdate: boolean, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } };

export type GetStorageUnitsQueryVariables = Exact<{
  type: DatabaseType;
  schema: Scalars['String']['input'];
}>;


export type GetStorageUnitsQuery = { __typename?: 'Query', StorageUnit: Array<{ __typename?: 'StorageUnit', Name: string, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };

export type UpdateStorageUnitMutationVariables = Exact<{
  type: DatabaseType;
  schema: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  values: Array<RecordInput> | RecordInput;
}>;


export type UpdateStorageUnitMutation = { __typename?: 'Mutation', UpdateStorageUnit: { __typename?: 'StatusResponse', Status: boolean } };


export const GetProfilesDocument = gql`
    query GetProfiles {
  Profiles {
    Id
    Type
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
export function useGetProfilesSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetProfilesQuery, GetProfilesQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetProfilesQuery, GetProfilesQueryVariables>(GetProfilesDocument, options);
        }
export type GetProfilesQueryHookResult = ReturnType<typeof useGetProfilesQuery>;
export type GetProfilesLazyQueryHookResult = ReturnType<typeof useGetProfilesLazyQuery>;
export type GetProfilesSuspenseQueryHookResult = ReturnType<typeof useGetProfilesSuspenseQuery>;
export type GetProfilesQueryResult = Apollo.QueryResult<GetProfilesQuery, GetProfilesQueryVariables>;
export const GetSchemaDocument = gql`
    query GetSchema($type: DatabaseType!) {
  Schema(type: $type)
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
 *      type: // value for 'type'
 *   },
 * });
 */
export function useGetSchemaQuery(baseOptions: Apollo.QueryHookOptions<GetSchemaQuery, GetSchemaQueryVariables> & ({ variables: GetSchemaQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetSchemaQuery, GetSchemaQueryVariables>(GetSchemaDocument, options);
      }
export function useGetSchemaLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetSchemaQuery, GetSchemaQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetSchemaQuery, GetSchemaQueryVariables>(GetSchemaDocument, options);
        }
export function useGetSchemaSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetSchemaQuery, GetSchemaQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetSchemaQuery, GetSchemaQueryVariables>(GetSchemaDocument, options);
        }
export type GetSchemaQueryHookResult = ReturnType<typeof useGetSchemaQuery>;
export type GetSchemaLazyQueryHookResult = ReturnType<typeof useGetSchemaLazyQuery>;
export type GetSchemaSuspenseQueryHookResult = ReturnType<typeof useGetSchemaSuspenseQuery>;
export type GetSchemaQueryResult = Apollo.QueryResult<GetSchemaQuery, GetSchemaQueryVariables>;
export const GetDatabaseDocument = gql`
    query GetDatabase($type: DatabaseType!) {
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
export function useGetDatabaseSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetDatabaseQuery, GetDatabaseQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
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
export const GetGraphDocument = gql`
    query GetGraph($type: DatabaseType!, $schema: String!) {
  Graph(type: $type, schema: $schema) {
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
 *      type: // value for 'type'
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
export function useGetGraphSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetGraphQuery, GetGraphQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetGraphQuery, GetGraphQueryVariables>(GetGraphDocument, options);
        }
export type GetGraphQueryHookResult = ReturnType<typeof useGetGraphQuery>;
export type GetGraphLazyQueryHookResult = ReturnType<typeof useGetGraphLazyQuery>;
export type GetGraphSuspenseQueryHookResult = ReturnType<typeof useGetGraphSuspenseQuery>;
export type GetGraphQueryResult = Apollo.QueryResult<GetGraphQuery, GetGraphQueryVariables>;
export const RawExecuteDocument = gql`
    query RawExecute($type: DatabaseType!, $query: String!) {
  RawExecute(type: $type, query: $query) {
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
 *      type: // value for 'type'
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
export function useRawExecuteSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<RawExecuteQuery, RawExecuteQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<RawExecuteQuery, RawExecuteQueryVariables>(RawExecuteDocument, options);
        }
export type RawExecuteQueryHookResult = ReturnType<typeof useRawExecuteQuery>;
export type RawExecuteLazyQueryHookResult = ReturnType<typeof useRawExecuteLazyQuery>;
export type RawExecuteSuspenseQueryHookResult = ReturnType<typeof useRawExecuteSuspenseQuery>;
export type RawExecuteQueryResult = Apollo.QueryResult<RawExecuteQuery, RawExecuteQueryVariables>;
export const AddRowDocument = gql`
    mutation AddRow($type: DatabaseType!, $schema: String!, $storageUnit: String!, $values: [RecordInput!]!) {
  AddRow(type: $type, schema: $schema, storageUnit: $storageUnit, values: $values) {
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
 *      type: // value for 'type'
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
    mutation AddStorageUnit($type: DatabaseType!, $schema: String!, $storageUnit: String!, $fields: [RecordInput!]!) {
  AddStorageUnit(
    type: $type
    schema: $schema
    storageUnit: $storageUnit
    fields: $fields
  ) {
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
 *      type: // value for 'type'
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
    mutation DeleteRow($type: DatabaseType!, $schema: String!, $storageUnit: String!, $values: [RecordInput!]!) {
  DeleteRow(
    type: $type
    schema: $schema
    storageUnit: $storageUnit
    values: $values
  ) {
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
 *      type: // value for 'type'
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
    query GetStorageUnitRows($type: DatabaseType!, $schema: String!, $storageUnit: String!, $where: String!, $pageSize: Int!, $pageOffset: Int!) {
  Row(
    type: $type
    schema: $schema
    storageUnit: $storageUnit
    where: $where
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
 *      type: // value for 'type'
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      where: // value for 'where'
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
export function useGetStorageUnitRowsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>(GetStorageUnitRowsDocument, options);
        }
export type GetStorageUnitRowsQueryHookResult = ReturnType<typeof useGetStorageUnitRowsQuery>;
export type GetStorageUnitRowsLazyQueryHookResult = ReturnType<typeof useGetStorageUnitRowsLazyQuery>;
export type GetStorageUnitRowsSuspenseQueryHookResult = ReturnType<typeof useGetStorageUnitRowsSuspenseQuery>;
export type GetStorageUnitRowsQueryResult = Apollo.QueryResult<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>;
export const GetStorageUnitsDocument = gql`
    query GetStorageUnits($type: DatabaseType!, $schema: String!) {
  StorageUnit(type: $type, schema: $schema) {
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
 *      type: // value for 'type'
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
export function useGetStorageUnitsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>(GetStorageUnitsDocument, options);
        }
export type GetStorageUnitsQueryHookResult = ReturnType<typeof useGetStorageUnitsQuery>;
export type GetStorageUnitsLazyQueryHookResult = ReturnType<typeof useGetStorageUnitsLazyQuery>;
export type GetStorageUnitsSuspenseQueryHookResult = ReturnType<typeof useGetStorageUnitsSuspenseQuery>;
export type GetStorageUnitsQueryResult = Apollo.QueryResult<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>;
export const UpdateStorageUnitDocument = gql`
    mutation UpdateStorageUnit($type: DatabaseType!, $schema: String!, $storageUnit: String!, $values: [RecordInput!]!) {
  UpdateStorageUnit(
    type: $type
    schema: $schema
    storageUnit: $storageUnit
    values: $values
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
 *      type: // value for 'type'
 *      schema: // value for 'schema'
 *      storageUnit: // value for 'storageUnit'
 *      values: // value for 'values'
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