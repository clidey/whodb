/*
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

import {useCallback, useEffect} from "react";
import {useNavigate} from "react-router-dom";
import {InternalRoutes} from "../config/routes";
import {useSourceContract} from "./useSourceContract";
import {useAppSelector} from "../store/hooks";
import {matchesShortcut, resolveShortcut, SHORTCUTS} from "../utils/shortcuts";
import {isUnscopedRouteEnabled} from "../config/route-registry";

export const useSidebarShortcuts = () => {
    const navigate = useNavigate();
    const currentType = useAppSelector(state => state.auth.current?.Type);
    const isLoggedIn = useAppSelector(state => state.auth.status === "logged-in");
    const { supportsChat, supportsGraph, supportsScratchpad } = useSourceContract(currentType);

    const handleKeyDown = useCallback((event: KeyboardEvent) => {
        // Only handle when logged in
        if (!isLoggedIn || !currentType) return;

        // Ignore if typing in an input or textarea
        if (
            event.target instanceof HTMLInputElement ||
            event.target instanceof HTMLTextAreaElement ||
            (event.target as HTMLElement)?.isContentEditable
        ) {
            return;
        }

        // Build route list based on database type (same logic as sidebar)
        const routes: string[] = [];

        // Chat is first for SQL databases
        if (supportsChat && isUnscopedRouteEnabled(InternalRoutes.Chat.path)) {
            routes.push(InternalRoutes.Chat.path);
        }

        // Storage Units
        if (isUnscopedRouteEnabled(InternalRoutes.Dashboard.StorageUnit.path)) {
            routes.push(InternalRoutes.Dashboard.StorageUnit.path);
        }

        // Graph
        if (supportsGraph && isUnscopedRouteEnabled(InternalRoutes.Graph.path)) {
            routes.push(InternalRoutes.Graph.path);
        }

        // Scratchpad (if supported)
        if (supportsScratchpad && isUnscopedRouteEnabled(InternalRoutes.RawExecute.path)) {
            routes.push(InternalRoutes.RawExecute.path);
        }

        // Number navigation
        const navDefs = [SHORTCUTS.navFirst, SHORTCUTS.navSecond, SHORTCUTS.navThird, SHORTCUTS.navFourth];
        for (let i = 0; i < navDefs.length; i++) {
            if (matchesShortcut(event, resolveShortcut(navDefs[i]))) {
                if (routes[i]) {
                    event.preventDefault();
                    void navigate(routes[i]);
                }
                return;
            }
        }

        // Cmd/Ctrl+B for sidebar toggle
        if (matchesShortcut(event, SHORTCUTS.toggleSidebar)) {
            event.preventDefault();
            window.dispatchEvent(new CustomEvent('menu:toggle-sidebar'));
        }
    }, [
	isLoggedIn,
	navigate,
	supportsChat,
	supportsGraph,
	supportsScratchpad
]);

    useEffect(() => {
        window.addEventListener('keydown', handleKeyDown);
        return () => { window.removeEventListener('keydown', handleKeyDown); };
    }, [handleKeyDown]);
};
