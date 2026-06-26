import { useCallback } from 'react'
import { useConnectionStore } from '@/stores/useConnectionStore'
import { connectionToNode } from './SidebarTree/types'
import type { ModalState } from './Sidebar'
import type { Alert } from '@/components/ui/types'
import type { TreeNodeData } from './SidebarTree/types'

import { CreateDatabaseModal } from '@/components/database/CreateDatabaseModal'
import { EditDatabaseModal } from '@/components/database/EditDatabaseModal'
import { DeleteDatabaseModal } from '@/components/database/DeleteDatabaseModal'
import { ExportDatabaseModal } from '@/components/database/ExportDatabaseModal'
import { DatabaseImportModal } from '@/components/database/import/DatabaseImportModal'
import { CreateTableModal } from '@/components/database/sql/CreateTableModal'
import { EditTableModal } from '@/components/database/sql/EditTable/EditTableModal'
import { DeleteTableModal } from '@/components/database/sql/DeleteTableModal'
import { ExportDataModal } from '@/components/database/sql/ExportDataModal'
import { ClearTableDataModal } from '@/components/database/sql/ClearTableDataModal'
import { CopyTableModal } from '@/components/database/sql/CopyTableModal'
import { RenameTableModal } from '@/components/database/sql/RenameTableModal'
import { ExportCollectionModal } from '@/components/database/mongodb/ExportCollectionModal'
import { CreateCollectionModal } from '@/components/database/mongodb/CreateCollectionModal'
import { DropCollectionModal } from '@/components/database/mongodb/DropCollectionModal'
import { RedisKeyModal } from '@/components/database/redis/RedisKeyModal'
import { DeleteRedisKeyModal } from '@/components/database/redis/DeleteRedisKeyModal'
import { ExportRedisKeyModal } from '@/components/database/redis/ExportRedisKeyModal'
import { AlertModal } from '@/components/ui/AlertModal'
import { useTabStore } from '@/stores/useTabStore'

interface SidebarModalsProps {
  activeModal: ModalState | null
  closeModal: () => void
  alert: Alert | null
  closeAlert: () => void
  refreshNode: (node: TreeNodeData) => void
}

/** Renders all sidebar-triggered modals and the alert overlay. */
export function SidebarModals({
  activeModal,
  closeModal,
  alert,
  closeAlert,
  refreshNode,
}: SidebarModalsProps) {
  const { connections, selectItem } = useConnectionStore()

  const onOpenChange = useCallback(
    (open: boolean) => { if (!open) closeModal() },
    [closeModal],
  )

  /** Refresh the schema or database node after a mutation. */
  const refreshSchemaOrDb = useCallback(
    (connectionId: string, databaseName: string, schema?: string) => {
      if (schema) {
        const schemaId = `${connectionId}-${databaseName}-${schema}`
        refreshNode({
          id: schemaId,
          name: schema,
          type: 'schema',
          connectionId,
          metadata: { database: databaseName, schema },
        })
      } else {
        const dbId = `${connectionId}-${databaseName}`
        refreshNode({
          id: dbId,
          name: databaseName,
          type: 'database',
          connectionId,
          metadata: { database: databaseName },
        })
      }
    },
    [refreshNode],
  )

  /** Refresh the redis_keys_folder node after a key mutation. */
  const refreshRedisKeysFolder = useCallback(
    (connectionId: string, databaseName: string) => {
      const folderId = `${connectionId}-${databaseName}-keys`
      refreshNode({
        id: folderId,
        name: 'keys',
        type: 'redis_keys_folder',
        connectionId,
        metadata: { database: databaseName },
      })
    },
    [refreshNode],
  )

  return (
    <>
      {/* Create Database */}
      {activeModal?.type === "create_database" && (
        <CreateDatabaseModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          onSuccess={() => {
            const conn = connections.find((c) => c.id === activeModal.params.connectionId)
            if (conn) refreshNode(connectionToNode(conn))
          }}
        />
      )}

      {/* Create Table */}
      {activeModal?.type === "create_table" && (
        <CreateTableModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          databaseName={activeModal.params.databaseName}
          schema={activeModal.params.schema}
          onSuccess={() => {
            const isPostgres = connections.find((c) => c.id === activeModal.params.connectionId)?.type === "POSTGRES"
            refreshSchemaOrDb(
              activeModal.params.connectionId,
              activeModal.params.databaseName,
              activeModal.params.schema || (isPostgres ? "public" : undefined),
            )
          }}
        />
      )}

      {/* Create Collection (MongoDB) */}
      {activeModal?.type === "create_collection" && (
        <CreateCollectionModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          databaseName={activeModal.params.databaseName}
          onSuccess={() => {
            refreshSchemaOrDb(activeModal.params.connectionId, activeModal.params.databaseName)
          }}
        />
      )}

      {/* Edit Database */}
      {activeModal?.type === "edit_database" && (
        <EditDatabaseModal
          open
          onOpenChange={onOpenChange}
          databaseName={activeModal.params.databaseName}
          onSuccess={() => {
            const conn = connections.find((c) => c.id === activeModal.params.connectionId)
            if (conn) refreshNode(connectionToNode(conn))
          }}
        />
      )}

      {/* Delete Database */}
      {activeModal?.type === "delete_database" && (
        <DeleteDatabaseModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          databaseName={activeModal.params.databaseName}
          onSuccess={() => {
            selectItem(null)
            const conn = connections.find((c) => c.id === activeModal.params.connectionId)
            if (conn) refreshNode(connectionToNode(conn))
          }}
        />
      )}

      {/* Edit Table */}
      {activeModal?.type === "edit_table" && (
        <EditTableModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          databaseName={activeModal.params.databaseName}
          tableName={activeModal.params.tableName}
          schema={activeModal.params.schema}
          onSuccess={() => {
            refreshSchemaOrDb(activeModal.params.connectionId, activeModal.params.databaseName, activeModal.params.schema)
          }}
        />
      )}

      {/* Delete Table */}
      {activeModal?.type === "delete_table" && (
        <DeleteTableModal
          open
          onOpenChange={onOpenChange}
          databaseName={activeModal.params.databaseName}
          schema={activeModal.params.schema}
          tableName={activeModal.params.tableName}
          onSuccess={() => {
            selectItem(null)
            refreshSchemaOrDb(activeModal.params.connectionId, activeModal.params.databaseName, activeModal.params.schema)
          }}
        />
      )}

      {/* Export Data */}
      {activeModal?.type === "export_data" && (
        <ExportDataModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          databaseName={activeModal.params.databaseName}
          schema={activeModal.params.schema}
          tableName={activeModal.params.tableName}
        />
      )}

      {/* Export Database */}
      {activeModal?.type === "export_database" && (
        <ExportDatabaseModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          databaseName={activeModal.params.databaseName}
          schema={activeModal.params.schema}
        />
      )}

      {/* Import Database */}
      {activeModal?.type === "import_database" && (
        <DatabaseImportModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          databaseName={activeModal.params.databaseName}
          schema={activeModal.params.schema}
          tableName={activeModal.params.tableName}
          onSuccess={(context) => {
            refreshSchemaOrDb(
              activeModal.params.connectionId,
              context.databaseName,
              context.schema ?? undefined,
            )
            if (context.tableName) {
              useConnectionStore.getState().triggerTableRefresh()
            }
          }}
        />
      )}

      {/* Clear Table Data */}
      {activeModal?.type === "clear_table_data" && (
        <ClearTableDataModal
          open
          onOpenChange={onOpenChange}
          databaseName={activeModal.params.databaseName}
          schema={activeModal.params.schema}
          tableName={activeModal.params.tableName}
          onSuccess={() => {
            useConnectionStore.getState().triggerTableRefresh()
          }}
        />
      )}

      {/* Copy Table */}
      {activeModal?.type === "copy_table" && (
        <CopyTableModal
          open
          onOpenChange={onOpenChange}
          databaseName={activeModal.params.databaseName}
          schema={activeModal.params.schema}
          tableName={activeModal.params.tableName}
          onSuccess={() => {
            refreshSchemaOrDb(activeModal.params.connectionId, activeModal.params.databaseName, activeModal.params.schema)
          }}
        />
      )}

      {/* Rename Table */}
      {activeModal?.type === "rename_table" && (
        <RenameTableModal
          open
          onOpenChange={onOpenChange}
          databaseName={activeModal.params.databaseName}
          schema={activeModal.params.schema}
          tableName={activeModal.params.tableName}
          onSuccess={() => {
            refreshSchemaOrDb(activeModal.params.connectionId, activeModal.params.databaseName, activeModal.params.schema)
          }}
        />
      )}

      {/* Export Collection */}
      {activeModal?.type === "export_collection" && (
        <ExportCollectionModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          databaseName={activeModal.params.databaseName}
          collectionName={activeModal.params.collectionName}
        />
      )}

      {/* Drop Collection */}
      {activeModal?.type === "drop_collection" && (
        <DropCollectionModal
          open
          onOpenChange={onOpenChange}
          databaseName={activeModal.params.databaseName}
          collectionName={activeModal.params.collectionName}
          onSuccess={() => {
            selectItem(null)
            refreshSchemaOrDb(activeModal.params.connectionId, activeModal.params.databaseName)
          }}
        />
      )}

      {/* Create Redis Key */}
      {activeModal?.type === "create_redis_key" && (
        <RedisKeyModal
          open
          onOpenChange={onOpenChange}
          databaseName={activeModal.params.databaseName}
          onSuccess={() => {
            refreshRedisKeysFolder(activeModal.params.connectionId, activeModal.params.databaseName)
          }}
        />
      )}

      {/* Export Redis Key */}
      {activeModal?.type === "export_redis_key" && (
        <ExportRedisKeyModal
          open
          onOpenChange={onOpenChange}
          connectionId={activeModal.params.connectionId}
          databaseName={activeModal.params.databaseName}
          keyName={activeModal.params.keyName}
        />
      )}

      {/* Delete Redis Key */}
      {activeModal?.type === "delete_redis_key" && (
        <DeleteRedisKeyModal
          open
          onOpenChange={onOpenChange}
          databaseName={activeModal.params.databaseName}
          keyName={activeModal.params.keyName}
          onSuccess={() => {
            selectItem(null)
            const existingTab = useTabStore.getState().findExistingTab(
              'redis_key_detail',
              activeModal.params.connectionId,
              activeModal.params.keyName,
              activeModal.params.databaseName,
            )
            if (existingTab) useTabStore.getState().closeTab(existingTab.id)
            refreshRedisKeysFolder(activeModal.params.connectionId, activeModal.params.databaseName)
          }}
        />
      )}

      {/* Alert Modal */}
      <AlertModal
        isOpen={alert !== null}
        onClose={closeAlert}
        title={alert?.title ?? ''}
        message={alert?.message ?? ''}
        type={alert?.type}
      />
    </>
  )
}
