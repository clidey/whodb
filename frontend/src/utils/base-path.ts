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

function isDesktopContext(): boolean {
    if (typeof window === "undefined") {
        return false;
    }

    const wailsGo = (window as any).go;
    return !!(wailsGo?.main?.App || wailsGo?.common?.App);
}

function normalizeBasePath(pathname: string): string {
    if (pathname === "/" || pathname === "") {
        return "";
    }

    return pathname.replace(/\/+$/, "");
}

/**
 * Returns the configured application base path for bundled web builds.
 */
export function getBasePath(): string {
    if (typeof window === "undefined" || typeof document === "undefined" || isDesktopContext()) {
        return "";
    }

    return normalizeBasePath(new URL(document.baseURI || window.location.href).pathname);
}

/**
 * Prefixes a root-relative application path with the current base path.
 */
export function withBasePath(path: string): string {
    if (!path.startsWith("/") || path.startsWith("//")) {
        return path;
    }

    const basePath = getBasePath();
    if (path === "/") {
        return basePath === "" ? "/" : `${basePath}/`;
    }

    return basePath === "" ? path : `${basePath}${path}`;
}

/**
 * Navigates to an app route via window.location, bypassing React Router.
 * Uses hash navigation on desktop (HashRouter) and full href on web (BrowserRouter).
 */
export function navigateWithBasePath(path: string): void {
    if (isDesktopContext()) {
        window.location.hash = path;
    } else {
        window.location.href = withBasePath(path);
    }
}

/**
 * Checks if the current window location matches a given app route.
 * Handles both HashRouter (desktop) and BrowserRouter (web) contexts.
 */
export function isOnRoute(path: string): boolean {
    if (isDesktopContext()) {
        return window.location.hash.startsWith('#' + path);
    }
    return window.location.pathname.startsWith(withBasePath(path));
}
