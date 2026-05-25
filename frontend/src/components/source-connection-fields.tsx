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

import { Button, cn, Input, Label, Switch } from '@clidey/ux';
import { ph } from '@/utils/privacy';
import {
    SourceConnectionFieldKind,
    SourceConnectionFieldSection,
} from '@graphql';
import type { ReactElement, Ref } from 'react';
import { SearchSelect } from '@/components/ux';
import { ChevronDownIcon, CircleStackIcon } from '@/components/heroicons';
import type { SourceTypeItem } from '@/config/source-types';
import {
    findConnectionFieldByKey,
    usesFileTransport,
} from '@/utils/source-connection-form';

type SourceConnectionFieldItem = NonNullable<SourceTypeItem['connectionFields']>[number];

const STANDARD_PRIMARY_KEYS = new Set([
    'hostname',
    'port',
    'username',
    'password',
    'database',
    'search path',
]);

/**
 * Option rendered in a selectable source connection field.
 */
export interface SourceConnectionFieldOption {
    value: string;
    label: string;
    icon?: ReactElement;
}

/**
 * Props for the shared standard source connection-field renderer.
 */
export interface SourceConnectionFieldsProps {
    databaseType: SourceTypeItem;
    hostName: string;
    onHostNameChange: (value: string) => void;
    onHostNamePaste?: (value: string) => boolean;
    username: string;
    setUsername: (value: string) => void;
    password: string;
    setPassword: (value: string) => void;
    database: string;
    setDatabase: (value: string) => void;
    advancedForm: Record<string, string>;
    onAdvancedFormChange: (key: string, value: string) => void;
    translate: (key: string) => string;
    layout?: 'login' | 'sheet';
    promotedKeys?: ReadonlySet<string>;
    portValue?: string;
    onPortChange?: (value: string) => void;
    usernameInputRef?: Ref<HTMLInputElement>;
    showPasswordToggle?: boolean;
    passwordPlaceholder?: string;
    onPasswordFocus?: () => void;
    isDesktop?: boolean;
    onBrowseDatabaseFile?: () => void;
    databaseOptions?: SourceConnectionFieldOption[];
    databaseOptionsLoading?: boolean;
    hasError?: boolean;
    errorId?: string;
}

/**
 * Renders the standard backend-declared primary connection fields for a source.
 *
 * @param props Connection-field rendering props.
 * @returns Standard source connection field inputs.
 */
export function SourceConnectionFields({
    databaseType,
    hostName,
    onHostNameChange,
    onHostNamePaste,
    username,
    setUsername,
    password,
    setPassword,
    database,
    setDatabase,
    advancedForm,
    onAdvancedFormChange,
    translate,
    layout = 'login',
    promotedKeys,
    portValue,
    onPortChange,
    usernameInputRef,
    showPasswordToggle = true,
    passwordPlaceholder,
    onPasswordFocus,
    isDesktop = false,
    onBrowseDatabaseFile,
    databaseOptions = [],
    databaseOptionsLoading = false,
    hasError = false,
    errorId,
}: SourceConnectionFieldsProps): ReactElement {
    const hostnameField = findConnectionFieldByKey(databaseType, 'Hostname');
    const portField = promotedConnectionField(databaseType, 'Port', promotedKeys);
    const usernameField = findConnectionFieldByKey(databaseType, 'Username');
    const passwordField = findConnectionFieldByKey(databaseType, 'Password');
    const databaseField = findConnectionFieldByKey(databaseType, 'Database');
    const searchPathField = promotedConnectionField(databaseType, 'Search Path', promotedKeys);
    const port = portValue ?? advancedForm.Port ?? portField?.DefaultValue ?? '';
    const setPort = onPortChange ?? ((value: string) => onAdvancedFormChange('Port', value));
    const containerClassName = cn(layout === 'login' ? 'flex flex-col gap-lg w-full' : 'space-y-4', ph.noCapture);
    const fieldClassName = layout === 'login' ? 'flex flex-col gap-sm w-full' : 'grid gap-2';

    if (usesFileTransport(databaseType) && databaseField != null) {
        return (
            <div className={containerClassName}>
                <div className={fieldClassName}>
                    <Label htmlFor="source-database">{translate(databaseField.LabelKey)}</Label>
                    {isDesktop && onBrowseDatabaseFile != null ? (
                        <div className="flex flex-col gap-sm w-full">
                            <Input
                                id="source-database"
                                value={database}
                                onChange={(e) => setDatabase(e.target.value)}
                                placeholder={fieldPlaceholder(databaseField, translate)}
                                data-testid="database"
                                aria-required={databaseField.Required ? 'true' : undefined}
                                aria-invalid={hasError ? 'true' : undefined}
                                aria-describedby={hasError ? errorId : undefined}
                            />
                            <Button
                                onClick={onBrowseDatabaseFile}
                                variant="outline"
                                className="w-full"
                            >
                                {translate('browseForSqliteFile')}
                            </Button>
                        </div>
                    ) : layout === 'login' || databaseOptions.length > 0 || databaseOptionsLoading ? (
                        <SearchSelect
                            value={database}
                            onChange={setDatabase}
                            disabled={databaseOptionsLoading}
                            options={databaseOptionsLoading ? [] : databaseOptions}
                            placeholder={translate('selectDatabase')}
                            buttonProps={{
                                'data-testid': 'database',
                                'aria-required': databaseField.Required ? 'true' : undefined,
                                'aria-invalid': hasError ? 'true' : undefined,
                                'aria-describedby': hasError ? errorId : undefined,
                            }}
                            contentClassName="w-[var(--radix-popover-trigger-width)] login-select-popover"
                            rightIcon={<ChevronDownIcon className="w-4 h-4" />}
                        />
                    ) : (
                        <Input
                            id="source-database"
                            value={database}
                            onChange={(e) => setDatabase(e.target.value)}
                            placeholder={fieldPlaceholder(databaseField, translate)}
                            data-testid="database"
                            aria-required={databaseField.Required ? 'true' : undefined}
                            aria-invalid={hasError ? 'true' : undefined}
                            aria-describedby={hasError ? errorId : undefined}
                        />
                    )}
                </div>
            </div>
        );
    }

    const customPrimaryFields = (databaseType.connectionFields ?? []).filter(field =>
        field.Section === SourceConnectionFieldSection.Primary &&
        !STANDARD_PRIMARY_KEYS.has(field.Key.toLowerCase()) &&
        shouldPromoteField(field, promotedKeys)
    );

    return (
        <div className={containerClassName}>
            {hostnameField != null && shouldPromoteField(hostnameField, promotedKeys) && (
                <div className={layout === 'login' ? 'flex gap-sm w-full items-end' : 'grid gap-2'}>
                    <div className={layout === 'login' ? 'flex flex-col gap-sm flex-1' : 'grid gap-2'}>
                        <Label htmlFor="source-hostname">
                            {translate('hostNameOrUrl')}
                        </Label>
                        <Input
                            id="source-hostname"
                            value={hostName}
                            onChange={(e) => onHostNameChange(e.target.value)}
                            onPaste={(event) => {
                                const pastedValue = event.clipboardData.getData('text');
                                if (pastedValue.length > 0 && onHostNamePaste?.(pastedValue)) {
                                    event.preventDefault();
                                }
                            }}
                            data-testid="hostname"
                            placeholder={fieldPlaceholder(hostnameField, translate)}
                            aria-required={hostnameField.Required ? 'true' : undefined}
                            aria-invalid={hasError ? 'true' : undefined}
                            aria-describedby={hasError ? errorId : undefined}
                        />
                    </div>
                    {portField != null && layout === 'login' && (
                        <div className="flex flex-col gap-sm w-24">
                            <Label htmlFor="source-port">{translate(portField.LabelKey)}</Label>
                            <Input
                                id="source-port"
                                value={port}
                                onChange={(e) => setPort(e.target.value)}
                                data-testid="port"
                                placeholder={portField.DefaultValue ?? ''}
                            />
                        </div>
                    )}
                </div>
            )}
            <div className={layout === 'login' ? 'contents' : 'grid grid-cols-2 gap-4'}>
                {usernameField != null && shouldPromoteField(usernameField, promotedKeys) && (
                    <div className={fieldClassName}>
                        <Label htmlFor="source-username">{translate(usernameField.LabelKey)}</Label>
                        <Input
                            ref={usernameInputRef}
                            id="source-username"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            data-testid="username"
                            placeholder={fieldPlaceholder(usernameField, translate)}
                            aria-required={usernameField.Required ? 'true' : undefined}
                            aria-invalid={hasError ? 'true' : undefined}
                            aria-describedby={hasError ? errorId : undefined}
                        />
                    </div>
                )}
                {passwordField != null && shouldPromoteField(passwordField, promotedKeys) && (
                    <div className={fieldClassName}>
                        <Label htmlFor="source-password">{translate(passwordField.LabelKey)}</Label>
                        <Input
                            id="source-password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            onFocus={onPasswordFocus}
                            type="password"
                            data-testid="password"
                            placeholder={passwordPlaceholder ?? fieldPlaceholder(passwordField, translate)}
                            aria-required={passwordField.Required ? 'true' : undefined}
                            aria-invalid={hasError ? 'true' : undefined}
                            aria-describedby={hasError ? errorId : undefined}
                            showPasswordToggle={showPasswordToggle}
                        />
                    </div>
                )}
            </div>
            {databaseField != null && shouldPromoteField(databaseField, promotedKeys) && (
                <div className={fieldClassName}>
                    <Label htmlFor="source-database">{translate(databaseField.LabelKey)}</Label>
                    <Input
                        id="source-database"
                        value={database}
                        onChange={(e) => setDatabase(e.target.value)}
                        data-testid="database"
                        placeholder={fieldPlaceholder(databaseField, translate)}
                        aria-required={databaseField.Required ? 'true' : undefined}
                        aria-invalid={hasError ? 'true' : undefined}
                        aria-describedby={hasError ? errorId : undefined}
                    />
                </div>
            )}
            {portField != null && layout === 'sheet' && (
                <div className={fieldClassName}>
                    <Label htmlFor="source-port">{translate(portField.LabelKey)}</Label>
                    <Input
                        id="source-port"
                        value={port}
                        onChange={(e) => setPort(e.target.value)}
                        data-testid="port"
                        placeholder={portField.DefaultValue ?? ''}
                    />
                </div>
            )}
            {searchPathField != null && (
                <div className={fieldClassName}>
                    <Label htmlFor="source-search-path">{translate(searchPathField.LabelKey)}</Label>
                    <Input
                        id="source-search-path"
                        value={advancedForm['Search Path'] ?? ''}
                        onChange={(e) => onAdvancedFormChange('Search Path', e.target.value)}
                        data-testid="search-path"
                        placeholder={fieldPlaceholder(searchPathField, translate)}
                        aria-required={searchPathField.Required ? 'true' : undefined}
                    />
                </div>
            )}
            {customPrimaryFields.map(field => (
                <PrimaryAdvancedValueField
                    key={field.Key}
                    field={field}
                    value={advancedForm[field.Key] ?? field.DefaultValue ?? ''}
                    onChange={(value) => onAdvancedFormChange(field.Key, value)}
                    translate={translate}
                    className={fieldClassName}
                    showPasswordToggle={showPasswordToggle}
                />
            ))}
        </div>
    );
}

function promotedConnectionField(
    databaseType: SourceTypeItem,
    key: string,
    promotedKeys: ReadonlySet<string> | undefined
): SourceConnectionFieldItem | undefined {
    const field = findConnectionFieldByKey(databaseType, key);
    if (field == null || !shouldPromoteField(field, promotedKeys)) {
        return undefined;
    }
    return field;
}

function shouldPromoteField(
    field: SourceConnectionFieldItem,
    promotedKeys: ReadonlySet<string> | undefined
): boolean {
    if (promotedKeys == null) {
        return field.Section === SourceConnectionFieldSection.Primary;
    }
    return promotedKeys.has(field.Key);
}

function fieldPlaceholder(
    field: SourceConnectionFieldItem,
    translate: (key: string) => string
): string | undefined {
    return field.PlaceholderKey ? translate(field.PlaceholderKey) : undefined;
}

function PrimaryAdvancedValueField({
    field,
    value,
    onChange,
    translate,
    className,
    showPasswordToggle,
}: {
    field: SourceConnectionFieldItem;
    value: string;
    onChange: (value: string) => void;
    translate: (key: string) => string;
    className: string;
    showPasswordToggle: boolean;
}): ReactElement {
    if (field.Kind === SourceConnectionFieldKind.Boolean) {
        return (
            <div className="flex items-center justify-between gap-4">
                <Label>{translate(field.LabelKey)}</Label>
                <Switch
                    checked={value.toLowerCase() === 'true'}
                    onCheckedChange={checked => onChange(checked ? 'true' : 'false')}
                />
            </div>
        );
    }

    return (
        <div className={className}>
            <Label htmlFor={`source-${field.Key}`}>{translate(field.LabelKey)}</Label>
            <Input
                id={`source-${field.Key}`}
                value={value}
                onChange={(e) => onChange(e.target.value)}
                data-testid={`${field.Key}-input`}
                type={field.Kind === SourceConnectionFieldKind.Password ? 'password' : 'text'}
                placeholder={fieldPlaceholder(field, translate)}
                showPasswordToggle={field.Kind === SourceConnectionFieldKind.Password && showPasswordToggle}
            />
        </div>
    );
}

/**
 * Builds selectable database options for standard source connection fields.
 *
 * @param values Database names returned by the backend field-options query.
 * @returns Options ready for the shared connection-field renderer.
 */
export function buildDatabaseFieldOptions(values: readonly string[] | null | undefined): SourceConnectionFieldOption[] {
    return values?.map(value => ({
        value,
        label: value,
        icon: <CircleStackIcon className="w-4 h-4" />,
    })) ?? [];
}
