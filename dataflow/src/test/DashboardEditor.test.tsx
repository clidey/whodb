import { screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { DashboardEditor } from '@/components/analysis/editor/DashboardEditor'
import { DEFAULT_CHART_CONFIG } from '@/components/analysis/chart-utils'
import { renderWithI18n } from '@/test/renderWithI18n'
import { useAnalysisDefinitionStore } from '@/stores/analysisDefinitionStore'
import { useAnalysisRuntimeStore } from '@/stores/analysisRuntimeStore'
import { useAnalysisUIStore } from '@/stores/analysisUiStore'

vi.mock('@/components/ui/SafeECharts', () => ({
  SafeECharts: ({ option }: { option: any }) => (
    <pre data-testid="chart-option">{JSON.stringify(option)}</pre>
  ),
}))

describe('DashboardEditor', () => {
  beforeEach(() => {
    useAnalysisDefinitionStore.setState({
      dashboards: [{
        id: 'dash-1',
        name: 'Revenue',
        description: 'Revenue dashboard',
        refreshRule: 'on-demand',
        createdAt: '2026-04-02T00:00:00Z',
        updatedAt: '2026-04-02T00:00:00Z',
        widgets: [{
          id: 'widget-1',
          type: 'chart',
          title: 'Monthly revenue',
          description: 'Primary KPI',
          layout: { i: 'widget-1', x: 0, y: 0, w: 4, h: 6 },
          visualization: { chartConfig: DEFAULT_CHART_CONFIG },
          snapshot: {
            config: {
              type: 'bar',
              xAxis: ['Jan'],
              series: [{ name: 'revenue', type: 'bar', data: [10] }],
              chartConfig: DEFAULT_CHART_CONFIG,
            },
            data: {},
            executedAt: '2026-04-02T00:00:00Z',
          },
          sortOrder: 0,
        }],
      }],
      activeDashboardId: 'dash-1',
      isInitialized: true,
      loadError: null,
    })

    useAnalysisRuntimeStore.setState({
      widgetStatesById: {},
      refreshDashboard: vi.fn().mockImplementation(() => new Promise(() => {})),
      refreshWidget: vi.fn(),
      clearDashboardRuntime: vi.fn(),
    })

    useAnalysisUIStore.setState({
      isChartModalOpen: false,
      editingWidgetId: null,
      selectedWidgetId: null,
      maximizedWidgetId: null,
      deletingWidgetId: null,
    })
  })

  it('renders snapshot data before the live refresh completes', async () => {
    renderWithI18n(<DashboardEditor />)

    expect(screen.getByText('Monthly revenue')).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByTestId('chart-option').textContent).toContain('"Jan"')
    })
  })

  it('does not restart dashboard refresh when a widget snapshot changes', async () => {
    const refreshDashboard = vi.fn(async () => {
      useAnalysisDefinitionStore.setState(state => ({
        dashboards: state.dashboards.map(dashboard => ({
          ...dashboard,
          widgets: dashboard.widgets.map(widget => widget.id === 'widget-1'
            ? {
                ...widget,
                snapshot: {
                  ...widget.snapshot!,
                  executedAt: '2026-04-02T00:01:00Z',
                },
              }
            : widget),
        })),
      }))
    })

    useAnalysisRuntimeStore.setState({
      widgetStatesById: {},
      refreshDashboard,
      refreshWidget: vi.fn(),
      clearDashboardRuntime: vi.fn(),
    })

    renderWithI18n(<DashboardEditor />)

    await waitFor(() => {
      expect(refreshDashboard).toHaveBeenCalledTimes(1)
    })

    await new Promise(resolve => setTimeout(resolve, 50))

    expect(refreshDashboard).toHaveBeenCalledTimes(1)
  })
})
