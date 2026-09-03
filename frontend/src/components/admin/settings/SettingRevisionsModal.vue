<template>
  <BaseDialog :show="show" :title="t('admin.settings.revisions.title')" width="extra-wide" @close="close">
    <div class="space-y-3">
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.revisions.hint') }}</p>
      <div v-if="loading" class="py-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
      <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-500">
        {{ t('admin.settings.revisions.empty') }}
      </div>
      <div v-else class="space-y-2">
        <div
          v-for="item in items"
          :key="item.id"
          class="flex items-center justify-between gap-3 rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-600"
        >
          <div class="min-w-0">
            <div class="font-medium text-gray-900 dark:text-white">#{{ item.id }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ formatTime(item.created_at) }} · {{ item.note || t('admin.settings.revisions.autoNote') }}
            </div>
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-sm shrink-0"
            :disabled="rollingBack === item.id"
            @click="handleRollback(item.id)"
          >
            {{ rollingBack === item.id ? t('admin.settings.revisions.rollingBack') : t('admin.settings.revisions.rollback') }}
          </button>
        </div>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button class="btn btn-secondary" @click="close">{{ t('common.close') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores/app'
import { apiClient } from '@/api/client'

interface SettingRevisionItem {
  id: number
  note: string
  created_at: string
}

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const items = ref<SettingRevisionItem[]>([])
const rollingBack = ref<number | null>(null)

const load = async () => {
  loading.value = true
  try {
    const { data } = await apiClient.get<{ items: SettingRevisionItem[]; total: number }>(
      '/admin/setting-revisions'
    )
    items.value = data?.items ?? []
  } catch {
    items.value = []
  } finally {
    loading.value = false
  }
}

const handleRollback = async (id: number) => {
  rollingBack.value = id
  try {
    await apiClient.post(`/admin/setting-revisions/${id}/rollback`)
    appStore.showSuccess(t('admin.settings.revisions.rollbackSuccess'))
    emit('close')
  } catch (err: any) {
    appStore.showError(err?.response?.data?.detail || t('admin.settings.revisions.rollbackFailed'))
  } finally {
    rollingBack.value = null
  }
}

const formatTime = (value: string) => (value || '').replace('T', ' ').slice(0, 19)

const close = () => emit('close')

watch(
  () => props.show,
  (visible) => {
    if (visible) load()
  }
)
</script>
