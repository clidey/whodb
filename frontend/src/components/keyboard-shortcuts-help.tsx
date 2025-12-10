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

import {Dialog, DialogContent, DialogHeader, DialogTitle,} from "@clidey/ux";
import {FC, useCallback, useEffect, useState} from "react";
import {useTranslation} from "@/hooks/use-translation";
import {getKeyDisplay, isMacPlatform} from "@/utils/platform";

interface ShortcutDef {
    keys: string[];
    description: string;
}

interface ShortcutCategory {
    title: string;
    shortcuts: ShortcutDef[];
}

const Kbd: FC<{ children: string }> = ({ children }) => (
    <kbd className="inline-flex items-center justify-center min-w-[1.5rem] h-6 px-1.5 text-xs font-medium bg-neutral-100 dark:bg-neutral-800 border border-neutral-300 dark:border-neutral-600 rounded shadow-sm">
        {children}
    </kbd>
);

const ShortcutRow: FC<{ shortcut: ShortcutDef }> = ({ shortcut }) => (
    <div className="flex items-center justify-between py-1.5">
        <span className="text-sm text-neutral-600 dark:text-neutral-400">
            {shortcut.description}
        </span>
        <div className="flex items-center gap-1">
            {shortcut.keys.map((key, idx) => (
                <span key={idx} className="flex items-center gap-0.5">
                    <Kbd>{getKeyDisplay(key)}</Kbd>
                    {idx < shortcut.keys.length - 1 && !isMacPlatform && (
                        <span className="text-neutral-400 text-xs">+</span>
                    )}
                </span>
            ))}
        </div>
    </div>
);

const ShortcutSection: FC<{ category: ShortcutCategory }> = ({ category }) => (
    <div className="mb-4">
        <h3 className="text-sm font-bold text-neutral-700 dark:text-neutral-200 mb-2">
            {category.title}
        </h3>
        <div className="divide-y divide-neutral-200 dark:divide-neutral-700">
            {category.shortcuts.map((shortcut, idx) => (
                <ShortcutRow key={idx} shortcut={shortcut} />
            ))}
        </div>
    </div>
);

interface KeyboardShortcutsHelpProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export const KeyboardShortcutsHelp: FC<KeyboardShortcutsHelpProps> = ({
    open,
    onOpenChange,
}) => {
    const { t } = useTranslation('components/keyboard-shortcuts-help');

    const shortcutCategories: ShortcutCategory[] = [
        {
            title: t('categoryGlobal', 'Global'),
            shortcuts: [
                { keys: ["?"], description: t('showShortcuts', 'Show keyboard shortcuts') },
                { keys: ["Mod", "K"], description: t('commandPalette', 'Open command palette') },
                { keys: ["Escape"], description: t('closeDialogs', 'Close dialogs/sheets') },
                { keys: ["Mod", "B"], description: t('toggleSidebar', 'Toggle sidebar') },
            ],
        },
        {
            title: t('categoryNavigation', 'Navigation'),
            shortcuts: [
                { keys: ["Alt", "1"], description: t('navFirst', 'Go to first view (Chat or Tables)') },
                { keys: ["Alt", "2"], description: t('navSecond', 'Go to second view') },
                { keys: ["Alt", "3"], description: t('navThird', 'Go to third view') },
                { keys: ["Alt", "4"], description: t('navFourth', 'Go to fourth view (if available)') },
            ],
        },
        {
            title: t('categoryTableNavigation', 'Table Navigation'),
            shortcuts: [
                { keys: ["ArrowDown"], description: t('moveDown', 'Move to next row') },
                { keys: ["ArrowUp"], description: t('moveUp', 'Move to previous row') },
                { keys: ["Home"], description: t('moveFirst', 'Move to first row') },
                { keys: ["End"], description: t('moveLast', 'Move to last row') },
                { keys: ["PageDown"], description: t('pageDown', 'Jump down by visible rows') },
                { keys: ["PageUp"], description: t('pageUp', 'Jump up by visible rows') },
                { keys: ["Mod", "ArrowRight"], description: t('nextPage', 'Go to next page') },
                { keys: ["Mod", "ArrowLeft"], description: t('prevPage', 'Go to previous page') },
            ],
        },
        {
            title: t('categoryTableSelection', 'Table Selection'),
            shortcuts: [
                { keys: ["Space"], description: t('toggleSelect', 'Toggle row selection') },
                { keys: ["Shift", "ArrowDown"], description: t('extendSelectDown', 'Extend selection down') },
                { keys: ["Shift", "ArrowUp"], description: t('extendSelectUp', 'Extend selection up') },
                { keys: ["Mod", "A"], description: t('selectAll', 'Select/deselect all rows') },
            ],
        },
        {
            title: t('categoryTableActions', 'Table Actions'),
            shortcuts: [
                { keys: ["Enter"], description: t('editRow', 'Edit focused row') },
                { keys: ["Mod", "Delete"], description: t('deleteRow', 'Delete focused row') },
                { keys: ["Mod", "Backspace"], description: t('deleteRowAlt', 'Delete focused row') },
                { keys: ["Mod", "E"], description: t('editRowAlt', 'Edit focused row') },
                { keys: ["Mod", "M"], description: t('mockData', 'Generate mock data') },
                { keys: ["Mod", "R"], description: t('refresh', 'Refresh table') },
                { keys: ["Mod", "Shift", "E"], description: t('export', 'Export data') },
            ],
        },
        {
            title: t('categoryEditor', 'Query Editor'),
            shortcuts: [
                { keys: ["Mod", "Enter"], description: t('executeQuery', 'Execute query') },
            ],
        },
    ];

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-lg max-h-[80vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        {t('title', 'Keyboard Shortcuts')}
                    </DialogTitle>
                </DialogHeader>
                <div className="mt-4">
                    {shortcutCategories.map((category, idx) => (
                        <ShortcutSection key={idx} category={category} />
                    ))}
                </div>
                <div className="mt-4 pt-4 border-t border-neutral-200 dark:border-neutral-700">
                    <p className="text-xs text-neutral-500 dark:text-neutral-400 text-center">
                        {t('hint', 'Press ? anywhere to show this dialog')}
                    </p>
                </div>
            </DialogContent>
        </Dialog>
    );
};

export const useKeyboardShortcutsHelp = () => {
    const [open, setOpen] = useState(false);

    const handleKeyDown = useCallback((event: KeyboardEvent) => {
        // Ignore if typing in an input or textarea
        if (
            event.target instanceof HTMLInputElement ||
            event.target instanceof HTMLTextAreaElement ||
            (event.target as HTMLElement)?.isContentEditable
        ) {
            return;
        }

        // ? key (Shift+/ on most keyboards, or direct ?)
        if (event.key === "?" || (event.shiftKey && event.key === "/")) {
            event.preventDefault();
            setOpen(true);
        }
    }, []);

    useEffect(() => {
        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, [handleKeyDown]);

    return {
        open,
        setOpen,
        KeyboardShortcutsHelpModal: (
            <KeyboardShortcutsHelp open={open} onOpenChange={setOpen} />
        ),
    };
};
