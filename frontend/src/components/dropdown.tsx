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
import { FC, ReactElement, cloneElement, useCallback, useState } from "react";
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

    const handleClick = useCallback((item: IDropdownItem) => {
        setOpen(false);
        props.onChange?.(item);
    }, [props]);
    
    const handleToggleOpen = useCallback(() => {
        setOpen(o => !o);
    }, []);

    const handleClose = useCallback(() => {
        setOpen(false);
    }, []);

    return (
        <div className={classNames("relative", props.className)}>
            {open && <div className="fixed inset-0" onClick={handleClose} />}
            {props.loading ? <div className="flex h-full w-full items-center justify-center">
                <Loading hideText={true} size="sm" />
            </div> :
            <>  <button className="group/dropdown flex gap-1 justify-between items-center border border-neutral-600/20 rounded-lg w-full p-1 h-[34px] px-2 dark:bg-[#2C2F33] dark:border-white/5" onClick={handleToggleOpen} data-testid={props.testId}>
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
                <div className={classNames("absolute z-10 divide-y rounded-lg shadow-sm bg-white py-1 border border-gray-200 overflow-y-auto max-h-40 dark:bg-[#2C2F33] dark:border-white/20", {
                    "hidden": !open,
                    "block animate-fade": open,
                    "w-fit min-w-[200px]": !props.fullWidth,
                    "w-full": props.fullWidth,
                }, props.dropdownContainerHeight)}>
                    <ul className={classNames(ClassNames.Text, "py-1 text-sm nowheel flex flex-col")}>
                        {
                            props.items.map((item, i) => (
                                <button key={`dropdown-item-${i}`} className={classNames(ITEM_CLASS, {
                                    "hover:gap-2": item.icon != null,
                                })} onClick={() => handleClick(item)} value={item.id}>
                                    <div>{props.value?.id === item.id ? Icons.CheckCircle : item.icon}</div>
                                    <div className="whitespace-nowrap">{item.label}</div>
                                    {(props.enableAction?.(i) ?? true) && props.action != null && cloneElement(props.action, {
                                        className: "absolute right-2 top-1/2 -translate-y-1/2 cursor-pointer transition-all opacity-0 group-hover/item:opacity-100",
                                        onClick: (e: MouseEvent) => {
                                            props.action?.props?.onClick?.(e, item);
                                            e.stopPropagation();
                                        },
                                    })}
                                </button>
                            ))
                        }
                        {
                            props.defaultItem != null &&
                            <button className={classNames(ITEM_CLASS, {
                                "hover:scale-105": props.defaultItem.icon == null,
                            }, props.defaultItemClassName)} onClick={props.onDefaultItemClick}>
                                <div>{props.defaultItem.icon}</div>
                                <div>{props.defaultItem.label}</div>
                            </button>
                        }
                        {
                            props.items.length === 0 && props.defaultItem == null &&
                            <button className="flex items-center gap-1 px-2 dark:text-neutral-300" onClick={props.onDefaultItemClick}>
                                <div>{Icons.SadSmile}</div>
                                <div>{props.noItemsLabel}</div>
                            </button>
                        }
                    </ul>
                </div>
            </>}
        </div>
    )
}

export const DropdownWithLabel: FC<IDropdownProps & { label: string, testId?: string }> = ({ label, testId, ...props }) => {
    return <div className="flex flex-col gap-1" data-testid={testId}>
        <Label label={label} />
        <Dropdown {...props} />
    </div>
}
