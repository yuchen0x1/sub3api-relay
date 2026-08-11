<template>
  <div class="app-shell relative min-h-screen overflow-x-hidden bg-token-bg text-token-text">
    <div class="app-shell-aurora" aria-hidden="true" />
    <div class="app-shell-grid" aria-hidden="true" />
    <div class="app-shell-glow app-shell-glow-one" aria-hidden="true" />
    <div class="app-shell-glow app-shell-glow-two" aria-hidden="true" />

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative z-10 min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[92px]' : 'lg:ml-[276px]']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="app-main px-3 pb-6 pt-5 sm:px-4 md:px-6 lg:px-8 lg:pb-8">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: false
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
