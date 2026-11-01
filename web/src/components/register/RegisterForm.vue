<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { cn } from '@/lib/utils'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { resolveAPIErrorMessage } from '@/lib/api/error-messages'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const loadingPolicy = ref(true)
const loadFailed = ref(false)
const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const isSubmitting = ref(false)

const registrationEnabled = computed(
  () => authStore.publicAuthConfig?.registrationEnabled === true,
)

onMounted(() => {
  void loadPolicy()
})

async function loadPolicy() {
  loadingPolicy.value = true
  try {
    await authStore.refreshPublicState()
    loadFailed.value = false
  } catch (error) {
    loadFailed.value = true
    toast.error(
      resolveAPIErrorMessage(t, error) ?? t('auth.signUp.policyLoadFailed'),
    )
  } finally {
    loadingPolicy.value = false
  }
}

async function handleSubmit() {
  if (isSubmitting.value || !registrationEnabled.value) {
    return
  }
  if (password.value !== confirmPassword.value) {
    toast.error(t('auth.signUp.passwordMismatch'))
    return
  }

  isSubmitting.value = true
  const registerPromise = authStore.register({
    username: username.value.trim(),
    email: normalizeOptionalEmail(email.value),
    password: password.value,
  })

  toast.promise(registerPromise, {
    loading: t('auth.signUp.creating'),
    success: () => t('auth.signUp.registerSuccess'),
    error: (error: unknown) =>
      resolveAPIErrorMessage(t, error) ?? t('auth.signUp.registerFailed'),
  })

  try {
    await registerPromise
    await router.push({ name: 'dashboard' })
  } catch {
    await loadPolicy()
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div :class="cn('flex flex-col gap-6', props.class)">
    <div
      v-if="loadingPolicy"
      class="space-y-4 rounded-xl border p-6"
    >
      <div class="space-y-2 text-center">
        <Skeleton class="mx-auto rounded-md block-6 inline-40" />
        <Skeleton class="mx-auto rounded-md block-4 inline-60" />
      </div>
      <div class="space-y-3">
        <Skeleton class="rounded-xl block-18" />
        <Skeleton class="rounded-xl block-18" />
        <Skeleton class="rounded-xl block-18" />
        <Skeleton class="rounded-xl block-18" />
        <Skeleton class="rounded-md block-10" />
      </div>
    </div>

    <Card v-else-if="loadFailed">
      <CardHeader class="text-center">
        <CardTitle>{{ t('auth.signUp.loadFailedTitle') }}</CardTitle>
        <CardDescription>{{
          t('auth.signUp.policyLoadFailed')
        }}</CardDescription>
      </CardHeader>
      <CardContent class="flex justify-center">
        <Button @click="loadPolicy">{{ t('auth.signUp.retry') }}</Button>
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
            :placeholder="t('auth.signUp.usernamePlaceholder')"
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
            :placeholder="t('auth.signUp.emailPlaceholder')"
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
            :placeholder="t('auth.signUp.passwordPlaceholder')"
            minlength="8"
            required
            type="password"
          />
        </Field>

        <Field>
          <FieldLabel for="confirmPassword">{{
            t('auth.signUp.confirmPassword')
          }}</FieldLabel>
          <Input
            id="confirmPassword"
            v-model="confirmPassword"
            autocomplete="new-password"
            :placeholder="t('auth.signUp.confirmPasswordPlaceholder')"
            minlength="8"
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
              isSubmitting ? t('auth.signUp.creating') : t('auth.signUp.submit')
            }}
          </Button>
        </Field>
      </FieldGroup>
    </form>
  </div>
</template>
