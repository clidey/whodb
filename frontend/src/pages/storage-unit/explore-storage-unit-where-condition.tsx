import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { ActionButton, AnimatedButton } from "../../components/button";
import { createDropdownItem, Dropdown, IDropdownItem } from "../../components/dropdown";
import { Icons } from "../../components/icons";
import { Input, Label } from "../../components/input";

export type IExploreStorageUnitWhereConditionFilter = {
    field: string;
    operator: string;
    value: string;
}

type IExploreStorageUnitWhereConditionProps = {
    options: string[];
    operators: string[];
    onChange?: (filters: IExploreStorageUnitWhereConditionFilter[]) => void;
}

export const ExploreStorageUnitWhereCondition: FC<IExploreStorageUnitWhereConditionProps> = ({ options, onChange, operators }) => {
    const [newFilter, setNewFilter] = useState<IExploreStorageUnitWhereConditionFilter>({ field: options[0], operator: operators[0], value: "" });
    const [filters, setFitlers] = useState<IExploreStorageUnitWhereConditionFilter[]>([]);
    const [show, setShow] = useState(false);

    const handleClick = useCallback(() => {
        setShow(s => !s);
    }, []);

    const fieldsDropdownItems = useMemo(() => {
        return options.map(option => createDropdownItem(option));
    }, [options]);

    const handleFieldSelect = useCallback((item: IDropdownItem) => {
        setNewFilter(val => ({
            ...val,
            field: item.id,
        }));
    }, []);

    const handleOperatorSelector = useCallback((item: IDropdownItem) => {
        setNewFilter(val => ({
            ...val,
            operator: item.id,
        }));
    }, []);

    const handleInputChange = useCallback((newValue: string) => {
        setNewFilter(val => ({
            ...val,
            value: newValue,
        }));
    }, []);

    const handleAddFilter = useCallback(() => {
        const newFilters = [...filters, newFilter];
        setFitlers(newFilters);
        setNewFilter({ field: newFilter.field, operator: operators[0], value: "" });
        onChange?.(newFilters);
    }, [filters, newFilter, onChange, operators]);

    const handleRemove = useCallback((index: number) => {
        setFitlers(oldFilters => oldFilters.filter((_, i) => i !== index));
    }, []);

    useEffect(() => {
        setNewFilter(f => ({
            ...f,
            operator: operators[0],
        }));
    }, [operators]);

    const validOperators = useMemo(() => {
        return operators.map(operator => createDropdownItem(operator));
    }, [operators]);
    
    return <div className="flex flex-col gap-1 h-full relative">
        <Label label="Where condition" />
        <div className="flex gap-1 items-center max-w-[min(500px,calc(100vw-20px))] flex-wrap">
            {
                filters.map((filter, i) => (
                    <div className="group/filter-item flex gap-1 items-center text-xs px-2 py-1 rounded-2xl dark:bg-white/5 cursor-pointer relative overflow-hidden shadow-sm border border-neutral-100 dark:border-neutral-800"
                        onClick={() => handleRemove(i)}>
                        <div className="max-w-[350px] truncate  dark:text-neutral-300">
                            {filter.field} {filter.operator} {filter.value}
                        </div>
                        <ActionButton icon={Icons.Cancel} containerClassName="hover:scale-125 absolute right-0 top-1/2 -translate-y-1/2 z-10 h-4 w-4 opacity-0 group-hover/filter-item:opacity-100" />
                    </div>
                ))
            }
            <ActionButton className={classNames("transition-all", {
                "rotate-45": show,
            })} icon={Icons.Add} containerClassName="h-8 w-8" onClick={handleClick} />
        </div>
        <AnimatePresence mode="wait">
            {
                show &&
                <motion.div className="flex gap-1 z-[5] py-2 px-4 absolute top-full mt-1 rounded-lg shadow-md border border-neutral-100  dark:border-white/5 dark:bg-white/20 dark:backdrop-blur-xl translate-y-full bg-white" initial={{
                    y: -10,
                    opacity: 0,
                }} animate={{
                    y: 0,
                    opacity: 1,
                }} exit={{
                    y: -10,
                    opacity: 0,
                }}>
                    <Dropdown className="min-w-[100px]" value={createDropdownItem(newFilter.field)} items={fieldsDropdownItems} onChange={handleFieldSelect} />
                    <Dropdown className="min-w-20" value={createDropdownItem(newFilter.operator)} items={validOperators} onChange={handleOperatorSelector} />
                    <Input inputProps={{
                        className: "min-w-[150px]"
                    }} placeholder="Enter filter value" value={newFilter.value} setValue={handleInputChange} />
                    <AnimatedButton className="dark:bg-white/5" icon={Icons.CheckCircle} label="Add" onClick={handleAddFilter} />
                </motion.div>
            }
        </AnimatePresence>
    </div>
}