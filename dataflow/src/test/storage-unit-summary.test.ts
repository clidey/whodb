import { describe, expect, it } from 'vitest'

import { storageUnitToSummary } from '@/stores/useConnectionStore'

describe('storageUnitToSummary', () => {
  it('promotes the system-object attribute to a typed flag', () => {
    expect(storageUnitToSummary({
      Name: 'postgres_log',
      Attributes: [
        { Key: 'Type', Value: 'FOREIGN' },
        { Key: 'whodb:system-object', Value: 'true' },
      ],
    })).toEqual({ name: 'postgres_log', type: 'FOREIGN', system: true })
  })

  it('leaves unmarked units as user-authored', () => {
    expect(storageUnitToSummary({
      Name: 'orders',
      Attributes: [{ Key: 'Type', Value: 'BASE TABLE' }],
    })).toEqual({ name: 'orders', type: 'BASE TABLE', system: false })
  })

  it('defaults the unit type when the attribute is missing', () => {
    expect(storageUnitToSummary({ Name: 'events', Attributes: [] })).toEqual({
      name: 'events',
      type: 'table',
      system: false,
    })
  })
})
