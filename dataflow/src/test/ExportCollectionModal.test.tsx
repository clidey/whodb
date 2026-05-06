import { fireEvent, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ExportCollectionModal } from '@/components/database/mongodb/ExportCollectionModal'
import { downloadBlob } from '@/utils/export-utils'
import { renderWithI18n } from '@/test/renderWithI18n'
import { useConnectionStore, type Connection } from '@/stores/useConnectionStore'

vi.mock('@/utils/export-utils', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/utils/export-utils')>()),
  downloadBlob: vi.fn(),
}))

const originalState = useConnectionStore.getState()

const mongoConnection: Connection = {
  id: 'mongo-1',
  name: 'MongoDB @ localhost',
  type: 'MONGODB',
  host: 'localhost',
  port: '27017',
  user: 'root',
  password: '',
  database: 'admin',
  createdAt: '2026-04-02T00:00:00.000Z',
}

describe('ExportCollectionModal', () => {
  beforeEach(() => {
    useConnectionStore.setState(originalState)
    vi.mocked(downloadBlob).mockReset()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(new Blob(['xlsx']), {
          status: 200,
          headers: {
            'Content-Disposition': 'attachment; filename="analytics_events.xlsx"',
          },
        }),
      ),
    )
  })

  it('exports a MongoDB collection as Excel through the export endpoint', async () => {
    useConnectionStore.setState({
      ...useConnectionStore.getState(),
      connections: [mongoConnection],
    })

    renderWithI18n(
      <ExportCollectionModal
        open
        onOpenChange={vi.fn()}
        connectionId={mongoConnection.id}
        databaseName="analytics"
        collectionName="events"
      />,
      'en',
    )

    fireEvent.click(screen.getByRole('button', { name: 'Excel' }))
    fireEvent.click(screen.getByRole('button', { name: 'Start Export' }))

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/export',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            schema: 'analytics',
            storageUnit: 'events',
            format: 'excel',
          }),
        }),
      )
    })

    expect(downloadBlob).toHaveBeenCalledWith(expect.any(Blob), 'analytics_events.xlsx')
  })

  it('includes the latest filter and limit when exporting', async () => {
    useConnectionStore.setState({
      ...useConnectionStore.getState(),
      connections: [mongoConnection],
    })

    renderWithI18n(
      <ExportCollectionModal
        open
        onOpenChange={vi.fn()}
        connectionId={mongoConnection.id}
        databaseName="analytics"
        collectionName="events"
      />,
      'en',
    )

    fireEvent.change(screen.getByPlaceholderText('{ "status": "active" }'), {
      target: { value: '{ "status": "active" }' },
    })
    fireEvent.change(screen.getByPlaceholderText('No limit'), {
      target: { value: '25' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Start Export' }))

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/export',
        expect.objectContaining({
          body: JSON.stringify({
            schema: 'analytics',
            storageUnit: 'events',
            format: 'ndjson',
            filter: '{ "status": "active" }',
            limit: 25,
          }),
        }),
      )
    })
  })
})
