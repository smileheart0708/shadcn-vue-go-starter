<script setup lang="ts">
import { useI18n } from 'vue-i18n'
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
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { useSystemSettings } from '@/composables/system-settings/useSystemSettings'

const { t } = useI18n()

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
    <section class="space-y-1">
      <h1 class="text-2xl font-semibold">{{ t('systemConfig.title') }}</h1>
      <p class="text-sm text-muted-foreground">
        {{ t('systemConfig.description') }}
      </p>
    </section>

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

    <div
      v-else-if="loadFailed"
      class="rounded-xl border bg-card p-4"
    >
      <Empty>
        <EmptyHeader>
          <EmptyTitle>{{
            t('systemConfig.feedback.loadFailedTitle')
          }}</EmptyTitle>
          <EmptyDescription>{{
            t('systemConfig.feedback.loadFailed')
          }}</EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button @click="loadSettings">{{
            t('systemConfig.actions.retry')
          }}</Button>
        </EmptyContent>
      </Empty>
    </div>

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
          <Switch v-model="form.publicRegistrationEnabled" />
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
          <Switch v-model="form.selfServiceAccountDeletionEnabled" />
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
            :disabled="saving || !isDirty"
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
