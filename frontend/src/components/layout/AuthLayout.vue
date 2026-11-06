<template>
  <div class="relative flex min-h-screen items-center justify-center overflow-hidden bg-token-bg p-4">
    <!-- Background -->
    <div class="app-shell-aurora" aria-hidden="true" />
    <div class="app-shell-grid" aria-hidden="true" />

    <!-- Decorative Elements -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="absolute -right-32 -top-32 h-80 w-80 rounded-full bg-primary-400/15 blur-[110px]" />
      <div class="absolute -bottom-40 -left-24 h-96 w-96 rounded-full bg-accent-500/12 blur-[120px]" />
      <div class="absolute left-1/2 top-0 h-px w-[46rem] -translate-x-1/2 bg-gradient-to-r from-transparent via-primary-400/45 to-transparent" />
    </div>

    <!-- Content Container -->
    <div class="relative z-10 w-full max-w-md">
      <!-- Logo/Brand -->
      <div class="mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <div
            class="relative mb-5 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-2xl border border-primary-400/25 bg-[rgba(6,11,20,0.55)] p-2 shadow-[0_0_0_1px_rgba(34,211,238,0.14),0_0_42px_rgba(34,211,238,0.16)] dark:bg-[rgba(6,11,20,0.55)]"
          >
            <img :src="siteLogo || '/logo.svg'" alt="Sub3API" class="h-full w-full object-contain" />
          </div>
          <h1 class="text-gradient mb-2 text-3xl font-bold tracking-tight">
            {{ siteName }}
          </h1>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            {{ siteSubtitle }}
          </p>
          <div class="mx-auto mt-4 flex w-40 items-center gap-2" aria-hidden="true">
            <span class="h-px flex-1 bg-gradient-to-r from-transparent to-primary-400/40" />
            <span class="h-1 w-1 rotate-45 bg-primary-400/70" />
            <span class="h-px flex-1 bg-gradient-to-l from-transparent to-primary-400/40" />
          </div>
        </template>
      </div>

      <!-- Card Container -->
      <div class="card hud-corners neon-rail rounded-2xl p-6 shadow-glass sm:p-8">
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub3API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
@reference "../../style.css";
.text-gradient {
  @apply bg-gradient-to-r from-primary-600 to-primary-500 bg-clip-text text-transparent;
}
</style>
