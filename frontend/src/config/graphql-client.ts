/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {ApolloClient, createHttpLink, InMemoryCache} from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import {onError} from '@apollo/client/link/error';
import {toast} from '@clidey/ux';
import {reduxStore} from '../store';
import { addAuthHeader } from '../utils/auth-headers';

// Always use a relative URI so that:
// - Desktop/Wails uses the embedded router handler
// - Dev server (vite) proxies to the backend via server.proxy in vite.config.ts
const uri = "/api/query";

const httpLink = createHttpLink({
  uri,
  credentials: "include",
});

// Add Authorization header in desktop/webview environments where cookies are not supported.
const authLink = setContext((_, { headers }) => {
  return {
    headers: addAuthHeader(headers)
  };
});

/**
 * Global error handling for unauthorized responses.
 *
 * When a GraphQL operation returns an "unauthorized" error, this handler will:
 * 1. Check if there's a current profile stored in Redux store
 * 2. If a profile exists, automatically attempt to login using that profile
 *    - If the profile is a saved profile, use LoginWithProfile mutation
 *    - Otherwise, use Login mutation with credentials
 * 3. If login is successful, refresh the page to reload with correct values
 * 4. If no profile exists or login fails, redirect to the login page
 *
 * This ensures seamless user experience when sessions expire.
 */
const errorLink = onError(({networkError}) => {
    if (networkError && 'statusCode' in networkError && networkError.statusCode === 401) {
        // @ts-ignore
        const authState = reduxStore.getState().auth;
        const currentProfile = authState.current;

        if (currentProfile) {
            handleAutoLogin(currentProfile);
        } else {
            toast.error("Session expired. Please login again.");
            window.location.href = '/login';
        }
    } else if (networkError) {
        console.error('Network error:', networkError);
    }
});

/**
 * Handles automatic login using the current profile.
 *
 * If the profile is a saved profile, use LoginWithProfile mutation.
 * Otherwise, use Login mutation with credentials.
 *
 * @param currentProfile - The current user profile from Redux store
 */
async function handleAutoLogin(currentProfile: any) {
    try {
        let response, result;
        if (currentProfile.Saved) {
            // Login with profile
            response = await fetch(uri, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                credentials: 'include',
                body: JSON.stringify({
                    operationName: 'LoginWithProfile',
                    query: `
            mutation LoginWithProfile($profile: LoginProfileInput!) {
              LoginWithProfile(profile: $profile) {
                Status
              }
            }
          `,
                    variables: {
                        profile: {
                            Id: currentProfile.Id,
                            Type: currentProfile.Type,
                        },
                    },
                }),
            });
            result = await response.json();
            if (result.data?.LoginWithProfile?.Status) {
                toast.success("Automatically re-authenticated");
                window.location.reload();
                return;
            } else {
                toast.error("Auto-login failed. Please login manually.");
                window.location.href = '/login';
                return;
            }
        } else {
            // Normal login with credentials
            response = await fetch(uri, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                credentials: 'include',
                body: JSON.stringify({
                    operationName: 'Login',
                    query: `
            mutation Login($credentials: LoginCredentials!) {
              Login(credentials: $credentials) {
                Status
              }
            }
          `,
                    variables: {
                        credentials: {
                            Type: currentProfile.Type,
                            Hostname: currentProfile.Hostname,
                            Database: currentProfile.Database,
                            Username: currentProfile.Username,
                            Password: currentProfile.Password,
                            Advanced: currentProfile.Advanced || [],
                        },
                    },
                }),
            });
            result = await response.json();
            if (result.data?.Login?.Status) {
                toast.success("Automatically re-authenticated");
                window.location.reload();
                return;
            } else {
                toast.error("Auto-login failed. Please login manually.");
                window.location.href = '/login';
                return;
            }
        }
    } catch (error) {
        console.error('Auto-login error:', error);
        toast.error("Auto-login failed. Please login manually.");
        window.location.href = '/login';
    }
}

export const graphqlClient = new ApolloClient({
    link: errorLink.concat(authLink.concat(httpLink)),
  cache: new InMemoryCache(),
  defaultOptions: {
      query: {
        fetchPolicy: "no-cache",
      },
      mutate: {
        fetchPolicy: "no-cache",
      },
  }
});
