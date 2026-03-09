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
import {getKeyDisplay, getEffectiveIsMac} from "@/utils/platform";
import {matchesShortcut, resolveShortcut, SHORTCUTS} from "@/utils/shortcuts";

interface ShortcutEntry {
    keys: string[];
    description: string;
}

interface ShortcutCategory {
    title: string;
    shortcuts: ShortcutEntry[];
}

const Kbd: FC<{ children: string }> = ({ children }) => (
    <kbd className="inline-flex items-center justify-center min-w-[1.5rem] h-6 px-1.5 text-xs font-medium bg-muted border text-muted-foreground rounded shadow-sm">
        {children}
    </kbd>
);

const ShortcutRow: FC<{ shortcut: ShortcutEntry }> = ({ shortcut }) => (
    <div className="flex items-center justify-between py-1.5">
        <span className="text-sm text-neutral-600 dark:text-neutral-400">
            {shortcut.description}
        </span>
        <div className="flex items-center gap-1">
            {shortcut.keys.map((key, idx) => (
                <span key={idx} className="flex items-center gap-0.5">
                    <Kbd>{getKeyDisplay(key)}</Kbd>
                    {idx < shortcut.keys.length - 1 && !getEffectiveIsMac() && (
                        <span className="text-muted-foreground text-xs">+</span>
                    )}
                </span>
            ))}
        </div>
    </div>
);

const ShortcutSection: FC<{ category: ShortcutCategory; testId?: string }> = ({ category, testId }) => (
    <div className="mb-4" data-testid={testId}>
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
            title: t('categoryGlobal'),
            shortcuts: [
                { keys: SHORTCUTS.showShortcuts.displayKeys, description: t('showShortcuts') },
                { keys: SHORTCUTS.commandPalette.displayKeys, description: t('commandPalette') },
                { keys: SHORTCUTS.closeDialogs.displayKeys, description: t('closeDialogs') },
                { keys: SHORTCUTS.toggleSidebar.displayKeys, description: t('toggleSidebar') },
            ],
        },
        {
            title: t('categoryNavigation'),
            shortcuts: [
                { keys: resolveShortcut(SHORTCUTS.navFirst).displayKeys, description: t('navFirst') },
                { keys: resolveShortcut(SHORTCUTS.navSecond).displayKeys, description: t('navSecond') },
                { keys: resolveShortcut(SHORTCUTS.navThird).displayKeys, description: t('navThird') },
                { keys: resolveShortcut(SHORTCUTS.navFourth).displayKeys, description: t('navFourth') },
            ],
        },
        {
            title: t('categoryTableNavigation'),
            shortcuts: [
                { keys: SHORTCUTS.moveDown.displayKeys, description: t('moveDown') },
                { keys: SHORTCUTS.moveUp.displayKeys, description: t('moveUp') },
                { keys: SHORTCUTS.moveFirst.displayKeys, description: t('moveFirst') },
                { keys: SHORTCUTS.moveLast.displayKeys, description: t('moveLast') },
                { keys: SHORTCUTS.pageDown.displayKeys, description: t('pageDown') },
                { keys: SHORTCUTS.pageUp.displayKeys, description: t('pageUp') },
                { keys: SHORTCUTS.nextPage.displayKeys, description: t('nextPage') },
                { keys: SHORTCUTS.prevPage.displayKeys, description: t('prevPage') },
            ],
        },
        {
            title: t('categoryTableSelection'),
            shortcuts: [
                { keys: SHORTCUTS.toggleSelect.displayKeys, description: t('toggleSelect') },
                { keys: SHORTCUTS.extendSelectDown.displayKeys, description: t('extendSelectDown') },
                { keys: SHORTCUTS.extendSelectUp.displayKeys, description: t('extendSelectUp') },
                { keys: SHORTCUTS.selectAll.displayKeys, description: t('selectAll') },
            ],
        },
        {
            title: t('categoryTableActions'),
            shortcuts: [
                { keys: SHORTCUTS.editRow.displayKeys, description: t('editRow') },
                { keys: SHORTCUTS.deleteRow.displayKeys, description: t('deleteRow') },
                { keys: SHORTCUTS.deleteRowAlt.displayKeys, description: t('deleteRowAlt') },
                { keys: SHORTCUTS.editRowAlt.displayKeys, description: t('editRowAlt') },
                { keys: SHORTCUTS.mockData.displayKeys, description: t('mockData') },
                { keys: SHORTCUTS.refresh.displayKeys, description: t('refresh') },
                { keys: SHORTCUTS.exportData.displayKeys, description: t('export') },
                { keys: SHORTCUTS.importData.displayKeys, description: t('import') },
            ],
        },
        {
            title: t('categoryEditor'),
            shortcuts: [
                { keys: SHORTCUTS.executeQuery.displayKeys, description: t('executeQuery') },
                { keys: SHORTCUTS.clearEditor.displayKeys, description: t('clearEditor') },
            ],
        },
    ];

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-lg max-h-[80vh] overflow-y-auto" data-testid="shortcuts-modal">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        {t('title')}
                    </DialogTitle>
                </DialogHeader>
                <div className="mt-4">
                    {shortcutCategories.map((category, idx) => (
                        <ShortcutSection
                            key={idx}
                            category={category}
                            testId={`shortcuts-category-${category.title.toLowerCase().replace(/\s+/g, '-')}`}
                        />
                    ))}
                </div>
                <div className="mt-4 pt-4 border-t border-neutral-200 dark:border-neutral-700">
                    <p className="text-xs text-muted-foreground text-center">
                        {t('hint')}
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

        if (matchesShortcut(event, SHORTCUTS.showShortcuts)) {
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
