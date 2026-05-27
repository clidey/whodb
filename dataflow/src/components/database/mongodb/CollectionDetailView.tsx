import { useMemo } from 'react'
import { CollectionViewProvider, useCollectionView } from './CollectionView/CollectionViewProvider'
import { CollectionViewDocumentList } from './CollectionView/CollectionView.DocumentList'
import { CollectionViewToolbar } from './CollectionView/CollectionView.Toolbar'
import { AddDocumentModal } from './CollectionView/CollectionView.AddDocumentModal'
import { EditDocumentModal } from './CollectionView/CollectionView.EditDocumentModal'
import { buildPreviewCommands, summarizeChanges } from './CollectionView/changeset-mongo-preview'
import { DataView } from '@/components/database/shared/DataView'
import { FindBar } from '@/components/database/shared/FindBar'
import { ExportCollectionModal } from './ExportCollectionModal'
import { FilterCollectionModal } from './FilterCollectionModal'
import { ConfirmationModal } from '@/components/ui/ConfirmationModal'
import { AlertModal } from '@/components/ui/AlertModal'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useI18n } from '@/i18n/useI18n'

interface CollectionDetailViewProps {
  connectionId: string
  databaseName: string
  collectionName: string
}

/** MongoDB collection detail view composed from Provider + subcomponents. */
export function CollectionDetailView(props: CollectionDetailViewProps) {
  return (
    <CollectionViewProvider {...props}>
      <CollectionDetailViewContent {...props} />
    </CollectionViewProvider>
  )
}

/** Inner content rendered within the CollectionViewProvider context. */
function CollectionDetailViewContent({ databaseName, collectionName, connectionId }: CollectionDetailViewProps) {
  const { t } = useI18n()
  const { state, actions } = useCollectionView()

  const previewCommands = buildPreviewCommands(collectionName, state.changes)
  const summary = summarizeChanges(state.changes)

  if (state.loading && !state.documents.length && !state.showAddModal) {
    return (
      <div
        className="flex h-full"
        data-testid="mongodb.collection.detail-loading"
        data-qa-module="mongodb"
        data-qa-object="collection-detail"
        data-qa-state="loading"
        data-qa-loading="true"
        data-qa-connection-id={connectionId}
        data-qa-database={databaseName}
        data-qa-resource-type="collection"
        data-qa-resource-id={collectionName}
      >
        <DataView.Loading />
      </div>
    )
  }

  /** Extract all top-level field names from visible documents for FindBar. */
  const docColumns = useMemo(() => {
    const keys = new Set<string>()
    state.documents.forEach((doc) => {
      if (typeof doc === 'object' && doc !== null) {
        Object.keys(doc).forEach((k) => keys.add(k))
      }
    })
    return Array.from(keys)
  }, [state.documents])

  return (
    <div
      className="flex flex-col h-full bg-background"
      data-testid="mongodb.collection.detail"
      data-qa-module="mongodb"
      data-qa-object="collection-detail"
      data-qa-state={state.error ? 'error' : state.loading ? 'loading' : 'ready'}
      data-qa-loading={state.loading ? 'true' : 'false'}
      data-qa-connection-id={connectionId}
      data-qa-database={databaseName}
      data-qa-resource-type="collection"
      data-qa-resource-id={collectionName}
    >
      <CollectionViewToolbar connectionId={connectionId} databaseName={databaseName} collectionName={collectionName} />

      {state.error ? (
        <DataView.Error message={state.error} />
      ) : (
        <FindBar.Provider
          rows={state.documents}
          columns={docColumns}
          searchTerm={state.searchTerm}
          onSearchTermChange={actions.setSearchTerm}
        >
          <FindBar.Bar />
          <div
            className="flex-1 overflow-auto p-4 space-y-4"
            data-testid="mongodb.collection.document-list-region"
            data-qa-module="mongodb"
            data-qa-object="document-list"
            data-qa-state={state.documents.length > 0 ? 'ready' : 'empty'}
          >
            <CollectionViewDocumentList />
          </div>
        </FindBar.Provider>
      )}

      {state.total > 0 && (
        <DataView.Pagination
          currentPage={state.currentPage}
          totalPages={state.totalPages}
          pageSize={state.pageSize}
          total={state.total}
          loading={state.loading}
          itemLabel={t('mongodb.collection.documents')}
          onPageChange={actions.handlePageChange}
          onPageSizeChange={actions.handlePageSizeChange}
        />
      )}

      <AddDocumentModal
        open={state.showAddModal}
        onOpenChange={actions.setShowAddModal}
        content={state.addContent}
        onContentChange={actions.setAddContent}
        onSave={actions.handleAddSave}
      />

      <EditDocumentModal
        open={state.editingRowKey !== null}
        onOpenChange={(open) => {
          if (!open) actions.cancelEdit()
        }}
        content={state.editContent}
        onContentChange={actions.setEditContent}
        onSave={actions.handleEditSave}
      />

      <ExportCollectionModal
        open={state.showExportModal}
        onOpenChange={(open) => {
          if (!open) actions.setShowExportModal(false)
        }}
        connectionId={connectionId}
        databaseName={databaseName}
        collectionName={collectionName}
      />

      <FilterCollectionModal
        open={state.isFilterModalOpen}
        onOpenChange={actions.setIsFilterModalOpen}
        onApply={actions.handleFilterApply}
        fields={state.availableFields}
        initialFilter={state.activeFilter}
      />

      <Dialog open={state.showPreviewModal} onOpenChange={actions.setShowPreviewModal}>
        <DialogContent
          className="sm:max-w-3xl"
          data-testid="mongodb.collection.changes-preview-dialog"
          data-qa-module="mongodb"
          data-qa-object="changes-preview"
          data-qa-state="open"
          data-qa-risk="resource_mutation"
          data-qa-resource-type="collection"
          data-qa-resource-id={collectionName}
        >
          <DialogHeader>
            <DialogTitle>{t('mongodb.changes.previewTitle')}</DialogTitle>
            <DialogDescription>
              {t('mongodb.changes.previewDescription', { count: state.pendingChangeCount })}
            </DialogDescription>
          </DialogHeader>
          <ScrollArea className="max-h-[60vh] rounded-md border bg-muted/20">
            <pre
              className="whitespace-pre-wrap p-4 font-mono text-xs"
              data-testid="mongodb.collection.changes-preview-command"
              data-qa-module="mongodb"
              data-qa-object="changes-preview"
              data-qa-state={state.pendingChangeCount > 0 ? 'ready' : 'empty'}
            >
              {previewCommands.join('\n\n')}
            </pre>
          </ScrollArea>
        </DialogContent>
      </Dialog>

      <ConfirmationModal
        isOpen={state.showSubmitModal}
        onClose={() => actions.setShowSubmitModal(false)}
        onConfirm={actions.submitChanges}
        title={t('mongodb.changes.submitConfirmTitle', { count: state.pendingChangeCount })}
        message={t('mongodb.changes.submitConfirmMessage', {
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
        title={t('mongodb.changes.discardTitle')}
        message={t('mongodb.changes.discardMessage', { count: state.pendingChangeCount })}
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
