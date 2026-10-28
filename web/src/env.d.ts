/// <reference types="vite/client" />
// Declares the `virtual:msw` module provided by the `msw/vite` plugin.
/// <reference types="msw/vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'

  const component: DefineComponent<
    Record<string, never>,
    Record<string, never>,
    unknown
  >
  export default component
}

interface ImportMetaEnv {
  // Vite exposes every `VITE_*` variable as a string, never a boolean. The parsed
  // boolean is injected separately as the `__VITE_API_MOCKING__` define below, which is
  // what application code should read.
  readonly VITE_API_MOCKING?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare const __VITE_API_MOCKING__: boolean
