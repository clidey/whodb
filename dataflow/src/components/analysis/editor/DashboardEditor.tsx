import { useCallback, useEffect, useRef } from "react";
import { Plus, Layout, RefreshCw } from "lucide-react";

import { useAnalysisDefinitionStore } from "@/stores/analysisDefinitionStore";
import { useAnalysisRuntimeStore } from "@/stores/analysisRuntimeStore";
import { useAnalysisUIStore } from "@/stores/analysisUiStore";
import { Button } from "@/components/ui/Button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useI18n } from '@/i18n/useI18n'

import { EditorCanvas } from "./EditorCanvas";
import { ChartCreateModal } from "../chart-create/ChartCreateModal";
import { MaximizeChartModal } from "./MaximizeChartModal";
import { DeleteComponentModal } from "./DeleteComponentModal";

export function DashboardEditor() {
    const { t } = useI18n()
    const dashboards = useAnalysisDefinitionStore(state => state.dashboards);
    const activeDashboardId = useAnalysisDefinitionStore(state => state.activeDashboardId);
    const refreshDashboard = useAnalysisRuntimeStore(state => state.refreshDashboard);

    const isChartModalOpen = useAnalysisUIStore(state => state.isChartModalOpen);
    const editingWidgetId = useAnalysisUIStore(state => state.editingWidgetId);
    const maximizedWidgetId = useAnalysisUIStore(state => state.maximizedWidgetId);
    const deletingWidgetId = useAnalysisUIStore(state => state.deletingWidgetId);
    const openCreateChartModal = useAnalysisUIStore(state => state.openCreateChartModal);
    const openEditChartModal = useAnalysisUIStore(state => state.openEditChartModal);
    const setChartModalOpen = useAnalysisUIStore(state => state.setChartModalOpen);
    const setMaximizedWidgetId = useAnalysisUIStore(state => state.setMaximizedWidgetId);
    const setDeletingWidgetId = useAnalysisUIStore(state => state.setDeletingWidgetId);

    const dashboard = dashboards.find(d => d.id === activeDashboardId);
    const dashboardId = dashboard?.id;

    const handleRefresh = useCallback(async () => {
        if (!dashboardId) return;
        await refreshDashboard(dashboardId);
    }, [dashboardId, refreshDashboard]);

    const handleRefreshRef = useRef(handleRefresh);
    handleRefreshRef.current = handleRefresh;

    useEffect(() => {
        if (!dashboardId) return;
        void handleRefresh();
    }, [dashboardId, handleRefresh]);

    useEffect(() => {
        if (dashboard?.refreshRule !== 'by-minute') return;
        const id = setInterval(() => { void handleRefreshRef.current() }, 60_000);
        return () => clearInterval(id);
    }, [dashboard?.refreshRule]);

    if (!dashboard) return null;

    const handleEditComponent = (id: string) => {
        openEditChartModal(id);
    };

    return (
        <div
            className="flex flex-col h-full w-full bg-background overflow-hidden"
            data-testid="analysis.dashboard.editor"
            data-qa-module="analysis"
            data-qa-object="dashboard-editor"
            data-qa-state={dashboard.widgets.length > 0 ? 'ready' : 'empty'}
            data-qa-resource-type="dashboard"
            data-qa-resource-id={dashboard.id}
        >
            <div
                className="h-14 border-b flex items-center justify-between px-6 shrink-0 bg-background z-20"
                data-testid="analysis.dashboard.toolbar"
                data-qa-module="analysis"
                data-qa-object="dashboard-toolbar"
                data-qa-resource-type="dashboard"
                data-qa-resource-id={dashboard.id}
            >
                <div className="flex items-center gap-4">
                    <div className="flex flex-col justify-center">
                        <div className="font-bold text-lg leading-tight">
                            {dashboard.name}
                        </div>
                        {dashboard.description && (
                            <div className="text-xs text-muted-foreground leading-tight">
                                {dashboard.description}
                            </div>
                        )}
                    </div>
                </div>

                <TooltipProvider>
                    <div className="flex items-center gap-1 ml-auto">
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={() => { void handleRefresh() }}
                                    data-testid="analysis.dashboard.refresh-button"
                                    data-qa-module="analysis"
                                    data-qa-object="dashboard"
                                    data-qa-action="refresh"
                                    data-qa-resource-type="dashboard"
                                    data-qa-resource-id={dashboard.id}
                                >
                                    <RefreshCw className="w-4 h-4" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>{t('analysis.dashboard.refresh')}</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={openCreateChartModal}
                                    data-testid="analysis.dashboard.add-widget-button"
                                    data-qa-module="analysis"
                                    data-qa-object="widget"
                                    data-qa-action="create"
                                    data-qa-resource-type="dashboard"
                                    data-qa-resource-id={dashboard.id}
                                >
                                    <Plus className="w-4 h-4" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>{t('analysis.chart.add')}</TooltipContent>
                        </Tooltip>
                    </div>
                </TooltipProvider>
            </div>

            <div
                className="flex-1 overflow-auto"
                data-testid="analysis.dashboard.canvas-region"
                data-qa-module="analysis"
                data-qa-object="dashboard-canvas"
                data-qa-state={dashboard.widgets.length > 0 ? 'ready' : 'empty'}
            >
                {dashboard.widgets.length > 0 ? (
                    <EditorCanvas
                        dashboard={dashboard}
                        isReadOnly={false}
                        onEditComponent={handleEditComponent}
                        onMaximizeComponent={setMaximizedWidgetId}
                        onDeleteComponent={setDeletingWidgetId}
                    />
                ) : (
                    <div
                        className="h-full min-h-[600px] flex flex-col items-center justify-center text-center p-8"
                        data-testid="analysis.dashboard.editor-empty"
                        data-qa-module="analysis"
                        data-qa-object="dashboard-canvas"
                        data-qa-state="empty"
                    >
                        <div className="w-16 h-16 bg-muted/50 rounded-full flex items-center justify-center mb-4">
                            <Layout className="w-8 h-8 text-muted-foreground/50" />
                        </div>
                        <h3 className="text-lg font-medium text-foreground mb-2">
                            {t('analysis.editor.emptyTitle')}
                        </h3>
                        <p className="text-sm text-muted-foreground max-w-sm mb-6">
                            {t('analysis.editor.emptyDescription')}
                        </p>
                        <Button
                            onClick={openCreateChartModal}
                            data-testid="analysis.dashboard.empty-add-widget-button"
                            data-qa-module="analysis"
                            data-qa-object="widget"
                            data-qa-action="create"
                            data-qa-resource-type="dashboard"
                            data-qa-resource-id={dashboard.id}
                        >
                            <Plus className="w-4 h-4" />
                            {t('analysis.chart.add')}
                        </Button>
                    </div>
                )}
            </div>

            <ChartCreateModal
                open={isChartModalOpen}
                onOpenChange={(open) => {
                    if (!open) {
                        setChartModalOpen(false);
                    }
                }}
                editComponentId={editingWidgetId}
            />

            <MaximizeChartModal
                open={!!maximizedWidgetId}
                onOpenChange={(open) => { if (!open) setMaximizedWidgetId(null) }}
                componentId={maximizedWidgetId}
            />

            <DeleteComponentModal
                open={!!deletingWidgetId}
                onOpenChange={(open) => { if (!open) setDeletingWidgetId(null) }}
                componentId={deletingWidgetId ?? ''}
                onSuccess={() => setDeletingWidgetId(null)}
            />
        </div>
    );
}
