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
import {ServerError} from '@apollo/client/errors';
import {SetContextLink} from '@apollo/client/link/context';
import {ErrorLink} from '@apollo/client/link/error';
import {HttpLink} from '@apollo/client/link/http';
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
import {withBasePath} from '../utils/base-path';
import {isAwsHostname} from '../utils/cloud-connection-prefill';
import {getTranslation, loadTranslationsSync} from '../utils/i18n';
import {type SupportedLanguage, DEFAULT_LANGUAGE} from '../utils/languages';

// Always use an application-relative URI so that:
// - Desktop/Wails uses the embedded router handler
// - Dev server (vite) proxies to the backend via server.proxy in vite.config.ts
// - Bundled web builds honor WHODB_BASE_PATH
const uri = withBasePath("/api/query");
const loginWithProfileQuery = print(LoginWithProfileDocument);
const loginMutationQuery = print(LoginDocument);

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
    window.location.href = withBasePath('/login');
};

const httpLink = new HttpLink({
  uri,
  credentials: "include",
});

// Add Authorization header in desktop/webview environments where cookies are not supported.
const authLink = new SetContextLink((prevContext) => {
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
 *    - If the profile is a saved profile, use LoginWithProfile mutation
 *    - Otherwise, use Login mutation with credentials
 * 3. If login is successful, refresh the page to reload with correct values
 * 4. If no profile exists or login fails, redirect to the login page
 *
 * This ensures seamless user experience when sessions expire.
 */
const errorLink = new ErrorLink(({error}) => {
    if (ServerError.is(error) && error.statusCode === 401) {
        // @ts-ignore
        const authState = reduxStore.getState().auth;
        const currentProfile = authState.current;

        if (currentProfile) {
            void handleAutoLogin(currentProfile);
        } else {
            // Don't redirect if already on login page to avoid infinite loop
            if (!window.location.pathname.startsWith(withBasePath('/login'))) {
                redirectToLoginWithMessage('sessionExpired');
            }
        }
    } else {
        console.error('Network error:', error);
    }
});

/**
 * Handles automatic login using the current profile.
 */
async function handleAutoLogin(currentProfile: LocalLoginProfile) {
    const t = getTranslator();
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
                redirectToLoginWithMessage('autoLoginFailed', t);
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
    await graphqlClient.clearStore();
}
