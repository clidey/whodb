import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { twMerge } from "tailwind-merge";
import { ActionButton, AnimatedButton } from "../../components/button";
import { createDropdownItem, Dropdown, IDropdownItem } from "../../components/dropdown";
import { Icons } from "../../components/icons";
import { Input, Label } from "../../components/input";
import { AtomicWhereCondition, WhereCondition, WhereConditionType } from "../../generated/graphql";

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

    const handleClick = useCallback(() => {
        const shouldShow = !newFilter;
        if (shouldShow) {
            setEditingFilter(-1);
            setCurrentFilter({ ColumnType: "string", Key: "", Operator: "", Value: "" });
        }
        setNewFilter(!newFilter);
    }, [newFilter]);

    const fieldsDropdownItems = useMemo(() => columns.map(column => createDropdownItem(column)), [columns]);

    const handleFieldSelect = useCallback((item: IDropdownItem) => {
        setCurrentFilter(val => ({ ...val, Key: item.id, ColumnType: columnTypes[columns.findIndex(col => col === item.id)] }));
    }, [columnTypes, columns]);

    const handleOperatorSelector = useCallback((item: IDropdownItem) => {
        setCurrentFilter(val => ({
            ...val,
            Operator: item.id,
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
        if (filters.And?.Children?.at(index) != null && filters.And?.Children?.at(index)?.Type === WhereConditionType.Atomic) {
            setCurrentFilter((filters.And?.Children?.at(index) ?? { ColumnType: "string", Key: "", Operator: "", Value: "" }) as AtomicWhereCondition);
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
        return operators.map(operator => createDropdownItem(operator));
    }, [operators]);

    useEffect(() => {
        if (defaultWhere == null) {
            return;
        }
        setFilters(defaultWhere);
    }, [defaultWhere]);

    return (
        <div className="flex flex-col gap-1 h-full relative">
            <Label label="Where condition" />
            <div className="flex gap-1 items-center max-w-[min(500px,calc(100vw-20px))] flex-wrap">
                {filters.And?.Children?.map((filter, i) => (
                    <div key={`explore-storage-unit-filter-${i}`} className="group/filter-item flex gap-1 items-center text-xs rounded-2xl dark:bg-white/5 cursor-pointer relative shadow-sm border border-neutral-100 dark:border-neutral-800"
                        data-testid="where-condition">
                        <div className={twMerge(classNames("px-2 py-1 h-full max-w-[350px] truncate dark:text-neutral-300 rounded-2xl", {
                            "dark:bg-white/10": editingFilter === i,
                        }))} onClick={() => handleEdit(i)}>
                            {filter.Atomic?.Key} {filter.Atomic?.Operator} {filter.Atomic?.Value}
                        </div>
                        <ActionButton icon={Icons.Cancel} containerClassName="hover:scale-125 absolute right-2 top-1/2 -translate-y-1/2 z-10 h-4 w-4 opacity-0 group-hover/filter-item:opacity-100"
                            onClick={() => handleRemove(i)} testId="remove-where-condition-button" />
                        <AnimatePresence mode="wait">
                            {editingFilter === i && (
                                <motion.div className="flex gap-1 z-[5] py-2 px-4 absolute left-0 top-2 mt-2 rounded-lg shadow-md border border-neutral-100 dark:border-[#23272A] dark:bg-neutral-700 dark:backdrop-blur-xl translate-y-full bg-white"
                                    initial={{ y: -10, opacity: 0 }}
                                    animate={{ y: 0, opacity: 1 }}
                                    exit={{ y: -10, opacity: 0 }}>
                                    <Dropdown className="min-w-[100px]" value={createDropdownItem(currentFilter.Key)} items={fieldsDropdownItems} onChange={handleFieldSelect} />
                                    <Dropdown noItemsLabel="No operators found" className="min-w-20" value={createDropdownItem(currentFilter.Operator)} items={validOperators} onChange={handleOperatorSelector} />
                                    <Input inputProps={{ className: "min-w-[150px]" }} placeholder="Enter filter value" value={currentFilter.Value} setValue={handleInputChange} />
                                    <AnimatedButton className="dark:bg-white/5" icon={Icons.Cancel} label="Cancel" onClick={() => handleEdit(i)} />
                                    <AnimatedButton className="dark:bg-white/5" icon={Icons.CheckCircle} label="Save" onClick={() => handleSaveFilter(i)} />
                                </motion.div>
                            )}
                        </AnimatePresence>
                    </div>
                ))}
                <ActionButton className={classNames("transition-all", { "rotate-45": newFilter })} icon={Icons.Add} containerClassName="h-8 w-8" onClick={handleClick} testId="where-button" />
            </div>
            <AnimatePresence mode="wait">
                {newFilter && (
                    <motion.div className="flex gap-1 z-[5] py-2 px-4 absolute top-2 mt-1 rounded-lg shadow-md border border-neutral-100 dark:border-white/5 dark:bg-[#23272A] translate-y-full bg-white dark:backdrop-blur-xl"
                        initial={{ y: -10, opacity: 0 }}
                        animate={{ y: 0, opacity: 1 }}
                        exit={{ y: -10, opacity: 0 }}>
                        <Dropdown className="min-w-[100px]" value={createDropdownItem(currentFilter.Key)} items={fieldsDropdownItems} onChange={handleFieldSelect} />
                        <Dropdown noItemsLabel="No operators found" className="min-w-20" value={createDropdownItem(currentFilter.Operator)} items={validOperators} onChange={handleOperatorSelector} />
                        <Input inputProps={{
                            className: "min-w-[150px]",
                        }} placeholder="Enter filter value" value={currentFilter.Value} setValue={handleInputChange} />
                        <AnimatedButton className="dark:bg-white/5" icon={Icons.Cancel} label="Cancel" onClick={handleClick} />
                        <AnimatedButton className="dark:bg-white/5" icon={Icons.CheckCircle} label="Add" onClick={handleAddFilter} />
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
};
