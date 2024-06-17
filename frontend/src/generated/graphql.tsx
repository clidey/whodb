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

export type AuthResponse = {
  __typename?: 'AuthResponse';
  Status: Scalars['Boolean']['output'];
};

export type Column = {
  __typename?: 'Column';
  Name: Scalars['String']['output'];
  Type: Scalars['String']['output'];
};

export enum DatabaseType {
  Postgres = 'Postgres'
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
  Database: Scalars['String']['input'];
  Hostname: Scalars['String']['input'];
  Password: Scalars['String']['input'];
  Type: Scalars['String']['input'];
  Username: Scalars['String']['input'];
};

export type Mutation = {
  __typename?: 'Mutation';
  CreateStorageUnit: StorageUnit;
  Login: AuthResponse;
  Logout: AuthResponse;
};


export type MutationCreateStorageUnitArgs = {
  type: DatabaseType;
};


export type MutationLoginArgs = {
  credentails: LoginCredentials;
};

export type Query = {
  __typename?: 'Query';
  Graph: Array<GraphUnit>;
  RawExecute: RowsResult;
  Row: RowsResult;
  StorageUnit: Array<StorageUnit>;
};


export type QueryGraphArgs = {
  type: DatabaseType;
};


export type QueryRawExecuteArgs = {
  query: Scalars['String']['input'];
  type: DatabaseType;
};


export type QueryRowArgs = {
  pageOffset: Scalars['Int']['input'];
  pageSize: Scalars['Int']['input'];
  storageUnit: Scalars['String']['input'];
  type: DatabaseType;
  where: Scalars['String']['input'];
};


export type QueryStorageUnitArgs = {
  type: DatabaseType;
};

export type Record = {
  __typename?: 'Record';
  Key: Scalars['String']['output'];
  Value: Scalars['String']['output'];
};

export type RowsResult = {
  __typename?: 'RowsResult';
  Columns: Array<Column>;
  Rows: Array<Array<Scalars['String']['output']>>;
};

export type StorageUnit = {
  __typename?: 'StorageUnit';
  Attributes: Array<Record>;
  Name: Scalars['String']['output'];
};

export type LoginMutationVariables = Exact<{
  credentails: LoginCredentials;
}>;


export type LoginMutation = { __typename?: 'Mutation', Login: { __typename?: 'AuthResponse', Status: boolean } };

export type LogoutMutationVariables = Exact<{ [key: string]: never; }>;


export type LogoutMutation = { __typename?: 'Mutation', Logout: { __typename?: 'AuthResponse', Status: boolean } };

export type GetGraphQueryVariables = Exact<{
  type: DatabaseType;
}>;


export type GetGraphQuery = { __typename?: 'Query', Graph: Array<{ __typename?: 'GraphUnit', Unit: { __typename?: 'StorageUnit', Name: string, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }, Relations: Array<{ __typename?: 'GraphUnitRelationship', Name: string, Relationship: GraphUnitRelationshipType }> }> };

export type RawExecuteQueryVariables = Exact<{
  type: DatabaseType;
  query: Scalars['String']['input'];
}>;


export type RawExecuteQuery = { __typename?: 'Query', RawExecute: { __typename?: 'RowsResult', Rows: Array<Array<string>>, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } };

export type GetStorageUnitRowsQueryVariables = Exact<{
  type: DatabaseType;
  storageUnit: Scalars['String']['input'];
  where: Scalars['String']['input'];
  pageSize: Scalars['Int']['input'];
  pageOffset: Scalars['Int']['input'];
}>;


export type GetStorageUnitRowsQuery = { __typename?: 'Query', Row: { __typename?: 'RowsResult', Rows: Array<Array<string>>, Columns: Array<{ __typename?: 'Column', Type: string, Name: string }> } };

export type GetStorageUnitsQueryVariables = Exact<{
  type: DatabaseType;
}>;


export type GetStorageUnitsQuery = { __typename?: 'Query', StorageUnit: Array<{ __typename?: 'StorageUnit', Name: string, Attributes: Array<{ __typename?: 'Record', Key: string, Value: string }> }> };


export const LoginDocument = gql`
    mutation Login($credentails: LoginCredentials!) {
  Login(credentails: $credentails) {
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
 *      credentails: // value for 'credentails'
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
    query GetGraph($type: DatabaseType!) {
  Graph(type: $type) {
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
export const GetStorageUnitRowsDocument = gql`
    query GetStorageUnitRows($type: DatabaseType!, $storageUnit: String!, $where: String!, $pageSize: Int!, $pageOffset: Int!) {
  Row(
    type: $type
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
    query GetStorageUnits($type: DatabaseType!) {
  StorageUnit(type: $type) {
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