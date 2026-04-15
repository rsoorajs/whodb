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

import { useMemo } from "react";
import { IDatabaseDropdownItem } from "../config/database-types";
import { useAppSelector } from "../store/hooks";
import { BackendCapabilities } from "../store/database-metadata";
import { resolveDatabaseFeatureFlags, type DatabaseFeatureFlags } from "../utils/database-features";
import {
    getDatabaseStorageUnitLabelForDatabaseType,
    isNoSQLDatabaseType,
} from "../utils/functions";
import { useDatabaseTypeDropdownItem } from "./useDatabaseCatalog";

/**
 * Fully resolved UI traits for a database type.
 */
export interface DatabaseTraits extends DatabaseFeatureFlags {
    /** Decorated catalog entry for the requested database type. */
    item?: IDatabaseDropdownItem;
    /** Resolved backend plugin type for the database. */
    pluginType?: string;
    /** Whether the catalog is still loading without cached data. */
    loading: boolean;
    /** Whether the database behaves like a NoSQL database in the UI. */
    isNoSQL: boolean;
    /** Plural storage-unit label for the database. */
    storageUnitLabel: string;
    /** Singular storage-unit label for the database. */
    singularStorageUnitLabel: string;
}

function resolveLiveCapabilities(
    metadataDatabaseType: string | null,
    pluginType: string | undefined,
    capabilities: BackendCapabilities | null
): BackendCapabilities | null {
    if (!pluginType || metadataDatabaseType !== pluginType) {
        return null;
    }

    return capabilities;
}

/**
 * Resolves the catalog-backed UI traits for a database type.
 *
 * This hook keeps the live backend metadata override for active-connection
 * capabilities, while all static traits continue to come from the shared
 * connectable database catalog.
 *
 * @param databaseType Database type identifier.
 * @returns Resolved database traits for the UI.
 */
export function useDatabaseTraits(databaseType: string | undefined): DatabaseTraits {
    const { item, loading } = useDatabaseTypeDropdownItem(databaseType);
    const metadataDatabaseType = useAppSelector(state => state.databaseMetadata.databaseType);
    const metadataCapabilities = useAppSelector(state => state.databaseMetadata.capabilities);

    return useMemo(() => {
        const pluginType = item?.pluginType ?? databaseType;
        const liveCapabilities = resolveLiveCapabilities(
            metadataDatabaseType,
            pluginType,
            metadataCapabilities
        );
        const featureFlags = resolveDatabaseFeatureFlags(item, liveCapabilities);

        return {
            item,
            pluginType,
            loading,
            isNoSQL: isNoSQLDatabaseType(databaseType, pluginType),
            storageUnitLabel: getDatabaseStorageUnitLabelForDatabaseType(databaseType, pluginType),
            singularStorageUnitLabel: getDatabaseStorageUnitLabelForDatabaseType(databaseType, pluginType, true),
            ...featureFlags,
        };
    }, [databaseType, item, loading, metadataCapabilities, metadataDatabaseType]);
}
