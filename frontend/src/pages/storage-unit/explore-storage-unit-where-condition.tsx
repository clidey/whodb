// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { ActionButton, AnimatedButton } from "../../components/button";
import { createDropdownItem, Dropdown, IDropdownItem } from "../../components/dropdown";
import { Icons } from "../../components/icons";
import { Input, Label } from "../../components/input";
import { twMerge } from "tailwind-merge";

export type IExploreStorageUnitWhereConditionFilter = {
    field: string;
    operator: string;
    value: string;
}

type IExploreStorageUnitWhereConditionProps = {
    defaultFilters?: IExploreStorageUnitWhereConditionFilter[];
    options: string[];
    operators: string[];
    onChange?: (filters: IExploreStorageUnitWhereConditionFilter[]) => void;
}

export const ExploreStorageUnitWhereCondition: FC<IExploreStorageUnitWhereConditionProps> = ({ defaultFilters, options, onChange, operators }) => {
    const [currentFilter, setCurrentFilter] = useState<IExploreStorageUnitWhereConditionFilter>({ field: options[0], operator: operators[0], value: "" });
    const [filters, setFilters] = useState<IExploreStorageUnitWhereConditionFilter[]>([]);
    const [newFilter, setNewFilter] = useState(false);
    const [editingFilter, setEditingFilter] = useState(-1);

    const handleClick = useCallback(() => {
        const shouldShow = !newFilter;
        if (shouldShow) {
            setEditingFilter(-1);
            setCurrentFilter({ field: currentFilter.field, operator: operators[0], value: "" });
        }
        setNewFilter(!newFilter);
    }, [currentFilter.field, operators, newFilter]);

    const fieldsDropdownItems = useMemo(() => {
        return options.map(option => createDropdownItem(option));
    }, [options]);

    const handleFieldSelect = useCallback((item: IDropdownItem) => {
        setCurrentFilter(val => ({
            ...val,
            field: item.id,
        }));
    }, []);

    const handleOperatorSelector = useCallback((item: IDropdownItem) => {
        setCurrentFilter(val => ({
            ...val,
            operator: item.id,
        }));
    }, []);

    const handleInputChange = useCallback((newValue: string) => {
        setCurrentFilter(val => ({
            ...val,
            value: newValue,
        }));
    }, []);

    const handleAddFilter = useCallback(() => {
        const newFilters = [...filters, currentFilter];
        setFilters(newFilters);
        setCurrentFilter({ field: currentFilter.field, operator: operators[0], value: "" });
        onChange?.(newFilters);
    }, [filters, currentFilter, onChange, operators]);

    const handleRemove = useCallback((index: number) => {
        setEditingFilter(-1);
        const newFilters = filters.filter((_, i) => i !== index)
        setFilters(newFilters);
        onChange?.(newFilters);
    }, [filters, onChange]);

    const handleEdit = useCallback((index: number) => {
        if (editingFilter === index) {
            setEditingFilter(-1);
            setCurrentFilter({ field: currentFilter.field, operator: operators[0], value: "" });
            return;
        }
        setNewFilter(false);
        setCurrentFilter(filters[index]);
        setEditingFilter(index);
    }, [editingFilter, filters, currentFilter.field, operators]);

    const handleSaveFilter = useCallback((index: number) => {
        const newFilters = [...filters];
        newFilters[index] = {...currentFilter};
        setFilters(newFilters);
        setEditingFilter(-1);
        setCurrentFilter({ field: currentFilter.field, operator: operators[0], value: "" });
        onChange?.(newFilters);
    }, [currentFilter, filters, onChange, operators]);

    useEffect(() => {
        setCurrentFilter(f => ({
            ...f,
            operator: operators[0],
        }));
    }, [operators]);

    const validOperators = useMemo(() => {
        return operators.map(operator => createDropdownItem(operator));
    }, [operators]);

    useEffect(() => {
        setFilters(defaultFilters ?? []);
    }, [defaultFilters]);

    return <div className="flex flex-col gap-1 h-full relative">
        <Label label="Where condition" />
        <div className="flex gap-1 items-center max-w-[min(500px,calc(100vw-20px))] flex-wrap">
            {
                filters.map((filter, i) => (
                    <div key={`explore-storage-unit-filter-${i}`} className="group/filter-item flex gap-1 items-center text-xs rounded-2xl dark:bg-white/5 cursor-pointer relative shadow-sm border border-neutral-100 dark:border-neutral-800">
                        <div className={twMerge(classNames("px-2 py-1 h-full max-w-[350px] truncate dark:text-neutral-300 rounded-2xl", {
                            "dark:bg-white/10": editingFilter === i,
                        }))} onClick={() => handleEdit(i)}>
                            {filter.field} {filter.operator} {filter.value}
                        </div>
                        <ActionButton icon={Icons.Cancel} containerClassName="hover:scale-125 absolute right-2 top-1/2 -translate-y-1/2 z-10 h-4 w-4 opacity-0 group-hover/filter-item:opacity-100"
                            onClick={() => handleRemove(i)} />
                        <AnimatePresence mode="wait">
                            {
                                editingFilter === i &&
                                <motion.div className="flex gap-1 z-[5] py-2 px-4 absolute left-0 top-full mt-2 rounded-lg shadow-md border border-neutral-100  dark:border-white/5 dark:bg-white/20 dark:backdrop-blur-xl translate-y-full bg-white" initial={{
                                    y: -10,
                                    opacity: 0,
                                }} animate={{
                                    y: 0,
                                    opacity: 1,
                                }} exit={{
                                    y: -10,
                                    opacity: 0,
                                }}>
                                    <Dropdown noItemsLabel="No fields found" className="min-w-[100px]" value={createDropdownItem(currentFilter.field)} items={fieldsDropdownItems.length === 0 ? [createDropdownItem(currentFilter.field)] : fieldsDropdownItems} onChange={handleFieldSelect} />
                                    <Dropdown noItemsLabel="No operators found" className="min-w-20" value={createDropdownItem(currentFilter.operator)} items={validOperators} onChange={handleOperatorSelector} />
                                    <Input inputProps={{
                                        className: "min-w-[150px]"
                                    }} placeholder="Enter filter value" value={currentFilter.value} setValue={handleInputChange} />
                                    <AnimatedButton className="dark:bg-white/5" icon={Icons.Cancel} label="Cancel" onClick={() => handleEdit(i)} />
                                    <AnimatedButton className="dark:bg-white/5" icon={Icons.CheckCircle} label="Save" onClick={() => handleSaveFilter(i)} />
                                </motion.div>
                            }
                        </AnimatePresence>
                    </div>
                ))
            }
            <ActionButton className={classNames("transition-all", {
                "rotate-45": newFilter,
            })} icon={Icons.Add} containerClassName="h-8 w-8" onClick={handleClick} />
        </div>
        <AnimatePresence mode="wait">
            {
                newFilter &&
                <motion.div className="flex gap-1 z-[5] py-2 px-4 absolute top-full mt-1 rounded-lg shadow-md border border-neutral-100 dark:border-white/5 dark:bg-white/20 translate-y-full bg-white" initial={{
                    y: -10,
                    opacity: 0,
                }} animate={{
                    y: 0,
                    opacity: 1,
                }} exit={{
                    y: -10,
                    opacity: 0,
                }}>
                    <div className="hidden absolute inset-0 rounded-lg dark:flex dark:backdrop-blur-xl -z-[1]" />
                    <Dropdown noItemsLabel="No fields found" className="min-w-[100px]" value={createDropdownItem(currentFilter.field)} items={fieldsDropdownItems} onChange={handleFieldSelect} />
                    <Dropdown noItemsLabel="No operators found" className="min-w-20" value={createDropdownItem(currentFilter.operator)} items={validOperators} onChange={handleOperatorSelector} />
                    <Input inputProps={{
                        className: "min-w-[150px]",
                    }} placeholder="Enter filter value" value={currentFilter.value} setValue={handleInputChange} />
                    <AnimatedButton className="dark:bg-white/5" icon={Icons.Cancel} label="Cancel" onClick={handleClick} />
                    <AnimatedButton className="dark:bg-white/5" icon={Icons.CheckCircle} label="Add" onClick={handleAddFilter} />
                </motion.div>
            }
        </AnimatePresence>
    </div>
}