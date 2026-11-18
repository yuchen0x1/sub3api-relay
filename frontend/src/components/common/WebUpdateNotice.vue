<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useWebUpdate } from '@/composables/useWebUpdate'

const { t } = useI18n()
const { available, refresh, later } = useWebUpdate()
</script>

<template>
  <aside v-if="available" :aria-label="t('common.webUpdateTitle')"
    class="fixed inset-x-4 bottom-5 z-50 mx-auto max-w-sm rounded-xl border border-gray-200 bg-white p-4 shadow-lg dark:border-dark-600 dark:bg-dark-800 sm:left-auto sm:right-6">
    <p role="status" class="font-semibold text-gray-900 dark:text-white">{{ t('common.webUpdateTitle') }}</p>
    <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ t('common.webUpdateSave') }}</p>
    <div class="mt-3 flex gap-2">
      <button class="btn btn-primary" type="button" @click="refresh">{{ t('common.webUpdateRefresh') }}</button>
      <button class="btn btn-secondary" type="button" @click="later">{{ t('common.webUpdateLater') }}</button>
    </div>
  </aside>
</template>
