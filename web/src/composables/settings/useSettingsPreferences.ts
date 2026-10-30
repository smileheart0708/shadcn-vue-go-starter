import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { useCurrentViewerPolling } from '@/composables/auth/useCurrentViewerPolling'
import {
  normalizePollingIntervalSeconds,
  usePollingStore,
} from '@/stores/polling'
import { useLocaleStore } from '@/stores/locale'
import { useThemeStore } from '@/stores/theme'
import { isSupportedLocale } from '@/plugins/i18n/locales'
import {
  getLocaleOptions,
  isThemePreference,
} from '@/utils/settings/preferences'

export function useSettingsPreferences() {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  const localeStore = useLocaleStore()
  const pollingStore = usePollingStore()

  const pollingIntervalSliderValue = ref([
    pollingStore.currentUserIntervalSeconds,
  ])
  const localeOptions = getLocaleOptions()
  const currentUserPollingIntervalSeconds = computed(() =>
    normalizePollingIntervalSeconds(
      pollingIntervalSliderValue.value[0] ??
        pollingStore.currentUserIntervalSeconds,
    ),
  )

  // The same ref-counted task the app shell runs, so this page only observes it instead of
  // starting a second timer.
  const currentUserPolling = useCurrentViewerPolling()

  // The shared task only polls on its interval, so ask for one immediate refresh when this
  // page mounts: it is the only place that surfaces a failing viewer refresh. Ordering
  // with `usePollingTask`'s own `onMounted` does not matter — the manager keeps a pending
  // request until the task starts.
  onMounted(() => {
    currentUserPolling.refreshNow()
  })

  watch(
    () => pollingStore.currentUserIntervalSeconds,
    (seconds) => {
      pollingIntervalSliderValue.value = [seconds]
    },
    { immediate: true },
  )

  watch(
    () => currentUserPolling.consecutiveFailures.value,
    (failures) => {
      // Announce only the first failure of a streak. The task keeps retrying with
      // backoff, so reacting to `error` (a new object per attempt) would stack several
      // toasts for a single network blip.
      if (failures !== 1) {
        return
      }

      toast.error(t('common.feedback.networkError'))
    },
  )

  function handleThemeChange(value: unknown) {
    if (isThemePreference(value)) {
      themeStore.setTheme(value)
    }
  }

  function handleLocaleChange(value: unknown) {
    if (isSupportedLocale(value)) {
      localeStore.setLocale(value)
    }
  }

  function handlePollingIntervalSliderChange(value: number[] | undefined) {
    if (!Array.isArray(value) || value.length === 0) {
      return
    }

    const sliderValue = value[0]
    pollingIntervalSliderValue.value = [
      normalizePollingIntervalSeconds(sliderValue),
    ]
  }

  function commitPollingInterval(value: number[] | undefined) {
    if (!Array.isArray(value) || value.length === 0) {
      return
    }

    const sliderValue = value[0]
    const nextSeconds = normalizePollingIntervalSeconds(sliderValue)
    pollingIntervalSliderValue.value = [nextSeconds]

    if (nextSeconds === pollingStore.currentUserIntervalSeconds) {
      return
    }

    pollingStore.setCurrentUserIntervalSeconds(nextSeconds)
    // No pause/resume dance here: the manager watches `intervalMs` and re-arms the next
    // tick itself, so a manual restart would only add a redundant request.
  }

  return {
    theme: computed(() => themeStore.theme),
    locale: computed(() => localeStore.locale),
    localeOptions,
    pollingIntervalSliderValue,
    currentUserPollingIntervalSeconds,
    handleThemeChange,
    handleLocaleChange,
    handlePollingIntervalSliderChange,
    commitPollingInterval,
  }
}
