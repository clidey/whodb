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

import React from 'react';
import { Card } from './card';
import { StarIcon } from '@heroicons/react/24/outline';

export const EEFeatureCard: React.FC<{ feature: string; description?: string }> = ({ feature, description }) => {
    return (
        <Card className="p-8 text-center">
            <div className="flex flex-col items-center space-y-4">
                <StarIcon className="w-4 h-4" />
                <h3 className="text-xl font-semibold">{feature}</h3>
                <p className="text-gray-600 dark:text-gray-400">
                    {description || 'This feature is available in WhoDB Enterprise Edition'}
                </p>
                <a 
                    href="https://github.com/clidey/whodb/blob/main/ee/README.md" 
                    target="_blank" 
                    rel="noopener noreferrer"
                    className="text-blue-600 dark:text-blue-400 hover:underline"
                >
                    Learn more about Enterprise features →
                </a>
            </div>
        </Card>
    );
};

