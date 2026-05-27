import { useMemo } from "react";
import { Responsive, WidthProvider } from "react-grid-layout";
import { useAnalysisDefinitionStore, type DashboardDefinition } from "@/stores/analysisDefinitionStore";
import { useAnalysisUIStore } from "@/stores/analysisUiStore";
import { DashboardWidget } from "./DashboardWidget";

// Import RGL styles
import "react-grid-layout/css/styles.css";
import "react-resizable/css/styles.css";

const ResponsiveGridLayout = WidthProvider(Responsive);

interface EditorCanvasProps {
    dashboard: DashboardDefinition;
    isReadOnly: boolean;
    onEditComponent?: (id: string) => void;
    onMaximizeComponent?: (id: string) => void;
    onDeleteComponent?: (id: string) => void;
}

export function EditorCanvas({ dashboard, isReadOnly, onEditComponent, onMaximizeComponent, onDeleteComponent }: EditorCanvasProps) {
    const updateWidgetLayouts = useAnalysisDefinitionStore(state => state.updateWidgetLayouts);
    const selectedWidgetId = useAnalysisUIStore(state => state.selectedWidgetId);
    const setSelectedWidgetId = useAnalysisUIStore(state => state.setSelectedWidgetId);

    const layouts = useMemo(() => {
        return {
            lg: dashboard.widgets.map(c => c.layout),
            md: dashboard.widgets.map((c, i) => ({
                ...c.layout,
                w: 5,
                h: 6, // Fixed height for consistency
                x: (i % 2) * 5,
                y: Math.floor(i / 2) * 6
            })),
            sm: dashboard.widgets.map((c, i) => ({
                ...c.layout,
                w: 3,
                h: 6, // Fixed height for consistency
                x: (i % 2) * 3,
                y: Math.floor(i / 2) * 6
            })),
            xs: dashboard.widgets.map((c, i) => ({ ...c.layout, w: 4, x: 0, y: i })), // 1 col
            xxs: dashboard.widgets.map((c, i) => ({ ...c.layout, w: 2, x: 0, y: i })) // 1 col
        };
    }, [dashboard.widgets]);

    const handleLayoutChange = (layout: any[]) => {
        if (!isReadOnly) {
            void updateWidgetLayouts(dashboard.id, layout);
        }
    };

    return (
        <div
            className="p-4 min-h-[800px]"
            onClick={() => setSelectedWidgetId(null)}
            data-testid="analysis.dashboard.canvas"
            data-qa-module="analysis"
            data-qa-object="dashboard-canvas"
            data-qa-resource-type="dashboard"
            data-qa-resource-id={dashboard.id}
            data-qa-state={isReadOnly ? 'read_only' : 'editable'}
        >
            <ResponsiveGridLayout
                className="layout"
                layouts={layouts}
                breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 }}
                cols={{ lg: 12, md: 10, sm: 6, xs: 4, xxs: 2 }}
                rowHeight={60}
                isDraggable={!isReadOnly}
                isResizable={!isReadOnly}
                onLayoutChange={handleLayoutChange}
                margin={[16, 16]}
                containerPadding={[0, 0]}
                draggableHandle=".drag-handle"
            >
                {dashboard.widgets.map(widget => (
                    <div
                        key={widget.layout.i}
                        className="h-full"
                        data-testid="analysis.dashboard.widget-layout"
                        data-qa-module="analysis"
                        data-qa-object="widget-layout"
                        data-qa-resource-type="widget"
                        data-qa-resource-id={widget.id}
                    >
                        <DashboardWidget
                            widget={widget}
                            isReadOnly={isReadOnly}
                            isSelected={selectedWidgetId === widget.id}
                            onEdit={onEditComponent}
                            onMaximize={onMaximizeComponent}
                            onDelete={onDeleteComponent}
                            onSelect={setSelectedWidgetId}
                        />
                    </div>
                ))}
            </ResponsiveGridLayout>
        </div>
    );
}
