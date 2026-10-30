import { defineConfig } from 'vitest/config'
// The extension is required: Vite's native config loader cannot resolve an extensionless
// import from a config file.
import { resolveAlias } from './build/alias.ts'

/**
 * Separate from `vite.config.ts` on purpose: the suite covers pure logic and state machines
 * (see AGENTS.md), so the app's plugins and the `__VITE_API_MOCKING__` define are not loaded
 * here. Only the `@` alias is shared — a spec that imports app components or `mocks/` modules
 * would need the rest, and that is the moment to switch to a real component-test setup.
 */
export default defineConfig({
  resolve: { alias: resolveAlias },
  test: {
    // The polling manager reads `document.visibilityState` through vueuse, so the suite
    // needs a DOM rather than a bare node environment.
    environment: 'jsdom',
    include: ['src/**/*.spec.ts'],
  },
})
