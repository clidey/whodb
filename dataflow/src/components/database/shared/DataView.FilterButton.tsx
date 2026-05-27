import { Filter } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { useI18n } from '@/i18n/useI18n'

/** Filter button with optional active count badge. */
export function DataViewFilterButton({ onClick, count }: { onClick: () => void; count?: number }) {
  const { t } = useI18n()

  return (
    <Button
      className="rounded-lg gap-2.5 min-w-[86px]"
      onClick={onClick}
      data-testid="data-view.filter-button"
      data-qa-module="data-view"
      data-qa-object="filter"
      data-qa-action="open"
      data-qa-state={count ? 'active' : 'inactive'}
    >
      <Filter className="h-4 w-4" />
      {t('common.actions.filter')}
      {count ? (
        <span className="flex items-center justify-center min-w-4 h-4 text-[10px] font-bold rounded-full bg-primary-foreground text-primary px-1">
          {count}
        </span>
      ) : null}
    </Button>
  )
}
