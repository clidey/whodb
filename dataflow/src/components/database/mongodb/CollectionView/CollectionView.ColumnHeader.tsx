import {
  ArrowDownAZ,
  ArrowUpAZ,
  ListFilter,
  MoreHorizontal,
  X,
} from 'lucide-react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useI18n } from '@/i18n/useI18n'
import { cn } from '@/lib/utils'
import { useCollectionView } from './CollectionViewProvider'

interface CollectionViewColumnHeaderProps {
  column: string
  index: number
}

/** Renders a MongoDB collection table column header with sort and filter actions. */
export function CollectionViewColumnHeader({ column, index }: CollectionViewColumnHeaderProps) {
  const { t } = useI18n()
  const { state, actions } = useCollectionView()
  const isSorted = state.sortColumn === column
  const hasFilter = Object.prototype.hasOwnProperty.call(state.activeFilter, column)
  const width = state.columnWidths[column] || 160
  const fieldType = state.fieldTypes[column]

  return (
    <th
      style={{ minWidth: `${width}px`, ...(state.resizedColumns.has(column) && { maxWidth: `${width}px` }) }}
      className="sticky top-0 z-40 overflow-hidden border-b border-r border-border/50 bg-background px-4 py-2 text-left text-sm font-medium text-muted-foreground select-none"
    >
      <div className="flex items-center justify-between gap-4">
        <div className="flex min-w-0 flex-col">
          <div className="flex min-w-0 items-center gap-1">
            <span className="truncate" title={column}>{column}</span>
            {column === '_id' && (
              <Badge variant="secondary" className="h-4 shrink-0 px-1 py-0 text-[10px]">
                {t('mongodb.table.idBadge')}
              </Badge>
            )}
            {isSorted && (
              <span className="shrink-0 text-primary">
                {state.sortDirection === 'asc' ? <ArrowUpAZ className="h-3 w-3" /> : <ArrowDownAZ className="h-3 w-3" />}
              </span>
            )}
            {hasFilter && <ListFilter className="h-3 w-3 shrink-0 text-primary" />}
          </div>
          {fieldType && (
            <span className="truncate text-xs font-normal text-muted-foreground/80">
              {fieldType}
            </span>
          )}
        </div>

        <DropdownMenu
          open={state.activeColumnMenu === column}
          onOpenChange={(open) => actions.setActiveColumnMenu(open ? column : null)}
        >
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              className={cn(
                'shrink-0 text-muted-foreground',
                state.activeColumnMenu === column && 'bg-muted text-foreground',
              )}
            >
              <MoreHorizontal className="h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align={index === 0 ? 'start' : 'end'} className="w-44">
            <DropdownMenuLabel className="text-[10px] text-muted-foreground">
              {t('mongodb.table.columnActions')}
            </DropdownMenuLabel>
            <DropdownMenuItem
              onSelect={() => actions.handleSort(column, 'asc')}
              className={cn(isSorted && state.sortDirection === 'asc' && 'bg-primary/5 font-medium text-primary')}
            >
              <ArrowUpAZ className="h-3.5 w-3.5" />
              {t('mongodb.table.sortAsc')}
            </DropdownMenuItem>
            <DropdownMenuItem
              onSelect={() => actions.handleSort(column, 'desc')}
              className={cn(isSorted && state.sortDirection === 'desc' && 'bg-primary/5 font-medium text-primary')}
            >
              <ArrowDownAZ className="h-3.5 w-3.5" />
              {t('mongodb.table.sortDesc')}
            </DropdownMenuItem>
            {isSorted && (
              <DropdownMenuItem onSelect={() => actions.clearSort()}>
                <X className="h-3.5 w-3.5" />
                {t('mongodb.table.clearSort')}
              </DropdownMenuItem>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => actions.openFilterForField(column)}>
              <ListFilter className="h-3.5 w-3.5" />
              {t('mongodb.table.filterColumn')}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
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
    </th>
  )
}
