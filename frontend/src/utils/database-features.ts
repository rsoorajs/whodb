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

import type { IDatabaseDropdownItem } from '../config/database-types';
import type { DatabaseMetadataCapabilities } from './database-metadata-cache';

/**
 * Catalog-backed database feature flags resolved for the current UI.
 */
export interface DatabaseFeatureFlags {
    supportsScratchpad: boolean;
    supportsSchema: boolean;
    supportsDatabaseSwitching: boolean;
    usesSchemaForGraph: boolean;
    usesDatabaseInsteadOfSchema: boolean;
    supportsMockData: boolean;
    supportsModifiers: boolean;
}

/**
 * Resolves feature flags from the backend catalog plus live backend metadata.
 *
 * The live metadata takes precedence for capabilities that depend on the
 * active connection, while the catalog remains the source of truth for static
 * database traits.
 *
 * @param item Decorated catalog entry for the database type.
 * @param capabilities Live backend capabilities for the active plugin.
 * @returns The resolved feature flags for the database type.
 */
export function resolveDatabaseFeatureFlags(
    item: IDatabaseDropdownItem | undefined,
    capabilities: DatabaseMetadataCapabilities | null
): DatabaseFeatureFlags {
    return {
        supportsScratchpad: capabilities?.supportsScratchpad ?? item?.supportsScratchpad ?? false,
        supportsSchema: capabilities?.supportsSchema ?? item?.supportsSchema ?? false,
        supportsDatabaseSwitching: capabilities?.supportsDatabaseSwitch ?? item?.supportsDatabaseSwitching ?? false,
        usesSchemaForGraph: item?.usesSchemaForGraph ?? true,
        usesDatabaseInsteadOfSchema: item?.usesDatabaseInsteadOfSchema ?? false,
        supportsMockData: item?.supportsMockData ?? false,
        supportsModifiers: capabilities?.supportsModifiers ?? item?.supportsModifiers ?? false,
    };
}
