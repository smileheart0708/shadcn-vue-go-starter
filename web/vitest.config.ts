import { defineConfig } from 'vitest/config'

/**
 * Separate from `vite.config.ts` on purpose: the suite covers pure logic and state machines
 * (see AGENTS.md), so the app's plugins and the `__VITE_API_MOCKING__` define are not loaded
 * here. Only the `@` alias matters, and it comes from `paths` in tsconfig.app.json through
 * Vite 8's native `resolve.tsconfigPaths` — the same setting the build uses, so there is no
 * second copy to keep in sync.
 */
export default defineConfig({
  resolve: { tsconfigPaths: true },
  test: {
    // The polling manager reads `document.visibilityState` through vueuse, so the suite
    // needs a DOM rather than a bare node environment.
    environment: 'jsdom',
    include: ['src/**/*.spec.ts'],
  },
})
