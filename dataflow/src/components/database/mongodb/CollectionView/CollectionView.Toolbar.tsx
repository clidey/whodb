import { useState } from 'react'
import { Download, Plus, Minus, Undo2, Eye, SendHorizontal, RefreshCw, TerminalSquare, BarChart3, Table2, FileJson } from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import { useCollectionView } from './CollectionViewProvider'
import type { MongoCollectionViewMode } from './types'
import { DataView } from '@/components/database/shared/DataView'
import { Button } from '@/components/ui/Button'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { useI18n } from '@/i18n/useI18n'
import { useTabStore } from '@/stores/useTabStore'
import { buildMongoCollectionCommand } from '@/utils/mongodb-shell'
import { ChartCreateModal } from '@/components/analysis/chart-create'

interface CollectionViewToolbarProps {
  connectionId: string
  databaseName: string
  collectionName: string
}

/** Toolbar actions for refreshing and mutating MongoDB collection documents. */
export function CollectionViewToolbar({ connectionId, databaseName, collectionName }: CollectionViewToolbarProps) {
  const { t } = useI18n()
  const { state, actions } = useCollectionView()
  const openTab = useTabStore((s) => s.openTab)
  const [isChartModalOpen, setIsChartModalOpen] = useState(false)

  const handleOpenQuery = () => {
    openTab({
      type: 'query',
      title: t('sidebar.tab.queryWithDatabase', { database: databaseName }),
      connectionId,
      databaseName,
      sqlContent: `${buildMongoCollectionCommand(collectionName, 'find', '{}')};`,
    })
  }

  return (
    <div
      className="flex items-center justify-between h-12 px-2"
      data-testid="mongodb.collection.toolbar"
      data-qa-module="mongodb"
      data-qa-object="collection-toolbar"
      data-qa-state={state.loading ? 'loading' : state.hasPendingChanges ? 'dirty' : 'ready'}
      data-qa-connection-id={connectionId}
      data-qa-database={databaseName}
      data-qa-resource-type="collection"
      data-qa-resource-id={collectionName}
    >
      <div className="flex items-center">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={actions.refresh}
              disabled={state.loading}
              data-testid="mongodb.collection.refresh-button"
              data-qa-module="mongodb"
              data-qa-object="collection-data"
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

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={actions.handleAddClick}
              data-testid="mongodb.collection.add-document-button"
              data-qa-module="mongodb"
              data-qa-object="document"
              data-qa-action="create"
              data-qa-risk="resource_mutation"
            >
              <Plus className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('mongodb.collection.addData')}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Button
                variant="ghost"
                size="icon"
                onClick={actions.markSelectedForDelete}
                disabled={state.selectedRowKeys.size === 0}
                data-testid="mongodb.collection.delete-selected-button"
                data-qa-module="mongodb"
                data-qa-object="document"
                data-qa-action="mark-delete"
                data-qa-risk="resource_mutation"
                data-qa-disabled-reason={state.selectedRowKeys.size === 0 ? 'no_selection' : undefined}
              >
                <Minus className="h-4 w-4" />
              </Button>
            </span>
          </TooltipTrigger>
          <TooltipContent>{t('mongodb.actions.markDelete')}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Button
                variant="ghost"
                size="icon"
                onClick={actions.undoLastChange}
                disabled={state.undoStack.length === 0}
                data-testid="mongodb.collection.undo-change-button"
                data-qa-module="mongodb"
                data-qa-object="changeset"
                data-qa-action="undo"
                data-qa-disabled-reason={state.undoStack.length === 0 ? 'no_pending_undo' : undefined}
              >
                <Undo2 className="h-4 w-4" />
              </Button>
            </span>
          </TooltipTrigger>
          <TooltipContent>{t('mongodb.actions.undo')}</TooltipContent>
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
                data-testid="mongodb.collection.preview-changes-button"
                data-qa-module="mongodb"
                data-qa-object="changeset"
                data-qa-action="preview"
                data-qa-risk="resource_mutation"
                data-qa-disabled-reason={!state.hasPendingChanges ? 'no_pending_changes' : undefined}
              >
                <Eye className="h-4 w-4" />
              </Button>
            </span>
          </TooltipTrigger>
          <TooltipContent>{t('mongodb.actions.previewChanges')}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => actions.setShowSubmitModal(true)}
                disabled={!state.hasPendingChanges}
                data-testid="mongodb.collection.submit-changes-button"
                data-qa-module="mongodb"
                data-qa-object="changeset"
                data-qa-action="submit"
                data-qa-risk="resource_mutation"
                data-qa-disabled-reason={!state.hasPendingChanges ? 'no_pending_changes' : undefined}
              >
                <SendHorizontal className="h-4 w-4" />
              </Button>
            </span>
          </TooltipTrigger>
          <TooltipContent>{t('mongodb.actions.submitChanges')}</TooltipContent>
        </Tooltip>

        <Separator orientation="vertical" className="mx-1 data-[orientation=vertical]:h-4" />

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setIsChartModalOpen(true)}
              data-testid="mongodb.collection.create-chart-button"
              data-qa-module="mongodb"
              data-qa-object="collection-data"
              data-qa-action="create-chart"
            >
              <BarChart3 className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('analysis.chart.create')}</TooltipContent>
        </Tooltip>

        <Separator orientation="vertical" className="mx-1 data-[orientation=vertical]:h-4" />

        <CollectionViewModeSwitch
          currentMode={state.viewMode}
          onSelect={actions.setViewMode}
        />
      </div>
      <div className="flex items-center gap-2">
        <DataView.FilterButton
          onClick={() => actions.setIsFilterModalOpen(true)}
          count={Object.keys(state.activeFilter).length}
        />
        <Button
          className="rounded-lg gap-2.5 min-w-[86px]"
          onClick={() => actions.setShowExportModal(true)}
          data-testid="mongodb.collection.export-button"
          data-qa-module="mongodb"
          data-qa-object="collection-data"
          data-qa-action="export"
        >
          <Download className="h-4 w-4" />
          {t('common.actions.export')}
        </Button>
        <Button
          className="rounded-lg gap-2.5 min-w-[86px]"
          onClick={handleOpenQuery}
          data-testid="mongodb.collection.open-query-button"
          data-qa-module="mongodb"
          data-qa-object="collection-data"
          data-qa-action="open-query"
        >
          <TerminalSquare className="h-4 w-4" />
          {t('common.actions.query')}
        </Button>
      </div>
      <ChartCreateModal
        open={isChartModalOpen}
        onOpenChange={setIsChartModalOpen}
        initialData={state.documents.length > 0 ? {
          connectionId,
          databaseName,
          query: `${buildMongoCollectionCommand(collectionName, 'find', '{}')};`,
          columns: state.availableFields,
          rows: state.documents,
        } : undefined}
      />
    </div>
  )
}

interface CollectionViewModeSwitchProps {
  currentMode: MongoCollectionViewMode
  onSelect: (mode: MongoCollectionViewMode) => void
}

/** Two-option MongoDB collection view-mode switch. */
function CollectionViewModeSwitch({ currentMode, onSelect }: CollectionViewModeSwitchProps) {
  const { t } = useI18n()

  return (
    <div
      className="relative inline-grid h-9 w-[72px] grid-cols-2 items-center overflow-hidden rounded-md border border-accent bg-background"
      role="group"
      aria-label={t('mongodb.view.selectorLabel')}
      data-testid="mongodb.collection.view-mode-buttons"
      data-qa-module="mongodb"
      data-qa-object="collection-view-mode"
      data-qa-state={currentMode}
    >
      <span
        aria-hidden="true"
        style={{ transform: currentMode === 'json' ? 'translateX(36px)' : 'translateX(0)' }}
        className={cn(
          'pointer-events-none absolute -left-px -top-px h-[calc(100%+2px)] w-[37px] rounded-l-[inherit] bg-accent transition-[transform,border-radius] duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] motion-reduce:transition-none',
          currentMode === 'json' && 'rounded-l-none rounded-r-[inherit]',
        )}
      />
      <CollectionViewModeButton
        mode="table"
        currentMode={currentMode}
        ariaLabel={t('mongodb.view.table')}
        tooltip={currentMode === 'table' ? t('mongodb.view.table') : t('mongodb.view.switchToTable')}
        onSelect={onSelect}
      />
      <CollectionViewModeButton
        mode="json"
        currentMode={currentMode}
        ariaLabel={t('mongodb.view.json')}
        tooltip={currentMode === 'json' ? t('mongodb.view.json') : t('mongodb.view.switchToJson')}
        onSelect={onSelect}
      />
    </div>
  )
}

interface CollectionViewModeButtonProps {
  mode: MongoCollectionViewMode
  currentMode: MongoCollectionViewMode
  ariaLabel: string
  tooltip: string
  onSelect: (mode: MongoCollectionViewMode) => void
}

/** Single option in the MongoDB collection view-mode control. */
function CollectionViewModeButton({ mode, currentMode, ariaLabel, tooltip, onSelect }: CollectionViewModeButtonProps) {
  const active = currentMode === mode
  const Icon = mode === 'table' ? Table2 : FileJson

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={ariaLabel}
          aria-pressed={active}
          onClick={() => onSelect(mode)}
          data-testid={`mongodb.collection.view-${mode}-button`}
          data-qa-module="mongodb"
          data-qa-object="collection-view-mode"
          data-qa-action={`switch-to-${mode}`}
          data-qa-state={active ? 'active' : 'inactive'}
          className={cn(
            'relative z-10 size-9 rounded-md text-muted-foreground transition-colors hover:bg-transparent hover:text-foreground',
            active && 'text-foreground hover:text-foreground',
          )}
        >
          <Icon className="h-4 w-4" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  )
}
