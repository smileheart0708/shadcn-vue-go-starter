import { defaultDocument, useMediaQuery } from '@vueuse/core'
import {
  computed,
  getCurrentInstance,
  ref,
  watch,
  type ComputedRef,
  type Ref,
} from 'vue'
import { z } from 'zod'
import { parseExternalValueOrDefault } from '@/lib/external-input'
import { WIDE_LAYOUT_MEDIA_QUERY } from '@/lib/layout/breakpoints'
import { SIDEBAR_COOKIE_MAX_AGE, SIDEBAR_COOKIE_NAME } from './utils'

export interface SidebarControllerProps {
  defaultOpen?: boolean | null
  open?: boolean | null
}

export interface SidebarController {
  state: ComputedRef<'expanded' | 'collapsed'>
  open: Ref<boolean>
  setOpen: (value: boolean) => void
  /** True at 840px and up: the sidebar is a fixed bar. */
  isWide: Ref<boolean>
  /** True below 840px: the sidebar is a sheet. The exact negation of `isWide`. */
  isNarrow: Ref<boolean>
  openSheet: Ref<boolean>
  setOpenSheet: (value: boolean) => void
  toggleSidebar: () => void
}

const storedSidebarOpenSchema = z
  .enum(['true', 'false'])
  .transform((value) => value === 'true')

export function useSidebarController(
  props: SidebarControllerProps,
  emitUpdateOpen: (value: boolean) => void,
): SidebarController {
  // Wide is exactly the query the `wide:` variant uses; narrow is its negation, so no
  // viewport width — including a fractional one produced by browser zoom — can fall
  // between the fixed bar and the sheet. A second `(max-width: …)` threshold is the
  // bug this shape exists to prevent.
  const isWide = useMediaQuery(WIDE_LAYOUT_MEDIA_QUERY)
  const isNarrow = computed(() => !isWide.value)
  const openSheet = ref(false)
  const uncontrolledOpen = ref(readStoredSidebarOpen())
  const instance = getCurrentInstance()
  // Vue normalizes omitted Boolean props, so we read vnode props once here to
  // distinguish an omitted prop from an explicit `false`.
  const isOpenControlled = hasPassedProp(instance, 'open')
  const hasDefaultOpenProp = hasPassedProp(instance, 'defaultOpen')

  watch(
    () => props.defaultOpen,
    (defaultOpen) => {
      if (
        hasDefaultOpenProp &&
        defaultOpen !== null &&
        defaultOpen !== undefined
      ) {
        uncontrolledOpen.value = defaultOpen
      }
    },
    { immediate: true, once: true },
  )

  const open = computed({
    get: () =>
      isOpenControlled
        ? (props.open ?? uncontrolledOpen.value)
        : uncontrolledOpen.value,
    set: (value: boolean) => {
      if (!isOpenControlled) {
        uncontrolledOpen.value = value
      }

      emitUpdateOpen(value)
    },
  })

  watch(
    open,
    (value) => {
      persistStoredSidebarOpen(value)
    },
    { immediate: true },
  )

  function setOpen(value: boolean) {
    open.value = value
  }

  function setOpenSheet(value: boolean) {
    openSheet.value = value
  }

  function toggleSidebar() {
    if (isNarrow.value) {
      setOpenSheet(!openSheet.value)
      return
    }

    setOpen(!open.value)
  }

  const state = computed(() => (open.value ? 'expanded' : 'collapsed'))

  return {
    state,
    open,
    setOpen,
    isWide,
    isNarrow,
    openSheet,
    setOpenSheet,
    toggleSidebar,
  }
}

function hasPassedProp(
  instance: ReturnType<typeof getCurrentInstance>,
  name: 'open' | 'defaultOpen',
) {
  return Boolean(
    instance?.vnode.props &&
    Object.prototype.hasOwnProperty.call(instance.vnode.props, name),
  )
}

function persistStoredSidebarOpen(value: boolean) {
  if (defaultDocument === undefined) {
    return
  }

  const cookieValue = value ? 'true' : 'false'
  defaultDocument.cookie = `${SIDEBAR_COOKIE_NAME}=${cookieValue}; path=/; max-age=${String(SIDEBAR_COOKIE_MAX_AGE)}`
}

function readStoredSidebarOpen(): boolean {
  if (defaultDocument === undefined) {
    return true
  }

  return parseExternalValueOrDefault(
    storedSidebarOpenSchema,
    readCookieValue(SIDEBAR_COOKIE_NAME),
    true,
  )
}

function readCookieValue(name: string): string | null {
  for (const cookie of defaultDocument?.cookie.split(';') ?? []) {
    const [cookieName, ...cookieValueParts] = cookie.trim().split('=')
    if (cookieName === name) {
      return cookieValueParts.join('=')
    }
  }

  return null
}
