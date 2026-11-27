import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import {
  SYSTEM_THEME_MEDIA_QUERY,
  THEME_REVEAL_CSS,
  THEME_STORAGE_KEY,
  useThemeStore,
} from '@/stores/theme'

// CSS 与 index.html 都无法从 TS 读取常量，只能由测试读文件校验；路径从本模块位置推导，见
// lib/layout/breakpoints.spec.ts 里对 jsdom 下 import.meta.url 的说明。
const styleSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../style.css'),
  'utf8',
)
const indexHtmlSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../../index.html'),
  'utf8',
)

function collectRevealCssNames(source: string): string[] {
  const names: string[] = []

  for (const match of source.matchAll(
    /--theme-reveal-[\w-]+|data-theme-reveal/g,
  )) {
    names.push(match[0])
  }

  return [...new Set(names)].sort()
}

function installViewTransition() {
  const runCallbacks: (() => void)[] = []
  const startViewTransition = vi.fn((callback: () => void) => {
    runCallbacks.push(callback)
    return { ready: Promise.resolve(), finished: Promise.resolve() }
  })

  Object.defineProperty(document, 'startViewTransition', {
    configurable: true,
    value: startViewTransition,
  })

  return { startViewTransition, runCallbacks }
}

type MediaChangeListener = (event: { matches: boolean }) => void

// jsdom 没有实现 matchMedia；store 的主题落地与「减少动效」判断都要经过它。
// 返回的函数模拟系统主题切换，只通知已注册的 change 监听。
function stubMatchMedia(initialMatches: boolean) {
  const listeners = new Set<MediaChangeListener>()
  let matches = initialMatches

  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: () => ({
      get matches() {
        return matches
      },
      addEventListener: (_event: string, listener: MediaChangeListener) => {
        listeners.add(listener)
      },
      removeEventListener: (_event: string, listener: MediaChangeListener) => {
        listeners.delete(listener)
      },
    }),
  })

  return (nextMatches: boolean) => {
    matches = nextMatches
    for (const listener of listeners) {
      listener({ matches })
    }
  }
}

describe('theme store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    window.localStorage.clear()
    document.documentElement.className = ''
    document.documentElement.removeAttribute(THEME_REVEAL_CSS.attribute)
    document.documentElement.removeAttribute('style')
    stubMatchMedia(false)
  })

  afterEach(() => {
    Reflect.deleteProperty(document, 'startViewTransition')
    Reflect.deleteProperty(window, 'matchMedia')
  })

  it('同一帧内重复设置同一主题只启动一次过渡', () => {
    const { startViewTransition, runCallbacks } = installViewTransition()
    const store = useThemeStore()

    // reka-ui 的 Tabs 一次点击会在 mousedown 与 focus 各 emit 一次；主题值要等
    // updateCallback 才落地，所以去重必须依赖同步状态，这里是它的回归防线。
    store.setTheme('dark')
    store.setTheme('dark')
    expect(startViewTransition).toHaveBeenCalledTimes(1)

    // 旧快照在过渡启动时拍摄，主题必须等到回调里才写到响应式状态。
    expect(store.theme).toBe('system')

    runCallbacks[0]?.()

    expect(store.theme).toBe('dark')
    expect(window.localStorage.getItem('app.theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('不传触发元素时移除揭示开关，回到浏览器默认过渡', () => {
    const { runCallbacks } = installViewTransition()
    const store = useThemeStore()
    document.documentElement.setAttribute(THEME_REVEAL_CSS.attribute, '')

    store.setTheme('light')
    runCallbacks[0]?.()

    expect(
      document.documentElement.hasAttribute(THEME_REVEAL_CSS.attribute),
    ).toBe(false)
  })

  it('传触发元素时以元素中心设置揭示圆，半径覆盖整个视口', () => {
    installViewTransition()
    const store = useThemeStore()
    const origin = document.createElement('button')
    vi.spyOn(origin, 'getBoundingClientRect').mockReturnValue(
      new DOMRect(100, 20, 40, 40),
    )

    store.setTheme('dark', origin)

    const root = document.documentElement
    expect(root.hasAttribute(THEME_REVEAL_CSS.attribute)).toBe(true)
    expect(root.style.getPropertyValue(THEME_REVEAL_CSS.center)).toBe(
      '120px 40px',
    )

    // CSS 里 circle() 必须带单位，漏掉 px 会让揭示静默退化成瞬间切换。
    expect(root.style.getPropertyValue(THEME_REVEAL_CSS.radius)).toMatch(/px$/)
    const radius = Number.parseFloat(
      root.style.getPropertyValue(THEME_REVEAL_CSS.radius),
    )
    // 半径要够到视口四角，否则动画结束时边缘会突然跳变。
    const corners: readonly (readonly [number, number])[] = [
      [0, 0],
      [window.innerWidth, 0],
      [0, window.innerHeight],
      [window.innerWidth, window.innerHeight],
    ]

    for (const [cornerX, cornerY] of corners) {
      expect(radius).toBeGreaterThanOrEqual(
        Math.hypot(cornerX - 120, cornerY - 40),
      )
    }

    // 上界防止「够大就行」的常量半径：那样第一帧就盖满视口，等于没有揭示。
    expect(radius).toBeLessThanOrEqual(
      Math.hypot(window.innerWidth, window.innerHeight),
    )
  })

  it('过渡挂起期间切换另一个主题不会被丢弃', () => {
    const { runCallbacks } = installViewTransition()
    const store = useThemeStore()

    store.setTheme('dark')
    store.setTheme('light')

    // 新过渡会 skip 旧过渡，旧回调可能不执行；终态必须来自最后一次请求。
    runCallbacks[1]?.()

    expect(store.theme).toBe('light')
    expect(window.localStorage.getItem('app.theme')).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('浏览器不支持 View Transitions 时同步应用主题', () => {
    const store = useThemeStore()

    store.setTheme('dark')

    expect(store.theme).toBe('dark')
    expect(window.localStorage.getItem('app.theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('启动时应用已保存的主题偏好', () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'dark')

    useThemeStore()

    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('system 偏好下跟随系统主题变化', async () => {
    const setSystemPrefersDark = stubMatchMedia(false)
    const store = useThemeStore()
    // useMediaQuery 的 change 监听在 post flush 注册，先让出一帧再模拟系统切换。
    await nextTick()

    setSystemPrefersDark(true)

    expect(store.theme).toBe('system')
    expect(store.resolvedTheme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('style.css 与 store 使用同一组揭示契约名字', () => {
    expect(collectRevealCssNames(styleSource)).toEqual(
      [
        THEME_REVEAL_CSS.attribute,
        THEME_REVEAL_CSS.center,
        THEME_REVEAL_CSS.radius,
      ].sort(),
    )
  })

  it('系统要求减少动效时不启动整页过渡，主题同步落地', () => {
    const { startViewTransition } = installViewTransition()
    stubMatchMedia(true)
    const store = useThemeStore()
    const origin = document.createElement('button')

    store.setTheme('dark', origin)

    expect(startViewTransition).not.toHaveBeenCalled()
    expect(store.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    // 揭示开关只能由过渡消费；没启动过渡就不能留下它，否则下一次路由跳转会误播圆形揭示。
    expect(
      document.documentElement.hasAttribute(THEME_REVEAL_CSS.attribute),
    ).toBe(false)
  })

  it('index.html 首帧脚本与 store 使用同一主题存储键与系统主题查询', () => {
    // 键名或 query 不同步都不会报错，只会在深色主题下先闪一次白屏，所以在这里锁住。
    expect(indexHtmlSource).toContain(THEME_STORAGE_KEY)
    expect(indexHtmlSource).toContain(SYSTEM_THEME_MEDIA_QUERY)
  })
})
