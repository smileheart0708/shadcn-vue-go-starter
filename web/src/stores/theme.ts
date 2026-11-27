import { useMediaQuery, useStorage } from '@vueuse/core'
import { computed, watch } from 'vue'
import { defineStore } from 'pinia'
import { z } from 'zod'
import { parseExternalValueOrDefault } from '@/lib/external-input'
import { prefersReducedMotion } from '@/utils/motion'

// index.html 的首帧脚本要在这个模块执行之前独立读出同一个键，改名必须同步那里；
// theme.spec.ts 会读 index.html 校验，漏改会让测试失败而不是静默白闪。
export const THEME_STORAGE_KEY = 'app.theme'
// 同理：index.html 的首帧脚本无法 import 本模块，query 字符串在那里重复一份，
// theme.spec.ts 会读 index.html 校验，改名必须同步两处。
export const SYSTEM_THEME_MEDIA_QUERY = '(prefers-color-scheme: dark)'

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

// 只切 `.dark` 类，`color-scheme` 由 style.css 的 `:root` / `.dark` 规则跟随：
// index.html 的首帧脚本也走同一条路径，两处各写一份内联样式会多出一个真值。
function applyTheme(theme: ResolvedTheme) {
  document.documentElement.classList.toggle(
    'dark',
    theme === THEME_OPTIONS.dark,
  )
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
  const theme = useStorage<ThemePreference>(
    THEME_STORAGE_KEY,
    THEME_OPTIONS.system,
    undefined,
    {
      // The old store only wrote after an explicit choice, and index.html treats a missing key
      // exactly like 'system'; do not create the key on first visit.
      writeDefaults: false,
      // Persist synchronously like the old persistTheme, inside the View Transition callback.
      flush: 'sync',
      // Keep the current single-tab semantics; cross-tab propagation is a separate decision.
      listenToStorageChanges: false,
      serializer: {
        // An unknown stored value must fall back to 'system'; the default string read would
        // pass it through and it would become the resolved theme.
        read: (raw) =>
          parseExternalValueOrDefault(
            themePreferenceSchema,
            raw,
            THEME_OPTIONS.system,
          ),
        write: (value) => value,
      },
    },
  )
  const systemPrefersDark = useMediaQuery(SYSTEM_THEME_MEDIA_QUERY)

  const resolvedTheme = computed<ResolvedTheme>(() =>
    theme.value === THEME_OPTIONS.system
      ? systemPrefersDark.value
        ? THEME_OPTIONS.dark
        : THEME_OPTIONS.light
      : theme.value,
  )

  // Sync so the class lands inside the View Transition callback, not one flush later.
  watch(resolvedTheme, applyTheme, { immediate: true, flush: 'sync' })

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
      pendingTheme = null
    }

    // 减少动效偏好下直接落地：style.css 的时长归零管不到 `::view-transition-*`
    // 这棵浏览器伪元素树，整页快照的交叉淡入仍会完整播放。
    if (!('startViewTransition' in document) || prefersReducedMotion()) {
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

  return { theme, resolvedTheme, setTheme }
})
