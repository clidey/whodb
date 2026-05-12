import { beforeEach, describe, expect, it, vi } from 'vitest'

const queryMock = vi.fn()
const mutateMock = vi.fn()
const sealosInitializeMock = vi.fn()

let sealosState = {
  session: null,
  initialize: sealosInitializeMock,
}

vi.mock('@/config/graphql-client', () => ({
  graphqlClient: {
    query: queryMock,
    mutate: mutateMock,
  },
}))

vi.mock('@/stores/useSealosStore', () => ({
  useSealosStore: {
    getState: () => sealosState,
  },
}))

describe('useAuthStore standalone flow', () => {
  beforeEach(() => {
    vi.resetModules()
    queryMock.mockReset()
    mutateMock.mockReset()
    sealosInitializeMock.mockReset()
    sealosState = {
      session: null,
      initialize: sealosInitializeMock,
    }
    sessionStorage.clear()
    window.history.pushState({}, '', '/')
  })

  it('enters the unauthenticated standalone entry when direct access has no session and standalone login is enabled', async () => {
    queryMock.mockResolvedValue({
      data: {
        SettingsConfig: {
          StandaloneLoginEnabled: true,
          DisableCredentialForm: false,
        },
      },
    })

    const { useAuthStore } = await import('@/stores/useAuthStore')

    await useAuthStore.getState().initialize()

    expect(useAuthStore.getState()).toMatchObject({
      status: 'unauthenticated',
      session: null,
      standaloneLoginDisabled: false,
      error: null,
    })
    expect(sealosInitializeMock).not.toHaveBeenCalled()
  })

  it('marks standalone login disabled when either standalone or credential forms are disabled', async () => {
    queryMock.mockResolvedValue({
      data: {
        SettingsConfig: {
          StandaloneLoginEnabled: true,
          DisableCredentialForm: true,
        },
      },
    })

    const { useAuthStore } = await import('@/stores/useAuthStore')

    await useAuthStore.getState().initialize()

    expect(useAuthStore.getState()).toMatchObject({
      status: 'unauthenticated',
      standaloneLoginDisabled: true,
    })
  })

  it('restores an existing standalone auth session without querying settings', async () => {
    sessionStorage.setItem(
      'dataflow_auth',
      JSON.stringify({
        session: {
          sessionToken: 'opaque-token',
          type: 'Postgres',
          hostname: 'localhost',
          port: '5432',
          database: 'postgres',
          displayName: 'Postgres @ localhost/postgres',
          expiresAt: '2026-05-13T00:00:00Z',
        },
        bootstrap: null,
      }),
    )

    const { useAuthStore } = await import('@/stores/useAuthStore')

    await useAuthStore.getState().initialize()

    expect(useAuthStore.getState()).toMatchObject({
      status: 'authenticated',
      session: {
        sessionToken: 'opaque-token',
      },
      bootstrapDescriptor: null,
    })
    expect(queryMock).not.toHaveBeenCalled()
  })

  it('clears a stale standalone session and returns to the standalone entry', async () => {
    sessionStorage.setItem(
      'dataflow_auth',
      JSON.stringify({
        session: {
          sessionToken: 'stale-token',
          type: 'Postgres',
          hostname: 'localhost',
          port: '5432',
          database: 'postgres',
          displayName: 'Postgres @ localhost/postgres',
          expiresAt: '2026-05-13T00:00:00Z',
        },
        bootstrap: null,
      }),
    )
    queryMock.mockResolvedValue({
      data: {
        SettingsConfig: {
          StandaloneLoginEnabled: true,
          DisableCredentialForm: false,
        },
      },
    })

    const { useAuthStore } = await import('@/stores/useAuthStore')
    const { getAuthSession, triggerStandaloneUnauthorized } = await import('@/config/auth-store')

    await useAuthStore.getState().initialize()
    await triggerStandaloneUnauthorized()

    expect(getAuthSession()).toBeNull()
    expect(useAuthStore.getState()).toMatchObject({
      status: 'unauthenticated',
      session: null,
      standaloneLoginDisabled: false,
    })
  })

  it('persists only the returned standalone session summary after login', async () => {
    mutateMock.mockResolvedValue({
      data: {
        CreateStandaloneSession: {
          sessionToken: 'opaque-token',
          type: 'Postgres',
          hostname: 'localhost',
          port: '5432',
          database: 'postgres',
          displayName: 'Postgres @ localhost/postgres',
          expiresAt: '2026-05-13T00:00:00Z',
        },
      },
    })

    const { useAuthStore } = await import('@/stores/useAuthStore')

    await useAuthStore.getState().createStandaloneSession({
      Type: 'Postgres',
      Hostname: 'localhost',
      Username: 'postgres',
      Password: 'secret-password',
      Database: 'postgres',
      Advanced: [{ Key: 'Port', Value: '5432' }],
    })

    expect(useAuthStore.getState()).toMatchObject({
      status: 'authenticated',
      session: {
        sessionToken: 'opaque-token',
      },
      bootstrapDescriptor: null,
    })
    expect(sessionStorage.getItem('dataflow_auth')).not.toContain('secret-password')
    expect(sessionStorage.getItem('dataflow_auth')).not.toContain('Password')
  })
})
