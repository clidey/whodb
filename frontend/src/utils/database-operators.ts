/**
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

import { DatabaseType } from '@graphql';
import { reduxStore } from '../store';

/**
 * Get valid operators for a database type from the backend-driven Redux store.
 *
 * @param databaseType The database type (can be CE or EE type)
 * @returns Array of valid operators for the database
 */
export function getDatabaseOperators(databaseType: DatabaseType | string): string[] {
    const metadataState = reduxStore.getState().databaseMetadata;

    if (
        metadataState.databaseType === databaseType &&
        metadataState.operators.length > 0
    ) {
        return metadataState.operators;
    }

    // If we reach here, metadata hasn't been fetched yet.
    console.warn(
        `[database-operators] No operators found for ${databaseType}. ` +
            `Ensure DatabaseMetadata query has completed.`
    );
    return [];
}
