import 'vue-i18n'
import type { MessageSchema } from '@/locales'

declare module 'vue-i18n' {
  // Extends the whole schema rather than listing blocks by hand, so this declaration
  // cannot drift as blocks are added or removed.
  //
  // Note: this improves `tm()`/editor completion but does NOT make `t()` reject unknown
  // keys — vue-i18n types `t` as `<Key extends string = string>`, which accepts any
  // string. Key existence is enforced statically by the ESLint rule
  // `@intlify/vue-i18n/no-missing-keys`, which reads `src/locales/*.json`.
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  export interface DefineLocaleMessage extends MessageSchema {}
}
