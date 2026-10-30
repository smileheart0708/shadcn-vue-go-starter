import type { Capability } from '@/lib/auth/capabilities.generated'

/**
 * The single frontend implementation of "may this viewer do X".
 *
 * Everyone that needs a permission answer — the route guard, the sidebar
 * navigation, view-level buttons, and the auth store — funnels through here, so
 * the frontend cannot develop two subtly different definitions of "allowed".
 * This mirrors the server, which remains the real boundary.
 */
export function hasCapability(
  granted: readonly Capability[],
  required: Capability,
): boolean {
  return granted.includes(required)
}

export function hasAllCapabilities(
  granted: readonly Capability[],
  required: readonly Capability[],
): boolean {
  return required.every((capability) => granted.includes(capability))
}
