import { describe, expect, it } from 'vitest'
import { isInternalRedirectPath } from '@/utils/auth/redirect'

function expectPaths(paths: string[], expected: boolean) {
  for (const value of paths) {
    expect(isInternalRedirectPath(value), value).toBe(expected)
  }
}

describe('isInternalRedirectPath', () => {
  it('accepts absolute in-app paths', () => {
    expectPaths(
      [
        '/',
        '/dashboard',
        '/dashboard?q=1#section',
        '/javascript:alert(1)',
        '/%2f%2fevil.com',
        '/%5cevil.com',
        '/dashboard?next=//evil.com',
        '/dashboard\\evil.com',
        '/dashboard?q=\\evil.com',
      ],
      true,
    )
  })

  it('rejects relative paths and empty values', () => {
    expectPaths(['', 'dashboard', '../dashboard'], false)
  })

  it('rejects external, script and protocol-relative URLs', () => {
    expectPaths(
      [
        '//evil.com',
        '///evil.com',
        '//app.example/dashboard',
        '//[',
        'https://evil.com',
        'javascript:alert(1)',
      ],
      false,
    )
  })

  it('rejects URL-parser normalization bypasses', () => {
    expectPaths(
      [
        '/\\evil.com',
        '\\evil.com',
        '/\t/evil.com',
        '/\n/evil.com',
        '/\t\\evil.com',
      ],
      false,
    )
  })

  it('rejects same-origin protocol-relative URLs after normalization', () => {
    expectPaths(
      [
        '/\\app.example/dashboard',
        '/\\app.example',
        '/\\evil.com@app.example',
        '/\\/app.example/dashboard',
        '/\t/app.example/dashboard',
        '/\n/app.example/dashboard',
        '/\r/app.example/dashboard',
        '/\t\\app.example/dashboard',
        '/\\app.example:443/dashboard',
      ],
      false,
    )
  })
})
