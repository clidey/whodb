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

import { useMutation, useQuery } from "@apollo/client/react";
import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
    Button,
    Checkbox,
    Input,
    Label,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Separator,
    SheetTitle,
    toast,
} from "@clidey/ux";
import {
    CreateSourceObjectFromDefinitionDocument,
    SourceObjectCreationMetadataDocument,
    TypeCategory,
    type ColumnDefinitionInput,
    type RecordInput,
    type SourceObjectRefInput,
} from "@graphql";
import type { FC} from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { TypeSelector } from "../../components/type-selector";
import { CheckCircleIcon, PlusCircleIcon, XCircleIcon } from "../../components/heroicons";
import { findColumnTypeDefinition } from "../../utils/source-column-types";
import { trackFrontendEvent } from "../../config/posthog";
import { useTranslation } from "../../hooks/use-translation";

type ColumnFormState = {
    Name: string;
    Type: string;
    Nullable: boolean;
    Primary: boolean;
    Unique: boolean;
    Identity: boolean;
    DefaultValue: string;
    CheckValues: string;
    CheckMin: string;
    CheckMax: string;
    ForeignKeyTable: string;
    ForeignKeyColumn: string;
};

type CreateSourceObjectCardProps = {
    databaseType?: string;
    parentRef?: SourceObjectRefInput;
    referenceColumnsByName?: Record<string, string[]>;
    referenceObjects?: { name: string }[];
    singularStorageUnitLabel: string;
    onCreated: () => void;
    onErrorChange: (error?: string) => void;
    onClose: () => void;
};

const emptyColumn = (type = ""): ColumnFormState => ({
    Name: "",
    Type: type,
    Nullable: false,
    Primary: false,
    Unique: false,
    Identity: false,
    DefaultValue: "",
    CheckValues: "",
    CheckMin: "",
    CheckMax: "",
    ForeignKeyTable: "",
    ForeignKeyColumn: "",
});

type CreationTypeDefinition = {
    id: string;
    label: string;
    category: TypeCategory;
};

const defaultValueCategories = new Set<TypeCategory>([
    TypeCategory.Boolean,
    TypeCategory.Datetime,
    TypeCategory.Numeric,
    TypeCategory.Other,
    TypeCategory.Text,
]);

function supportsDefaultForType(typeDef: CreationTypeDefinition | undefined): boolean {
    return typeDef != null && defaultValueCategories.has(typeDef.category);
}

function supportsCheckValuesForType(typeDef: CreationTypeDefinition | undefined): boolean {
    if (typeDef == null) {
        return false;
    }
    const typeName = `${typeDef.id} ${typeDef.label}`.toLowerCase();
    return typeName.includes("enum");
}

function supportsCheckRangeForType(typeDef: CreationTypeDefinition | undefined): boolean {
    return typeDef?.category === TypeCategory.Numeric;
}

/**
 * Renders the metadata-driven source object creation form.
 */
export const CreateSourceObjectCard: FC<CreateSourceObjectCardProps> = ({
    databaseType,
    parentRef,
    referenceColumnsByName,
    referenceObjects = [],
    singularStorageUnitLabel,
    onCreated,
    onErrorChange,
    onClose,
}) => {
    const { t } = useTranslation("pages/storage-unit");
    const [name, setName] = useState("");
    const [columns, setColumns] = useState<ColumnFormState[]>([emptyColumn()]);
    const [tableOptions, setTableOptions] = useState<Record<string, string>>({});
    const [createSourceObject] = useMutation(CreateSourceObjectFromDefinitionDocument);
    const { data } = useQuery(SourceObjectCreationMetadataDocument, {
        variables: { parent: parentRef },
    });
    const metadata = data?.SourceObjectCreationMetadata;
    const firstType = metadata?.TypeDefinitions[0]?.id ?? "";
    const capabilities = metadata?.ColumnCapabilities;
    const labels = metadata?.ColumnLabels;
    const typeDefinitions = metadata?.TypeDefinitions ?? [];

    useEffect(() => {
        if (!firstType) {
            return;
        }
        setColumns(current => current.map(column => column.Type === "" ? { ...column, Type: firstType } : column));
    }, [firstType]);

    useEffect(() => {
        const initialOptions = Object.fromEntries(
            (metadata?.TableOptions ?? []).map(option => [option.Key, option.Values[0] ?? ""])
        );
        setTableOptions(initialOptions);
    }, [metadata?.TableOptions]);

    const showColumns = (metadata?.RequiresColumns ?? true) || (metadata?.TypeDefinitions.length ?? 0) > 0;

    const handleColumnChange = useCallback(<K extends keyof ColumnFormState>(index: number, key: K, value: ColumnFormState[K]) => {
        setColumns(current => current.map((column, columnIndex) => columnIndex === index ? { ...column, [key]: value } : column));
    }, []);

    const handleAddColumn = useCallback(() => {
        setColumns(current => [...current, emptyColumn(firstType)]);
        void trackFrontendEvent("ui.storage_unit_field_added", {
            database_type: databaseType ?? "unknown",
        });
    }, [databaseType, firstType]);

    const handleRemoveColumn = useCallback((index: number) => {
        setColumns(current => current.length <= 1 ? current : current.filter((_, columnIndex) => columnIndex !== index));
    }, []);

    const definitionColumns = useMemo<ColumnDefinitionInput[]>(() => {
        if (!showColumns) {
            return [];
        }
        const activeColumns = metadata?.RequiresColumns
            ? columns
            : columns.filter(column => column.Name.trim() !== "");
        return activeColumns.map(column => {
            const typeDef = typeDefinitions.find(definition => definition.id === column.Type);
            return {
                Name: column.Name,
                Type: column.Type,
                Nullable: capabilities?.Nullable ? column.Nullable : undefined,
                Primary: capabilities?.PrimaryKey ? column.Primary : false,
                Unique: capabilities?.Unique ? column.Unique : false,
                Identity: capabilities?.Identity ? column.Identity : false,
                DefaultValue: capabilities?.DefaultValue && supportsDefaultForType(typeDef) && column.DefaultValue.trim() !== "" ? column.DefaultValue.trim() : undefined,
                CheckValues: capabilities?.CheckValues && supportsCheckValuesForType(typeDef) && column.CheckValues.trim() !== ""
                    ? column.CheckValues.split(",").map(value => value.trim()).filter(Boolean)
                    : undefined,
                CheckMin: capabilities?.CheckMinMax && supportsCheckRangeForType(typeDef) && column.CheckMin.trim() !== "" ? Number(column.CheckMin) : undefined,
                CheckMax: capabilities?.CheckMinMax && supportsCheckRangeForType(typeDef) && column.CheckMax.trim() !== "" ? Number(column.CheckMax) : undefined,
                ForeignKey: capabilities?.ForeignKey && column.ForeignKeyTable.trim() !== "" && column.ForeignKeyColumn.trim() !== ""
                    ? { Table: column.ForeignKeyTable.trim(), Column: column.ForeignKeyColumn.trim() }
                    : undefined,
            };
        });
    }, [capabilities, columns, metadata?.RequiresColumns, showColumns, typeDefinitions]);

    const definitionTableOptions = useMemo<RecordInput[]>(() => {
        return Object.entries(tableOptions)
            .filter(([, value]) => value !== "")
            .map(([Key, Value]) => ({ Key, Value }));
    }, [tableOptions]);

    const handleSubmit = useCallback(() => {
        if (name.trim().length === 0) {
            onErrorChange(t("nameRequired"));
            return;
        }
        if (metadata?.RequiresColumns && definitionColumns.some(column => column.Name.trim().length === 0 || column.Type.trim().length === 0)) {
            onErrorChange(t("fieldsCannotBeEmpty"));
            return;
        }
        onErrorChange(undefined);
        createSourceObject({
            variables: {
                parent: parentRef,
                definition: {
                    Name: name.trim(),
                    Columns: definitionColumns,
                    TableOptions: definitionTableOptions,
                },
            },
            onCompleted() {
                toast.success(t("createSuccessMessage", { storageUnit: `${singularStorageUnitLabel} ${name.trim()}` }));
                void trackFrontendEvent("ui.storage_unit_created", {
                    database_type: databaseType ?? "unknown",
                    field_count: definitionColumns.length,
                });
                setName("");
                setColumns([emptyColumn(firstType)]);
                onCreated();
                onClose();
            },
            onError(error) {
                toast.error(error.message);
            },
        });
    }, [
	createSourceObject,
	databaseType,
	definitionColumns,
	definitionTableOptions,
	firstType,
	metadata?.RequiresColumns,
	name,
	onClose,
	onCreated,
	onErrorChange,
	parentRef,
	singularStorageUnitLabel,
	t
]);

    if (metadata?.Supported === false) {
        return null;
    }

    return <div className="flex h-full min-h-0 grow flex-col my-2 gap-4 overflow-hidden">
        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto overflow-x-hidden pr-1">
            <SheetTitle className="flex items-center gap-2">
                <PlusCircleIcon className="w-5 h-5" />
                {t("createTitle", { storageUnit: singularStorageUnitLabel })}
            </SheetTitle>
            <div className="flex flex-col gap-2">
                <Label>{t("nameLabel")}</Label>
                <Input value={name} onChange={event => setName(event.target.value)} placeholder={t("namePlaceholder")} />
            </div>
            {metadata?.TableOptions.map(option => (
                <div className="flex flex-col gap-2" key={option.Key}>
                    <Label>{option.Label}</Label>
                    <Select value={tableOptions[option.Key] ?? ""} onValueChange={value => setTableOptions(current => ({ ...current, [option.Key]: value }))}>
                        <SelectTrigger className="w-full">
                            <SelectValue placeholder={option.Label} />
                        </SelectTrigger>
                        <SelectContent>
                            {option.Values.map(value => (
                                <SelectItem key={value} value={value}>{value}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            ))}
            {showColumns && <div className="flex flex-col gap-sm">
                <div className="flex flex-col gap-4">
                    {columns.map((column, index) => {
                        const typeDef = databaseType && column.Type ? findColumnTypeDefinition(column.Type, databaseType) : undefined;
                        const creationTypeDef = typeDefinitions.find(definition => definition.id === column.Type);
                        const showDefaultValue = capabilities?.DefaultValue && supportsDefaultForType(creationTypeDef);
                        const showCheckValues = capabilities?.CheckValues && supportsCheckValuesForType(creationTypeDef);
                        const showCheckRange = capabilities?.CheckMinMax && supportsCheckRangeForType(creationTypeDef);
                        const showModifiers = capabilities?.PrimaryKey || capabilities?.Nullable || capabilities?.Unique || capabilities?.Identity;
                        const showAdvancedOptions = showModifiers || showDefaultValue || showCheckValues || showCheckRange || capabilities?.ForeignKey;
                        const foreignKeyColumnOptions = referenceColumnsByName?.[column.ForeignKeyTable] ?? [];
                        return <div className="flex min-w-0 flex-col gap-lg relative" key={`field-${index}`} data-testid="create-field-card">
                            <Label>{t("fieldNameLabel")}</Label>
                            <Input value={column.Name} onChange={event => handleColumnChange(index, "Name", event.target.value)} placeholder={t("fieldNamePlaceholder")} />
                            {capabilities?.Types && <>
                                <Label>{t("fieldTypeLabel")}</Label>
                                <TypeSelector
                                    sourceType={databaseType}
                                    value={column.Type}
                                    onChange={value => handleColumnChange(index, "Type", value)}
                                    placeholder={t("fieldTypePlaceholder")}
                                    searchPlaceholder={t("searchTypePlaceholder")}
                                    buttonProps={{ "data-testid": `field-type-${index}` }}
                                />
                            </>}
                            {typeDef?.tableModel ? <p className="text-xs text-muted-foreground">{t("aggregateKeyHint")}</p> : null}
                            {showAdvancedOptions && <Accordion type="single" collapsible className="w-full">
                                <AccordionItem value={`field-options-${index}`}>
                                    <AccordionTrigger className="py-2" data-testid={`field-options-trigger-${index}`}>
                                        {t("modifiersLabel")}
                                    </AccordionTrigger>
                                    <AccordionContent>
                                        <div className="flex min-w-0 flex-col gap-lg pt-2">
                                            {showModifiers && <div className="grid min-w-0 grid-cols-1 gap-3">
                                                {capabilities?.PrimaryKey && <ModifierCheckbox label={labels?.PrimaryKey ?? t("primaryModifier")} checked={column.Primary} onChange={value => handleColumnChange(index, "Primary", value)} />}
                                                {capabilities?.Nullable && <ModifierCheckbox label={labels?.Nullable ?? t("nullableModifier")} checked={column.Nullable} onChange={value => handleColumnChange(index, "Nullable", value)} />}
                                                {capabilities?.Unique && <ModifierCheckbox label={labels?.Unique ?? t("uniqueModifier")} checked={column.Unique} onChange={value => handleColumnChange(index, "Unique", value)} />}
                                                {capabilities?.Identity && <ModifierCheckbox label={labels?.Identity ?? t("identityModifier")} checked={column.Identity} onChange={value => handleColumnChange(index, "Identity", value)} />}
                                            </div>}
                                            {showDefaultValue && <div className="flex flex-col gap-2">
                                                <Label>{labels?.DefaultValue ?? t("defaultValueLabel")}</Label>
                                                <Input value={column.DefaultValue} onChange={event => handleColumnChange(index, "DefaultValue", event.target.value)} placeholder={t("defaultValuePlaceholder")} />
                                            </div>}
                                            {showCheckValues && <div className="flex flex-col gap-2">
                                                <Label>{labels?.CheckValues ?? t("checkValuesLabel")}</Label>
                                                <Input value={column.CheckValues} onChange={event => handleColumnChange(index, "CheckValues", event.target.value)} placeholder={t("checkValuesPlaceholder")} />
                                            </div>}
                                            {showCheckRange && <div className="grid min-w-0 grid-cols-2 gap-2">
                                                <div className="flex min-w-0 flex-col gap-2">
                                                    <Label>{labels?.CheckMin ?? t("checkMinLabel")}</Label>
                                                    <Input value={column.CheckMin} onChange={event => handleColumnChange(index, "CheckMin", event.target.value)} placeholder={t("checkMinPlaceholder")} />
                                                </div>
                                                <div className="flex min-w-0 flex-col gap-2">
                                                    <Label>{labels?.CheckMax ?? t("checkMaxLabel")}</Label>
                                                    <Input value={column.CheckMax} onChange={event => handleColumnChange(index, "CheckMax", event.target.value)} placeholder={t("checkMaxPlaceholder")} />
                                                </div>
                                            </div>}
                                            {capabilities?.ForeignKey && <div className="grid min-w-0 grid-cols-2 gap-2">
                                                <div className="flex min-w-0 flex-col gap-2">
                                                    <Label>{t("foreignTableLabel")}</Label>
                                                    <Select
                                                        value={column.ForeignKeyTable}
                                                        onValueChange={value => {
                                                            handleColumnChange(index, "ForeignKeyTable", value);
                                                            handleColumnChange(index, "ForeignKeyColumn", "");
                                                        }}
                                                        disabled={referenceObjects.length === 0}
                                                    >
                                                        <SelectTrigger className="w-full" data-testid={`foreign-table-${index}`}>
                                                            <SelectValue placeholder={t("foreignTablePlaceholder")} />
                                                        </SelectTrigger>
                                                        <SelectContent>
                                                            {referenceObjects.map(object => (
                                                                <SelectItem key={object.name} value={object.name}>{object.name}</SelectItem>
                                                            ))}
                                                        </SelectContent>
                                                    </Select>
                                                </div>
                                                <div className="flex min-w-0 flex-col gap-2">
                                                    <Label>{t("foreignColumnLabel")}</Label>
                                                    <Select
                                                        value={column.ForeignKeyColumn}
                                                        onValueChange={value => handleColumnChange(index, "ForeignKeyColumn", value)}
                                                        disabled={column.ForeignKeyTable === "" || foreignKeyColumnOptions.length === 0}
                                                    >
                                                        <SelectTrigger className="w-full" data-testid={`foreign-column-${index}`}>
                                                            <SelectValue placeholder={t("foreignColumnPlaceholder")} />
                                                        </SelectTrigger>
                                                        <SelectContent>
                                                            {foreignKeyColumnOptions.map(columnName => (
                                                                <SelectItem key={columnName} value={columnName}>{columnName}</SelectItem>
                                                            ))}
                                                        </SelectContent>
                                                    </Select>
                                                </div>
                                            </div>}
                                        </div>
                                    </AccordionContent>
                                </AccordionItem>
                            </Accordion>}
                            {columns.length > 1 && <Button variant="destructive" onClick={() => handleRemoveColumn(index)} data-testid="remove-field-button" className="w-full mt-1">
                                <XCircleIcon className="w-4 h-4" /> <span>{t("remove")}</span>
                            </Button>}
                            {index !== columns.length - 1 && <Separator className="mt-2" />}
                        </div>;
                    })}
                </div>
                <Button className="self-end" onClick={handleAddColumn} data-testid="add-field-button" variant="secondary">
                    <PlusCircleIcon className="w-4 h-4" /> {t("addField")}
                </Button>
            </div>}
        </div>
        <Button onClick={handleSubmit} data-testid="submit-button" className="w-full">
            <CheckCircleIcon className="w-4 h-4" /> {t("create")}
        </Button>
    </div>;
};

const ModifierCheckbox: FC<{
    label: string;
    checked: boolean;
    onChange: (value: boolean) => void;
}> = ({ label, checked, onChange }) => (
    <div className="flex min-w-0 items-center gap-2">
        <Checkbox className="shrink-0" checked={checked} onCheckedChange={value => onChange(value === true)} />
        <Label className="block min-w-0 whitespace-normal break-all leading-snug [overflow-wrap:anywhere]">{label}</Label>
    </div>
);
