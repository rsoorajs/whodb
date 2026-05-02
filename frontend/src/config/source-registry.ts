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

import type { SourceTypeOverride } from './source-types';

/**
 * Source type registry for UI-only overrides supplied by extension modules.
 *
 * The backend catalog remains the source of truth for connection behavior.
 */
let registeredSourceTypeOverrides: SourceTypeOverride[] = [];

/** Register source-type UI overrides (called by extension modules at boot). */
export const registerSourceTypeOverrides = (overrides: SourceTypeOverride[]) => {
    registeredSourceTypeOverrides = overrides;
};

/** Get the registered source-type UI overrides. */
export const getRegisteredSourceTypeOverrides = (): SourceTypeOverride[] => {
    return registeredSourceTypeOverrides;
};
