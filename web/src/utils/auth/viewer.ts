import type { Viewer } from '@/lib/api/auth'

/**
 * Structural equality for the payload of `/api/auth/me`.
 *
 * Polling hands back a freshly parsed object every tick even when nothing changed, and
 * swapping the store ref unconditionally would invalidate every viewer-derived computed and
 * re-render the shell once per interval (it would also make a future `watch(viewer)` fire on
 * every tick). Keeping the reference stable while the payload is unchanged is the contract.
 *
 * The comparison is deliberately derived from the payload instead of a hand-written field
 * list: `viewerSchema` is the single source of truth for the shape, and a field list that is
 * not derived from it silently drops updates the moment the schema grows. Both sides come out
 * of the same schema, so their key order is identical and the serialized form is comparable.
 */
export function isSameViewer(current: Viewer | null, next: Viewer | null) {
  return JSON.stringify(current) === JSON.stringify(next)
}
