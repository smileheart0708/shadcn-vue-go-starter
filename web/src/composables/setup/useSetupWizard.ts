import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { resolveAPIErrorMessage } from '@/lib/api/error-messages'
import {
  completeSetup,
  getInstallState,
  type DatabaseDriver,
} from '@/lib/api/setup'
import { useAuthStore } from '@/stores/auth'
import { toSetupDatabaseInput } from '@/composables/setup/database-input'
import { createSetupSteps } from '@/composables/setup/setupSteps'
import type {
  AdminDraft,
  SetupDatabaseDraft,
  SetupStepId,
  SetupWizard,
  TransitionDirection,
} from '@/composables/setup/types'

const MOTION_SAFE_RESET =
  'motion-reduce:translate-x-0 motion-reduce:opacity-100'
const TRANSITION_REST = `translate-x-0 opacity-100 ${MOTION_SAFE_RESET}`

export function useSetupWizard(): SetupWizard {
  const { t } = useI18n()
  const router = useRouter()
  const authStore = useAuthStore()

  const steps = createSetupSteps(t)
  const totalSteps = steps.length
  const currentStepId = ref<SetupStepId>('admin')
  const transitionDirection = ref<TransitionDirection>('forward')
  const submitting = ref(false)
  const autoLoginFailed = ref(false)

  const adminDraft = ref<AdminDraft>({
    username: '',
    password: '',
  })
  const databaseDraft = ref<SetupDatabaseDraft>({
    driver: 'sqlite',
    postgresql: {
      host: 'localhost',
      port: 5432,
      database: '',
      username: '',
      password: '',
      sslMode: 'require',
    },
  })

  const currentStepIndex = computed(() =>
    steps.findIndex((step) => step.id === currentStepId.value),
  )
  const currentStep = computed(() => steps[currentStepIndex.value])
  const setupCompleted = computed(() => authStore.isSetupComplete)
  const finalizationRequired = computed(
    () => authStore.installState?.phase === 'finalization_required',
  )
  const finalizationDriver = computed<DatabaseDriver | null>(() => {
    if (!finalizationRequired.value) {
      return null
    }
    return authStore.installState?.databaseDriver ?? null
  })

  const stepperValue = computed<number>({
    get: () => currentStepIndex.value + 1,
    set: (value) => {
      if (!isStepSelectable(value)) {
        return
      }
      transitionDirection.value =
        value > currentStepIndex.value ? 'forward' : 'backward'
      currentStepId.value = steps[value - 1].id
    },
  })

  const transitionClasses = computed(() => {
    const goingBackward = transitionDirection.value === 'backward'
    const enterOffset = goingBackward ? '-translate-x-8' : 'translate-x-8'
    const leaveOffset = goingBackward ? 'translate-x-8' : '-translate-x-8'

    return {
      enterActive:
        'transition duration-300 ease-out motion-reduce:transition-none',
      enterFrom: `${enterOffset} opacity-0 ${MOTION_SAFE_RESET}`,
      enterTo: TRANSITION_REST,
      leaveActive:
        'transition duration-200 ease-in motion-reduce:transition-none',
      leaveFrom: TRANSITION_REST,
      leaveTo: `${leaveOffset} opacity-0 ${MOTION_SAFE_RESET}`,
    }
  })

  function isStepSelectable(stepNumber: number) {
    if (stepNumber < 1 || stepNumber > totalSteps) {
      return false
    }
    if (setupCompleted.value) {
      return stepNumber === totalSteps
    }
    return stepNumber <= currentStepIndex.value + 1
  }

  function goTo(stepId: SetupStepId) {
    const targetIndex = steps.findIndex((step) => step.id === stepId)
    if (targetIndex < 0) {
      return
    }
    transitionDirection.value =
      targetIndex > currentStepIndex.value ? 'forward' : 'backward'
    currentStepId.value = stepId
  }

  function completeAdmin(draft: AdminDraft) {
    if (submitting.value || setupCompleted.value) {
      return
    }
    adminDraft.value = draft
    goTo('database')
  }

  async function enterDoneStep() {
    try {
      await authStore.login({
        identifier: adminDraft.value.username,
        password: adminDraft.value.password,
      })
    } catch {
      autoLoginFailed.value = true
    }

    adminDraft.value.password = ''
    goTo('done')
    void authStore.refreshPublicState().catch(() => undefined)
  }

  async function completeDatabase() {
    if (submitting.value || setupCompleted.value) {
      return
    }

    submitting.value = true
    try {
      const state = await completeSetup({
        username: adminDraft.value.username,
        password: adminDraft.value.password,
        database: toSetupDatabaseInput(
          databaseDraft.value,
          finalizationRequired.value,
        ),
      })
      authStore.applyInstallState(state)
      await enterDoneStep()
    } catch (error: unknown) {
      // A request can fail after the server committed the owner or persisted the
      // selection; re-read the phase so the wizard cannot stay in a mode the
      // server has already left.
      try {
        authStore.applyInstallState(await getInstallState())
      } catch {
        toast.error(resolveAPIErrorMessage(t, error) ?? t('setup.failed'))
        return
      }
      if (authStore.isSetupComplete) {
        await enterDoneStep()
        return
      }
      toast.error(resolveAPIErrorMessage(t, error) ?? t('setup.failed'))
    } finally {
      submitting.value = false
    }
  }

  function back() {
    if (submitting.value || currentStepIndex.value <= 0) {
      return
    }
    transitionDirection.value = 'backward'
    currentStepId.value = steps[currentStepIndex.value - 1].id
  }

  async function finish() {
    await router.replace(
      authStore.isAuthenticated ? { name: 'dashboard' } : { name: 'login' },
    )
  }

  return {
    steps,
    currentStep,
    stepperValue,
    transitionClasses,
    submitting,
    finalizationRequired,
    finalizationDriver,
    autoLoginFailed,
    adminDraft,
    databaseDraft,
    completeAdmin,
    completeDatabase,
    back,
    finish,
    isStepSelectable,
  }
}
