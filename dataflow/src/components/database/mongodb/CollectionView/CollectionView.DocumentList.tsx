import { use, useMemo } from 'react'
import { FileJson, Edit2 } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { useCollectionView } from './CollectionViewProvider'
import { useI18n } from '@/i18n/useI18n'
import { FindBarContext } from '@/components/database/shared/FindBar.Provider'
import { buildRenderedMongoDocuments } from './mongo-table-utils'
import { stringifyMongoDocument } from '@/utils/mongodb-shell'

/** List of MongoDB document cards with selection checkboxes and change indicators. */
export function CollectionViewDocumentList() {
  const { t } = useI18n()
  const { state, actions } = useCollectionView()
  const findBar = use(FindBarContext)

  const pageOffset = (state.currentPage - 1) * state.pageSize

  const renderedDocs = useMemo(() => buildRenderedMongoDocuments({
    documents: state.documents as Record<string, unknown>[],
    changes: state.changes,
    newRowOrder: state.newRowOrder,
    documentFieldOrders: state.documentFieldOrders,
    pageOffset,
  }), [pageOffset, state.changes, state.documentFieldOrders, state.documents, state.newRowOrder])

  if (state.documents.length === 0 && state.newRowOrder.length === 0) {
    return (
      <div
        className="text-center py-12"
        data-testid="mongodb.collection.document-list-empty"
        data-qa-module="mongodb"
        data-qa-object="document-list"
        data-qa-state="empty"
      >
        <FileJson className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <p className="text-sm text-muted-foreground">{t('mongodb.collection.noDocuments')}</p>
      </div>
    )
  }

  return (
    <>
      {renderedDocs.map((item, idx) => {
        // FindBar matching only applies to existing documents (not pending inserts)
        const existingIdx = state.documents.indexOf(item.doc)
        const findBarIdx = existingIdx >= 0 ? existingIdx : -1
        const hasMatch = findBarIdx >= 0 && findBar?.state.total
          ? findBar.state.matches.some((m) => m.rowIndex === findBarIdx)
          : false
        const hasCurrentMatch = findBarIdx >= 0 && findBar?.state.total
          ? findBar.state.matches[findBar.state.currentMatchIndex]?.rowIndex === findBarIdx
          : false

        const isSelected = state.selectedRowKeys.has(item.rowKey)

        return (
          <div
            key={item.rowKey}
            data-testid="mongodb.collection.document-card"
            data-qa-module="mongodb"
            data-qa-object="document"
            data-qa-state={item.changeType ?? (isSelected ? 'selected' : 'ready')}
            data-qa-resource-type="document"
            data-qa-resource-id={item.rowKey}
            data-find-current={hasCurrentMatch ? 'true' : undefined}
            className={cn(
              'rounded-xl p-4 group relative transition-colors duration-200 cursor-pointer',
              // Change type styling
              item.changeType === 'insert' && 'bg-blue-50 border border-blue-200',
              item.changeType === 'delete' && 'bg-red-50/60 border border-red-200 opacity-60',
              item.changeType === 'update' && 'bg-green-50/60 border border-green-200',
              // FindBar match styling (only when no change type)
              !item.changeType && hasCurrentMatch && 'bg-blue-100 border border-blue-300 shadow-sm',
              !item.changeType && !hasCurrentMatch && hasMatch && 'bg-blue-50/60 border border-blue-200',
              // Default styling
              !item.changeType && !hasMatch && 'bg-background border border-border/50 hover:bg-muted/30 hover:shadow-sm',
              // Selection: outline sits outside border, no layout shift
              isSelected && 'outline-2 outline-primary',
            )}
            onClick={() => actions.toggleRowSelection(item.rowKey)}
            onDoubleClick={() => { if (!item.isDeleted) actions.handleEditClick(item.rowKey) }}
          >
            <div className="relative">
              {!item.isDeleted && (
                <div className="absolute top-0 right-0 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        onClick={(e) => { e.stopPropagation(); actions.handleEditClick(item.rowKey) }}
                        variant="ghost"
                        size="icon"
                        data-testid="mongodb.collection.edit-document-button"
                        data-qa-module="mongodb"
                        data-qa-object="document"
                        data-qa-action="edit"
                        data-qa-resource-type="document"
                        data-qa-resource-id={item.rowKey}
                        className="h-8 w-8 text-muted-foreground hover:text-primary"
                      >
                        <Edit2 className="h-4 w-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t('mongodb.document.editAction')}</TooltipContent>
                  </Tooltip>
                </div>
              )}
              <pre className={cn(
                'text-sm overflow-x-auto font-mono text-foreground/80',
                item.isDeleted && 'line-through',
              )}>
                {stringifyMongoDocument(item.doc, item.fieldOrder, 2).replace(/^\{\n/, '').replace(/\n\}$/, '')}
              </pre>
            </div>
          </div>
        )
      })}
    </>
  )
}
