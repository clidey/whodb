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

import classNames from "classnames";
import { FC, ReactElement, cloneElement, useCallback, useState, useRef, useEffect, KeyboardEvent } from "react";
import { Icons } from "./icons";
import { Label } from "./input";
import { Loading } from "./loading";
import { ClassNames } from "./classes";

export function createDropdownItem(option: string, icon?: ReactElement): IDropdownItem {
    return {
        id: option,
        label: option,
        icon,
    };
}

export type IDropdownItem<T extends unknown = any> = {
    id: string;
    label: string;
    icon?: ReactElement;
    extra?: T;
    info?: ReactElement;
};

export type IDropdownProps = {
    className?: string;
    items: IDropdownItem[];
    loading?: boolean;
    value?: IDropdownItem;
    onChange?: (item: IDropdownItem) => void;
    fullWidth?: boolean;
    defaultItem?: Pick<IDropdownItem, "label" | "icon">;
    onDefaultItemClick?: () => void;
    defaultItemClassName?: string;
    action?: ReactElement;
    enableAction?: (index: number) => boolean;
    noItemsLabel?: string;
    showIconOnly?: boolean;
    testId?: string;
    dropdownContainerHeight?: string;
}

const ITEM_CLASS = "group/item flex items-center gap-1 transition-all cursor-pointer relative hover:bg-black/10 py-1 mx-2 px-4 rounded-lg pl-1 dark:text-neutral-300/100";

export const Dropdown: FC<IDropdownProps> = (props) => {
    const [open, setOpen] = useState(false);
    const [focusedIndex, setFocusedIndex] = useState(-1);
    const dropdownRef = useRef<HTMLDivElement>(null);
    const triggerRef = useRef<HTMLButtonElement>(null);
    const itemsRef = useRef<HTMLDivElement[]>([]);
    const blurTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const handleClick = useCallback((item: IDropdownItem) => {
        setOpen(false);
        props.onChange?.(item);
    }, [props]);
    
    const handleToggleOpen = useCallback(() => {
        setOpen(o => !o);
    }, []);

    const handleClose = useCallback(() => {
        setOpen(false);
        setFocusedIndex(-1);
        // Clear any pending blur timeout
        if (blurTimeoutRef.current) {
            clearTimeout(blurTimeoutRef.current);
            blurTimeoutRef.current = null;
        }
        // Ensure focus returns to trigger button
        setTimeout(() => {
            triggerRef.current?.focus();
        }, 0);
    }, []);

    const handleDropdownBlur = useCallback((event: React.FocusEvent<HTMLDivElement>) => {
        // Clear any existing timeout
        if (blurTimeoutRef.current) {
            clearTimeout(blurTimeoutRef.current);
        }
        
        // Set a timeout to check if focus moved outside the dropdown
        blurTimeoutRef.current = setTimeout(() => {
            if (dropdownRef.current && !dropdownRef.current.contains(document.activeElement)) {
                setOpen(false);
                setFocusedIndex(-1);
            }
        }, 100);
    }, []);

    const handleDropdownFocus = useCallback(() => {
        // Clear the blur timeout if focus returns to dropdown
        if (blurTimeoutRef.current) {
            clearTimeout(blurTimeoutRef.current);
            blurTimeoutRef.current = null;
        }
    }, []);

    const handleKeyDown = useCallback((event: KeyboardEvent<HTMLButtonElement>) => {
        switch (event.key) {
            case 'Enter':
            case ' ':
            case 'ArrowDown':
                event.preventDefault();
                setOpen(true);
                setFocusedIndex(0);
                break;
            case 'ArrowUp':
                event.preventDefault();
                setOpen(true);
                setFocusedIndex(props.items.length - 1);
                break;
            case 'Escape':
                handleClose();
                break;
        }
    }, [props.items.length, handleClose]);

    const handleItemKeyDown = useCallback((event: KeyboardEvent<HTMLDivElement>, item: IDropdownItem, index: number) => {
        switch (event.key) {
            case 'Enter':
            case ' ':
                event.preventDefault();
                handleClick(item);
                break;
            case 'ArrowDown':
                event.preventDefault();
                const nextIndex = Math.min(index + 1, props.items.length - 1);
                setFocusedIndex(nextIndex);
                break;
            case 'ArrowUp':
                event.preventDefault();
                const prevIndex = Math.max(index - 1, 0);
                setFocusedIndex(prevIndex);
                break;
            case 'Escape':
                event.preventDefault();
                handleClose();
                break;
            case 'Tab':
                handleClose();
                break;
        }
    }, [handleClick, props.items.length, handleClose]);

    useEffect(() => {
        if (open && focusedIndex >= 0 && itemsRef.current[focusedIndex]) {
            itemsRef.current[focusedIndex].focus();
        }
    }, [open, focusedIndex]);

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
                handleClose();
            }
        };

        if (open) {
            document.addEventListener('mousedown', handleClickOutside);
            return () => document.removeEventListener('mousedown', handleClickOutside);
        }
    }, [open, handleClose]);

    return (
        <div 
            ref={dropdownRef} 
            className={classNames("relative", props.className)}
            onBlur={handleDropdownBlur}
            onFocus={handleDropdownFocus}
        >
            {open && <div className="fixed inset-0" onClick={handleClose} />}
            {props.loading ? <div className="flex h-full w-full items-center justify-center">
                <Loading hideText={true} size="sm" />
            </div> :
            <>  <button 
                    ref={triggerRef}
                    tabIndex={0} 
                    className={ClassNames.Dropdown} 
                    onClick={handleToggleOpen} 
                    onKeyDown={handleKeyDown}
                    aria-haspopup="listbox"
                    aria-expanded={open}
                    aria-labelledby={props.testId ? `${props.testId}-label` : undefined}
                    data-testid={props.testId}>
                    <div className={classNames(ClassNames.Text, "flex gap-1 text-sm truncate items-center")}>
                        {props.value?.icon != null && <div className="flex items-center w-6">
                            {props.value.icon}
                        </div>}
                        {!props.showIconOnly && props.value?.label}
                    </div>
                    {cloneElement(Icons.DownCaret, {
                        className: "absolute right-2 top-1/2 -translate-y-1/2 p-1 w-5 h-5 stroke-neutral-700 dark:stroke-neutral-400 group-hover/dropdown:backdrop-blur-xs rounded-full",
                    })}
                </button>
                <div className={classNames(ClassNames.DropdownPanel, {
                    "hidden": !open,
                    "block animate-fade": open,
                    "w-fit min-w-[200px]": !props.fullWidth,
                    "w-full": props.fullWidth,
                }, props.dropdownContainerHeight)}
                     role="listbox"
                     aria-labelledby={props.testId ? `${props.testId}-label` : undefined}>
                    <ul className={classNames(ClassNames.Text, "py-1 text-sm nowheel flex flex-col")}>
                        {
                            props.items.map((item, i) => (
                                <div 
                                    role="option" 
                                    tabIndex={focusedIndex === i ? 0 : -1}
                                    key={`dropdown-item-${i}`} 
                                    ref={el => {
                                        if (el) itemsRef.current[i] = el;
                                    }}
                                    className={classNames(ITEM_CLASS, {
                                        "hover:gap-2": item.icon != null,
                                        "bg-blue-100 dark:bg-blue-900/30": focusedIndex === i,
                                    })} 
                                    onClick={() => handleClick(item)}
                                    onKeyDown={(e) => handleItemKeyDown(e, item, i)}
                                    aria-selected={props.value?.id === item.id}
                                    data-value={item.id}>
                                    <div>{props.value?.id === item.id ? Icons.CheckCircle : item.icon}</div>
                                    <div className="whitespace-nowrap flex-1">{item.label}</div>
                                    {item.info && (
                                        <div 
                                            className="ml-8"
                                            onClick={(e) => e.stopPropagation()}
                                        >
                                            {item.info}
                                        </div>
                                    )}
                                    {(props.enableAction?.(i) ?? true) && props.action != null && cloneElement(props.action, {
                                        className: classNames("cursor-pointer transition-all opacity-0 group-hover/item:opacity-100", {
                                            "absolute right-4 top-1/2 -translate-y-1/2": !item.info,
                                            "absolute right-10 top-1/2 -translate-y-1/2": item.info,
                                        }),
                                        onClick: (e: MouseEvent) => {
                                            props.action?.props?.onClick?.(e, item);
                                            e.stopPropagation();
                                        },
                                    })}
                                </div>
                            ))
                        }
                        {
                            props.defaultItem != null &&
                            <div 
                                role="option" 
                                tabIndex={0} 
                                className={classNames(ITEM_CLASS, {
                                    "hover:scale-105": props.defaultItem.icon == null,
                                }, props.defaultItemClassName)} 
                                onClick={props.onDefaultItemClick}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' || e.key === ' ') {
                                        e.preventDefault();
                                        props.onDefaultItemClick?.();
                                    }
                                }}>
                                <div>{props.defaultItem.icon}</div>
                                <div>{props.defaultItem.label}</div>
                            </div>
                        }
                        {
                            props.items.length === 0 && props.defaultItem == null &&
                            <div 
                                role="option" 
                                tabIndex={0} 
                                className="flex items-center gap-1 px-2 dark:text-neutral-300" 
                                onClick={props.onDefaultItemClick}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' || e.key === ' ') {
                                        e.preventDefault();
                                        props.onDefaultItemClick?.();
                                    }
                                }}>
                                <div>{Icons.SadSmile}</div>
                                <div>{props.noItemsLabel}</div>
                            </div>
                        }
                    </ul>
                </div>
            </>}
        </div>
    )
}

export const DropdownWithLabel: FC<IDropdownProps & { label: string, testId?: string }> = ({ label, testId, ...props }) => {
    const dropdownId = testId ? `${testId}-dropdown` : `dropdown-${label.toLowerCase().replace(/\s+/g, '-')}`;
    return <div className="flex flex-col gap-1" data-testid={testId}>
        <Label label={label} htmlFor={dropdownId} />
        <Dropdown {...props} testId={dropdownId} />
    </div>
}
