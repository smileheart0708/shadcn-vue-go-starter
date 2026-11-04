import type { SetupDatabaseInput } from '@/lib/api/setup'
import type { SetupDatabaseDraft } from '@/composables/setup/types'

export function toSetupDatabaseInput(
  draft: SetupDatabaseDraft,
  finalizationRequired: boolean,
): SetupDatabaseInput | undefined {
  if (finalizationRequired) {
    return undefined
  }
  if (draft.driver === 'sqlite') {
    return { driver: 'sqlite' }
  }
  return {
    driver: 'postgresql',
    postgresql: {
      host: draft.postgresql.host.trim(),
      port: draft.postgresql.port,
      database: draft.postgresql.database.trim(),
      username: draft.postgresql.username.trim(),
      password: draft.postgresql.password,
      sslMode: draft.postgresql.sslMode,
    },
  }
}
