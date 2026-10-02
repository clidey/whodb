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

import {
    Command,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
    Dialog,
    DialogContent,
} from "@clidey/ux";
import {useQuery} from '@apollo/client/react';
import {GetStorageUnitsDocument} from '@graphql';
import type {FC} from "react";
import { useCallback, useEffect, useMemo, useState} from "react";
import {useNavigate} from "react-router-dom";
import {useTranslation} from "@/hooks/use-translation";
import {useAppSelector} from "@/store/hooks";
import {getKeyDisplay} from "@/utils/platform";
import {matchesShortcut, resolveShortcut, SHORTCUTS} from "@/utils/shortcuts";
import {useEffectiveIsMac} from "@/hooks/useEffectiveIsMac";
import {useSourceContract} from "@/hooks/useSourceContract";
import {InternalRoutes} from "@/config/routes";
import {performLogout} from "@/config/logout-handler";
import {WhoDBChatIcon} from './whodb-chat-icon';
import {buildSourceParentObjectRef, buildSourceParentRef} from '../utils/source-refs';
import {
    ArrowPathIcon,
    ChevronUpDownIcon,
    CogIcon,
} from "./heroicons";

export interface CommandPaletteProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

/** Shared command palette styling for command-group headings. */
export const COMMAND_PALETTE_GROUP_CLASS = "[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-neutral-500 dark:[&_[cmdk-group-heading]]:text-neutral-400";

/** Renders platform-aware keyboard shortcut keycaps in a command palette item. */
export const CommandPaletteShortcut: FC<{ keys: string[]; isMac: boolean }> = ({keys, isMac}) => (
    <div className="ml-auto flex items-center gap-0.5">
        {keys.map((key, idx) => (
            <span key={key} className="flex items-center gap-0.5">
                <kbd className="inline-flex items-center justify-center min-w-[1.5rem] h-6 px-1.5 text-xs font-medium bg-neutral-100 dark:bg-neutral-800 border border-neutral-300 dark:border-neutral-600 rounded shadow-sm">
                    {getKeyDisplay(key)}
                </kbd>
                {idx < keys.length - 1 && !isMac && (
                    <span className="text-neutral-400 text-xs">+</span>
                )}
            </span>
        ))}
    </div>
);

interface CommandAction {
    id: string;
    label: string;
    icon: React.ReactNode;
    shortcut?: string[];
    onSelect: () => void;
}

const CommandPalette: FC<CommandPaletteProps> = ({open, onOpenChange}) => {
    const {t} = useTranslation('components/command-palette');
    const navigate = useNavigate();
    const currentType = useAppSelector(state => state.auth.current?.Type);
    const current = useAppSelector(state => state.auth.current);
    const schema = useAppSelector(state => state.database.schema);
    const isLoggedIn = useAppSelector(state => state.auth.status === "logged-in");
    const isEmbedded = useAppSelector(state => state.auth.isEmbedded);
    const isMac = useEffectiveIsMac();
    const [availableColumns, setAvailableColumns] = useState<string[]>([]);
    const [search, setSearch] = useState("");
    const { supportsChat, supportsGraph, supportsScratchpad, item } = useSourceContract(currentType);
    const parentRef = useMemo(() => buildSourceParentRef(item, current, schema), [item, current, schema]);
    const {data: sourceObjects} = useQuery(GetStorageUnitsDocument, {
        variables: {parent: parentRef},
        skip: !open || !item || !current,
    });
    const sourceTables = (sourceObjects?.StorageUnit ?? []).filter(unit => unit.Kind === item?.contract?.DefaultObjectKind);
    const matchingTables = sourceTables.filter(unit => unit.Name.toLowerCase().includes(search.trim().toLowerCase()));
    const handlePaletteOpenChange = (nextOpen: boolean) => {
        if (!nextOpen) setSearch("");
        onOpenChange(nextOpen);
    };

    // Listen for columns broadcast from storage unit page
    useEffect(() => {
        const handleColumnsUpdate = (event: CustomEvent<{ columns: string[] }>) => {
            setAvailableColumns(event.detail.columns || []);
        };

        window.addEventListener('table:columns-available', handleColumnsUpdate as EventListener);
        return () => {
            window.removeEventListener('table:columns-available', handleColumnsUpdate as EventListener);
        };
    }, []);

    const navigationActions: CommandAction[] = [];
    const tableActions: CommandAction[] = [];
    const sortActions: CommandAction[] = [];

    if (isLoggedIn && currentType) {
        // Navigation actions - only show relevant ones based on database type
        const navDefs = [SHORTCUTS.navFirst, SHORTCUTS.navSecond, SHORTCUTS.navThird, SHORTCUTS.navFourth];

        let shortcutIndex = 0;

        if (supportsChat) {
            navigationActions.push({
                id: "nav-chat",
                label: t('goToChat'),
                icon: <WhoDBChatIcon name="chat" />,
                shortcut: resolveShortcut(navDefs[shortcutIndex]).displayKeys,
                onSelect: () => {
                    void navigate(InternalRoutes.Chat.path);
                    handlePaletteOpenChange(false);
                },
            });
            shortcutIndex += 1;
        }

        navigationActions.push({
            id: "nav-storage-units",
            label: t('goToStorageUnits'),
            icon: <WhoDBChatIcon name="table" />,
            shortcut: resolveShortcut(navDefs[shortcutIndex]).displayKeys,
            onSelect: () => {
                void navigate(InternalRoutes.Dashboard.StorageUnit.path);
                handlePaletteOpenChange(false);
            },
        });
        shortcutIndex += 1;

        if (supportsGraph) {
            navigationActions.push({
                id: "nav-graph",
                label: t('goToGraph'),
                icon: <WhoDBChatIcon name="relation" />,
                shortcut: resolveShortcut(navDefs[shortcutIndex]).displayKeys,
                onSelect: () => {
                    void navigate(InternalRoutes.Graph.path);
                    handlePaletteOpenChange(false);
                },
            });
            shortcutIndex += 1;
        }

        if (supportsScratchpad) {
            navigationActions.push({
                id: "nav-scratchpad",
                label: t('goToScratchpad'),
                icon: <WhoDBChatIcon name="code" />,
                shortcut: resolveShortcut(navDefs[shortcutIndex]).displayKeys,
                onSelect: () => {
                    void navigate(InternalRoutes.RawExecute.path);
                    handlePaletteOpenChange(false);
                },
            });
        }

        // Table/Data actions
        tableActions.push({
            id: "action-refresh",
            label: t('refreshData'),
            icon: <ArrowPathIcon className="w-4 h-4" />,
            shortcut: SHORTCUTS.refresh.displayKeys,
            onSelect: () => {
                window.dispatchEvent(new CustomEvent('app:refresh-data'));
                handlePaletteOpenChange(false);
            },
        });

        tableActions.push({
            id: "action-export",
            label: t('exportData'),
            icon: <WhoDBChatIcon name="download" />,
            shortcut: SHORTCUTS.exportData.displayKeys,
            onSelect: () => {
                window.dispatchEvent(new CustomEvent('menu:trigger-export'));
                handlePaletteOpenChange(false);
            },
        });

        if (!isEmbedded) {
            tableActions.push({
                id: "action-disconnect",
                label: t('disconnect'),
                icon: <WhoDBChatIcon name="signout" />,
                onSelect: () => {
                    performLogout(navigate);
                    handlePaletteOpenChange(false);
                },
            });
        }

        tableActions.push({
            id: "action-import",
            label: t('importData'),
            icon: <WhoDBChatIcon name="upload" />,
            shortcut: SHORTCUTS.importData.displayKeys,
            onSelect: () => {
                window.dispatchEvent(new CustomEvent('menu:trigger-import'));
                handlePaletteOpenChange(false);
            },
        });

        tableActions.push({
            id: "action-toggle-sidebar",
            label: t('toggleSidebar'),
            icon: <CogIcon className="w-4 h-4" />,
            shortcut: SHORTCUTS.toggleSidebar.displayKeys,
            onSelect: () => {
                window.dispatchEvent(new CustomEvent('menu:toggle-sidebar'));
                handlePaletteOpenChange(false);
            },
        });

        // Add sort actions for available columns
        availableColumns.forEach((column) => {
            sortActions.push({
                id: `sort-${column}`,
                label: t('sortByColumn', { column }),
                icon: <ChevronUpDownIcon className="w-4 h-4" />,
                onSelect: () => {
                    window.dispatchEvent(new CustomEvent('table:sort-column', {
                        detail: { column }
                    }));
                    handlePaletteOpenChange(false);
                },
            });
        });
    }

    return (
        <Dialog open={open} onOpenChange={handlePaletteOpenChange}>
            <DialogContent className="ce-command-palette p-0 overflow-hidden top-[18vh] translate-y-0" showCloseButton={false} data-testid="command-palette">
                <Command className={COMMAND_PALETTE_GROUP_CLASS} shouldFilter={false}>
                    <CommandInput
                        placeholder={t('searchPlaceholder')}
                        data-testid="command-palette-input"
                        value={search}
                        onValueChange={setSearch}
                    />
                    <kbd className="ce-command-escape" aria-hidden="true">esc</kbd>
                    <CommandList className="max-h-[400px]">
                        {matchingTables.length > 0 && <CommandGroup heading={t('tables')}>
                            {matchingTables.map(unit => <CommandItem key={unit.Name} value={unit.Name} onSelect={() => {
                                void navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
                                    state: {unit, parentRef: buildSourceParentObjectRef(item, unit.Ref), trail: []},
                                });
                                handlePaletteOpenChange(false);
                            }} data-testid={`command-table-${unit.Name}`}>
                                <WhoDBChatIcon name="table" /><span>{unit.Name}</span><kbd className="ce-command-enter" aria-hidden="true">↵</kbd>
                            </CommandItem>)}
                        </CommandGroup>}

                        {navigationActions.length > 0 && (
                            <CommandGroup heading={t('goTo')}>
                                {navigationActions.map((action) => (
                                    <CommandItem
                                        key={action.id}
                                        value={action.label}
                                        onSelect={action.onSelect}
                                        data-testid={`command-${action.id}`}
                                        className="text-muted-foreground"
                                    >
                                        {action.icon}
                                        <span className="ml-2">{action.label}</span>
                                        {action.shortcut && <CommandPaletteShortcut keys={action.shortcut} isMac={isMac} />}
                                    </CommandItem>
                                ))}
                            </CommandGroup>
                        )}

                        {tableActions.length > 0 && (
                            <CommandGroup heading={t('actions')}>
                                {tableActions.map((action) => (
                                    <CommandItem
                                        key={action.id}
                                        value={action.label}
                                        onSelect={action.onSelect}
                                        data-testid={`command-${action.id}`}
                                        className="text-muted-foreground"
                                    >
                                        {action.icon}
                                        <span className="ml-2">{action.label}</span>
                                        {action.shortcut && <CommandPaletteShortcut keys={action.shortcut} isMac={isMac} />}
                                    </CommandItem>
                                ))}
                            </CommandGroup>
                        )}

                        {sortActions.length > 0 && (
                            <CommandGroup heading={t('sortBy')}>
                                {sortActions.map((action) => (
                                    <CommandItem
                                        key={action.id}
                                        value={action.label}
                                        onSelect={action.onSelect}
                                        data-testid={`command-${action.id}`}
                                    >
                                        {action.icon}
                                        <span className="ml-2">{action.label}</span>
                                    </CommandItem>
                                ))}
                            </CommandGroup>
                        )}
                    </CommandList>
                </Command>
            </DialogContent>
        </Dialog>
    );
};

type CommandPaletteOverride = FC<CommandPaletteProps>;
let commandPaletteOverride: CommandPaletteOverride | null = null;

/** Register an override component for the command palette (used by EE). */
export const registerCommandPaletteOverride = (component: FC<CommandPaletteProps>) => {
    commandPaletteOverride = component;
};

export const useCommandPalette = () => {
    const [open, setOpen] = useState(false);
    const isLoggedIn = useAppSelector(state => state.auth.status === "logged-in");

    const handleKeyDown = useCallback((event: KeyboardEvent) => {
        // Only allow when logged in
        if (!isLoggedIn) return;

        // Skip if typing in an input
        if (
            event.target instanceof HTMLInputElement ||
            event.target instanceof HTMLTextAreaElement ||
            (event.target as HTMLElement)?.isContentEditable
        ) {
            return;
        }

        if (matchesShortcut(event, SHORTCUTS.commandPalette)) {
            event.preventDefault();
            setOpen(prev => !prev);
        }
    }, [isLoggedIn]);

    useEffect(() => {
        window.addEventListener('keydown', handleKeyDown);
        return () => { window.removeEventListener('keydown', handleKeyDown); };
    }, [handleKeyDown]);

    useEffect(() => {
        const handleOpen = () => { setOpen(true); };
        window.addEventListener('command-palette:open', handleOpen);
        return () => { window.removeEventListener('command-palette:open', handleOpen); };
    }, []);

    const PaletteComponent = commandPaletteOverride ?? CommandPalette;

    return {
        open,
        setOpen,
        CommandPaletteModal: <PaletteComponent open={open} onOpenChange={setOpen} />,
    };
};
