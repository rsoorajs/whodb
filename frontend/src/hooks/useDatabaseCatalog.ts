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

import type { ApolloError } from "@apollo/client";
import { useEffect, useMemo, useState } from "react";
import { useGetConnectableDatabasesQuery } from "@graphql";
import {
    BackendConnectableDatabase,
    DatabaseTypeFilterOptions,
    findDatabaseTypeDropdownItem,
    IDatabaseDropdownItem,
    readCachedDatabaseCatalog,
    resolveDatabasePluginType,
    resolveDatabaseTypeDropdownItems,
    writeCachedDatabaseCatalog,
} from "../config/database-types";

/**
 * Result of resolving the connectable database catalog for UI consumers.
 */
export interface UseDatabaseTypeDropdownItemsResult {
    /** Decorated, filtered database picker items. */
    items: IDatabaseDropdownItem[];
    /** Whether the live catalog query is still loading without any cached data. */
    loading: boolean;
    /** Query error, if the live fetch failed. */
    error?: ApolloError;
}

/**
 * Result of resolving a single connectable database catalog item.
 */
export interface UseDatabaseTypeDropdownItemResult extends UseDatabaseTypeDropdownItemsResult {
    /** Matching catalog item for the requested database type. */
    item?: IDatabaseDropdownItem;
}

/**
 * Loads the connectable database catalog with a React/Apollo-owned lifecycle.
 *
 * The backend remains the source of truth. The frontend only reuses the
 * version-scoped local cache as initial data between app launches.
 *
 * @param options Optional UI filters for the returned database list.
 * @returns Decorated database type items plus loading/error state.
 */
export function useDatabaseTypeDropdownItems(
    options: DatabaseTypeFilterOptions = {}
): UseDatabaseTypeDropdownItemsResult {
    const [cachedCatalog] = useState<BackendConnectableDatabase[]>(() => readCachedDatabaseCatalog());
    const { data, loading, error } = useGetConnectableDatabasesQuery({
        fetchPolicy: "cache-first",
    });
    const cloudProvidersEnabled = options.cloudProvidersEnabled;

    useEffect(() => {
        if (data?.ConnectableDatabases) {
            writeCachedDatabaseCatalog(data.ConnectableDatabases);
        }
    }, [data?.ConnectableDatabases]);

    const items = useMemo(() => {
        const catalog = data?.ConnectableDatabases ?? cachedCatalog;
        return resolveDatabaseTypeDropdownItems(catalog, { cloudProvidersEnabled });
    }, [cachedCatalog, cloudProvidersEnabled, data?.ConnectableDatabases]);

    return {
        items,
        loading: loading && data?.ConnectableDatabases == null && cachedCatalog.length === 0,
        error: error ?? undefined,
    };
}

/**
 * Resolves a single database catalog item by id.
 *
 * @param databaseType Database type identifier.
 * @param options Optional UI filters for catalog resolution.
 * @returns Matching database item plus loading/error state.
 */
export function useDatabaseTypeDropdownItem(
    databaseType: string | undefined,
    options: DatabaseTypeFilterOptions = {}
): UseDatabaseTypeDropdownItemResult {
    const result = useDatabaseTypeDropdownItems(options);

    const item = useMemo(() => {
        return findDatabaseTypeDropdownItem(result.items, databaseType);
    }, [databaseType, result.items]);

    return {
        ...result,
        item,
    };
}

/**
 * Resolves a displayed database type to its underlying plugin type.
 *
 * @param databaseType Database type identifier.
 * @param options Optional UI filters for catalog resolution.
 * @returns The resolved plugin type, or the original type if no catalog entry is available.
 */
export function useResolvedDatabasePluginType(
    databaseType: string | undefined,
    options: DatabaseTypeFilterOptions = {}
): string | undefined {
    const { item } = useDatabaseTypeDropdownItem(databaseType, options);
    return resolveDatabasePluginType(databaseType, item);
}
