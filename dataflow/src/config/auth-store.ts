// dataflow/src/config/auth-store.ts

/**
 * Module-level auth credential store with sessionStorage backing.
 *
 * Apollo Client's authLink runs outside the React tree and needs synchronous
 * access to the current credentials. This module holds them in memory and
 * mirrors writes to sessionStorage for cross-refresh persistence.
 *
 * useAuthStore.initialize() controls when to restore from storage (not auto-restored on load).
 */

const STORAGE_KEY = 'dataflow_auth';

export interface AuthSessionSummary {
  sessionToken: string;
  type: string;
  hostname: string;
  port: string;
  database: string;
  displayName: string;
  expiresAt: string;
}

export interface BootstrapDescriptor {
  dbType: string;
  resourceName: string;
  databaseName: string;
  host?: string;
  port?: string;
  namespace?: string;
  fingerprint: string;
}

export interface PersistedAuthState {
  session: AuthSessionSummary | null;
  bootstrap: BootstrapDescriptor | null;
}

let currentAuthState: PersistedAuthState = {
  session: null,
  bootstrap: null,
};
let rebootstrapHandler: (() => Promise<boolean>) | null = null
let standaloneUnauthorizedHandler: (() => Promise<boolean>) | null = null

/** Set the opaque auth session after a successful bootstrap. */
export function setAuthSession(session: AuthSessionSummary): void {
  currentAuthState = { ...currentAuthState, session };
  persistAuthState();
}

/** Set the current bootstrap descriptor used for Sealos rebootstrap. */
export function setBootstrapDescriptor(bootstrap: BootstrapDescriptor | null): void {
  currentAuthState = { ...currentAuthState, bootstrap };
  persistAuthState();
}

/** Set both persisted auth records at once. */
export function setPersistedAuthState(state: PersistedAuthState): void {
  currentAuthState = state;
  persistAuthState();
}

/** Clear auth session and bootstrap descriptor. */
export function clearAuth(): void {
  currentAuthState = { session: null, bootstrap: null };
  sessionStorage.removeItem(STORAGE_KEY);
}

/** Read the current auth session (used by request plumbing). */
export function getAuthSession(): AuthSessionSummary | null {
  return currentAuthState.session;
}

/** Read the current Sealos bootstrap descriptor. */
export function getBootstrapDescriptor(): BootstrapDescriptor | null {
  return currentAuthState.bootstrap;
}

/** Register the shared rebootstrap handler used by request plumbing. */
export function registerRebootstrapHandler(handler: (() => Promise<boolean>) | null): void {
  rebootstrapHandler = handler
}

/** Trigger a single shared rebootstrap attempt when a session expires. */
export async function triggerRebootstrap(): Promise<boolean> {
  if (!rebootstrapHandler) return false
  return rebootstrapHandler()
}

/** Register the handler used when a standalone session token is rejected. */
export function registerStandaloneUnauthorizedHandler(handler: (() => Promise<boolean>) | null): void {
  standaloneUnauthorizedHandler = handler
}

/** Trigger the standalone stale-session handler. */
export async function triggerStandaloneUnauthorized(): Promise<boolean> {
  if (!standaloneUnauthorizedHandler) return false
  return standaloneUnauthorizedHandler()
}

/** Restore persisted auth state from sessionStorage. */
export function restoreFromStorage(): PersistedAuthState | null {
  const stored = sessionStorage.getItem(STORAGE_KEY);
  if (!stored) return null;

  try {
    const state = JSON.parse(stored) as PersistedAuthState;
    currentAuthState = {
      session: state.session ?? null,
      bootstrap: state.bootstrap ?? null,
    };
    return currentAuthState;
  } catch {
    sessionStorage.removeItem(STORAGE_KEY);
    return null;
  }
}

function persistAuthState(): void {
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(currentAuthState));
}
