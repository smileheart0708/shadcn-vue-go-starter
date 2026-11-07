<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AppShellPage from '@/components/app-shell/AppShellPage.vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import AccountSettingsTab from '@/components/settings/AccountSettingsTab.vue'
import BasicSettingsTab from '@/components/settings/BasicSettingsTab.vue'

const { t } = useI18n()
</script>

<template>
  <AppShellPage>
    <Tabs
      default-value="basic"
      class="flex flex-col gap-6"
    >
      <TabsList class="self-start">
        <TabsTrigger value="basic">{{ t('settings.tabs.basic') }}</TabsTrigger>
        <TabsTrigger value="account">{{
          t('settings.tabs.account')
        }}</TabsTrigger>
      </TabsList>

      <TabsContent
        value="basic"
        class="space-y-8"
      >
        <BasicSettingsTab />
      </TabsContent>

      <TabsContent
        value="account"
        class="space-y-8"
      >
        <AccountSettingsTab />
      </TabsContent>
    </Tabs>
  </AppShellPage>
</template>
