import { describe, expect, it } from 'vitest'

import { messagesByLocale } from '@/i18n/messages'

describe('standalone login localization', () => {
  it('keeps the disabled state copy exact in every locale', () => {
    expect(messagesByLocale.en['standaloneLogin.disabled']).toBe('Standalone login is disabled')
    expect(messagesByLocale.zh['standaloneLogin.disabled']).toBe('Standalone login is disabled')
  })
})
