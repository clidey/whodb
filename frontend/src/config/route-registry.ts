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

import type { ComponentType, LazyExoticComponent } from "react";
import { lazy } from "react";

type RouteFactory = () => Promise<{ default: ComponentType<any> }>;

export type RouteOptions = {
    scoped?: boolean;
};

export type RegisteredRoute = {
    name: string;
    path: string;
    /** Stable lazy component created once at registration time. */
    lazyComponent: LazyExoticComponent<ComponentType<any>>;
    /** If true, this route is scoped under /:orgSlug/:projectSlug */
    scoped?: boolean;
};

const registrations: RegisteredRoute[] = [];
const publicRegistrations: RegisteredRoute[] = [];

/**
 * Registers an additional route to be included in the app router.
 * Call during the extension init phase (e.g. EE register.ts) before the app boots.
 * routes.tsx reads these via getRegisteredRoutes() when building the route list.
 *
 * The lazy() wrapper is created here (once) rather than in getRoutes() so that
 * React sees a stable component reference across re-renders.
 */
export function registerRoute(name: string, path: string, factory: RouteFactory, options?: RouteOptions): void {
    registrations.push({ name, path, lazyComponent: lazy(factory), scoped: options?.scoped });
}

/** Registers a public route (no auth required). */
export function registerPublicRoute(name: string, path: string, factory: RouteFactory): void {
    publicRegistrations.push({ name, path, lazyComponent: lazy(factory) });
}

export function getRegisteredRoutes(): RegisteredRoute[] {
    return registrations;
}

export function getRegisteredScopedRoutes(): RegisteredRoute[] {
    return registrations.filter(r => r.scoped);
}

export function getRegisteredUnscopedRoutes(): RegisteredRoute[] {
    return registrations.filter(r => !r.scoped);
}

export function getRegisteredPublicRoutes(): RegisteredRoute[] {
    return publicRegistrations;
}

let surfaceFallbackPath = "/storage-unit";

export function setSurfaceFallbackPath(path: string): void {
    surfaceFallbackPath = path;
}

export function getSurfaceFallbackPath(): string {
    return surfaceFallbackPath;
}

type ScopedLayoutConfig = {
    pathPattern: string;
    layoutFactory: () => Promise<{ default: ComponentType<any> }>;
    lazyComponent: LazyExoticComponent<ComponentType<any>>;
};

let scopedLayout: ScopedLayoutConfig | null = null;

export function registerScopedLayout(pathPattern: string, layoutFactory: () => Promise<{ default: ComponentType<any> }>): void {
    scopedLayout = { pathPattern, layoutFactory, lazyComponent: lazy(layoutFactory) };
}

export function getScopedLayout(): ScopedLayoutConfig | null {
    return scopedLayout;
}
