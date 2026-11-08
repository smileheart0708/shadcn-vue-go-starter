<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const props = defineProps<{
  open: boolean
  username: string
  email: string
  avatarImageSrc: string | null
  avatarFallbackText: string
  profileError: string
  isSavingProfile: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'update:username': [value: string]
  'update:email': [value: string]
  'choose-avatar': []
  'save-profile': []
}>()

const { t } = useI18n()

function handleOpenUpdate(value: boolean) {
  emit('update:open', value)
}

function handleUsernameUpdate(value: string | number) {
  emit('update:username', String(value))
}

function handleEmailUpdate(value: string | number) {
  emit('update:email', String(value))
}

function chooseAvatar() {
  emit('choose-avatar')
}

function saveProfile() {
  emit('save-profile')
}
</script>

<template>
  <Dialog
    :open="props.open"
    @update:open="handleOpenUpdate"
  >
    <DialogTrigger as-child>
      <Button
        variant="outline"
        size="sm"
      >
        {{ t('common.action.edit') }}
      </Button>
    </DialogTrigger>
    <DialogContent class="sm:max-inline-110">
      <DialogHeader>
        <DialogTitle>{{ t('settings.account.editProfile') }}</DialogTitle>
        <DialogDescription>{{
          t('settings.account.editProfileDesc')
        }}</DialogDescription>
      </DialogHeader>

      <div class="grid gap-4 py-4">
        <div class="flex flex-col items-center gap-2">
          <Avatar class="rounded-full block-20 inline-20">
            <AvatarImage
              v-if="props.avatarImageSrc !== null"
              :src="props.avatarImageSrc"
              :alt="props.username"
            />
            <AvatarFallback class="rounded-full">{{
              props.avatarFallbackText
            }}</AvatarFallback>
          </Avatar>
          <Button
            variant="outline"
            size="sm"
            type="button"
            @click="chooseAvatar"
          >
            {{ t('common.avatar.change') }}
          </Button>
          <p class="text-xs text-muted-foreground">
            {{ t('common.avatar.hint') }}
          </p>
          <p
            v-if="props.profileError !== ''"
            class="text-xs text-destructive"
          >
            {{ props.profileError }}
          </p>
        </div>

        <div class="space-y-2">
          <Label for="edit-username">{{ t('common.field.username') }}</Label>
          <Input
            id="edit-username"
            :model-value="props.username"
            :placeholder="t('common.placeholder.username')"
            @update:model-value="handleUsernameUpdate"
          />
        </div>

        <div class="space-y-2">
          <Label for="edit-email">{{ t('common.field.email') }}</Label>
          <Input
            id="edit-email"
            :model-value="props.email"
            type="email"
            :placeholder="t('common.placeholder.email')"
            @update:model-value="handleEmailUpdate"
          />
        </div>
      </div>

      <DialogFooter>
        <DialogClose as-child>
          <Button variant="outline">{{ t('common.action.cancel') }}</Button>
        </DialogClose>
        <Button
          :disabled="props.isSavingProfile"
          @click="saveProfile"
        >
          {{
            props.isSavingProfile
              ? t('common.state.saving')
              : t('common.action.save')
          }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
