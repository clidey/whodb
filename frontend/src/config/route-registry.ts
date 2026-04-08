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

import { FC, ReactNode } from "react";

/** A single navigable route in the application. */
export type IInternalRoute = {
    name: string;
    path: string;
    component: ReactNode;
    public?: boolean;
};

let _eeRoutes: IInternalRoute[] = [];

/**
 * Called by EE during module initialisation to register EE-specific routes.
 * Must be called before the first React render. Routes with the same path as
 * a CE route will override the CE route in getRoutes().
 */
export function registerEERoutes(routes: IInternalRoute[]): void {
    _eeRoutes = routes;
}

/** Returns the currently registered EE routes. */
export function getEERoutes(): IInternalRoute[] {
    return _eeRoutes;
}

/** A provider component that wraps all authenticated route content. */
export type EEWrapper = FC<{ children: ReactNode }>;

let _eeWrapper: EEWrapper | null = null;

/**
 * Called by EE to register a provider that wraps all authenticated routes.
 * Must be called before the first React render (i.e. during initEE()).
 */
export function registerEEWrapper(wrapper: EEWrapper): void {
    _eeWrapper = wrapper;
}

/** Returns the registered EE wrapper, or null in CE mode. */
export function getEEWrapper(): EEWrapper | null {
    return _eeWrapper;
}
