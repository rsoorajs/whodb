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
import { DatabaseType, LoginDocument, LoginWithProfileDocument } from '@graphql';
import { useAppDispatch } from '@/store/hooks';
import { AuthActions, LocalLoginProfile } from '@/store/auth';
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
 * Backend-known profiles: Uses LoginWithProfile mutation (AWS, config, env vars)
 * Local profiles: Uses Login mutation with full credentials
 */
export const useProfileSwitch = (options?: UseProfileSwitchOptions) => {
    const dispatch = useAppDispatch();
    const navigate = useNavigate();
    const [login, { loading: loginLoading }] = useMutation(LoginDocument);
    const [loginWithProfile, { loading: loginWithProfileLoading }] = useMutation(LoginWithProfileDocument);

    const loading = loginLoading || loginWithProfileLoading;

    const switchProfile = useCallback(async (profile: LocalLoginProfile, database?: string) => {
        const targetDatabase = database ?? profile.Database;

        // Clear schema before switching
        dispatch(DatabaseActions.setSchema(""));

        // Use LoginWithProfile for saved/environment-defined profiles (backend knows about them)
        // Use Login for local profiles (only stored in frontend)
        try {
            const switchSucceeded = profile.Saved || profile.IsEnvironmentDefined
                ? (await loginWithProfile({
                    variables: {
                        profile: {
                            Id: profile.Id,
                            Type: profile.Type as DatabaseType,
                            Database: targetDatabase,
                        }
                    },
                })).data?.LoginWithProfile.Status
                : (await login({
                    variables: {
                        credentials: {
                            Type: profile.Type,
                            Database: targetDatabase,
                            Hostname: profile.Hostname,
                            Password: profile.Password,
                            Username: profile.Username,
                            Advanced: profile.Advanced,
                        }
                    },
                })).data?.Login.Status;

            if (!switchSucceeded) {
                const errorMsg = options?.errorMessage ?? 'Failed to switch profile';
                toast.error(errorMsg);
                options?.onError?.(errorMsg);
                return;
            }

            updateProfileLastAccessed(profile.Id);
            if (database) {
                dispatch(AuthActions.setLoginProfileDatabase({ id: profile.Id, database }));
            }
            await clearGraphqlStore();
            dispatch(DatabaseActions.setSchema(""));
            dispatch(AuthActions.switch({ id: profile.Id }));
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
            options?.onSuccess?.();
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            const errorMsg = `${options?.errorMessage ?? 'Failed to switch profile'}: ${errorMessage}`;
            toast.error(errorMsg);
            options?.onError?.(errorMessage);
        }
    }, [login, loginWithProfile, dispatch, navigate, options]);

    return {
        switchProfile,
        loading,
    };
};
