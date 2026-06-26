import { TableViewProvider, useTableView } from './TableView/TableViewProvider'
import { TableViewDataGrid } from './TableView/TableView.DataGrid'
import { TableViewToolbar } from './TableView/TableView.Toolbar'
import { buildPreviewSql, summarizeChanges } from './TableView/changeset-sql-preview'
import { DataView } from '@/components/database/shared/DataView'
import { FindBar } from '@/components/database/shared/FindBar'
import { FilterTableModal } from './FilterTableModal'
import { ExportDataModal } from './ExportDataModal'
import { DatabaseImportModal } from '@/components/database/import/DatabaseImportModal'
import { ConfirmationModal } from '@/components/ui/ConfirmationModal'
import { AlertModal } from '@/components/ui/AlertModal'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useI18n } from '@/i18n/useI18n'
import { useConnectionStore } from '@/stores/useConnectionStore'

interface TableDetailViewProps {
  tabId: string
  connectionId: string
  databaseName: string
  tableName: string
  schema?: string
}

export function TableDetailView(props: TableDetailViewProps) {
  return (
    <TableViewProvider {...props}>
      <TableDetailViewContent {...props} />
    </TableViewProvider>
  )
}

function TableDetailViewContent({ connectionId, databaseName, tableName, schema }: TableDetailViewProps) {
  const { t } = useI18n()
  const { state, actions } = useTableView()

  const previewStatements = buildPreviewSql(tableName, state.changes)
  const summary = summarizeChanges(state.changes)

  return (
    <div
      className="flex h-full flex-col"
      data-testid="sql.table.detail"
      data-qa-module="sql"
      data-qa-object="table-detail"
      data-qa-state={state.error ? "error" : state.loading ? "loading" : "ready"}
      data-qa-loading={state.loading ? "true" : "false"}
      data-qa-connection-id={connectionId}
      data-qa-database={databaseName}
      data-qa-schema={schema}
      data-qa-resource-type="table"
      data-qa-resource-id={tableName}
    >
      <TableViewToolbar connectionId={connectionId} databaseName={databaseName} tableName={tableName} schema={schema} />

      {state.error ? (
        <DataView.Error message={state.error} onRetry={() => actions.handleSubmitRequest()} />
      ) : (
        <FindBar.Provider
          rows={state.renderedRows.map((row) => row.values)}
          columns={state.visibleColumns}
        >
          <FindBar.Bar />
          <TableViewDataGrid />
        </FindBar.Provider>
      )}

      {state.total > 0 && (
        <DataView.Pagination
          currentPage={state.currentPage}
          totalPages={state.totalPages}
          pageSize={state.pageSize}
          total={state.total}
          loading={state.loading}
          onPageChange={actions.handlePageChange}
          onPageSizeChange={actions.handlePageSizeChange}
        />
      )}

      <FilterTableModal
        open={state.isFilterModalOpen}
        onOpenChange={actions.setIsFilterModalOpen}
        columns={state.data?.columns || []}
        initialSelectedColumns={state.visibleColumns}
        initialConditions={state.filterConditions}
        onApply={actions.handleFilterApply}
      />

      {state.showExportModal && (
        <ExportDataModal
          open={state.showExportModal}
          onOpenChange={(open) => { if (!open) actions.setShowExportModal(false) }}
          connectionId={connectionId}
          databaseName={databaseName}
          schema={schema}
          tableName={tableName}
        />
      )}

      {state.showImportModal && (
        <DatabaseImportModal
          open={state.showImportModal}
          onOpenChange={(open) => { if (!open) actions.setShowImportModal(false) }}
          connectionId={connectionId}
          databaseName={databaseName}
          schema={schema}
          tableName={tableName}
          onSuccess={() => {
            useConnectionStore.getState().triggerSidebarRefresh()
            actions.refresh()
          }}
        />
      )}

      <Dialog open={state.showPreviewModal} onOpenChange={actions.setShowPreviewModal}>
        <DialogContent
          className="sm:max-w-3xl"
          data-testid="sql.table.changes-preview-dialog"
          data-qa-module="sql"
          data-qa-object="changes-preview"
          data-qa-state="open"
          data-qa-risk="resource_mutation"
          data-qa-resource-type="table"
          data-qa-resource-id={tableName}
        >
          <DialogHeader>
            <DialogTitle>{t('sql.changes.previewTitle')}</DialogTitle>
            <DialogDescription>
              {t('sql.changes.previewDescription', { count: state.pendingChangeCount })}
            </DialogDescription>
          </DialogHeader>

          <ScrollArea className="max-h-[60vh] rounded-md border bg-muted/20">
            <pre
              className="whitespace-pre-wrap p-4 font-mono text-xs"
              data-testid="sql.table.changes-preview-sql"
              data-qa-module="sql"
              data-qa-object="changes-preview"
              data-qa-state={state.pendingChangeCount > 0 ? "ready" : "empty"}
            >
              {previewStatements.join('\n\n')}
            </pre>
          </ScrollArea>
        </DialogContent>
      </Dialog>

      <ConfirmationModal
        isOpen={state.showSubmitModal}
        onClose={() => actions.setShowSubmitModal(false)}
        onConfirm={actions.submitChanges}
        title={t('sql.changes.submitConfirmTitle', { count: state.pendingChangeCount })}
        message={t('sql.changes.submitConfirmMessage', {
          count: state.pendingChangeCount,
          updates: summary.updates,
          inserts: summary.inserts,
          deletes: summary.deletes,
        })}
        confirmText={t('common.actions.confirm')}
      />

      <ConfirmationModal
        isOpen={state.showDiscardModal}
        onClose={() => actions.setShowDiscardModal(false)}
        onConfirm={actions.confirmDiscardAndContinue}
        title={t('sql.changes.discardTitle')}
        message={t('sql.changes.discardMessage', { count: state.pendingChangeCount })}
        confirmText={t('common.actions.discard')}
      />

      {state.alert && (
        <AlertModal
          isOpen
          onClose={actions.closeAlert}
          {...state.alert}
        />
      )}
    </div>
  )
}
