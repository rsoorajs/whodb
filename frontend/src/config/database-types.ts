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

import { ComponentType, ReactElement, createElement } from "react";
import {
    GetConnectableDatabasesDocument,
    GetConnectableDatabasesQuery,
} from "@graphql";
import { Icons } from "../components/icons";
import { getEdition } from "./edition";
import { graphqlClient } from "./graphql-client";
import { getRegisteredDatabaseTypeOverrides } from "./database-registry";

/**
 * Type category for grouping database types in the UI.
 */
export type TypeCategory = 'numeric' | 'text' | 'binary' | 'datetime' | 'boolean' | 'json' | 'other';

/**
 * SSL mode option for database connections.
 * Matches backend ssl.SSLModeInfo structure.
 */
export interface SSLModeOption {
    /** Mode value used in configuration (e.g., "required", "verify-ca") */
    value: string;
    /** Accepted aliases for this mode (e.g., PostgreSQL's "require" for "required") */
    aliases?: string[];
}

/**
 * Defines a canonical database type for use in type selectors.
 * Types are from each database's official documentation.
 */
export interface TypeDefinition {
    /** Canonical type name (e.g., "VARCHAR", "INTEGER") - stored internally */
    id: string;
    /** Display label shown in UI (e.g., "varchar", "integer") - database's preferred case */
    label: string;
    /** Shows length input when selected (VARCHAR, CHAR) */
    hasLength?: boolean;
    /** Shows precision/scale inputs when selected (DECIMAL, NUMERIC) */
    hasPrecision?: boolean;
    /** Default length value for types with hasLength */
    defaultLength?: number;
    /** Default precision for types with hasPrecision */
    defaultPrecision?: number;
    /** Type category for grouping and icon selection */
    category: TypeCategory;
    /** Function to wrap INSERT values (e.g. "TO_BITMAP") — aggregate types only */
    insertFunc?: string;
    /** Required table key model (e.g. "AGGREGATE") — aggregate types only */
    tableModel?: string;
}

/**
 * Props passed to a custom login form renderer.
 * Allows custom database types to fully control the login form fields.
 */
export interface CustomLoginFormProps {
    hostName: string;
    setHostName: (value: string) => void;
    username: string;
    setUsername: (value: string) => void;
    password: string;
    setPassword: (value: string) => void;
    advancedForm: Record<string, string>;
    setAdvancedForm: (value: Record<string, string>) => void;
}

/**
 * Dropdown item type with backend-provided connection behavior and a thin
 * frontend decoration layer.
 */
export interface IDatabaseDropdownItem {
    id: string;
    label: string;
    pluginType: string;
    icon: ReactElement;
    extra: Record<string, string>;
    fields?: {
        hostname?: boolean;
        username?: boolean;
        password?: boolean;
        database?: boolean;
        searchPath?: boolean;
    };
    requiredFields?: {
        hostname?: boolean;
        username?: boolean;
        password?: boolean;
        database?: boolean;
    };
    operators?: string[];
    /** Canonical type definitions for type selectors */
    typeDefinitions?: TypeDefinition[];
    /** Maps type aliases to canonical names (e.g., INT4 -> INTEGER) */
    aliasMap?: Record<string, string>;
    /** Whether this database supports field modifiers (primary, nullable) */
    supportsModifiers?: boolean;
    /** Whether this database supports scratchpad/raw query execution */
    supportsScratchpad?: boolean;
    /** Whether this database supports schemas */
    supportsSchema?: boolean;
    /** Whether this database supports switching between databases in the UI */
    supportsDatabaseSwitching?: boolean;
    /** Whether this database should use the schema field for graph queries */
    usesSchemaForGraph?: boolean;
    /** Whether this database type uses database selection instead of schema selection */
    usesDatabaseInsteadOfSchema?: boolean;
    /** Whether this database supports mock data generation */
    supportsMockData?: boolean;
    /** Whether this database type is an AWS managed service */
    isAwsManaged?: boolean;
    /** SSL modes supported by this database */
    sslModes?: SSLModeOption[];
    /** Optional custom login form renderer */
    customFormRenderer?: ComponentType<CustomLoginFormProps>;
}

/**
 * UI-only override for a backend-owned database catalog entry.
 */
export type DatabaseTypeOverride = Pick<IDatabaseDropdownItem, 'id'> &
    Partial<Omit<IDatabaseDropdownItem, 'id'>>;

/**
 * Filter options for database type retrieval.
 */
export interface DatabaseTypeFilterOptions {
    /** When false, AWS managed database types (ElastiCache, DocumentDB) are excluded */
    cloudProvidersEnabled?: boolean;
}

type BackendConnectableDatabase = GetConnectableDatabasesQuery['ConnectableDatabases'][number];

const shouldUseCatalogCache = !import.meta.env.DEV;

let cachedDatabaseCatalog: BackendConnectableDatabase[] = [];
let catalogCacheLoaded = false;

function getCatalogCacheKey(): string {
    return `whodb_database_catalog_${getEdition()}_${__APP_VERSION__}`;
}

function readCachedDatabaseCatalog(): BackendConnectableDatabase[] {
    if (!shouldUseCatalogCache) {
        return [];
    }

    try {
        const raw = localStorage.getItem(getCatalogCacheKey());
        if (!raw) {
            return [];
        }

        const parsed = JSON.parse(raw) as BackendConnectableDatabase[];
        return Array.isArray(parsed) ? parsed : [];
    } catch {
        return [];
    }
}

function writeCachedDatabaseCatalog(items: BackendConnectableDatabase[]): void {
    if (!shouldUseCatalogCache) {
        return;
    }

    try {
        localStorage.setItem(getCatalogCacheKey(), JSON.stringify(items));
    } catch {
        // Ignore storage failures; the in-memory cache is still valid.
    }
}

function ensureCatalogCacheLoaded(): void {
    if (catalogCacheLoaded) {
        return;
    }

    cachedDatabaseCatalog = readCachedDatabaseCatalog();
    catalogCacheLoaded = true;
}

function mapExtra(extra: BackendConnectableDatabase['extra']): Record<string, string> {
    return extra.reduce((acc, item) => {
        acc[item.Key] = item.Value;
        return acc;
    }, {} as Record<string, string>);
}

function resolveIcon(databaseType: string, pluginType: string): ReactElement {
    const logos = Icons.Logos as Record<string, ReactElement>;
    return logos[databaseType] ?? logos[pluginType] ?? createElement("span", { className: "w-6 h-6" });
}

function decorateDatabaseType(item: BackendConnectableDatabase): IDatabaseDropdownItem {
    return {
        id: item.id,
        label: item.label,
        pluginType: item.pluginType,
        icon: resolveIcon(item.id, item.pluginType),
        extra: mapExtra(item.extra),
        fields: {
            hostname: item.fields.hostname,
            username: item.fields.username,
            password: item.fields.password,
            database: item.fields.database,
            searchPath: item.fields.searchPath,
        },
        requiredFields: {
            hostname: item.requiredFields.hostname,
            username: item.requiredFields.username,
            password: item.requiredFields.password,
            database: item.requiredFields.database,
        },
        supportsModifiers: item.supportsModifiers,
        supportsScratchpad: item.supportsScratchpad,
        supportsSchema: item.supportsSchema,
        supportsDatabaseSwitching: item.supportsDatabaseSwitching,
        usesSchemaForGraph: item.usesSchemaForGraph,
        usesDatabaseInsteadOfSchema: item.usesDatabaseInsteadOfSchema,
        supportsMockData: item.supportsMockData,
        isAwsManaged: item.isAwsManaged,
        sslModes: item.sslModes.map(mode => ({
            value: mode.value,
            aliases: mode.aliases.length > 0 ? mode.aliases : undefined,
        })),
    };
}

function getDecoratedBaseDatabaseTypes(): IDatabaseDropdownItem[] {
    ensureCatalogCacheLoaded();
    return cachedDatabaseCatalog.map(decorateDatabaseType);
}

function filterDatabaseTypes(
    items: IDatabaseDropdownItem[],
    options: DatabaseTypeFilterOptions = {}
): IDatabaseDropdownItem[] {
    const { cloudProvidersEnabled = true } = options;

    if (cloudProvidersEnabled) {
        return items;
    }

    return items.filter(item => !item.isAwsManaged);
}

function mergeDatabaseTypeOverride(
    item: IDatabaseDropdownItem,
    override: DatabaseTypeOverride
): IDatabaseDropdownItem {
    return {
        ...item,
        ...override,
        extra: override.extra ? { ...item.extra, ...override.extra } : item.extra,
        fields: override.fields ? { ...item.fields, ...override.fields } : item.fields,
        requiredFields: override.requiredFields ? { ...item.requiredFields, ...override.requiredFields } : item.requiredFields,
    };
}

function withRegisteredDatabaseTypes(baseTypes: IDatabaseDropdownItem[]): IDatabaseDropdownItem[] {
    const overrides = getRegisteredDatabaseTypeOverrides();
    if (overrides.length === 0) {
        return baseTypes;
    }

    return baseTypes.map(item => {
        const override = overrides.find(candidate => candidate.id === item.id);
        return override ? mergeDatabaseTypeOverride(item, override) : item;
    });
}

/**
 * Preloads the backend catalog and returns decorated database types.
 */
export const preloadDatabaseTypeDropdownItems = async (): Promise<IDatabaseDropdownItem[]> => {
    ensureCatalogCacheLoaded();

    if (cachedDatabaseCatalog.length === 0) {
        const { data } = await graphqlClient.query<GetConnectableDatabasesQuery>({
            query: GetConnectableDatabasesDocument,
        });

        cachedDatabaseCatalog = data.ConnectableDatabases;
        catalogCacheLoaded = true;
        writeCachedDatabaseCatalog(cachedDatabaseCatalog);
    }

    return withRegisteredDatabaseTypes(getDecoratedBaseDatabaseTypes());
};

/**
 * Get all database types (backend catalog + registered extension types),
 * optionally filtered.
 *
 * @param options Filter options for database types.
 * @returns Filtered list of database types.
 */
export const getDatabaseTypeDropdownItems = async (
    options: DatabaseTypeFilterOptions = {}
): Promise<IDatabaseDropdownItem[]> => {
    const items = await preloadDatabaseTypeDropdownItems();
    return filterDatabaseTypes(items, options);
};

/**
 * Synchronous version that reads the in-memory or cached backend catalog.
 *
 * @param options Filter options for database types.
 * @returns Filtered list of database types.
 */
export const getDatabaseTypeDropdownItemsSync = (
    options: DatabaseTypeFilterOptions = {}
): IDatabaseDropdownItem[] => {
    const items = withRegisteredDatabaseTypes(getDecoratedBaseDatabaseTypes());
    return filterDatabaseTypes(items, options);
};

/**
 * Get a single database type configuration from the cached catalog.
 *
 * @param databaseType Database type identifier.
 * @returns The cached database type config, if present.
 */
export const getDatabaseTypeDropdownItemSync = (
    databaseType: string | undefined
): IDatabaseDropdownItem | undefined => {
    if (!databaseType) {
        return undefined;
    }

    return getDatabaseTypeDropdownItemsSync().find(item => item.id === databaseType);
};

/**
 * Resolve a database type to its underlying plugin type using the cached
 * backend catalog.
 *
 * @param databaseType Database type identifier.
 * @returns The resolved plugin type, or the original type when not cached.
 */
export const getResolvedDatabasePluginTypeSync = (
    databaseType: string | undefined
): string | undefined => {
    if (!databaseType) {
        return undefined;
    }

    return getDatabaseTypeDropdownItemSync(databaseType)?.pluginType ?? databaseType;
};
