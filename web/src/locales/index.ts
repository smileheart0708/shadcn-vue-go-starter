import enUSMessages from './en-US.json'
import zhCNMessages from './zh-CN.json'

/**
 * 语言清单：顺序即切换器显示顺序，首项同时是默认语言与未知浏览器语言的兜底；
 * 只保留一个语言包时，它就是默认语言。
 *
 * 新增语言时只改这里和 `messages`，再放一个 `<tag>.json`；切换器、浏览器探测、
 * 持久化校验和目录一致性测试都会自动跟随。语言名写在这里而不是语言包里，
 * 是为了让 `MessageSchema` 保持纯消息，不被保留键污染。
 */
export const supportedLocales = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
] as const

export type AppLocale = (typeof supportedLocales)[number]['value']

/**
 * 语言包本体是 JSON，类型由数据推导而不是手写：`MessageSchema` 取自下面 import 的
 * 参考包（当前 `zh-CN.json`），与默认语言、切换器顺序无关——只保留一个语言包时把
 * import 换成它即可。新增/删除一个键只需要动参考包，`MessageSchema` 自动跟随。
 *
 * JSON import 的叶子类型是 `string`（不是字面量），因此可以直接用作
 * `Record<AppLocale, MessageSchema>` 的值类型。切勿给语言包加 `as const`——
 * 那会把叶子收窄成中文字面量类型，导致英文语言包无法赋值。
 *
 * 类型层只保证「其他语言不缺参考包已有的键」；「多出的键」由 ESLint 的
 * `@intlify/vue-i18n/no-missing-keys-in-other-locales` 兜住，因为 `satisfies`
 * 与普通类型标注对 import 出来的对象都不做多余属性检查。
 */
export type MessageSchema = typeof zhCNMessages

/**
 * 用具名类型标注而不是 `satisfies`：一旦某个语言缺键，`satisfies` 会让整个字面量
 * 不再是 `Record<AppLocale, MessageSchema>`，`createI18n` 的泛型推导随之失败，
 * 报错会扩散成一片 `i18n.global is of type 'unknown'`。标注则把错误收敛到下面
 * 这一处赋值，其余位置继续拿到完整类型。
 */
export const messages: Record<AppLocale, MessageSchema> = {
  'zh-CN': zhCNMessages,
  'en-US': enUSMessages,
}

/** 首项即默认语言；只剩一个语言包时，默认就是它。 */
export const defaultLocale: AppLocale = supportedLocales[0].value

export const fallbackLocale: AppLocale = defaultLocale
