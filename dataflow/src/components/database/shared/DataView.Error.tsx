import { Database } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { useI18n } from '@/i18n/useI18n'

interface DataViewErrorProps {
  message: string
  onRetry?: () => void
}

/** Error card with optional retry button for data views. */
export function DataViewError({ message, onRetry }: DataViewErrorProps) {
  const { t } = useI18n()

  return (
    <div
      className="flex h-full items-center justify-center bg-muted/5"
      data-testid="data-view.error"
      data-qa-module="data-view"
      data-qa-object="data-load"
      data-qa-state="error"
      data-qa-error-code="data_load_failed"
    >
      <div className="text-center p-8 bg-background rounded-xl shadow-sm border">
        <Database className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <p className="text-sm text-muted-foreground">{message}</p>
        {onRetry && (
          <Button
            variant="outline"
            className="mt-4"
            onClick={onRetry}
            data-testid="data-view.retry-button"
            data-qa-module="data-view"
            data-qa-object="data-load"
            data-qa-action="retry"
          >
            {t('common.actions.retry')}
          </Button>
        )}
      </div>
    </div>
  )
}
