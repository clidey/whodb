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

import { Input, Label, Switch } from '@clidey/ux';
import { useQuery } from '@apollo/client/react';
import { SearchSelect } from '@/components/ux';
import { SourceConnectionFieldKind, SourceConnectionFieldOptionsDocument } from '@graphql';
import { useMemo, type FC } from 'react';
import type { SourceTypeItem } from '@/config/source-types';
import { SSLConfig } from '@/components/ssl-config';
import type { SourceAdvancedSectionState } from '@/utils/source-connection-form';

function toCamelCase(value: string): string {
    const parts = value
        .replace(/[-_]/g, ' ')
        .trim()
        .split(/\s+/)
        .filter(Boolean)
        .map(part => part.toLowerCase());

    return parts.map((part, index) => {
        if (index === 0) {
            return part;
        }
        return part.charAt(0).toUpperCase() + part.slice(1);
    }).join('');
}

/**
 * Shared advanced-field renderer for source connection forms.
 */
export interface SourceAdvancedFieldsProps {
    databaseType: SourceTypeItem;
    advancedState: SourceAdvancedSectionState;
    advancedForm: Record<string, string>;
    sourceValues?: Record<string, string>;
    onAdvancedFormChange: (key: string, value: string) => void;
    translate: (key: string) => string;
    showPasswordToggle?: boolean;
    fieldClassName?: string;
    checkboxClassName?: string;
}

/**
 * Renders the non-primary advanced fields plus the shared SSL configuration section.
 *
 * @param props Advanced-field rendering props.
 * @returns Advanced-field content for one source form.
 */
export const SourceAdvancedFields: FC<SourceAdvancedFieldsProps> = ({
    databaseType,
    advancedState,
    advancedForm,
    sourceValues,
    onAdvancedFormChange,
    translate,
    showPasswordToggle = true,
    fieldClassName = 'grid gap-2',
    checkboxClassName = 'flex items-center justify-between gap-4',
}) => {
    return (
        <>
            {advancedState.declaredAdvancedFields.map(field => {
                const value = advancedForm[field.Key] ?? field.DefaultValue ?? '';
                if (field.Kind === SourceConnectionFieldKind.Boolean) {
                    return (
                        <div className={checkboxClassName} key={field.Key}>
                            <Label>{translate(field.LabelKey)}</Label>
                            <Switch
                                checked={value.toLowerCase() === 'true'}
                                onCheckedChange={(checked) => { onAdvancedFormChange(field.Key, checked ? 'true' : 'false'); }}
                            />
                        </div>
                    );
                }

                return field.SupportsOptions ? (
                    <SourceConnectionOptionsField
                        key={field.Key}
                        databaseType={databaseType}
                        field={field}
                        value={value}
                        sourceValues={sourceValues ?? advancedForm}
                        onChange={(nextValue) => { onAdvancedFormChange(field.Key, nextValue); }}
                        translate={translate}
                        fieldClassName={fieldClassName}
                    />
                ) : (
                    <div className={fieldClassName} key={field.Key}>
                        <Label>{translate(field.LabelKey)}</Label>
                        <Input
                            value={value}
                            onChange={(e) => { onAdvancedFormChange(field.Key, e.target.value); }}
                            data-testid={`${field.Key}-input`}
                            type={field.Kind === SourceConnectionFieldKind.Password ? 'password' : 'text'}
                            placeholder={field.PlaceholderKey ? translate(field.PlaceholderKey) : undefined}
                            showPasswordToggle={field.Kind === SourceConnectionFieldKind.Password && showPasswordToggle}
                        />
                    </div>
                );
            })}
            {advancedState.fallbackAdvancedEntries.map(([key, value]) => (
                <div className={fieldClassName} key={key}>
                    <Label>{translate(`advancedFields.${toCamelCase(key)}`)}</Label>
                    <Input
                        value={value}
                        onChange={(e) => { onAdvancedFormChange(key, e.target.value); }}
                        data-testid={`${key}-input`}
                    />
                </div>
            ))}
            <SSLConfig
                supportsCustomCAContent={databaseType.traits?.connection.supportsCustomCAContent ?? true}
                sslModes={databaseType.sslModes}
                advancedForm={advancedForm}
                onAdvancedFormChange={onAdvancedFormChange}
            />
        </>
    );
};

const SourceConnectionOptionsField: FC<{
    databaseType: SourceTypeItem;
    field: SourceAdvancedSectionState['declaredAdvancedFields'][number];
    value: string;
    sourceValues: Record<string, string>;
    onChange: (value: string) => void;
    translate: (key: string) => string;
    fieldClassName: string;
}> = ({ databaseType, field, value, sourceValues, onChange, translate, fieldClassName }) => {
    const values = useMemo(
        () => Object.entries(sourceValues).map(([Key, Value]) => ({ Key, Value })),
        [sourceValues],
    );
    const { data, loading } = useQuery(SourceConnectionFieldOptionsDocument, {
        variables: { sourceType: databaseType.id, fieldKey: field.Key, values },
    });
    const options = data?.SourceFieldOptions ?? (value === '' ? [] : [value]);

    return (
        <div className={fieldClassName}>
            <Label>{translate(field.LabelKey)}</Label>
            <SearchSelect
                value={value}
                onChange={onChange}
                disabled={loading}
                options={options.map(option => ({
                    value: option,
                    label: option.replace(/(^|[-_\s])(\p{L})/gu, (_, prefix: string, letter: string) => `${prefix}${letter.toUpperCase()}`),
                }))}
                contentClassName="w-[var(--radix-popover-trigger-width)]"
            />
        </div>
    );
};
