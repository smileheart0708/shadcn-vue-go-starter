import type { AppLocale } from '@/plugins/i18n/locales'
import enUSMessages from './en-US.json'
import zhCNMessages from './zh-CN.json'

/**
 * 语言包本体是 JSON（`zh-CN.json` 为唯一事实来源），类型由数据推导而不是手写：
 * 新增/删除一个键只需要动 JSON，`MessageSchema` 自动跟随。
 *
 * JSON import 的叶子类型是 `string`（不是字面量），因此可以直接用作
 * `Record<AppLocale, MessageSchema>` 的值类型。切勿给语言包加 `as const`——
 * 那会把叶子收窄成中文字面量类型，导致英文语言包无法赋值。
 *
 * 类型层只保证「英文不缺中文已有的键」；「英文多出的键」由 ESLint 的
 * `@intlify/vue-i18n/no-missing-keys-in-other-locales` 兜住，因为 `satisfies`
 * 与普通类型标注对 import 出来的对象都不做多余属性检查。
 */
export type MessageSchema = typeof zhCNMessages

/**
 * 用具名类型标注而不是 `satisfies`：一旦英文缺键，`satisfies` 会让整个字面量
 * 不再是 `Record<AppLocale, MessageSchema>`，`createI18n` 的泛型推导随之失败，
 * 报错会扩散成一片 `i18n.global is of type 'unknown'`。标注则把错误收敛到下面
 * 这一处赋值，其余位置继续拿到完整类型。
 */
export const messages: Record<AppLocale, MessageSchema> = {
  'zh-CN': zhCNMessages,
  'en-US': enUSMessages,
}
