import { fireEvent, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ActivityBar } from '@/components/layout/ActivityBar'
import { StandaloneLoginForm } from '@/components/auth/StandaloneLogin'
import { renderWithI18n } from './renderWithI18n'

describe('semantic test tags', () => {
  it('renders standalone login semantic tags', () => {
    renderWithI18n(<StandaloneLoginForm onSubmit={vi.fn()} />, 'en')

    expect(screen.getByTestId('auth.standalone.form')).toHaveAttribute('data-qa-action', 'create')
    expect(screen.getByTestId('auth.standalone.host-input')).toHaveAttribute('data-qa-field', 'host')
    expect(screen.getByTestId('auth.standalone.submit-button')).toHaveAttribute('data-qa-state', 'ready')
  })

  it('renders activity tab semantics with resource binding', () => {
    const onTabChange = vi.fn()
    renderWithI18n(<ActivityBar activeTab="connections" onTabChange={onTabChange} />, 'en')

    const tabs = screen.getAllByTestId('layout.activity.tab')
    expect(tabs[0]).toHaveAttribute('data-qa-resource-id', 'connections')
    expect(tabs[0]).toHaveAttribute('data-qa-state', 'active')
    expect(tabs[1]).toHaveAttribute('data-qa-resource-id', 'analysis')

    fireEvent.click(tabs[1])
    expect(onTabChange).toHaveBeenCalledWith('analysis')
  })
})
