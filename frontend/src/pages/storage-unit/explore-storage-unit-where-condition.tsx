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
    Badge,
    Button,
    cn,
    Input,
    Label,
    Popover,
    PopoverContent,
    PopoverTrigger,
    Sheet,
    SheetContent,
    SheetFooter,
    SheetTitle
} from "@clidey/ux";
import { SearchSelect } from "../../components/ux";
import type { AtomicWhereCondition, WhereCondition} from '@graphql';
import { WhereConditionType } from '@graphql';
import classNames from "classnames";
import type { FC} from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { twMerge } from "tailwind-merge";
import { PlusCircleIcon, XCircleIcon, XMarkIcon } from "../../components/heroicons";
import {WhoDBChatIcon} from "../../components/whodb-chat-icon";
import { useTranslation } from "@/hooks/use-translation";
import {whereConditionToSql} from "../../utils/where-condition-to-sql";

type FilterJoin = typeof WhereConditionType.And | typeof WhereConditionType.Or;
type FilterRow = {condition: AtomicWhereCondition; join: FilterJoin};

const flattenFilters = (where: WhereCondition | undefined): FilterRow[] => {
    if (!where) return [];
    if (where.Type === WhereConditionType.Atomic) {
        return where.Atomic ? [{condition: where.Atomic, join: WhereConditionType.And}] : [];
    }

    const join = where.Type === WhereConditionType.Or ? WhereConditionType.Or : WhereConditionType.And;
    const children = join === WhereConditionType.Or ? where.Or?.Children : where.And?.Children;
    return (children ?? []).flatMap((child, index) => {
        const rows = flattenFilters(child);
        if (index > 0 && rows.length > 0) rows[0] = {...rows[0], join};
        return rows;
    });
};

const buildFilters = (rows: FilterRow[]): WhereCondition => {
    let where: WhereCondition = {Type: WhereConditionType.And, And: {Children: []}};
    for (const [index, {condition, join}] of rows.entries()) {
        const atomic: WhereCondition = {Type: WhereConditionType.Atomic, Atomic: condition};
        if (index > 0 && join === WhereConditionType.Or) {
            where = where.Type === WhereConditionType.Or
                ? {Type: WhereConditionType.Or, Or: {Children: [...(where.Or?.Children ?? []), atomic]}}
                : {Type: WhereConditionType.Or, Or: {Children: [where, atomic]}};
        } else {
            where = where.Type === WhereConditionType.And
                ? {Type: WhereConditionType.And, And: {Children: [...(where.And?.Children ?? []), atomic]}}
                : {Type: WhereConditionType.And, And: {Children: [where, atomic]}};
        }
    }
    return where;
};

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
    trigger?: React.ReactNode;
    t: (key: string, params?: Record<string, any>) => string;
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
                                                editingIndex = -1,
                                                trigger,
                                                t
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
            {trigger ?? <div />}
        </PopoverTrigger>
        <PopoverContent
            className={cn("ce-filter-popover z-[50]", className)}
            side="bottom"
            align="start"
            tabIndex={0}
        >
            <strong className="ce-filter-popover-title">{t(isEditing ? 'editCondition' : 'addFilter')}</strong>
            <div className="ce-filter-fields">
            <div className="ce-filter-field">
                <Label className="text-xs">
                    {t('field')}
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
            <div className="ce-filter-field">
                <Label className="text-xs">
                    {t('operator')}
                </Label>
                <SearchSelect
                    value={currentFilter.Operator}
                    options={validOperators}
                    onChange={handleOperatorSelector}
                    buttonProps={{
                        "data-testid": "field-operator",
                    }}
                />
            </div>
            <div className="ce-filter-field">
                <Label className="text-xs">
                    {t('value')}
                </Label>
                <Input
                    className="min-w-[150px] w-full"
                    placeholder={t('enterFilterValue')}
                    value={currentFilter.Value}
                    onChange={e => { handleInputChange(e.target.value); }}
                    data-testid="field-value"
                />
            </div>
            </div>
            {currentFilter.Key && <p className="ce-filter-type-hint">{t('columnTypeHint', {column: currentFilter.Key, type: currentFilter.ColumnType?.toLowerCase() ?? ''})}</p>}
            <div className="ce-filter-actions">
                <Button
                    className="flex-1"
                    onClick={handleCancel}
                    data-testid="cancel-button"
                    variant="secondary"
                >
                    {t('cancel')}
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
                    {isEditing ? t('update') : t('applyFilter')}
                </Button>
            </div>
        </PopoverContent>
    </Popover>
}

type IExploreStorageUnitWhereConditionProps = {
    defaultWhere?: WhereCondition;
    objectName?: string;
    sheetOnly?: boolean;
    columns: string[];
    operators: string[];
    columnTypes: string[];
    onChange?: (filters: WhereCondition) => void;
}

/** Renders the shared Explore filter chips, quick add popover, and all filters sheet. */
export const ExploreStorageUnitWhereCondition: FC<IExploreStorageUnitWhereConditionProps> = ({ defaultWhere, objectName, sheetOnly = false, columns, columnTypes, onChange, operators }) => {
    const { t } = useTranslation('pages/where-condition');
    const [currentFilter, setCurrentFilter] = useState<AtomicWhereCondition>({ ColumnType: "string", Key: "", Operator: "", Value: "" });
    const [filters, setFilters] = useState<WhereCondition>(defaultWhere ?? {
        Type: WhereConditionType.And,
        And: { Children: [] }
    });
    const [newFilter, setNewFilter] = useState(false);
    const [editingFilter, setEditingFilter] = useState(-1);
    const [sheetOpen, setSheetOpen] = useState(false);
    const [sheetFilters, setSheetFilters] = useState<AtomicWhereCondition[]>([]);
    const [sheetJoins, setSheetJoins] = useState<FilterJoin[]>([]);
    const [sheetRowIds, setSheetRowIds] = useState<string[]>([]);
    const newFilterRef = useRef<HTMLDivElement>(null);
    const editFilterRef = useRef<HTMLDivElement>(null);

    // Maximum number of conditions to show in the main view
    const MAX_VISIBLE_CONDITIONS = 3;
    const filterRows = useMemo(() => flattenFilters(filters), [filters]);

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
        const updatedFilters = buildFilters([...filterRows, {condition: currentFilter, join: WhereConditionType.And}]);

        setFilters(updatedFilters);
        setNewFilter(false);
        setCurrentFilter({ ColumnType: "", Key: "", Operator: "", Value: "" });
        onChange?.(updatedFilters);
    }, [filterRows, currentFilter, onChange]);

    const handleRemove = useCallback((index: number) => {
        setEditingFilter(-1);
        const updatedFilters = buildFilters(filterRows.filter((_, i) => i !== index));
        setFilters(updatedFilters);
        onChange?.(updatedFilters);
    }, [filterRows, onChange]);

    const handleEdit = useCallback((index: number) => {
        if (editingFilter === index) {
            setEditingFilter(-1);
            setCurrentFilter({ ColumnType: "string", Key: "", Operator: "", Value: "" });
            return;
        }
        setNewFilter(false);
        setCurrentFilter(filterRows[index]?.condition ?? { ColumnType: "string", Key: "", Operator: "", Value: "" });
        setEditingFilter(index);
    }, [editingFilter, filterRows]);

    const handleSaveFilter = useCallback((index: number) => {
        const updatedRows = [...filterRows];
        updatedRows[index] = {...updatedRows[index], condition: {...currentFilter}};
        const updatedFilters = buildFilters(updatedRows);
        setFilters(updatedFilters);
        setEditingFilter(-1);
        setCurrentFilter({ ColumnType: "string", Key: "", Operator: "", Value: "" });
        onChange?.(updatedFilters);
    }, [filterRows, currentFilter, onChange]);

    const validOperators = useMemo(() => {
        return operators.map(operator => ({ value: operator, label: operator }));
    }, [operators]);

    // Sheet management functions
    const handleOpenSheet = useCallback(() => {
        const rows: FilterRow[] = filterRows.length > 0 ? filterRows : [{condition: {ColumnType: "string", Key: "", Operator: "", Value: ""}, join: WhereConditionType.And}];
        setSheetFilters(rows.map(row => row.condition));
        setSheetJoins(rows.map(row => row.join));
        setSheetRowIds(rows.map(() => crypto.randomUUID()));
        setSheetOpen(true);
    }, [filterRows]);

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
        setSheetJoins(prev => [...prev, WhereConditionType.And]);
        setSheetRowIds(prev => [...prev, crypto.randomUUID()]);
    }, []);

    const handleSheetRemoveFilter = useCallback((index: number) => {
        setSheetFilters(prev => prev.filter((_, i) => i !== index));
        setSheetJoins(prev => prev.filter((_, i) => i !== index));
        setSheetRowIds(prev => prev.filter((_, i) => i !== index));
    }, []);

    const handleSheetSave = useCallback(() => {
        const updatedFilters = buildFilters(sheetFilters.map((condition, index) => ({
            condition,
            join: sheetJoins[index] ?? WhereConditionType.And,
        })));
        setFilters(updatedFilters);
        onChange?.(updatedFilters);
        setSheetOpen(false);
    }, [sheetFilters, sheetJoins, onChange]);

    useEffect(() => {
        setFilters(defaultWhere ?? { Type: WhereConditionType.And, And: { Children: [] } });
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

    const visibleFilters = filterRows.slice(0, MAX_VISIBLE_CONDITIONS);
    const hiddenCount = filterRows.length - MAX_VISIBLE_CONDITIONS;
    const totalConditionCount = filterRows.length;
    const sheetIsValid = sheetFilters.every(filter => filter.Key && filter.Operator && filter.Value);
    const sheetWhereClause = whereConditionToSql(buildFilters(sheetFilters
        .map((condition, index) => ({condition, join: sheetJoins[index] ?? WhereConditionType.And}))
        .filter(row => row.condition.Key && row.condition.Operator && row.condition.Value)));

    return (
        <div className="flex flex-col" data-condition-count={totalConditionCount} data-condition-mode={filterRows.some(row => row.join === WhereConditionType.Or) ? "mixed" : "and"}>
            <Label className="mb-2">{t('whereCondition')}</Label>
            <div className="flex flex-row gap-xs max-w-[min(500px,calc(100vw-20px))] flex-wrap">
                {visibleFilters.map(({condition: filter}, i) => {
                    const filterKey = `${filter.Key}-${filter.Operator}-${filter.Value}-${i}`;
                    return (<div
                        key={filterKey}
                        className="group/filter-item flex gap-xs items-center text-xs rounded-2xl cursor-pointer h-[36px]"
                        data-testid="where-condition"
                        data-condition-key={filter.Key}
                        data-condition-operator={filter.Operator}
                        data-condition-value={filter.Value}
                    >
                        <Badge
                            className={twMerge(
                                classNames(
                                    "flex items-center gap-xs pl-4 pr-2 h-full max-w-[350px] truncate cursor-pointer py-0",
                                    { "ring-2 ring-primary-500 dark:ring-primary-400": editingFilter === i }
                                )
                            )}
                            onClick={() => { if (sheetOnly) handleOpenSheet(); else handleEdit(i); }}
                            data-testid="where-condition-badge"
                            variant="secondary"
                        >
                            <div className="flex items-center gap-xs h-full">
                                {filter.Key} {filter.Operator} {filter.Value}
                                <Button className="size-8 h-full" onClick={event => { event.stopPropagation(); handleRemove(i); }} data-testid="remove-where-condition-button" variant="ghost" size="icon" aria-label={t('removeCondition')}>
                                    <XCircleIcon aria-hidden="true" />
                                </Button>
                            </div>
                        </Badge>
                        {!sheetOnly && <PopoverCard
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
                            t={t}
                        />}
                    </div>);
                })}
                {hiddenCount > 0 && (
                    <Button onClick={handleOpenSheet} data-testid="more-conditions-button" variant="secondary">
                        {t('moreConditions', { count: hiddenCount })}
                    </Button>
                )}
                {sheetOnly ? <Button data-testid="where-button" data-sheet-only="true" variant="secondary" onClick={handleOpenSheet}><WhoDBChatIcon name="plus-circle" /> {t('filter')}</Button> : <PopoverCard
                    open={newFilter}
                    onOpenChange={(open) => { if (open) handleClick(); else setNewFilter(false); }}
                    trigger={<Button data-testid="where-button" variant="secondary"><WhoDBChatIcon name="plus-circle" /> {t('filter')}</Button>}
                    currentFilter={currentFilter}
                    fieldsDropdownItems={fieldsDropdownItems}
                    validOperators={validOperators}
                    handleFieldSelect={handleFieldSelect}
                    handleOperatorSelector={handleOperatorSelector}
                    handleInputChange={handleInputChange}
                    handleAddFilter={handleAddFilter}
                    handleSaveFilter={handleSaveFilter}
                    handleCancel={handleCancelNewFilter}
                    t={t}
                />}
            </div>

            {/* Sheet for managing all conditions */}
            <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
                <SheetContent side="right" className="ce-all-filters-sheet w-[min(380px,100vw)] max-w-full p-0" footer={
                    <SheetFooter className="ce-all-filters-footer">
                        <Button variant="ghost" size="sm" onClick={() => { setSheetFilters([]); setSheetJoins([]); setSheetRowIds([]); }}>
                            {t('clearAll')}
                        </Button>
                        <div>
                            <Button variant="outline" size="sm" onClick={() => { setSheetOpen(false); }} data-testid="cancel-manage-conditions">
                                {t('cancel')}
                            </Button>
                            <Button size="sm" onClick={handleSheetSave} disabled={!sheetIsValid} data-testid="apply-filters-button">
                                {t('applyFilters', {count: sheetFilters.length})}
                            </Button>
                        </div>
                    </SheetFooter>
                }>
                    <div className="ce-all-filters-header">
                        {objectName && <span>{objectName}</span>}
                        <SheetTitle>{t('filters')}</SheetTitle>
                    </div>
                    <div className="ce-all-filters-body">
                        {sheetFilters.map((filter, index) => {
                            return (<div key={sheetRowIds[index]} className="ce-all-filters-condition">
                                {index > 0 && <div className="ce-all-filters-join" role="group" aria-label={t('joinConditions')}>
                                    <Button variant="ghost" size="sm" aria-pressed={sheetJoins[index] !== WhereConditionType.Or}
                                        onClick={() => { setSheetJoins(prev => prev.map((join, i) => i === index ? WhereConditionType.And : join)); }}>
                                        {t('and')}
                                    </Button>
                                    <Button variant="ghost" size="sm" aria-pressed={sheetJoins[index] === WhereConditionType.Or}
                                        onClick={() => { setSheetJoins(prev => prev.map((join, i) => i === index ? WhereConditionType.Or : join)); }}>
                                        {t('or')}
                                    </Button>
                                </div>}
                                <div className="ce-all-filters-row" data-testid={`sheet-filter-row-${index}`}>
                                    <SearchSelect
                                        value={filter.Key}
                                        options={fieldsDropdownItems}
                                        onChange={(value) =>{  handleSheetFieldChange(index, 'Key', value); }}
                                        buttonProps={{
                                            "data-testid": `sheet-field-key-${index}`,
                                            "aria-label": t('field'),
                                        }}
                                    />
                                    <SearchSelect
                                        value={filter.Operator}
                                        options={validOperators}
                                        onChange={(value) =>{  handleSheetFieldChange(index, 'Operator', value); }}
                                        buttonProps={{
                                            "data-testid": `sheet-field-operator-${index}`,
                                            "aria-label": t('operator'),
                                        }}
                                    />
                                    <Input
                                        value={filter.Value}
                                        onChange={(e) =>{  handleSheetFieldChange(index, 'Value', e.target.value); }}
                                        placeholder={t('enterFilterValue')}
                                        data-testid={`sheet-field-value-${index}`}
                                        aria-label={t('value')}
                                    />
                                    <Button variant="ghost" size="icon" onClick={() => { handleSheetRemoveFilter(index); }}
                                        data-testid={`remove-sheet-filter-${index}`} aria-label={t('removeCondition')}>
                                        <XMarkIcon className="size-4" aria-hidden="true" />
                                    </Button>
                                </div>
                            </div>);
                        })}
                        <Button onClick={handleSheetAddFilter} data-testid="add-sheet-filter-button" variant="outline" size="sm" className="ce-all-filters-add">
                            <PlusCircleIcon className="size-4" aria-hidden="true" /> {t('addCondition')}
                        </Button>
                        {sheetWhereClause && <div className="ce-all-filters-preview">
                            <span>{t('whereClause')}</span>
                            <code>{t('whereKeyword')} {sheetWhereClause}</code>
                        </div>}
                    </div>
                </SheetContent>
            </Sheet>
        </div>
    );
};
