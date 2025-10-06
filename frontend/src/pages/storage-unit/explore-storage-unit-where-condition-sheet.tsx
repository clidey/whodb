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
    Button,
    cn,
    Input,
    Label,
    SearchSelect,
    Sheet,
    SheetContent,
    SheetFooter,
    SheetTitle
} from "@clidey/ux";
import { AtomicWhereCondition, WhereCondition, WhereConditionType } from '@graphql';
import React, { FC, useCallback, useEffect, useMemo, useState } from "react";
import { AdjustmentsHorizontalIcon, PlusCircleIcon, XCircleIcon } from "../../components/heroicons";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ScratchpadActions } from "../../store/scratchpad";

type IExploreStorageUnitWhereConditionSheetProps = {
    defaultWhere?: WhereCondition;
    columns: string[];
    operators: string[];
    columnTypes: string[];
    onChange?: (filters: WhereCondition) => void;
}

export const ExploreStorageUnitWhereConditionSheet: FC<IExploreStorageUnitWhereConditionSheetProps> = ({ 
    defaultWhere, 
    columns, 
    columnTypes, 
    onChange, 
    operators 
}) => {
    const dispatch = useAppDispatch();
    const { pages, activePageId } = useAppSelector(state => state.scratchpad);
    const [filters, setFilters] = useState<WhereCondition>(defaultWhere ?? {
        Type: WhereConditionType.And,
        And: { Children: [] }
    });
    const [sheetOpen, setSheetOpen] = useState(false);
    const [sheetFilters, setSheetFilters] = useState<AtomicWhereCondition[]>([{ColumnType: "string", Key: "", Operator: "", Value: ""}]);
    const [editingIndex, setEditingIndex] = useState<number>(0);
    const [editingExistingIndex, setEditingExistingIndex] = useState<number>(-1);
    const [editingExistingFilter, setEditingExistingFilter] = useState<AtomicWhereCondition | null>(null);
    const [selectedPageId, setSelectedPageId] = useState<string>("");
    const [newPageName, setNewPageName] = useState<string>("");

    const fieldsDropdownItems = useMemo(() => columns.map(column => ({ value: column, label: column })), [columns]);
    const validOperators = useMemo(() => {
        return operators.map(operator => ({ value: operator, label: operator }));
    }, [operators]);

    // Create page options excluding current page
    const pageOptions = useMemo(() => {
        const availablePages = pages.filter(page => page.id !== activePageId);
        return [
            ...availablePages.map(page => ({ value: page.id, label: page.name })),
            { value: "new", label: "Create new page" }
        ];
    }, [pages, activePageId]);

    // Open sheet with a clean "add new" section; existing conditions are listed separately
    const handleOpenSheet = useCallback(() => {
        setSheetFilters([{ ColumnType: "string", Key: "", Operator: "", Value: "" }]);
        setEditingIndex(0);
        setSelectedPageId("");
        setNewPageName("");
        setEditingExistingIndex(-1);
        setEditingExistingFilter(null);
        setSheetOpen(true);
    }, []);

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
        setEditingIndex(sheetFilters.length); // Edit the newly added filter
    }, [sheetFilters.length]);

    const handleDeleteExistingFilter = useCallback((index: number) => {
        const currentFilters = filters.And?.Children ?? [];
        const updatedFilters = currentFilters.filter((_, i) => i !== index);
        
        const newWhereCondition = {
            Type: WhereConditionType.And,
            And: {
                Children: updatedFilters
            }
        };
        
        setFilters(newWhereCondition);
        onChange?.(newWhereCondition);
    }, [filters, onChange]);

    const handleEditExistingFilter = useCallback((index: number) => {
        const currentFilters = filters.And?.Children ?? [];
        const filterToEdit = currentFilters[index];
        
        if (filterToEdit?.Atomic) {
            setEditingExistingIndex(index);
            setEditingExistingFilter(filterToEdit.Atomic);
            setSheetFilters([]); // Clear new filters when editing existing
        }
    }, [filters]);

    const handleSheetSave = useCallback(() => {
        if (editingExistingIndex >= 0 && editingExistingFilter) {
            // Update existing filter
            const currentFilters = filters.And?.Children ?? [];
            const updatedFilters = [...currentFilters];
            updatedFilters[editingExistingIndex] = {
                Type: WhereConditionType.Atomic,
                Atomic: editingExistingFilter
            };
            
            const newWhereCondition = {
                Type: WhereConditionType.And,
                And: {
                    Children: updatedFilters
                }
            };
            
            setFilters(newWhereCondition);
            onChange?.(newWhereCondition);
            
            // Reset editing state
            setEditingExistingIndex(-1);
            setEditingExistingFilter(null);
            setSheetFilters([{ColumnType: "string", Key: "", Operator: "", Value: ""}]);
            setEditingIndex(0);
        } else {
            // Add new filter
            const validFilters = sheetFilters.filter(filter => filter.Key && filter.Operator && filter.Value);
            
            if (validFilters.length > 0) {
                const newWhereCondition = {
                    Type: WhereConditionType.And,
                    And: {
                        Children: validFilters.map(filter => ({
                            Type: WhereConditionType.Atomic,
                            Atomic: filter
                        }))
                    }
                };

                // If a page is selected, add condition to that page
                if (selectedPageId) {
                    if (selectedPageId === "new") {
                        // Create new page with condition
                        const pageName = newPageName.trim() || `Page ${pages.length + 1}`;
                        dispatch(ScratchpadActions.addPage({ name: pageName }));
                        // The condition will be added after the page is created
                        setTimeout(() => {
                            const newPage = pages.find(p => p.name === pageName);
                            if (newPage) {
                                dispatch(ScratchpadActions.addConditionToPage({ 
                                    pageId: newPage.id, 
                                    condition: newWhereCondition 
                                }));
                            }
                        }, 100);
                    } else {
                        // Add condition to existing page
                        dispatch(ScratchpadActions.addConditionToPage({ 
                            pageId: selectedPageId, 
                            condition: newWhereCondition 
                        }));
                    }
                } else {
                    // No page selected, add to current filters
                    const currentFilters = filters.And?.Children ?? [];
                    const updatedFilters = {
                        Type: WhereConditionType.And,
                        And: {
                            Children: [
                                ...currentFilters,
                                ...validFilters.map(filter => ({
                                    Type: WhereConditionType.Atomic,
                                    Atomic: filter
                                }))
                            ]
                        }
                    };
                    setFilters(updatedFilters);
                    onChange?.(updatedFilters);
                }
            }
            
            // Reset form
            setSheetFilters([{ColumnType: "string", Key: "", Operator: "", Value: ""}]);
            setEditingIndex(0);
            setSelectedPageId("");
            setNewPageName("");
        }
    }, [sheetFilters, onChange, editingExistingIndex, editingExistingFilter, filters, selectedPageId, newPageName, pages, dispatch]);

    const handleEditExistingFilterChange = useCallback((field: keyof AtomicWhereCondition, value: string) => {
        if (editingExistingFilter) {
            const updatedFilter = { ...editingExistingFilter };
            if (field === 'Key') {
                updatedFilter.Key = value;
                updatedFilter.ColumnType = columnTypes[columns.findIndex(col => col === value)];
            } else {
                updatedFilter[field] = value;
            }
            setEditingExistingFilter(updatedFilter);
        }
    }, [editingExistingFilter, columnTypes, columns]);

    const handleCloseSheet = useCallback(() => {
        setSheetOpen(false);
        setEditingIndex(-1);
        setEditingExistingIndex(-1);
        setEditingExistingFilter(null);
        setSheetFilters([{ColumnType: "string", Key: "", Operator: "", Value: ""}]);
        setSelectedPageId("");
        setNewPageName("");
    }, []);

    useEffect(() => {
        if (defaultWhere == null) {
            return;
        }
        setFilters(defaultWhere);
    }, [defaultWhere]);

    // Initialize scratchpad if needed
    useEffect(() => {
        if (pages.length === 0) {
            dispatch(ScratchpadActions.ensurePagesHaveCells());
        }
    }, [dispatch, pages.length]);

    // Existing conditions list
    const existingFilters = filters.And?.Children ?? [];
    
    // Calculate total condition count for button text
    const totalConditions = filters.And?.Children?.length ?? 0;
    const getConditionButtonText = () => {
        if (totalConditions === 0) {
            return "Add";
        } else if (totalConditions > 10) {
            return "10+ Conditions";
        } else {
            return `${totalConditions} Condition${totalConditions === 1 ? '' : 's'}`;
        }
    };

    return (
        <div className="flex flex-col">
            <Label className="mb-2">Where condition</Label>
            <div className="flex flex-row gap-xs max-w-[min(500px,calc(100vw-20px))] flex-wrap">
                <Button onClick={handleOpenSheet} data-testid="where-button" variant="secondary">
                    <PlusCircleIcon className="w-4 h-4" /> {getConditionButtonText()}
                </Button>
            </div>
            {/* Sheet for managing all conditions */}
            <Sheet open={sheetOpen} onOpenChange={handleCloseSheet}>
                <SheetContent side="right" className="w-[500px] max-w-full p-8 h-full">
                    <SheetTitle className="flex items-center gap-2"><AdjustmentsHorizontalIcon className="w-5 h-5" /> Conditions</SheetTitle>
                    {/* Display existing conditions as editable cards */}
                    {existingFilters.length > 0 && (
                        <div className="flex flex-col gap-sm mt-4">
                            {existingFilters.map((filter, i) => {
                                const isEditing = editingExistingIndex === i;
                                return (
                                    <div
                                        key={`existing-condition-card-${i}`}
                                        className="flex flex-col gap-2 p-4 border rounded-lg"
                                        data-testid={`existing-condition-card-${i}`}
                                    >
                                        {isEditing ? (
                                            // Editing mode - show form fields
                                            <div className="flex flex-col gap-2">
                                                <div className="flex items-center justify-between">
                                                    <Label className="text-sm font-medium">Editing Condition</Label>
                                                </div>
                                                <div className="flex flex-col gap-2">
                                                    <Label className="text-xs">Field</Label>
                                                    <SearchSelect
                                                        value={editingExistingFilter?.Key || ""}
                                                        options={fieldsDropdownItems}
                                                        onChange={(value) => handleEditExistingFilterChange('Key', value)}
                                                        buttonProps={{
                                                            "data-testid": `edit-existing-field-key-${i}`,
                                                        }}
                                                    />
                                                </div>
                                                <div className="flex flex-col gap-2">
                                                    <Label className="text-xs">Operator</Label>
                                                    <SearchSelect
                                                        value={editingExistingFilter?.Operator || ""}
                                                        options={validOperators}
                                                        onChange={(value) => handleEditExistingFilterChange('Operator', value)}
                                                        buttonProps={{
                                                            "data-testid": `edit-existing-field-operator-${i}`,
                                                        }}
                                                    />
                                                </div>
                                                <div className="flex flex-col gap-2">
                                                    <Label className="text-xs">Value</Label>
                                                    <Input
                                                        value={editingExistingFilter?.Value || ""}
                                                        onChange={(e) => handleEditExistingFilterChange('Value', e.target.value)}
                                                        placeholder="Enter filter value"
                                                        data-testid={`edit-existing-field-value-${i}`}
                                                    />
                                                </div>
                                                <div className="flex gap-2 mt-2 w-full">
                                                    <Button
                                                        variant="secondary"
                                                        onClick={() => {
                                                            setEditingExistingIndex(-1);
                                                            setEditingExistingFilter(null);
                                                        }}
                                                        data-testid={`cancel-edit-existing-filter-${i}`}
                                                        size="sm"
                                                        className="flex-1"
                                                    >
                                                        Cancel
                                                    </Button>
                                                    <Button
                                                        className="flex-1"
                                                        onClick={() => {
                                                            if (editingExistingFilter && editingExistingIndex >= 0) {
                                                                // Update existing filter
                                                                const currentFilters = filters.And?.Children ?? [];
                                                                const updatedFilters = [...currentFilters];
                                                                updatedFilters[editingExistingIndex] = {
                                                                    Type: WhereConditionType.Atomic,
                                                                    Atomic: editingExistingFilter
                                                                };
                                                                
                                                                const newWhereCondition = {
                                                                    Type: WhereConditionType.And,
                                                                    And: {
                                                                        Children: updatedFilters
                                                                    }
                                                                };
                                                                
                                                                setFilters(newWhereCondition);
                                                                onChange?.(newWhereCondition);
                                                                
                                                                // Reset editing state
                                                                setEditingExistingIndex(-1);
                                                                setEditingExistingFilter(null);
                                                            }
                                                        }}
                                                        data-testid={`update-existing-filter-${i}`}
                                                        size="sm"
                                                    >
                                                        Update
                                                    </Button>
                                                </div>
                                            </div>
                                        ) : (
                                            // View mode - show condition and action buttons
                                            <div className="flex items-center justify-between">
                                                <div className="text-sm">
                                                    {filter.Atomic?.Key} {filter.Atomic?.Operator} {filter.Atomic?.Value}
                                                </div>
                                                <div className="flex items-center gap-1">
                                                    <Button 
                                                        className="size-6 h-full ml-1" 
                                                        onClick={(e: React.MouseEvent) => {
                                                            e.stopPropagation();
                                                            handleEditExistingFilter(i);
                                                        }} 
                                                        data-testid={`edit-existing-filter-${i}`} 
                                                        variant="ghost" 
                                                        size="icon"
                                                    >
                                                        <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                                        </svg>
                                                    </Button>
                                                    <Button 
                                                        className="size-6 h-full" 
                                                        onClick={(e: React.MouseEvent) => {
                                                            e.stopPropagation();
                                                            handleDeleteExistingFilter(i);
                                                        }} 
                                                        data-testid={`delete-existing-filter-${i}`} 
                                                        variant="ghost" 
                                                        size="icon"
                                                    >
                                                        <XCircleIcon className="w-3 h-3" />
                                                    </Button>
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    )}
                    
                    {/* Page Selection */}
                    {/* <div className="flex flex-col gap-2 mt-6">
                        <Label className="text-sm font-medium">Add condition to page</Label>
                        <Select value={selectedPageId} onValueChange={setSelectedPageId}>
                            <SelectTrigger className="w-full" data-testid="page-select">
                                <SelectValue placeholder="Choose a page to add condition to..." />
                            </SelectTrigger>
                            <SelectContent>
                                {pageOptions.map((option) => (
                                    <SelectItem key={option.value} value={option.value}>
                                        {option.label}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        {selectedPageId === "new" && (
                            <div className="flex flex-col gap-2">
                                <Label className="text-xs">New page name</Label>
                                <Input
                                    value={newPageName}
                                    onChange={(e) => setNewPageName(e.target.value)}
                                    placeholder="Enter page name"
                                    data-testid="new-page-name-input"
                                />
                            </div>
                        )}
                    </div> */}
                    
                    {/* Add new conditions section */}
                    <div className="flex flex-col gap-lg mt-6 overflow-y-auto h-full">
                        {sheetFilters.map((filter, index) => (
                            <div 
                                key={index} 
                                className="flex flex-col gap-lg p-4 border rounded-lg"
                            >
                                <div className="flex items-center justify-between">
                                    <Label className="text-sm font-medium">
                                        Condition {index + 1}
                                    </Label>
                                    {sheetFilters.length > 1 && (
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => {
                                                setSheetFilters(prev => prev.filter((_, i) => i !== index));
                                                if (editingIndex >= index) {
                                                    setEditingIndex(Math.max(0, editingIndex - 1));
                                                }
                                            }}
                                            data-testid={`remove-condition-${index}`}
                                        >
                                            <XCircleIcon className="w-4 h-4" />
                                        </Button>
                                    )}
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
                        <div className="flex items-center justify-end">
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={handleSheetAddFilter}
                                data-testid="add-new-condition-button"
                            >
                                <PlusCircleIcon className="w-4 h-4" />
                                Add Another
                            </Button>
                        </div>
                    </div>
                    <SheetFooter className="flex flex-row w-full gap-sm px-0 mt-6">
                        <Button
                            className="flex-1"
                            variant="secondary"
                            onClick={handleCloseSheet}
                            data-testid="cancel-add-conditions"
                        >
                            Cancel
                        </Button>
                        <Button 
                            className="flex-1"
                            onClick={handleSheetSave}
                            data-testid="add-conditions-button"
                        >
                            {selectedPageId ? 'Add to Page' : 'Add Condition'}
                        </Button>
                    </SheetFooter>
                </SheetContent>
            </Sheet>
        </div>
    );
};
