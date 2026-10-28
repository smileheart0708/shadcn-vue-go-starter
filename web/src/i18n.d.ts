import 'vue-i18n'
import type { MessageSchema } from '@/locales/schema'

declare module 'vue-i18n' {
  // Extends the whole schema rather than listing blocks by hand, so this declaration
  // cannot drift as blocks are added or removed.
  //
  // Note: this improves `tm()`/editor completion but does NOT make `t()` reject unknown
  // keys — vue-i18n types `t` as `<Key extends string = string>`, which accepts any
  // string. Typos in `t('...')` therefore surface at runtime (the key renders verbatim),
  // not at build time.
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  export interface DefineLocaleMessage extends MessageSchema {}
}
