import { create } from 'zustand'
import {
  BootstrapSealosSessionDocument,
  CreateStandaloneSessionDocument,
  SettingsConfigDocument,
  type BootstrapSealosSessionMutation,
  type BootstrapSealosSessionMutationVariables,
  type CreateStandaloneSessionMutation,
  type CreateStandaloneSessionMutationVariables,
  type LoginCredentials,
  type SettingsConfigQuery,
  type SettingsConfigQueryVariables,
} from '@graphql'
import { graphqlClient } from '@/config/graphql-client'
import {
  type AuthSessionSummary,
  type BootstrapDescriptor,
  getBootstrapDescriptor,
  registerRebootstrapHandler,
  registerStandaloneUnauthorizedHandler,
  restoreFromStorage,
  setPersistedAuthState,
  clearAuth,
} from '@/config/auth-store'
import {
  getDefaultDatabase,
  isSealosContext,
} from '@/config/sealos'
import { replaceBootstrapUrl } from '@/i18n/url-params'
import { useSealosStore } from '@/stores/useSealosStore'

type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated' | 'error'

interface AuthState {
  status: AuthStatus
  session: AuthSessionSummary | null
  bootstrapDescriptor: BootstrapDescriptor | null
  standaloneLoginDisabled: boolean
  error: string | null
  initialize: () => Promise<void>
  createStandaloneSession: (credentials: LoginCredentials) => Promise<AuthSessionSummary>
  rebootstrap: () => Promise<boolean>
}

interface AuthStoreState extends AuthState {
  rebootstrapWithDescriptor: (descriptor: BootstrapDescriptor) => Promise<boolean>
}

async function bootstrapSealosSession(
  descriptor: BootstrapDescriptor,
  kubeconfig: string,
): Promise<AuthSessionSummary> {
  const result = await graphqlClient.mutate<
    BootstrapSealosSessionMutation,
    BootstrapSealosSessionMutationVariables
  >({
    mutation: BootstrapSealosSessionDocument,
    variables: {
      input: {
        kubeconfig,
        dbType: descriptor.dbType,
        resourceName: descriptor.resourceName,
        databaseName: descriptor.databaseName || undefined,
        host: descriptor.host,
        port: descriptor.port,
        namespace: descriptor.namespace,
      },
    },
  })

  const payload = result.data?.BootstrapSealosSession
  if (!payload) {
    throw new Error('Missing bootstrap payload')
  }

  return {
    sessionToken: payload.sessionToken,
    type: payload.type,
    hostname: payload.hostname,
    port: payload.port,
    database: payload.database,
    displayName: payload.displayName,
    expiresAt: payload.expiresAt,
  }
}

async function getStandaloneLoginDisabled(): Promise<boolean> {
  const result = await graphqlClient.query<SettingsConfigQuery, SettingsConfigQueryVariables>({
    query: SettingsConfigDocument,
  })
  const config = result.data?.SettingsConfig
  if (!config) {
    throw new Error('Missing settings config')
  }
  return !config.StandaloneLoginEnabled || config.DisableCredentialForm
}

async function createStandaloneAuthSession(credentials: LoginCredentials): Promise<AuthSessionSummary> {
  const result = await graphqlClient.mutate<
    CreateStandaloneSessionMutation,
    CreateStandaloneSessionMutationVariables
  >({
    mutation: CreateStandaloneSessionDocument,
    variables: {
      credentials,
    },
  })

  const payload = result.data?.CreateStandaloneSession
  if (!payload) {
    throw new Error('Missing standalone session payload')
  }

  return {
    sessionToken: payload.sessionToken,
    type: payload.type,
    hostname: payload.hostname,
    port: payload.port,
    database: payload.database,
    displayName: payload.displayName,
    expiresAt: payload.expiresAt,
  }
}

function buildBootstrapDescriptor(params: URLSearchParams): BootstrapDescriptor {
  const dbType = params.get('dbType') ?? ''
  const resourceName = params.get('resourceName') ?? ''
  const databaseName = params.get('databaseName') ?? params.get('dbName') ?? getDefaultDatabase(dbType)
  const host = params.get('host') ?? undefined
  const port = params.get('port') ?? undefined
  const namespace = params.get('namespace') ?? undefined
  const fingerprint = [
    dbType,
    resourceName,
    host ?? '',
    port ?? '',
    databaseName,
  ].join(':')

  return {
    dbType,
    resourceName,
    databaseName,
    host,
    port,
    namespace,
    fingerprint,
  }
}

export const useAuthStore = create<AuthStoreState>((set, get) => ({
  status: 'loading',
  session: null,
  bootstrapDescriptor: null,
  standaloneLoginDisabled: false,
  error: null,

  initialize: async () => {
    const params = new URLSearchParams(window.location.search)

    if (isSealosContext(params)) {
      await useSealosStore.getState().initialize()
      const descriptor = buildBootstrapDescriptor(params)
      const restored = restoreFromStorage()

      if (restored?.session && restored.bootstrap?.fingerprint === descriptor.fingerprint) {
        replaceBootstrapUrl(window.location.search)
        set({
          session: restored.session,
          bootstrapDescriptor: restored.bootstrap,
          standaloneLoginDisabled: false,
          status: 'authenticated',
          error: null,
        })
        return
      }

      const ok = await get().rebootstrapWithDescriptor(descriptor)
      if (ok) {
        replaceBootstrapUrl(window.location.search)
      }
      return
    }

    const restored = restoreFromStorage()
    if (restored?.session) {
      set({
        session: restored.session,
        bootstrapDescriptor: restored.bootstrap,
        standaloneLoginDisabled: false,
        status: 'authenticated',
        error: null,
      })
      return
    }

    const standaloneLoginDisabled = await getStandaloneLoginDisabled()
    set({
      session: null,
      bootstrapDescriptor: null,
      standaloneLoginDisabled,
      error: null,
      status: 'unauthenticated',
    })
  },

  createStandaloneSession: async (credentials) => {
    const session = await createStandaloneAuthSession(credentials)
    setPersistedAuthState({ session, bootstrap: null })
    set({
      session,
      bootstrapDescriptor: null,
      standaloneLoginDisabled: false,
      status: 'authenticated',
      error: null,
    })
    return session
  },

  rebootstrap: async () => {
    const descriptor = get().bootstrapDescriptor ?? getBootstrapDescriptor()
    if (!descriptor) return false
    return get().rebootstrapWithDescriptor(descriptor)
  },

  rebootstrapWithDescriptor: async (descriptor: BootstrapDescriptor) => {
    clearAuth()
    set({
      status: 'loading',
      error: null,
      session: null,
      standaloneLoginDisabled: false,
      bootstrapDescriptor: descriptor,
    })

    const sealosSession = useSealosStore.getState().session
    const kubeconfig = sealosSession?.kubeconfig ?? ''
    if (!kubeconfig) {
      set({
        status: 'error',
        error: 'Missing Sealos kubeconfig',
      })
      return false
    }

    try {
      const session = await bootstrapSealosSession(descriptor, kubeconfig)
      setPersistedAuthState({ session, bootstrap: descriptor })
      set({
        session,
        bootstrapDescriptor: descriptor,
        standaloneLoginDisabled: false,
        status: 'authenticated',
        error: null,
      })
      return true
    } catch (error) {
      clearAuth()
      set({
        session: null,
        bootstrapDescriptor: descriptor,
        standaloneLoginDisabled: false,
        status: 'error',
        error: error instanceof Error ? error.message : String(error),
      })
      return false
    }
  },
}))

registerRebootstrapHandler(() => useAuthStore.getState().rebootstrap())
registerStandaloneUnauthorizedHandler(async () => {
  clearAuth()
  const standaloneLoginDisabled = await getStandaloneLoginDisabled()
  useAuthStore.setState({
    status: 'unauthenticated',
    session: null,
    bootstrapDescriptor: null,
    standaloneLoginDisabled,
    error: null,
  })
  return true
})
