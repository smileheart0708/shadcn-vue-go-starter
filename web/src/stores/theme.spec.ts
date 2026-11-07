import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { THEME_REVEAL_CSS, useThemeStore } from '@/stores/theme'

// CSS 无法从 TS 读取常量，只能由测试读文件校验；路径从本模块位置推导，见
// lib/layout/breakpoints.spec.ts 里对 jsdom 下 import.meta.url 的说明。
const styleSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../style.css'),
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

describe('theme store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    window.localStorage.clear()
    document.documentElement.className = ''
    document.documentElement.removeAttribute(THEME_REVEAL_CSS.attribute)
    document.documentElement.removeAttribute('style')
  })

  afterEach(() => {
    Reflect.deleteProperty(document, 'startViewTransition')
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

  it('style.css 与 store 使用同一组揭示契约名字', () => {
    expect(collectRevealCssNames(styleSource)).toEqual(
      [
        THEME_REVEAL_CSS.attribute,
        THEME_REVEAL_CSS.center,
        THEME_REVEAL_CSS.radius,
      ].sort(),
    )
  })
})
