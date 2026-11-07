<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CircleAlert } from '@lucide/vue'
import PageHeader from '@/components/app/PageHeader.vue'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import SettingsGroup from '@/components/common/SettingsGroup.vue'
import SettingsRow from '@/components/common/SettingsRow.vue'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { useAuthStore } from '@/stores/auth'
import { useSystemSettings } from '@/composables/system-settings/useSystemSettings'

const { t } = useI18n()
const authStore = useAuthStore()

// The route only requires `systemSettingsRead`, so a read-only operator can inspect the
// current policy. Writing is a separate capability; without this guard the switches stay
// clickable and the PATCH answers 403.
const canUpdateSettings = computed(() =>
  authStore.can(CAPABILITY.systemSettingsUpdate),
)

const {
  loading,
  saving,
  loadFailed,
  form,
  loadSettings,
  setPublicRegistrationEnabled,
  setSelfServiceAccountDeletionEnabled,
} = useSystemSettings()
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <PageHeader :title="t('systemConfig.title')" />

    <SettingsGroup
      v-if="loading"
      :title="t('systemConfig.groupTitle')"
    >
      <div
        v-for="row in 2"
        :key="row"
        class="space-y-2 px-4 py-3"
      >
        <Skeleton class="rounded-md block-4 inline-32" />
        <Skeleton class="rounded-md block-4 inline-64" />
      </div>
    </SettingsGroup>

    <!--
      No bordered card wraps this placeholder, and it carries no error description:
      the cause is already raised as a toast by `useSystemSettings`, so repeating it
      here would report the same failure twice. The retry action stays inline because
      the toast disappears on its own.
    -->
    <Empty v-else-if="loadFailed">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <CircleAlert />
        </EmptyMedia>
        <EmptyTitle>{{
          t('systemConfig.feedback.loadFailedTitle')
        }}</EmptyTitle>
      </EmptyHeader>
      <EmptyContent>
        <Button @click="loadSettings">{{
          t('systemConfig.actions.retry')
        }}</Button>
      </EmptyContent>
    </Empty>

    <SettingsGroup
      v-else
      :title="t('systemConfig.groupTitle')"
    >
      <SettingsRow
        :title="t('systemConfig.fields.publicRegistrationEnabled.title')"
        :description="
          t('systemConfig.fields.publicRegistrationEnabled.description')
        "
      >
        <!-- 行标题不是 <label>，开关靠 aria-label 拿到可读名称。 -->
        <Switch
          :model-value="form.publicRegistrationEnabled"
          :disabled="!canUpdateSettings || saving"
          :aria-label="t('systemConfig.fields.publicRegistrationEnabled.title')"
          @update:model-value="setPublicRegistrationEnabled"
        />
      </SettingsRow>

      <SettingsRow
        :title="
          t('systemConfig.fields.selfServiceAccountDeletionEnabled.title')
        "
        :description="
          t('systemConfig.fields.selfServiceAccountDeletionEnabled.description')
        "
      >
        <Switch
          :model-value="form.selfServiceAccountDeletionEnabled"
          :disabled="!canUpdateSettings || saving"
          :aria-label="
            t('systemConfig.fields.selfServiceAccountDeletionEnabled.title')
          "
          @update:model-value="setSelfServiceAccountDeletionEnabled"
        />
      </SettingsRow>
    </SettingsGroup>
  </div>
</template>
