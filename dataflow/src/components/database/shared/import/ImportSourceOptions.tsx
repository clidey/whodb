import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

/** Sentinel for the "auto detect" delimiter, since Radix Select forbids an empty-string item value. */
const AUTO_DELIMITER = '__auto__'

interface ImportDelimiterOption {
  value: string
  label: string
}

interface ImportSourceOptionsProps {
  showDelimiter: boolean
  showSheet: boolean
  delimiterLabel: string
  delimiterOptions: ImportDelimiterOption[]
  delimiter: string
  onDelimiterChange: (value: string) => void
  sheetLabel: string
  sheetPlaceholder: string
  sheetOptions: string[]
  sheet: string
  onSheetChange: (value: string) => void
  disabled?: boolean
}

/** CSV delimiter and Excel sheet inputs, shown per detected file format. Shared by import dialogs. */
export function ImportSourceOptions({
  showDelimiter,
  showSheet,
  delimiterLabel,
  delimiterOptions,
  delimiter,
  onDelimiterChange,
  sheetLabel,
  sheetPlaceholder,
  sheetOptions,
  sheet,
  onSheetChange,
  disabled,
}: ImportSourceOptionsProps) {
  if (!showDelimiter && !showSheet) return null

  return (
    <div className="flex w-full flex-col gap-3">
      {showDelimiter && (
        <div className="flex w-full flex-col gap-2">
          <label className="text-sm font-medium text-foreground">{delimiterLabel}</label>
          <Select
            value={delimiter === '' ? AUTO_DELIMITER : delimiter}
            onValueChange={(next) => onDelimiterChange(next === AUTO_DELIMITER ? '' : next)}
            disabled={disabled}
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {delimiterOptions.map((option) => (
                <SelectItem key={option.label} value={option.value === '' ? AUTO_DELIMITER : option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}
      {showSheet && (
        <div className="flex w-full flex-col gap-2">
          <label className="text-sm font-medium text-foreground">{sheetLabel}</label>
          <Select value={sheet || undefined} onValueChange={onSheetChange} disabled={disabled}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder={sheetPlaceholder} />
            </SelectTrigger>
            <SelectContent>
              {sheetOptions.map((option) => (
                <SelectItem key={option} value={option}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}
    </div>
  )
}
