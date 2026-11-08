const PROBE_ORIGIN = 'https://redirect-probe.invalid'

/**
 * `?redirect=` is user-controlled: only absolute in-app paths are allowed.
 * WHATWG folds `\` to `/` and strips tab/LF/CR, so `/\app.example/x` looks in-app while the
 * address bar ends up doubled and the route falls into a catch-all NotFound. Resolving
 * against a sentinel origin rejects any value carrying its own host, without maintaining
 * WHATWG's normalization characters here.
 */
export function isInternalRedirectPath(value: string) {
  if (!value.startsWith('/')) {
    return false
  }

  try {
    return new URL(value, PROBE_ORIGIN).origin === PROBE_ORIGIN
  } catch {
    // Host parsing throws for values such as `//[`.
    return false
  }
}
