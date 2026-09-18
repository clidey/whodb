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

import { updateDocumentMeta } from './meta';
import { defaultFeatures, type FeatureFlags } from './feature-defaults';

export type { FeatureFlags } from './feature-defaults';

export let featureFlags: FeatureFlags = {} as FeatureFlags;
export const extensions: Record<string, any> = {};
export const sources: Record<string, any> = {};
export const settingsDefaults: Record<string, any> = {};

export const getAppName = (): string => extensions.AppName ?? "WhoDB";

/** Initialize feature flags with defaults. */
export const initialize = () => {
    featureFlags = defaultFeatures;
    updateDocumentMeta({});
};

initialize();
