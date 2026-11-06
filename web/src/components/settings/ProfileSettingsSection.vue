<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Badge } from '@/components/ui/badge'
import UserAvatar from '@/components/common/UserAvatar.vue'
import SettingsGroup from '@/components/common/SettingsGroup.vue'
import EditProfileDialog from '@/components/settings/EditProfileDialog.vue'
import { useProfileSettings } from '@/composables/settings/useProfileSettings'

const { t } = useI18n()

const {
  currentIdentity,
  avatarImageSrc,
  editDialogOpen,
  editForm,
  editAvatarFallbackText,
  editAvatarImageSrc,
  profileError,
  isSavingProfile,
  roleLabel,
  roleBadgeVariant,
  openAvatarPicker,
  saveProfile,
} = useProfileSettings()
</script>

<template>
  <SettingsGroup :title="t('settings.account.profile')">
    <!--
      这一行不复用 SettingsRow：左侧是头像 + 用户名 + 两个徽章的组合块，塞不进它的
      title/description 契约。行容器类必须与 SettingsRow 保持一致，避免两处间距漂移。
    -->
    <div
      class="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-8"
    >
      <div class="flex items-center gap-4">
        <UserAvatar
          class="rounded-full block-12 inline-12"
          fallback-class="rounded-full"
          :username="currentIdentity?.username ?? ''"
          :avatar-url="avatarImageSrc"
        />
        <div class="space-y-1">
          <div class="flex flex-wrap items-center gap-2">
            <p class="font-medium">{{ currentIdentity?.username }}</p>
            <Badge :variant="roleBadgeVariant">{{ roleLabel }}</Badge>
            <Badge
              :variant="
                currentIdentity?.status === 'disabled' ? 'secondary' : 'outline'
              "
            >
              {{
                currentIdentity?.status === 'disabled'
                  ? t('common.state.disabled')
                  : t('settings.account.statusActive')
              }}
            </Badge>
          </div>
          <p class="text-sm text-muted-foreground">
            {{ currentIdentity?.email ?? t('settings.account.emailNotSet') }}
          </p>
        </div>
      </div>

      <EditProfileDialog
        v-model:open="editDialogOpen"
        v-model:username="editForm.username"
        v-model:email="editForm.email"
        :avatar-image-src="editAvatarImageSrc"
        :avatar-fallback-text="editAvatarFallbackText"
        :profile-error="profileError"
        :is-saving-profile="isSavingProfile"
        @choose-avatar="openAvatarPicker"
        @save-profile="saveProfile"
      />
    </div>
  </SettingsGroup>
</template>
