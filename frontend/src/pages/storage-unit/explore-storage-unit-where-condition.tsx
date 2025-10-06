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

import {
    Badge,
    Button,
    cn,
    Input,
    Label,
    Popover,
    PopoverContent,
    PopoverTrigger,
    SearchSelect,
    Sheet,
    SheetContent,
    SheetFooter,
    SheetTitle
} from "@clidey/ux";
import { AtomicWhereCondition, WhereCondition, WhereConditionType } from '@graphql';
import classNames from "classnames";
import { FC, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { twMerge } from "tailwind-merge";
import { AdjustmentsVerticalIcon, CheckCircleIcon, PlusCircleIcon, XCircleIcon, XMarkIcon } from "../../components/heroicons";

type IPopoverCardProps = {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    currentFilter: AtomicWhereCondition;
    fieldsDropdownItems: { value: string, label: string }[];
    validOperators: { value: string, label: string }[];
    handleFieldSelect: (item: string) => void;
    handleOperatorSelector: (item: string) => void;
    handleInputChange: (newValue: string) => void;
    handleAddFilter: () => void;
    handleSaveFilter?: (index: number) => void;
    handleCancel: () => void;
    className?: string;
    isEditing?: boolean;
    editingIndex?: number;
}

const PopoverCard: FC<IPopoverCardProps> = ({
                                                open,
                                                onOpenChange,
                                                currentFilter,
                                                fieldsDropdownItems,
                                                validOperators,
                                                handleFieldSelect,
                                                handleOperatorSelector,
                                                handleInputChange,
                                                handleAddFilter,
                                                handleSaveFilter,
                                                handleCancel,
                                                className,
                                                isEditing = false,
                                                editingIndex = -1
                                            }) => {
    const handleAction = useCallback(() => {
        if (isEditing && handleSaveFilter && editingIndex !== -1) {
            handleSaveFilter(editingIndex);
        } else {
            handleAddFilter();
        }
    }, [isEditing, handleSaveFilter, editingIndex, handleAddFilter]);

    return  <Popover open={open} onOpenChange={onOpenChange}>
        <PopoverTrigger asChild>
            <div />
        </PopoverTrigger>
        <PopoverContent
            className={cn("flex flex-col gap-md z-[5] py-4 px-6 mt-1 rounded-lg shadow-md min-w-[260px]", className)}
            side="bottom"
            align="center"
            tabIndex={0}
        >
            <div className="flex flex-col gap-sm w-full">
                <Label className="text-xs">
                    Field
                </Label>
                <SearchSelect
                    value={currentFilter.Key}
                    options={fieldsDropdownItems}
                    onChange={handleFieldSelect}
                    contentClassName="w-[var(--radix-popover-trigger-width)]"
                    buttonProps={{
                        "data-testid": "field-key",
                    }}
                />
            </div>
            <div className="flex flex-col gap-sm w-full">
                <Label className="text-xs">
                    Operator
                </Label>
                <SearchSelect
                    value={currentFilter.Operator}
                    options={validOperators}
                    onChange={handleOperatorSelector}
                    contentClassName="w-[var(--radix-popover-trigger-width)]"
                    buttonProps={{
                        "data-testid": "field-operator",
                    }}
                />
            </div>
            <div className="flex flex-col gap-sm w-full">
                <Label className="text-xs">
                    Value
                </Label>
                <Input
                    className="min-w-[150px] w-full"
                    placeholder="Enter filter value"
                    value={currentFilter.Value}
                    onChange={e => handleInputChange(e.target.value)}
                    data-testid="field-value"
                />
            </div>
            <div className="flex gap-sm mt-2">
                <Button
                    className="flex-1"
                    onClick={handleCancel}
                    data-testid="cancel-button"
                    variant="secondary"
                >
                    <XCircleIcon className="w-4 h-4" /> Cancel
                </Button>
                <Button
                    className="flex-1"
                    onClick={handleAction}
                    disabled={
                        !currentFilter.Key ||
                        !currentFilter.Operator ||
                        !currentFilter.Value
                    }
                    data-testid={isEditing ? "update-condition-button" : "add-condition-button"}
                >
                    <CheckCircleIcon className="w-4 h-4"/> {isEditing ? "Update" : "Add"}
                </Button>
            </div>
        </PopoverContent>
    </Popover>
}

type IExploreStorageUnitWhereConditionProps = {
    defaultWhere?: WhereCondition;
    columns: string[];
    operators: string[];
    columnTypes: string[];
    onChange?: (filters: WhereCondition) => void;
}

export const ExploreStorageUnitWhereCondition: FC<IExploreStorageUnitWhereConditionProps> = ({ defaultWhere, columns, columnTypes, onChange, operators }) => {
    const [currentFilter, setCurrentFilter] = useState<AtomicWhereCondition>({ ColumnType: "string", Key: "", Operator: "", Value: "" });
    const [filters, setFilters] = useState<WhereCondition>(defaultWhere ?? {
        Type: WhereConditionType.And,
        And: { Children: [] }
    });
    const [newFilter, setNewFilter] = useState(false);
    const [editingFilter, setEditingFilter] = useState(-1);
    const [sheetOpen, setSheetOpen] = useState(false);
    const [sheetFilters, setSheetFilters] = useState<AtomicWhereCondition[]>([]);
    const newFilterRef = useRef<HTMLDivElement>(null);
    const editFilterRef = useRef<HTMLDivElement>(null);

    // Maximum number of conditions to show in the main view
    const MAX_VISIBLE_CONDITIONS = 2;

    const handleClick = useCallback(() => {
        const shouldShow = !newFilter;
        if (shouldShow) {
            setEditingFilter(-1);
            setCurrentFilter({ ColumnType: "string", Key: "", Operator: "", Value: "" });
        }
        setNewFilter(!newFilter);
    }, [newFilter]);

    const handleCancelNewFilter = useCallback(() => {
        setNewFilter(false);
        setCurrentFilter({ColumnType: "string", Key: "", Operator: "", Value: ""});
    }, []);

    const handleCancelEditFilter = useCallback(() => {
        setEditingFilter(-1);
        setCurrentFilter({ColumnType: "string", Key: "", Operator: "", Value: ""});
    }, []);

    const fieldsDropdownItems = useMemo(() => columns.map(column => ({ value: column, label: column })), [columns]);

    const handleFieldSelect = useCallback((item: string) => {
        setCurrentFilter(val => ({ ...val, Key: item, ColumnType: columnTypes[columns.findIndex(col => col === item)] }));
    }, [columnTypes, columns]);

    const handleOperatorSelector = useCallback((item: string) => {
        setCurrentFilter(val => ({
            ...val,
            Operator: item,
        }));
    }, []);

    const handleInputChange = useCallback((newValue: string) => {
        setCurrentFilter(val => ({ ...val, Value: newValue }));
    }, []);

    const handleAddFilter = useCallback(() => {
        const newAtomicCondition: WhereCondition = {
            Type: WhereConditionType.Atomic,
            Atomic: currentFilter
        };

        const updatedFilters = {
            Type: WhereConditionType.And,
            And: { Children: [...filters.And?.Children ?? [], newAtomicCondition] }
        };

        setFilters(updatedFilters);
        setCurrentFilter({ ColumnType: "", Key: "", Operator: "", Value: "" });
        onChange?.(updatedFilters);
    }, [filters, currentFilter, onChange]);

    const handleRemove = useCallback((index: number) => {
        setEditingFilter(-1);
        const updatedFilters = {
            Type: WhereConditionType.And,
            And: { Children: filters.And?.Children?.filter((_, i) => i !== index) ??[] }
        };
        setFilters(updatedFilters);
        onChange?.(updatedFilters);
    }, [filters, onChange]);

    const handleEdit = useCallback((index: number) => {
        if (editingFilter === index) {
            setEditingFilter(-1);
            setCurrentFilter({ ColumnType: "string", Key: "", Operator: "", Value: "" });
            return;
        }
        setNewFilter(false);
        const filter = filters.And?.Children?.[index];
        if (filter?.Type === WhereConditionType.Atomic) {
            setCurrentFilter(filter.Atomic ?? { ColumnType: "string", Key: "", Operator: "", Value: "" });
        }
        setEditingFilter(index);
    }, [editingFilter, filters]);

    const handleSaveFilter = useCallback((index: number) => {
        const updatedFilters = { ...filters };
        updatedFilters.And!.Children[index] = { Type: WhereConditionType.Atomic, Atomic: { ...currentFilter } };
        setFilters(updatedFilters);
        setEditingFilter(-1);
        setCurrentFilter({ ColumnType: "string", Key: "", Operator: "", Value: "" });
        onChange?.(updatedFilters);
    }, [filters, currentFilter, onChange]);

    const validOperators = useMemo(() => {
        return operators.map(operator => ({ value: operator, label: operator }));
    }, [operators]);

    // Sheet management functions
    const handleOpenSheet = useCallback(() => {
        // Convert filters to sheet format
        const atomicFilters = filters.And?.Children?.map(child =>
            child.Type === WhereConditionType.Atomic ? child.Atomic! : {
                ColumnType: "string",
                Key: "",
                Operator: "",
                Value: ""
            }
        ) ?? [];
        setSheetFilters(atomicFilters);
        setSheetOpen(true);
    }, [filters]);

    const handleSheetFieldChange = useCallback((index: number, field: keyof AtomicWhereCondition, value: string) => {
        setSheetFilters(prev => {
            const newFilters = [...prev];
            if (field === 'Key') {
                newFilters[index] = {
                    ...newFilters[index],
                    Key: value,
                    ColumnType: columnTypes[columns.findIndex(col => col === value)]
                };
            } else {
                newFilters[index] = {...newFilters[index], [field]: value};
            }
            return newFilters;
        });
    }, [columnTypes, columns]);

    const handleSheetAddFilter = useCallback(() => {
        setSheetFilters(prev => [...prev, {ColumnType: "string", Key: "", Operator: "", Value: ""}]);
    }, []);

    const handleSheetRemoveFilter = useCallback((index: number) => {
        setSheetFilters(prev => prev.filter((_, i) => i !== index));
    }, []);

    const handleSheetSave = useCallback(() => {
        const updatedFilters = {
            Type: WhereConditionType.And,
            And: {
                Children: sheetFilters
                    .filter(filter => filter.Key && filter.Operator && filter.Value)
                    .map(filter => ({
                        Type: WhereConditionType.Atomic,
                        Atomic: filter
                    }))
            }
        };
        setFilters(updatedFilters);
        onChange?.(updatedFilters);
        setSheetOpen(false);
    }, [sheetFilters, onChange]);

    useEffect(() => {
        if (defaultWhere == null) {
            return;
        }
        setFilters(defaultWhere);
    }, [defaultWhere]);

    const hasFilterContent = useCallback(() => {
        return currentFilter.Key !== "" || currentFilter.Operator !== "" || currentFilter.Value !== "";
    }, [currentFilter]);

    const handleKeyDown = useCallback((e: KeyboardEvent) => {
        if (e.key === 'Escape') {
            // Only close popups if no content has been entered
            if (!hasFilterContent()) {
                if (newFilter) {
                    setNewFilter(false);
                }
                if (editingFilter !== -1) {
                    setEditingFilter(-1);
                    setCurrentFilter({ ColumnType: "string", Key: "", Operator: "", Value: "" });
                }
            }
        }
    }, [newFilter, editingFilter, hasFilterContent]);

    const handleClickOutside = useCallback((e: MouseEvent) => {
        // For new filter popup
        if (newFilter && newFilterRef.current && !newFilterRef.current.contains(e.target as Node)) {
            // Only close if no content has been entered
            if (!hasFilterContent()) {
                setNewFilter(false);
            }
        }

        // For edit filter popup
        if (editingFilter !== -1 && editFilterRef.current && !editFilterRef.current.contains(e.target as Node)) {
            // Only close if no content has been modified
            if (!hasFilterContent()) {
                setEditingFilter(-1);
                setCurrentFilter({ ColumnType: "string", Key: "", Operator: "", Value: "" });
            }
        }
    }, [newFilter, editingFilter, hasFilterContent]);

    useEffect(() => {
        // Only add event listeners if a popup is open
        if (newFilter || editingFilter !== -1) {
            document.addEventListener('keydown', handleKeyDown);
            document.addEventListener('mousedown', handleClickOutside);

            // Clean up event listeners
            return () => {
                document.removeEventListener('keydown', handleKeyDown);
                document.removeEventListener('mousedown', handleClickOutside);
            };
        }
    }, [newFilter, editingFilter, handleKeyDown, handleClickOutside]);

    const visibleFilters = filters.And?.Children?.slice(0, MAX_VISIBLE_CONDITIONS) ?? [];
    const hiddenCount = (filters.And?.Children?.length ?? 0) - MAX_VISIBLE_CONDITIONS;

    return (
        <div className="flex flex-col">
            <Label className="mb-2">Where condition</Label>
            <div className="flex flex-row gap-xs max-w-[min(500px,calc(100vw-20px))] flex-wrap">
                {visibleFilters.map((filter, i) => (
                    <div
                        key={`explore-storage-unit-filter-${i}`}
                        className="group/filter-item flex gap-xs items-center text-xs rounded-2xl cursor-pointer h-[36px]"
                        data-testid="where-condition"
                    >
                        <Badge
                            className={twMerge(
                                classNames(
                                    "flex items-center gap-xs pl-4 pr-2 h-full max-w-[350px] truncate cursor-pointer py-0",
                                    { "ring-2 ring-primary-500 dark:ring-primary-400": editingFilter === i }
                                )
                            )}
                            onClick={() => handleEdit(i)}
                            data-testid="where-condition-badge"
                            variant="secondary"
                        >
                            <div className="flex items-center gap-xs h-full">
                                {filter.Atomic?.Key} {filter.Atomic?.Operator} {filter.Atomic?.Value}
                                <Button className="size-8 h-full" onClick={() => handleRemove(i)} data-testid="remove-where-condition-button" variant="ghost" size="icon">
                                    <XCircleIcon />
                                </Button>
                            </div>
                        </Badge>
                        <PopoverCard
                            className="mt-8"
                            open={editingFilter === i}
                            onOpenChange={() => {
                                setEditingFilter(editingFilter === i ? -1 : i);
                                setCurrentFilter({ ColumnType: "string", Key: "", Operator: "", Value: "" });
                                setNewFilter(false);
                            }}
                            currentFilter={currentFilter}
                            fieldsDropdownItems={fieldsDropdownItems}
                            validOperators={validOperators}
                            handleFieldSelect={handleFieldSelect}
                            handleOperatorSelector={handleOperatorSelector}
                            handleInputChange={handleInputChange}
                            handleAddFilter={handleAddFilter}
                            handleSaveFilter={handleSaveFilter}
                            handleCancel={handleCancelEditFilter}
                            isEditing={true}
                            editingIndex={i}
                        />
                    </div>
                ))}
                {hiddenCount > 0 && (
                    <Button onClick={handleOpenSheet} data-testid="more-conditions-button" variant="secondary">
                        +{hiddenCount} more
                    </Button>
                )}
                <Button onClick={handleClick} data-testid="where-button" variant="secondary">
                    <PlusCircleIcon className="w-4 h-4" /> Add
                </Button>
            </div>
            <PopoverCard
                open={newFilter}
                onOpenChange={setNewFilter}
                currentFilter={currentFilter}
                fieldsDropdownItems={fieldsDropdownItems}
                validOperators={validOperators}
                handleFieldSelect={handleFieldSelect}
                handleOperatorSelector={handleOperatorSelector}
                handleInputChange={handleInputChange}
                handleAddFilter={handleAddFilter}
                handleSaveFilter={handleSaveFilter}
                handleCancel={handleCancelNewFilter}
                isEditing={false}
                editingIndex={-1}
            />

            {/* Sheet for managing all conditions */}
            <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
                <SheetContent side="right" className="w-[500px] max-w-full p-8">
                    <SheetTitle><AdjustmentsVerticalIcon className="w-5 h-5" /> Manage Where Conditions</SheetTitle>
                    <div className="flex flex-col gap-lg mt-6 overflow-y-auto max-h-[calc(100vh-200px)]">
                        {sheetFilters.map((filter, index) => (
                            <div key={index} className="flex flex-col gap-lg p-4 border rounded-lg">
                                <div className="flex items-center justify-between">
                                    <Label className="text-sm font-medium">Condition {index + 1}</Label>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        onClick={() => handleSheetRemoveFilter(index)}
                                        data-testid={`remove-sheet-filter-${index}`}
                                    >
                                        <XMarkIcon className="w-4 h-4"/>
                                    </Button>
                                </div>
                                <div className="flex flex-col gap-2">
                                    <Label className="text-xs">Field</Label>
                                    <SearchSelect
                                        value={filter.Key}
                                        options={fieldsDropdownItems}
                                        onChange={(value) => handleSheetFieldChange(index, 'Key', value)}
                                        buttonProps={{
                                            "data-testid": `sheet-field-key-${index}`,
                                        }}
                                    />
                                </div>
                                <div className="flex flex-col gap-2">
                                    <Label className="text-xs">Operator</Label>
                                    <SearchSelect
                                        value={filter.Operator}
                                        options={validOperators}
                                        onChange={(value) => handleSheetFieldChange(index, 'Operator', value)}
                                        buttonProps={{
                                            "data-testid": `sheet-field-operator-${index}`,
                                        }}
                                    />
                                </div>
                                <div className="flex flex-col gap-2">
                                    <Label className="text-xs">Value</Label>
                                    <Input
                                        value={filter.Value}
                                        onChange={(e) => handleSheetFieldChange(index, 'Value', e.target.value)}
                                        placeholder="Enter filter value"
                                        data-testid={`sheet-field-value-${index}`}
                                    />
                                </div>
                            </div>
                        ))}
                        <Button onClick={handleSheetAddFilter} data-testid="add-sheet-filter-button" variant="secondary"
                                className="self-start">
                            <PlusCircleIcon className="w-4 h-4"/> Add Condition
                        </Button>
                    </div>
                    <SheetFooter className="flex gap-sm px-0 mt-6">
                        <Button
                            className="flex-1"
                            variant="secondary"
                            onClick={() => setSheetOpen(false)}
                            data-testid="cancel-manage-conditions"
                        >
                            Cancel
                        </Button>
                        <Button className="flex-1" onClick={handleSheetSave}>
                            Save Changes
                        </Button>
                    </SheetFooter>
                </SheetContent>
            </Sheet>
        </div>
    );
};