<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { ArrowRight } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { MIN_PASSWORD_LENGTH } from '@/lib/auth/password'
import { isValidUsername, USERNAME_MAX_LENGTH } from '@/utils/username'
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
  const usernameValue = username.value.trim()
  if (usernameValue.length === 0) {
    toast.error(t('common.validation.usernameRequired'))
    return
  }
  if (!isValidUsername(usernameValue)) {
    toast.error(t('common.validation.usernameInvalid'))
    return
  }
  if (password.value !== confirmPassword.value) {
    toast.error(t('common.validation.passwordMismatch'))
    return
  }

  wizard.completeAdmin({
    username: usernameValue,
    password: password.value,
  })
}
</script>

<template>
  <form
    class="flex flex-col"
    @submit.prevent="submit"
  >
    <h1 class="mbe-6 text-xl font-semibold tracking-tight">
      {{ t('setup.admin.title') }}
    </h1>

    <FieldGroup class="flex flex-col gap-5">
      <Field>
        <FieldLabel for="setup-username">{{
          t('common.field.username')
        }}</FieldLabel>
        <Input
          id="setup-username"
          v-model="username"
          autocomplete="username"
          :maxlength="USERNAME_MAX_LENGTH"
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
          :minlength="MIN_PASSWORD_LENGTH"
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
          :minlength="MIN_PASSWORD_LENGTH"
          required
          type="password"
        />
      </Field>

      <div class="pbs-2">
        <Button
          type="submit"
          size="lg"
          class="inline-full"
        >
          {{ t('common.action.next') }}
          <ArrowRight class="ms-2 block-4 inline-4" />
        </Button>
      </div>
    </FieldGroup>
  </form>
</template>
