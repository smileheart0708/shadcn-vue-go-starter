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

  // Acquires the same ref-counted task the app shell already runs, so that changing the
  // interval here can restart it immediately instead of waiting out the current tick.
  const currentUserPolling = useCurrentViewerPolling()

  // The shared task only polls on its interval, so ask for one immediate refresh when this
  // page mounts: it is the only place that surfaces a failing viewer refresh.
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
    () => currentUserPolling.error.value,
    (error) => {
      if (error === null) {
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

    if (!currentUserPolling.running.value) {
      return
    }

    currentUserPolling.pause()
    currentUserPolling.resume()
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
