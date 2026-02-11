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

import sqlHelpers from './sql.mjs';
import documentHelpers from './document.mjs';
import keyvalueHelpers from './keyvalue.mjs';

/**
 * Get category-specific helpers based on database category
 * @param {string} category - 'sql', 'document', or 'keyvalue'
 * @returns {Object} Category helpers
 */
export function getCategoryHelpers(category) {
    switch (category) {
        case 'sql':
            return sqlHelpers;
        case 'document':
            return documentHelpers;
        case 'keyvalue':
            return keyvalueHelpers;
        default:
            throw new Error(`Unknown category: ${category}`);
    }
}

export {sqlHelpers, documentHelpers, keyvalueHelpers};
