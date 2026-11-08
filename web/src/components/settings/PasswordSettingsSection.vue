<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import SettingsGroup from '@/components/common/SettingsGroup.vue'
import { usePasswordSettings } from '@/composables/settings/usePasswordSettings'

const { t } = useI18n()
const { passwordForm, passwordError, isUpdatingPassword, updatePassword } =
  usePasswordSettings()
</script>

<template>
  <SettingsGroup :title="t('settings.account.password')">
    <!-- 表单是卡片的单个子节点，组内分隔线不会落到字段之间。 -->
    <div class="space-y-4 p-4">
      <div class="space-y-2">
        <Label for="current-password">{{
          t('common.field.currentPassword')
        }}</Label>
        <Input
          id="current-password"
          v-model="passwordForm.currentPassword"
          type="password"
          :placeholder="t('common.placeholder.currentPassword')"
        />
      </div>

      <div class="space-y-2">
        <Label for="new-password">{{ t('common.field.newPassword') }}</Label>
        <Input
          id="new-password"
          v-model="passwordForm.newPassword"
          type="password"
          :placeholder="t('common.placeholder.newPassword')"
        />
      </div>

      <div class="space-y-2">
        <Label for="confirm-password">{{
          t('common.field.confirmPassword')
        }}</Label>
        <Input
          id="confirm-password"
          v-model="passwordForm.confirmPassword"
          type="password"
          :placeholder="t('common.placeholder.confirmPassword')"
        />
      </div>

      <p
        v-if="passwordError !== ''"
        class="text-sm text-destructive"
      >
        {{ passwordError }}
      </p>

      <div class="flex justify-end">
        <Button
          variant="outline"
          :disabled="isUpdatingPassword"
          @click="updatePassword"
        >
          {{
            isUpdatingPassword
              ? t('common.state.saving')
              : t('common.action.save')
          }}
        </Button>
      </div>
    </div>
  </SettingsGroup>
</template>
