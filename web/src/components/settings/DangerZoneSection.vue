<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Spinner } from '@/components/ui/spinner'
import SettingsGroup from '@/components/common/SettingsGroup.vue'
import SettingsRow from '@/components/common/SettingsRow.vue'
import { useDeleteAccountFlow } from '@/composables/settings/useDeleteAccountFlow'

const { t } = useI18n()
const {
  deleteDialogOpen,
  deleteCountdown,
  deleteAccountConfirmed,
  canDeleteAccount,
  canOpenDeleteDialog,
  canSubmitDelete,
  isDeletingAccount,
  confirmDelete,
} = useDeleteAccountFlow()
</script>

<template>
  <SettingsGroup
    v-if="canDeleteAccount"
    :title="t('settings.account.dangerZone')"
    title-class="text-destructive"
    content-class="border-destructive/60"
  >
    <SettingsRow :title="t('settings.account.dangerZoneConfirm')">
      <!-- 行标题不再是 <label>，复选框要靠 aria-label 拿到可读名称。 -->
      <Checkbox
        v-model="deleteAccountConfirmed"
        :aria-label="t('settings.account.dangerZoneConfirm')"
      />
    </SettingsRow>

    <SettingsRow :title="t('settings.account.deleteAccount')">
      <AlertDialog v-model:open="deleteDialogOpen">
        <AlertDialogTrigger as-child>
          <Button
            variant="destructive"
            :disabled="!canOpenDeleteDialog"
          >
            {{ t('settings.account.deleteAccount') }}
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{{
              t('settings.account.deleteAccount')
            }}</AlertDialogTitle>
            <AlertDialogDescription>{{
              t('settings.account.deleteAccountConfirm')
            }}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel :disabled="isDeletingAccount">{{
              t('common.action.cancel')
            }}</AlertDialogCancel>
            <!-- 让确认动作在对话框自动关闭前先进入提交分支。 -->
            <AlertDialogAction
              as-child
              :disabled="!canSubmitDelete || isDeletingAccount"
              @click.prevent="confirmDelete"
            >
              <Button
                variant="destructive"
                :disabled="!canSubmitDelete || isDeletingAccount"
              >
                <Spinner
                  v-if="isDeletingAccount"
                  class="me-2"
                />
                {{
                  deleteCountdown > 0
                    ? `${deleteCountdown}s`
                    : t('settings.account.deleteAccount')
                }}
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsRow>
  </SettingsGroup>
</template>
