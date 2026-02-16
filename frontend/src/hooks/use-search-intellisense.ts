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

import { useCallback, useEffect, useMemo, useState } from "react";
import { IntellisenseSuggestion } from "../components/search-intellisense";

type UseSearchIntellisenseProps = {
    columns: string[];
    operators: string[];
    value: string;
    cursorPosition: number;
    inputRef: React.RefObject<HTMLInputElement>;
    onCursorPositionChange?: (position: number) => void;
    onValueChange: (value: string) => void;
};

type IntellisenseContext = {
    type: 'column' | 'operator' | 'value' | 'none';
    currentToken: string;
    tokenStart: number;
    tokenEnd: number;
};

/**
 * Hook to manage search intellisense state and suggestions
 */
export function useSearchIntellisense({
    columns,
    operators,
    value,
    cursorPosition,
    inputRef,
    onCursorPositionChange,
    onValueChange
}: UseSearchIntellisenseProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [selectedIndex, setSelectedIndex] = useState(0);
    const [dropdownPosition, setDropdownPosition] = useState({ top: 0, left: 0, width: 0 });

    // Parse the current context (what is the user typing?)
    const context = useMemo((): IntellisenseContext => {
        if (!value || cursorPosition === 0) {
            return { type: 'column', currentToken: '', tokenStart: 0, tokenEnd: 0 };
        }

        // Get text up to cursor
        const textBeforeCursor = value.slice(0, cursorPosition);

        // Find the start of the current token by looking backwards from cursor
        // Tokens are separated by spaces, operators, or AND/OR keywords
        let tokenStart = cursorPosition - 1;
        while (tokenStart >= 0) {
            const char = textBeforeCursor[tokenStart];
            // Break on whitespace or operator characters
            if (char === ' ' || char === '=' || char === '!' || char === '<' || char === '>') {
                tokenStart++;
                break;
            }
            tokenStart--;
        }
        tokenStart = Math.max(0, tokenStart);

        // Find end of token (look forward from cursor)
        let tokenEnd = cursorPosition;
        while (tokenEnd < value.length) {
            const char = value[tokenEnd];
            if (char === ' ' || char === '=' || char === '!' || char === '<' || char === '>') {
                break;
            }
            tokenEnd++;
        }

        const currentToken = value.slice(tokenStart, tokenEnd);

        // Determine context type based on what comes before
        const textBeforeToken = value.slice(0, tokenStart).trim();

        // Check if we're after AND/OR (suggesting column name)
        if (textBeforeToken.toUpperCase().endsWith('AND') ||
            textBeforeToken.toUpperCase().endsWith('OR') ||
            textBeforeToken === '') {
            return { type: 'column', currentToken, tokenStart, tokenEnd };
        }

        // Check if we're after a potential column name (suggesting operator)
        // Look for pattern: "word" followed by cursor
        const colonOrOperatorMatch = textBeforeToken.match(/([a-zA-Z_][a-zA-Z0-9_]*)\s*$/);
        if (colonOrOperatorMatch) {
            // Check if this word is a valid column
            const potentialColumn = colonOrOperatorMatch[1];
            if (columns.some(col => col.toLowerCase() === potentialColumn.toLowerCase())) {
                return { type: 'operator', currentToken, tokenStart, tokenEnd };
            }
        }

        // Check if we're typing an operator
        const operatorPattern = /([a-zA-Z_][a-zA-Z0-9_]*)\s*([=!<>A-Z]*)\s*$/i;
        const operatorMatch = textBeforeToken.match(operatorPattern);
        if (operatorMatch) {
            const [, potentialColumn, partialOperator] = operatorMatch;
            if (columns.some(col => col.toLowerCase() === potentialColumn.toLowerCase())) {
                if (partialOperator && !partialOperator.trim()) {
                    return { type: 'operator', currentToken, tokenStart, tokenEnd };
                }
                // Check if currentToken could be an operator
                if (currentToken && operators.some(op => op.toUpperCase().startsWith(currentToken.toUpperCase()))) {
                    return { type: 'operator', currentToken, tokenStart, tokenEnd };
                }
            }
        }

        // Default to column name suggestion
        return { type: 'column', currentToken, tokenStart, tokenEnd };
    }, [value, cursorPosition, columns, operators]);

    // Generate suggestions based on context
    const suggestions = useMemo((): IntellisenseSuggestion[] => {
        const token = context.currentToken.toLowerCase();

        if (context.type === 'column') {
            // Suggest columns and keywords (AND/OR)
            const columnSuggestions: IntellisenseSuggestion[] = columns
                .filter(col => col.toLowerCase().includes(token))
                .map(col => ({
                    value: col,
                    type: 'column' as const,
                    description: 'Column name'
                }));

            const keywordSuggestions: IntellisenseSuggestion[] = [];
            if (value.trim().length > 0) {
                if ('and'.includes(token) || token === '') {
                    keywordSuggestions.push({
                        value: 'AND',
                        type: 'keyword' as const,
                        description: 'Logical AND operator'
                    });
                }
                if ('or'.includes(token) || token === '') {
                    keywordSuggestions.push({
                        value: 'OR',
                        type: 'keyword' as const,
                        description: 'Logical OR operator'
                    });
                }
            }

            return [...columnSuggestions, ...keywordSuggestions];
        }

        if (context.type === 'operator') {
            // Suggest operators
            return operators
                .filter(op => op.toLowerCase().includes(token) || token === '')
                .map(op => ({
                    value: op,
                    type: 'operator' as const,
                    description: getOperatorDescription(op)
                }));
        }

        return [];
    }, [context, columns, operators, value]);

    // Reset selected index when suggestions change
    useEffect(() => {
        setSelectedIndex(0);
    }, [suggestions]);

    // Show dropdown when suggestions are available
    useEffect(() => {
        if (suggestions.length > 0 && context.currentToken.length > 0) {
            setIsOpen(true);
        } else if (suggestions.length === 0) {
            setIsOpen(false);
        }
    }, [suggestions, context.currentToken]);

    // Update dropdown position when open or when scrolling/resizing
    useEffect(() => {
        if (!isOpen || !inputRef.current) return;

        const updatePosition = () => {
            if (!inputRef.current) return;
            const rect = inputRef.current.getBoundingClientRect();
            setDropdownPosition({
                top: rect.bottom + 4,
                left: rect.left,
                width: rect.width,
            });
        };

        updatePosition();

        window.addEventListener('scroll', updatePosition, true);
        window.addEventListener('resize', updatePosition);

        return () => {
            window.removeEventListener('scroll', updatePosition, true);
            window.removeEventListener('resize', updatePosition);
        };
    }, [isOpen, inputRef]);

    const acceptSuggestion = useCallback((suggestion: IntellisenseSuggestion) => {
        const newValue = value.slice(0, context.tokenStart) +
                        suggestion.value +
                        (suggestion.type === 'operator' || suggestion.type === 'keyword' ? ' ' : '') +
                        value.slice(context.tokenEnd);

        // Update the value through callback
        onValueChange(newValue);

        // Set cursor position after the inserted text
        const newCursorPos = context.tokenStart + suggestion.value.length +
                            (suggestion.type === 'operator' || suggestion.type === 'keyword' ? 1 : 0);

        // Focus and set cursor position
        setTimeout(() => {
            if (inputRef.current) {
                inputRef.current.focus();
                inputRef.current.setSelectionRange(newCursorPos, newCursorPos);
            }
            onCursorPositionChange?.(newCursorPos);
        }, 0);

        setIsOpen(false);
    }, [value, context, inputRef, onCursorPositionChange, onValueChange]);

    const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
        if (!isOpen) {
            // Ctrl+Space to show suggestions
            if (e.key === ' ' && e.ctrlKey) {
                e.preventDefault();
                setIsOpen(true);
                return true;
            }
            return false;
        }

        // Handle navigation when dropdown is open
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            setSelectedIndex(prev => Math.min(prev + 1, suggestions.length - 1));
            return true;
        }

        if (e.key === 'ArrowUp') {
            e.preventDefault();
            setSelectedIndex(prev => Math.max(prev - 1, 0));
            return true;
        }

        if (e.key === 'Escape') {
            e.preventDefault();
            setIsOpen(false);
            return true;
        }

        if (e.key === 'Enter' || e.key === 'Tab') {
            e.preventDefault();
            if (suggestions[selectedIndex]) {
                acceptSuggestion(suggestions[selectedIndex]);
            }
            return true;
        }

        return false;
    }, [isOpen, suggestions, selectedIndex, acceptSuggestion]);

    return {
        isOpen,
        suggestions,
        selectedIndex,
        handleKeyDown,
        acceptSuggestion,
        closeDropdown: () => setIsOpen(false),
        openDropdown: () => setIsOpen(true),
        dropdownPosition,
    };
}

function getOperatorDescription(operator: string): string {
    const descriptions: Record<string, string> = {
        '=': 'Equal to',
        '!=': 'Not equal to',
        '<>': 'Not equal to',
        '>': 'Greater than',
        '<': 'Less than',
        '>=': 'Greater than or equal to',
        '<=': 'Less than or equal to',
        'LIKE': 'Pattern matching (use % for wildcard)',
        'NOT LIKE': 'Pattern not matching',
        'IN': 'Value in list',
        'NOT IN': 'Value not in list',
        'IS': 'Is (for NULL)',
        'IS NOT': 'Is not (for NULL)',
    };

    return descriptions[operator] || operator;
}
