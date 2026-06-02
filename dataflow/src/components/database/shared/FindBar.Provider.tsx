import { createContext, use, useState, useCallback, useMemo, useRef, useEffect, type ReactNode } from 'react'

/** A single cell match found by find-in-page. */
export interface FindMatch {
  rowIndex: number
  columnKey: string
}

/** State exposed to consumers via context. */
export interface FindBarState {
  searchTerm: string
  matches: FindMatch[]
  currentMatchIndex: number
  total: number
}

/** Actions exposed to consumers via context. */
export interface FindBarActions {
  setSearchTerm: (term: string) => void
  goToNext: () => void
  goToPrevious: () => void
  clear: () => void
}

/** Metadata exposed to consumers via context. */
export interface FindBarMeta {
  inputRef: React.RefObject<HTMLInputElement | null>
}

export interface FindBarContextValue {
  state: FindBarState
  actions: FindBarActions
  meta: FindBarMeta
}

export const FindBarContext = createContext<FindBarContextValue | null>(null)

/** Access FindBar context. Must be used within FindBar.Provider. */
export function useFindBar(): FindBarContextValue {
  const ctx = use(FindBarContext)
  if (!ctx) throw new Error('useFindBar must be used within FindBar.Provider')
  return ctx
}

interface FindBarProviderProps {
  /** Visible row data to search through. */
  rows: Record<string, unknown>[] | undefined
  /** Column keys to search. */
  columns: string[] | undefined
  children: ReactNode
}

/** Provider that manages find-in-page state. Wrap around both the FindBar UI and the data grid. */
export function FindBarProvider({
  rows,
  columns,
  children,
}: FindBarProviderProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [currentMatchIndex, setCurrentMatchIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement | null>(null)

  const matches = useMemo<FindMatch[]>(() => {
    if (!searchTerm.trim() || !rows || !columns) return []
    const term = searchTerm.toLowerCase()
    const result: FindMatch[] = []
    for (let rowIndex = 0; rowIndex < rows.length; rowIndex++) {
      const row = rows[rowIndex]
      for (const columnKey of columns) {
        const value = row[columnKey]
        if (value != null && String(value).toLowerCase().includes(term)) {
          result.push({ rowIndex, columnKey })
        }
      }
    }
    return result
  }, [searchTerm, rows, columns])

  // Reset currentMatchIndex when matches change
  useEffect(() => {
    setCurrentMatchIndex(0)
  }, [matches])

  const goToNext = useCallback(() => {
    if (matches.length === 0) return
    setCurrentMatchIndex((prev) => (prev + 1) % matches.length)
  }, [matches.length])

  const goToPrevious = useCallback(() => {
    if (matches.length === 0) return
    setCurrentMatchIndex((prev) => (prev - 1 + matches.length) % matches.length)
  }, [matches.length])

  const clear = useCallback(() => {
    setSearchTerm('')
    setCurrentMatchIndex(0)
  }, [])

  // Scroll to the current match element whenever it changes
  useEffect(() => {
    if (matches.length === 0) return
    // Allow a microtask for React to render the data attribute before querying
    const raf = requestAnimationFrame(() => {
      const el = document.querySelector('[data-find-current="true"]')
      el?.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'smooth' })
    })
    return () => cancelAnimationFrame(raf)
  }, [currentMatchIndex, matches])

  // Global ⌘F / Ctrl+F shortcut to focus the search input
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
        e.preventDefault()
        inputRef.current?.focus()
        inputRef.current?.select()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  const state: FindBarState = {
    searchTerm,
    matches,
    currentMatchIndex,
    total: matches.length,
  }

  const actions: FindBarActions = {
    setSearchTerm,
    goToNext,
    goToPrevious,
    clear,
  }

  const meta: FindBarMeta = { inputRef }

  return (
    <FindBarContext value={{ state, actions, meta }}>
      {children}
    </FindBarContext>
  )
}
