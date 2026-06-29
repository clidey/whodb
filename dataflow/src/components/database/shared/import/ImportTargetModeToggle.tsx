import { cn } from '@/lib/utils'

interface ImportTargetModeToggleProps {
  label: string
  mode: 'existing' | 'new'
  onModeChange: (mode: 'existing' | 'new') => void
  existingLabel: string
  newLabel: string
  /** Disable only the "existing" option (e.g. no current table/collection to target). */
  existingDisabled?: boolean
  disabled?: boolean
}

/** Segmented existing/new target toggle shared by import dialogs. Each dialog renders its own target body below it. */
export function ImportTargetModeToggle({
  label,
  mode,
  onModeChange,
  existingLabel,
  newLabel,
  existingDisabled,
  disabled,
}: ImportTargetModeToggleProps) {
  return (
    <div className="flex flex-col gap-2">
      <span className="text-sm font-medium text-foreground">{label}</span>
      <div className="inline-flex w-fit rounded-md border border-input p-1">
        <button
          type="button"
          onClick={() => onModeChange('existing')}
          disabled={disabled || existingDisabled}
          aria-pressed={mode === 'existing'}
          className={cn(
            'rounded-sm px-3 py-1.5 text-sm transition-colors disabled:opacity-50',
            mode === 'existing' ? 'bg-highlight-background text-foreground' : 'text-muted-foreground hover:text-foreground',
          )}
        >
          {existingLabel}
        </button>
        <button
          type="button"
          onClick={() => onModeChange('new')}
          disabled={disabled}
          aria-pressed={mode === 'new'}
          className={cn(
            'rounded-sm px-3 py-1.5 text-sm transition-colors',
            mode === 'new' ? 'bg-highlight-background text-foreground' : 'text-muted-foreground hover:text-foreground',
          )}
        >
          {newLabel}
        </button>
      </div>
    </div>
  )
}
