import type { Alert } from '@/components/database/shared/types'
import type { FlatMongoFilter } from '@/components/database/mongodb/filter-collection.types'

// ---- Document changeset types ----

export type DocumentChangesetRowKey = string

export interface DocumentChange {
  type: 'update' | 'insert' | 'delete'
  originalDocument: Record<string, unknown>
  document: Record<string, unknown>
}

export interface DocumentUndoEntryEdit {
  kind: 'edit'
  rowKey: DocumentChangesetRowKey
  previousDocument: Record<string, unknown>
}

export interface DocumentUndoEntryAdd {
  kind: 'add'
  rowKey: DocumentChangesetRowKey
}

export interface DocumentUndoEntryDelete {
  kind: 'delete'
  rowKeys: DocumentChangesetRowKey[]
  previousChanges: Array<[DocumentChangesetRowKey, DocumentChange | undefined]>
}

export type DocumentUndoEntry = DocumentUndoEntryEdit | DocumentUndoEntryAdd | DocumentUndoEntryDelete

/** Context value exposed by CollectionViewProvider. */
export interface CollectionViewContextValue {
  state: CollectionViewState
  actions: CollectionViewActions
}

/** All state managed by the CollectionView provider. */
export interface CollectionViewState {
  loading: boolean
  documents: any[]
  error: string | null
  currentPage: number
  pageSize: number
  total: number
  totalPages: number
  activeFilter: FlatMongoFilter
  availableFields: string[]
  showExportModal: boolean
  isFilterModalOpen: boolean
  alert: Alert | null

  // Changeset state
  changes: Map<DocumentChangesetRowKey, DocumentChange>
  undoStack: DocumentUndoEntry[]
  selectedRowKeys: Set<DocumentChangesetRowKey>
  newRowOrder: DocumentChangesetRowKey[]
  pendingChangeCount: number
  hasPendingChanges: boolean
  showPreviewModal: boolean
  showSubmitModal: boolean
  showDiscardModal: boolean

  // Document editing (modal-based add/edit)
  showAddModal: boolean
  addContent: string
  editingRowKey: DocumentChangesetRowKey | null
  editContent: string
}

/** All actions exposed by the CollectionView provider. */
export interface CollectionViewActions {
  refresh: () => void
  handlePageChange: (page: number) => void
  handlePageSizeChange: (size: number) => void
  setIsFilterModalOpen: (open: boolean) => void
  handleFilterApply: (filter: FlatMongoFilter) => void
  setShowExportModal: (open: boolean) => void
  showAlert: (title: string, message: string, type: Alert['type']) => void
  closeAlert: () => void

  // Changeset actions
  toggleRowSelection: (rowKey: DocumentChangesetRowKey) => void
  markSelectedForDelete: () => void
  undoLastChange: () => void
  discardChanges: () => void
  submitChanges: () => Promise<void>
  setShowPreviewModal: (open: boolean) => void
  setShowSubmitModal: (open: boolean) => void
  setShowDiscardModal: (open: boolean) => void
  confirmDiscardAndContinue: () => void

  // Document editing (modal-based add/edit)
  handleAddClick: () => void
  setAddContent: (content: string) => void
  handleAddSave: () => Promise<void>
  setShowAddModal: (open: boolean) => void
  handleEditClick: (rowKey: DocumentChangesetRowKey) => void
  setEditContent: (content: string) => void
  handleEditSave: () => Promise<void>
  cancelEdit: () => void
}
