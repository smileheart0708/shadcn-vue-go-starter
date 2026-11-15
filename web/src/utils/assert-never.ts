/**
 * Compile-time exhaustiveness guard for `switch` statements over a closed union.
 *
 * The `never` parameter is the check: if a new member is added to the union and not handled,
 * the value reaching the default branch is no longer assignable to `never` and the build fails
 * here instead of the code silently falling through to a wrong default at runtime.
 */
export function assertNever(value: never): never {
  throw new Error(`unhandled union member: ${String(value)}`)
}
