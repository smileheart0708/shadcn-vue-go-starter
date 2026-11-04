import { describe, expect, it } from 'vitest'
import { toSetupDatabaseInput } from '@/composables/setup/database-input'
import type { SetupDatabaseDraft } from '@/composables/setup/types'

const postgresDraft: SetupDatabaseDraft = {
  driver: 'postgresql',
  postgresql: {
    host: ' db.internal ',
    port: 5432,
    database: ' starter ',
    username: ' app ',
    password: ' p@ss ',
    sslMode: 'verify-full',
  },
}

describe('toSetupDatabaseInput', () => {
  it('omits the database when finalization already has one', () => {
    expect(toSetupDatabaseInput(postgresDraft, true)).toBeUndefined()
  })

  it('returns a SQLite selection without connection secrets', () => {
    expect(
      toSetupDatabaseInput({ ...postgresDraft, driver: 'sqlite' }, false),
    ).toEqual({ driver: 'sqlite' })
  })

  it('trims PostgreSQL fields but preserves the password verbatim', () => {
    expect(toSetupDatabaseInput(postgresDraft, false)).toEqual({
      driver: 'postgresql',
      postgresql: {
        host: 'db.internal',
        port: 5432,
        database: 'starter',
        username: 'app',
        password: ' p@ss ',
        sslMode: 'verify-full',
      },
    })
  })
})
