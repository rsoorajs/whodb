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
import { SourceTypeItem } from "../config/source-types";
import { resolveSourceContractFlags, type SourceContractFlags } from "../utils/source-contract-flags";
import {
    getSourceObjectLabelForType,
    isNoSQLSourceType,
} from "../utils/functions";
import { useSourceTypeItem } from "./useSourceCatalog";

/**
 * Fully resolved UI traits for a source type.
 */
export interface SourceContractState extends SourceContractFlags {
    /** Decorated catalog entry for the requested source type. */
    item?: SourceTypeItem;
    /** Resolved backend connector id for the source. */
    connector?: string;
    /** Whether the catalog is still loading without cached data. */
    loading: boolean;
    /** Whether the source behaves like a NoSQL database in the UI. */
    isNoSQL: boolean;
    /** Plural storage-unit label for the source. */
    storageUnitLabel: string;
    /** Singular storage-unit label for the source. */
    singularStorageUnitLabel: string;
}

/**
 * Resolves the catalog-backed UI traits for a source type.
 *
 * @param sourceType Source type identifier.
 * @returns Resolved source traits for the UI.
 */
export function useSourceContract(sourceType: string | undefined): SourceContractState {
    const { item, loading } = useSourceTypeItem(sourceType);

    return useMemo(() => {
        const connector = item?.connector ?? sourceType;
        const featureFlags = resolveSourceContractFlags(item);

        return {
            item,
            connector,
            loading,
            isNoSQL: isNoSQLSourceType(sourceType, connector),
            storageUnitLabel: getSourceObjectLabelForType(sourceType, connector),
            singularStorageUnitLabel: getSourceObjectLabelForType(sourceType, connector, true),
            ...featureFlags,
        };
    }, [sourceType, item, loading]);
}
