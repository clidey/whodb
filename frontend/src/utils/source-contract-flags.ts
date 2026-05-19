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

import { SourceAction, SourceModel, SourceObjectKind, SourceSurface } from '@graphql';
import {
    sourceSupportsAction,
    sourceSupportsSurface,
    sourceUsesObjectKind,
    type SourceTypeItem,
} from '../config/source-types';

/**
 * Catalog-backed source contract flags resolved for the current UI.
 */
export interface SourceContractFlags {
    supportsChat: boolean;
    supportsGraph: boolean;
    supportsScratchpad: boolean;
    supportsScripts: boolean;
    supportsStreaming: boolean;
    supportsMultiStatement: boolean;
    supportsSqlImport: boolean;
    supportsSchema: boolean;
    supportsDatabaseSwitching: boolean;
    usesSchemaForGraph: boolean;
    usesDatabaseInsteadOfSchema: boolean;
    supportsMockData: boolean;
    supportsImportData: boolean;
    supportsModifiers: boolean;
}

/**
 * Resolves feature flags from the catalog-derived source contract.
 *
 * @param item Decorated catalog entry for the database type.
 * @returns The resolved feature flags for the database type.
 */
export function resolveSourceContractFlags(
    item: SourceTypeItem | undefined
): SourceContractFlags {
    return {
        supportsChat: sourceSupportsSurface(item, SourceSurface.Chat),
        supportsGraph: sourceSupportsSurface(item, SourceSurface.Graph),
        supportsScratchpad: sourceSupportsSurface(item, SourceSurface.Query),
        supportsScripts: item?.traits?.query.supportsScripts ?? false,
        supportsStreaming: item?.traits?.query.supportsStreaming ?? false,
        supportsMultiStatement: item?.traits?.query.supportsMultiStatement ?? false,
        supportsSqlImport: item?.traits?.query.supportsSqlImport ?? false,
        supportsSchema: sourceUsesObjectKind(item, SourceObjectKind.Schema),
        supportsDatabaseSwitching: sourceUsesObjectKind(item, SourceObjectKind.Database),
        usesSchemaForGraph: item?.usesSchemaForGraph ?? true,
        usesDatabaseInsteadOfSchema: item?.usesDatabaseInsteadOfSchema ?? false,
        supportsMockData: sourceSupportsAction(item, SourceAction.GenerateMockData),
        supportsImportData: sourceSupportsAction(item, SourceAction.ImportData),
        supportsModifiers: item?.contract?.Model === SourceModel.Relational,
    };
}
