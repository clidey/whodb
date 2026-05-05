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

import { SourceConnectionFieldSection, SourceConnectionTransport } from '@graphql';
import type { SourceTypeItem } from '@/config/source-types';
import { getSSLAdvancedKeys } from '@/utils/source-ssl';

type SourceConnectionFieldItem = NonNullable<SourceTypeItem['connectionFields']>[number];

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
