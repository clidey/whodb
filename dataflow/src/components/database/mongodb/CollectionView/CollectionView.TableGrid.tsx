import { use, useCallback, useMemo, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { FileJson } from 'lucide-react'
import { useI18n } from '@/i18n/useI18n'
import { cn } from '@/lib/utils'
import { FindBarContext } from '@/components/database/shared/FindBar.Provider'
import { useCollectionView } from './CollectionViewProvider'
import { CollectionViewColumnHeader } from './CollectionView.ColumnHeader'
import {
  buildRenderedMongoDocuments,
  coerceMongoCellDraft,
  hasDocumentField,
  isMongoCellChanged,
  isMongoScalarValue,
  parseMongoFieldJsonDraft,
} from './mongo-table-utils'
import { FieldJsonEditorDialog } from './CollectionView.FieldJsonEditorDialog'

interface ActiveMongoCell {
  rowKey: string
  column: string
  draftValue: string
}

interface ActiveMongoFieldJsonEditor {
  rowKey: string
  column: string
  content: string
}

/** Renders the MongoDB collection table view with scalar inline editing. */
export function CollectionViewTableGrid() {
  const { t } = useI18n()
  const { state, actions } = useCollectionView()
  const findBar = use(FindBarContext)
  const [activeCell, setActiveCell] = useState<ActiveMongoCell | null>(null)
  const [fieldJsonEditor, setFieldJsonEditor] = useState<ActiveMongoFieldJsonEditor | null>(null)

  const pageOffset = (state.currentPage - 1) * state.pageSize
  const renderedDocs = useMemo(() => buildRenderedMongoDocuments({
    documents: state.documents as Record<string, unknown>[],
    changes: state.changes,
    newRowOrder: state.newRowOrder,
    documentFieldOrders: state.documentFieldOrders,
    pageOffset,
  }), [pageOffset, state.changes, state.documentFieldOrders, state.documents, state.newRowOrder])

  const openCellEditor = useCallback((rowKey: string, column: string, value: unknown, fieldExists: boolean) => {
    if (column === '_id') return
    if (fieldExists && !isMongoScalarValue(value)) {
      setActiveCell(null)
      setFieldJsonEditor({
        rowKey,
        column,
        content: JSON.stringify(value, null, 2),
      })
      return
    }

    setActiveCell({
      rowKey,
      column,
      draftValue: fieldExists && value !== null ? String(value) : '',
    })
  }, [])

  const commitCellEdit = useCallback(() => {
    if (!activeCell) return

    const row = renderedDocs.find((item) => item.rowKey === activeCell.rowKey)
    if (!row) return

    const fieldExists = hasDocumentField(row.doc, activeCell.column)
    const result = coerceMongoCellDraft(row.doc[activeCell.column], activeCell.draftValue, fieldExists)

    if (!result.ok) {
      const messageKey = result.error === 'invalid-number'
        ? 'mongodb.table.invalidNumber'
        : result.error === 'invalid-boolean'
          ? 'mongodb.table.invalidBoolean'
          : 'mongodb.table.complexInlineEdit'
      actions.showAlert(t('common.alert.error'), t(messageKey), 'error')
      return
    }

    actions.stageDocumentEdit(activeCell.rowKey, {
      ...row.doc,
      [activeCell.column]: result.value,
    })
    setActiveCell(null)
  }, [activeCell, actions, renderedDocs, t])

  const handleCellKeyDown = useCallback((event: ReactKeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Escape') {
      event.preventDefault()
      setActiveCell(null)
      return
    }

    if (event.key === 'Enter') {
      event.preventDefault()
      commitCellEdit()
    }
  }, [commitCellEdit])

  const handleFieldJsonSave = useCallback(async () => {
    if (!fieldJsonEditor) return

    const row = renderedDocs.find((item) => item.rowKey === fieldJsonEditor.rowKey)
    if (!row) return

    const result = parseMongoFieldJsonDraft(fieldJsonEditor.content)
    if (!result.ok) {
      throw new Error(t('mongodb.fieldJson.invalidJson', { error: result.error }))
    }

    actions.stageDocumentEdit(fieldJsonEditor.rowKey, {
      ...row.doc,
      [fieldJsonEditor.column]: result.value,
    })
    setFieldJsonEditor(null)
  }, [actions, fieldJsonEditor, renderedDocs, t])

  if (renderedDocs.length === 0) {
    return (
      <div
        className="flex flex-1 flex-col items-center justify-center py-12 text-center"
        data-testid="mongodb.collection.table-empty"
        data-qa-module="mongodb"
        data-qa-object="collection-table"
        data-qa-state="empty"
      >
        <FileJson className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">{t('mongodb.collection.noDocuments')}</p>
      </div>
    )
  }

  return (
    <>
      <div
        className="flex-1 overflow-auto"
        data-testid="mongodb.collection.table-scroll"
        data-qa-module="mongodb"
        data-qa-object="collection-table"
        data-qa-state="ready"
        data-qa-row-count={renderedDocs.length}
      >
        <table
          className="min-w-full border-collapse text-sm"
          data-testid="mongodb.collection.table"
          data-qa-module="mongodb"
          data-qa-object="collection-table"
          data-qa-state="ready"
        >
          <thead className="border-b border-border bg-background">
            <tr>
              <th
                className="sticky left-0 top-0 z-50 w-16 min-w-16 max-w-16 border-b border-r border-border/50 bg-background px-2 py-2 text-center text-xs font-semibold text-muted-foreground"
              />
              {state.tableColumns.map((column, index) => (
                <CollectionViewColumnHeader key={column} column={column} index={index} />
              ))}
              <th className="sticky top-0 z-40 w-full border-b border-border/50 bg-background" />
            </tr>
          </thead>
          <tbody className="bg-background">
            {renderedDocs.map((row, rowIdx) => {
              const isSelected = state.selectedRowKeys.has(row.rowKey)

              return (
                <tr
                  key={row.rowKey}
                  data-testid="mongodb.collection.table-row"
                  data-qa-module="mongodb"
                  data-qa-object="document-row"
                  data-qa-state={row.isInserted ? 'inserted' : row.isDeleted ? 'deleted' : isSelected ? 'selected' : 'ready'}
                  data-qa-resource-type="document"
                  data-qa-resource-id={row.rowKey}
                  className={cn(
                    'group transition-colors',
                    row.isInserted && 'bg-blue-100/20',
                    row.isDeleted && 'bg-red-100/20',
                    !row.isInserted && !row.isDeleted && 'hover:bg-muted/50',
                  )}
                >
                  <td
                    className={cn(
                      'sticky left-0 z-30 w-16 min-w-16 max-w-16 border-b border-r border-border/50 bg-background px-2 py-2 text-center text-sm font-normal',
                      row.isInserted && 'bg-blue-100/60',
                      row.isDeleted && 'bg-red-100/60 text-muted-foreground line-through',
                      isSelected && 'bg-primary/10',
                    )}
                    onClick={() => actions.toggleRowSelection(row.rowKey)}
                    data-testid="mongodb.collection.table-row-selector"
                    data-qa-module="mongodb"
                    data-qa-object="document-row"
                    data-qa-action="select"
                    data-qa-state={isSelected ? 'selected' : 'ready'}
                    data-qa-resource-type="document"
                    data-qa-resource-id={row.rowKey}
                  >
                    {row.rowNumber ?? ''}
                  </td>

                  {state.tableColumns.map((column) => {
                    const width = state.columnWidths[column] || 160
                    const fieldExists = hasDocumentField(row.doc, column)
                    const value = row.doc[column]
                    const isComplex = fieldExists && !isMongoScalarValue(value)
                    const editable = !row.isDeleted && column !== '_id' && !isComplex
                    const isActive = activeCell?.rowKey === row.rowKey && activeCell.column === column
                    const changed = row.changeType === 'update' && isMongoCellChanged(row.originalDocument, row.doc, column)
                    const highlight = findBar?.state.total
                      ? findBar.state.matches.findIndex((match) => match.rowIndex === rowIdx && match.columnKey === column) === findBar.state.currentMatchIndex
                        ? 'current'
                        : findBar.state.matches.some((match) => match.rowIndex === rowIdx && match.columnKey === column)
                          ? 'match'
                          : null
                      : null
                    const displayValue = !fieldExists
                      ? t('mongodb.table.unset')
                      : value === null
                        ? t('mongodb.table.null')
                        : isComplex
                          ? JSON.stringify(value)
                          : String(value)

                    return (
                      <td
                        key={column}
                        className={cn(
                          'relative overflow-hidden border-b border-r border-border/50 text-sm text-foreground/80',
                          isActive ? 'p-0' : 'px-4 py-2',
                          row.isInserted && 'bg-blue-100/60',
                          row.isDeleted && 'bg-red-100/60 text-muted-foreground line-through',
                          changed && 'bg-green-100/60',
                          isSelected && !row.isInserted && !row.isDeleted && !changed && 'bg-primary/10',
                          highlight === 'current' && 'bg-blue-200',
                          highlight === 'match' && 'bg-blue-100/60',
                          editable && !isActive && 'cursor-default',
                          isComplex && !row.isDeleted && 'cursor-pointer',
                        )}
                        data-testid="mongodb.collection.table-cell"
                        data-qa-module="mongodb"
                        data-qa-object="document-cell"
                        data-qa-field={column}
                        data-qa-state={isActive ? 'editing' : changed ? 'changed' : row.isInserted ? 'inserted' : row.isDeleted ? 'deleted' : editable ? 'editable' : isComplex ? 'complex' : 'read_only'}
                        data-qa-disabled-reason={!editable ? row.isDeleted ? 'row_deleted' : column === '_id' ? 'document_id' : isComplex ? 'complex_value' : undefined : undefined}
                        data-qa-resource-type="document"
                        data-qa-resource-id={row.rowKey}
                        data-find-current={highlight === 'current' ? 'true' : undefined}
                        style={{
                          minWidth: `${width}px`,
                          maxWidth: state.resizedColumns.has(column) ? `${width}px` : '320px',
                        }}
                        onDoubleClick={() => {
                          if (row.isDeleted) return
                          openCellEditor(row.rowKey, column, value, fieldExists)
                        }}
                      >
                        {isActive ? (
                          <input
                            autoFocus
                            type="text"
                            data-testid="mongodb.collection.table-cell-editor"
                            data-qa-module="mongodb"
                            data-qa-object="document-cell"
                            data-qa-action="edit"
                            data-qa-field={column}
                            data-qa-state="editing"
                            data-qa-resource-type="document"
                            data-qa-resource-id={row.rowKey}
                            value={activeCell.draftValue}
                            onChange={(event) => setActiveCell((current) => current ? { ...current, draftValue: event.target.value } : current)}
                            onBlur={commitCellEdit}
                            onKeyDown={handleCellKeyDown}
                            className="min-h-9 w-full bg-transparent px-4 py-2 text-sm focus:bg-background focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary"
                          />
                        ) : (
                          <span
                            className={cn(
                              'block truncate',
                              (!fieldExists || value === null) && 'italic text-muted-foreground',
                              isComplex && 'font-mono text-xs',
                            )}
                            title={displayValue}
                          >
                            {displayValue}
                          </span>
                        )}
                        <div
                          data-resize-col={column}
                          className={cn(
                            'absolute right-0 top-0 -bottom-px z-20 w-1 cursor-col-resize data-[resize-active]:bg-primary/50',
                            state.resizingColumn === column && 'bg-primary/50',
                          )}
                          onMouseEnter={() => {
                            if (state.resizingColumn) return
                            document.querySelectorAll<HTMLElement>(`[data-resize-col="${column}"]`).forEach(element => { element.dataset.resizeActive = '' })
                          }}
                          onMouseLeave={() => {
                            if (state.resizingColumn) return
                            document.querySelectorAll<HTMLElement>(`[data-resize-col="${column}"]`).forEach(element => { delete element.dataset.resizeActive })
                          }}
                          onMouseDown={(event) => actions.handleResizeStart(event, column)}
                        />
                      </td>
                    )
                  })}

                  <td className="w-full border-b border-border/50 bg-background" />
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {fieldJsonEditor && (
        <FieldJsonEditorDialog
          open
          onOpenChange={(open) => {
            if (!open) setFieldJsonEditor(null)
          }}
          fieldName={fieldJsonEditor.column}
          content={fieldJsonEditor.content}
          onContentChange={(content) => setFieldJsonEditor((current) => current ? { ...current, content } : current)}
          onSave={handleFieldJsonSave}
        />
      )}
    </>
  )
}
