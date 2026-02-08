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

import { cn, Command, CommandItem, CommandList } from "@clidey/ux";
import { FC, useEffect, useRef } from "react";

export type IntellisenseSuggestion = {
    value: string;
    type: 'column' | 'operator' | 'keyword';
    description?: string;
};

type SearchIntellisenseProps = {
    suggestions: IntellisenseSuggestion[];
    selectedIndex: number;
    onSelect: (suggestion: IntellisenseSuggestion) => void;
    onClose: () => void;
    position: { top: number; left: number; width: number };
};

/**
 * Intellisense dropdown for search autocomplete using @clidey/ux Command component
 */
export const SearchIntellisense: FC<SearchIntellisenseProps> = ({
    suggestions,
    selectedIndex,
    onSelect,
    onClose,
    position
}) => {
    const dropdownRef = useRef<HTMLDivElement>(null);

    // Handle click outside to close
    useEffect(() => {
        const handleClickOutside = (e: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
                onClose();
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, [onClose]);

    // Scroll selected item into view
    useEffect(() => {
        if (dropdownRef.current) {
            const selectedElement = dropdownRef.current.querySelector('[data-selected="true"]');
            if (selectedElement) {
                selectedElement.scrollIntoView({ block: 'nearest' });
            }
        }
    }, [selectedIndex]);

    if (suggestions.length === 0) {
        return null;
    }

    return (
        <div
            ref={dropdownRef}
            className="absolute w-full z-[100]"
            data-testid="search-intellisense-dropdown"
        >
            <Command className="border border-gray-200 dark:border-gray-700 rounded-md shadow-lg bg-white dark:bg-gray-800">
                <CommandList className="max-h-64">
                    {suggestions.map((suggestion, index) => (
                        <CommandItem
                            key={`${suggestion.type}-${suggestion.value}`}
                            value={suggestion.value}
                            onSelect={() => onSelect(suggestion)}
                            className={cn("cursor-pointer", {
                                "bg-accent": index === selectedIndex,
                            })}
                            data-selected={index === selectedIndex}
                            data-testid={`suggestion-${index}`}
                        >
                            <div className="flex items-center gap-2 w-full">
                                <span className={cn("text-xs font-semibold px-2 py-0.5 rounded", {
                                    "bg-blue-200 dark:bg-blue-800 text-blue-800 dark:text-blue-200": suggestion.type === 'column',
                                    "bg-purple-200 dark:bg-purple-800 text-purple-800 dark:text-purple-200": suggestion.type === 'operator',
                                    "bg-green-200 dark:bg-green-800 text-green-800 dark:text-green-200": suggestion.type === 'keyword',
                                })}>
                                    {suggestion.type === 'column' && 'COL'}
                                    {suggestion.type === 'operator' && 'OP'}
                                    {suggestion.type === 'keyword' && 'KEY'}
                                </span>
                                <div className="flex-1">
                                    <div className="font-mono text-sm">{suggestion.value}</div>
                                    {suggestion.description && (
                                        <div className="text-xs text-muted-foreground">
                                            {suggestion.description}
                                        </div>
                                    )}
                                </div>
                            </div>
                        </CommandItem>
                    ))}
                </CommandList>
            </Command>
        </div>
    );
};
