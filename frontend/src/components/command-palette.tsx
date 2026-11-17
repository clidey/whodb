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
    Button,
    Command,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
    Dialog,
    DialogContent,
    Kbd,
    IconRefresh as ArrowPathIcon,
    IconUpDown as ChevronUpDownIcon,
    IconSettings as CogIcon,
} from '@clidey/ux';
import {useQuery} from '@apollo/client/react';
import {GetStorageUnitsDocument, type GetStorageUnitsQuery} from '@graphql';
import type {FC} from "react";
import { useCallback, useEffect, useMemo, useState} from "react";
import {useNavigate} from "react-router-dom";
import {useTranslation} from "@/hooks/use-translation";
import {useAppSelector} from "@/store/hooks";
import {getKeyDisplay, isMacPlatform} from "@/utils/platform";
import {matchesShortcut, resolveShortcut, SHORTCUTS} from "@/utils/shortcuts";
import {useEffectiveIsMac} from "@/hooks/useEffectiveIsMac";
import {useSourceContract} from "@/hooks/useSourceContract";
import {InternalRoutes} from "@/config/routes";
import {performLogout} from "@/config/logout-handler";
import {WhoDBChatIcon} from './whodb-chat-icon';
import {buildSourceParentObjectRef, buildSourceParentRef} from '../utils/source-refs';

/** Controlled visibility shared by command palette implementations. */
export interface CommandPaletteProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

/** Renders platform-aware keyboard shortcut keycaps in a command palette item. */
export const CommandPaletteShortcut: FC<{ keys: string[]; isMac: boolean }> = ({keys, isMac}) => (
    <div className="ml-auto flex items-center gap-0.5">
        {keys.map((key, idx) => (
            <span key={key} className="flex items-center gap-0.5">
                <Kbd>
                    {getKeyDisplay(key)}
                </Kbd>
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
    keywords?: string;
    onSelect: () => void;
}

type PaletteCategory = 'all' | 'tables' | 'actions' | 'navigation';
const paletteCategories: PaletteCategory[] = ['all', 'tables', 'actions', 'navigation'];

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
    const [category, setCategory] = useState<PaletteCategory>('all');
    const [activeValue, setActiveValue] = useState('');
    const { supportsChat, supportsGraph, supportsScratchpad, supportsSqlImport, item } = useSourceContract(currentType);
    const parentRef = useMemo(() => buildSourceParentRef(item, current, schema), [item, current, schema]);
    const {data: sourceObjects} = useQuery(GetStorageUnitsDocument, {
        variables: {parent: parentRef},
        skip: !open || !item || !current,
    });
    const sourceTables = (sourceObjects?.StorageUnit ?? []).filter(unit => unit.Kind === item?.contract?.DefaultObjectKind);
    const handlePaletteOpenChange = (nextOpen: boolean) => {
        if (!nextOpen) {
            setSearch("");
            setCategory('all');
            setActiveValue('');
        }
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
                window.dispatchEvent(new CustomEvent('menu:trigger-import', {cancelable: true}));
                handlePaletteOpenChange(false);
            },
        });

        if (supportsSqlImport) {
            tableActions.push({
                id: "action-import-sql",
                label: t('importSql'),
                keywords: t('importSqlKeywords'),
                icon: <WhoDBChatIcon name="upload" />,
                onSelect: () => {
                    window.dispatchEvent(new CustomEvent('menu:trigger-sql-import'));
                    handlePaletteOpenChange(false);
                },
            });
        }

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

    const commandMode = search.trimStart().startsWith('>');
    const searchQuery = (commandMode ? search.trimStart().slice(1) : search).trim().toLowerCase();
    const matchesAction = (action: CommandAction) => action.label.toLowerCase().includes(searchQuery)
        || (action.keywords?.toLowerCase().includes(searchQuery) ?? false);
    const matchedNavigation = navigationActions.filter(matchesAction);
    const matchedActions = tableActions.filter(matchesAction).sort((left, right) =>
        Number(right.id === 'action-import-sql') - Number(left.id === 'action-import-sql'));
    const matchedTables = sourceTables.filter(unit => unit.Name.toLowerCase().includes(searchQuery));
    const matchedSortActions = sortActions.filter(matchesAction);
    const visibleNavigationActions = commandMode || category === 'all' || category === 'navigation' ? matchedNavigation : [];
    const visibleTableActions = commandMode || category === 'all' || category === 'actions' ? matchedActions : [];
    const visibleSortActions = (commandMode || category === 'actions' || (category === 'all' && searchQuery)) ? matchedSortActions : [];
    const visibleTables = !commandMode && (category === 'tables' || (category === 'all' && searchQuery)) ? matchedTables : [];
    const showNoResults = visibleTables.length + visibleSortActions.length + visibleNavigationActions.length + visibleTableActions.length === 0;
    const visibleValues = [
        ...visibleTableActions.map(action => action.id),
        ...visibleTables.map(unit => `table:${unit.Name}`),
        ...visibleNavigationActions.map(action => action.id),
        ...visibleSortActions.map(action => action.id),
    ];
    const selectedValue = visibleValues.includes(activeValue) ? activeValue : (visibleValues[0] ?? '');
    const selectedTable = visibleTables.find(unit => `table:${unit.Name}` === selectedValue);
    const categoryCounts: Record<PaletteCategory, number> = {
        all: matchedTables.length + matchedActions.length + matchedNavigation.length + matchedSortActions.length,
        tables: matchedTables.length,
        actions: matchedActions.length + matchedSortActions.length,
        navigation: matchedNavigation.length,
    };
    const openTable = (unit: GetStorageUnitsQuery['StorageUnit'][number]) => {
        void navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
            state: {unit, parentRef: buildSourceParentObjectRef(item, unit.Ref), trail: []},
        });
        handlePaletteOpenChange(false);
    };
    const actionRow = (action: CommandAction) => (
        <CommandItem key={action.id} value={action.id} onSelect={action.onSelect}
            data-testid={`command-${action.id}`} className="ce-palette-row">
            <span className="ce-palette-row-icon">{action.icon}</span>
            <span className="ce-palette-row-label">{action.label}</span>
            {action.shortcut && <CommandPaletteShortcut keys={action.shortcut} isMac={isMac} />}
        </CommandItem>
    );

    return (
        <Dialog open={open} onOpenChange={handlePaletteOpenChange}>
            <DialogContent className="ce-command-palette" showCloseButton={false} data-testid="command-palette">
                <Command loop shouldFilter={false} value={selectedValue} onValueChange={setActiveValue}>
                    <div className="ce-palette-search">
                        <CommandInput placeholder={t('searchPlaceholder')} data-testid="command-palette-input" value={search}
                            onValueChange={setSearch} onKeyDown={event => {
                                if (event.key !== 'Tab' || commandMode) return;
                                event.preventDefault();
                                const next = (paletteCategories.indexOf(category) + (event.shiftKey ? -1 : 1) + paletteCategories.length) % paletteCategories.length;
                                setCategory(paletteCategories[next]);
                            }} />
                        <kbd>{t('escapeKey')}</kbd>
                    </div>
                    {!commandMode && <div className="ce-palette-filters" role="group" aria-label={t('filterResults')}>
                        {paletteCategories.map(value => <Button key={value} variant="ghost" size="sm"
                            data-testid={`command-category-${value}`} data-active={category === value} aria-pressed={category === value}
                            onClick={() => { setCategory(value); }}>
                            {t(value)}{(value !== 'all' || searchQuery) && <span>{categoryCounts[value]}</span>}
                        </Button>)}
                        <span className="ce-palette-tab-hint">{t('tabCycles')}</span>
                    </div>}
                    <div className="ce-palette-body">
                        <CommandList className="ce-palette-list">
                            {showNoResults && <p className="ce-palette-empty" role="status" data-testid="command-palette-empty">{t('noResults')}</p>}
                            {visibleTableActions.length > 0 && <CommandGroup heading={<span className="ce-palette-group-title"><span>{t('actions')}</span><span>{current?.Database}</span></span>}>
                                {visibleTableActions.map(actionRow)}
                            </CommandGroup>}
                            {visibleTables.length > 0 && <CommandGroup heading={t('tables')}>
                                {visibleTables.map(unit => <CommandItem key={unit.Name} value={`table:${unit.Name}`} onSelect={() => { openTable(unit); }}
                                    data-testid={`command-table-${unit.Name}`} className="ce-palette-row">
                                    <span className="ce-palette-row-icon"><WhoDBChatIcon name="table" /></span>
                                    <span className="ce-palette-row-label">{unit.Name}</span>
                                    <span className="ce-palette-row-type">{t('tableType')}</span>
                                </CommandItem>)}
                            </CommandGroup>}
                            {visibleNavigationActions.length > 0 && <CommandGroup heading={t('navigation')}>{visibleNavigationActions.map(actionRow)}</CommandGroup>}
                            {visibleSortActions.length > 0 && <CommandGroup heading={t('sortBy')}>{visibleSortActions.map(actionRow)}</CommandGroup>}
                        </CommandList>
                        {selectedTable && <aside className="ce-palette-preview" data-testid="command-table-preview">
                            <div className="ce-palette-preview-title"><WhoDBChatIcon name="table" /><div><strong>{selectedTable.Name}</strong><p>{current?.Database}</p></div></div>
                            <dl>{selectedTable.Attributes.map(attribute => <div key={attribute.Key}><dt>{attribute.Key}</dt><dd>{attribute.Value}</dd></div>)}</dl>
                            <Button variant="secondary" onClick={() => { openTable(selectedTable); }}>{t('openTable')}</Button>
                        </aside>}
                    </div>
                    <footer className="ce-palette-footer">
                        <span><kbd>↑↓</kbd>{t('move')}<kbd>↵</kbd>{t('openItem')}</span>
                        <span><b>&gt;</b>{t('commandsHint')}</span>
                    </footer>
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

        if (matchesShortcut(event, SHORTCUTS.commandPalette, isMacPlatform)) {
            event.preventDefault();
            setOpen(prev => !prev);
        }
    }, [isLoggedIn]);

    useEffect(() => {
        window.addEventListener('keydown', handleKeyDown, true);
        return () => { window.removeEventListener('keydown', handleKeyDown, true); };
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
