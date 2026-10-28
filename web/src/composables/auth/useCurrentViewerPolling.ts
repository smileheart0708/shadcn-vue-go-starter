import { usePollingTask } from '@/composables/usePollingTask'
import { useAuthStore } from '@/stores/auth'
import { usePollingStore } from '@/stores/polling'

/**
 * Registry key for the viewer refresh task.
 *
 * `acquirePollingTask` de-duplicates by this key and ref-counts consumers, so the app
 * shell and the settings page can both acquire it and still drive a single timer.
 */
const CURRENT_VIEWER_POLLING_KEY = 'auth.current-user'

/**
 * Keeps the signed-in viewer (identity, role, capabilities) fresh in the background.
 *
 * Mount this at the authenticated shell level, not on an individual page: it is what
 * makes a server-side change — account disabled, role revoked — take effect without a
 * reload, so it must keep running wherever the user navigates.
 *
 * Requests are flagged as background work so they do not flash the route loading bar,
 * and the first run happens after one interval — the router guard has just fetched the
 * viewer, so refreshing immediately would only repeat that request.
 */
export function useCurrentViewerPolling() {
  const authStore = useAuthStore()
  const pollingStore = usePollingStore()

  return usePollingTask({
    key: CURRENT_VIEWER_POLLING_KEY,
    intervalMs: () => pollingStore.currentUserIntervalMs,
    enabled: () => authStore.isAuthenticated,
    fetch: async ({ signal }) =>
      authStore.fetchViewer({ signal, backgroundRequest: true }),
    apply: (viewer) => {
      authStore.applyViewer(viewer)
    },
  })
}
