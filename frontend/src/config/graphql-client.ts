/*
 * Copyright 2026 Clidey, Inc.
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
import {setContext} from '@apollo/client/link/context';
import {onError} from '@apollo/client/link/error';
import {toast} from '@clidey/ux';
import {print} from 'graphql';
import {
    DatabaseType,
    LoginDocument,
    LoginMutationVariables,
    LoginWithProfileDocument,
    LoginWithProfileMutationVariables
} from '@graphql';
import {LocalLoginProfile} from '../store/auth';
import {reduxStore} from '../store';
import {addAuthHeader} from '../utils/auth-headers';
import {isAwsHostname} from '../utils/cloud-connection-prefill';
import {getTranslation, loadTranslations} from '../utils/i18n';
import {withBasePath} from './base-path';

// Always use a relative URI (with base path) so that:
// - Desktop/Wails uses the embedded router handler
// - Dev server (vite) proxies to the backend via server.proxy in vite.config.ts
const uri = withBasePath("/api/query");
const loginPath = withBasePath("/login");
const loginWithProfileQuery = print(LoginWithProfileDocument);
const loginMutationQuery = print(LoginDocument);

type SupportedLanguage = 'en' | 'es';
type GraphQLClientTranslationKey = 'sessionExpired' | 'autoLoginSuccess' | 'autoLoginFailed';
type TranslatorFn = (key: GraphQLClientTranslationKey) => string;

let cachedTranslationLanguage: SupportedLanguage | undefined;
let cachedTranslationsPromise: Promise<Record<string, string>> | null = null;

const getTranslator = async (): Promise<TranslatorFn> => {
    const language = (reduxStore.getState().settings.language ?? 'en') as SupportedLanguage;
    if (!cachedTranslationsPromise || cachedTranslationLanguage !== language) {
        cachedTranslationLanguage = language;
        cachedTranslationsPromise = loadTranslations('config/graphql-client', language);
    }
    const translations = await cachedTranslationsPromise;
    return (key: GraphQLClientTranslationKey) => getTranslation(translations, key);
};

const redirectToLoginWithMessage = async (
    key: GraphQLClientTranslationKey,
    translator?: TranslatorFn
) => {
    const t = translator ?? await getTranslator();
    toast.error(t(key));
    window.location.href = loginPath;
};

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
            void handleAutoLogin(currentProfile);
        } else {
            // Don't redirect if already on login page to avoid infinite loop
            if (!window.location.pathname.startsWith(loginPath)) {
                void redirectToLoginWithMessage('sessionExpired');
            }
        }
    } else if (networkError) {
        console.error('Network error:', networkError);
    }
});

/**
 * Handles automatic login using the current profile.
 */
async function handleAutoLogin(currentProfile: LocalLoginProfile) {
    const t = await getTranslator();
    try {
        // Don't auto-login to AWS connections when cloud providers are disabled
        const cloudProvidersEnabled = reduxStore.getState().settings.cloudProvidersEnabled;
        if (isAwsHostname(currentProfile.Hostname) && !cloudProvidersEnabled) {
            return;
        }

        let response, result;
        if (currentProfile.Saved) {
            // Login with profile
            const variables: LoginWithProfileMutationVariables = {
                profile: {
                    Id: currentProfile.Id,
                    Type: currentProfile.Type as DatabaseType,
                },
            };
            response = await fetch(uri, {
                method: 'POST',
                headers: addAuthHeader({
                    'Content-Type': 'application/json',
                }),
                credentials: 'include',
                body: JSON.stringify({
                    operationName: 'LoginWithProfile',
                    query: loginWithProfileQuery,
                    variables,
                }),
            });
            result = await response.json();
            if (result.data?.LoginWithProfile?.Status) {
                toast.success(t('autoLoginSuccess'));
                window.location.reload();
                return;
            } else {
                await redirectToLoginWithMessage('autoLoginFailed', t);
                return;
            }
        } else {
            // Normal login with credentials
            const variables: LoginMutationVariables = {
                credentials: {
                    Type: currentProfile.Type,
                    Hostname: currentProfile.Hostname,
                    Database: currentProfile.Database,
                    Username: currentProfile.Username,
                    Password: currentProfile.Password,
                    Advanced: currentProfile.Advanced || [],
                },
            };
            response = await fetch(uri, {
                method: 'POST',
                headers: addAuthHeader({
                    'Content-Type': 'application/json',
                }),
                credentials: 'include',
                body: JSON.stringify({
                    operationName: 'Login',
                    query: loginMutationQuery,
                    variables,
                }),
            });
            result = await response.json();
            if (result.data?.Login?.Status) {
                toast.success(t('autoLoginSuccess'));
                window.location.reload();
                return;
            } else {
                await redirectToLoginWithMessage('autoLoginFailed', t);
                return;
            }
        }
    } catch (error) {
        console.error('Auto-login error:', error);
        await redirectToLoginWithMessage('autoLoginFailed', t);
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
