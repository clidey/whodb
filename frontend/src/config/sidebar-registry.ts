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

import type { ReactElement } from "react";

export type RegisteredSidebarItem = {
    /** Unique key (used as React key). */
    name: string;
    /** Label shown next to the icon. */
    title: string;
    /** Route path to navigate to. */
    path: string;
    /** Icon element (e.g. Heroicon, 16x16). */
    icon: ReactElement;
    /** If true, the item is shown but not clickable. */
    disabled?: boolean;
};

const items: RegisteredSidebarItem[] = [];

/** Register a sidebar navigation item. Call during the extension init phase before the app boots. */
export function registerSidebarItem(item: RegisteredSidebarItem): void {
    items.push(item);
}

/** Returns all registered sidebar items (order = registration order). */
export function getRegisteredSidebarItems(): RegisteredSidebarItem[] {
    return items;
}
