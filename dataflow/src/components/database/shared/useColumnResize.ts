import { useCallback, useEffect, useRef, useState, type MouseEvent as ReactMouseEvent } from 'react'

interface ColumnResizeOptions {
  initialWidth?: number
  minimumWidth?: number
}

/**
 * Manages column resize state and document-level mouse event listeners for drag resizing.
 * Returns current column widths and a handler to initiate resizing on mousedown.
 */
export function useColumnResize(
  columns: string[] | undefined,
  { initialWidth = 120, minimumWidth = 60 }: ColumnResizeOptions = {},
): {
  columnWidths: Record<string, number>
  resizingColumn: string | null
  resizedColumns: Set<string>
  handleResizeStart: (event: ReactMouseEvent, column: string) => void
} {
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({})
  const [resizingColumn, setResizingColumn] = useState<string | null>(null)
  const [resizedColumns, setResizedColumns] = useState<Set<string>>(new Set())
  const resizingRef = useRef<{ column: string; startX: number; startWidth: number } | null>(null)

  const getInitialWidth = useCallback((column: string) => {
    return Math.max(initialWidth, column.length * 10 + 60)
  }, [initialWidth])

  // Initialize widths when columns first arrive.
  useEffect(() => {
    if (columns && columns.length > 0 && Object.keys(columnWidths).length === 0) {
      const initialWidths: Record<string, number> = {}
      columns.forEach((column) => {
        initialWidths[column] = getInitialWidth(column)
      })
      setColumnWidths(initialWidths)
    }
  }, [columnWidths, columns, getInitialWidth])

  // Document-level mousemove/mouseup for drag resizing.
  useEffect(() => {
    const handleMouseMove = (event: MouseEvent) => {
      if (resizingRef.current) {
        const { column, startX, startWidth } = resizingRef.current
        const diff = event.clientX - startX
        const newWidth = Math.max(minimumWidth, startWidth + diff)
        setColumnWidths(prev => ({ ...prev, [column]: newWidth }))
      }
    }

    const handleMouseUp = () => {
      if (resizingRef.current) {
        const column = resizingRef.current.column
        setResizedColumns(prev => {
          if (prev.has(column)) return prev
          const next = new Set(prev)
          next.add(column)
          return next
        })
        resizingRef.current = null
        setResizingColumn(null)
        document.body.style.cursor = 'default'
        document.querySelectorAll<HTMLElement>('[data-resize-active]').forEach(element => {
          delete element.dataset.resizeActive
        })
      }
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [minimumWidth])

  const handleResizeStart = useCallback((event: ReactMouseEvent, column: string) => {
    event.preventDefault()
    event.stopPropagation()
    resizingRef.current = {
      column,
      startX: event.clientX,
      startWidth: columnWidths[column] || getInitialWidth(column),
    }
    setResizingColumn(column)
    document.body.style.cursor = 'col-resize'
  }, [columnWidths, getInitialWidth])

  return { columnWidths, resizingColumn, resizedColumns, handleResizeStart }
}
