import type { ComputedRef, Ref } from 'vue'
import { createContext } from 'reka-ui'

export const SIDEBAR_COOKIE_NAME = 'sidebar_state'
export const SIDEBAR_COOKIE_MAX_AGE = 60 * 60 * 24 * 7
export const SIDEBAR_WIDTH = '16rem'
/** Width of the sidebar while it is a sheet, i.e. in the narrow layout. */
export const SIDEBAR_WIDTH_SHEET = '18rem'
export const SIDEBAR_WIDTH_ICON = '3rem'
export const SIDEBAR_KEYBOARD_SHORTCUT = 'b'

export const [useSidebar, provideSidebarContext] = createContext<{
  state: ComputedRef<'expanded' | 'collapsed'>
  open: Ref<boolean>
  setOpen: (value: boolean) => void
  /** Wide at 840px and up: the sidebar is a fixed bar. */
  isWide: Ref<boolean>
  /** The exact negation of `isWide`: the sidebar is a sheet. */
  isNarrow: Ref<boolean>
  openSheet: Ref<boolean>
  setOpenSheet: (value: boolean) => void
  toggleSidebar: () => void
}>('Sidebar')
