<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { cn } from '@/lib/utils'
import { MIN_PASSWORD_LENGTH } from '@/lib/auth/password'
import { normalizeOptionalEmail } from '@/utils/email'
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
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { resolveAPIErrorMessage } from '@/lib/api/error-messages'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const loadFailed = ref(false)
const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const isSubmitting = ref(false)

const registrationEnabled = computed(
  () => authStore.publicAuthConfig?.registrationEnabled === true,
)

// Nothing is loaded on mount on purpose: the route guard awaits `initialize()` before this page
// renders, so `publicAuthConfig` is already present. Re-fetching it here duplicated the request
// pair and put the form behind a skeleton with nothing to wait for. This is only used to
// re-read the policy after a failed submit — the usual cause is that an admin just disabled
// registration, and the form should switch to the "disabled" card instead of letting the user
// retry into the same error.
async function resyncPolicy() {
  await authStore.refreshPublicState()
  loadFailed.value = authStore.publicStateFailed
}

async function handleSubmit() {
  if (isSubmitting.value || !registrationEnabled.value) {
    return
  }
  if (password.value !== confirmPassword.value) {
    toast.error(t('common.validation.passwordMismatch'))
    return
  }

  isSubmitting.value = true
  const registerPromise = authStore.register({
    username: username.value.trim(),
    email: normalizeOptionalEmail(email.value),
    password: password.value,
  })

  toast.promise(registerPromise, {
    loading: t('common.state.creating'),
    success: () => t('auth.signUp.registerSuccess'),
    error: (error: unknown) =>
      resolveAPIErrorMessage(t, error) ?? t('auth.signUp.registerFailed'),
  })

  try {
    await registerPromise
    await router.push({ name: 'dashboard' })
  } catch {
    await resyncPolicy()
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div :class="cn('flex flex-col gap-6', props.class)">
    <Card v-if="loadFailed">
      <CardHeader class="text-center">
        <CardTitle>{{ t('auth.signUp.loadFailedTitle') }}</CardTitle>
        <CardDescription>{{
          t('auth.signUp.policyLoadFailed')
        }}</CardDescription>
      </CardHeader>
      <CardContent class="flex justify-center">
        <Button @click="resyncPolicy">{{ t('common.action.retry') }}</Button>
      </CardContent>
    </Card>

    <Card v-else-if="!registrationEnabled">
      <CardContent class="p-6">
        <Empty>
          <EmptyHeader>
            <EmptyTitle>{{ t('auth.signUp.disabledTitle') }}</EmptyTitle>
            <EmptyDescription>{{
              t('auth.signUp.disabledDescription')
            }}</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button as-child>
              <RouterLink :to="{ name: 'login' }">{{
                t('auth.signUp.signIn')
              }}</RouterLink>
            </Button>
          </EmptyContent>
        </Empty>
      </CardContent>
    </Card>

    <form
      v-else
      @submit.prevent="handleSubmit"
    >
      <FieldGroup>
        <div class="flex flex-col items-center gap-2 text-center">
          <h1 class="text-xl font-bold">{{ t('auth.signUp.title') }}</h1>
          <FieldDescription>
            {{ t('auth.signUp.description') }}
            <RouterLink
              to="/login"
              class="font-medium hover:underline"
            >
              {{ t('auth.signUp.signIn') }}
            </RouterLink>
          </FieldDescription>
        </div>

        <Field>
          <FieldLabel for="username">{{
            t('common.field.username')
          }}</FieldLabel>
          <Input
            id="username"
            v-model="username"
            autocomplete="username"
            :placeholder="t('common.placeholder.username')"
            required
            type="text"
          />
        </Field>

        <Field>
          <FieldLabel for="email">{{ t('common.field.email') }}</FieldLabel>
          <Input
            id="email"
            v-model="email"
            autocomplete="email"
            :placeholder="t('common.placeholder.email')"
            type="email"
          />
          <FieldDescription>{{
            t('auth.signUp.emailOptional')
          }}</FieldDescription>
        </Field>

        <Field>
          <FieldLabel for="password">{{
            t('common.field.password')
          }}</FieldLabel>
          <Input
            id="password"
            v-model="password"
            autocomplete="new-password"
            :placeholder="t('common.placeholder.password')"
            :minlength="MIN_PASSWORD_LENGTH"
            required
            type="password"
          />
        </Field>

        <Field>
          <FieldLabel for="confirmPassword">{{
            t('common.field.confirmPassword')
          }}</FieldLabel>
          <Input
            id="confirmPassword"
            v-model="confirmPassword"
            autocomplete="new-password"
            :placeholder="t('common.placeholder.confirmPassword')"
            :minlength="MIN_PASSWORD_LENGTH"
            required
            type="password"
          />
        </Field>

        <Field>
          <Button
            type="submit"
            :disabled="isSubmitting"
          >
            <Spinner
              v-if="isSubmitting"
              class="me-2"
            />
            {{
              isSubmitting
                ? t('common.state.creating')
                : t('auth.signUp.submit')
            }}
          </Button>
        </Field>
      </FieldGroup>
    </form>
  </div>
</template>
