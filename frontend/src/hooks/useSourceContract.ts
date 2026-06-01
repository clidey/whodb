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

import { useMemo } from "react";
import {
    SourceConnectionTransport,
    SourceHostInputMode,
    SourceHostInputUrlParser,
    SourceMetadataFidelity,
    SourceModel,
    SourceQueryExplainMode,
    SourceProfileLabelStrategy,
    SourceSchemaFidelity,
} from "@graphql";
import type { SourceTypeItem } from "../config/source-types";
import { resolveSourceContractFlags, type SourceContractFlags } from "../utils/source-contract-flags";
import { useSourceTypeItem } from "./useSourceCatalog";

/**
 * Fully resolved UI traits for a source type.
 */
export interface SourceContractState extends SourceContractFlags {
    /** Decorated catalog entry for the requested source type. */
    item?: SourceTypeItem;
    /** Resolved backend connector id for the source. */
    connector?: string;
    /** Transport used to reach the source. */
    connectionTransport: SourceConnectionTransport;
    /** Host input mode exposed by the source. */
    hostInputMode: SourceHostInputMode;
    /** Host URL parsing mode exposed by the source. */
    hostInputUrlParser: SourceHostInputUrlParser;
    /** Saved-profile label strategy exposed by the source. */
    profileLabelStrategy: SourceProfileLabelStrategy;
    /** Schema fidelity exposed by the source. */
    schemaFidelity: SourceSchemaFidelity;
    /** Whether the source supports explain/analyze-style query tooling. */
    supportsAnalyze: boolean;
    /** Whether the source supports source-native script execution. */
    supportsScripts: boolean;
    /** Whether the source supports streaming query execution. */
    supportsStreaming: boolean;
    /** Whether the source supports multi-statement script execution. */
    supportsMultiStatement: boolean;
    /** Whether the source supports SQL import (single or multi-statement). */
    supportsSqlImport: boolean;
    /** Source-owned explain mode for CLI/UI query planning. */
    explainMode: SourceQueryExplainMode;
    /** Whether mock-data generation can reason about relational dependencies. */
    supportsMockDataRelations: boolean;
    /** Fidelity of source column metadata. */
    columnMetadataFidelity: SourceMetadataFidelity;
    /** Fidelity of source constraint metadata. */
    constraintMetadataFidelity: SourceMetadataFidelity;
    /** Fidelity of source graph metadata. */
    graphMetadataFidelity: SourceMetadataFidelity;
    /** Fidelity of source-owned internal object filtering. */
    systemObjectFilteringFidelity: SourceMetadataFidelity;
    /** Whether the catalog is still loading without cached data. */
    loading: boolean;
    /** Whether the source behaves like a NoSQL database in the UI. */
    isNoSQL: boolean;
    /** Plural storage-unit label for the source. */
    storageUnitLabel: string;
    /** Singular storage-unit label for the source. */
    singularStorageUnitLabel: string;
}

/**
 * Resolves the catalog-backed UI traits for a source type.
 *
 * @param sourceType Source type identifier.
 * @returns Resolved source traits for the UI.
 */
export function useSourceContract(sourceType: string | undefined): SourceContractState {
    const { item, loading } = useSourceTypeItem(sourceType);

    return useMemo(() => {
        const connector = item?.connector ?? sourceType;
        const featureFlags = resolveSourceContractFlags(item);
        const contract = item?.contract;
        const defaultObjectKind = contract?.DefaultObjectKind;
        const defaultObjectType = contract?.ObjectTypes.find(objectType => objectType.Kind === defaultObjectKind);
        const model = contract?.Model;
        const traits = item?.traits;

        return {
            item,
            connector,
            connectionTransport: traits?.connection.transport ?? SourceConnectionTransport.Network,
            hostInputMode: traits?.connection.hostInputMode ?? SourceHostInputMode.None,
            hostInputUrlParser: traits?.connection.hostInputUrlParser ?? SourceHostInputUrlParser.None,
            profileLabelStrategy: traits?.presentation.profileLabelStrategy ?? SourceProfileLabelStrategy.Default,
            schemaFidelity: traits?.presentation.schemaFidelity ?? SourceSchemaFidelity.Exact,
            supportsAnalyze: traits?.query.supportsAnalyze ?? false,
            explainMode: traits?.query.explainMode ?? SourceQueryExplainMode.None,
            supportsMockDataRelations: traits?.mockData.supportsRelationalDependencies ?? true,
            columnMetadataFidelity: traits?.metadata.columns ?? SourceMetadataFidelity.Unknown,
            constraintMetadataFidelity: traits?.metadata.constraints ?? SourceMetadataFidelity.Unknown,
            graphMetadataFidelity: traits?.metadata.graph ?? SourceMetadataFidelity.Unsupported,
            systemObjectFilteringFidelity: traits?.metadata.systemObjectFiltering ?? SourceMetadataFidelity.Unsupported,
            loading,
            isNoSQL: model != null && model !== SourceModel.Relational,
            storageUnitLabel: item?.storageUnitLabel ?? defaultObjectType?.PluralLabel ?? "Storage Units",
            singularStorageUnitLabel: item?.singularStorageUnitLabel ?? defaultObjectType?.SingularLabel ?? "Storage Unit",
            ...featureFlags,
        };
    }, [sourceType, item, loading]);
}
