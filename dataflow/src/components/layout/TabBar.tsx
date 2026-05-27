import React, { useState } from 'react';
import { X, FileCode, Table, Database, Plus, SplitSquareHorizontal } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip';
import { useTabStore, type Tab, type TabType } from '@/stores/useTabStore';
import { cn } from '@/lib/utils';
import { ContextMenu } from '@/components/ui/ContextMenu';
import { ScrollArea } from "@/components/ui/scroll-area";
import { useI18n } from '@/i18n/useI18n';

function getTabIcon(type: TabType) {
    switch (type) {
        case 'query':
            return <FileCode className="h-4 w-4" />;
        case 'table':
            return <Table className="h-4 w-4" />;
        case 'collection':
            return <Database className="h-4 w-4" />;
        default:
            return <FileCode className="h-4 w-4" />;
    }
}

interface TabItemProps {
    tab: Tab;
    isActive: boolean;
    onActivate: () => void;
    onClose: (e: React.MouseEvent) => void;
    onContextMenu: (e: React.MouseEvent) => void;
    closeTitle: string;
}

function TabItem({ tab, isActive, onActivate, onClose, onContextMenu, closeTitle }: TabItemProps) {
    return (
        <div
            onClick={onActivate}
            onContextMenu={onContextMenu}
            data-testid="layout.tab.item"
            data-qa-module="layout"
            data-qa-object="tab"
            data-qa-action="activate"
            data-qa-state={[isActive ? 'active' : 'inactive', tab.isDirty ? 'dirty' : null].filter(Boolean).join(' ')}
            data-qa-resource-type="tab"
            data-qa-resource-id={tab.id}
            data-qa-tab-type={tab.type}
            data-qa-connection-id={tab.connectionId}
            data-qa-database={tab.databaseName}
            data-qa-schema={tab.schemaName}
            className={cn(
                "group flex items-center gap-1 p-2 pl-3 h-9 cursor-pointer border-r border-sidebar-border transition-colors duration-150 select-none",
                isActive
                    ? "bg-input text-foreground"
                    : "text-foreground hover:bg-muted"
            )}
        >
            <span className="flex-shrink-0 mr-1">
                {getTabIcon(tab.type)}
            </span>
            <span className="truncate text-sm font-normal whitespace-nowrap">
                {tab.title}
                {tab.isDirty && <span className="text-primary ml-1">•</span>}
            </span>
            <Tooltip>
                <TooltipTrigger asChild>
                    <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={onClose}
                        data-testid="layout.tab.close-button"
                        data-qa-module="layout"
                        data-qa-object="tab"
                        data-qa-action="close"
                        data-qa-resource-type="tab"
                        data-qa-resource-id={tab.id}
                        className={cn(
                            "flex-shrink-0 transition-colors text-muted-foreground cursor-pointer",
                            isActive
                                ? "hover:bg-muted-foreground/20"
                                : "hover:bg-input"
                        )}
                    >
                        <X className="h-4 w-4" />
                    </Button>
                </TooltipTrigger>
                <TooltipContent>{closeTitle}</TooltipContent>
            </Tooltip>
        </div>
    );
}

export function TabBar() {
    const { tabs, activeTabId, setActiveTab, closeTab, closeOtherTabs, closeAllTabs, openTab } = useTabStore();
    const { t } = useI18n();
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; tabId: string } | null>(null);

    if (tabs.length === 0) {
        return null;
    }

    const handleClose = (e: React.MouseEvent, tabId: string) => {
        e.stopPropagation();
        closeTab(tabId);
    };

    const handleContextMenu = (e: React.MouseEvent, tabId: string) => {
        e.preventDefault();
        setContextMenu({ x: e.clientX, y: e.clientY, tabId });
    };

    const handleMenuAction = (action: 'close' | 'closeOthers' | 'closeAll') => {
        if (!contextMenu) return;

        switch (action) {
            case 'close':
                closeTab(contextMenu.tabId);
                break;
            case 'closeOthers':
                closeOtherTabs(contextMenu.tabId);
                break;
            case 'closeAll':
                closeAllTabs();
                break;
        }
        setContextMenu(null);
    };

    const handleAddTab = () => {
        const activeTab = tabs.find(tab => tab.id === activeTabId);
        if (!activeTab) return;
        openTab({
            type: 'query',
            title: activeTab.databaseName
                ? t('sidebar.tab.queryWithDatabase', { database: activeTab.databaseName })
                : t('layout.tab.newQuery'),
            connectionId: activeTab.connectionId,
            databaseName: activeTab.databaseName,
            schemaName: activeTab.schemaName,
        });
    };

    return (
        <ScrollArea
            className="border-b border-sidebar-border mb-2"
            data-testid="layout.tab-bar"
            data-qa-module="layout"
            data-qa-object="tab-bar"
            data-qa-state={tabs.length > 0 ? 'ready' : 'empty'}
        >
            <div className="flex items-center pr-2">
                {tabs.map(tab => (
                    <TabItem
                        key={tab.id}
                        tab={tab}
                        isActive={tab.id === activeTabId}
                        onActivate={() => setActiveTab(tab.id)}
                        onClose={(e) => handleClose(e, tab.id)}
                        onContextMenu={(e) => handleContextMenu(e, tab.id)}
                        closeTitle={t('layout.tab.close')}
                    />
                ))}
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="ghost"
                            size="icon-xs"
                            onClick={handleAddTab}
                            data-testid="layout.tab.new-query-button"
                            data-qa-module="layout"
                            data-qa-object="tab"
                            data-qa-action="create-query"
                            data-qa-disabled-reason={!activeTabId ? 'not_ready' : undefined}
                            className="h-9 w-9 shrink-0 rounded-none border-r border-sidebar-border hover:bg-muted"
                        >
                            <Plus className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t('layout.tab.newQuery')}</TooltipContent>
                </Tooltip>
            </div>

            {contextMenu && (
                <ContextMenu
                    x={contextMenu.x}
                    y={contextMenu.y}
                    onClose={() => setContextMenu(null)}
                    items={[
                        {
                            label: t('layout.tab.close'),
                            onClick: () => handleMenuAction('close'),
                            icon: <X className="h-4 w-4" />
                        },
                        {
                            label: t('layout.tab.closeOthers'),
                            onClick: () => handleMenuAction('closeOthers'),
                            icon: <SplitSquareHorizontal className="h-4 w-4" />
                        },
                        { separator: true } as const,
                        {
                            label: t('layout.tab.closeAll'),
                            onClick: () => handleMenuAction('closeAll'),
                            icon: <X className="h-4 w-4" />
                        },
                    ]}
                />
            )}
        </ScrollArea>
    );
}
