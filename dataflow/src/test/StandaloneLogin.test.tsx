import { fireEvent, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { StandaloneLoginForm } from '@/components/auth/StandaloneLogin'
import { renderWithI18n } from './renderWithI18n'

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('StandaloneLoginForm', () => {
  const onSubmit = vi.fn()

  beforeEach(() => {
    onSubmit.mockReset()
  })

  it('renders PostgreSQL connection defaults', () => {
    renderWithI18n(<StandaloneLoginForm onSubmit={onSubmit} />, 'en')

    expect(screen.getByLabelText('Database type')).toHaveValue('Postgres')
    expect(screen.getByLabelText('Host')).toHaveValue('localhost')
    expect(screen.getByLabelText('Port')).toHaveValue('5432')
    expect(screen.getByLabelText('Database')).toHaveValue('postgres')
    expect(screen.getByRole('button', { name: 'Connect' })).toBeEnabled()
  })

  it('updates untouched port and database defaults when the database type changes', () => {
    renderWithI18n(<StandaloneLoginForm onSubmit={onSubmit} />, 'en')

    fireEvent.change(screen.getByLabelText('Database type'), { target: { value: 'MongoDB' } })

    expect(screen.getByLabelText('Port')).toHaveValue('27017')
    expect(screen.getByLabelText('Database')).toHaveValue('admin')
  })

  it('preserves user-edited port and database values when the database type changes', () => {
    renderWithI18n(<StandaloneLoginForm onSubmit={onSubmit} />, 'en')

    fireEvent.change(screen.getByLabelText('Port'), { target: { value: '15432' } })
    fireEvent.change(screen.getByLabelText('Database'), { target: { value: 'appdb' } })
    fireEvent.change(screen.getByLabelText('Database type'), { target: { value: 'Redis' } })

    expect(screen.getByLabelText('Port')).toHaveValue('15432')
    expect(screen.getByLabelText('Database')).toHaveValue('appdb')
  })

  it('submits login credentials with the port carried as an advanced option', async () => {
    onSubmit.mockResolvedValue(undefined)
    renderWithI18n(<StandaloneLoginForm onSubmit={onSubmit} />, 'en')

    fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'postgres' } })
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }))

    await waitFor(() => expect(onSubmit).toHaveBeenCalledOnce())
    expect(onSubmit).toHaveBeenCalledWith({
      Type: 'Postgres',
      Hostname: 'localhost',
      Username: 'postgres',
      Password: 'secret',
      Database: 'postgres',
      Advanced: [{ Key: 'Port', Value: '5432' }],
    })
  })

  it('disables submit while the login request is running', async () => {
    const pending = deferred<void>()
    onSubmit.mockReturnValue(pending.promise)
    renderWithI18n(<StandaloneLoginForm onSubmit={onSubmit} />, 'en')

    fireEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(screen.getByRole('button', { name: 'Connecting...' })).toBeDisabled()
    pending.resolve()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Connect' })).toBeEnabled())
  })

  it('shows backend login errors without adding fallback advice', async () => {
    onSubmit.mockRejectedValue(new Error('bad password'))
    renderWithI18n(<StandaloneLoginForm onSubmit={onSubmit} />, 'en')

    fireEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('Login failed: bad password')).toBeVisible()
  })
})
