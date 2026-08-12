<template>
  <div class="space-y-6">
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.features.channelMonitor.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.features.channelMonitor.description') }}
            </p>
            <p class="mt-1.5 text-xs">
              <router-link
                to="/admin/channels/monitor"
                class="inline-flex items-center gap-1 text-primary-600 hover:underline dark:text-primary-400"
              >
                {{ t('admin.settings.features.channelMonitor.configureLink') }}
                <span aria-hidden="true">→</span>
              </router-link>
            </p>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.features.channelMonitor.enabled') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.features.channelMonitor.enabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.channel_monitor_enabled" />
            </div>

            <div v-if="form.channel_monitor_enabled">
              <label class="input-label">
                {{ t('admin.settings.features.channelMonitor.defaultInterval') }}
                <span class="text-red-500">*</span>
              </label>
              <input
                v-model.number="form.channel_monitor_default_interval_seconds"
                type="number"
                min="15"
                max="3600"
                class="input"
              />
              <p class="mt-1 text-xs text-gray-400">
                {{ t('admin.settings.features.channelMonitor.defaultIntervalHint') }}
              </p>
            </div>

            <div class="mt-4 flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.features.channelMonitor.publicEnabled') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.features.channelMonitor.publicEnabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.channel_monitor_public_enabled" />
            </div>
          </div>
        </div>
</div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import Toggle from "@/components/common/Toggle.vue";
import type { SettingsForm } from "./settingsForm";

const props = defineProps<{
  form: SettingsForm;
}>();

const { form } = props;
const { t } = useI18n();
</script>
