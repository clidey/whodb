/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_BUILD_EDITION: 'ce' | 'ee' | undefined
  // Add other env variables as needed
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}