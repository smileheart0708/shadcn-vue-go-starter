/**
 * The single source of the AppShell's wide/narrow split.
 *
 * Wide means 840px and up, narrow means below 840px — a viewport decision, not a device
 * class: a tablet is wide in landscape and narrow in portrait. `wide:` (declared as
 * `--breakpoint-wide` in `style.css`) compiles to `(min-width: 840px)`, so the runtime
 * check evaluates exactly that query and derives narrow by negating it. Do not declare a
 * second `(max-width: 839px)` pair for the narrow side: two hand-written thresholds drift
 * apart, and this particular pair leaves a sub-pixel dead zone (`839 < width < 840`,
 * which browser zoom does produce) where neither the fixed bar nor the sheet matches.
 * One query cannot have a gap with its own negation.
 *
 * `--breakpoint-wide` and `WIDE_LAYOUT_MIN_WIDTH_PX` cannot read each other —
 * Tailwind cannot read TypeScript and `matchMedia` cannot read the theme — so
 * `breakpoints.spec.ts` asserts they match. Change both together.
 */
export const WIDE_LAYOUT_MIN_WIDTH_PX = 840

/** The `wide:` variant as a query `matchMedia` can evaluate. */
export const WIDE_LAYOUT_MEDIA_QUERY = `(min-width: ${String(WIDE_LAYOUT_MIN_WIDTH_PX)}px)`
