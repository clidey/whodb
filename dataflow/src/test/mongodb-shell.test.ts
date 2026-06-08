import { describe, expect, it } from 'vitest'

import {
  buildMongoCollectionAccessor,
  buildMongoDropDatabaseCommand,
  buildMongoInsertOneCommand,
  parseMongoDocumentInput,
} from '@/utils/mongodb-shell'

describe('parseMongoDocumentInput', () => {
  it('parses nested MongoDB document JSON without coercing nulls or arrays', () => {
    const document = parseMongoDocumentInput(`{
      "name": "Ada",
      "profile": { "active": true, "tags": ["admin", null] },
      "visits": 3,
      "deletedAt": null
    }`)

    expect(document).toEqual({
      name: 'Ada',
      profile: { active: true, tags: ['admin', null] },
      visits: 3,
      deletedAt: null,
    })
  })

  it('rejects non-object JSON payloads', () => {
    expect(() => parseMongoDocumentInput('["not", "a", "document"]')).toThrow(
      'MongoDB document input must be a JSON object',
    )
    expect(() => parseMongoDocumentInput('"scalar"')).toThrow(
      'MongoDB document input must be a JSON object',
    )
  })
})

describe('buildMongoInsertOneCommand', () => {
  it('builds insertOne query with preserved nested JSON', () => {
    const command = buildMongoInsertOneCommand('users', {
      profile: { active: true },
      tags: ['admin', null],
    })

    expect(command).toBe('db.users.insertOne({"profile":{"active":true},"tags":["admin",null]})')
  })

  it('falls back to getCollection for unsafe collection names', () => {
    const command = buildMongoInsertOneCommand('audit-log', { ok: true })

    expect(command).toBe('db.getCollection("audit-log").insertOne({"ok":true})')
  })

  it('uses explicit document field order when provided', () => {
    const command = buildMongoInsertOneCommand('users', { a: 1, z: 2 }, ['z', 'a'])

    expect(command).toBe('db.users.insertOne({"z":2,"a":1})')
  })
})

describe('buildMongoCollectionAccessor', () => {
  it('uses dot notation for safe collection names', () => {
    expect(buildMongoCollectionAccessor('system.users')).toBe('db.system.users')
  })
})

describe('buildMongoDropDatabaseCommand', () => {
  it('builds the MongoDB database drop command', () => {
    expect(buildMongoDropDatabaseCommand()).toBe('db.dropDatabase()')
  })
})
