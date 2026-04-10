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

import React from 'react';

/**
 * Component registry for dynamically loaded components.
 *
 * Extensions register components at boot.
 * CE code renders from the registry — if a component isn't registered, it's not shown.
 */
const registry = new Map<string, React.LazyExoticComponent<any>>();

/** Register a lazy-loaded component by name. Called by extension modules at boot. */
export const registerComponent = (name: string, loader: () => Promise<{ default: any }>) => {
    registry.set(name, React.lazy(loader));
};

/** Get a registered component by name. Returns undefined if not registered (CE build). */
export const getComponent = (name: string): React.LazyExoticComponent<any> | undefined => {
    return registry.get(name);
};

/** Check if a component is registered. */
export const hasComponent = (name: string): boolean => {
    return registry.has(name);
};
