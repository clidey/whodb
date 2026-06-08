import { describe, expect, it } from 'vitest'

import {
  buildMongoTableColumns,
  buildRenderedMongoDocuments,
  coerceMongoCellDraft,
  hasDocumentField,
  isMongoCellChanged,
  parseMongoDocumentRow,
  parseMongoFieldJsonDraft,
} from '@/components/database/mongodb/CollectionView/mongo-table-utils'

describe('buildMongoTableColumns', () => {
  it('keeps _id first and preserves first-seen document field order', () => {
    const columns = buildMongoTableColumns({
      documents: [
        { _id: '1', status: 'active', profile: {}, name: 'Ada', age: 21 },
        { _id: '2', email: 'ada@example.com', name: 'Ada', createdAt: '2026-01-01' },
      ],
      documentFieldOrders: [
        ['_id', 'status', 'profile', 'name', 'age'],
        ['_id', 'email', 'name', 'createdAt'],
      ],
      changes: new Map(),
    })

    expect(columns).toEqual(['_id', 'status', 'profile', 'name', 'age', 'email', 'createdAt'])
  })

  it('includes fields introduced by pending changes', () => {
    const columns = buildMongoTableColumns({
      documents: [{ _id: '1', name: 'Ada' }],
      documentFieldOrders: [['_id', 'name']],
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

describe('parseMongoDocumentRow', () => {
  it('preserves top-level field order from the raw JSON document', () => {
    const parsed = parseMongoDocumentRow('{"z":1,"nested":{"b":2,"a":1},"arr":[{"y":1,"x":2}],"_id":"1","a":3}')

    expect(parsed.fieldOrder).toEqual(['z', 'nested', 'arr', '_id', 'a'])
    expect(parsed.document).toEqual({
      z: 1,
      nested: { b: 2, a: 1 },
      arr: [{ y: 1, x: 2 }],
      _id: '1',
      a: 3,
    })
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

  it('coerces complete JSON object or array drafts into complex values', () => {
    expect(coerceMongoCellDraft('paid', '{"method":"card"}', true)).toEqual({
      ok: true,
      value: { method: 'card' },
    })
    expect(coerceMongoCellDraft(1, '[1,"a",{"x":true}]', true)).toEqual({
      ok: true,
      value: [1, 'a', { x: true }],
    })
    expect(coerceMongoCellDraft(false, '  {}  ', true)).toEqual({ ok: true, value: {} })
  })

  it('creates string values for null or unset fields', () => {
    expect(coerceMongoCellDraft(null, 'active', true)).toEqual({ ok: true, value: 'active' })
    expect(coerceMongoCellDraft(undefined, 'active', false)).toEqual({ ok: true, value: 'active' })
  })

  it('coerces complete JSON object or array drafts for null or unset fields', () => {
    expect(coerceMongoCellDraft(null, '{}', true)).toEqual({ ok: true, value: {} })
    expect(coerceMongoCellDraft(undefined, '[]', false)).toEqual({ ok: true, value: [] })
  })

  it('saves quoted object or array literals as strings when string values are allowed', () => {
    expect(coerceMongoCellDraft('paid', '"{}"', true)).toEqual({ ok: true, value: '{}' })
    expect(coerceMongoCellDraft(null, '"[]"', true)).toEqual({ ok: true, value: '[]' })
    expect(coerceMongoCellDraft(undefined, '"{\\"method\\":\\"card\\"}"', false)).toEqual({
      ok: true,
      value: '{"method":"card"}',
    })
  })

  it('keeps quoted object or array literals on the normal scalar path when string values are not allowed', () => {
    expect(coerceMongoCellDraft(1, '"{}"', true)).toEqual({ ok: false, error: 'invalid-number' })
    expect(coerceMongoCellDraft(false, '"[]"', true)).toEqual({ ok: false, error: 'invalid-boolean' })
  })

  it('keeps invalid object-like drafts on the normal scalar coercion path', () => {
    expect(coerceMongoCellDraft('paid', '{method:"card"}', true)).toEqual({ ok: true, value: '{method:"card"}' })
    expect(coerceMongoCellDraft(null, '[1,', true)).toEqual({ ok: true, value: '[1,' })
    expect(coerceMongoCellDraft(1, '{method:"card"}', true)).toEqual({ ok: false, error: 'invalid-number' })
    expect(coerceMongoCellDraft(false, '[1,', true)).toEqual({ ok: false, error: 'invalid-boolean' })
  })

  it('keeps scalar string whitespace when no object or array is detected', () => {
    expect(coerceMongoCellDraft('paid', '  active  ', true)).toEqual({ ok: true, value: '  active  ' })
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
