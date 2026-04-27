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

/**
 * AI provider selection persistence registry.
 *
 * CE defines the hook point, EE registers the implementation.
 * This allows EE to persist AI selections to the platform store without CE knowing about EE internals.
 */

export type AIPersistencePayload = {
    providerId: string | null;
    model: string | null;
};

type AIPersistenceHandler = (payload: AIPersistencePayload) => void;

let persistenceHandler: AIPersistenceHandler | null = null;

/**
 * Registers a handler for persisting AI provider selection.
 * Called by EE at boot to inject platform store persistence.
 */
export function registerAIPersistenceHandler(handler: AIPersistenceHandler): void {
    persistenceHandler = handler;
}

/**
 * Persists AI provider selection using the registered handler.
 * Called by CE when user changes AI provider/model.
 */
export function persistAISelection(payload: AIPersistencePayload): void {
    persistenceHandler?.(payload);
}
