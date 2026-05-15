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

import {ApolloClient, InMemoryCache} from '@apollo/client';
import {CombinedGraphQLErrors, CombinedProtocolErrors, ServerError} from '@apollo/client/errors';
import {setContext} from '@apollo/client/link/context';
import {onError} from '@apollo/client/link/error';
import {HttpLink} from '@apollo/client/link/http';
import {toast} from '@clidey/ux';
import {print} from 'graphql';
import {
    LoginSourceDocument,
    LoginSourceMutationVariables,
    LoginWithSourceProfileDocument,
    LoginWithSourceProfileMutationVariables
} from '@graphql';
import {LocalLoginProfile} from '../store/auth';
import {reduxStore} from '../store';
import {addAuthHeader} from '../utils/auth-headers';
import {isOnRoute, navigateWithBasePath, withBasePath} from '../utils/base-path';
import {isAwsHostname, isAzureHostname, isGcpHostname} from '../utils/cloud-connection-prefill';
import {getTranslation, loadTranslationsSync} from '../utils/i18n';
import {type SupportedLanguage, DEFAULT_LANGUAGE} from '../utils/languages';
import {clearSourceSessionMetadata} from '../utils/source-session-metadata-cache';

// Always use an application-relative URI so that:
// - Desktop/Wails uses the embedded router handler
// - Dev server (vite) proxies to the backend via server.proxy in vite.config.ts
// - Bundled web builds honor WHODB_BASE_PATH
const uri = withBasePath("/api/query");
const loginWithSourceProfileQuery = print(LoginWithSourceProfileDocument);
const loginMutationQuery = print(LoginSourceDocument);

/**
 * Optional hook for extensions (e.g. EE) to intercept 401 errors before the
 * default CE auto-login path runs. Return `true` if the extension handled it.
 */
let onUnauthorizedHandler: (() => Promise<boolean>) | null = null;

/** Registers an extension handler for 401 responses. */
export const registerOnUnauthorized = (fn: () => Promise<boolean>): void => {
    onUnauthorizedHandler = fn;
};

type GraphQLClientTranslationKey = 'sessionExpired' | 'autoLoginSuccess' | 'autoLoginFailed';
type TranslatorFn = (key: GraphQLClientTranslationKey) => string;

const getTranslator = (): TranslatorFn => {
    const language = (reduxStore.getState().settings.language ?? DEFAULT_LANGUAGE) as SupportedLanguage;
    const translations = loadTranslationsSync('config/graphql-client', language);
    return (key: GraphQLClientTranslationKey) => getTranslation(translations, key, language);
};

const redirectToLoginWithMessage = (
    key: GraphQLClientTranslationKey,
    translator?: TranslatorFn
) => {
    const t = translator ?? getTranslator();
    toast.error(t(key));
    navigateWithBasePath('/login');
};

const httpLink = new HttpLink({
  uri,
  credentials: "include",
});

// Add Authorization header in desktop/webview environments where cookies are not supported.
const authLink = setContext((_, prevContext) => {
    return {
        headers: addAuthHeader(prevContext.headers),
    };
});

/**
 * Global error handling for unauthorized responses.
 *
 * When a GraphQL operation returns an "unauthorized" error, this handler will:
 * 1. Check if there's a current profile stored in Redux store
 * 2. If a profile exists, automatically attempt to login using that profile
 *    - If the profile is a saved profile, use LoginWithSourceProfile mutation
 *    - Otherwise, use LoginSource mutation with credentials
 * 3. If login is successful, refresh the page to reload with correct values
 * 4. If no profile exists or login fails, redirect to the login page
 *
 * This ensures seamless user experience when sessions expire.
 */
const errorLink = onError(({error}) => {
    if (ServerError.is(error) && error.statusCode === 401) {
        if (onUnauthorizedHandler) {
            void onUnauthorizedHandler().then(handled => {
                if (!handled) {
                    fallbackAutoLogin();
                }
            });
            return;
        }
        fallbackAutoLogin();
    } else if (!CombinedGraphQLErrors.is(error) && !CombinedProtocolErrors.is(error)) {
        console.error('Network error:', error);
    }
});

function fallbackAutoLogin() {
    if (isOnRoute('/login')) {
        return;
    }

    // @ts-ignore
    const authState = reduxStore.getState().auth;
    const currentProfile = authState.current;

    if (currentProfile) {
        void handleAutoLogin(currentProfile);
    } else {
        redirectToLoginWithMessage('sessionExpired');
    }
}

/**
 * Handles automatic login using the current profile.
 */
async function handleAutoLogin(currentProfile: LocalLoginProfile) {
    const t = getTranslator();
    try {
        const settings = reduxStore.getState().settings;
        if (isAwsHostname(currentProfile.Hostname) && !settings.awsProviderEnabled) {
            return;
        }
        if (isAzureHostname(currentProfile.Hostname) && !settings.azureProviderEnabled) {
            return;
        }
        if (isGcpHostname(currentProfile.Hostname) && !settings.gcpProviderEnabled) {
            return;
        }

        let response, result;
        if (currentProfile.Saved) {
            // Login with source profile
            const variables: LoginWithSourceProfileMutationVariables = {
                profile: {
                    Id: currentProfile.Id,
                    Values: currentProfile.Database ? [{ Key: "Database", Value: currentProfile.Database }] : [],
                },
            };
            response = await fetch(uri, {
                method: 'POST',
                headers: addAuthHeader({
                    'Content-Type': 'application/json',
                }),
                credentials: 'include',
                body: JSON.stringify({
                    operationName: 'LoginWithSourceProfile',
                    query: loginWithSourceProfileQuery,
                    variables,
                }),
            });
            result = await response.json();
            if (result.data?.LoginWithSourceProfile?.Status) {
                toast.success(t('autoLoginSuccess'));
                window.location.reload();
                return;
            } else {
                redirectToLoginWithMessage('autoLoginFailed', t);
                return;
            }
        } else {
            // Normal login with credentials
            const variables: LoginSourceMutationVariables = {
                credentials: {
                    Id: currentProfile.Id,
                    SourceType: currentProfile.SourceType,
                    Values: currentProfile.Values,
                    AccessToken: currentProfile.AccessToken,
                },
            };
            response = await fetch(uri, {
                method: 'POST',
                headers: addAuthHeader({
                    'Content-Type': 'application/json',
                }),
                credentials: 'include',
                body: JSON.stringify({
                    operationName: 'LoginSource',
                    query: loginMutationQuery,
                    variables,
                }),
            });
            result = await response.json();
            if (result.data?.LoginSource?.Status) {
                toast.success(t('autoLoginSuccess'));
                window.location.reload();
                return;
            } else {
                redirectToLoginWithMessage('autoLoginFailed', t);
                return;
            }
        }
    } catch (error) {
        console.error('Auto-login error:', error);
        redirectToLoginWithMessage('autoLoginFailed', t);
    }
}

export const graphqlClient = new ApolloClient({
    link: errorLink.concat(authLink.concat(httpLink)),
  cache: new InMemoryCache(),
  defaultOptions: {
      watchQuery: {
        fetchPolicy: "cache-first",
        refetchWritePolicy: "overwrite",
      },
      query: {
        fetchPolicy: "no-cache",
      },
      mutate: {
        fetchPolicy: "no-cache",
      },
  }
});

/**
 * Clears all cached GraphQL data without refetching active queries.
 */
export async function clearGraphqlStore(): Promise<void> {
    try {
        await graphqlClient.clearStore();
    } catch (e) {}
    clearSourceSessionMetadata();
}
