import { describe, expect, it } from 'vitest'

import {
  buildMongoTableColumns,
  buildRenderedMongoDocuments,
  coerceMongoCellDraft,
  hasDocumentField,
  isMongoCellChanged,
  parseMongoFieldJsonDraft,
} from '@/components/database/mongodb/CollectionView/mongo-table-utils'

describe('buildMongoTableColumns', () => {
  it('keeps _id first and sorts discovered fields alphabetically', () => {
    const columns = buildMongoTableColumns({
      sampledFields: ['status', 'profile', 'name'],
      documents: [{ _id: '1', age: 21 }],
      changes: new Map(),
    })

    expect(columns).toEqual(['_id', 'age', 'name', 'profile', 'status'])
  })

  it('includes fields introduced by pending changes', () => {
    const columns = buildMongoTableColumns({
      sampledFields: [],
      documents: [{ _id: '1', name: 'Ada' }],
      changes: new Map([
        ['existing-0', {
          type: 'update',
          originalDocument: { _id: '1', name: 'Ada' },
          document: { _id: '1', name: 'Ada', role: 'admin' },
        }],
      ]),
    })

    expect(columns).toEqual(['_id', 'name', 'role'])
  })
})

describe('buildRenderedMongoDocuments', () => {
  it('renders pending inserts before existing documents and applies pending updates', () => {
    const rows = buildRenderedMongoDocuments({
      documents: [{ _id: '1', name: 'Ada' }],
      pageOffset: 0,
      newRowOrder: ['new-1'],
      changes: new Map([
        ['new-1', {
          type: 'insert',
          originalDocument: {},
          document: { name: 'Grace' },
        }],
        ['existing-0', {
          type: 'update',
          originalDocument: { _id: '1', name: 'Ada' },
          document: { _id: '1', name: 'Ada Lovelace' },
        }],
      ]),
    })

    expect(rows.map((row) => row.doc)).toEqual([
      { name: 'Grace' },
      { _id: '1', name: 'Ada Lovelace' },
    ])
    expect(rows[0].isInserted).toBe(true)
    expect(rows[1].changeType).toBe('update')
  })
})

describe('MongoDB table cell helpers', () => {
  it('distinguishes unset fields from null values', () => {
    const document = { present: null }

    expect(hasDocumentField(document, 'present')).toBe(true)
    expect(hasDocumentField(document, 'missing')).toBe(false)
  })

  it('detects fields added by editing an unset cell', () => {
    expect(isMongoCellChanged({ _id: '1' }, { _id: '1', status: 'active' }, 'status')).toBe(true)
  })

  it('preserves existing scalar types when coercing cell drafts', () => {
    expect(coerceMongoCellDraft(1, '2', true)).toEqual({ ok: true, value: 2 })
    expect(coerceMongoCellDraft(false, 'true', true)).toEqual({ ok: true, value: true })
    expect(coerceMongoCellDraft('Ada', 'Grace', true)).toEqual({ ok: true, value: 'Grace' })
  })

  it('creates string values for null or unset fields', () => {
    expect(coerceMongoCellDraft(null, 'active', true)).toEqual({ ok: true, value: 'active' })
    expect(coerceMongoCellDraft(undefined, 'active', false)).toEqual({ ok: true, value: 'active' })
  })

  it('accepts any valid JSON value for field JSON edits', () => {
    expect(parseMongoFieldJsonDraft('{"status":"paid"}')).toEqual({ ok: true, value: { status: 'paid' } })
    expect(parseMongoFieldJsonDraft('"paid"')).toEqual({ ok: true, value: 'paid' })
    expect(parseMongoFieldJsonDraft('null')).toEqual({ ok: true, value: null })
  })

  it('rejects empty field JSON edits instead of treating them as deletion', () => {
    expect(parseMongoFieldJsonDraft('')).toMatchObject({ ok: false })
  })
})
