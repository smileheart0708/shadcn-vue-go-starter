/**
 * Demo fixtures for the Tasks page.
 *
 * This data is intentionally static — TasksView is a UI showcase shipped with the
 * starter, not a real feature. Replace this module with an API call (see
 * `lib/api/admin-users.ts` for the established pattern: zod-validated envelope + a
 * `composables/` wrapper) when wiring up a real task backend.
 */

// These unions are rendered through `tasks.*` i18n keys, so keep both sides in sync.
export type TaskLabel = 'documentation' | 'bug' | 'feature'
export type TaskStatus = 'backlog' | 'todo' | 'inProgress' | 'done' | 'canceled'
export type TaskPriority = 'low' | 'medium' | 'high'

export interface TaskRow {
  id: string
  label: TaskLabel
  title: string
  status: TaskStatus
  priority: TaskPriority
}

export const demoTasks: readonly TaskRow[] = [
  {
    id: 'TASK-8782',
    label: 'documentation',
    title:
      "You can't compress the program without quantifying the open-source SSD pixel!",
    status: 'inProgress',
    priority: 'medium',
  },
  {
    id: 'TASK-7878',
    label: 'documentation',
    title:
      'Try to calculate the EXE feed, maybe it will index the multi-byte pixel!',
    status: 'backlog',
    priority: 'medium',
  },
  {
    id: 'TASK-7839',
    label: 'bug',
    title: 'We need to bypass the neural TCP card!',
    status: 'todo',
    priority: 'high',
  },
  {
    id: 'TASK-5562',
    label: 'feature',
    title:
      'The SAS interface is down, bypass the open-source pixel so we can back up the PNG bandwidth!',
    status: 'backlog',
    priority: 'medium',
  },
  {
    id: 'TASK-8686',
    label: 'feature',
    title:
      "I'll parse the wireless SSL protocol, that should driver the API panel!",
    status: 'canceled',
    priority: 'medium',
  },
  {
    id: 'TASK-1280',
    label: 'bug',
    title:
      'Use the digital TLS panel, then you can transmit the haptic system!',
    status: 'done',
    priority: 'high',
  },
  {
    id: 'TASK-7262',
    label: 'feature',
    title:
      'The UTF8 application is down, parse the neural bandwidth so we can back up the PNG firewall!',
    status: 'done',
    priority: 'high',
  },
  {
    id: 'TASK-1138',
    label: 'feature',
    title:
      "Generating the driver won't do anything, we need to quantify the 1080p SMTP bandwidth!",
    status: 'inProgress',
    priority: 'medium',
  },
  {
    id: 'TASK-7184',
    label: 'feature',
    title: 'We need to program the back-end THX pixel!',
    status: 'todo',
    priority: 'low',
  },
  {
    id: 'TASK-5160',
    label: 'documentation',
    title:
      "Calculating the bus won't do anything, we need to navigate the back-end JSON protocol!",
    status: 'inProgress',
    priority: 'high',
  },
]
