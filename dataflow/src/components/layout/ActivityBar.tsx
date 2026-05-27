import React from "react";
import { Database, LayoutDashboard } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { cn } from "@/lib/utils";
import { useI18n } from "@/i18n/useI18n";
import type { ActivityTab } from "@/stores/useLayoutStore";

interface ActivityBarProps {
    activeTab: ActivityTab;
    onTabChange: (tab: ActivityTab) => void;
}

export function ActivityBar({ activeTab, onTabChange }: ActivityBarProps) {
    const { t } = useI18n();
    const tabs: { id: ActivityTab; icon: React.ElementType; label: string }[] = [
        { id: 'connections', icon: Database, label: t('layout.activity.connections') },
        { id: 'analysis', icon: LayoutDashboard, label: t('layout.activity.analysis') },
    ];

    const renderTab = (tab: { id: ActivityTab; icon: React.ElementType; label: string }) => {
        const Icon = tab.icon;
        const isActive = activeTab === tab.id;

        return (
            <Button
                key={tab.id}
                variant="ghost"
                onClick={() => onTabChange(tab.id)}
                data-testid="layout.activity.tab"
                data-qa-module="layout"
                data-qa-object="activity-tab"
                data-qa-action="switch"
                data-qa-state={isActive ? 'active' : 'inactive'}
                data-qa-resource-type="activity-tab"
                data-qa-resource-id={tab.id}
                className={cn(
                    "flex size-16 flex-col items-center justify-center gap-1 rounded-lg px-2 py-3",
                    isActive
                        ? "bg-input text-foreground"
                        : "text-foreground hover:bg-input/50"
                )}
            >
                <Icon className="size-6" />
                <span className="text-xs leading-4">{tab.label}</span>
            </Button>
        );
    };

    return (
        <div
            className="flex h-full w-20 flex-col items-center border-r bg-background px-2 py-1.5"
            data-testid="layout.activity-bar"
            data-qa-module="layout"
            data-qa-object="activity-bar"
            data-qa-state={activeTab}
        >
            <div className="flex flex-col gap-2">
                {tabs.map(renderTab)}
            </div>
        </div>
    );
}
