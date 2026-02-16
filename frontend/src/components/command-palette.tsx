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
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
    Dialog,
    DialogContent,
} from "@clidey/ux";
import {FC, useCallback, useEffect, useState} from "react";
import {useNavigate} from "react-router-dom";
import {useTranslation} from "@/hooks/use-translation";
import {useAppSelector} from "@/store/hooks";
import {getKeyDisplay, isMacPlatform, isModKeyPressed} from "@/utils/platform";
import {isNoSQL} from "@/utils/functions";
import {databaseSupportsScratchpad} from "@/utils/database-features";
import {InternalRoutes} from "@/config/routes";
import {
    ArrowLeftStartOnRectangleIcon,
    ArrowPathIcon,
    ChatBubbleLeftRightIcon,
    ChevronUpDownIcon,
    CircleStackIcon,
    CogIcon,
    CommandLineIcon,
    RectangleGroupIcon,
    ShareIcon,
} from "./heroicons";

interface CommandPaletteProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

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
    const current = useAppSelector(state => state.auth.current);
    const isLoggedIn = useAppSelector(state => state.auth.status === "logged-in");
    const isEmbedded = useAppSelector(state => state.auth.isEmbedded);
    const [availableColumns, setAvailableColumns] = useState<string[]>([]);

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

    if (isLoggedIn && current) {
        // Navigation actions - only show relevant ones based on database type
        // Use Ctrl+Number on Mac (to avoid Option special chars), Alt+Number on Windows/Linux
        const navModKey = isMacPlatform ? "Ctrl" : "Alt";

        if (!isNoSQL(current.Type)) {
            navigationActions.push({
                id: "nav-chat",
                label: t('goToChat'),
                icon: <ChatBubbleLeftRightIcon className="w-4 h-4" />,
                shortcut: [navModKey, "1"],
                onSelect: () => {
                    navigate(InternalRoutes.Chat.path);
                    onOpenChange(false);
                },
            });
        }

        navigationActions.push({
            id: "nav-storage-units",
            label: t('goToStorageUnits'),
            icon: <RectangleGroupIcon className="w-4 h-4" />,
            shortcut: isNoSQL(current.Type) ? [navModKey, "1"] : [navModKey, "2"],
            onSelect: () => {
                navigate(InternalRoutes.Dashboard.StorageUnit.path);
                onOpenChange(false);
            },
        });

        navigationActions.push({
            id: "nav-graph",
            label: t('goToGraph'),
            icon: <ShareIcon className="w-4 h-4" />,
            shortcut: isNoSQL(current.Type) ? [navModKey, "2"] : [navModKey, "3"],
            onSelect: () => {
                navigate(InternalRoutes.Graph.path);
                onOpenChange(false);
            },
        });

        if (databaseSupportsScratchpad(current.Type)) {
            navigationActions.push({
                id: "nav-scratchpad",
                label: t('goToScratchpad'),
                icon: <CommandLineIcon className="w-4 h-4" />,
                shortcut: isNoSQL(current.Type) ? [navModKey, "3"] : [navModKey, "4"],
                onSelect: () => {
                    navigate(InternalRoutes.RawExecute.path);
                    onOpenChange(false);
                },
            });
        }

        // Table/Data actions
        tableActions.push({
            id: "action-refresh",
            label: t('refreshData'),
            icon: <ArrowPathIcon className="w-4 h-4" />,
            shortcut: ["Mod", "R"],
            onSelect: () => {
                window.dispatchEvent(new CustomEvent('app:refresh-data'));
                onOpenChange(false);
            },
        });

        tableActions.push({
            id: "action-export",
            label: t('exportData'),
            icon: <CircleStackIcon className="w-4 h-4" />,
            shortcut: ["Mod", "Shift", "E"],
            onSelect: () => {
                window.dispatchEvent(new CustomEvent('menu:trigger-export'));
                onOpenChange(false);
            },
        });

        tableActions.push({
            id: "action-import",
            label: t('importData'),
            icon: <CircleStackIcon className="w-4 h-4" />,
            shortcut: ["Mod", "Shift", "I"],
            onSelect: () => {
                window.dispatchEvent(new CustomEvent('menu:trigger-import'));
                onOpenChange(false);
            },
        });

        tableActions.push({
            id: "action-toggle-sidebar",
            label: t('toggleSidebar'),
            icon: <CogIcon className="w-4 h-4" />,
            shortcut: ["Mod", "B"],
            onSelect: () => {
                window.dispatchEvent(new CustomEvent('menu:toggle-sidebar'));
                onOpenChange(false);
            },
        });

        if (!isEmbedded) {
            tableActions.push({
                id: "action-disconnect",
                label: t('disconnect'),
                icon: <ArrowLeftStartOnRectangleIcon className="w-4 h-4" />,
                onSelect: () => {
                    navigate(InternalRoutes.Logout.path);
                    onOpenChange(false);
                },
            });
        }

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
                    onOpenChange(false);
                },
            });
        });
    }

    const renderShortcut = (keys: string[]) => (
        <div className="ml-auto flex items-center gap-0.5">
            {keys.map((key, idx) => (
                <span key={idx} className="flex items-center gap-0.5">
                    <kbd className="inline-flex items-center justify-center min-w-[1.5rem] h-6 px-1.5 text-xs font-medium bg-neutral-100 dark:bg-neutral-800 border border-neutral-300 dark:border-neutral-600 rounded shadow-sm">
                        {getKeyDisplay(key)}
                    </kbd>
                    {idx < keys.length - 1 && !isMacPlatform && (
                        <span className="text-neutral-400 text-xs">+</span>
                    )}
                </span>
            ))}
        </div>
    );

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="p-0 overflow-hidden max-w-md" data-testid="command-palette">
                <Command className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-neutral-500 dark:[&_[cmdk-group-heading]]:text-neutral-400">
                    <CommandInput
                        placeholder={t('searchPlaceholder')}
                        data-testid="command-palette-input"
                    />
                    <CommandList className="max-h-[400px]">
                        <CommandEmpty>{t('noResults')}</CommandEmpty>

                        {navigationActions.length > 0 && (
                            <CommandGroup heading={t('navigation')}>
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
                                        {action.shortcut && renderShortcut(action.shortcut)}
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
                                        {action.shortcut && renderShortcut(action.shortcut)}
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

        // Cmd+K (Mac) or Ctrl+K (Windows/Linux)
        if (isModKeyPressed(event) && event.key.toLowerCase() === 'k') {
            event.preventDefault();
            setOpen(prev => !prev);
        }
    }, [isLoggedIn]);

    useEffect(() => {
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [handleKeyDown]);

    return {
        open,
        setOpen,
        CommandPaletteModal: <CommandPalette open={open} onOpenChange={setOpen} />,
    };
};
