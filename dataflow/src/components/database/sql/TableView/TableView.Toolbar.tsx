import { useState } from 'react'
import { Plus, Minus, Download, RefreshCw, Undo2, Eye, SendHorizontal, TerminalSquare, BarChart3 } from 'lucide-react'
import { useTableView } from './TableViewProvider'
import { DataView } from '@/components/database/shared/DataView'
import { Button } from '@/components/ui/Button'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { useI18n } from '@/i18n/useI18n'
import { cn } from '@/lib/utils'
import { Separator } from '@/components/ui/separator'
import { useTabStore } from '@/stores/useTabStore'
import { useConnectionStore } from '@/stores/useConnectionStore'
import { buildStorageUnitReference } from '@/utils/ddl-sql'
import { ChartCreateModal } from '@/components/analysis/chart-create'

interface TableViewToolbarProps {
  connectionId: string
  databaseName: string
  tableName: string
  schema?: string
}

export function TableViewToolbar({ connectionId, databaseName, tableName, schema }: TableViewToolbarProps) {
  const { t } = useI18n()
  const { state, actions } = useTableView()
  const openTab = useTabStore((s) => s.openTab)
  const connections = useConnectionStore((s) => s.connections)
  const [isChartModalOpen, setIsChartModalOpen] = useState(false)

  const handleOpenQuery = () => {
    const connectionType = connections.find((connection) => connection.id === connectionId)?.type
    const qualifiedName = buildStorageUnitReference(connectionType, tableName, schema)
    openTab({
      type: 'query',
      title: t('sidebar.tab.queryWithDatabase', { database: databaseName }),
      connectionId,
      databaseName,
      schemaName: schema,
      sqlContent: `SELECT * FROM ${qualifiedName};`,
    })
  }

  return (
    <div
      className="flex items-center justify-between h-12 px-2"
      data-testid="sql.table.toolbar"
      data-qa-module="sql"
      data-qa-object="table-toolbar"
      data-qa-state={state.loading ? 'loading' : state.hasPendingChanges ? 'dirty' : 'ready'}
      data-qa-connection-id={connectionId}
      data-qa-database={databaseName}
      data-qa-schema={schema}
      data-qa-resource-type="table"
      data-qa-resource-id={tableName}
    >
      <div className="flex items-center">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={actions.refresh}
              disabled={state.loading}
              data-testid="sql.table.refresh-button"
              data-qa-module="sql"
              data-qa-object="table-data"
              data-qa-action="refresh"
              data-qa-state={state.loading ? 'loading' : 'ready'}
              data-qa-disabled-reason={state.loading ? 'loading' : undefined}
            >
              <RefreshCw className={cn('h-4 w-4', state.loading && 'animate-spin')} />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('common.actions.refresh')}</TooltipContent>
        </Tooltip>

        <Separator orientation="vertical" className="mx-1 data-[orientation=vertical]:h-4" />

        {state.canEdit && (
          <>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={actions.addPendingRow}
                  data-testid="sql.table.add-row-button"
                  data-qa-module="sql"
                  data-qa-object="table-row"
                  data-qa-action="create"
                  data-qa-risk="resource_mutation"
                >
                  <Plus className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t('sql.actions.addData')}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={actions.markSelectedRowsForDelete}
                    disabled={state.selectedRowKeys.size === 0}
                    data-testid="sql.table.delete-selected-button"
                    data-qa-module="sql"
                    data-qa-object="table-row"
                    data-qa-action="mark-delete"
                    data-qa-risk="resource_mutation"
                    data-qa-disabled-reason={state.selectedRowKeys.size === 0 ? 'no_selection' : undefined}
                  >
                    <Minus className="h-4 w-4" />
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent>{t('sql.changes.deleteSelected')}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={actions.undoLastChange}
                    disabled={state.undoStack.length === 0}
                    data-testid="sql.table.undo-change-button"
                    data-qa-module="sql"
                    data-qa-object="changeset"
                    data-qa-action="undo"
                    data-qa-disabled-reason={state.undoStack.length === 0 ? 'no_pending_undo' : undefined}
                  >
                    <Undo2 className="h-4 w-4" />
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent>{t('sql.changes.undo')}</TooltipContent>
            </Tooltip>

            <Separator orientation="vertical" className="mx-1 data-[orientation=vertical]:h-4" />

            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => actions.setShowPreviewModal(true)}
                    disabled={!state.hasPendingChanges}
                    data-testid="sql.table.preview-changes-button"
                    data-qa-module="sql"
                    data-qa-object="changeset"
                    data-qa-action="preview"
                    data-qa-risk="resource_mutation"
                    data-qa-disabled-reason={!state.hasPendingChanges ? 'no_pending_changes' : undefined}
                  >
                    <Eye className="h-4 w-4" />
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent>{t('sql.actions.previewChanges')}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => actions.setShowSubmitModal(true)}
                    disabled={!state.hasPendingChanges}
                    data-testid="sql.table.submit-changes-button"
                    data-qa-module="sql"
                    data-qa-object="changeset"
                    data-qa-action="submit"
                    data-qa-risk="resource_mutation"
                    data-qa-disabled-reason={!state.hasPendingChanges ? 'no_pending_changes' : undefined}
                  >
                    <SendHorizontal className="h-4 w-4" />
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent>{t('sql.actions.submitChanges')}</TooltipContent>
            </Tooltip>
          </>
        )}

        <Separator orientation="vertical" className="mx-1 data-[orientation=vertical]:h-4" />

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setIsChartModalOpen(true)}
              data-testid="sql.table.create-chart-button"
              data-qa-module="sql"
              data-qa-object="table-data"
              data-qa-action="create-chart"
            >
              <BarChart3 className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('analysis.chart.create')}</TooltipContent>
        </Tooltip>
      </div>
      <div className="flex items-center gap-2">
        <DataView.FilterButton
          onClick={() => actions.setIsFilterModalOpen(true)}
          count={state.filterConditions.length}
        />
        <Button
          className="rounded-lg gap-2.5 min-w-[86px]"
          onClick={() => actions.setShowExportModal(true)}
          data-testid="sql.table.export-button"
          data-qa-module="sql"
          data-qa-object="table-data"
          data-qa-action="export"
        >
          <Download className="h-4 w-4" />
          {t('common.actions.export')}
        </Button>
        <Button
          className="rounded-lg gap-2.5 min-w-[86px]"
          onClick={handleOpenQuery}
          data-testid="sql.table.open-query-button"
          data-qa-module="sql"
          data-qa-object="table-data"
          data-qa-action="open-query"
        >
          <TerminalSquare className="h-4 w-4" />
          {t('common.actions.query')}
        </Button>
      </div>
      <ChartCreateModal
        open={isChartModalOpen}
        onOpenChange={setIsChartModalOpen}
        initialData={state.data ? {
          connectionId,
          databaseName,
          schemaName: schema,
          query: `SELECT * FROM ${buildStorageUnitReference(connections.find(c => c.id === connectionId)?.type, tableName, schema)};`,
          columns: state.data.columns,
          rows: state.data.rows,
        } : undefined}
      />
    </div>
  )
}
