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

import { Input, Label } from '@clidey/ux';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from '../hooks/use-translation';
import { SearchSelect } from './ux';
import {
    findColumnTypeDefinition,
    formatTypeSpec,
    getSourceColumnTypeDefinitions,
    parseTypeSpec,
} from '../utils/source-column-types';
import type { TypeDefinition } from '../config/source-types';

export interface TypeSelectorProps {
    /** The source type identifier used to load backend-owned type definitions. */
    sourceType: string | undefined;
    /** Current value (full type spec like "VARCHAR(255)") */
    value: string;
    /** Called when the type changes */
    onChange: (value: string) => void;
    /** Placeholder for the type dropdown */
    placeholder?: string;
    /** Search placeholder for the type dropdown */
    searchPlaceholder?: string;
    /** Additional button props for the dropdown */
    buttonProps?: Record<string, unknown>;
}

/**
 * TypeSelector component for selecting source-owned column types with optional length/precision inputs.
 *
 * Shows a dropdown of canonical types from backend source metadata, and conditionally
 * displays length or precision/scale inputs based on the selected type's definition.
 */
export function TypeSelector({
    sourceType,
    value,
    onChange,
    placeholder,
    searchPlaceholder,
    buttonProps,
}: TypeSelectorProps) {
    const { t } = useTranslation('pages/storage-unit');

    // Parse the current value into components
    const parsed = useMemo(() => parseTypeSpec(value || ''), [value]);

    // Track the base type and modifiers separately for controlled inputs
    const [baseType, setBaseType] = useState(parsed.baseType);
    const [length, setLength] = useState<number | undefined>(parsed.length);
    const [precision, setPrecision] = useState<number | undefined>(parsed.precision);
    const [scale, setScale] = useState<number | undefined>(parsed.scale);

    const typeDefinitions = useMemo(() => {
        if (!sourceType) return [];
        return getSourceColumnTypeDefinitions(sourceType);
    }, [sourceType]);

    const currentTypeDef = useMemo((): TypeDefinition | undefined => {
        if (!sourceType || !baseType) return undefined;
        return findColumnTypeDefinition(baseType, sourceType);
    }, [sourceType, baseType]);

    // Create dropdown options from type definitions
    const typeOptions = useMemo(() => {
        return typeDefinitions.map(typeDef => ({
            value: typeDef.id,
            label: typeDef.label,
        }));
    }, [typeDefinitions]);

    // Update internal state when external value changes
    useEffect(() => {
        const newParsed = parseTypeSpec(value || '');
        setBaseType(newParsed.baseType);
        setLength(newParsed.length);
        setPrecision(newParsed.precision);
        setScale(newParsed.scale);
    }, [value]);

    // Emit the combined value whenever components change
    const emitValue = useCallback((
        newBaseType: string,
        newLength?: number,
        newPrecision?: number,
        newScale?: number,
    ) => {
        const typeDef = sourceType ? findColumnTypeDefinition(newBaseType, sourceType) : undefined;

        let finalValue: string;
        if (typeDef?.hasPrecision) {
            finalValue = formatTypeSpec(newBaseType, undefined, newPrecision, newScale);
        } else if (typeDef?.hasLength) {
            finalValue = formatTypeSpec(newBaseType, newLength);
        } else {
            finalValue = newBaseType;
        }

        onChange(finalValue);
    }, [sourceType, onChange]);

    // Handle base type change
    const handleTypeChange = useCallback((newType: string) => {
        setBaseType(newType);

        const typeDef = sourceType ? findColumnTypeDefinition(newType, sourceType) : undefined;

        let newLength: number | undefined;
        let newPrecision: number | undefined;
        let newScale: number | undefined;

        if (typeDef?.hasLength) {
            newLength = typeDef.defaultLength ?? 255;
            setLength(newLength);
        } else {
            setLength(undefined);
        }

        if (typeDef?.hasPrecision) {
            newPrecision = typeDef.defaultPrecision ?? 10;
            setPrecision(newPrecision);
            setScale(0);
            newScale = 0;
        } else {
            setPrecision(undefined);
            setScale(undefined);
        }

        emitValue(newType, newLength, newPrecision, newScale);
    }, [sourceType, emitValue]);

    // Handle length change
    const handleLengthChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const newLength = e.target.value ? parseInt(e.target.value, 10) : undefined;
        setLength(newLength);
        emitValue(baseType, newLength, precision, scale);
    }, [baseType, precision, scale, emitValue]);

    // Handle precision change
    const handlePrecisionChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const newPrecision = e.target.value ? parseInt(e.target.value, 10) : undefined;
        setPrecision(newPrecision);
        emitValue(baseType, length, newPrecision, scale);
    }, [baseType, length, scale, emitValue]);

    // Handle scale change
    const handleScaleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const newScale = e.target.value ? parseInt(e.target.value, 10) : undefined;
        setScale(newScale);
        emitValue(baseType, length, precision, newScale);
    }, [baseType, length, precision, emitValue]);

    // If no type definitions, fall back to just returning the value as-is
    if (typeDefinitions.length === 0) {
        return (
            <Input
                value={value}
                onChange={e => onChange(e.target.value)}
                placeholder={placeholder}
            />
        );
    }

    return (
        <div className="flex flex-col gap-2">
            <SearchSelect
                options={typeOptions}
                value={baseType}
                onChange={handleTypeChange}
                placeholder={placeholder}
                searchPlaceholder={searchPlaceholder}
                buttonProps={buttonProps}
            />

            {currentTypeDef?.hasLength && (
                <div className="flex items-center gap-2">
                    <Label className="min-w-16">{t('fieldLengthLabel')}</Label>
                    <Input
                        type="number"
                        min={1}
                        value={length ?? ''}
                        onChange={handleLengthChange}
                        placeholder={t('fieldLengthPlaceholder')}
                        className="w-24"
                    />
                </div>
            )}

            {currentTypeDef?.hasPrecision && (
                <div className="flex items-center gap-2">
                    <Label className="min-w-16">{t('fieldPrecisionLabel')}</Label>
                    <Input
                        type="number"
                        min={1}
                        value={precision ?? ''}
                        onChange={handlePrecisionChange}
                        placeholder="10"
                        className="w-20"
                    />
                    <Label className="min-w-12">{t('fieldScaleLabel')}</Label>
                    <Input
                        type="number"
                        min={0}
                        value={scale ?? ''}
                        onChange={handleScaleChange}
                        placeholder="0"
                        className="w-16"
                    />
                </div>
            )}
        </div>
    );
}
