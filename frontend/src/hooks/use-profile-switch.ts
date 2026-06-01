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

import { useMutation } from '@apollo/client/react';
import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { toast } from '@clidey/ux';
import { LoginSourceDocument, LoginWithSourceProfileDocument } from '@graphql';
import { useAppDispatch } from '@/store/hooks';
import type { LocalLoginProfile } from '@/store/auth';
import { AuthActions } from '@/store/auth';
import { DatabaseActions } from '@/store/database';
import { updateProfileLastAccessed } from '@/components/profile-info-tooltip';
import { InternalRoutes } from '@/config/routes';
import { clearGraphqlStore } from '@/config/graphql-client';

interface UseProfileSwitchOptions {
    onSuccess?: () => void;
    onError?: (error: string) => void;
    errorMessage?: string;
}

/**
 * Shared hook for switching between profiles.
 * Handles both backend-known profiles (saved/environment-defined) and local profiles.
 *
 * Backend-known profiles: Uses LoginWithSourceProfile mutation (AWS, config, env vars)
 * Local profiles: Uses LoginSource mutation with full credentials
 */
export const useProfileSwitch = (options?: UseProfileSwitchOptions) => {
    const dispatch = useAppDispatch();
    const navigate = useNavigate();
    const [login, { loading: loginLoading }] = useMutation(LoginSourceDocument);
    const [loginWithSourceProfile, { loading: loginWithSourceProfileLoading }] = useMutation(LoginWithSourceProfileDocument);

    const loading = loginLoading || loginWithSourceProfileLoading;

    const switchProfile = useCallback(async (profile: LocalLoginProfile, database?: string) => {
        const targetDatabase = database ?? profile.Database;

        // Clear schema before switching
        dispatch(DatabaseActions.setSchema(""));

        // Use LoginWithSourceProfile for saved/environment-defined profiles (backend knows about them)
        // Use LoginSource for local profiles (only stored in frontend)
        try {
            const switchSucceeded = profile.Saved || profile.IsEnvironmentDefined
                ? (await loginWithSourceProfile({
                    variables: {
                        profile: {
                            Id: profile.Id,
                            Values: targetDatabase ? [{ Key: "Database", Value: targetDatabase }] : [],
                        }
                    },
                })).data?.LoginWithSourceProfile.Status
                : (await login({
                    variables: {
                        credentials: {
                            Id: profile.Id,
                            SourceType: profile.SourceType,
                            Values: profile.Database === targetDatabase
                                ? profile.Values
                                : profile.Values.map(value =>
                                    value.Key === "Database" ? { ...value, Value: targetDatabase } : value
                                  ).concat(
                                    profile.Values.some(value => value.Key === "Database")
                                      ? []
                                      : [{ Key: "Database", Value: targetDatabase }]
                                  ),
                            AccessToken: profile.AccessToken,
                        }
                    },
                })).data?.LoginSource.Status;

            if (!switchSucceeded) {
                const errorMsg = options?.errorMessage ?? 'Failed to switch profile';
                toast.error(errorMsg);
                options?.onError?.(errorMsg);
                return;
            }

            updateProfileLastAccessed(profile.Id);
            await clearGraphqlStore();
            if (database) {
                dispatch(AuthActions.setLoginProfileDatabase({ id: profile.Id, database }));
            }
            dispatch(DatabaseActions.setSchema(""));
            dispatch(AuthActions.switch({ id: profile.Id }));
            navigate(InternalRoutes.Dashboard.StorageUnit.path, {
                state: {},
            });
            options?.onSuccess?.();
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            const errorMsg = `${options?.errorMessage ?? 'Failed to switch profile'}: ${errorMessage}`;
            toast.error(errorMsg);
            options?.onError?.(errorMessage);
        }
    }, [dispatch, login, loginWithSourceProfile, navigate, options]);

    return {
        switchProfile,
        loading,
    };
};
