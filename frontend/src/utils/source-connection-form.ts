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

import {
    parseNormalize,
    type ConnparseAddress,
    type NormalizedConnparseAddress,
    type QueryValue,
} from '@clidey/connparse';
import { SourceConnectionFieldSection, SourceConnectionTransport } from '@graphql';
import type { SourceTypeItem } from '@/config/source-types';
import { SSL_KEYS, getSSLAdvancedKeys } from '@/utils/source-ssl';

type SourceConnectionFieldItem = NonNullable<SourceTypeItem['connectionFields']>[number];
type ParsedSourceConnectionString = {
    hostName: string;
    username: string;
    password: string;
    database: string;
    advancedForm: Record<string, string>;
    showAdvanced: boolean;
    shouldWarn: boolean;
};

type SourceConnectionStringParseResult = {
    handled: boolean;
    values?: ParsedSourceConnectionString;
};

/**
 * One derived advanced-section state snapshot for one source form.
 */
export interface SourceAdvancedSectionState {
    declaredAdvancedFields: SourceConnectionFieldItem[];
    fallbackAdvancedEntries: Array<[string, string]>;
    hasAdvancedSection: boolean;
}

/**
 * Checks whether one source type uses a file transport instead of a network connection.
 *
 * @param databaseType Source type to inspect.
 * @returns `true` when the source is file-backed.
 */
export function usesFileTransport(databaseType: SourceTypeItem | null | undefined): boolean {
    return databaseType?.traits?.connection.transport === SourceConnectionTransport.File;
}

/**
 * Finds one declared connection field by key, case-insensitively.
 *
 * @param databaseType Source type to inspect.
 * @param key Connection-field key.
 * @returns Matching connection-field descriptor, if present.
 */
export function findConnectionFieldByKey(
    databaseType: SourceTypeItem | null | undefined,
    key: string
): SourceConnectionFieldItem | undefined {
    return databaseType?.connectionFields?.find(field =>
        field.Key.toLowerCase() === key.toLowerCase()
    );
}

/**
 * Checks whether the database field supports selectable options.
 *
 * @param databaseType Source type to inspect.
 * @returns `true` when the database field supports options.
 */
export function supportsDatabaseFieldOptions(
    databaseType: SourceTypeItem | null | undefined
): boolean {
    return databaseType?.connectionFields?.some(field =>
        field.Key.toLowerCase() === 'database' && field.SupportsOptions
    ) ?? false;
}

/**
 * Attempts to translate one pasted connection string into backend-owned source form fields.
 *
 * @param databaseType Source type that owns the target form.
 * @param input Raw pasted input from the hostname field.
 * @returns Parsed field values when the input can be mapped into the form.
 */
export function tryParseSourceConnectionString(
    databaseType: SourceTypeItem | null | undefined,
    input: string
): SourceConnectionStringParseResult {
    if (
        databaseType == null ||
        !looksLikeConnectionString(input) ||
        findConnectionFieldByKey(databaseType, 'Hostname') == null
    ) {
        return { handled: false };
    }

    const provider = resolveConnparseProvider(databaseType);
    if (provider == null) {
        return { handled: false };
    }

    const result = parseNormalize(input, {
        provider,
        includeCredentials: true,
        includeDefaultPort: true,
    });
    if (!result.ok || result.value == null) {
        return { handled: false };
    }

    const parsedAddress: NormalizedConnparseAddress = result.value;
    const endpoint = singleEndpointFromAddress(parsedAddress);
    if (endpoint == null || endpoint.hostName.length === 0) {
        return { handled: false };
    }

    const urlParamsField = findConnectionFieldByKey(databaseType, 'URL Params');
    const dnsEnabledField = findConnectionFieldByKey(databaseType, 'DNS Enabled');
    const advancedForm: Record<string, string> = {};
    const fieldKeyByNormalized = new Map<string, string>();

    for (const field of databaseType.connectionFields ?? []) {
        fieldKeyByNormalized.set(normalizeConnectionFieldKey(field.Key), field.Key);
    }
    fieldKeyByNormalized.set(normalizeConnectionFieldKey(SSL_KEYS.MODE), SSL_KEYS.MODE);
    fieldKeyByNormalized.set(normalizeConnectionFieldKey(SSL_KEYS.SERVER_NAME), SSL_KEYS.SERVER_NAME);

    if (endpoint.port != null && findConnectionFieldByKey(databaseType, 'Port') != null) {
        advancedForm.Port = String(endpoint.port);
    }

    const handledQueryKeys = new Set(parsedAddress.semantic?.consumed?.query ?? []);

    for (const [semanticKey, semanticValue] of Object.entries(parsedAddress.semantic?.fields ?? {})) {
        const fieldKey = fieldKeyByNormalized.get(normalizeConnectionFieldKey(semanticKey));
        if (fieldKey == null) {
            continue;
        }

        advancedForm[fieldKey] = String(semanticValue);
    }

    for (const [queryKey, queryValue] of Object.entries(parsedAddress.query)) {
        if (handledQueryKeys.has(queryKey)) {
            continue;
        }

        const fieldKey = fieldKeyByNormalized.get(normalizeConnectionFieldKey(queryKey));
        if (fieldKey != null) {
            advancedForm[fieldKey] = queryValueToInputValue(queryValue);
            handledQueryKeys.add(queryKey);
        }
    }

    if (dnsEnabledField != null && advancedForm[dnsEnabledField.Key]?.toLowerCase() === 'true') {
        if (findConnectionFieldByKey(databaseType, 'Port') != null) {
            advancedForm.Port = '';
        }
    }

    if (urlParamsField != null) {
        const urlParams = buildQueryString(filterQueryByHandledKeys(parsedAddress.query, handledQueryKeys));
        if (urlParams.length > 0) {
            advancedForm[urlParamsField.Key] = urlParams;
        }
    }

    const values: ParsedSourceConnectionString = {
        hostName: endpoint.hostName,
        username: findConnectionFieldByKey(databaseType, 'Username') != null
            ? (parsedAddress.credentials.username ?? '')
            : '',
        password: findConnectionFieldByKey(databaseType, 'Password') != null
            ? (parsedAddress.credentials.password ?? '')
            : '',
        database: findConnectionFieldByKey(databaseType, 'Database') != null
            ? (parsedAddress.resource.name ?? '')
            : '',
        advancedForm,
        showAdvanced: Object.keys(advancedForm).length > 0,
        shouldWarn: !canSubmitStandardConnectionForm(
            databaseType,
            endpoint.hostName,
            findConnectionFieldByKey(databaseType, 'Username') != null
                ? (parsedAddress.credentials.username ?? '')
                : '',
            findConnectionFieldByKey(databaseType, 'Password') != null
                ? (parsedAddress.credentials.password ?? '')
                : '',
            findConnectionFieldByKey(databaseType, 'Database') != null
                ? (parsedAddress.resource.name ?? '')
                : '',
            advancedForm
        ),
    };

    return {
        handled: true,
        values,
    };
}

function connectionFieldValue(
    field: SourceConnectionFieldItem,
    values: {
        hostName: string;
        username: string;
        password: string;
        database: string;
        advancedForm: Record<string, string>;
    }
): string {
    switch (field.Key.toLowerCase()) {
    case 'hostname':
        return values.hostName;
    case 'username':
        return values.username;
    case 'password':
        return values.password;
    case 'database':
        return values.database;
    default:
        return values.advancedForm[field.Key] ?? '';
    }
}

/**
 * Validates the standard field-based connection form against the backend-owned
 * required-field contract, including required advanced fields.
 *
 * @param databaseType Source type to validate against.
 * @param hostName Hostname value.
 * @param username Username value.
 * @param password Password value.
 * @param database Database value.
 * @param advancedForm Advanced field values.
 * @returns `true` when all required declared connection fields are present.
 */
export function canSubmitStandardConnectionForm(
    databaseType: SourceTypeItem,
    hostName: string,
    username: string,
    password: string,
    database: string,
    advancedForm: Record<string, string>
): boolean {
    return (databaseType.connectionFields ?? []).every(field => {
        if (!field.Required) {
            return true;
        }

        return connectionFieldValue(field, {
            hostName,
            username,
            password,
            database,
            advancedForm,
        }).trim().length > 0;
    });
}

/**
 * Validates a source type that owns its own custom connection form.
 *
 * @param databaseType Source type to validate against.
 * @param hostName Hostname value.
 * @param username Username value.
 * @param password Password value.
 * @param advancedForm Advanced field values.
 * @returns `true` when the custom form is complete enough to submit.
 */
export function canSubmitCustomConnectionForm(
    databaseType: SourceTypeItem,
    hostName: string,
    username: string,
    password: string,
    advancedForm: Record<string, string>,
): boolean {
    if (databaseType.customFormCanSubmit != null) {
        return databaseType.customFormCanSubmit({
            hostName,
            username,
            password,
            advancedForm,
        });
    }

    return hostName.length > 0 || Object.keys(advancedForm).length > 0;
}

/**
 * Builds the set of connection-field keys that are rendered outside the generic advanced section.
 *
 * @param databaseType Source type to inspect.
 * @param options Optional extra promoted keys or keys to omit from the promoted set.
 * @returns Promoted field keys.
 */
export function getPromotedConnectionFieldKeys(
    databaseType: SourceTypeItem | null | undefined,
    options: {
        includeKeys?: Iterable<string>;
        omitKeys?: Iterable<string>;
    } = {}
): ReadonlySet<string> {
    const keys = new Set<string>();

    for (const field of databaseType?.connectionFields ?? []) {
        if (field.Section !== SourceConnectionFieldSection.Advanced) {
            keys.add(field.Key);
        }
    }

    for (const key of options.includeKeys ?? []) {
        keys.add(key);
    }

    for (const key of options.omitKeys ?? []) {
        keys.delete(key);
    }

    return keys;
}

/**
 * Removes promoted primary-field keys from one advanced-form object.
 *
 * @param advancedForm Advanced values to filter.
 * @param promotedKeys Keys rendered outside the generic advanced section.
 * @returns Filtered advanced-form values.
 */
export function filterAdvancedFormByKeys(
    advancedForm: Record<string, string>,
    promotedKeys: ReadonlySet<string>
): Record<string, string> {
    return Object.fromEntries(
        Object.entries(advancedForm).filter(([key]) => !promotedKeys.has(key))
    );
}

/**
 * Derives the declared and fallback advanced fields for one source form.
 *
 * @param databaseType Source type to inspect.
 * @param advancedForm Current advanced values.
 * @param promotedKeys Keys rendered outside the generic advanced section.
 * @returns Derived advanced-section state.
 */
export function buildSourceAdvancedSectionState(
    databaseType: SourceTypeItem | null | undefined,
    advancedForm: Record<string, string>,
    promotedKeys: ReadonlySet<string>
): SourceAdvancedSectionState {
    if (databaseType == null) {
        return {
            declaredAdvancedFields: [],
            fallbackAdvancedEntries: [],
            hasAdvancedSection: false,
        };
    }

    const sslAdvancedKeys = getSSLAdvancedKeys();
    const excludedAdvancedKeys = new Set<string>([
        ...sslAdvancedKeys,
        ...promotedKeys,
    ]);

    const declaredAdvancedFields = (databaseType.connectionFields ?? []).filter(field =>
        field.Section === SourceConnectionFieldSection.Advanced &&
        !excludedAdvancedKeys.has(field.Key)
    );

    const declaredAdvancedFieldKeys = new Set<string>(declaredAdvancedFields.map(field => field.Key));
    const fallbackAdvancedEntries = Object.entries(advancedForm).filter(([key]) =>
        !excludedAdvancedKeys.has(key) && !declaredAdvancedFieldKeys.has(key)
    );

    return {
        declaredAdvancedFields,
        fallbackAdvancedEntries,
        hasAdvancedSection: declaredAdvancedFields.length > 0 ||
            fallbackAdvancedEntries.length > 0 ||
            (databaseType.sslModes?.length ?? 0) > 0,
    };
}

function looksLikeConnectionString(input: string): boolean {
    const trimmed = input.trim();
    return trimmed.includes('://') || trimmed.startsWith('jdbc:') || /^[a-z_][a-z0-9_]*=.+/i.test(trimmed);
}

function resolveConnparseProvider(databaseType: SourceTypeItem): string | undefined {
    return databaseType.connector || databaseType.id || undefined;
}

function normalizeConnectionFieldKey(key: string): string {
    return key.toLowerCase().replace(/[^a-z0-9]+/g, '');
}

function queryValueToInputValue(value: QueryValue): string {
    return Array.isArray(value) ? value[value.length - 1] ?? '' : value;
}

function buildQueryString(query: Record<string, QueryValue>): string {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(query)) {
        if (Array.isArray(value)) {
            for (const item of value) {
                params.append(key, item);
            }
            continue;
        }
        params.append(key, value);
    }
    const serialized = params.toString();
    return serialized.length > 0 ? `?${serialized}` : '';
}

function filterQueryByHandledKeys(
    query: Record<string, QueryValue>,
    handledQueryKeys: ReadonlySet<string>
): Record<string, QueryValue> {
    return Object.fromEntries(
        Object.entries(query).filter(([key]) => !handledQueryKeys.has(key))
    );
}
function singleEndpointFromAddress(address: ConnparseAddress): { hostName: string; port: number | null } | null {
    const hosts = Array.isArray(address.authority.hosts) ? address.authority.hosts : null;
    if (hosts != null && hosts.length > 0) {
        return null;
    }

    let hostName = typeof address.authority.host === 'string' ? address.authority.host.trim() : '';
    let port = typeof address.authority.port === 'number' ? address.authority.port : null;

    const tcpMatch = /^tcp\((.+)\)$/i.exec(hostName);
    if (tcpMatch != null) {
        try {
            const tcpUrl = new URL(`tcp://${tcpMatch[1]}`);
            hostName = tcpUrl.hostname;
            port = tcpUrl.port.length > 0 ? Number(tcpUrl.port) : port;
        } catch {
            return null;
        }
    }

    return {
        hostName,
        port,
    };
}
