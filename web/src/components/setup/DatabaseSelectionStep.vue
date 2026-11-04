<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Database, Rocket } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import type { SetupWizard } from '@/composables/setup/types'

const { wizard } = defineProps<{ wizard: SetupWizard }>()
const { t } = useI18n()

const draft = computed(() => wizard.databaseDraft.value)
const submitting = computed(() => wizard.submitting.value)
const finalizationDriver = computed(() => wizard.finalizationDriver.value)
const finalizationRequired = computed(() => wizard.finalizationRequired.value)

const selectedDriverLabel = computed(() => {
  const driver = finalizationDriver.value ?? draft.value.driver
  return driver === 'sqlite'
    ? t('setup.database.sqlite')
    : t('setup.database.postgresql')
})

function setDriver(value: unknown) {
  if (value === 'sqlite' || value === 'postgresql') {
    draft.value.driver = value
  }
}

function setSSLMode(value: unknown) {
  if (value === 'disable' || value === 'require' || value === 'verify-full') {
    draft.value.postgresql.sslMode = value
  }
}
</script>

<template>
  <form
    class="flex flex-1 flex-col"
    @submit.prevent="wizard.completeDatabase()"
  >
    <div class="mbe-6 space-y-2">
      <h2 class="text-xl font-semibold tracking-tight">
        {{ t('setup.database.title') }}
      </h2>
      <p class="text-sm text-muted-foreground">
        {{
          finalizationRequired
            ? t('setup.database.finalizationDescription', {
                driver: selectedDriverLabel,
              })
            : t('setup.database.description')
        }}
      </p>
    </div>

    <FieldGroup class="flex flex-1 flex-col gap-5">
      <template v-if="finalizationRequired">
        <div
          class="flex items-center gap-3 rounded-lg border border-border/70 bg-muted/30 p-4"
        >
          <Database class="text-primary block-5 inline-5" />
          <div>
            <p class="font-medium">{{ selectedDriverLabel }}</p>
            <p class="text-sm text-muted-foreground">
              {{ t('setup.database.finalizationHint') }}
            </p>
          </div>
        </div>
      </template>

      <template v-else>
        <Field>
          <FieldLabel>{{ t('setup.database.driver') }}</FieldLabel>
          <Select
            :model-value="draft.driver"
            @update:model-value="setDriver"
          >
            <SelectTrigger class="inline-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="sqlite">
                {{ t('setup.database.sqlite') }}
              </SelectItem>
              <SelectItem value="postgresql">
                {{ t('setup.database.postgresql') }}
              </SelectItem>
            </SelectContent>
          </Select>
          <p class="text-sm text-muted-foreground">
            {{
              draft.driver === 'sqlite'
                ? t('setup.database.sqliteDescription')
                : t('setup.database.postgresqlDescription')
            }}
          </p>
        </Field>

        <div
          v-if="draft.driver === 'postgresql'"
          class="grid gap-5 sm:grid-cols-2"
        >
          <Field>
            <FieldLabel for="setup-pg-host">{{
              t('setup.database.host')
            }}</FieldLabel>
            <Input
              id="setup-pg-host"
              v-model="draft.postgresql.host"
              autocomplete="off"
              required
            />
          </Field>

          <Field>
            <FieldLabel for="setup-pg-port">{{
              t('setup.database.port')
            }}</FieldLabel>
            <Input
              id="setup-pg-port"
              v-model.number="draft.postgresql.port"
              inputmode="numeric"
              max="65535"
              min="1"
              required
              type="number"
            />
          </Field>

          <Field class="sm:col-span-2">
            <FieldLabel for="setup-pg-database">{{
              t('setup.database.database')
            }}</FieldLabel>
            <Input
              id="setup-pg-database"
              v-model="draft.postgresql.database"
              autocomplete="off"
              required
            />
          </Field>

          <Field>
            <FieldLabel for="setup-pg-username">{{
              t('setup.database.username')
            }}</FieldLabel>
            <Input
              id="setup-pg-username"
              v-model="draft.postgresql.username"
              autocomplete="off"
              required
            />
          </Field>

          <Field>
            <FieldLabel for="setup-pg-password">{{
              t('setup.database.password')
            }}</FieldLabel>
            <Input
              id="setup-pg-password"
              v-model="draft.postgresql.password"
              autocomplete="off"
              required
              type="password"
            />
          </Field>

          <Field class="sm:col-span-2">
            <FieldLabel>{{ t('setup.database.sslMode') }}</FieldLabel>
            <Select
              :model-value="draft.postgresql.sslMode"
              @update:model-value="setSSLMode"
            >
              <SelectTrigger class="inline-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="disable">
                  {{ t('setup.database.ssl.disable') }}
                </SelectItem>
                <SelectItem value="require">
                  {{ t('setup.database.ssl.require') }}
                </SelectItem>
                <SelectItem value="verify-full">
                  {{ t('setup.database.ssl.verifyFull') }}
                </SelectItem>
              </SelectContent>
            </Select>
          </Field>
        </div>
      </template>

      <Field class="mbs-auto flex-row gap-3 pbs-2">
        <Button
          type="button"
          size="lg"
          variant="outline"
          :disabled="submitting"
          @click="wizard.back()"
        >
          <ArrowLeft class="me-2 block-4 inline-4" />
          {{ t('common.action.back') }}
        </Button>

        <Button
          type="submit"
          size="lg"
          class="flex-1"
          :disabled="submitting"
        >
          <Spinner
            v-if="submitting"
            class="me-2"
          />
          <Rocket
            v-else
            class="me-2 block-4 inline-4"
          />
          {{
            submitting
              ? t('setup.database.submitting')
              : t('setup.database.submit')
          }}
        </Button>
      </Field>
    </FieldGroup>
  </form>
</template>
