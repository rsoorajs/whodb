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

import { ComponentType } from "react";

type RouteFactory = () => Promise<{ default: ComponentType<any> }>;

export type RegisteredRoute = {
    name: string;
    path: string;
    factory: RouteFactory;
};

const registrations: RegisteredRoute[] = [];

/**
 * Registers an additional route to be included in the app router.
 * Call during the extension init phase (e.g. EE register.ts) before the app boots.
 * routes.tsx reads these via getRegisteredRoutes() when building the route list.
 */
export function registerRoute(name: string, path: string, factory: RouteFactory): void {
    registrations.push({ name, path, factory });
}

export function getRegisteredRoutes(): RegisteredRoute[] {
    return registrations;
}
