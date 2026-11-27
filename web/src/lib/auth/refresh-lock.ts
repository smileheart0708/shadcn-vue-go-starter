const REFRESH_LOCK_NAME = 'auth-refresh'

/**
 * Serializes refresh requests across tabs.
 *
 * The refresh cookie is shared by every tab, so two tabs rotating it at once make the
 * loser look like a token-reuse attack and the server revokes the whole session. Web
 * Locks is origin-wide, which is exactly the scope of that cookie. The fallback keeps
 * environments without it (older browsers, non-secure contexts) on the old behavior.
 */
export async function runWithRefreshLock<T>(
  task: () => Promise<T>,
): Promise<T> {
  const locks = typeof navigator === 'undefined' ? undefined : navigator.locks
  if (locks === undefined) {
    return task()
  }

  return locks.request(REFRESH_LOCK_NAME, task)
}
