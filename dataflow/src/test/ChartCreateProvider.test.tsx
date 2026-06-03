import { fireEvent, screen, waitFor } from '@testing-library/react'
import { useEffect } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { DEFAULT_CHART_CONFIG } from '@/components/analysis/chart-utils'
import { ChartCreateProvider, useChartCreateCtx } from '@/components/analysis/chart-create/ChartCreateProvider'
import { useAnalysisDefinitionStore } from '@/stores/analysisDefinitionStore'
import { renderWithI18n } from '@/test/renderWithI18n'

const addWidget = vi.fn()

function SaveHarness() {
  const {
    setTitle,
    setSqlQuery,
    handleConfigChange,
    handleQueryResults,
    handleSave,
  } = useChartCreateCtx()

  useEffect(() => {
    setTitle('Revenue chart')
    setSqlQuery('select month, revenue from revenue_by_month')
    handleConfigChange({
      chartType: 'bar',
      xAxisColumn: 'month',
      yAxisColumns: ['revenue'],
      options: DEFAULT_CHART_CONFIG.options,
      sortBy: 'data',
      sortOrder: 'asc',
    })
    handleQueryResults(
      ['month', 'revenue'],
      [{ month: 'Jan', revenue: 10 }],
      { database: 'analytics', schema: 'public' },
    )
  }, [handleConfigChange, handleQueryResults, setSqlQuery, setTitle])

  return <button onClick={handleSave}>save</button>
}

describe('ChartCreateProvider', () => {
  beforeEach(() => {
    addWidget.mockReset()
    addWidget.mockResolvedValue({
      id: 'widget-1',
      type: 'chart',
      title: 'Revenue chart',
      layout: { i: 'widget-1', x: 0, y: 0, w: 4, h: 6 },
      sortOrder: 0,
    })

    useAnalysisDefinitionStore.setState({
      dashboards: [],
      activeDashboardId: 'dash-1',
      isInitialized: true,
      loadError: null,
      addWidget,
    })
  })

  it('builds visualization and snapshot payloads on save', async () => {
    const onClose = vi.fn()

    renderWithI18n(
      <ChartCreateProvider onClose={onClose}>
        <SaveHarness />
      </ChartCreateProvider>,
      'en',
    )

    fireEvent.click(screen.getByText('save'))

    await waitFor(() => {
      expect(addWidget).toHaveBeenCalledTimes(1)
    })

    expect(addWidget).toHaveBeenCalledWith('dash-1', expect.objectContaining({
      title: 'Revenue chart',
      query: 'select month, revenue from revenue_by_month',
      queryContext: { database: 'analytics', schema: 'public' },
      visualization: {
        chartConfig: expect.objectContaining({
          chartType: 'bar',
          xAxisColumn: 'month',
          yAxisColumns: ['revenue'],
        }),
      },
      snapshot: expect.objectContaining({
        config: expect.objectContaining({
          type: 'bar',
          xAxis: ['Jan'],
        }),
        data: {},
        executedAt: expect.any(String),
      }),
    }))

    expect(onClose).toHaveBeenCalled()
  })
})
