import { fileURLToPath } from 'node:url'
import js from '@eslint/js'
import { defineConfig } from 'eslint/config'
import eslintConfigPrettier from 'eslint-config-prettier'
import globals from 'globals'
import betterTailwindcss from 'eslint-plugin-better-tailwindcss'
import pluginVue from 'eslint-plugin-vue'
import pluginVueI18n from '@intlify/eslint-plugin-vue-i18n'
import tseslint from 'typescript-eslint'

const tsconfigRootDir = fileURLToPath(new URL('.', import.meta.url))

// 语言包 JSON 是 i18n 规则的消息来源，本身也必须被 lint：`no-unused-keys` 只在
// 「被 lint 的语言包文件」上报告未使用键，扫源文件反而不报。
// 只取插件里针对 JSON 的那份解析配置，并把范围收窄到 locales 目录——否则
// 所有 *.json（例如 .prettierrc.json）都会被挂上 vue-eslint-parser，与
// typescript-eslint 的类型感知规则相撞。
const localeMessageFiles = ['src/locales/*.json']
const pluginVueI18nJsonConfig = pluginVueI18n.configs['flat/base'].find(
  (config) => config.files?.some((pattern) => pattern.endsWith('.json')),
)

export default defineConfig(
  { ignores: ['dist', 'node_modules'] },

  js.configs.recommended,
  ...tseslint.configs.recommendedTypeChecked,
  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,
  ...pluginVue.configs['flat/recommended'],

  {
    files: ['**/*.{ts,tsx,mts,cts,js,mjs,cjs,vue}'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: { ...globals.browser, ...globals.node },
      parserOptions: {
        parser: tseslint.parser,
        projectService: true,
        extraFileExtensions: ['.vue'],
        tsconfigRootDir,
      },
    },
  },

  {
    files: ['**/*.{js,mjs,cjs}'],
    extends: [tseslint.configs.disableTypeChecked],
  },

  {
    files: ['**/*.{ts,tsx,mts,cts,vue}'],
    rules: {
      'no-unused-vars': 'off',
      'no-use-before-define': 'off',
      'no-unused-expressions': 'off',
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
          destructuredArrayIgnorePattern: '^_',
          varsIgnorePattern: '^_',
        },
      ],
      '@typescript-eslint/no-use-before-define': [
        'error',
        {
          functions: false,
          classes: true,
          variables: true,
          typedefs: true,
          enums: true,
        },
      ],
      '@typescript-eslint/no-unused-expressions': 'error',

      '@typescript-eslint/consistent-type-assertions': [
        'error',
        { assertionStyle: 'never' },
      ],
      '@typescript-eslint/consistent-type-imports': [
        'error',
        {
          prefer: 'type-imports',
          fixStyle: 'separate-type-imports',
        },
      ],
      '@typescript-eslint/no-import-type-side-effects': 'error',

      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-floating-promises': [
        'error',
        { ignoreVoid: true, ignoreIIFE: false },
      ],
      '@typescript-eslint/no-misused-promises': 'error',
      '@typescript-eslint/no-misused-spread': 'error',

      '@typescript-eslint/no-unnecessary-condition': 'error',
      '@typescript-eslint/strict-boolean-expressions': 'error',
      '@typescript-eslint/prefer-nullish-coalescing': 'error',
      '@typescript-eslint/prefer-optional-chain': 'error',

      '@typescript-eslint/no-unnecessary-type-assertion': 'error',
      '@typescript-eslint/no-unsafe-type-assertion': 'error',
      '@typescript-eslint/no-unnecessary-type-arguments': 'error',
      '@typescript-eslint/no-unnecessary-type-conversion': 'error',
      '@typescript-eslint/no-unnecessary-template-expression': 'error',

      '@typescript-eslint/no-unnecessary-boolean-literal-compare': 'error',
      '@typescript-eslint/no-non-null-asserted-optional-chain': 'error',
      '@typescript-eslint/no-non-null-asserted-nullish-coalescing': 'error',
      '@typescript-eslint/no-meaningless-void-operator': 'error',
      '@typescript-eslint/promise-function-async': 'error',
      '@typescript-eslint/no-unsafe-argument': 'off',
      '@typescript-eslint/no-deprecated': 'error',
      'vue/multi-word-component-names': ['error', { ignores: ['App'] }],
      'vue/require-default-prop': 'off',

      'vue/no-mutating-props': 'error',
      'vue/no-ref-as-operand': 'error',
      'vue/no-ref-object-reactivity-loss': 'error',
      'vue/no-setup-props-reactivity-loss': 'error',
      'vue/no-template-shadow': 'error',
      'vue/no-use-v-if-with-v-for': [
        'error',
        { allowUsingIterationVar: false },
      ],
      'vue/no-use-computed-property-like-method': 'error',
      'vue/no-unused-components': 'error',
      // Catches components used in a template but never imported. Without this, a
      // missing import silently degrades to a runtime `resolveComponent` lookup that
      // fails only when that branch of the template renders — `RouterView`/`RouterLink`
      // are the legitimate globals registered by vue-router.
      'vue/no-undef-components': [
        'error',
        { ignorePatterns: ['RouterView', 'RouterLink'] },
      ],
      'vue/no-unused-vars': ['error', { ignorePattern: '^_' }],
      'vue/no-useless-v-bind': 'error',
      'vue/no-useless-mustaches': 'error',

      'vue/require-explicit-emits': 'error',
      'vue/define-props-declaration': ['error', 'type-based'],
      'vue/define-emits-declaration': ['error', 'type-literal'],
      'vue/define-macros-order': 'error',

      'vue/require-prop-types': 'error',
      'vue/require-prop-type-constructor': 'error',
      'vue/require-valid-default-prop': 'error',
      'vue/no-required-prop-with-default': 'error',
      'vue/no-boolean-default': 'error',

      'vue/require-typed-object-prop': 'error',
      'vue/require-typed-ref': 'error',
      'vue/prefer-use-template-ref': 'error',

      'vue/custom-event-name-casing': ['error', 'kebab-case'],
      'vue/v-on-event-hyphenation': ['error', 'always', { autofix: true }],

      'vue/html-button-has-type': 'error',
      'vue/block-order': [
        'error',
        {
          order: [
            ['script[setup]', 'script:not([setup])'],
            'template',
            'style',
          ],
        },
      ],
      'vue/attributes-order': 'error',
    },
  },

  {
    ...pluginVueI18nJsonConfig,
    files: localeMessageFiles,
    // JSON 没有类型信息，必须关掉类型感知规则。
    extends: [tseslint.configs.disableTypeChecked],
  },

  {
    // i18n 静态校验：语言包是 JSON，规则才能从磁盘读出键全集。
    // 四条规则合起来把「键写错」「两语言包不对称」「键没人用」「运行期拼键」都变成
    // lint error。最后一条是关键：动态拼键对 no-unused-keys 不可见，键会被当成
    // 没人用而删除，运行时再静默渲染成原始键名——所以运行期拼键必须直接禁止，
    // 数据驱动的取键只能收敛成穷尽 switch 或内联到定义处。
    //
    // 只注册插件、不使用 `configs['flat/base']`：那套配置给**所有** *.json 挂上
    // vue-eslint-parser（用于直接 lint 语言包文件），会和本项目 typescript-eslint 的
    // 类型感知规则相撞（.prettierrc.json 等非语言包文件会直接报解析错误）。
    plugins: {
      '@intlify/vue-i18n': pluginVueI18n,
    },
    settings: {
      'vue-i18n': {
        localeDir: './src/locales/*.json',
        messageSyntaxVersion: '^11.0.0',
      },
    },
    rules: {
      '@intlify/vue-i18n/no-unused-keys': [
        'error',
        {
          src: './src',
          extensions: ['.vue', '.ts'],
          // 自动删除未使用键在有动态键的项目里是灾难，必须人工确认。
          enableFix: false,
        },
      ],
      '@intlify/vue-i18n/no-missing-keys': 'error',
      '@intlify/vue-i18n/no-missing-keys-in-other-locales': 'error',
      '@intlify/vue-i18n/no-dynamic-keys': 'error',
    },
  },

  {
    files: ['src/components/ui/**/*.vue'],
    rules: {
      'vue/multi-word-component-names': 'off',
    },
  },

  {
    files: ['**/*.{ts,tsx,mts,cts,js,mjs,cjs,vue}'],
    plugins: {
      'better-tailwindcss': betterTailwindcss,
    },
    settings: {
      'better-tailwindcss': {
        cwd: tsconfigRootDir,
        entryPoint: 'src/style.css',
      },
    },
    rules: {
      'better-tailwindcss/no-unknown-classes': 'error',
      'better-tailwindcss/no-conflicting-classes': 'error',
      'better-tailwindcss/no-deprecated-classes': 'error',
      'better-tailwindcss/no-duplicate-classes': 'error',
      'better-tailwindcss/no-unnecessary-whitespace': 'error',

      'better-tailwindcss/enforce-canonical-classes': 'error',
      'better-tailwindcss/enforce-shorthand-classes': 'error',
      'better-tailwindcss/enforce-logical-properties': 'error',
      'better-tailwindcss/enforce-consistent-class-order': 'error',
      'better-tailwindcss/enforce-consistent-important-position': 'error',
      'better-tailwindcss/enforce-consistent-variable-syntax': 'error',
      'better-tailwindcss/enforce-consistent-line-wrapping': 'off',
    },
  },

  eslintConfigPrettier,
)
