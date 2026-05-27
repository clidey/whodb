import { useState, useCallback, useRef, useEffect } from "react";
import { Sidebar } from "@/components/sidebar/Sidebar";
import { DashboardSidebar } from "@/components/dashboard-sidebar";
import { useLayoutStore } from "@/stores/useLayoutStore";

import { ActivityBar } from "./ActivityBar";
import { AnalysisView } from "../analysis/AnalysisView";
import { TabBar } from "./TabBar";
import { TabContent } from "./TabContent";

const SIDEBAR_MIN_WIDTH = 180;
const SIDEBAR_MAX_WIDTH = 480;
const SIDEBAR_DEFAULT_WIDTH = 256;

export function MainLayout() {
    const activeTab = useLayoutStore(state => state.activeTab);
    const setActiveTab = useLayoutStore(state => state.setActiveTab);
    const [sidebarWidth, setSidebarWidth] = useState(SIDEBAR_DEFAULT_WIDTH);
    const isResizing = useRef(false);

    const handleMouseDown = useCallback((e: React.MouseEvent) => {
        e.preventDefault();
        isResizing.current = true;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    }, []);

    useEffect(() => {
        const handleMouseMove = (e: MouseEvent) => {
            if (!isResizing.current) return;
            // Account for ActivityBar width (80px = w-20)
            const newWidth = e.clientX - 80;
            setSidebarWidth(Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, newWidth)));
        };

        const handleMouseUp = () => {
            if (!isResizing.current) return;
            isResizing.current = false;
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        };

        document.addEventListener('mousemove', handleMouseMove);
        document.addEventListener('mouseup', handleMouseUp);
        return () => {
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', handleMouseUp);
        };
    }, []);

    return (
        <div
            className="flex h-screen w-full overflow-hidden bg-background"
            data-testid="layout.shell"
            data-qa-module="layout"
            data-qa-object="app-shell"
            data-qa-state={activeTab}
        >
            <ActivityBar activeTab={activeTab} onTabChange={setActiveTab} />

            <div
                className="relative shrink-0"
                style={{ width: sidebarWidth }}
                data-testid="layout.sidebar-region"
                data-qa-module="layout"
                data-qa-object="sidebar"
                data-qa-state={activeTab}
            >
                {activeTab === 'connections' ? <Sidebar /> : <DashboardSidebar />}
                <div
                    className="absolute top-0 right-0 h-full w-1 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 z-10"
                    onMouseDown={handleMouseDown}
                    data-testid="layout.sidebar-resize-handle"
                    data-qa-module="layout"
                    data-qa-object="sidebar"
                    data-qa-action="resize"
                />
            </div>

            <main
                className="flex flex-1 flex-col overflow-hidden relative bg-sidebar"
                data-testid="layout.main-region"
                data-qa-module="layout"
                data-qa-object="main"
                data-qa-state={activeTab}
            >
                {activeTab === 'connections' ? (
                    <>
                        <TabBar />
                        <TabContent />
                    </>
                ) : activeTab === 'analysis' ? (
                    <AnalysisView />
                ) : null}

            </main>
        </div>
    );
}
