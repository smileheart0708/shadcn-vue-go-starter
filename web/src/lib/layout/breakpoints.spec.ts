import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import {
  WIDE_LAYOUT_MEDIA_QUERY,
  WIDE_LAYOUT_MIN_WIDTH_PX,
} from '@/lib/layout/breakpoints'

// The stylesheet has to be read from disk: Vitest answers a `?raw` CSS import with an
// empty module while `css` is disabled. The path is derived from this module's own
// location rather than `process.cwd()`, because a spec that only works when the runner
// happens to start in `web/` fails at collection time (ENOENT) instead of on drift.
// Do not pass the result of `new URL(relative, import.meta.url)` to `readFileSync`
// either: in this jsdom environment the `URL` global does not keep the `file:` base —
// `import.meta.url` reads as `file:///…`, but the constructed URL comes out as
// `http://localhost:3000/src/style.css` and `readFileSync` rejects it. Deriving the
// path from `fileURLToPath(import.meta.url)` is the form that actually works.
const styleSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../../style.css'),
  'utf8',
)

describe('layout breakpoints', () => {
  it('keeps style.css and the matchMedia query on one threshold', () => {
    const cssBreakpoint = readWideBreakpoint(styleSource)

    expect(cssBreakpoint).toBe(WIDE_LAYOUT_MIN_WIDTH_PX)
    // The expected query is built from the stylesheet, not from
    // `WIDE_LAYOUT_MIN_WIDTH_PX`: a hand-written `(max-width: ...)` counterpart — which
    // leaves a sub-pixel dead zone where neither the fixed bar nor the sheet renders —
    // fails here instead of silently drifting away from the `wide:` variant.
    expect(WIDE_LAYOUT_MEDIA_QUERY).toBe(
      `(min-width: ${String(cssBreakpoint)}px)`,
    )
  })
})

function readWideBreakpoint(source: string): number {
  // Anchored to a declaration line and required to be unique: an explanatory comment
  // quoting the old value must not be able to satisfy (or break) this assertion.
  const declarations = [
    ...source.matchAll(/^[ \t]*--breakpoint-wide:\s*(\d+)px/gm),
  ]
  const value = declarations.length === 1 ? declarations[0]?.[1] : undefined
  if (value === undefined) {
    throw new Error(
      'style.css must declare --breakpoint-wide in px exactly once',
    )
  }

  return Number(value)
}
