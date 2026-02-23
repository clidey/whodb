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
 * WhoDB Playwright helper class.
 *
 * Domain methods are split into modules under ./whodb/ and composed onto the
 * prototype via Object.assign. The class API is identical — only the internal
 * file organization changed.
 */

import { coreMethods, platformKeys, isMac } from "./whodb/core.mjs";
import { tableMethods } from "./whodb/table.mjs";
import { rowsMethods } from "./whodb/rows.mjs";
import { whereMethods } from "./whodb/where.mjs";
import { scratchpadMethods } from "./whodb/scratchpad.mjs";
import { chatMethods } from "./whodb/chat.mjs";
import { extrasMethods } from "./whodb/extras.mjs";

export { platformKeys, isMac };

export class WhoDB {
    /**
     * @param {import("@playwright/test").Page} page
     */
    constructor(page) {
        this.page = page;
        /** @type {Array<Object>|null} Test-scoped chat response storage */
        this._chatMockResponses = null;
    }
}

Object.assign(WhoDB.prototype, coreMethods);
Object.assign(WhoDB.prototype, tableMethods);
Object.assign(WhoDB.prototype, rowsMethods);
Object.assign(WhoDB.prototype, whereMethods);
Object.assign(WhoDB.prototype, scratchpadMethods);
Object.assign(WhoDB.prototype, chatMethods);
Object.assign(WhoDB.prototype, extrasMethods);
