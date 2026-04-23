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

import { getSourceSessionMetadataState } from './source-session-metadata-cache';

/**
 * Get valid operators for a source type from the backend-driven Apollo store.
 *
 * @param sourceType The source type identifier
 * @returns Array of valid operators for the source
 */
export function getSourceOperators(sourceType: string): string[] {
    const metadataState = getSourceSessionMetadataState();

    if (
        metadataState.sourceType === sourceType &&
        metadataState.operators.length > 0
    ) {
        return metadataState.operators;
    }

    console.warn(
        `[source-operators] No operators found for ${sourceType}. ` +
            `Ensure SourceSessionMetadata query has completed.`
    );
    return [];
}
