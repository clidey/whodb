import { Checkbox } from '@/components/ui/checkbox'
import { cn } from '@/lib/utils'

interface ImportPreviewTableProps {
  columns: string[]
  rows: ReadonlyArray<ReadonlyArray<string | null>>
  caption?: string
  /** When true, each column header gets a skip checkbox driven by `skipped`/`onToggleColumn`. */
  skippable?: boolean
  skipped?: Set<string>
  onToggleColumn?: (column: string) => void
  truncated?: boolean
  truncatedText?: string
  /** Tailwind min-width utility for the table, e.g. `min-w-128`. */
  minWidthClassName?: string
}

/** Tabular preview of parsed file rows, optionally with per-column skip toggles. Shared by import dialogs. */
export function ImportPreviewTable({
  columns,
  rows,
  caption,
  skippable,
  skipped,
  onToggleColumn,
  truncated,
  truncatedText,
  minWidthClassName,
}: ImportPreviewTableProps) {
  return (
    <div className="flex flex-col gap-2">
      {caption && <span className="text-sm font-medium text-foreground">{caption}</span>}
      <div className="max-h-60 overflow-auto rounded-md border border-input">
        <table className={cn('w-full text-left text-xs', minWidthClassName)}>
          <thead className="bg-muted/50 text-muted-foreground">
            <tr>
              {columns.map((column) => (
                <th key={column} className="px-3 py-2 font-medium">
                  {skippable ? (
                    <label className="flex items-center gap-1.5">
                      <Checkbox checked={!skipped?.has(column)} onCheckedChange={() => onToggleColumn?.(column)} />
                      <span className={cn('truncate', skipped?.has(column) && 'text-muted-foreground line-through')}>{column}</span>
                    </label>
                  ) : (
                    column
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, rowIndex) => (
              <tr key={rowIndex} className="border-t">
                {columns.map((column, columnIndex) => (
                  <td key={`${rowIndex}-${column}`} className={cn('px-3 py-2 font-mono', skippable && skipped?.has(column) && 'text-muted-foreground/50')}>
                    {row[columnIndex] ?? ''}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {truncated && truncatedText && <span className="text-xs text-muted-foreground">{truncatedText}</span>}
    </div>
  )
}
