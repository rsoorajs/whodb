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

import {DatabaseType} from '@graphql';
import {getDatabaseTypeDropdownItemSync} from '../config/database-types';
import {reduxStore} from '../store';

/**
 * Get backend capabilities from the Redux store.
 * Returns null if capabilities haven't been fetched yet.
 */
function getBackendCapabilities() {
    return reduxStore.getState().databaseMetadata.capabilities;
}

/**
 * Check if a database supports scratchpad/raw query execution.
 * Reads from backend capabilities first, falls back to config/hardcoded lists.
 */
export function databaseSupportsScratchpad(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }

    const capabilities = getBackendCapabilities();
    if (capabilities != null) {
        return capabilities.supportsScratchpad;
    }

    const dbConfig = getDatabaseTypeDropdownItemSync(databaseType);
    if (dbConfig?.supportsScratchpad != null) {
        return dbConfig.supportsScratchpad;
    }
    return false;
}

/**
 * Check if a database supports schemas.
 * Reads from backend capabilities first, falls back to config/hardcoded lists.
 */
export function databaseSupportsSchema(databaseType: DatabaseType | string | undefined): boolean {
    if (databaseType == null) {
        return false;
    }

    const capabilities = getBackendCapabilities();
    if (capabilities != null) {
        return capabilities.supportsSchema;
    }

    const dbConfig = getDatabaseTypeDropdownItemSync(databaseType);
    if (dbConfig?.supportsSchema != null) {
        return dbConfig.supportsSchema;
    }
    return false;
}

/**
 * Check if a database supports switching between databases in the UI.
 * Reads from backend capabilities first, falls back to config/hardcoded lists.
 */
export function databaseSupportsDatabaseSwitching(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }

    const capabilities = getBackendCapabilities();
    if (capabilities != null) {
        return capabilities.supportsDatabaseSwitch;
    }

    const dbConfig = getDatabaseTypeDropdownItemSync(databaseType);
    if (dbConfig?.supportsDatabaseSwitching !== undefined) {
        return dbConfig.supportsDatabaseSwitching;
    }
    return false;
}

/**
 * Check if a database should use the schema field for graph queries.
 */
export function databaseUsesSchemaForGraph(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return true;
    }

    const dbConfig = getDatabaseTypeDropdownItemSync(databaseType);
    if (dbConfig?.usesSchemaForGraph !== undefined) {
        return dbConfig.usesSchemaForGraph;
    }

    return true;
}

/**
 * Check if a database type uses the "database" concept instead of "schema".
 */
export function databaseTypesThatUseDatabaseInsteadOfSchema(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }

    const dbConfig = getDatabaseTypeDropdownItemSync(databaseType);
    if (dbConfig?.usesDatabaseInsteadOfSchema !== undefined) {
        return dbConfig.usesDatabaseInsteadOfSchema;
    }

    return false;
}

/**
 * Check if a database supports mock data generation.
 */
export function databaseSupportsMockData(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }

    const dbConfig = getDatabaseTypeDropdownItemSync(databaseType);
    if (dbConfig?.supportsMockData !== undefined) {
        return dbConfig.supportsMockData;
    }

    return false;
}
