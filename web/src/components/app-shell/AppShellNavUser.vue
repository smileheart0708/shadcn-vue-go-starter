<script setup lang="ts">
import { computed } from 'vue'
import {
  Bell,
  CircleUserRound,
  CreditCard,
  EllipsisVertical,
  Globe,
  LogOut,
} from '@lucide/vue'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { getUserRoleBadgeVariant, getUserRoleLabel } from '@/lib/auth/roles'
import type { RoleKey } from '@/lib/api/auth'
import { useI18n } from 'vue-i18n'
import {
  isSupportedLocale,
  localeNames,
  supportedLocales,
} from '@/plugins/i18n/locales'
import { useLocaleStore } from '@/stores/locale'

const props = defineProps<{
  user: {
    username: string
    email: string | null
    avatarUrl: string | null
    // Null until the viewer is loaded: the badge must not guess a role.
    role: RoleKey | null
  }
}>()

const emit = defineEmits<{ logout: [] }>()

const { isMobile } = useSidebar()
const { t } = useI18n()
const localeStore = useLocaleStore()
const roleLabel = computed(() => getUserRoleLabel(t, props.user.role))
const roleBadgeVariant = computed(() =>
  getUserRoleBadgeVariant(props.user.role),
)

// reka-ui's radio group emits `AcceptableValue`, so narrow instead of asserting —
// the repo lints `consistent-type-assertions: never`, and template expressions are
// outside that rule's reach, making a cast here an easy way to smuggle one in.
function switchLanguage(newLocale: unknown) {
  if (isSupportedLocale(newLocale)) {
    localeStore.setLocale(newLocale)
  }
}
</script>

<template>
  <SidebarMenu>
    <SidebarMenuItem>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <SidebarMenuButton
            size="lg"
            class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
          >
            <UserAvatar
              :username="user.username"
              :avatar-url="user.avatarUrl"
              class="rounded-lg block-8 inline-8"
              fallback-class="rounded-lg"
            />
            <div class="grid flex-1 text-start text-sm/tight">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium">{{ user.username }}</span>
                <Badge
                  :variant="roleBadgeVariant"
                  class="px-1.5 py-0 text-[10px]"
                >
                  {{ roleLabel }}
                </Badge>
              </div>
              <span class="truncate text-xs text-muted-foreground">
                {{ user.email ?? ' ' }}
              </span>
            </div>
            <EllipsisVertical class="ms-auto block-4 inline-4" />
          </SidebarMenuButton>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          class="rounded-lg inline-(--reka-dropdown-menu-trigger-width) min-inline-56"
          :side="isMobile ? 'bottom' : 'right'"
          :side-offset="4"
          align="end"
        >
          <DropdownMenuLabel class="p-0 font-normal">
            <div class="flex items-center gap-2 px-1 py-1.5 text-start text-sm">
              <UserAvatar
                :username="user.username"
                :avatar-url="user.avatarUrl"
                class="rounded-lg block-8 inline-8"
                fallback-class="rounded-lg"
              />
              <div class="grid flex-1 text-start text-sm/tight">
                <div class="flex items-center gap-2">
                  <span class="truncate font-medium">{{ user.username }}</span>
                  <Badge
                    :variant="roleBadgeVariant"
                    class="px-1.5 py-0 text-[10px]"
                  >
                    {{ roleLabel }}
                  </Badge>
                </div>
                <span class="truncate text-xs text-muted-foreground">
                  {{ user.email ?? ' ' }}
                </span>
              </div>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem>
              <CircleUserRound />
              {{ t('nav.user.account') }}
            </DropdownMenuItem>
            <DropdownMenuItem>
              <CreditCard />
              {{ t('nav.user.billing') }}
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Bell />
              {{ t('nav.user.notifications') }}
            </DropdownMenuItem>
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Globe />
                {{ t('nav.user.language') }}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuRadioGroup
                  :model-value="localeStore.locale"
                  @update:model-value="switchLanguage($event)"
                >
                  <DropdownMenuRadioItem
                    v-for="lang in supportedLocales"
                    :key="lang"
                    :value="lang"
                  >
                    {{ localeNames[lang] }}
                  </DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem @click="emit('logout')">
            <LogOut />
            {{ t('nav.user.logout') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </SidebarMenuItem>
  </SidebarMenu>
</template>
