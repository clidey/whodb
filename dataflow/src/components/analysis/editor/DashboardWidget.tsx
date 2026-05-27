import React, { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";
import type { ChartWidgetDefinition } from "@/stores/analysisDefinitionStore";
import { useAnalysisDefinitionStore } from "@/stores/analysisDefinitionStore";
import { useAnalysisRuntimeStore } from "@/stores/analysisRuntimeStore";
import { cn } from "@/lib/utils";
import { GripHorizontal, MoreVertical, Trash2, Maximize2, Settings, ImageDown } from "lucide-react";
import { SafeECharts, NativeEChartsHandle } from "@/components/ui/SafeECharts";
import { buildWidgetChartOption } from "../chart-utils";
import { downloadBlob } from "@/utils/export-utils";
import { ContextMenu } from "../../ui/ContextMenu";
import { useI18n } from '@/i18n/useI18n'

interface DashboardWidgetProps {
    widget: ChartWidgetDefinition;
    isReadOnly: boolean;
    isSelected: boolean;
    onEdit?: (id: string) => void;
    onMaximize?: (id: string) => void;
    onDelete?: (id: string) => void;
    onSelect: (id: string) => void;
}

export function DashboardWidget({
    widget,
    isReadOnly,
    isSelected,
    onEdit,
    onMaximize,
    onDelete,
    onSelect
}: DashboardWidgetProps) {
    const { t } = useI18n()
    const runtimeState = useAnalysisRuntimeStore(state => state.widgetStatesById[widget.id]);
    const updateWidget = useAnalysisDefinitionStore(state => state.updateWidget);
    const [contextMenu, setContextMenu] = useState<{
        x: number; y: number;
        side: "top" | "right" | "bottom" | "left";
        align: "start" | "end";
    } | null>(null);
    const [isRenaming, setIsRenaming] = useState(false);
    const [renameValue, setRenameValue] = useState(widget.title);
    const renameInputRef = useRef<HTMLInputElement>(null);
    const chartRef = useRef<NativeEChartsHandle>(null);

    useEffect(() => {
        if (isRenaming) {
            renameInputRef.current?.focus();
            renameInputRef.current?.select();
        }
    }, [isRenaming]);

    const commitRename = () => {
        const trimmed = renameValue.trim();
        if (trimmed && trimmed !== widget.title) {
            void updateWidget(widget.id, { title: trimmed });
        } else {
            setRenameValue(widget.title);
        }
        setIsRenaming(false);
    };

    const handleContextMenu = (e: React.MouseEvent) => {
        e.preventDefault();
        e.stopPropagation();
        setContextMenu({ x: e.clientX, y: e.clientY, side: "bottom", align: "start" });
    };

    const handleExportPNG = async () => {
        setContextMenu(null);
        const blob = await chartRef.current?.exportPNG(2);
        if (!blob) return;
        downloadBlob(blob, `${widget.title || t('analysis.defaultTitle.chart')}.png`);
    };

    const menuItems = [
        {
            label: t('analysis.widget.maximize'),
            icon: <Maximize2 className="w-4 h-4" />,
            onClick: () => onMaximize?.(widget.id)
        },
        ...(widget.type === 'chart' ? [{
            label: t('analysis.chart.exportPng'),
            icon: <ImageDown className="w-4 h-4" />,
            onClick: handleExportPNG
        }] : []),
        {
            label: t('analysis.widget.settings'),
            icon: <Settings className="w-4 h-4" />,
            onClick: () => onEdit?.(widget.id)
        },
        {
            label: t('analysis.widget.delete'),
            icon: <Trash2 className="w-4 h-4" />,
            danger: true,
            onClick: () => onDelete?.(widget.id)
        }
    ];

    return (
        <div
            data-testid="analysis.dashboard.widget"
            data-qa-module="analysis"
            data-qa-object="widget"
            data-qa-state={runtimeState?.status ?? 'idle'}
            data-qa-loading={runtimeState?.status === 'loading' ? 'true' : 'false'}
            data-qa-error-code={runtimeState?.status === 'error' ? 'widget_query_failed' : undefined}
            data-qa-resource-type="widget"
            data-qa-resource-id={widget.id}
            className={cn(
                "bg-accent rounded-lg overflow-clip flex flex-col p-1 relative h-full transition-all group",
                isSelected ? "ring-2 ring-primary" : "",
                isReadOnly && "pointer-events-auto"
            )}
            onClick={(e) => {
                e.stopPropagation();
                onSelect(widget.id);
            }}
            onContextMenu={!isReadOnly ? handleContextMenu : undefined}
        >
            {/* Widget Header */}
            <div className="h-9 flex items-center justify-between shrink-0 relative z-10">
                <div
                    data-testid="analysis.dashboard.widget-title"
                    data-qa-module="analysis"
                    data-qa-object="widget"
                    data-qa-field="title"
                    data-qa-state={isRenaming ? 'editing' : 'ready'}
                    data-qa-resource-type="widget"
                    data-qa-resource-id={widget.id}
                    className={cn(
                        "flex items-center h-8 px-2.5 rounded-md w-[45%] min-w-0 transition-colors",
                        isRenaming ? "bg-input" : !isReadOnly && "hover:bg-input cursor-text"
                    )}
                    onDoubleClick={() => {
                        if (isReadOnly) return;
                        setRenameValue(widget.title);
                        setIsRenaming(true);
                    }}
                >
                    {!isReadOnly && isRenaming ? (
                        <input
                            ref={renameInputRef}
                            value={renameValue}
                            data-testid="analysis.dashboard.widget-title-input"
                            data-qa-module="analysis"
                            data-qa-object="widget"
                            data-qa-field="title"
                            data-qa-state="editing"
                            data-qa-resource-type="widget"
                            data-qa-resource-id={widget.id}
                            onChange={(e) => setRenameValue(e.target.value)}
                            onBlur={commitRename}
                            onKeyDown={(e) => {
                                if (e.key === 'Enter') commitRename();
                                if (e.key === 'Escape') {
                                    setRenameValue(widget.title);
                                    setIsRenaming(false);
                                }
                            }}
                            onMouseDown={(e) => e.stopPropagation()}
                            className="text-sm text-foreground bg-transparent outline-none w-full"
                        />
                    ) : (
                        <span className="text-sm text-foreground truncate">{widget.title}</span>
                    )}
                </div>

                {!isReadOnly && (
                    <GripHorizontal className="w-4 h-4 text-foreground/40 cursor-grab drag-handle absolute left-1/2 -translate-x-1/2 top-1/2 -translate-y-1/2" />
                )}

                {!isReadOnly && (
                    <button
                        onMouseDown={(e) => e.stopPropagation()}
                        onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            const rect = e.currentTarget.getBoundingClientRect();
                            setContextMenu({ x: rect.right, y: rect.top, side: "right", align: "start" });
                        }}
                        data-testid="analysis.dashboard.widget-menu-button"
                        data-qa-module="analysis"
                        data-qa-object="widget"
                        data-qa-action="open-menu"
                        data-qa-resource-type="widget"
                        data-qa-resource-id={widget.id}
                        className={cn(
                            "flex items-center justify-center size-8 rounded-lg text-foreground/60 hover:bg-input transition-colors",
                            contextMenu && "bg-input"
                        )}
                    >
                        <MoreVertical className="w-4 h-4" />
                    </button>
                )}
            </div>

            {/* Widget Content */}
            <div className="flex-1 overflow-hidden relative">
                <div className="absolute inset-0 p-3 pb-6 z-10">
                    <WidgetContent widget={widget} chartRef={chartRef} runtimeState={runtimeState} />
                </div>
            </div>

            {runtimeState?.status === 'error' && (
                <div
                    className="px-3 pb-2 text-[10px] text-destructive truncate"
                    data-testid="analysis.dashboard.widget-error"
                    data-qa-module="analysis"
                    data-qa-object="widget"
                    data-qa-state="error"
                    data-qa-error-code="widget_query_failed"
                    data-qa-resource-type="widget"
                    data-qa-resource-id={widget.id}
                >
                    {runtimeState.error}
                </div>
            )}


            {/* Portal Context Menu — must escape react-grid-layout's CSS transform */}
            {contextMenu && createPortal(
                <ContextMenu
                    x={contextMenu.x}
                    y={contextMenu.y}
                    side={contextMenu.side}
                    align={contextMenu.align}
                    items={menuItems}
                    onClose={() => setContextMenu(null)}
                />,
                document.body
            )}
        </div>
    );
}

function WidgetContent({
    widget,
    chartRef,
    runtimeState,
}: {
    widget: ChartWidgetDefinition;
    chartRef: React.RefObject<NativeEChartsHandle | null>;
    runtimeState?: { status: 'idle' | 'loading' | 'success' | 'error'; config?: any; isStale: boolean };
}) {
    const { t } = useI18n()
    switch (widget.type) {
        case 'chart': {
            const config = runtimeState?.status === 'success' && !runtimeState.isStale
                ? runtimeState.config
                : widget.snapshot?.config;
            const option = buildWidgetChartOption(config);
            if (!option) return <div className="flex items-center justify-center h-full text-muted-foreground text-xs">{t('analysis.chart.noData')}</div>;
            return (
                <SafeECharts
                    ref={chartRef}
                    option={option}
                    className="h-full w-full overflow-hidden"
                />
            );
        }

        default:
            return (
                <div className="w-full h-full flex items-center justify-center text-muted-foreground text-xs">
                    {t('analysis.widget.unknownType', { type: widget.type })}
                </div>
            );
    }
}
