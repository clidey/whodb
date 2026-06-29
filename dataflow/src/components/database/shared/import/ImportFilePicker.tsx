import type { ChangeEvent, Ref } from 'react'
import { Input } from '@/components/ui/Input'

interface ImportFilePickerProps {
  label: string
  accept: string
  ariaLabel: string
  onChange: (event: ChangeEvent<HTMLInputElement>) => void
  /** Already-localized "Selected: file.csv" line. */
  selectedFileName?: string | null
  /** Muted hint shown when no file is selected. */
  hint?: string
  error?: string | null
  disabled?: boolean
  inputRef?: Ref<HTMLInputElement>
}

/** File input row shared by import dialogs: label, native picker, then selected name / hint / error. */
export function ImportFilePicker({
  label,
  accept,
  ariaLabel,
  onChange,
  selectedFileName,
  hint,
  error,
  disabled,
  inputRef,
}: ImportFilePickerProps) {
  return (
    <div className="flex flex-col gap-2">
      <label className="text-sm font-medium text-foreground">{label}</label>
      <Input ref={inputRef} type="file" accept={accept} onChange={onChange} disabled={disabled} aria-label={ariaLabel} />
      {selectedFileName ? (
        <span className="min-w-0 truncate text-xs text-muted-foreground">{selectedFileName}</span>
      ) : hint ? (
        <span className="text-xs text-muted-foreground">{hint}</span>
      ) : null}
      {error && <span className="text-xs text-destructive">{error}</span>}
    </div>
  )
}
