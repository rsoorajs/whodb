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

import { SourceAction } from '@graphql';
import type { SourceTypeItem } from '../config/source-types';

/**
 * Catalog-backed source contract flags resolved for the current UI.
 */
export interface SourceContractFlags {
    supportsChat: boolean;
    supportsGraph: boolean;
    supportsScratchpad: boolean;
    supportsSchema: boolean;
    supportsDatabaseSwitching: boolean;
    usesSchemaForGraph: boolean;
    usesDatabaseInsteadOfSchema: boolean;
    supportsMockData: boolean;
    supportsImportData: boolean;
    supportsModifiers: boolean;
}

/**
 * Resolves feature flags from the catalog-derived source contract.
 *
 * @param item Decorated catalog entry for the database type.
 * @returns The resolved feature flags for the database type.
 */
export function resolveSourceContractFlags(
    item: SourceTypeItem | undefined
): SourceContractFlags {
    return {
        supportsChat: item?.supportsChat ?? false,
        supportsGraph: item?.supportsGraph ?? false,
        supportsScratchpad: item?.supportsScratchpad ?? false,
        supportsSchema: item?.supportsSchema ?? false,
        supportsDatabaseSwitching: item?.supportsDatabaseSwitching ?? false,
        usesSchemaForGraph: item?.usesSchemaForGraph ?? true,
        usesDatabaseInsteadOfSchema: item?.usesDatabaseInsteadOfSchema ?? false,
        supportsMockData: item?.supportsMockData ?? false,
        supportsImportData: item?.contract?.ObjectTypes.some(objectType => objectType.Actions.includes(SourceAction.ImportData)) ?? false,
        supportsModifiers: item?.supportsModifiers ?? false,
    };
}
