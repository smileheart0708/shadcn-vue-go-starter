import { accountHandlers } from '@/mocks/handlers/account'
import { authHandlers } from '@/mocks/handlers/auth'
import { installHandlers } from '@/mocks/handlers/install'
import { managementUsersHandlers } from '@/mocks/handlers/management-users'
import { systemLogsHandlers } from '@/mocks/handlers/system-logs'
import { systemSettingsHandlers } from '@/mocks/handlers/system-settings'

/**
 * Every domain contributes its own handler list; adding a backend domain means adding a
 * file next to these and one line here.
 */
export const handlers = [
  ...installHandlers,
  ...authHandlers,
  ...accountHandlers,
  ...systemSettingsHandlers,
  ...managementUsersHandlers,
  ...systemLogsHandlers,
]
