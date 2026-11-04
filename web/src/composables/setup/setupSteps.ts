import { Check, Database, User } from '@lucide/vue'
import AdminAccountStep from '@/components/setup/AdminAccountStep.vue'
import DatabaseSelectionStep from '@/components/setup/DatabaseSelectionStep.vue'
import SetupDoneStep from '@/components/setup/SetupDoneStep.vue'
import type { SetupStepDefinition } from '@/composables/setup/types'

type Translate = (
  key: 'setup.steps.admin' | 'setup.steps.database' | 'setup.steps.done',
) => string

export function createSetupSteps(t: Translate): SetupStepDefinition[] {
  return [
    {
      id: 'admin',
      icon: User,
      component: AdminAccountStep,
      title: () => t('setup.steps.admin'),
    },
    {
      id: 'database',
      icon: Database,
      component: DatabaseSelectionStep,
      title: () => t('setup.steps.database'),
    },
    {
      id: 'done',
      icon: Check,
      component: SetupDoneStep,
      title: () => t('setup.steps.done'),
    },
  ]
}
