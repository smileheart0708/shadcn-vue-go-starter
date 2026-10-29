<script setup lang="ts">
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { useRouteLoadingErrorRevision } from '@/router/route-loading'

const routeLoadingErrorRevision = useRouteLoadingErrorRevision()
const { t } = useI18n()

watch(
  routeLoadingErrorRevision,
  (revision, previousRevision) => {
    if (revision === previousRevision || revision === 0) {
      return
    }

    toast.error(t('common.feedback.loadFailed'))
  },
  { immediate: true },
)
</script>

<template>
  <span hidden />
</template>
