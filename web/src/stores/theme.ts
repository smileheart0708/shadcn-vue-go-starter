import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { z } from 'zod'
import { parseExternalValueOrDefault } from '@/lib/external-input'

const THEME_STORAGE_KEY = 'app.theme'
const SYSTEM_THEME_MEDIA_QUERY = '(prefers-color-scheme: dark)'

// TS 与 style.css 不能共享标识符，这组名字是二者唯一的契约；theme.spec.ts 会读
// style.css 校验，改名必须两处同步。
export const THEME_REVEAL_CSS = {
  attribute: 'data-theme-reveal',
  center: '--theme-reveal-center',
  radius: '--theme-reveal-radius',
} as const

const THEME_OPTIONS = {
  light: 'light',
  dark: 'dark',
  system: 'system',
} as const
const themePreferenceSchema = z.enum([
  THEME_OPTIONS.light,
  THEME_OPTIONS.dark,
  THEME_OPTIONS.system,
])

export type ThemePreference = z.infer<typeof themePreferenceSchema>
type ResolvedTheme = Exclude<ThemePreference, 'system'>

function readStoredTheme(): ThemePreference {
  if (typeof window === 'undefined') {
    return THEME_OPTIONS.system
  }

  const storedTheme = window.localStorage.getItem(THEME_STORAGE_KEY)
  return parseExternalValueOrDefault(
    themePreferenceSchema,
    storedTheme,
    THEME_OPTIONS.system,
  )
}

function getSystemTheme(): ResolvedTheme {
  return window.matchMedia(SYSTEM_THEME_MEDIA_QUERY).matches
    ? THEME_OPTIONS.dark
    : THEME_OPTIONS.light
}

function applyTheme(theme: ResolvedTheme) {
  document.documentElement.classList.toggle(
    'dark',
    theme === THEME_OPTIONS.dark,
  )
  document.documentElement.style.colorScheme = theme
}

// 揭示开关只在启动新过渡前被重写或移除，永远不要在过渡运行中清理（例如挂到
// transition.finished 上）：过渡被 skip 时 finished 会立刻 resolve，先于新过渡播完，
// 移除属性会把正在播放的揭示动画打断成瞬间切换。没有开关时 style.css 不碰伪元素，
// 浏览器保留默认 cross-fade。
function prepareThemeReveal(origin: Element) {
  const rect = origin.getBoundingClientRect()
  const x = rect.left + rect.width / 2
  const y = rect.top + rect.height / 2
  const radius = Math.hypot(
    Math.max(x, window.innerWidth - x),
    Math.max(y, window.innerHeight - y),
  )
  const root = document.documentElement
  root.setAttribute(THEME_REVEAL_CSS.attribute, '')
  root.style.setProperty(
    THEME_REVEAL_CSS.center,
    `${String(x)}px ${String(y)}px`,
  )
  root.style.setProperty(THEME_REVEAL_CSS.radius, `${String(radius)}px`)
}

export const useThemeStore = defineStore('theme', () => {
  const initialized = ref(false)
  const theme = ref<ThemePreference>(THEME_OPTIONS.system)
  const systemTheme = ref<ResolvedTheme>(THEME_OPTIONS.light)

  const resolvedTheme = computed<ResolvedTheme>(() =>
    theme.value === THEME_OPTIONS.system ? systemTheme.value : theme.value,
  )

  let mediaQueryList: MediaQueryList | null = null

  function persistTheme(nextTheme: ThemePreference) {
    window.localStorage.setItem(THEME_STORAGE_KEY, nextTheme)
  }

  function syncResolvedTheme() {
    applyTheme(resolvedTheme.value)
  }

  let pendingTheme: ThemePreference | null = null

  function setTheme(nextTheme: ThemePreference, origin?: Element) {
    // reka-ui 的 Tabs 一次点击会 emit 两次 update:model-value；updateCallback 是异步的，
    // 只能靠同步的 pendingTheme 去重，否则第二次调用会把刚启动的过渡 skip 掉。
    if (nextTheme === (pendingTheme ?? theme.value)) {
      return
    }
    pendingTheme = nextTheme

    const apply = () => {
      theme.value = nextTheme
      persistTheme(nextTheme)
      syncResolvedTheme()
      pendingTheme = null
    }

    if (!('startViewTransition' in document)) {
      apply()
      return
    }

    if (origin === undefined) {
      document.documentElement.removeAttribute(THEME_REVEAL_CSS.attribute)
    } else {
      prepareThemeReveal(origin)
    }

    document.startViewTransition(apply)
  }

  function initialize() {
    if (initialized.value) {
      return
    }

    theme.value = readStoredTheme()
    systemTheme.value = getSystemTheme()
    syncResolvedTheme()

    mediaQueryList ??= window.matchMedia(SYSTEM_THEME_MEDIA_QUERY)
    mediaQueryList.addEventListener('change', (event) => {
      systemTheme.value = event.matches
        ? THEME_OPTIONS.dark
        : THEME_OPTIONS.light

      if (theme.value === THEME_OPTIONS.system) {
        syncResolvedTheme()
      }
    })

    initialized.value = true
  }

  return { initialized, theme, resolvedTheme, setTheme, initialize }
})
