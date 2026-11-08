import { describe, expect, it } from 'vitest'
import enUSMessages from '@/locales/en-US.json'
import zhCNMessages from '@/locales/zh-CN.json'

/**
 * Two language-pack invariants no other gate can see:
 * - `MessageSchema` and `no-missing-keys-in-other-locales` compare which keys exist, not
 *   their order, so the files can silently stop being diffable side by side (the reason
 *   AGENTS.md asks for identical order).
 * - `no-unused-keys` treats every `@:` target in the pack as used, so a typo'd link target
 *   passes `pnpm lint` + `vue-tsc` and only shows up at runtime as a raw key on screen.
 * Values are intentionally not compared: each locale owns its own copy.
 */
describe('locale messages', () => {
  it('keeps en-US keys in the same order as the zh-CN source', () => {
    expect(walkLocale(enUSMessages).keyPaths).toEqual(
      walkLocale(zhCNMessages).keyPaths,
    )
  })

  it('resolves every linked message inside its own pack', () => {
    expect(unresolvedLinks(zhCNMessages)).toEqual([])
    expect(unresolvedLinks(enUSMessages)).toEqual([])
  })
})

// Linked messages look like `@:key`, `@.lower:key` or `@:key(args)`; only the target key is
// compared here.
const LINK_PATTERN = /@(?:\.[A-Za-z]+)*:([A-Za-z0-9_.]+)/g

type JsonObject = Record<string, unknown>

interface LocaleLeaf {
  readonly keyPath: string
  readonly message: string
}

interface LocaleWalk {
  /** Dot paths of every key in file order, intermediate objects included. */
  readonly keyPaths: string[]
  /** Raw message of every leaf string. */
  readonly leaves: LocaleLeaf[]
}

function walkLocale(messages: unknown): LocaleWalk {
  const walk: LocaleWalk = { keyPaths: [], leaves: [] }
  appendKeyPaths(messages, '', walk)
  return walk
}

function appendKeyPaths(
  value: unknown,
  prefix: string,
  walk: LocaleWalk,
): void {
  if (!isJsonObject(value)) {
    return
  }

  for (const [key, child] of Object.entries(value)) {
    const keyPath = prefix === '' ? key : `${prefix}.${key}`
    walk.keyPaths.push(keyPath)
    if (typeof child === 'string') {
      walk.leaves.push({ keyPath, message: child })
      continue
    }

    appendKeyPaths(child, keyPath, walk)
  }
}

function isJsonObject(value: unknown): value is JsonObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function unresolvedLinks(messages: unknown): string[] {
  const { keyPaths, leaves } = walkLocale(messages)
  const knownPaths = new Set(keyPaths)
  const unresolved: string[] = []

  for (const leaf of leaves) {
    for (const match of leaf.message.matchAll(LINK_PATTERN)) {
      // `at(1)` instead of `match[1]` keeps the possible `undefined` explicit, which the
      // strict index rules in this repo would otherwise flag as an unnecessary check.
      const target = match.at(1)
      if (target !== undefined && !knownPaths.has(target)) {
        unresolved.push(`${leaf.keyPath} -> @:${target}`)
      }
    }
  }

  return unresolved
}
