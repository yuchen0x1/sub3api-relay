<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { chartPngFilename, downloadChartPng, type ExportableChart } from '@/utils/chartExport'

const props = withDefaults(
  defineProps<{
    /** Returns the live Chart.js instance; evaluated on click so a re-mounted chart is picked up. */
    getChart: () => ExportableChart | null | undefined
    /** Export filename prefix; a timestamp and .png are appended. */
    filePrefix: string
    disabled?: boolean
  }>(),
  { disabled: false }
)

const { t } = useI18n()

function handleDownload(): void {
  if (props.disabled) return
  downloadChartPng(props.getChart(), chartPngFilename(props.filePrefix))
}
</script>

<template>
  <button
    type="button"
    class="inline-flex shrink-0 items-center rounded-lg border border-gray-200 bg-white px-2 py-1 text-[11px] font-semibold text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300 dark:hover:bg-dark-800"
    :disabled="disabled"
    :title="t('admin.ops.charts.downloadChartHint')"
    data-test="chart-export"
    @click="handleDownload"
  >
    {{ t('admin.ops.charts.downloadChart') }}
  </button>
</template>
