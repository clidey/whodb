import { createContext, use, useRef, useState, useEffect, useCallback, type ReactNode } from 'react'

import { useAnalysisDefinitionStore, type DashboardDefinition, type ChartWidgetDefinition, type WidgetLayout } from '@/stores/analysisDefinitionStore'
import { useAnalysisRuntimeStore } from '@/stores/analysisRuntimeStore'
import { useConnectionStore } from '@/stores/useConnectionStore'
import { useLayoutStore } from '@/stores/useLayoutStore'
import { useTabStore } from '@/stores/useTabStore'
import { useWorkspaceTabLeaveGuard } from '@/components/layout/useWorkspaceTabLeaveGuard'
import {
    DEFAULT_CHART_CONFIG,
    buildEChartsOption,
    toWidgetConfig,
    fromWidgetConfig,
    type ChartConfig,
    type QueryData,
} from '../chart-utils'

type ModalView = 'chart-config' | 'data-config'

/** Pre-loaded data and connection context from a data view toolbar. */
export interface ChartCreateInitialData {
    connectionId: string
    databaseName: string
    schemaName?: string
    query: string
    columns: string[]
    rows: Record<string, any>[]
}

interface ChartCreateCtxValue {
    activeView: ModalView
    title: string
    chartConfig: ChartConfig
    queryData: QueryData | null
    sqlQuery: string
    isEditing: boolean
    canSave: boolean
    previewOption: ReturnType<typeof buildEChartsOption>
    editorContext: { connectionId: string; databaseName: string; schemaName?: string } | null
    dashboards: DashboardDefinition[]
    selectedDashboardId: string
    setSelectedDashboardId: (id: string) => void
    setActiveView: (view: ModalView) => void
    setTitle: (title: string) => void
    handleConfigChange: (updates: Partial<ChartConfig>) => void
    setSqlQuery: (sql: string) => void
    handleQueryResults: (
        columns: string[],
        rows: Record<string, any>[],
        ctx: { database?: string; schema?: string },
    ) => void
    handleSave: () => void
}

const ChartCreateCtx = createContext<ChartCreateCtxValue | null>(null)

export function useChartCreateCtx(): ChartCreateCtxValue {
    const ctx = use(ChartCreateCtx)
    if (!ctx) throw new Error('useChartCreateCtx must be used within ChartCreateProvider')
    return ctx
}

interface ChartCreateProviderProps {
    editComponent?: ChartWidgetDefinition | null
    initialQuery?: string
    initialData?: ChartCreateInitialData
    onClose: () => void
    children: ReactNode
}

function getNextWidgetLayout(widgets: ChartWidgetDefinition[]): WidgetLayout {
    const width = 4
    const height = 6

    if (widgets.length === 0) {
        return { i: crypto.randomUUID(), x: 0, y: 0, w: width, h: height }
    }

    const sorted = [...widgets].sort((a, b) => {
        if (a.layout.y !== b.layout.y) return a.layout.y - b.layout.y
        return a.layout.x - b.layout.x
    })
    const last = sorted[sorted.length - 1]!

    const maxY = widgets.reduce((currentMax, widget) => (
        Math.max(currentMax, widget.layout.y + widget.layout.h)
    ), 0)

    if (last.layout.x + last.layout.w + width <= 12) {
        return {
            i: crypto.randomUUID(),
            x: last.layout.x + last.layout.w,
            y: last.layout.y,
            w: width,
            h: height,
        }
    }

    return { i: crypto.randomUUID(), x: 0, y: maxY, w: width, h: height }
}

export function ChartCreateProvider({ editComponent, initialQuery, initialData, onClose, children }: ChartCreateProviderProps) {
    const addWidget = useAnalysisDefinitionStore(state => state.addWidget)
    const updateWidget = useAnalysisDefinitionStore(state => state.updateWidget)
    const createDashboard = useAnalysisDefinitionStore(state => state.createDashboard)
    const dashboards = useAnalysisDefinitionStore(state => state.dashboards)
    const activeDashboardId = useAnalysisDefinitionStore(state => state.activeDashboardId)
    const isInitialized = useAnalysisDefinitionStore(state => state.isInitialized)
    const initializeFromAPI = useAnalysisDefinitionStore(state => state.initializeFromAPI)
    const openDashboard = useAnalysisDefinitionStore(state => state.openDashboard)
    const { connections } = useConnectionStore()
    const tabs = useTabStore(state => state.tabs)
    const { confirmWorkspaceTabLeave, leaveGuardDialog } = useWorkspaceTabLeaveGuard()

    useEffect(() => {
        if (!isInitialized) void initializeFromAPI()
    }, [isInitialized, initializeFromAPI])

    const isEditing = !!editComponent
    const initialQueryData = editComponent ? fromWidgetConfig(editComponent) : null

    const [activeView, setActiveView] = useState<ModalView>('chart-config')
    const [title, setTitle] = useState(editComponent?.title ?? '')
    const [chartConfig, setChartConfig] = useState<ChartConfig>(
        editComponent?.visualization?.chartConfig ?? DEFAULT_CHART_CONFIG,
    )
    const chartConfigRef = useRef<ChartConfig>(editComponent?.visualization?.chartConfig ?? DEFAULT_CHART_CONFIG)
    const [queryData, setQueryData] = useState<QueryData | null>(
        initialQueryData ?? (initialData ? {
            columns: initialData.columns,
            rows: initialData.rows,
            query: initialData.query,
            database: initialData.databaseName,
            schema: initialData.schemaName,
        } : null),
    )
    const initialSql = editComponent?.query ?? initialData?.query ?? initialQuery ?? ''
    const [sqlQuery, setSqlQueryState] = useState(initialSql)
    const sqlQueryRef = useRef(initialSql)
    const [selectedDashboardId, setSelectedDashboardId] = useState<string | null>(null)
    const effectiveDashboardId = selectedDashboardId ?? activeDashboardId ?? dashboards[0]?.id ?? '__new__'

    const connection = connections[0]
    const editorContext = initialData
        ? { connectionId: initialData.connectionId, databaseName: initialData.databaseName, schemaName: initialData.schemaName }
        : connection
            ? { connectionId: connection.id, databaseName: connection.database }
            : null

    const setSqlQuery = useCallback((sql: string) => {
        sqlQueryRef.current = sql
        setSqlQueryState(sql)
    }, [])

    const handleConfigChange = useCallback((updates: Partial<ChartConfig>) => {
        const prev = chartConfigRef.current
        const next = {
            ...prev,
            ...updates,
            options: updates.options ? { ...prev.options, ...updates.options } : prev.options,
        }
        if (updates.xAxisColumn && updates.xAxisColumn !== prev.xAxisColumn) {
            next.yAxisColumns = next.yAxisColumns.filter(col => col !== updates.xAxisColumn)
        }
        chartConfigRef.current = next
        setChartConfig(next)
    }, [])

    const handleQueryResults = useCallback((
        columns: string[],
        rows: Record<string, any>[],
        ctx: { database?: string; schema?: string },
    ) => {
        setQueryData({
            columns,
            rows,
            query: sqlQueryRef.current,
            database: ctx.database,
            schema: ctx.schema,
        })
        setChartConfig(prev => {
            const colSet = new Set(columns)
            const base = chartConfigRef.current
            const next = {
                ...prev,
                ...base,
                xAxisColumn: colSet.has(base.xAxisColumn) ? base.xAxisColumn : '',
                yAxisColumns: base.yAxisColumns.filter(column => colSet.has(column)),
            }
            chartConfigRef.current = next
            return next
        })
    }, [])

    const canSave = title.trim() !== ''
        && queryData !== null
        && chartConfig.xAxisColumn !== ''
        && chartConfig.yAxisColumns.length > 0

    const previewOption = buildEChartsOption(chartConfig, queryData)

    const handleSave = useCallback(async () => {
        if (!queryData || !title.trim()) return

        if (!editComponent && initialData) {
            const confirmed = await confirmWorkspaceTabLeave(tabs)
            if (!confirmed) return
        }

        const { config, data } = toWidgetConfig(chartConfig, queryData)
        const executedAt = new Date().toISOString()
        const payload = {
            title: title.trim(),
            query: queryData.query,
            queryContext: { database: queryData.database, schema: queryData.schema },
            visualization: { chartConfig },
            snapshot: { config, data, executedAt },
        }

        let widgetId = editComponent?.id
        let targetId: string | undefined
        if (editComponent) {
            await updateWidget(editComponent.id, {
                ...payload,
                layout: editComponent.layout,
                sortOrder: editComponent.sortOrder,
            })
        } else if (effectiveDashboardId) {
            targetId = effectiveDashboardId
            let targetWidgets: ChartWidgetDefinition[] = []
            if (targetId === '__new__') {
                const created = await createDashboard(title.trim())
                targetId = created.id
            } else {
                targetWidgets = dashboards.find(d => d.id === targetId)?.widgets ?? []
            }
            const widget = await addWidget(targetId, {
                ...payload,
                layout: getNextWidgetLayout(targetWidgets),
                sortOrder: targetWidgets.length,
            })
            widgetId = widget.id
        }

        if (widgetId) {
            useAnalysisRuntimeStore.setState(state => ({
                widgetStatesById: {
                    ...state.widgetStatesById,
                    [widgetId]: {
                        status: 'success',
                        config,
                        data,
                        executedAt,
                        isStale: false,
                    },
                },
            }))
        }

        onClose()

        if (!editComponent && initialData && targetId) {
            openDashboard(targetId)
            useLayoutStore.getState().setActiveTab('analysis')
        }
    }, [
        queryData,
        title,
        editComponent,
        initialData,
        chartConfig,
        updateWidget,
        effectiveDashboardId,
        confirmWorkspaceTabLeave,
        tabs,
        createDashboard,
        dashboards,
        addWidget,
        onClose,
        openDashboard,
    ])

    return (
        <ChartCreateCtx value={{
            activeView,
            title,
            chartConfig,
            queryData,
            sqlQuery,
            isEditing,
            canSave,
            previewOption,
            editorContext,
            dashboards,
            selectedDashboardId: effectiveDashboardId,
            setSelectedDashboardId,
            setActiveView,
            setTitle,
            handleConfigChange,
            setSqlQuery,
            handleQueryResults,
            handleSave,
        }}>
            {children}
            {leaveGuardDialog}
        </ChartCreateCtx>
    )
}
