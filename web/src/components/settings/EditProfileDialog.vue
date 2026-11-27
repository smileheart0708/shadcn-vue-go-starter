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
import { USERNAME_MAX_LENGTH } from '@/utils/username'

const props = defineProps<{
  avatarImageSrc: string | null
  avatarFallbackText: string
  profileError: string
  isSavingProfile: boolean
}>()

const emit = defineEmits<{
  'choose-avatar': []
  'save-profile': []
}>()

// The dialog owns no state: `open`/`username`/`email` are the parent's, so they use
// `defineModel` like the sibling AdminUserFormDialog instead of five forwarding wrappers.
const open = defineModel<boolean>('open', { required: true })
const username = defineModel<string>('username', { required: true })
const email = defineModel<string>('email', { required: true })

const { t } = useI18n()

// `Input` models `string | number` because it also backs numeric fields; both profile
// fields are text, so narrow at the boundary rather than widening the model.
function toText(value: string | number): string {
  return String(value)
}
</script>

<template>
  <Dialog v-model:open="open">
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
              :alt="username"
            />
            <AvatarFallback class="rounded-full">{{
              props.avatarFallbackText
            }}</AvatarFallback>
          </Avatar>
          <Button
            variant="outline"
            size="sm"
            type="button"
            @click="emit('choose-avatar')"
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
            :model-value="username"
            :maxlength="USERNAME_MAX_LENGTH"
            :placeholder="t('common.placeholder.username')"
            @update:model-value="username = toText($event)"
          />
        </div>

        <div class="space-y-2">
          <Label for="edit-email">{{ t('common.field.email') }}</Label>
          <Input
            id="edit-email"
            :model-value="email"
            type="email"
            :placeholder="t('common.placeholder.email')"
            @update:model-value="email = toText($event)"
          />
        </div>
      </div>

      <DialogFooter>
        <DialogClose as-child>
          <Button variant="outline">{{ t('common.action.cancel') }}</Button>
        </DialogClose>
        <Button
          :disabled="props.isSavingProfile"
          @click="emit('save-profile')"
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
