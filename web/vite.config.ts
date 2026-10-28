import { rm } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import type { Plugin, ResolvedConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig, loadEnv } from 'vite'
import { compression, defineAlgorithm } from 'vite-plugin-compression2'
import vueDevTools from 'vite-plugin-vue-devtools'

const MOCK_SERVICE_WORKER_FILE = 'mockServiceWorker.js'

/**
 * Drops the MSW service worker from the production bundle.
 *
 * `msw init` writes the worker into `public/`, which Vite copies verbatim on every
 * build. Mocking is dev-only (see `main.ts`), so shipping it would embed a request
 * interceptor into the Go binary via `//go:embed web/dist` and expose it publicly.
 */
function excludeMockServiceWorker(): Plugin {
  // Vite keeps `build.outDir` as configured (`dist` by default) and resolves it against
  // `root` when it builds, so resolve it the same way here. Hardcoding `<config dir>/dist/`
  // ignored both `--outDir` and a custom `root`, leaving the worker in the real output dir.
  let outDir = ''

  return {
    name: 'exclude-mock-service-worker',
    apply: 'build',
    configResolved(config: ResolvedConfig) {
      outDir = resolve(config.root, config.build.outDir)
    },
    // Runs in `closeBundle`, not `generateBundle`: files under `public/` are copied
    // straight to the output dir and never enter the Rollup bundle graph, so they can
    // only be removed after the copy completes. `.gz` covers the compression plugin's
    // output, which is produced from the copied file.
    async closeBundle() {
      await Promise.all(
        [MOCK_SERVICE_WORKER_FILE, `${MOCK_SERVICE_WORKER_FILE}.gz`].map(
          async (fileName) => rm(join(outDir, fileName), { force: true }),
        ),
      )
    },
  }
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    define: {
      __VITE_API_MOCKING__: env.VITE_API_MOCKING === 'true',
    },
    plugins: [
      tailwindcss(),
      vueDevTools(),
      vue(),
      excludeMockServiceWorker(),
      compression({
        include: [/\.(css|html|js|json|map|svg|txt|xml)$/],
        algorithms: [defineAlgorithm('gzip', { level: 9 })],
        skipIfLargerOrEqual: true,
      }),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: { proxy: { '/api': { target: 'http://localhost:8080' } } },
    build: { target: 'es2022' },
  }
})
