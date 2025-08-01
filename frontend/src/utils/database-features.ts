/**
 * Copyright 2025 Clidey, Inc.
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

import { DatabaseType } from '@graphql';
import { getDatabaseTypeDropdownItemsSync } from '../config/database-types';

/**
 * Check if a database supports scratchpad/raw query execution
 * @param databaseType The database type (can be CE or EE type)
 * @returns boolean indicating if the database supports scratchpad
 */
export function databaseSupportsScratchpad(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }
    
    // Try to get scratchpad support from the database configuration first
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);
    
    if (dbConfig?.supportsScratchpad !== undefined) {
        return dbConfig.supportsScratchpad;
    }
    
    // Fall back to checking known databases that don't support scratchpad
    const databasesThatDontSupportScratchpad = [
        DatabaseType.MongoDb, 
        DatabaseType.Redis, 
        DatabaseType.ElasticSearch
    ];
    
    return !databasesThatDontSupportScratchpad.includes(databaseType as DatabaseType);
}

/**
 * Check if a database supports schemas
 * @param databaseType The database type (can be CE or EE type)
 * @returns boolean indicating if the database supports schemas
 */
export function databaseSupportsSchema(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }
    
    // Try to get schema support from the database configuration first
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);
    
    if (dbConfig?.supportsSchema !== undefined) {
        return dbConfig.supportsSchema;
    }
    
    // Fall back to checking known databases that don't support schemas
    const databasesThatDontSupportSchema = [
        DatabaseType.Sqlite3, 
        DatabaseType.Redis, 
        DatabaseType.ElasticSearch
    ];
    
    return !databasesThatDontSupportSchema.includes(databaseType as DatabaseType);
}

/**
 * Get databases that don't support scratchpad (for backward compatibility)
 * @returns Array of database types that don't support scratchpad
 */
export function getDatabasesThatDontSupportScratchpad(): DatabaseType[] {
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const databasesThatDontSupport: DatabaseType[] = [];
    
    // Check all configured databases
    dbTypeItems.forEach(dbConfig => {
        if (dbConfig.supportsScratchpad === false) {
            databasesThatDontSupport.push(dbConfig.id as DatabaseType);
        }
    });
    
    // Include default databases if not found in config
    const defaults = [DatabaseType.MongoDb, DatabaseType.Redis, DatabaseType.ElasticSearch];
    defaults.forEach(dbType => {
        if (!databasesThatDontSupport.includes(dbType)) {
            const dbConfig = dbTypeItems.find(item => item.id === dbType);
            if (!dbConfig || dbConfig.supportsScratchpad !== true) {
                databasesThatDontSupport.push(dbType);
            }
        }
    });
    
    return databasesThatDontSupport;
}

/**
 * Get databases that don't support schemas (for backward compatibility)
 * @returns Array of database types that don't support schemas
 */
export function getDatabasesThatDontSupportSchema(): DatabaseType[] {
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const databasesThatDontSupport: DatabaseType[] = [];
    
    // Check all configured databases
    dbTypeItems.forEach(dbConfig => {
        if (dbConfig.supportsSchema === false) {
            databasesThatDontSupport.push(dbConfig.id as DatabaseType);
        }
    });
    
    // Include default databases if not found in config
    const defaults = [DatabaseType.Sqlite3, DatabaseType.Redis, DatabaseType.ElasticSearch];
    defaults.forEach(dbType => {
        if (!databasesThatDontSupport.includes(dbType)) {
            const dbConfig = dbTypeItems.find(item => item.id === dbType);
            if (!dbConfig || dbConfig.supportsSchema !== true) {
                databasesThatDontSupport.push(dbType);
            }
        }
    });
    
    return databasesThatDontSupport;
}