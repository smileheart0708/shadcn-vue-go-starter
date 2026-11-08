<script setup lang="ts">
import { Moon, Monitor, Sun } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useThemeStore } from '@/stores/theme'
import type { ThemePreference } from '@/stores/theme'
import { useI18n } from 'vue-i18n'
import type { ComponentPublicInstance } from 'vue'
import { useTemplateRef } from 'vue'
import { unrefElement } from '@vueuse/core'

const { t } = useI18n()
const themeStore = useThemeStore()

const trigger = useTemplateRef<ComponentPublicInstance>('trigger')

const icon = { light: Sun, dark: Moon, system: Monitor }
const themes = ['light', 'dark', 'system'] as const

function selectTheme(theme: ThemePreference) {
  themeStore.setTheme(theme, unrefElement(trigger) ?? undefined)
}

// 穷尽 switch：i18n lint 只认 t() 里的字面量键，模板字符串键会被判为未使用。
function themeLabel(theme: ThemePreference): string {
  switch (theme) {
    case 'light':
      return t('common.theme.light')
    case 'dark':
      return t('common.theme.dark')
    case 'system':
      return t('common.theme.system')
  }
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        ref="trigger"
        variant="ghost"
        size="icon"
      >
        <component
          :is="icon[themeStore.theme]"
          class="block-[1.2rem] inline-[1.2rem]"
        />
        <span class="sr-only">{{ t('common.aria.toggleTheme') }}</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuItem
        v-for="theme in themes"
        :key="theme"
        @click="selectTheme(theme)"
      >
        <component
          :is="icon[theme]"
          class="me-2 block-4 inline-4"
        />
        {{ themeLabel(theme) }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
