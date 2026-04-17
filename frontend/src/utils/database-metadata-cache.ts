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

import { makeVar } from '@apollo/client';
import { useReactiveVar } from '@apollo/client/react';
import { GetDatabaseMetadataQuery } from '@graphql';
import { TypeDefinition } from '../config/database-types';

type DatabaseMetadataPayload = NonNullable<GetDatabaseMetadataQuery['DatabaseMetadata']>;

/**
 * Backend-declared capability flags for the active database plugin.
 */
export type DatabaseMetadataCapabilities = DatabaseMetadataPayload['capabilities'];

/**
 * Session-scoped database metadata stored in Apollo state.
 */
export interface DatabaseMetadataState {
    /** The database type this metadata belongs to. */
    databaseType: string | null;
    /** Canonical type definitions for the active database. */
    typeDefinitions: TypeDefinition[];
    /** Valid operators for the active database. */
    operators: string[];
    /** Alias map for normalizing type names. */
    aliasMap: Record<string, string>;
    /** Capability flags declared by the backend plugin. */
    capabilities: DatabaseMetadataCapabilities | null;
    /** Timestamp of the last successful fetch. */
    lastFetched: number | null;
    /** Whether metadata is currently being fetched. */
    loading: boolean;
}

/**
 * Cache duration for database metadata (5 minutes).
 */
export const METADATA_CACHE_DURATION = 5 * 60 * 1000;

function createInitialDatabaseMetadataState(): DatabaseMetadataState {
    return {
        databaseType: null,
        typeDefinitions: [],
        operators: [],
        aliasMap: {},
        capabilities: null,
        lastFetched: null,
        loading: false,
    };
}

function mapTypeDefinitions(typeDefinitions: DatabaseMetadataPayload['typeDefinitions']): TypeDefinition[] {
    return typeDefinitions.map(typeDefinition => ({
        id: typeDefinition.id,
        label: typeDefinition.label,
        hasLength: typeDefinition.hasLength || undefined,
        hasPrecision: typeDefinition.hasPrecision || undefined,
        defaultLength: typeDefinition.defaultLength ?? undefined,
        defaultPrecision: typeDefinition.defaultPrecision ?? undefined,
        category: typeDefinition.category,
    }));
}

function mapAliasMap(aliasMap: DatabaseMetadataPayload['aliasMap']): Record<string, string> {
    return aliasMap.reduce((acc, item) => {
        acc[item.Key] = item.Value;
        return acc;
    }, {} as Record<string, string>);
}

const databaseMetadataStateVar = makeVar<DatabaseMetadataState>(createInitialDatabaseMetadataState());

/**
 * Reads the current session-scoped database metadata snapshot.
 *
 * @returns Current Apollo-backed database metadata state.
 */
export function getDatabaseMetadataState(): DatabaseMetadataState {
    return databaseMetadataStateVar();
}

/**
 * Subscribes a component to Apollo-backed database metadata updates.
 *
 * @returns Current Apollo-backed database metadata state.
 */
export function useDatabaseMetadataState(): DatabaseMetadataState {
    return useReactiveVar(databaseMetadataStateVar);
}

/**
 * Updates the in-memory loading flag for database metadata.
 *
 * @param loading Whether a metadata request is in flight.
 */
export function setDatabaseMetadataLoading(loading: boolean): void {
    const currentState = databaseMetadataStateVar();

    if (currentState.loading === loading) {
        return;
    }

    databaseMetadataStateVar({
        ...currentState,
        loading,
    });
}

/**
 * Writes a fresh database metadata payload into Apollo session state.
 *
 * @param metadata GraphQL metadata payload returned by the backend.
 */
export function setDatabaseMetadata(metadata: DatabaseMetadataPayload): void {
    databaseMetadataStateVar({
        databaseType: metadata.databaseType,
        typeDefinitions: mapTypeDefinitions(metadata.typeDefinitions),
        operators: metadata.operators,
        aliasMap: mapAliasMap(metadata.aliasMap),
        capabilities: metadata.capabilities,
        lastFetched: Date.now(),
        loading: false,
    });
}

/**
 * Clears the Apollo-backed database metadata snapshot.
 */
export function clearDatabaseMetadata(): void {
    databaseMetadataStateVar(createInitialDatabaseMetadataState());
}

/**
 * Determines whether the current session metadata should be refetched.
 *
 * @param currentDbType Active database type for the current session.
 * @returns True when metadata is missing, stale, or for a different database.
 */
export function shouldRefreshDatabaseMetadata(currentDbType: string): boolean {
    const state = databaseMetadataStateVar();

    if (state.loading) {
        return false;
    }

    if (state.databaseType !== currentDbType) {
        return true;
    }

    if (!state.lastFetched) {
        return true;
    }

    return Date.now() - state.lastFetched > METADATA_CACHE_DURATION;
}
