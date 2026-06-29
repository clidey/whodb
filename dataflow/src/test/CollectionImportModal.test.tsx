import { fireEvent, render, screen } from '@testing-library/react'
import { MockedProvider } from '@apollo/client/testing'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { CollectionImportModal } from '@/components/database/mongodb/CollectionImportModal'
import { I18nProvider } from '@/i18n/I18nProvider'
import { GetStorageUnitsDocument } from '@graphql'
import { useConnectionStore, type Connection } from '@/stores/useConnectionStore'

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

const originalState = useConnectionStore.getState()

function renderModal(
  props: Partial<React.ComponentProps<typeof CollectionImportModal>>,
  mocks: React.ComponentProps<typeof MockedProvider>['mocks'] = [],
) {
  return render(
    <MockedProvider mocks={mocks} addTypename={false}>
      <I18nProvider locale="en">
        <CollectionImportModal
          open
          onOpenChange={vi.fn()}
          connectionId={mongoConnection.id}
          databaseName="analytics"
          {...props}
        />
      </I18nProvider>
    </MockedProvider>,
  )
}

describe('CollectionImportModal', () => {
  beforeEach(() => {
    useConnectionStore.setState({ ...originalState, connections: [mongoConnection] })
  })

  it('renders an import dialog locked to a target collection', () => {
    renderModal({ collectionName: 'events' })

    expect(screen.getByText('Import Collection')).toBeInTheDocument()
    expect(screen.getByText('analytics / events')).toBeInTheDocument()
    expect(screen.getByLabelText('Upload a JSON, CSV, or Excel file')).toBeInTheDocument()

    // No file chosen yet, so the import action is disabled.
    expect(screen.getByRole('button', { name: 'Run Import' })).toBeDisabled()

    // The mode control defaults to Append for an existing collection. The custom Select renders
    // its options in a portal only when opened, so assert the trigger value, not the option list.
    expect(screen.getByRole('combobox')).toHaveTextContent('Append documents')

    // The target switcher is hidden when a collection is pre-selected.
    expect(screen.queryByRole('button', { name: 'New collection' })).not.toBeInTheDocument()
  })

  it('offers existing/new targets when opened from the database node', () => {
    const mocks = [
      {
        request: { query: GetStorageUnitsDocument, variables: { schema: 'analytics' } },
        result: { data: { StorageUnit: [{ Name: 'events', Attributes: [] }] } },
      },
    ]

    renderModal({ collectionName: null }, mocks)

    expect(screen.getByRole('button', { name: 'Existing collection' })).toBeInTheDocument()

    const newTarget = screen.getByRole('button', { name: 'New collection' })
    fireEvent.click(newTarget)

    // Switching to a new collection reveals the name input.
    expect(screen.getByLabelText('New collection name')).toBeInTheDocument()
  })
})
