import { fileURLToPath } from 'node:url'
import js from '@eslint/js'
import { defineConfig } from 'eslint/config'
import eslintConfigPrettier from 'eslint-config-prettier'
import globals from 'globals'
import betterTailwindcss from 'eslint-plugin-better-tailwindcss'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'

const tsconfigRootDir = fileURLToPath(new URL('.', import.meta.url))

export default defineConfig(
  { ignores: ['dist', 'node_modules', 'public/mockServiceWorker.js'] },

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
