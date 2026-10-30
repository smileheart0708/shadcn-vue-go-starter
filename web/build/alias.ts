import { fileURLToPath, URL } from 'node:url'

/**
 * Single definition of the `@ -> src` alias, imported by both `vite.config.ts` (dev/build)
 * and `vitest.config.ts` (tests). Keeping it in its own module avoids two problems the
 * alternatives have: a second copy that silently drifts, and importing the whole app config
 * into the test run just to read one constant.
 *
 * `tsconfig.json` / `tsconfig.app.json` mirror this in `compilerOptions.paths`; TypeScript
 * cannot read it from here, so those two entries must be changed together with this one.
 */
export const resolveAlias = {
  '@': fileURLToPath(new URL('../src', import.meta.url)),
}
