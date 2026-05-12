import { beforeEach, describe, expect, it, vi } from 'vitest'

const getAuthSessionMock = vi.fn()
const getBootstrapDescriptorMock = vi.fn()
const triggerRebootstrapMock = vi.fn()
const triggerStandaloneUnauthorizedMock = vi.fn()

vi.mock('@/config/auth-store', () => ({
  getAuthSession: getAuthSessionMock,
  getBootstrapDescriptor: getBootstrapDescriptorMock,
  triggerRebootstrap: triggerRebootstrapMock,
  triggerStandaloneUnauthorized: triggerStandaloneUnauthorizedMock,
}))

describe('graphql client auth fetch', () => {
  beforeEach(() => {
    vi.resetModules()
    getAuthSessionMock.mockReset()
    getBootstrapDescriptorMock.mockReset()
    triggerRebootstrapMock.mockReset()
    triggerStandaloneUnauthorizedMock.mockReset()
    vi.stubGlobal('fetch', vi.fn())
  })

  it('clears standalone auth state when a request returns 401 without a Sealos bootstrap descriptor', async () => {
    getBootstrapDescriptorMock.mockReturnValue(null)
    vi.mocked(fetch).mockResolvedValue(new Response('', { status: 401 }))

    const { authFetch } = await import('@/config/graphql-client')

    const response = await authFetch('/api/query', { method: 'POST' })

    expect(response.status).toBe(401)
    expect(triggerStandaloneUnauthorizedMock).toHaveBeenCalledOnce()
    expect(triggerRebootstrapMock).not.toHaveBeenCalled()
    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('retries a Sealos bootstrapped session once after successful rebootstrap', async () => {
    getBootstrapDescriptorMock.mockReturnValue({
      dbType: 'postgresql',
      resourceName: 'my-db',
      databaseName: 'postgres',
      fingerprint: 'postgresql:my-db::postgres',
    })
    getAuthSessionMock.mockReturnValue({
      sessionToken: 'new-token',
      type: 'Postgres',
      hostname: 'db.ns.svc',
      port: '5432',
      database: 'postgres',
      displayName: 'my-db',
      expiresAt: '2026-05-13T00:00:00Z',
    })
    triggerRebootstrapMock.mockResolvedValue(true)
    vi.mocked(fetch)
      .mockResolvedValueOnce(new Response('', { status: 401 }))
      .mockResolvedValueOnce(new Response('', { status: 200 }))

    const { authFetch } = await import('@/config/graphql-client')

    const response = await authFetch('/api/query', {
      method: 'POST',
      headers: {
        'X-WhoDB-Database': 'analytics',
      },
    })

    expect(response.status).toBe(200)
    expect(triggerStandaloneUnauthorizedMock).not.toHaveBeenCalled()
    expect(triggerRebootstrapMock).toHaveBeenCalledOnce()
    expect(fetch).toHaveBeenCalledTimes(2)
    expect(vi.mocked(fetch).mock.calls[1]?.[1]?.headers).toMatchObject({
      Authorization: 'Bearer session:new-token',
      'X-WhoDB-Database': 'analytics',
      'X-WhoDB-Retry-Attempt': '1',
    })
  })
})
