<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { z } from 'zod'
import { resolveAPIErrorMessage } from '@/lib/api/error-messages'
import { parseExternalValue } from '@/lib/external-input'
import { useAuthStore } from '@/stores/auth'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const identifier = ref('')
const password = ref('')
const isSubmitting = ref(false)
const canRegister = computed(
  () => authStore.publicAuthConfig?.registrationEnabled === true,
)

const internalRedirectPathSchema = z.string().refine(isInternalRedirectPath)
const dashboardRedirectTarget = { name: 'dashboard' as const }

onMounted(() => {
  void authStore.refreshPublicState()
})

async function handleSubmit() {
  if (isSubmitting.value) {
    return
  }

  isSubmitting.value = true
  const loginPromise = authStore.login({
    identifier: identifier.value,
    password: password.value,
  })

  toast.promise(loginPromise, {
    loading: t('auth.signIn.signingIn'),
    success: () => t('auth.signIn.loginSuccess'),
    error: (error: unknown) =>
      resolveAPIErrorMessage(t, error) ?? t('auth.signIn.loginFailed'),
  })

  try {
    await loginPromise
    const redirectTarget =
      parseExternalValue(internalRedirectPathSchema, route.query.redirect) ??
      dashboardRedirectTarget
    await router.push(redirectTarget)
  } catch {
    return
  } finally {
    isSubmitting.value = false
  }
}

function isInternalRedirectPath(value: string) {
  return (
    value.startsWith('/') &&
    !value.startsWith('//') &&
    !value.startsWith('/\\') &&
    !/^[A-Za-z][A-Za-z\d+.-]*:/.test(value)
  )
}
</script>

<template>
  <div :class="cn('flex flex-col gap-6', props.class)">
    <form @submit.prevent="handleSubmit">
      <FieldGroup>
        <div class="flex flex-col items-center gap-2 text-center">
          <img
            src="/logo.svg"
            alt="Logo"
            class="block-8 inline-8"
          />
          <h1 class="text-xl font-bold">{{ t('auth.signIn.title') }}</h1>
          <p class="text-sm text-muted-foreground">
            {{ t('auth.signIn.description') }}
          </p>
        </div>
        <Field>
          <FieldLabel for="identifier">
            {{ t('common.field.usernameOrEmail') }}
          </FieldLabel>
          <Input
            id="identifier"
            v-model="identifier"
            type="text"
            :placeholder="t('auth.signIn.identifierPlaceholder')"
            autocomplete="username"
            required
          />
        </Field>
        <Field>
          <FieldLabel for="password">
            {{ t('common.field.password') }}
          </FieldLabel>
          <Input
            id="password"
            v-model="password"
            type="password"
            :placeholder="t('auth.signIn.passwordPlaceholder')"
            autocomplete="current-password"
            required
          />
        </Field>
        <Field>
          <Button
            type="submit"
            :disabled="isSubmitting"
          >
            {{
              isSubmitting
                ? t('auth.signIn.signingIn')
                : t('auth.signIn.submit')
            }}
          </Button>
        </Field>
        <Field v-if="canRegister">
          <FieldDescription class="text-center">
            {{ t('auth.signIn.noAccount') }}
            <RouterLink
              to="/register"
              class="font-medium hover:underline"
            >
              {{ t('auth.signIn.register') }}
            </RouterLink>
          </FieldDescription>
        </Field>
      </FieldGroup>
    </form>
  </div>
</template>
