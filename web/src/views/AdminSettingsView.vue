<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppShellPage from '@/components/app-shell/AppShellPage.vue'
import SettingsGroup from '@/components/common/SettingsGroup.vue'
import SettingsRow from '@/components/common/SettingsRow.vue'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
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
  loadFailed,
  saving,
  form,
  setPublicRegistrationEnabled,
  setSelfServiceAccountDeletionEnabled,
} = useSystemSettings()

// 失败后骨架留在原位但停止脉冲：它表示「值未知」，不表示「正在加载」。
// 失败原因由 useSystemSettings 的 toast 说明，页面内不再重复一次。
const skeletonClass = computed(() => ({
  'animate-none opacity-60': loadFailed.value && !loading.value,
}))
</script>

<template>
  <AppShellPage>
    <!--
      行标题与描述都是静态文案，不依赖请求，所以三态（加载中 / 失败 / 成功）共用同一套行结构，
      只把「值」的位置在骨架胶囊与开关之间切换。胶囊与开关同尺寸，页面高度恒定；
      失败时也绝不把 form 的默认值当成真实策略渲染出来。
    -->
    <SettingsGroup :title="t('systemConfig.groupTitle')">
      <SettingsRow
        :title="t('systemConfig.fields.publicRegistrationEnabled.title')"
        :description="
          t('systemConfig.fields.publicRegistrationEnabled.description')
        "
      >
        <template v-if="form">
          <!-- 行标题不是 <label>，开关靠 aria-label 拿到可读名称。 -->
          <Switch
            :model-value="form.publicRegistrationEnabled"
            :disabled="!canUpdateSettings || saving"
            :aria-label="
              t('systemConfig.fields.publicRegistrationEnabled.title')
            "
            @update:model-value="setPublicRegistrationEnabled"
          />
        </template>
        <Skeleton
          v-else
          class="rounded-full block-5.5 inline-10"
          :class="skeletonClass"
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
        <template v-if="form">
          <Switch
            :model-value="form.selfServiceAccountDeletionEnabled"
            :disabled="!canUpdateSettings || saving"
            :aria-label="
              t('systemConfig.fields.selfServiceAccountDeletionEnabled.title')
            "
            @update:model-value="setSelfServiceAccountDeletionEnabled"
          />
        </template>
        <Skeleton
          v-else
          class="rounded-full block-5.5 inline-10"
          :class="skeletonClass"
        />
      </SettingsRow>
    </SettingsGroup>
  </AppShellPage>
</template>
