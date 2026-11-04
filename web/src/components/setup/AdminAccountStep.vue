<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { ArrowRight } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import type { SetupWizard } from '@/composables/setup/types'

const { wizard } = defineProps<{ wizard: SetupWizard }>()
const { t } = useI18n()

const username = ref('')
const password = ref('')
const confirmPassword = ref('')

onMounted(() => {
  username.value = wizard.adminDraft.value.username
  password.value = wizard.adminDraft.value.password
  confirmPassword.value = wizard.adminDraft.value.password
})

function submit() {
  if (password.value !== confirmPassword.value) {
    toast.error(t('setup.passwordMismatch'))
    return
  }

  wizard.completeAdmin({
    username: username.value.trim(),
    password: password.value,
  })
}
</script>

<template>
  <form
    class="flex flex-1 flex-col"
    @submit.prevent="submit"
  >
    <div class="mbe-6 space-y-2">
      <h2 class="text-xl font-semibold tracking-tight">
        {{ t('setup.admin.title') }}
      </h2>
      <p class="text-sm text-muted-foreground">
        {{ t('setup.admin.description') }}
      </p>
    </div>

    <FieldGroup class="flex flex-1 flex-col gap-5">
      <Field>
        <FieldLabel for="setup-username">{{
          t('common.field.username')
        }}</FieldLabel>
        <Input
          id="setup-username"
          v-model="username"
          autocomplete="username"
          required
        />
      </Field>

      <Field>
        <FieldLabel for="setup-password">{{
          t('common.field.password')
        }}</FieldLabel>
        <Input
          id="setup-password"
          v-model="password"
          autocomplete="new-password"
          minlength="8"
          required
          type="password"
        />
      </Field>

      <Field>
        <FieldLabel for="setup-confirm-password">{{
          t('common.field.confirmPassword')
        }}</FieldLabel>
        <Input
          id="setup-confirm-password"
          v-model="confirmPassword"
          autocomplete="new-password"
          minlength="8"
          required
          type="password"
        />
      </Field>

      <Field class="mbs-auto pbs-2">
        <Button
          type="submit"
          size="lg"
          class="inline-full"
        >
          {{ t('setup.admin.next') }}
          <ArrowRight class="ms-2 block-4 inline-4" />
        </Button>
      </Field>
    </FieldGroup>
  </form>
</template>
