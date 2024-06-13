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
  Postgres = 'Postgres'
}

export type LoginCredentials = {
  Hostname: Scalars['String']['input'];
  Password: Scalars['String']['input'];
  Port: Scalars['Int']['input'];
  Type: Scalars['String']['input'];
  Username: Scalars['String']['input'];
};

export type LoginResponse = {
  __typename?: 'LoginResponse';
  Status: Scalars['Boolean']['output'];
};

export type Mutation = {
  __typename?: 'Mutation';
  CreateStorageUnit: Scalars['String']['output'];
  Login: LoginResponse;
};


export type MutationCreateStorageUnitArgs = {
  type: DatabaseType;
};


export type MutationLoginArgs = {
  credentails: LoginCredentials;
};

export type Query = {
  __typename?: 'Query';
  Column: Array<Scalars['String']['output']>;
  Row: RowsResult;
  StorageUnit: Array<Scalars['String']['output']>;
};


export type QueryColumnArgs = {
  row: Scalars['String']['input'];
  storageUnit: Scalars['String']['input'];
  type: DatabaseType;
};


export type QueryRowArgs = {
  storageUnit: Scalars['String']['input'];
  type: DatabaseType;
};


export type QueryStorageUnitArgs = {
  type: DatabaseType;
};

export type RowsResult = {
  __typename?: 'RowsResult';
  Columns: Array<Column>;
  Rows: Array<Array<Scalars['String']['output']>>;
};

export type LoginMutationVariables = Exact<{
  credentails: LoginCredentials;
}>;


export type LoginMutation = { __typename?: 'Mutation', Login: { __typename?: 'LoginResponse', Status: boolean } };


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