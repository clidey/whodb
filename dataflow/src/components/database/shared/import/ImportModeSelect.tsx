import { ImportMode } from '@graphql'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

interface ImportModeOption {
  value: ImportMode
  label: string
}

interface ImportModeSelectProps {
  label: string
  value: ImportMode
  options: ImportModeOption[]
  onChange: (mode: ImportMode) => void
  /** Muted note under the select, e.g. "New tables only support append". */
  note?: string
  disabled?: boolean
}

/** Import mode (append / upsert / overwrite) select, shared by import dialogs. */
export function ImportModeSelect({ label, value, options, onChange, note, disabled }: ImportModeSelectProps) {
  return (
    <div className="flex flex-col gap-2">
      <span className="text-sm font-medium text-foreground">{label}</span>
      <Select value={value} onValueChange={(next) => onChange(next as ImportMode)} disabled={disabled}>
        <SelectTrigger className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
          ))}
        </SelectContent>
      </Select>
      {note && <span className="text-xs text-muted-foreground">{note}</span>}
    </div>
  )
}
