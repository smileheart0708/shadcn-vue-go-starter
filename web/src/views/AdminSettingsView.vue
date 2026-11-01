<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CircleAlert } from '@lucide/vue'
import PageHeader from '@/components/app/PageHeader.vue'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyContent,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { useAuthStore } from '@/stores/auth'
import { useSystemSettings } from '@/composables/system-settings/useSystemSettings'

const { t } = useI18n()
const authStore = useAuthStore()

// The route only requires `systemSettingsRead`, so a read-only operator can inspect the
// current policy. Saving is a separate capability; without this guard the switches and
// the save button stay clickable and the PATCH answers 403.
const canUpdateSettings = computed(() =>
  authStore.can(CAPABILITY.systemSettingsUpdate),
)

const {
  loading,
  saving,
  loadFailed,
  form,
  isDirty,
  loadSettings,
  saveSettings,
} = useSystemSettings()
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <PageHeader
      :title="t('systemConfig.title')"
      :description="t('systemConfig.description')"
    />

    <Card v-if="loading">
      <CardHeader>
        <Skeleton class="rounded-md block-6 inline-32" />
      </CardHeader>
      <CardContent class="space-y-4">
        <Skeleton class="rounded-lg block-16" />
        <Skeleton class="rounded-lg block-16" />
        <div class="flex justify-end gap-2">
          <Skeleton class="rounded-md block-10 inline-20" />
          <Skeleton class="rounded-md block-10 inline-20" />
        </div>
      </CardContent>
    </Card>

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

    <Card v-else>
      <CardHeader>
        <CardTitle>{{ t('systemConfig.cardTitle') }}</CardTitle>
        <CardDescription>{{
          t('systemConfig.cardDescription')
        }}</CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <label
          class="flex items-center justify-between gap-4 rounded-lg border p-4"
        >
          <div class="space-y-1">
            <div class="font-medium">
              {{ t('systemConfig.fields.publicRegistrationEnabled.title') }}
            </div>
            <p class="text-sm text-muted-foreground">
              {{
                t('systemConfig.fields.publicRegistrationEnabled.description')
              }}
            </p>
          </div>
          <Switch
            v-model="form.publicRegistrationEnabled"
            :disabled="!canUpdateSettings"
          />
        </label>

        <label
          class="flex items-center justify-between gap-4 rounded-lg border p-4"
        >
          <div class="space-y-1">
            <div class="font-medium">
              {{
                t('systemConfig.fields.selfServiceAccountDeletionEnabled.title')
              }}
            </div>
            <p class="text-sm text-muted-foreground">
              {{
                t(
                  'systemConfig.fields.selfServiceAccountDeletionEnabled.description',
                )
              }}
            </p>
          </div>
          <Switch
            v-model="form.selfServiceAccountDeletionEnabled"
            :disabled="!canUpdateSettings"
          />
        </label>

        <div class="flex justify-end gap-2">
          <Button
            variant="outline"
            :disabled="saving || !isDirty"
            @click="loadSettings"
          >
            {{ t('common.action.cancel') }}
          </Button>
          <Button
            :disabled="saving || !isDirty || !canUpdateSettings"
            @click="saveSettings"
          >
            <Spinner
              v-if="saving"
              class="me-2"
            />
            {{ t('common.action.save') }}
          </Button>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
