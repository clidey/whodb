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

import { TourConfig } from '../components/tour/tour-step';
import {
    ChatBubbleLeftRightIcon,
    CodeBracketIcon,
    ShareIcon,
    SparklesIcon,
    TableCellsIcon,
    AdjustmentsHorizontalIcon,
    ArrowDownTrayIcon,
} from '../components/heroicons';
import { InternalRoutes } from './routes';

export const sampleDatabaseTour: TourConfig = {
    id: 'sample-database-tour',
    steps: [
        {
            target: '#whodb-app-container',
            title: 'Welcome to WhoDB',
            description: 'Let\'s take a quick tour to show you how to make the most of WhoDB with our sample database. This will only take a minute!',
            icon: <SparklesIcon className="w-6 h-6 text-brand-foreground" />,
            position: 'center',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: '[href="/chat"]',
            title: 'AI Chat Assistant',
            description: 'Ask questions in plain English like "Show me all customers" or "What are the top products?". The AI will generate and run SQL queries for you.',
            icon: <ChatBubbleLeftRightIcon className="w-6 h-6 text-brand-foreground" />,
            position: 'right',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: '[href="/graph"]',
            title: 'Visual Schema Explorer',
            description: 'See your entire database structure at a glance. Interactive graph shows all tables, columns, and relationships with zoom and pan controls.',
            icon: <ShareIcon className="w-6 h-6 text-brand-foreground" />,
            position: 'right',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: '[data-testid="storage-unit-card-list"]',
            title: 'Browse Database Tables',
            description: 'Here are all the tables in your database. Click on any table card to view and edit its data in a spreadsheet-like grid. You can sort, filter, and modify data with ease.',
            icon: <TableCellsIcon className="w-6 h-6 text-brand-foreground" />,
            position: 'bottom',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: '[href="/scratchpad"]',
            title: 'SQL Editor & Scratchpad',
            description: 'Write custom SQL queries with syntax highlighting and auto-completion. All your queries are automatically saved in history.',
            icon: <CodeBracketIcon className="w-6 h-6 text-brand-foreground" />,
            position: 'right',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: '[data-testid="data-button"]',
            title: 'View Table Data',
            description: 'Click the "Data" button on any table card to open it in the data grid. From there, you can filter records, export data, and edit cells directly like a spreadsheet.',
            icon: <AdjustmentsHorizontalIcon className="w-6 h-6 text-brand-foreground" />,
            position: 'left',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
        {
            target: '#whodb-app-container',
            title: 'You\'re All Set!',
            description: 'You now know the key features of WhoDB. Start exploring the sample database or connect your own database from the sidebar. Happy exploring!',
            icon: <SparklesIcon className="w-6 h-6 text-brand-foreground" />,
            position: 'center',
            path: InternalRoutes.Dashboard.StorageUnit.path,
        },
    ],
};
