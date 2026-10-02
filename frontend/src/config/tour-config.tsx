/*
 * Copyright 2025 Clidey, Inc.
 *
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

import type { TourConfig } from '../components/tour/tour-step';
import { InternalRoutes } from './routes';
import { withBasePath } from '../utils/base-path';

/** Creates the sample database tour in the active language. */
export const sampleDatabaseTour = (t: (key: string) => string): TourConfig => ({
    id: 'sample-database-tour',
    steps: [
        {
            target: '#whodb-app-container',
            title: t('welcomeTitle'),
            description: t('welcomeDescription'),
            position: 'center',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: `[href="${withBasePath(InternalRoutes.Chat.path)}"]`,
            title: t('chatTitle'),
            description: t('chatDescription'),
            position: 'right',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: `[href="${withBasePath(InternalRoutes.Graph.path)}"]`,
            title: t('graphTitle'),
            description: t('graphDescription'),
            position: 'right',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: '[data-testid="storage-unit-card-list"]',
            title: t('tablesTitle'),
            description: t('tablesDescription'),
            position: 'bottom',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: `[href="${withBasePath(InternalRoutes.RawExecute.path)}"]`,
            title: t('scratchpadTitle'),
            description: t('scratchpadDescription'),
            position: 'right',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: '#whodb-app-container',
            title: t('completeTitle'),
            description: t('completeDescription'),
            position: 'center',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
    ],
});
