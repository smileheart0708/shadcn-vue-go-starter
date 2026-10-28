import { fileURLToPath, URL } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { msw } from 'msw/vite'
import { defineConfig, loadEnv } from 'vite'
import { compression, defineAlgorithm } from 'vite-plugin-compression2'
import vueDevTools from 'vite-plugin-vue-devtools'

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
      // Serves `mockServiceWorker.js` from memory in `vite dev` and emits it at build time
      // only for bundles that import the `virtual:msw` module. Mocking is dev-only (see
      // `main.ts`), so the worker never lands in `dist` or the embedded Go binary.
      msw(),
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
