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

import { ComponentType, ReactElement, createElement } from "react";
import {
    SourceTypesQuery,
    SourceAction,
    SourceConnectionFieldSection,
    SourceModel,
    SourceObjectKind,
    SourceSurface,
    SourceView,
} from "@graphql";
import { Icons } from "../components/icons";
import { getEdition } from "./edition";
import { getRegisteredSourceTypeOverrides } from "./source-registry";

/**
 * Type category for grouping database types in the UI.
 */
export type TypeCategory = 'numeric' | 'text' | 'binary' | 'datetime' | 'boolean' | 'json' | 'other';

/**
 * SSL mode option for database connections.
 * Matches backend ssl.SSLModeInfo structure.
 */
export interface SSLModeOption {
    /** Mode value used in configuration (e.g., "required", "verify-ca") */
    value: string;
    /** Accepted aliases for this mode (e.g., PostgreSQL's "require" for "required") */
    aliases?: string[];
}

export interface SourceConnectionFieldDescriptor {
    Key: string;
    Kind: SourceTypesQuery['SourceTypes'][number]['connectionFields'][number]['Kind'];
    Section: SourceTypesQuery['SourceTypes'][number]['connectionFields'][number]['Section'];
    Required: boolean;
    LabelKey: string;
    PlaceholderKey?: string | null;
    DefaultValue?: string | null;
    SupportsOptions: boolean;
}

export interface SourceObjectTypeDescriptor {
    Kind: SourceObjectKind;
    DataShape: SourceTypesQuery['SourceTypes'][number]['contract']['ObjectTypes'][number]['DataShape'];
    Actions: SourceAction[];
    Views: SourceView[];
    SingularLabel: string;
    PluralLabel: string;
}

export interface SourceContractDescriptor {
    Model: SourceModel;
    Surfaces: SourceSurface[];
    RootActions: SourceAction[];
    BrowsePath: SourceObjectKind[];
    DefaultObjectKind: SourceObjectKind;
    GraphScopeKind?: SourceObjectKind | null;
    ObjectTypes: SourceObjectTypeDescriptor[];
}

/**
 * Connection traits exposed by the backend source catalog.
 */
export interface SourceConnectionTraitsDescriptor {
    transport: NonNullable<SourceTypesQuery['SourceTypes'][number]['traits']>['connection']['transport'];
    hostInputMode: NonNullable<SourceTypesQuery['SourceTypes'][number]['traits']>['connection']['hostInputMode'];
    hostInputUrlParser: NonNullable<SourceTypesQuery['SourceTypes'][number]['traits']>['connection']['hostInputUrlParser'];
    supportsCustomCAContent: NonNullable<SourceTypesQuery['SourceTypes'][number]['traits']>['connection']['supportsCustomCAContent'];
}

/**
 * Presentation traits exposed by the backend source catalog.
 */
export interface SourcePresentationTraitsDescriptor {
    profileLabelStrategy: NonNullable<SourceTypesQuery['SourceTypes'][number]['traits']>['presentation']['profileLabelStrategy'];
    schemaFidelity: NonNullable<SourceTypesQuery['SourceTypes'][number]['traits']>['presentation']['schemaFidelity'];
}

/**
 * Query-surface traits exposed by the backend source catalog.
 */
export interface SourceQueryTraitsDescriptor {
    supportsAnalyze: NonNullable<SourceTypesQuery['SourceTypes'][number]['traits']>['query']['supportsAnalyze'];
    explainMode: NonNullable<SourceTypesQuery['SourceTypes'][number]['traits']>['query']['explainMode'];
}

/**
 * Mock-data traits exposed by the backend source catalog.
 */
export interface SourceMockDataTraitsDescriptor {
    supportsRelationalDependencies: NonNullable<SourceTypesQuery['SourceTypes'][number]['traits']>['mockData']['supportsRelationalDependencies'];
}

/**
 * Backend-owned source traits that do not belong in the CRUD contract.
 */
export interface SourceTraitsDescriptor {
    connection: SourceConnectionTraitsDescriptor;
    presentation: SourcePresentationTraitsDescriptor;
    query: SourceQueryTraitsDescriptor;
    mockData: SourceMockDataTraitsDescriptor;
}

/**
 * Discovery-prefill metadata condition exposed by the backend source catalog.
 */
export interface SourceDiscoveryMetadataConditionDescriptor {
    Key: string;
    Value: string;
}

/**
 * Discovery-prefill advanced-field default exposed by the backend source catalog.
 */
export interface SourceDiscoveryAdvancedDefaultDescriptor {
    Key: string;
    Value: string;
    MetadataKey: string;
    DefaultValue: string;
    ProviderTypes: string[];
    Conditions: SourceDiscoveryMetadataConditionDescriptor[];
}

/**
 * Discovery-prefill metadata exposed by the backend source catalog.
 */
export interface SourceDiscoveryPrefillDescriptor {
    AdvancedDefaults: SourceDiscoveryAdvancedDefaultDescriptor[];
}

/**
 * Defines a canonical source type for use in type selectors.
 * Types are from each database's official documentation.
 */
export interface TypeDefinition {
    /** Canonical type name (e.g., "VARCHAR", "INTEGER") - stored internally */
    id: string;
    /** Display label shown in UI (e.g., "varchar", "integer") - database's preferred case */
    label: string;
    /** Shows length input when selected (VARCHAR, CHAR) */
    hasLength?: boolean;
    /** Shows precision/scale inputs when selected (DECIMAL, NUMERIC) */
    hasPrecision?: boolean;
    /** Default length value for types with hasLength */
    defaultLength?: number;
    /** Default precision for types with hasPrecision */
    defaultPrecision?: number;
    /** Type category for grouping and icon selection */
    category: TypeCategory;
    /** Function to wrap INSERT values (e.g. "TO_BITMAP") — aggregate types only */
    insertFunc?: string;
    /** Required table key model (e.g. "AGGREGATE") — aggregate types only */
    tableModel?: string;
}

/**
 * Props passed to a custom login form renderer.
 * Allows custom source types to fully control the login form fields.
 */
export interface CustomLoginFormProps {
    hostName: string;
    setHostName: (value: string) => void;
    username: string;
    setUsername: (value: string) => void;
    password: string;
    setPassword: (value: string) => void;
    advancedForm: Record<string, string>;
    setAdvancedForm: (value: Record<string, string>) => void;
}

/**
 * Stateful values exposed to custom connection forms and validators.
 */
export interface CustomLoginFormState {
    hostName: string;
    username: string;
    password: string;
    advancedForm: Record<string, string>;
}

/**
 * Source type item with backend-provided connection behavior and a thin
 * frontend decoration layer.
 */
export interface SourceTypeItem {
    id: string;
    label: string;
    connector: string;
    icon: ReactElement;
    extra: Record<string, string>;
    category?: SourceTypesQuery['SourceTypes'][number]['category'];
    traits?: SourceTraitsDescriptor;
    connectionFields?: SourceConnectionFieldDescriptor[];
    contract?: SourceContractDescriptor;
    discoveryPrefill?: SourceDiscoveryPrefillDescriptor;
    fields?: {
        hostname?: boolean;
        username?: boolean;
        password?: boolean;
        database?: boolean;
        searchPath?: boolean;
    };
    requiredFields?: {
        hostname?: boolean;
        username?: boolean;
        password?: boolean;
        database?: boolean;
    };
    operators?: string[];
    /** Canonical type definitions for type selectors */
    typeDefinitions?: TypeDefinition[];
    /** Maps type aliases to canonical names (e.g., INT4 -> INTEGER) */
    aliasMap?: Record<string, string>;
    /** Whether this database supports field modifiers (primary, nullable) */
    supportsModifiers?: boolean;
    /** Whether this database supports scratchpad/raw query execution */
    supportsScratchpad?: boolean;
    /** Whether this database supports the chat surface. */
    supportsChat?: boolean;
    /** Whether this database supports the graph surface. */
    supportsGraph?: boolean;
    /** Whether this database supports schemas */
    supportsSchema?: boolean;
    /** Whether this database supports switching between databases in the UI */
    supportsDatabaseSwitching?: boolean;
    /** Whether this database should use the schema field for graph queries */
    usesSchemaForGraph?: boolean;
    /** Whether this database type uses database selection instead of schema selection */
    usesDatabaseInsteadOfSchema?: boolean;
    /** Whether this database supports mock data generation */
    supportsMockData?: boolean;
    /** Singular storage object label derived from the source contract. */
    singularStorageUnitLabel?: string;
    /** Plural storage object label derived from the source contract. */
    storageUnitLabel?: string;
    /** Whether this database type is an AWS managed service */
    isAwsManaged?: boolean;
    /** SSL modes supported by this database */
    sslModes?: SSLModeOption[];
    /** Optional custom login form renderer */
    customFormRenderer?: ComponentType<CustomLoginFormProps>;
    /** Optional custom form submit validation */
    customFormCanSubmit?: (state: CustomLoginFormState) => boolean;
}

/**
 * UI-only override for a backend-owned source catalog entry.
 */
export type SourceTypeOverride = Pick<SourceTypeItem, 'id'> &
    Partial<Omit<SourceTypeItem, 'id'>>;

/**
 * Filter options for source type retrieval.
 */
export interface SourceTypeFilterOptions {
    /** When false, all cloud-managed source types are excluded. */
    cloudProvidersEnabled?: boolean;
    /** When false, AWS managed source types are excluded. */
    awsProviderEnabled?: boolean;
}

/**
 * Raw backend catalog entry returned by the GraphQL catalog query.
 */
export type BackendSourceType = SourceTypesQuery['SourceTypes'][number];

const shouldUseCatalogCache = !import.meta.env.DEV;

function getCatalogCacheKey(): string {
    return `whodb_source_catalog_${getEdition()}_${__APP_VERSION__}`;
}

/**
 * Reads the persisted backend catalog cache for the current edition/version.
 *
 * @returns Cached raw backend catalog entries.
 */
export function readCachedSourceCatalog(): BackendSourceType[] {
    if (!shouldUseCatalogCache) {
        return [];
    }

    try {
        const raw = localStorage.getItem(getCatalogCacheKey());
        if (!raw) {
            return [];
        }

        const parsed = JSON.parse(raw) as BackendSourceType[];
        return Array.isArray(parsed) ? parsed : [];
    } catch {
        return [];
    }
}

/**
 * Persists the backend catalog cache for the current edition/version.
 *
 * @param items Raw backend catalog entries to cache.
 */
export function writeCachedSourceCatalog(items: BackendSourceType[]): void {
    if (!shouldUseCatalogCache) {
        return;
    }

    try {
        localStorage.setItem(getCatalogCacheKey(), JSON.stringify(items));
    } catch {
        // Ignore storage failures; the live query result is still valid.
    }
}

function findConnectionField(
    connectionFields: BackendSourceType['connectionFields'],
    key: string
): BackendSourceType['connectionFields'][number] | undefined {
    return connectionFields.find(field => field.Key.toLowerCase() === key.toLowerCase());
}

function mapAdvancedDefaults(
    connectionFields: BackendSourceType['connectionFields']
): Record<string, string> {
    return connectionFields.reduce<Record<string, string>>((acc, field) => {
        if (field.Section === SourceConnectionFieldSection.Advanced) {
            acc[field.Key] = field.DefaultValue ?? "";
        }
        return acc;
    }, {});
}

function resolveIcon(sourceType: string, connector: string): ReactElement {
    const logos = Icons.Logos as Record<string, ReactElement>;
    return logos[sourceType] ?? logos[connector] ?? createElement("span", { className: "w-6 h-6" });
}

function decorateSourceType(item: BackendSourceType): SourceTypeItem {
    const hostnameField = findConnectionField(item.connectionFields, "Hostname");
    const usernameField = findConnectionField(item.connectionFields, "Username");
    const passwordField = findConnectionField(item.connectionFields, "Password");
    const databaseField = findConnectionField(item.connectionFields, "Database");
    const searchPathField = findConnectionField(item.connectionFields, "Search Path");
    const browsePath = item.contract.BrowsePath;
    const objectTypes = item.contract.ObjectTypes;
    const defaultObjectType = objectTypes.find(objectType => objectType.Kind === item.contract.DefaultObjectKind);

    return {
        id: item.id,
        label: item.label,
        connector: item.connector,
        icon: resolveIcon(item.id, item.connector),
        extra: mapAdvancedDefaults(item.connectionFields),
        category: item.category,
        traits: item.traits,
        connectionFields: item.connectionFields,
        contract: item.contract,
        discoveryPrefill: item.discoveryPrefill,
        fields: {
            hostname: hostnameField != null,
            username: usernameField != null,
            password: passwordField != null,
            database: databaseField != null,
            searchPath: searchPathField != null,
        },
        requiredFields: {
            hostname: hostnameField?.Required ?? false,
            username: usernameField?.Required ?? false,
            password: passwordField?.Required ?? false,
            database: databaseField?.Required ?? false,
        },
        supportsModifiers: item.contract.Model === SourceModel.Relational,
        supportsChat: item.contract.Surfaces.includes(SourceSurface.Chat),
        supportsGraph: item.contract.Surfaces.includes(SourceSurface.Graph),
        supportsScratchpad: item.contract.Surfaces.includes(SourceSurface.Query),
        supportsSchema: browsePath.includes(SourceObjectKind.Schema),
        supportsDatabaseSwitching: browsePath.includes(SourceObjectKind.Database),
        usesSchemaForGraph: item.contract.GraphScopeKind === SourceObjectKind.Schema,
        usesDatabaseInsteadOfSchema:
            item.contract.GraphScopeKind === SourceObjectKind.Database ||
            (!browsePath.includes(SourceObjectKind.Schema) && browsePath.includes(SourceObjectKind.Database)),
        supportsMockData: objectTypes.some(objectType => objectType.Actions.includes(SourceAction.GenerateMockData)),
        isAwsManaged: item.isAwsManaged,
        singularStorageUnitLabel: defaultObjectType?.SingularLabel,
        storageUnitLabel: defaultObjectType?.PluralLabel,
        sslModes: item.sslModes.map(mode => ({
            value: mode.value,
            aliases: mode.aliases.length > 0 ? mode.aliases : undefined,
        })),
    };
}

function filterSourceTypes(
    items: SourceTypeItem[],
    options: SourceTypeFilterOptions = {}
): SourceTypeItem[] {
    const cloudProvidersEnabled = options.cloudProvidersEnabled ?? true;
    const awsProviderEnabled = options.awsProviderEnabled ?? cloudProvidersEnabled;

    if (cloudProvidersEnabled) {
        return items.filter(item => awsProviderEnabled || !item.isAwsManaged);
    }

    return items.filter(item => !item.isAwsManaged);
}

function mergeSourceTypeOverride(
    item: SourceTypeItem,
    override: SourceTypeOverride
): SourceTypeItem {
    return {
        ...item,
        ...override,
        extra: override.extra ? { ...item.extra, ...override.extra } : item.extra,
        traits: override.traits
            ? {
                connection: { ...(item.traits?.connection ?? {}), ...override.traits.connection },
                presentation: { ...(item.traits?.presentation ?? {}), ...override.traits.presentation },
                query: { ...(item.traits?.query ?? {}), ...override.traits.query },
                mockData: { ...(item.traits?.mockData ?? {}), ...override.traits.mockData },
            }
            : item.traits,
        fields: override.fields ? { ...item.fields, ...override.fields } : item.fields,
        requiredFields: override.requiredFields ? { ...item.requiredFields, ...override.requiredFields } : item.requiredFields,
    };
}

function withRegisteredSourceTypes(baseTypes: SourceTypeItem[]): SourceTypeItem[] {
    const overrides = getRegisteredSourceTypeOverrides();
    if (overrides.length === 0) {
        return baseTypes;
    }

    return baseTypes.map(item => {
        const override = overrides.find(candidate => candidate.id === item.id);
        return override ? mergeSourceTypeOverride(item, override) : item;
    });
}

/**
 * Decorates raw backend catalog data with frontend-only presentation details.
 *
 * @param catalog Raw backend catalog entries.
 * @param options Optional UI filters for the resulting list.
 * @returns Decorated source type items ready for rendering.
 */
export function resolveSourceTypeItems(
    catalog: BackendSourceType[],
    options: SourceTypeFilterOptions = {}
): SourceTypeItem[] {
    const items = withRegisteredSourceTypes(catalog.map(decorateSourceType));
    return filterSourceTypes(items, options);
}

/**
 * Finds a single decorated source type entry by id.
 *
 * @param items Decorated source type items.
 * @param sourceType Source type identifier.
 * @returns Matching catalog item, if present.
 */
export function findSourceTypeItem(
    items: SourceTypeItem[],
    sourceType: string | undefined
): SourceTypeItem | undefined {
    if (!sourceType) {
        return undefined;
    }

    return items.find(item => item.id === sourceType);
}

/**
 * Resolves a displayed source type to its underlying connector id.
 *
 * @param sourceType Source type identifier.
 * @param item Decorated catalog entry for the source type.
 * @returns The resolved connector id, or the original type when no catalog item is available.
 */
export function resolveSourceConnector(
    sourceType: string | undefined,
    item: SourceTypeItem | undefined
): string | undefined {
    if (!sourceType) {
        return undefined;
    }

    return item?.connector ?? sourceType;
}

/**
 * Resolves one declared source object-type descriptor by kind.
 *
 * @param item Decorated source type item.
 * @param kind Source object kind to resolve.
 * @returns Matching object-type descriptor when present.
 */
export function findSourceObjectType(
    item: SourceTypeItem | undefined,
    kind: SourceObjectKind | undefined | null
): SourceObjectTypeDescriptor | undefined {
    if (!item?.contract || kind == null) {
        return undefined;
    }

    return item.contract.ObjectTypes.find(objectType => objectType.Kind === kind);
}

/**
 * Resolves the default browsed object-type descriptor for one source type.
 *
 * @param item Decorated source type item.
 * @returns The default object-type descriptor when present.
 */
export function getDefaultSourceObjectType(
    item: SourceTypeItem | undefined
): SourceObjectTypeDescriptor | undefined {
    return findSourceObjectType(item, item?.contract?.DefaultObjectKind);
}
