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

import { SourceObjectKind, type SourceObjectRefInput } from '@graphql';
import type { SourceTypeItem } from '../config/source-types';
import type { LocalLoginProfile } from '../store/auth';

function buildPathThroughIndex(
    item: SourceTypeItem | undefined,
    profile: LocalLoginProfile | undefined,
    endIndex: number,
    schema?: string,
): string[] | undefined {
    if (!item?.contract || endIndex < 0) {
        return [];
    }

    const path: string[] = [];
    for (let index = 0; index <= endIndex; index += 1) {
        const kind = item.contract.BrowsePath[index];
        if (kind == null) {
            return undefined;
        }

        if (kind === SourceObjectKind.Database) {
            if (!profile?.Database) {
                return undefined;
            }
            path.push(profile.Database);
            continue;
        }

        if (kind === SourceObjectKind.Schema) {
            if (!schema) {
                return undefined;
            }
            path.push(schema);
            continue;
        }

        return undefined;
    }

    return path;
}

function buildRefForKind(
    item: SourceTypeItem | undefined,
    profile: LocalLoginProfile | undefined,
    kind: SourceObjectKind | undefined | null,
    schema?: string,
): SourceObjectRefInput | undefined {
    if (!item?.contract || kind == null) {
        return undefined;
    }

    const targetIndex = item.contract.BrowsePath.indexOf(kind);
    if (targetIndex < 0) {
        return undefined;
    }

    const path = buildPathThroughIndex(item, profile, targetIndex, schema);
    if (path == null) {
        return undefined;
    }

    return {
        Kind: kind,
        Path: path,
    };
}

function buildParentKind(
    item: SourceTypeItem | undefined,
    pathLength: number
): SourceObjectKind | undefined {
    if (!item?.contract || pathLength <= 0) {
        return undefined;
    }

    const maxParentIndex = item.contract.BrowsePath.length - 2;
    if (maxParentIndex < 0) {
        return undefined;
    }

    let parentIndex = pathLength - 1;
    if (parentIndex > maxParentIndex) {
        parentIndex = maxParentIndex;
    }
    if (parentIndex < 0 || parentIndex >= item.contract.BrowsePath.length) {
        return undefined;
    }

    return item.contract.BrowsePath[parentIndex];
}

/**
 * Builds the parent ref for the current source's default browsed object kind.
 *
 * @param item Active source catalog item.
 * @param profile Active source profile.
 * @param schema Selected schema when applicable.
 * @returns The parent ref used to list browsed objects.
 */
export function buildSourceParentRef(
    item: SourceTypeItem | undefined,
    profile: LocalLoginProfile | undefined,
    schema?: string,
): SourceObjectRefInput | undefined {
    if (!item?.contract) {
        return undefined;
    }

    const defaultIndex = item.contract.BrowsePath.indexOf(item.contract.DefaultObjectKind);
    if (defaultIndex <= 0) {
        return undefined;
    }

    return buildRefForKind(item, profile, item.contract.BrowsePath[defaultIndex - 1], schema);
}

/**
 * Builds the ref for a concrete browsed object.
 *
 * @param item Active source catalog item.
 * @param profile Active source profile.
 * @param schema Selected schema when applicable.
 * @param objectName Object name to append to the browse path.
 * @returns The resolved source object ref.
 */
export function buildSourceObjectRef(
    item: SourceTypeItem | undefined,
    profile: LocalLoginProfile | undefined,
    schema: string | undefined,
    objectName: string,
): SourceObjectRefInput {
    const parent = buildSourceParentRef(item, profile, schema);
    return {
        Kind: item?.contract?.DefaultObjectKind ?? SourceObjectKind.Table,
        Path: [...(parent?.Path ?? []), objectName],
    };
}

/**
 * Builds the parent ref for one concrete source object ref.
 *
 * @param item Active source catalog item.
 * @param ref Source object ref to move one level up from.
 * @returns Parent ref for the provided object, or undefined at the root.
 */
export function buildSourceParentObjectRef(
    item: SourceTypeItem | undefined,
    ref: Pick<SourceObjectRefInput, 'Kind' | 'Path' | 'Locator'>
): SourceObjectRefInput | undefined {
    if (ref.Path.length <= 1) {
        return undefined;
    }

    const parentPath = ref.Path.slice(0, -1);
    return {
        Kind: buildParentKind(item, parentPath.length) ?? ref.Kind,
        Path: parentPath,
    };
}

/**
 * Builds the scope ref used for graph/chat-like operations.
 *
 * @param item Active source catalog item.
 * @param profile Active source profile.
 * @param schema Selected schema when applicable.
 * @returns The graph/chat scope ref for the current source.
 */
export function buildSourceScopeRef(
    item: SourceTypeItem | undefined,
    profile: LocalLoginProfile | undefined,
    schema?: string,
): SourceObjectRefInput | undefined {
    return buildRefForKind(item, profile, item?.contract?.GraphScopeKind, schema)
        ?? buildSourceParentRef(item, profile, schema);
}

/**
 * Builds the query input for listing schema-like containers for the current source.
 *
 * @param item Active source catalog item.
 * @param profile Active source profile.
 * @returns Parent ref and kind filter for schema listing.
 */
export function buildSourceSchemaQuery(
    item: SourceTypeItem | undefined,
    profile: LocalLoginProfile | undefined,
): { parent?: SourceObjectRefInput; kinds: SourceObjectKind[] } {
    if (!item?.contract) {
        return { kinds: [] };
    }

    const schemaIndex = item.contract.BrowsePath.indexOf(SourceObjectKind.Schema);
    if (schemaIndex < 0) {
        return { kinds: [] };
    }
    if (schemaIndex === 0) {
        return { kinds: [SourceObjectKind.Schema] };
    }

    const path = buildPathThroughIndex(item, profile, schemaIndex - 1);
    if (path == null) {
        return { kinds: [] };
    }

    return {
        parent: {
            Kind: item.contract.BrowsePath[schemaIndex - 1] as SourceObjectKind,
            Path: path,
        },
        kinds: [SourceObjectKind.Schema],
    };
}

/**
 * Returns the last path segment from a source ref.
 *
 * @param ref Source ref-like object.
 * @returns The terminal object name from the ref path.
 */
export function getObjectNameFromRef(ref: { Path: string[] }): string {
    return ref.Path[ref.Path.length - 1] ?? '';
}
