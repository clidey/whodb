import type { ReactNode } from 'react'
import { AlertTriangle, CheckCircle, Info, Loader2, type LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

/** Severity of an import notice. Tone encodes consequence (risk/result/progress), never "which step you're on". */
export type ImportNoticeTone = 'warning' | 'error' | 'success' | 'info'

const TONE_STYLES: Record<ImportNoticeTone, string> = {
  warning: 'border-warning/20 bg-warning/5 text-warning',
  error: 'border-destructive/20 bg-destructive/5 text-destructive',
  success: 'border-success/20 bg-success/5 text-success',
  info: 'border-primary/20 bg-primary/5 text-primary',
}

const TONE_ICONS: Record<ImportNoticeTone, LucideIcon> = {
  warning: AlertTriangle,
  error: AlertTriangle,
  success: CheckCircle,
  info: Info,
}

interface ImportNoticeProps {
  tone: ImportNoticeTone
  children: ReactNode
  /** Show a spinner instead of the tone icon, for in-progress (info) notices. */
  loading?: boolean
  className?: string
}

/** Inline colored notice box shared across import dialogs. */
export function ImportNotice({ tone, children, loading = false, className }: ImportNoticeProps) {
  const Icon = loading ? Loader2 : TONE_ICONS[tone]

  return (
    <div role={tone === 'error' ? 'alert' : 'status'} className={cn('flex items-start gap-2 rounded-md border p-3 text-sm', TONE_STYLES[tone], className)}>
      <Icon className={cn('mt-0.5 h-4 w-4 shrink-0', loading && 'animate-spin')} />
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  )
}
