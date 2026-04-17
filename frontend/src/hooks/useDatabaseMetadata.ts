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

import { useLazyQuery } from '@apollo/client/react';
import { useEffect, useCallback } from 'react';
import { GetDatabaseMetadataDocument } from '@graphql';
import { useAppSelector } from '../store/hooks';
import {
    clearDatabaseMetadata,
    setDatabaseMetadata,
    setDatabaseMetadataLoading,
    shouldRefreshDatabaseMetadata,
    useDatabaseMetadataState,
} from '../utils/database-metadata-cache';

/**
 * Hook that fetches and caches database metadata from the backend.
 *
 * Use this hook in components that need access to database metadata.
 * The metadata is automatically fetched when:
 * - User logs in (database type changes)
 * - Cache expires (5 minutes)
 * - Manual refresh is triggered
 *
 * @returns Object with metadata state and refresh function
 */
export const useDatabaseMetadata = () => {
    const auth = useAppSelector(state => state.auth);
    const metadata = useDatabaseMetadataState();
    const currentDbType = auth.current?.Type;

    const [fetchMetadata, { data, error, loading }] = useLazyQuery(GetDatabaseMetadataDocument, {
        fetchPolicy: 'network-only',
    });

    useEffect(() => {
        if (data?.DatabaseMetadata) {
            setDatabaseMetadata(data.DatabaseMetadata);
        }
    }, [data?.DatabaseMetadata]);

    useEffect(() => {
        if (error) {
            console.error('Failed to fetch database metadata:', error);
            setDatabaseMetadataLoading(false);
        }
    }, [error]);

    // Fetch metadata when database type changes or the session cache expires.
    useEffect(() => {
        if (auth.status === 'logged-in' && currentDbType) {
            if (shouldRefreshDatabaseMetadata(currentDbType)) {
                setDatabaseMetadataLoading(true);
                void fetchMetadata();
            }
        }
    }, [auth.status, currentDbType, fetchMetadata, metadata.databaseType, metadata.lastFetched]);

    // Clear metadata on logout.
    useEffect(() => {
        if (auth.status === 'unauthorized') {
            clearDatabaseMetadata();
        }
    }, [auth.status]);

    // Manual refresh function.
    const refresh = useCallback(() => {
        if (auth.status === 'logged-in') {
            setDatabaseMetadataLoading(true);
            void fetchMetadata();
        }
    }, [auth.status, fetchMetadata]);

    return {
        typeDefinitions: metadata.typeDefinitions,
        operators: metadata.operators,
        aliasMap: metadata.aliasMap,
        capabilities: metadata.capabilities,
        databaseType: metadata.databaseType,
        loading: loading || metadata.loading,
        hasFetched: metadata.lastFetched !== null,
        refresh,
    };
};
