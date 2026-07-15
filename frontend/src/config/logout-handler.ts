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

import type { NavigateFunction } from "react-router-dom";
import { InternalRoutes } from "./routes";

type LogoutHandler = () => void;

let logoutHandler: LogoutHandler | null = null;

/** Registers an edition-specific logout action. */
export function registerLogoutHandler(handler: LogoutHandler): void {
    logoutHandler = handler;
}

/** Runs the registered logout action, or navigates to the default logout route. */
export function performLogout(navigate: NavigateFunction): void {
    if (logoutHandler) {
        logoutHandler();
        return;
    }
    void navigate(InternalRoutes.Logout.path);
}
