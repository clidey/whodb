import { useCallback, useRef, useState, type ReactNode } from 'react'
import { ConfirmationModal } from '@/components/ui/ConfirmationModal'
import { useI18n } from '@/i18n/useI18n'
import { useTabStore, type Tab } from '@/stores/useTabStore'

type PendingLeaveAction = {
  tabIds: string[]
  resolve: (confirmed: boolean) => void
}

type GuardedActionOptions = {
  candidateTabs: Tab[]
  run: () => void
}

/** Coordinates confirmed discards before closing database-edit tabs or leaving the workbench. */
export function useWorkspaceTabLeaveGuard(): {
  runWithWorkspaceTabLeaveGuard: (options: GuardedActionOptions) => void
  confirmWorkspaceTabLeave: (candidateTabs: Tab[]) => Promise<boolean>
  leaveGuardDialog: ReactNode
} {
  const { t } = useI18n()
  const discardUnsavedDatabaseEdits = useTabStore((state) => state.discardUnsavedDatabaseEdits)
  const [pendingAction, setPendingAction] = useState<PendingLeaveAction | null>(null)
  const confirmedCloseRef = useRef(false)

  const confirmWorkspaceTabLeave = useCallback((candidateTabs: Tab[]) => {
    const guardedTabs = candidateTabs.filter((tab) => tab.hasUnsavedDatabaseEdits)
    if (guardedTabs.length === 0) {
      return Promise.resolve(true)
    }

    return new Promise<boolean>((resolve) => {
      setPendingAction({
        tabIds: guardedTabs.map((tab) => tab.id),
        resolve,
      })
    })
  }, [])

  const runWithWorkspaceTabLeaveGuard = useCallback(({ candidateTabs, run }: GuardedActionOptions) => {
    void confirmWorkspaceTabLeave(candidateTabs).then((confirmed) => {
      if (confirmed) run()
    })
  }, [confirmWorkspaceTabLeave])

  const handleClose = useCallback(() => {
    if (confirmedCloseRef.current) {
      confirmedCloseRef.current = false
      return
    }

    pendingAction?.resolve(false)
    setPendingAction(null)
  }, [pendingAction])

  const handleConfirm = useCallback(() => {
    if (!pendingAction) return
    discardUnsavedDatabaseEdits(pendingAction.tabIds)
    pendingAction.resolve(true)
    confirmedCloseRef.current = true
    setPendingAction(null)
  }, [discardUnsavedDatabaseEdits, pendingAction])

  const count = pendingAction?.tabIds.length ?? 0
  const leaveGuardDialog = (
    <ConfirmationModal
      isOpen={!!pendingAction}
      onClose={handleClose}
      onConfirm={handleConfirm}
      title={t('layout.leaveGuard.title', { count })}
      message={t('layout.leaveGuard.message', { count })}
      confirmText={t('common.actions.discard')}
      isDestructive
    />
  )

  return { runWithWorkspaceTabLeaveGuard, confirmWorkspaceTabLeave, leaveGuardDialog }
}
