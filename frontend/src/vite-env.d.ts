/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_BUILD_EDITION: 'ce' | 'ee' | undefined
  readonly VITE_E2E_TEST: 'true' | 'false' | undefined
  // Add other env variables as needed
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}